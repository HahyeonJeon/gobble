import {
  attachmentPreview,
  closeAttachmentPreview,
  openPaneActions,
  openAgentRoster,
  launch,
  chooseProject,
  projectFixture,
  readyView,
} from './support';
import { test, expect, type ElectronApplication, type Page } from '@playwright/test';
import { mkdtemp, rm, copyFile, chmod, writeFile, readFile } from 'node:fs/promises';
import { join, resolve } from 'node:path';
import { tmpdir } from 'node:os';

let application: ElectronApplication;
let page: Page;
let base: string;
let profile: string;
let executable: string;
let root: string;
test.beforeEach(async () => {
  base = await mkdtemp(join(tmpdir(), 'gobble-evidence-electron-'));
  profile = join(base, 'profile');
  executable = join(base, 'codex');
  await copyFile(resolve('desktop/tests/fixtures/codex.cjs'), executable);
  await chmod(executable, 0o700);
  ({ application, page } = await launch(profile, { GOBBLE_CODEX_EXECUTABLE: executable }));
  root = await projectFixture(base);
  await chooseProject(application, page, root);
  await application.evaluate(({ shell }) => {
    shell.openExternal = async () => {};
  });
  await page.getByRole('button', { name: 'Account', exact: true }).click();
  await page.getByRole('button', { name: 'Sign in with ChatGPT' }).click();
  await expect(page.getByText('ChatGPT connected', { exact: true })).toBeVisible();
  await page.getByRole('button', { name: 'Close Codex account' }).click();
  for (const name of ['Researcher', 'Reviewer']) {
    await openAgentRoster(page);
    await page.getByRole('button', { name: 'Add agent', exact: true }).click();
    const dialog = page.getByRole('dialog', { name: 'Add agent', exact: true });
    await dialog.getByLabel('Name', { exact: true }).fill(name);
    await dialog.getByRole('button', { name: 'Add agent', exact: true }).click();
    await expect(dialog).not.toBeVisible();
  }
  await page
    .getByRole('combobox', { name: 'Conversation recipient' })
    .selectOption({ label: 'To Reviewer' });
});
test.afterEach(async () => {
  await application?.close();
  if (base) await rm(base, { recursive: true, force: true });
});
const draftAttachments = () => page.getByLabel('Message attachments', { exact: true });
async function selectImage() {
  await page.getByRole('button', { name: 'quality.png', exact: true }).click();
  await readyView(page);
  await page.getByRole('button', { name: 'Select region', exact: true }).click();
  for (const [label, value] of [
    ['Left', '20'],
    ['Top', '25'],
    ['Width', '50'],
    ['Height', '50'],
  ])
    await page.getByRole('spinbutton', { name: label!, exact: true }).fill(value!);
  await page.getByRole('button', { name: 'Use region', exact: true }).click();
  await page.getByRole('button', { name: 'Hide region controls', exact: true }).click();
  await page.getByRole('button', { name: 'Add to message from quality.png', exact: true }).click();
}
async function complete() {
  await page.getByRole('button', { name: 'Send', exact: true }).click();
  await expect(page.locator('.message-state').getByText('completed', { exact: true })).toHaveCount(
    1,
  );
  await expect(draftAttachments()).toHaveCount(0);
}
test('addresses exact row and cropped image in one composer, retaining immutable history after restart', async ({}, info) => {
  await page.getByRole('button', { name: 'samples.csv', exact: true }).click();
  await readyView(page);
  await page.getByRole('checkbox', { name: /Select row 3:/ }).check();
  await page.getByRole('checkbox', { name: 'Include column Read 1' }).uncheck();
  await page.getByRole('checkbox', { name: 'Include column Read 2' }).uncheck();
  await page.getByRole('button', { name: 'Add to message from samples.csv' }).click();
  await expect(page.getByText('1 attachment · Ready for Reviewer', { exact: true })).toBeVisible();
  await draftAttachments()
    .getByRole('button', { name: /^samples\.csv/ })
    .click();
  await expect(attachmentPreview(page).getByRole('cell')).toHaveText(['S03', 'Treatment']);
  await closeAttachmentPreview(page);
  await expect(draftAttachments().getByRole('button', { name: /^samples\.csv/ })).toBeFocused();
  await selectImage();
  await openPaneActions(page, 'quality.png');
  await page.getByRole('button', { name: 'Move quality.png to other pane' }).click();
  await readyView(page);
  await readyView(page, 'Secondary pane');
  await expect(page.getByText('2 attachments · Ready for Reviewer', { exact: true })).toBeVisible();
  const draft = page.getByRole('textbox', { name: 'Message draft' });
  await draft.fill('Compare this sample with the selected Plot region.');
  const timelineBefore = await page.locator('.message-list').boundingBox();
  await draftAttachments()
    .getByRole('button', { name: /^quality\.png/ })
    .click();
  const preview = attachmentPreview(page).getByRole('img', {
    name: 'Attached preview of quality.png',
  });
  await expect(preview).toBeVisible();
  const actual = await preview.evaluate((element: HTMLImageElement) => ({
    width: element.naturalWidth,
    height: element.naturalHeight,
  }));
  const project = await page.evaluate(async () => {
    const result = await window.gobble.projects.list();
    if (!result.ok) throw new Error(result.error.message);
    return result.value[0]!.projectId;
  });
  const doc = await page.evaluate(async (projectId) => {
    const result = await window.gobble.workspace.read({ projectId });
    if (!result.ok) throw new Error(result.error.message);
    return result.value;
  }, project);
  const sel = doc.chat.attachments![1]!.evidence.selection;
  if (sel?.kind !== 'image') throw new Error('Image selection missing');
  expect(actual).toEqual({
    width:
      Math.ceil((sel.x + sel.width) * sel.originalWidth) - Math.floor(sel.x * sel.originalWidth),
    height:
      Math.ceil((sel.y + sel.height) * sel.originalHeight) - Math.floor(sel.y * sel.originalHeight),
  });
  await page.screenshot({ path: info.outputPath('05-6b-attachment-preview.png') });
  await application.evaluate(({ BrowserWindow }) =>
    BrowserWindow.getAllWindows()[0]!.setBounds({ width: 900, height: 650 }),
  );
  await expect(preview).toBeVisible();
  const modalBounds = (await attachmentPreview(page).boundingBox())!;
  expect(modalBounds.y).toBeGreaterThanOrEqual(0);
  expect(modalBounds.y + modalBounds.height).toBeLessThanOrEqual(
    await page.evaluate(() => innerHeight),
  );
  await page.screenshot({ path: info.outputPath('05-6b-compact-preview.png') });
  await application.evaluate(({ BrowserWindow }) =>
    BrowserWindow.getAllWindows()[0]!.setBounds({ width: 1280, height: 840 }),
  );
  await closeAttachmentPreview(page);
  await expect(draft).toHaveValue('Compare this sample with the selected Plot region.');
  await expect
    .poll(async () => (await page.locator('.message-list').boundingBox())!.height)
    .toBe(timelineBefore!.height);
  await expect(page.locator('textarea[aria-label="Message draft"]')).toHaveCount(1);
  await expect(page.getByRole('button', { name: 'Send', exact: true })).toBeEnabled();
  await page.screenshot({ path: info.outputPath('addressed-evidence.png') });
  await complete();
  const state = JSON.parse(await readFile(join(profile, 'codex/home/fixture-state.json'), 'utf8'));
  const threads = Object.values(state.threads) as {
    name: string;
    tools: unknown[];
    turns: { items: { content?: { type: string; text?: string; url?: string }[] }[] }[];
  }[];
  expect(threads).toHaveLength(1);
  expect(threads[0]!.name).toBe('Reviewer');
  expect(threads[0]!.tools).toEqual([]);
  const input = threads[0]!.turns[0]!.items[0]!.content!;
  expect(input.filter((item) => item.type === 'image')).toHaveLength(1);
  expect(JSON.stringify(input)).toContain('S03');
  expect(JSON.stringify(input)).not.toContain('S01_R1');
  await writeFile(join(root, 'samples.csv'), 'Sample,Group\nChanged,Later\n');
  await application.close();
  ({ application, page } = await launch(profile, { GOBBLE_CODEX_EXECUTABLE: executable }));
  await expect(page.locator('.message-exchange')).toHaveCount(1);
  const sent = page.getByLabel('Sent attachments', { exact: true });
  await sent.getByRole('button', { name: /^samples\.csv/ }).click();
  await expect(attachmentPreview(page).getByRole('cell')).toHaveText(['S03', 'Treatment']);
  await closeAttachmentPreview(page);
  await sent.getByRole('button', { name: /^quality\.png/ }).click();
  await expect(attachmentPreview(page).getByRole('img')).toBeVisible();
  await closeAttachmentPreview(page);
  const restored = JSON.parse(
    await readFile(join(profile, 'codex/home/fixture-state.json'), 'utf8'),
  );
  expect(
    Object.values(restored.threads).flatMap((thread) => (thread as { turns: unknown[] }).turns),
  ).toHaveLength(1);
});
test('retains draft attachments across restart and blocks stale, unsupported and foreign sends', async () => {
  await selectImage();
  await page.getByRole('textbox', { name: 'Message draft' }).fill('Keep this message.');
  await expect(page.getByText('1 attachment · Ready for Reviewer', { exact: true })).toBeVisible();
  await page.getByRole('combobox', { name: 'Message model' }).selectOption('fixture-text-model');
  await expect(page.locator('.attachment-status')).toContainText(
    'This model cannot receive images. Choose a supported model.',
  );
  await expect(page.getByRole('button', { name: 'Send', exact: true })).toBeDisabled();
  await expect(page.getByRole('textbox', { name: 'Message draft' })).toHaveValue(
    'Keep this message.',
  );
  await page
    .getByRole('combobox', { name: 'Conversation recipient' })
    .selectOption({ label: 'To Researcher' });
  await expect(
    page.getByText('1 attachment · Ready for Researcher', { exact: true }),
  ).toBeVisible();
  await application.close();
  ({ application, page } = await launch(profile, { GOBBLE_CODEX_EXECUTABLE: executable }));
  await expect(draftAttachments().getByRole('button', { name: /^quality\.png/ })).toBeVisible();
  await expect(page.getByRole('textbox', { name: 'Message draft' })).toHaveValue(
    'Keep this message.',
  );
  await page.getByRole('button', { name: 'Account', exact: true }).click();
  await page.getByRole('button', { name: 'Refresh connection', exact: true }).click();
  await page.getByRole('button', { name: 'Close Codex account' }).click();
  await expect(
    page.getByText('1 attachment · Ready for Researcher', { exact: true }),
  ).toBeVisible();
  await writeFile(join(root, 'quality.png'), Buffer.from('Changed source'));
  await page.getByRole('button', { name: 'Send', exact: true }).click();
  await expect(page.locator('.message-exchange')).toHaveCount(0);
  await expect(page.getByRole('textbox', { name: 'Message draft' })).toHaveValue(
    'Keep this message.',
  );
  await expect(draftAttachments()).toBeVisible();
  await draftAttachments().getByRole('button', { name: 'Remove attachment quality.png' }).click();
  await expect(draftAttachments()).toHaveCount(0);
  await expect(page.getByRole('button', { name: 'Send', exact: true })).toBeEnabled();
});
test('sends an explicit whole preview without typed text and reports missing saved evidence', async () => {
  await page.getByRole('button', { name: 'notes.txt', exact: true }).click();
  await readyView(page);
  await page.getByRole('button', { name: 'Add view preview of notes.txt to message' }).click();
  await expect(page.getByText('1 attachment · Ready for Reviewer', { exact: true })).toBeVisible();
  await expect(page.getByRole('textbox', { name: 'Message draft' })).toHaveValue('');
  await application.evaluate(({ BrowserWindow }) =>
    BrowserWindow.getAllWindows()[0]!.setBounds({ width: 900, height: 600 }),
  );
  await page
    .getByRole('navigation', { name: 'Project regions' })
    .getByRole('button', { name: 'Chat', exact: true })
    .click();
  expect((await page.locator('.message-list').boundingBox())!.height).toBeGreaterThanOrEqual(250);
  const send = (await page.getByRole('button', { name: 'Send', exact: true }).boundingBox())!;
  expect(send.y + send.height).toBeLessThanOrEqual(await page.evaluate(() => innerHeight));

  await complete();
  const doc = await page.evaluate(async () => {
    const p = await window.gobble.projects.list();
    if (!p.ok) throw new Error(p.error.message);
    const r = await window.gobble.workspace.read({ projectId: p.value[0]!.projectId });
    if (!r.ok) throw new Error(r.error.message);
    return r.value;
  });
  const asset = doc.collaboration!.submissions[0]!.evidence![0]!;
  const assetPath = join(
    profile,
    'workspace/evidence',
    doc.workspace.projectId,
    asset.asset.hash.slice(7) + '.blob',
  );
  const bytes = await readFile(assetPath);
  await rm(assetPath);
  await page
    .getByLabel('Sent attachments', { exact: true })
    .getByRole('button', { name: /^notes\.txt/ })
    .click();
  await expect(
    page.getByText(
      'Evidence unavailable: its saved content is missing or corrupt. The original record has been preserved.',
      { exact: true },
    ),
  ).toBeVisible();
  await writeFile(assetPath, bytes);
  await attachmentPreview(page).getByRole('button', { name: 'Retry preview' }).click();
  await expect(attachmentPreview(page)).toContainText('Inspect sample S03.');
  expect(
    await page.evaluate(
      async (projectId) =>
        window.gobble.evidence.preview({
          kind: 'sent',
          projectId,
          requestId: 'req_foreign',
          attachmentId: 'att_foreign',
        }),
      doc.workspace.projectId,
    ),
  ).toMatchObject({ ok: false, error: { code: 'not_found' } });
});

