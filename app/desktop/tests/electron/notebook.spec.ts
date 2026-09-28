import { test, expect, type ElectronApplication, type Page } from '@playwright/test';
import { mkdtemp, rm, copyFile, chmod, writeFile, readFile } from 'node:fs/promises';
import { join, resolve } from 'node:path';
import { tmpdir } from 'node:os';
import { code, encode, notebook, fixture, png } from '../../../qualification/notebook/fixtures';
import {
  launch,
  chooseProject,
  projectFixture,
  readyView,
  captureWindow,
  attachmentPreview,
  closeAttachmentPreview,
  openPaneActions,
  openAgentRoster,
} from './support';

let application: ElectronApplication,
  page: Page,
  base: string,
  profile: string,
  root: string,
  executable: string;
const savedPng = png();
function research() {
  const cell = code(
    'filter',
    '# Review sample quality\r\nkeep = qc["mapped_pct"] >= 80\r\nqc.loc[keep]',
  );
  cell.outputs = [
    {
      output_type: 'execute_result',
      execution_count: 7,
      metadata: {},
      data: {
        'text/plain': '    sample  mapped_pct\n0   S01          94.2\n2   S03          91.7',
        'text/html': '<script>window.notebookAttack=true</script>',
      },
    },
    {
      output_type: 'display_data',
      metadata: {},
      data: { 'image/png': savedPng.toString('base64') },
    },
    {
      output_type: 'display_data',
      metadata: {},
      data: { 'application/vnd.jupyter.widget-view+json': { model_id: 'example' } },
    },
  ];
  return encode(
    notebook([
      {
        cell_type: 'markdown',
        id: 'intro',
        metadata: {},
        source:
          '# Sample quality review\nCompare the saved code and output before deciding on the threshold.',
      },
      cell,
    ]),
  );
}
async function readDocument() {
  return page.evaluate(async () => {
    const projects = await window.gobble.projects.list();
    if (!projects.ok) throw new Error(projects.error.message);
    const result = await window.gobble.workspace.read({ projectId: projects.value[0]!.projectId });
    if (!result.ok) throw new Error(result.error.message);
    return result.value;
  });
}
async function open() {
  await page.getByRole('button', { name: 'analysis.ipynb', exact: true }).click();
  return readyView(page);
}
test.beforeEach(async () => {
  base = await mkdtemp(join(tmpdir(), 'gobble-r4a2-'));
  profile = join(base, 'profile');
  executable = join(base, 'codex');
  await copyFile(resolve('desktop/tests/fixtures/codex.cjs'), executable);
  await chmod(executable, 0o700);
  root = await projectFixture(base, 'Notebook research');
  await writeFile(join(root, 'analysis.ipynb'), research());
  ({ application, page } = await launch(profile, { GOBBLE_CODEX_EXECUTABLE: executable }));
  await chooseProject(application, page, root);
});
test.afterEach(async () => {
  await application?.close();
  await rm(base, { recursive: true, force: true });
});

