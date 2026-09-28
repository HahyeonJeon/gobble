import { test, expect, type ElectronApplication, type Page } from '@playwright/test';
import { mkdtemp, rm, copyFile, chmod, writeFile, readFile } from 'node:fs/promises';
import { join, resolve } from 'node:path';
import { tmpdir } from 'node:os';
import { makePdf, fixture } from '../../../qualification/pdf/fixtures';
import {
  launch,
  chooseProject,
  projectFixture,
  readyView,
  openAgentRoster,
  openPaneActions,
  attachmentPreview,
  closeAttachmentPreview,
  captureWindow,
} from './support';

async function document(page: Page) {
  return page.evaluate(async () => {
    const projects = await window.gobble.projects.list();
    if (!projects.ok) throw new Error(projects.error.message);
    const state = await window.gobble.workspace.read({ projectId: projects.value[0]!.projectId });
    if (!state.ok) throw new Error(state.error.message);
    return state.value;
  });
}
async function decoders(application: ElectronApplication) {
  return application.evaluate(({ BrowserWindow }) =>
    BrowserWindow.getAllWindows()
      .filter((w) => w.webContents.getURL().startsWith('pdf-host:'))
      .map((w) => ({
        id: w.id,
        visible: w.isVisible(),
        preferences: (
          w.webContents as typeof w.webContents & {
            getLastWebPreferences(): Record<string, unknown>;
          }
        ).getLastWebPreferences(),
      })),
  );
}
let application: ElectronApplication,
  page: Page,
  base: string,
  profile: string,
  root: string,
  executable: string;
test.beforeEach(async () => {
  base = await mkdtemp(join(tmpdir(), 'gobble-r3c2-'));
  profile = join(base, 'profile');
  executable = join(base, 'codex');
  await copyFile(resolve('desktop/tests/fixtures/codex.cjs'), executable);
  await chmod(executable, 0o700);
  root = await projectFixture(base, 'PDF research');
  await writeFile(join(root, 'report.pdf'), makePdf({ pages: 4 }));
  ({ application, page } = await launch(profile, { GOBBLE_CODEX_EXECUTABLE: executable }));
  await chooseProject(application, page, root);
});
test.afterEach(async () => {
  await application?.close();
  await rm(base, { recursive: true, force: true });
});

test('PDF page and pointer/keyboard region become immutable draft and sent evidence across refresh and restart', async ({}, info) => {
  test.setTimeout(90_000);
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
  await page.getByRole('button', { name: 'report.pdf', exact: true }).click();
  let primary = await readyView(page);
  await primary.getByRole('button', { name: 'Next PDF page' }).click();
  await readyView(page);
  await expect(primary.getByRole('spinbutton', { name: 'PDF page number' })).toHaveValue('2');
  await primary.getByRole('button', { name: 'Add page to message' }).click();
  await expect(page.getByText('1 attachment · Ready for Reviewer', { exact: true })).toBeVisible();
  await primary.getByRole('button', { name: 'Select region', exact: true }).click();
  const area = primary.getByRole('group', { name: 'PDF page 2 selection area' });
  const bounds = (await area.boundingBox())!;
  await page.mouse.move(bounds.x + bounds.width * 0.1, bounds.y + 40);
  await page.mouse.down();
  await page.mouse.move(bounds.x + bounds.width * 0.6, bounds.y + 150);
  await page.mouse.up();
  await area.press('ArrowRight');
  await area.press('Shift+ArrowDown');
  await area.press('Enter');
  await primary.getByRole('button', { name: 'Add to message from report.pdf' }).click();
  await expect(page.getByText('2 attachments · Ready for Reviewer', { exact: true })).toBeVisible();
  const captured = (await document(page)).chat.attachments!;
  expect(
    captured.map((a) => a.evidence.selection?.kind === 'pdf' && a.evidence.selection.scope),
  ).toEqual(['page', 'region']);
  expect(captured[1]!.capture).toMatchObject({
    kind: 'pdf',
    representation: { kind: 'image', pdf: { pageIndex: 1 } },
  });
  expect((await document(page)).collaboration?.submissions ?? []).toHaveLength(0);
  await page
    .getByRole('textbox', { name: 'Message draft' })
    .fill('Compare this page with the selected region.');
  await captureWindow(application, info.outputPath('pdf-discussion.png'));
  await application.evaluate(({ BrowserWindow }) =>
    BrowserWindow.getAllWindows()
      .find((w) => w.webContents.getURL().startsWith('app://gobble/'))!
      .setBounds({ width: 900, height: 650 }),
  );
  await expect(page.getByRole('textbox', { name: 'Message draft' })).toBeInViewport();
  await page.getByRole('button', { name: 'Workspace', exact: true }).click();
  await readyView(page);
  await expect(
    primary.getByRole('button', { name: 'Select region', exact: true }),
  ).toBeInViewport();
  await captureWindow(application, info.outputPath('pdf-compact.png'));
  await application.evaluate(({ BrowserWindow }) =>
    BrowserWindow.getAllWindows()
      .find((w) => w.webContents.getURL().startsWith('app://gobble/'))!
      .setBounds({ width: 1440, height: 900 }),
  );
  await readyView(page);
  await writeFile(join(root, 'report.pdf'), makePdf({ pages: 4, rotate: 90 }));
  await primary.getByRole('button', { name: 'Refresh PDF' }).click();
  await readyView(page);
  expect((await document(page)).chat.attachments).toEqual(captured);
  expect((await document(page)).selections).toHaveLength(0);
  await application.close();
  ({ application, page } = await launch(profile, { GOBBLE_CODEX_EXECUTABLE: executable }));
  primary = await readyView(page);
  await expect(primary.getByRole('spinbutton', { name: 'PDF page number' })).toHaveValue('2');
  expect((await document(page)).chat.attachments).toEqual(captured);
  await expect(page.getByRole('textbox', { name: 'Message draft' })).toHaveValue(
    'Compare this page with the selected region.',
  );
  await rm(join(root, 'report.pdf'));
  await primary.getByRole('button', { name: 'Refresh PDF' }).click();
  await expect(primary.getByRole('button', { name: 'Retry view', exact: true })).toBeVisible();
  await page.getByRole('button', { name: 'Account', exact: true }).click();
  await page.getByRole('button', { name: 'Refresh connection', exact: true }).click();
  await page.getByRole('button', { name: 'Close Codex account' }).click();
  await expect(page.getByText('2 attachments · Ready for Reviewer', { exact: true })).toBeVisible();
  await page
    .getByLabel('Message attachments', { exact: true })
    .locator('.attachment-toggle')
    .nth(1)
    .click();
  const image = attachmentPreview(page).getByRole('img');
  await expect(image).toBeVisible();
  const dimensions = await image.evaluate((img: HTMLImageElement) => ({
    width: img.naturalWidth,
    height: img.naturalHeight,
  }));
  expect(captured[1]!.capture!.representation).toMatchObject(dimensions);
  await captureWindow(application, info.outputPath('pdf-captured-preview.png'));
  await closeAttachmentPreview(page);
  await page.getByRole('button', { name: 'Send', exact: true }).click();
  await expect(page.locator('.message-state').getByText('completed', { exact: true })).toHaveCount(
    1,
  );
  const sent = JSON.parse(await readFile(join(profile, 'codex/home/fixture-state.json'), 'utf8'));
  expect(JSON.stringify(sent.threads)).toContain('data:image/png;base64,');
  expect(JSON.stringify(sent.threads)).toContain('pdf-user-space');
  const before = (await document(page)).collaboration;
  await application.close();
  ({ application, page } = await launch(profile, { GOBBLE_CODEX_EXECUTABLE: executable }));
  expect((await document(page)).collaboration?.submissions).toEqual(before?.submissions);
});

