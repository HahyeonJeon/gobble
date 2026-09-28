import { focusWindow } from './support';
import { test, expect, type ElectronApplication, type Page } from '@playwright/test';
import { mkdtemp, rm, copyFile, chmod, writeFile, readFile } from 'node:fs/promises';
import { join, resolve } from 'node:path';
import { tmpdir } from 'node:os';
import { makePdf } from '../../../qualification/pdf/fixtures';
import {
  launch,
  chooseProject,
  projectFixture,
  readyView,
  openAgentRoster,
  captureWindow,
} from './support';

async function document(page: Page) {
  return page.evaluate(async () => {
    const projects = await window.gobble.projects.list();
    if (!projects.ok) throw new Error(projects.error.message);
    const result = await window.gobble.workspace.read({ projectId: projects.value[0]!.projectId });
    if (!result.ok) throw new Error(result.error.message);
    return result.value;
  });
}
const userState = async (page: Page) => {
  const d = await document(page);
  return { surfaces: d.workspace.surfaces, selections: d.selections, chat: d.chat };
};
let application: ElectronApplication,
  page: Page,
  base: string,
  profile: string,
  root: string,
  executable: string;
test.beforeEach(async () => {
  base = await mkdtemp(join(tmpdir(), 'gobble-r3c3-'));
  profile = join(base, 'profile');
  executable = join(base, 'codex');
  await copyFile(resolve('desktop/tests/fixtures/codex.cjs'), executable);
  await chmod(executable, 0o700);
  root = await projectFixture(base, 'PDF shared review');
  await writeFile(join(root, 'report.pdf'), makePdf({ pages: 4 }));
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
  await page.getByRole('button', { name: 'report.pdf', exact: true }).click();
  await readyView(page);
});
test.afterEach(async () => {
  await application?.close();
  await rm(base, { recursive: true, force: true });
});
const file = (name: string) => join(profile, 'codex/home', name);
async function begin(message: string) {
  await focusWindow(application);
  const draft = page.getByRole('textbox', { name: 'Message draft' });
  await draft.fill(message);
  await page.getByRole('button', { name: 'Send', exact: true }).click();
  await expect(draft).toHaveValue('');
  await draft.fill('My next instruction stays local.');
  const before = await userState(page);
  await focusWindow(application);
  await writeFile(file('pdf-gate'), 'ready');
  return before;
}
async function delivered() {
  await expect(page.locator('.message-state').getByText('completed', { exact: true })).toHaveCount(
    1,
  );
  return JSON.parse(await readFile(file('pdf-delivery.json'), 'utf8'));
}