test('saved source and output attach through the existing composer, preview offline and survive source changes and restart', async ({}, info) => {
  test.setTimeout(60_000);
  let primary = await open();
  const cell = primary.getByRole('article', { name: 'Notebook cell 2' });
  await expect(cell.getByText('No supported saved image or plain-text alternative.')).toBeVisible();
  await page
    .getByRole('textbox', { name: 'Message draft' })
    .fill('Should we revisit this threshold?');
  await cell.getByRole('button', { name: 'Select displayed text', exact: true }).first().click();
  await primary.getByRole('button', { name: 'Add to message from analysis.ipynb' }).click();
  let doc = await readDocument();
  const first = doc.chat.attachments![0]!;
  expect(first.evidence).toMatchObject({
    schemaVersion: 6,
    selection: { cell: { kind: 'id', id: 'filter' }, part: { kind: 'source' } },
  });
  expect(first.capture?.kind).toBe('notebook');
  await page
    .getByLabel('Message attachments', { exact: true })
    .locator('.attachment-toggle')
    .first()
    .click();
  await expect(attachmentPreview(page).locator('.evidence-preview-body pre')).toContainText(
    'keep = qc',
  );
  await captureWindow(application, info.outputPath('notebook-offline-preview.png'));
  await closeAttachmentPreview(page);
  const output = cell.getByRole('region', { name: 'Cell 2 output 1', exact: true });
  await output.getByRole('button', { name: 'Select displayed text', exact: true }).click();
  await primary.getByRole('button', { name: 'Add to message from analysis.ipynb' }).click();
  doc = await readDocument();
  expect(doc.chat.attachments).toHaveLength(2);
  expect(doc.chat.attachments![1]!.evidence.selection).toMatchObject({
    part: { kind: 'output', index: 0, mime: 'text/plain' },
  });
  const captured = doc.chat.attachments;
  await captureWindow(application, info.outputPath('notebook-discussion.png'));
  await writeFile(join(root, 'analysis.ipynb'), fixture('changed'));
  await primary.getByRole('button', { name: 'Refresh Notebook', exact: true }).click();
  await expect.poll(async () => (await readDocument()).selections.length).toBe(0);
  await readyView(page);
  expect((await readDocument()).selections).toHaveLength(0);
  expect((await readDocument()).chat.attachments).toEqual(captured);
  await application.close();
  ({ application, page } = await launch(profile, { GOBBLE_CODEX_EXECUTABLE: executable }));
  primary = await readyView(page);
  expect((await readDocument()).chat.attachments).toEqual(captured);
  await expect(page.getByRole('textbox', { name: 'Message draft' })).toHaveValue(
    'Should we revisit this threshold?',
  );
  await rm(join(root, 'analysis.ipynb'));
  await primary.getByRole('button', { name: 'Refresh Notebook', exact: true }).click();
  await expect(primary.getByRole('button', { name: 'Retry view', exact: true })).toBeVisible();
  await page
    .getByLabel('Message attachments', { exact: true })
    .locator('.attachment-toggle')
    .first()
    .click();
  await expect(attachmentPreview(page).locator('.evidence-preview-body pre')).toContainText(
    'keep = qc',
  );
  await closeAttachmentPreview(page);
  await writeFile(join(root, 'analysis.ipynb'), research());
  await primary.getByRole('button', { name: 'Retry view', exact: true }).click();
  await readyView(page);
  expect((await readDocument()).collaboration?.submissions ?? []).toHaveLength(0);
});

test('native text drag uses exact display offsets and rejects stale or foreign image requests', async () => {
  const primary = await open(),
    text = primary.getByLabel('Cell 2 source', { exact: true });
  const points = await text.evaluate((pre) => {
    const node = pre.firstChild!;
    const a = document.createRange(),
      b = document.createRange();
    a.setStart(node, 0);
    a.setEnd(node, 1);
    b.setStart(node, 8);
    b.setEnd(node, 9);
    const start = a.getBoundingClientRect(),
      end = b.getBoundingClientRect();
    return { x1: start.left, y: start.top + start.height / 2, x2: end.left };
  });
  await page.mouse.move(points.x1, points.y);
  await page.mouse.down();
  await page.mouse.move(points.x2, points.y, { steps: 8 });
  await page.mouse.up();
  await expect
    .poll(async () => (await readDocument()).selections[0]?.evidence.selection)
    .toMatchObject({ kind: 'notebook', selector: { kind: 'text', start: 0, end: 8 } });
  const result = await page.evaluate(async () => {
    const projects = await window.gobble.projects.list();
    if (!projects.ok) throw new Error('Projects unavailable');
    const projectId = projects.value[0]!.projectId;
    const doc = await window.gobble.workspace.read({ projectId });
    const connected = await window.gobble.workspace.connect();
    if (!doc.ok || !connected.ok) throw new Error('Workspace unavailable');
    // Reconnection retires every old ready lease before a stale image request.
    const s = doc.value.workspace.surfaces[0]!;
    const selected = doc.value.selections[0]!.evidence;
    if (selected.schemaVersion !== 6) throw new Error('Missing Notebook selection');
    const ack = {
      schemaVersion: 3 as const,
      projectId,
      surfaceId: s.surfaceId,
      rendererSessionId: connected.value.rendererSessionId,
      requestId: 'req_forged',
      generation: 1,
      dataRevision: selected.dataRevision,
      presentation: { spec: 0, view: 0, filter: 0 },
    };
    const stale = await window.gobble.workspace.notebookImage({
      acknowledgment: ack,
      target: selected,
    });
    const foreign = await window.gobble.workspace.notebookImage({
      acknowledgment: ack,
      target: { ...selected, projectId: 'prj_foreign' },
    });
    return { stale, foreign };
  });
  expect(result.stale).toMatchObject({ ok: false, error: { code: 'stale_revision' } });
  expect(result.foreign).toMatchObject({ ok: false, error: { code: 'forbidden' } });
});