test('PDF readers isolate navigation, reject stale or cross-Pane capture and dispose when hidden or switched', async () => {
  test.setTimeout(60_000);
  await page.getByRole('button', { name: 'report.pdf', exact: true }).click();
  const primary = await readyView(page);
  const raw = await page.evaluate(async () => {
    const p = await window.gobble.projects.list();
    if (!p.ok) throw new Error('Projects unavailable');
    const d = await window.gobble.workspace.read({ projectId: p.value[0]!.projectId });
    if (!d.ok) throw new Error('Workspace unavailable');
    const surface = d.value.workspace.surfaces[0]!;
    if (surface.resource.kind !== 'file') throw new Error('File unavailable');
    return window.gobble.files.read({
      projectId: surface.projectId,
      resourceId: surface.resource.resourceId,
    });
  });
  expect(raw.ok).toBe(false);
  if (!raw.ok) expect(raw.error.code).toBe('unsupported');
  await primary.getByRole('button', { name: 'Add page to message' }).click();
  await expect.poll(async () => (await document(page)).chat.attachments?.length).toBe(1);
  const first = (await document(page)).chat.attachments![0]!;
  await openPaneActions(primary, 'report.pdf');
  await primary.getByRole('button', { name: 'Duplicate report.pdf in other pane' }).click();
  const secondary = await readyView(page, 'Secondary pane');
  await expect.poll(async () => (await decoders(application)).length).toBe(2);
  for (const host of await decoders(application)) {
    expect(host.visible).toBe(false);
    expect(host.preferences).toMatchObject({
      sandbox: true,
      contextIsolation: true,
      nodeIntegration: false,
    });
  }
  const rights = await application.evaluate(async ({ BrowserWindow }) =>
    Promise.all(
      BrowserWindow.getAllWindows()
        .filter((w) => w.webContents.getURL().startsWith('pdf-host:'))
        .map((w) =>
          w.webContents.executeJavaScript(
            '({workspace:typeof window.gobble,node:typeof process,bridge:typeof window.pdfJob})',
          ),
        ),
    ),
  );
  expect(rights.every((r) => r.workspace === 'undefined' && r.node === 'undefined')).toBe(true);
  expect(rights.every((r) => r.bridge === 'object')).toBe(true);
  const restrictions = await application.evaluate(async ({ BrowserWindow }) => {
    const host = BrowserWindow.getAllWindows().find((w) =>
      w.webContents.getURL().startsWith('pdf-host:'),
    )!;
    return host.webContents.executeJavaScript(`(async () => {
      const response = await fetch('/index.html');
      return { policy:response.headers.get('content-security-policy'),
        deniedAsset:(await fetch('/private.txt')).status,
        external:await fetch('https://example.invalid/blocked').then(()=>false,()=>true),
        child:window.open('https://example.invalid/blocked')===null };
    })()`);
  });
  expect(restrictions).toMatchObject({ deniedAsset: 404, external: true, child: true });
  expect(restrictions.policy).toContain("default-src 'none'");
  expect(restrictions.policy).toContain("connect-src 'self'");
  await secondary.getByRole('button', { name: 'Next PDF page' }).click();
  await readyView(page, 'Secondary pane');
  await secondary.getByRole('button', { name: 'Rotate', exact: true }).click();
  await readyView(page, 'Secondary pane');
  await secondary.getByRole('combobox', { name: 'PDF zoom' }).selectOption('1.25');
  await readyView(page, 'Secondary pane');
  await expect(primary.getByRole('spinbutton', { name: 'PDF page number' })).toHaveValue('1');
  await expect(secondary.getByRole('spinbutton', { name: 'PDF page number' })).toHaveValue('2');
  const rejection = await page.evaluate(async (evidence) => {
    const project = await window.gobble.projects.list();
    if (!project.ok) throw new Error('Projects unavailable');
    const d = await window.gobble.workspace.read({ projectId: project.value[0]!.projectId });
    if (!d.ok) throw new Error('Workspace unavailable');
    return window.gobble.workspace.command({
      projectId: d.value.workspace.projectId,
      expectedRevision: d.value.workspace.revision,
      requestId: 'req_pdf_cross_pane',
      action: {
        kind: 'attach',
        surfaceId:
          d.value.workspace.layout.kind === 'split'
            ? d.value.workspace.layout.secondary.activeSurfaceId!
            : '',
        evidence,
      },
    });
  }, first.evidence);
  expect(rejection.ok).toBe(false);
  if (!rejection.ok) expect(rejection.error.code).toBe('forbidden');
  expect((await document(page)).chat.attachments).toHaveLength(1);
  await openPaneActions(secondary, 'report.pdf');
  await secondary.getByRole('button', { name: 'Close report.pdf', exact: true }).click();
  await expect.poll(async () => (await decoders(application)).length).toBe(1);
  await primary.getByRole('tab', { name: 'report.pdf', exact: true }).click();
  await page.getByRole('button', { name: 'notes.txt', exact: true }).click();
  await readyView(page);
  await expect.poll(async () => (await decoders(application)).length).toBe(0);
  await page.getByRole('tab', { name: /report.pdf/ }).click();
  await readyView(page);
  await expect.poll(async () => (await decoders(application)).length).toBe(1);
  const other = await projectFixture(base, 'Other research');
  await chooseProject(application, page, other);
  await expect.poll(async () => (await decoders(application)).length).toBe(0);
});

