import { focusWindow } from './support';
import { test, expect, type ElectronApplication, type Page } from '@playwright/test';
import { mkdtemp, rm, copyFile, chmod, writeFile, readFile } from 'node:fs/promises';
import { join, resolve } from 'node:path';
import { tmpdir } from 'node:os';
import { code, encode, notebook, png } from '../../../qualification/notebook/fixtures';
import {
  launch,
  chooseProject,
  projectFixture,
  readyView,
  openAgentRoster,
  captureWindow,
} from './support';
let application: ElectronApplication, page: Page, base: string, profile: string, root: string;
const file = (name: string) => join(profile, 'codex/home', name);
async function document() {
  return page.evaluate(async () => {
    const projects = await window.gobble.projects.list();
    if (!projects.ok) throw new Error(projects.error.message);
    const value = await window.gobble.workspace.read({ projectId: projects.value[0]!.projectId });
    if (!value.ok) throw new Error(value.error.message);
    return value.value;
  });
}
async function userState() {
  const d = await document();
  return { surfaces: d.workspace.surfaces, selections: d.selections, chat: d.chat };
}
test.beforeEach(async () => {
  base = await mkdtemp(join(tmpdir(), 'gobble-r4a3-'));
  profile = join(base, 'profile');
  root = await projectFixture(base, 'Notebook shared research');
  const cell = code(
    'quality',
    '# Review saved sample quality\r\nkeep = qc["mapped_pct"] >= 80\r\nqc.loc[keep]',
  );
  cell.outputs = [
    {
      output_type: 'execute_result',
      execution_count: 7,
      metadata: {},
      data: { 'text/plain': 'Sample S01: 94.2%\nSample S03: 91.7%' },
    },
    { output_type: 'display_data', metadata: {}, data: { 'image/png': png().toString('base64') } },
  ];
  await writeFile(
    join(root, 'analysis.ipynb'),
    encode(
      notebook([
        cell,
        code(
          'long',
          Array.from({ length: 100 }, (_, i) => 'row_' + i + ' = ' + 'x'.repeat(240)).join('\n'),
        ),
      ]),
    ),
  );
  const executable = join(base, 'codex');
  await copyFile(resolve('desktop/tests/fixtures/codex.cjs'), executable);
  await chmod(executable, 0o700);
  await copyFile(
    resolve('desktop/tests/fixtures/notebook-tools.cjs'),
    join(base, 'notebook-tools.cjs'),
  );
  ({ application, page } = await launch(profile, { GOBBLE_CODEX_EXECUTABLE: executable }));
  await chooseProject(application, page, root);
  await application.evaluate(({ shell }) => {
    shell.openExternal = async () => {};
  });
  await page.getByRole('button', { name: 'Account', exact: true }).click();
  await page.getByRole('button', { name: 'Sign in with ChatGPT' }).click();
  await expect(page.getByText('ChatGPT connected', { exact: true })).toBeVisible();
  await page.getByRole('button', { name: 'Close Codex account' }).click();
  await openAgentRoster(page);
  await page.getByRole('button', { name: 'Add agent', exact: true }).click();
  const editor = page.getByRole('dialog', { name: 'Add agent', exact: true });
  await editor.getByLabel('Name', { exact: true }).fill('Researcher');
  await editor.getByLabel('Project access').selectOption('sharedViews');
  await editor.getByRole('button', { name: 'Add agent', exact: true }).click();
  await expect(editor).not.toBeVisible();
  await page
    .getByRole('combobox', { name: 'Conversation recipient' })
    .selectOption({ label: 'To Researcher' });
  await page.getByRole('button', { name: 'analysis.ipynb', exact: true }).click();
  await readyView(page);
});
test.afterEach(async () => {
  await application?.close();
  await rm(base, { recursive: true, force: true });
});
async function begin(message: string) {
  await focusWindow(application);
  const draft = page.getByRole('textbox', { name: 'Message draft' });
  await draft.fill(message);
  await page.getByRole('button', { name: 'Send', exact: true }).click();
  await expect(draft).toHaveValue('');
  await draft.fill('Keep my next instruction local.');
  const before = await userState();
  await focusWindow(application);
  await writeFile(file('notebook-gate'), 'ready');
  return before;
}
async function delivered() {
  await expect(page.locator('.message-state').getByText('completed', { exact: true })).toHaveCount(
    1,
  );
  return JSON.parse(await readFile(file('notebook-delivery.json'), 'utf8'));
}