test('native pointer and keyboard image selection at 150 percent captures the exact original pixels', async ({}, info) => {
  const primary = await open(),
    output = primary.getByRole('region', { name: 'Cell 2 output 2', exact: true });
  await output.getByRole('button', { name: 'View saved image' }).click();
  await expect(output.getByRole('img')).toBeVisible();
  await output.getByRole('combobox', { name: 'Notebook image zoom' }).selectOption('1.5');
  await output.getByRole('button', { name: 'Select image region' }).click();
  const area = output.getByRole('group', { name: 'Notebook image selection area' });
  await area.scrollIntoViewIfNeeded();
  const bounds = (await area.boundingBox())!;
  await page.mouse.move(
    bounds.x + (20 / 320) * bounds.width,
    bounds.y + (20 / 160) * bounds.height,
  );
  await page.mouse.down();
  await page.mouse.move(
    bounds.x + (200 / 320) * bounds.width,
    bounds.y + (100 / 160) * bounds.height,
    { steps: 8 },
  );
  await page.mouse.up();
  await area.press('ArrowRight');
  await area.press('Shift+ArrowDown');
  await area.press('Enter');
  await primary.getByRole('button', { name: 'Add to message from analysis.ipynb' }).click();
  const attachment = (await readDocument()).chat.attachments![0]!;
  expect(attachment.capture).toMatchObject({
    kind: 'notebook',
    representation: {
      kind: 'image',
      width: 180,
      height: 81,
      originalWidth: 320,
      originalHeight: 160,
      crop: { x: 21, y: 20, width: 180, height: 81 },
    },
  });
  await captureWindow(application, info.outputPath('notebook-image-selection.png'));

  // Preview is authorized through the draft identity, not direct filesystem paths.
  const preview = await page.evaluate(
    async (a) =>
      window.gobble.evidence.preview({
        kind: 'draft',
        projectId: a.evidence.projectId,
        attachmentId: a.attachmentId,
      }),
    attachment,
  );
  expect(preview.ok).toBe(true);
  if (!preview.ok || preview.value.kind !== 'image') throw new Error('Missing image preview');
  const exact = await application.evaluate(
    ({ nativeImage }, { source, capture }) => {
      const original = nativeImage
        .createFromBuffer(Buffer.from(source, 'base64'))
        .crop({ x: 21, y: 20, width: 180, height: 81 });
      return original
        .toBitmap()
        .equals(nativeImage.createFromBuffer(Buffer.from(capture, 'base64')).toBitmap());
    },
    { source: savedPng.toString('base64'), capture: preview.value.base64 },
  );
  expect(exact).toBe(true);
});

test('two Notebook Panes remain independent and compact and enlarged layouts retain the composer', async ({}, info) => {
  await writeFile(join(root, 'analysis.ipynb'), fixture('many'));
  let primary = await open();
  await primary.getByRole('button', { name: 'Next Notebook cells' }).click();
  await readyView(page);
  await expect(
    primary.getByRole('article', { name: 'Notebook cell 21', exact: true }),
  ).toBeVisible();
  await openPaneActions(primary, 'analysis.ipynb');
  await page.getByRole('button', { name: 'Duplicate analysis.ipynb in other pane' }).click();
  let secondary = await readyView(page, 'Secondary pane');
  await secondary.getByRole('button', { name: 'Previous Notebook cells' }).click();
  await readyView(page, 'Secondary pane');
  await expect(
    primary.getByRole('article', { name: 'Notebook cell 21', exact: true }),
  ).toBeVisible();
  await expect(
    secondary.getByRole('article', { name: 'Notebook cell 1', exact: true }),
  ).toBeVisible();
  await application.close();
  ({ application, page } = await launch(profile, { GOBBLE_CODEX_EXECUTABLE: executable }));
  primary = await readyView(page);
  secondary = await readyView(page, 'Secondary pane');
  await expect(
    primary.getByRole('article', { name: 'Notebook cell 21', exact: true }),
  ).toBeVisible();
  await expect(
    secondary.getByRole('article', { name: 'Notebook cell 1', exact: true }),
  ).toBeVisible();
  await application.evaluate(({ BrowserWindow }) =>
    BrowserWindow.getAllWindows()
      .find((w) => w.webContents.getURL().startsWith('app://gobble/'))!
      .setBounds({ width: 900, height: 650 }),
  );
  await captureWindow(application, info.outputPath('notebook-compact-workspace.png'));
  await page.getByRole('button', { name: 'Chat', exact: true }).click();
  await expect(page.getByRole('textbox', { name: 'Message draft' })).toBeInViewport();
  await captureWindow(application, info.outputPath('notebook-compact.png'));
  await page.getByRole('button', { name: 'Workspace', exact: true }).click();
  await expect(
    primary.getByRole('article', { name: 'Notebook cell 21', exact: true }),
  ).toBeVisible();
  await page.getByRole('button', { name: 'Chat', exact: true }).click();
  await application.evaluate(({ BrowserWindow }) => {
    const w = BrowserWindow.getAllWindows().find((w) =>
      w.webContents.getURL().startsWith('app://gobble/'),
    )!;
    w.setBounds({ width: 1440, height: 900 });
    w.webContents.setZoomFactor(1.5);
  });
  await expect(page.getByRole('textbox', { name: 'Message draft' })).toBeInViewport();
  await captureWindow(application, info.outputPath('notebook-zoom-150.png'));
});