test('migrates v2 shared marks and captured evidence, reopens without origin, and distinguishes changed or missing sources', async ({}, info) => {
  await page.getByRole('button', { name: 'samples.csv', exact: true }).click();
  await readyView(page);
  await page.getByRole('checkbox', { name: /Select row 3:/ }).check();
  await page.getByRole('checkbox', { name: 'Include column Read 1' }).uncheck();
  await page.getByRole('checkbox', { name: 'Include column Read 2' }).uncheck();
  await page.getByRole('button', { name: 'Share mark from samples.csv' }).click();
  await expect(
    page.getByRole('article', { name: 'You shared a reference', exact: true }),
  ).toBeVisible();
  await page.getByRole('button', { name: 'Add to message from samples.csv' }).click();
  await page.getByRole('textbox', { name: 'Message draft' }).fill('Discuss S03');
  await complete();
  await page.getByRole('button', { name: 'Close samples.csv', exact: true }).click();
  const projectId = await page.evaluate(async () => {
    const result = await window.gobble.projects.list();
    if (!result.ok) throw new Error(result.error.message);
    return result.value[0]!.projectId;
  });
  await application.close();
  const path = join(profile, 'workspace/projects', projectId + '.json');
  const current = JSON.parse(
    await readFile(path, 'utf8'),
  ) as import('@gobble/contracts').WorkspaceDocument;
  const originalAsset = current.collaboration!.submissions[0]!.evidence![0]!.asset;
  function legacyReference(ref: import('@gobble/contracts').EvidenceRef) {
    if (ref.schemaVersion === 4 || ref.schemaVersion === 5 || ref.schemaVersion === 6)
      throw new Error('Legacy fixture required');
    const { schemaVersion: _version, origin, selection, ...target } = ref;
    const { coordinateSpace: _space, ...legacySelection } = selection!;
    return { ...target, surfaceId: origin!.surfaceId, selection: legacySelection };
  }
  const { viewLinks: _links, ...versionThreeFields } = current;
  const legacy = {
    ...versionThreeFields,
    schemaVersion: 2,
    selections: [],
    sharedReferences: current.sharedReferences!.map((item) => ({
      ...item,
      evidence: legacyReference(item.evidence),
    })),
    collaboration: {
      submissions: current.collaboration!.submissions.map((item) => ({
        ...item,
        evidence: item.evidence!.map((manifest) => ({
          ...manifest,
          evidence: legacyReference(manifest.evidence),
        })),
      })),
    },
  };
  const raw = JSON.stringify(legacy, null, 2) + '\n';
  await writeFile(path, raw);
  ({ application, page } = await launch(profile, { GOBBLE_CODEX_EXECUTABLE: executable }));
  const event = () => page.getByRole('article', { name: 'You shared a reference', exact: true });
  await expect(event()).toBeVisible();
  await event().getByRole('button', { name: 'Reveal', exact: true }).click();
  await readyView(page);
  await expect(page.locator('.shared-cell')).toHaveCount(2);
  expect(await readFile(path + '.v2.backup', 'utf8')).toBe(raw);
  const migrated = JSON.parse(
    await readFile(path, 'utf8'),
  ) as import('@gobble/contracts').WorkspaceDocument;
  expect(migrated.schemaVersion).toBe(21);
  expect(migrated.sharedReferences![0]!.evidence.schemaVersion).toBe(2);
  expect(migrated.collaboration!.submissions[0]!.evidence![0]!.asset).toEqual(originalAsset);
  await page.screenshot({ path: info.outputPath('r1-migrated-exact.png') });
  await application.close();
  // Verify that a saved origin is optional, independent of migration provenance.
  const pointer = migrated.sharedReferences![0]!.evidence;
  if (pointer.schemaVersion === 4) throw new Error('Legacy fixture required');
  delete pointer.origin;
  const migratedTarget = migrated.collaboration!.submissions[0]!.evidence![0]!.evidence;
  if (migratedTarget.schemaVersion === 4) throw new Error('Legacy fixture required');
  delete migratedTarget.origin;
  await writeFile(path, JSON.stringify(migrated));
  await writeFile(
    join(root, 'samples.csv'),
    'Sample,Group\nS01,Changed\nS02,Changed\nS03,Different\n',
  );
  ({ application, page } = await launch(profile, { GOBBLE_CODEX_EXECUTABLE: executable }));
  await readyView(page);
  await event().getByRole('button', { name: 'Reveal', exact: true }).click();
  await expect(page.getByText(/This shared reference targets an older version/)).toBeVisible();
  await expect(page.locator('.shared-cell')).toHaveCount(0);
  await page.screenshot({ path: info.outputPath('r1-historical.png') });
  const sent = page.getByLabel('Sent attachments', { exact: true });
  await sent.getByRole('button', { name: /^samples\.csv/ }).click();
  await expect(attachmentPreview(page).getByRole('cell')).toHaveText(['S03', 'Treatment']);
  await page.screenshot({ path: info.outputPath('r1-captured-evidence.png') });
  await closeAttachmentPreview(page);
  await application.close();
  await rm(join(root, 'samples.csv'));
  ({ application, page } = await launch(profile, { GOBBLE_CODEX_EXECUTABLE: executable }));
  await expect(page.getByRole('heading', { name: 'This view is unavailable' })).toBeVisible();
  await expect(event()).toBeVisible();
  await page.screenshot({ path: info.outputPath('r1-source-unavailable.png') });
  await page
    .getByLabel('Sent attachments', { exact: true })
    .getByRole('button', { name: /^samples\.csv/ })
    .click();
  await expect(attachmentPreview(page).getByRole('cell')).toHaveText(['S03', 'Treatment']);
  expect(await readFile(path + '.v2.backup', 'utf8')).toBe(raw);
});