test('Agent text and immutable question evidence preserve User state through Show/Return and missing-source fallback', async ({}, info) => {
  test.setTimeout(90_000);
  const primary = await readyView(page);
  await primary.getByRole('button', { name: 'Select displayed text', exact: true }).first().click();
  const before = await begin('notebook-point'),
    result = await delivered();
  expect(await userState()).toEqual(before);
  await expect(page.getByRole('textbox', { name: 'Message draft' })).toBeFocused();
  expect(result.visible.value.receipt.evidenceId).toBeUndefined();
  expect(result.selected.value.receipt).toMatchObject({
    pointable: true,
    evidence: { schemaVersion: 6 },
  });
  expect(result.selected.value.receipt.evidenceId).toBeTruthy();
  await expect(primary.locator('.notebook-authored-text').first()).toBeVisible();
  await captureWindow(application, info.outputPath('notebook-agent-text.png'));
  const event = page
    .locator('.shared-reference-event')
    .filter({ hasText: 'Review this Notebook selection.' });
  await primary.getByRole('button', { name: 'Collapse outputs', exact: true }).click();
  await primary.locator('.notebook-scroll').evaluate((el) => {
    el.scrollTop = 160;
  });
  const scroll = await primary.locator('.notebook-scroll').evaluate((el) => el.scrollTop),
    state = await userState();
  await event.getByRole('button', { name: 'Show in view', exact: true }).click();
  await readyView(page);
  await expect(primary.getByLabel('Notebook reference content')).toContainText('keep = qc');
  expect(await userState()).toEqual(state);
  await captureWindow(application, info.outputPath('notebook-agent-show.png'));
  await primary.getByRole('button', { name: 'Return to my view' }).click();
  await readyView(page);
  expect(await userState()).toEqual(state);
  expect(await primary.locator('.notebook-scroll').evaluate((el) => el.scrollTop)).toBe(scroll);
  await expect(primary.getByRole('button', { name: 'Show outputs', exact: true })).toBeVisible();
  await rm(join(root, 'analysis.ipynb'));
  await event.getByRole('button', { name: 'Show in view', exact: true }).click();
  await expect(page.getByText(/This Notebook version is unavailable/)).toBeVisible();
  await expect(
    event.getByRole('button', { name: 'View captured evidence', exact: true }),
  ).toBeVisible();
  await event.getByRole('button', { name: 'View captured evidence', exact: true }).click();
  await expect(page.locator('.evidence-preview-body')).toContainText('keep = qc');
  await captureWindow(application, info.outputPath('notebook-agent-captured.png'));
});

test('only explicit painted image observation receives pixels and preserves an unfinished User region through Show/Return', async ({}, info) => {
  test.setTimeout(90_000);
  const primary = await readyView(page);
  await primary.getByRole('button', { name: 'View saved image' }).click();
  await expect(primary.getByRole('img', { name: 'Saved Notebook output' })).toBeVisible();
  await primary.getByRole('button', { name: 'Select image region' }).click();
  const before = await begin('notebook-image'),
    result = await delivered();
  expect(await userState()).toEqual(before);
  expect(
    result.visible.response.contentItems.some((i: { type: string }) => i.type === 'inputImage'),
  ).toBe(false);
  expect(result.visible.value.content.imagesAvailable.length).toBe(1);
  expect(
    result.selected.response.contentItems.some((i: { type: string }) => i.type === 'inputImage'),
  ).toBe(true);
  await expect(primary.locator('.notebook-authored-image')).toContainText('Researcher');
  const region = await primary.locator('.notebook-image-region').getAttribute('style');
  const event = page.locator('.shared-reference-event');
  await event.getByRole('button', { name: 'Show in view', exact: true }).click();
  await readyView(page);
  await expect(primary.getByRole('img', { name: 'Exact Notebook reference region' })).toBeVisible();
  await captureWindow(application, info.outputPath('notebook-agent-image.png'));
  await primary.getByRole('button', { name: 'Return to my view' }).click();
  await readyView(page);
  await expect(primary.getByRole('button', { name: 'Use image region' })).toBeVisible();
  expect(await primary.locator('.notebook-image-region').getAttribute('style')).toBe(region);
  expect(await userState()).toEqual(before);
});