test('malformed sources recover explicitly and renderer replacement preserves draft and captured attachments', async () => {
  test.setTimeout(60_000);
  const primary = await open();
  await primary.getByRole('button', { name: 'Select displayed text', exact: true }).first().click();
  await primary.getByRole('button', { name: 'Add to message from analysis.ipynb' }).click();
  await page.getByRole('textbox', { name: 'Message draft' }).fill('Keep this Notebook review.');
  const saved = (await readDocument()).chat;
  await writeFile(join(root, 'analysis.ipynb'), Buffer.from('{broken'));
  await primary.getByRole('button', { name: 'Refresh Notebook' }).click();
  await expect(primary.getByRole('alert')).toContainText('malformed');
  await writeFile(join(root, 'analysis.ipynb'), research());
  await primary.getByRole('button', { name: 'Retry view' }).click();
  await readyView(page);
  const crashed = page.waitForEvent('crash');
  await application.evaluate(({ dialog, BrowserWindow }) => {
    const log: string[] = [];
    (process as typeof process & { recoveryLog: string[] }).recoveryLog = log;
    dialog.showMessageBox = async () => {
      log.push('dialog');
      return { response: 0, checkboxChecked: false };
    };
    dialog.showErrorBox = (a, b) => {
      log.push(a + ':' + b);
    };
    const w = BrowserWindow.getAllWindows().find((w) =>
      w.webContents.getURL().startsWith('app://gobble/'),
    )!;
    w.webContents.on('render-process-gone', (_, d) => log.push('gone:' + d.reason));
    w.webContents.on('did-start-loading', () => log.push('loading'));
    w.webContents.on('did-finish-load', () => log.push('loaded'));
    w.webContents.on('did-fail-load', (_, c, d) => log.push('failed:' + c + ':' + d));
    setTimeout(() => w.webContents.forcefullyCrashRenderer(), 100);
  });
  await crashed;
  await expect
    .poll(async () =>
      application.evaluate(async ({ BrowserWindow }) => {
        const w = BrowserWindow.getAllWindows().find((w) =>
          w.webContents.getURL().startsWith('app://gobble/'),
        )!;
        return {
          crashed: w.webContents.isCrashed(),
          loading: w.webContents.isLoading(),
          log: (process as typeof process & { recoveryLog: string[] }).recoveryLog,
        };
      }),
    )
    .toMatchObject({
      crashed: false,
      loading: false,
      log: expect.arrayContaining(['dialog', 'loaded']),
    });
  await expect
    .poll(async () =>
      application.evaluate(async ({ BrowserWindow }) => {
        const w = BrowserWindow.getAllWindows().find((w) =>
          w.webContents.getURL().startsWith('app://gobble/'),
        )!;
        return w.webContents.executeJavaScript(
          `(async()=>{const projects=await window.gobble.projects.list();const doc=await window.gobble.workspace.read({projectId:projects.value[0].projectId});return {draft:document.querySelector('[aria-label="Message draft"]')?.value,ready:document.querySelector('.surface-view')?.dataset.ready,chat:doc.value.chat};})()`,
        );
      }),
    )
    .toMatchObject({
      draft: 'Keep this Notebook review.',
      ready: 'true',
      chat: { attachments: saved.attachments },
    });
});

test('one MiB text stays bounded, continuation keeps absolute offsets and oversized capture leaves the draft intact', async () => {
  await writeFile(join(root, 'analysis.ipynb'), fixture('long-text'));
  const primary = await open();
  const text = primary.getByLabel('Cell 1 source', { exact: true });
  expect((await text.textContent())!.length).toBeLessThanOrEqual(2048);
  await primary.getByRole('button', { name: 'Next text', exact: true }).click();
  await primary.getByRole('button', { name: 'Select displayed text', exact: true }).click();
  expect((await readDocument()).selections[0]!.evidence.selection).toMatchObject({
    selector: { start: 2048, end: 4096 },
  });
  await primary.getByRole('button', { name: 'Add to message from analysis.ipynb' }).click();
  const saved = (await readDocument()).chat.attachments;
  await primary.getByText('Select a text range', { exact: true }).click();
  await primary.getByLabel('Cell 1 source start offset', { exact: true }).fill('0');
  await primary.getByLabel('Cell 1 source end offset', { exact: true }).fill('100000');
  await primary.getByRole('button', { name: 'Use range', exact: true }).click();
  await primary.getByRole('button', { name: 'Add to message from analysis.ipynb' }).click();
  await expect(page.getByRole('alert')).toContainText('64 KiB');
  expect((await readDocument()).chat.attachments).toEqual(saved);
});