test('unsupported PDFs fail locally and recover through the existing Pane retry', async () => {
  test.setTimeout(60_000);
  await writeFile(join(root, 'report.pdf'), fixture('corrupt-image'));
  await page.getByRole('button', { name: 'report.pdf', exact: true }).click();
  const primary = page.getByRole('region', { name: 'Primary pane', exact: true });
  await expect(primary.getByRole('button', { name: 'Retry view', exact: true })).toBeVisible();
  await expect.poll(async () => (await decoders(application)).length).toBe(0);
  await expect(page.getByRole('textbox', { name: 'Message draft' })).toBeEnabled();
  await writeFile(join(root, 'report.pdf'), makePdf());
  await primary.getByRole('button', { name: 'Retry view', exact: true }).click();
  await readyView(page);
  await primary.getByRole('button', { name: 'Add page to message' }).click();
  await expect.poll(async () => (await document(page)).chat.attachments?.length).toBe(1);
});

test('closing a loading PDF discards late work and leaves the Project usable', async () => {
  await writeFile(join(root, 'report.pdf'), makePdf({ dense: true }));
  await page.getByRole('button', { name: 'report.pdf', exact: true }).click();
  const primary = page.getByRole('region', { name: 'Primary pane', exact: true });
  await expect(primary.getByText('Loading report.pdf…', { exact: true })).toBeVisible();
  await primary.getByRole('button', { name: 'Close report.pdf', exact: true }).click();
  await expect.poll(async () => (await decoders(application)).length).toBe(0);
  await page.getByRole('button', { name: 'notes.txt', exact: true }).click();
  await readyView(page);
  await expect(primary.getByRole('textbox', { name: 'notes.txt text' })).toHaveValue(
    /Inspect sample/,
  );
  expect((await document(page)).workspace.surfaces.some((s) => s.view === 'pdf')).toBe(false);
  expect((await document(page)).chat.attachments ?? []).toHaveLength(0);
  await expect.poll(async () => (await decoders(application)).length).toBe(0);
});