test('actual scrolling revokes old Notebook receipts and clipped source-preview cannot authorize a mark', async () => {
  test.setTimeout(90_000);
  const primary = await readyView(page);
  await begin('notebook-scope');
  await expect
    .poll(async () => {
      try {
        return JSON.parse(await readFile(file('notebook-observed.json'), 'utf8')).value.content
          .textParts.length;
      } catch {
        return 0;
      }
    })
    .toBeGreaterThan(0);
  await primary.locator('.notebook-scroll').evaluate((el) => {
    el.scrollTop = el.scrollHeight;
  });
  // An actual browser frame lets the visibility protocol revoke the previous receipt.
  await page.evaluate(
    () => new Promise((resolve) => requestAnimationFrame(() => requestAnimationFrame(resolve))),
  );
  await focusWindow(application);
  await writeFile(file('notebook-next'), 'ready');
  const result = await delivered();
  for (const key of ['stale', 'previewPoint', 'missing']) expect(result[key].success).toBe(false);
  expect(result.fresh.success).toBe(true);
  expect(result.preview.value.receipt.pointable).toBe(false);
  const parts = result.current.value.content.textParts.filter(
    (part: { evidence: { selection: { cell: { id: string } } } }) =>
      part.evidence.selection.cell.id === 'long',
  );
  expect(parts.length).toBeGreaterThan(0);
  expect(
    parts.reduce((n: number, part: { text: string }) => n + part.text.length, 0),
  ).toBeLessThanOrEqual(4096);
  expect(parts[0].text.length).toBeLessThan(247);
});

test('User output marks use the same exact Show contract and remain usable in a compact enlarged view', async ({}, info) => {
  test.setTimeout(60_000);
  const primary = await readyView(page);
  await primary
    .getByRole('region', { name: 'Cell 1 output 1', exact: true })
    .getByRole('button', { name: 'Select displayed text' })
    .click();
  await page.getByRole('textbox', { name: 'Message draft' }).fill('Discuss the saved output.');
  await primary
    .getByRole('button', { name: 'Share mark from analysis.ipynb', exact: true })
    .click();
  const event = page.locator('.shared-reference-event');
  await expect(event).toContainText('You pointed');
  const state = await userState();
  await event.getByRole('button', { name: 'Show in view', exact: true }).click();
  await readyView(page);
  await expect(primary.getByLabel('Notebook reference content')).toContainText('Sample S03: 91.7%');
  await application.evaluate(({ BrowserWindow }) => {
    const w = BrowserWindow.getAllWindows().find((w) =>
      w.webContents.getURL().startsWith('app://gobble/'),
    )!;
    w.setContentSize(1100, 800);
    w.webContents.setZoomFactor(1.25);
  });
  await expect.poll(() => page.evaluate(() => innerWidth)).toBe(880);
  if (await page.getByRole('button', { name: 'Workspace', exact: true }).isVisible())
    await page.getByRole('button', { name: 'Workspace', exact: true }).click();
  await expect(primary.getByRole('button', { name: 'Return to my view' })).toBeInViewport();
  await captureWindow(application, info.outputPath('notebook-user-compact.png'));
  await primary.getByRole('button', { name: 'Return to my view' }).click();
  await readyView(page);
  expect((await userState()).selections).toEqual(state.selections);
  expect((await userState()).chat.draft).toBe(state.chat.draft);
});

for (const change of ['fold', 'zoom'] as const) {
  test(`Notebook ${change} invalidates an earlier Agent viewport receipt`, async () => {
    test.setTimeout(60_000);
    const primary = await readyView(page);
    await begin('notebook-scope');
    await expect
      .poll(async () => {
        try {
          return JSON.parse(await readFile(file('notebook-observed.json'), 'utf8')).value.receipt
            .pointable;
        } catch {
          return false;
        }
      })
      .toBe(true);
    if (change === 'fold')
      await primary.getByRole('button', { name: 'Collapse outputs', exact: true }).click();
    else
      await application.evaluate(({ BrowserWindow }) => {
        BrowserWindow.getAllWindows()
          .find((w) => w.webContents.getURL().startsWith('app://gobble/'))!
          .webContents.setZoomFactor(1.25);
      });
    await page.evaluate(
      () => new Promise((resolve) => requestAnimationFrame(() => requestAnimationFrame(resolve))),
    );
    await focusWindow(application);
    await writeFile(file('notebook-next'), 'ready');
    const result = await delivered();
    expect(result.stale.success).toBe(false);
    expect(result.previewPoint.success).toBe(false);
    expect(result.fresh.success).toBe(true);
  });
}