test('embedded image above generic text limits opens on demand and bounded exact capture succeeds', async () => {
  const bytes = fixture('large');
  expect(bytes.length).toBeGreaterThan(3 * 1024 * 1024);
  await writeFile(join(root, 'analysis.ipynb'), bytes);
  const primary = await open(),
    output = primary.getByRole('region', { name: 'Cell 3 output 2', exact: true });
  const surface = (await readDocument()).workspace.surfaces[0]!;
  if (surface.resource.kind !== 'file') throw new Error('Missing file');
  const raw = await page.evaluate(async (input) => window.gobble.files.read(input), {
    projectId: surface.projectId,
    resourceId: surface.resource.resourceId,
  });
  expect(raw).toMatchObject({ ok: false, error: { code: 'unsupported' } });
  await expect(output.getByRole('img')).toHaveCount(0);
  await output.getByRole('button', { name: 'View saved image' }).click();
  await expect(output.getByRole('img')).toBeVisible();
  await output.getByRole('button', { name: 'Select image region' }).click();
  await output.getByRole('button', { name: 'Use image region' }).click();
  await primary.getByRole('button', { name: 'Add to message from analysis.ipynb' }).click();
  expect((await readDocument()).chat.attachments![0]!.capture).toMatchObject({
    kind: 'notebook',
    representation: { width: 120, height: 80, originalWidth: 860, originalHeight: 860 },
  });
  await output.getByRole('button', { name: 'Hide image' }).click();
  await expect(output.getByRole('img')).toHaveCount(0);
  expect(await page.evaluate(() => Reflect.has(globalThis, 'notebookAttack'))).toBe(false);
});

test('explicit Send delivers captured Notebook text and image through the existing addressed path', async () => {
  const primary = await open();
  await primary.getByRole('button', { name: 'Select displayed text', exact: true }).first().click();
  await primary.getByRole('button', { name: 'Add to message from analysis.ipynb' }).click();
  const output = primary.getByRole('region', { name: 'Cell 2 output 2', exact: true });
  await output.getByRole('button', { name: 'View saved image' }).click();
  await expect(output.getByRole('img')).toBeVisible();
  await output.getByRole('button', { name: 'Select image region' }).click();
  await output.getByRole('button', { name: 'Use image region' }).click();
  await primary.getByRole('button', { name: 'Add to message from analysis.ipynb' }).click();
  await application.evaluate(({ shell }) => {
    shell.openExternal = async () => {};
  });
  await page.getByRole('button', { name: 'Account', exact: true }).click();
  await page.getByRole('button', { name: 'Sign in with ChatGPT' }).click();
  await expect(page.getByText('ChatGPT connected', { exact: true })).toBeVisible();
  await page.getByRole('button', { name: 'Close Codex account' }).click();
  await openAgentRoster(page);
  await page.getByRole('button', { name: 'Add agent', exact: true }).click();
  const dialog = page.getByRole('dialog', { name: 'Add agent', exact: true });
  await dialog.getByLabel('Name', { exact: true }).fill('Reviewer');
  await dialog.getByRole('button', { name: 'Add agent', exact: true }).click();
  await expect(dialog).not.toBeVisible();
  await page
    .getByRole('combobox', { name: 'Conversation recipient' })
    .selectOption({ label: 'To Reviewer' });
  await page
    .getByRole('textbox', { name: 'Message draft' })
    .fill('Review these saved Notebook excerpts.');
  await expect(page.getByText('2 attachments · Ready for Reviewer', { exact: true })).toBeVisible();
  await rm(join(root, 'analysis.ipynb'));
  await page.getByRole('button', { name: 'Send', exact: true }).click();
  await expect(page.locator('.message-state').getByText('completed', { exact: true })).toHaveCount(
    1,
  );
  const sent = JSON.parse(await readFile(join(profile, 'codex/home/fixture-state.json'), 'utf8'));
  expect(JSON.stringify(sent.threads)).toContain('notebook-display-utf16');
  expect(JSON.stringify(sent.threads)).toContain('natural-image-pixels');
  expect(JSON.stringify(sent.threads)).toContain('data:image/png;base64,');
  expect((await readDocument()).chat.attachments).toEqual([]);
});