test('Agent PDF marks and immutable question capture preserve User state across Show, Return, changed source and restart', async ({}, info) => {
  test.setTimeout(100_000);
  let primary = await readyView(page);
  const before = await begin('pdf-point');
  await expect(page.locator('.shared-reference-event')).toHaveCount(1);
  const delivery = await delivered();
  expect(await userState(page)).toEqual(before);
  await expect(page.getByRole('textbox', { name: 'Message draft' })).toBeFocused();
  await expect(primary.locator('.pdf-shared-mark')).toContainText('Researcher');
  expect(delivery.visible.value.receipt).toMatchObject({
    pointable: true,
    scope: 'view',
    evidence: { schemaVersion: 5 },
  });
  expect(delivery.selected.value.content.image.pdf.modelHash).toBe(
    delivery.selected.value.receipt.evidence.selection.modelHash,
  );
  const png = delivery.selected.response.contentItems.find(
    (i: { type: string }) => i.type === 'inputImage',
  );
  expect(png).toBeTruthy();
  await captureWindow(application, info.outputPath('pdf-agent-mark.png'));
  await primary.getByRole('button', { name: 'Next PDF page' }).click();
  await readyView(page);
  await primary.getByRole('button', { name: 'Next PDF page' }).click();
  await readyView(page);
  await primary.getByRole('combobox', { name: 'PDF zoom' }).selectOption('1.5');
  await readyView(page);
  await primary.getByRole('button', { name: 'Rotate', exact: true }).click();
  await readyView(page);
  await primary.getByRole('button', { name: 'Select region', exact: true }).click();
  await primary.getByRole('group', { name: 'PDF page 3 selection area' }).press('Enter');
  await expect.poll(async () => (await document(page)).selections.length).toBe(1);
  await primary.locator('.pdf-scroll').evaluate((el) => {
    el.scrollTop = 100;
  });
  const scroll = await primary.locator('.pdf-scroll').evaluate((el) => el.scrollTop);
  expect(scroll).toBeGreaterThan(0);
  const state = await userState(page);
  const event = page
    .locator('.shared-reference-event')
    .filter({ hasText: 'Review this PDF region.' });
  await event.getByRole('button', { name: 'Show in view', exact: true }).click();
  await readyView(page);
  const reference = primary.locator('.observed-reference-panel');
  await expect(reference.getByRole('img', { name: 'Referenced PDF page 1' })).toBeVisible();
  await expect(reference.locator('.pdf-shared-mark')).toContainText('Researcher');
  expect(await userState(page)).toEqual(state);
  await captureWindow(application, info.outputPath('pdf-agent-show.png'));
  await reference.getByRole('button', { name: 'Return to my view' }).click();
  await readyView(page);
  expect(await userState(page)).toEqual(state);
  await expect(primary.getByRole('spinbutton', { name: 'PDF page number' })).toHaveValue('3');
  expect(await primary.locator('.pdf-scroll').evaluate((el) => el.scrollTop)).toBe(scroll);
  await event.getByRole('button', { name: 'Show in view', exact: true }).click();
  await readyView(page);
  await application.evaluate(({ BrowserWindow }) =>
    BrowserWindow.getAllWindows()
      .find((w) => w.webContents.getURL().startsWith('app://gobble/'))!
      .setContentSize(900, 650),
  );
  await expect(primary.locator('.observed-reference-panel')).toBeVisible();
  await expect.poll(() => page.evaluate(() => innerWidth)).toBe(900);
  if (await page.getByRole('button', { name: 'Workspace', exact: true }).isVisible())
    await page.getByRole('button', { name: 'Workspace', exact: true }).click();
  await expect(primary.getByRole('button', { name: 'Return to my view' })).toBeInViewport();
  await captureWindow(application, info.outputPath('pdf-agent-compact.png'));
  await primary.getByRole('button', { name: 'Return to my view' }).click();
  await readyView(page);
  await application.evaluate(({ BrowserWindow }) =>
    BrowserWindow.getAllWindows()
      .find((w) => w.webContents.getURL().startsWith('app://gobble/'))!
      .setContentSize(1440, 900),
  );
  await expect.poll(() => page.evaluate(() => innerWidth)).toBe(1440);
  await writeFile(join(root, 'report.pdf'), makePdf({ pages: 4, rotate: 90 }));
  await primary.getByRole('button', { name: 'Refresh PDF' }).click();
  await readyView(page);
  await event.getByRole('button', { name: 'Show in view', exact: true }).click();
  await expect(page.getByText(/This PDF version is unavailable/)).toBeVisible();
  await expect(primary.locator('.observed-reference-panel')).toHaveCount(0);
  await event.getByRole('button', { name: 'View captured evidence' }).click();
  const capture = page.getByRole('dialog', { name: 'Captured evidence', exact: true });
  await expect(capture.getByRole('img')).toBeVisible();
  const dimensions = await capture
    .getByRole('img')
    .evaluate((img: HTMLImageElement) => ({ width: img.naturalWidth, height: img.naturalHeight }));
  expect(delivery.selected.value.content.image).toMatchObject(dimensions);
  await captureWindow(application, info.outputPath('pdf-agent-capture.png'));
  await capture.getByRole('button', { name: 'Close Captured evidence' }).click();
  await page.getByRole('button', { name: 'Reply', exact: true }).click();
  // Reply selection commits asynchronously. Send only after the composer shows
  // its target, so this checks changed evidence rather than an earlier send.
  await expect(page.getByRole('region', { name: 'Reply target', exact: true })).toBeVisible();
  await page.getByRole('button', { name: 'Send', exact: true }).click();
  await expect(page.getByText(/The question’s evidence changed/).first()).toBeVisible();
  await expect(page.getByRole('textbox', { name: 'Message draft' })).toHaveValue(
    'My next instruction stays local.',
  );
  expect((await document(page)).collaboration!.submissions).toHaveLength(1);
  await page.getByRole('button', { name: 'Cancel reply' }).click();
  const saved = await document(page);
  await application.close();
  ({ application, page } = await launch(profile, { GOBBLE_CODEX_EXECUTABLE: executable }));
  primary = await readyView(page);
  expect((await document(page)).sharedReferences).toEqual(saved.sharedReferences);
  expect((await document(page)).workspace.decisions).toEqual(saved.workspace.decisions);
  await expect(primary.locator('.observed-reference-panel')).toHaveCount(0);
  await expect(page.getByRole('textbox', { name: 'Message draft' })).toHaveValue(
    'My next instruction stays local.',
  );
  await writeFile(info.outputPath('pdf-scripted-delivery.json'), JSON.stringify(delivery, null, 2));
});

test('PDF tool scopes reject scroll-stale, source-preview, off-page, outside-visible and missing receipts', async ({}, info) => {
  test.setTimeout(70_000);
  const primary = await readyView(page);
  await primary.getByRole('combobox', { name: 'PDF zoom' }).selectOption('1.5');
  await readyView(page);
  await begin('pdf-scope');
  await expect
    .poll(async () => {
      try {
        return JSON.parse(await readFile(file('pdf-observed.json'), 'utf8')).value.receipt
          .pointable;
      } catch {
        return false;
      }
    })
    .toBe(true);
  await primary.locator('.pdf-scroll').evaluate((el) => {
    el.scrollTop = 150;
  });
  await expect
    .poll(async () => primary.locator('.pdf-scroll').evaluate((el) => el.scrollTop))
    .toBeGreaterThan(0);
  // A turn that saw the old rectangle cannot publish after the scroll report reaches Main.
  await primary
    .locator('.pdf-scroll')
    .evaluate(
      async () =>
        new Promise((resolve) => requestAnimationFrame(() => requestAnimationFrame(resolve))),
    );
  await focusWindow(application);
  await writeFile(file('pdf-next'), 'ready');
  const delivery = await delivered();
  for (const name of ['stale', 'sourcePoint', 'offPage', 'outside', 'missingReceipt'])
    expect(delivery[name].success, name).toBe(false);
  expect(delivery.fresh.success).toBe(true);
  expect(delivery.source.value.receipt.pointable).toBe(false);
  expect(delivery.current.value.content.visibleRegion).not.toEqual(
    delivery.visible.value.content.visibleRegion,
  );
  await expect(page.locator('.shared-reference-event')).toHaveCount(1);
  await writeFile(info.outputPath('pdf-scope-delivery.json'), JSON.stringify(delivery, null, 2));
});

test('User PDF mark needs no Agent turn; text-only models cannot observe PDF pixels', async () => {
  const primary = await readyView(page);
  await primary.getByRole('button', { name: 'Select region', exact: true }).click();
  await primary.getByRole('group', { name: 'PDF page 1 selection area' }).press('Enter');
  await primary.getByRole('button', { name: 'Share mark from report.pdf' }).click();
  await expect(page.locator('.shared-reference-event')).toHaveCount(1);
  expect((await document(page)).sharedReferences![0]!.author).toEqual({ kind: 'user' });
  expect((await document(page)).collaboration?.submissions ?? []).toHaveLength(0);
  await page
    .getByRole('combobox', { name: 'Message model' })
    .selectOption({ label: 'Fixture text model' });
  await begin('pdf-point');
  await expect(page.locator('.message-state').getByText('failed', { exact: true })).toHaveCount(1);
  await expect(page.getByText(/PDF observations require an image-capable model/)).toBeVisible();
  expect((await document(page)).sharedReferences).toHaveLength(1);
});

test('Agent opens a PDF beside a protected User view without taking focus', async () => {
  const primary = await readyView(page);
  await primary.getByRole('button', { name: 'Close report.pdf', exact: true }).click();
  await page.getByRole('button', { name: 'notes.txt', exact: true }).click();
  await readyView(page);
  await begin('pdf-open');
  await expect(page.locator('.message-state').getByText('completed', { exact: true })).toHaveCount(
    1,
  );
  await readyView(page, 'Secondary pane');
  await expect(primary.getByRole('textbox', { name: 'notes.txt text' })).toBeVisible();
  await expect(page.getByRole('textbox', { name: 'Message draft' })).toBeFocused();
  const result = JSON.parse(await readFile(file('pdf-open.json'), 'utf8'));
  expect(result.views[0]).toMatchObject({ pane: 'secondary' });
  const doc = await document(page);
  expect(doc.workspace.surfaces.find((s) => s.view === 'pdf')?.openedBy.kind).toBe('agent');
});
