import {
  openPaneActions,
  focusWindow,
  openAgentRoster,
  launch,
  chooseProject,
  projectFixture,
  readyView,
} from './support';
import { test, expect, type ElectronApplication, type Page } from '@playwright/test';
import { mkdtemp, rm, copyFile, chmod, writeFile } from 'node:fs/promises';
import { join, resolve } from 'node:path';
import { tmpdir } from 'node:os';

let application: ElectronApplication;
let page: Page;
let base: string;
let profile: string;
let executable: string;
let root: string;
test.beforeEach(async () => {
  base = await mkdtemp(join(tmpdir(), 'gobble-shared-electron-'));
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
});
test.afterEach(async () => {
  await application?.close();
  if (base) await rm(base, { recursive: true, force: true });
});
async function agent(shared = true) {
  await openAgentRoster(page);
  await page.getByRole('button', { name: 'Add agent', exact: true }).click();
  const dialog = page.getByRole('dialog', { name: 'Add agent', exact: true });
  await dialog.getByLabel('Name', { exact: true }).fill('Researcher');
  if (shared) await dialog.getByLabel('Project access').selectOption('sharedViews');
  await dialog.getByRole('button', { name: 'Add agent', exact: true }).click();
  await expect(dialog).not.toBeVisible();
  await page
    .getByRole('combobox', { name: 'Conversation recipient' })
    .selectOption({ label: 'To Researcher' });
}
async function send(text: string) {
  await focusWindow(application);
  await expect(page.locator('dialog[open]')).toHaveCount(0);
  await page.getByRole('textbox', { name: 'Message draft' }).fill(text);
  await page.getByRole('button', { name: 'Send', exact: true }).click();
  await expect(page.getByRole('textbox', { name: 'Message draft' })).toHaveValue('');
}
test('User and Agent share table/image pointers while local selection and composer focus stay independent', async ({}, info) => {
  await page.getByRole('button', { name: 'samples.csv', exact: true }).click();
  await readyView(page);
  await page.getByRole('checkbox', { name: /Select row 3:/ }).check();
  await page.getByRole('button', { name: 'Share mark from samples.csv' }).click();
  await expect(page.locator('.shared-reference-event')).toHaveCount(1);
  await expect(page.locator('.shared-cell')).toHaveCount(4);
  await page.getByRole('button', { name: 'quality.png', exact: true }).click();
  await readyView(page);
  await openPaneActions(page, 'quality.png');
  await page.getByRole('button', { name: 'Move quality.png to other pane' }).click();
  await readyView(page);
  await readyView(page, 'Secondary pane');
  await agent();
  await send('shared-point');
  await page.getByRole('textbox', { name: 'Message draft' }).fill('Keep this draft local.');
  await expect(page.locator('.shared-reference-event')).toHaveCount(3);
  await expect(page.locator('.message-state').getByText('completed', { exact: true })).toHaveCount(
    1,
  );
  await expect(page.locator('.shared-image-mark')).toHaveCount(1);
  await expect(page.getByRole('checkbox', { name: /Select row 3:/ })).toBeChecked();
  await expect(page.getByRole('textbox', { name: 'Message draft' })).toHaveValue(
    'Keep this draft local.',
  );
  await expect(page.getByRole('textbox', { name: 'Message draft' })).toBeFocused();
  const imageEvent = page
    .getByRole('article', { name: 'Researcher shared a reference' })
    .filter({ hasText: 'quality.png' });
  await page.getByRole('button', { name: 'Close quality.png', exact: true }).click();
  await expect(imageEvent).toContainText('quality.png');
  await imageEvent.getByRole('button', { name: 'Reveal', exact: true }).click();
  await readyView(page, 'Secondary pane');
  await expect(page.locator('.shared-image-mark')).toHaveCount(1);
  await page.screenshot({ path: info.outputPath('shared-pointing.png') });
  await application.close();
  ({ application, page } = await launch(profile, { GOBBLE_CODEX_EXECUTABLE: executable }));
  await expect(page.locator('.shared-reference-event')).toHaveCount(3);
  await expect(page.getByRole('textbox', { name: 'Message draft' })).toHaveValue(
    'Keep this draft local.',
  );
  const ownMark = page.getByRole('article', { name: 'You shared a reference', exact: true });
  await ownMark.getByText('Mark options', { exact: true }).click();
  await ownMark.getByRole('button', { name: 'Retract mark', exact: true }).click();
  await expect(ownMark).toContainText('Reference retracted · history preserved');
  await expect(page.locator('.shared-reference-event')).toHaveCount(3);
});
test('text pointing overlays a revision-bound range without replacing native local selection', async () => {
  await page.getByRole('button', { name: 'notes.txt', exact: true }).click();
  await readyView(page);
  const text = page.getByRole('textbox', { name: 'notes.txt text' });
  await text.focus();
  await text.press('ControlOrMeta+a');
  await text.press('ArrowLeft');
  for (let index = 0; index < 7; index++) await text.press('Shift+ArrowRight');
  expect(
    await text.evaluate((element: HTMLTextAreaElement) => [
      element.selectionStart,
      element.selectionEnd,
    ]),
  ).toEqual([0, 7]);
  await page.getByRole('button', { name: 'Use text selection' }).click();
  await page.getByRole('button', { name: 'Share mark from notes.txt' }).click();
  await expect(page.locator('.text-highlights mark')).toHaveText('Inspect');
  await agent();
  await send('shared-point');
  await expect(page.locator('.shared-reference-event')).toHaveCount(2);
  expect(
    await text.evaluate((element: HTMLTextAreaElement) => [
      element.selectionStart,
      element.selectionEnd,
    ]),
  ).toEqual([0, 7]);
  await page
    .locator('.shared-reference-event')
    .first()
    .getByRole('button', { name: 'Reveal', exact: true })
    .click();
  await writeFile(join(root, 'notes.txt'), 'A changed file.\nOriginal selection is stale.\n');
  await openPaneActions(page, 'notes.txt');
  await page.getByRole('button', { name: 'Refresh notes.txt' }).click();
  await readyView(page);
  await expect(page.locator('.text-highlights mark')).toHaveCount(0);
  await expect(
    page.getByText('This shared reference targets an older version.', { exact: false }),
  ).toBeVisible();
});
test('explicit shared-view opt-in starts a new conversation and retains earlier messages', async () => {
  await agent(false);
  await send('Earlier message');
  await expect(page.locator('.message-state').getByText('completed', { exact: true })).toHaveCount(
    1,
  );
  await openAgentRoster(page);
  await page.getByRole('button', { name: 'Settings for Researcher' }).click();
  await expect(page.getByText('Messages only', { exact: true })).toBeVisible();
  await page
    .getByRole('button', { name: 'Enable shared views and start new conversation', exact: true })
    .click();
  await expect(page.getByRole('dialog')).not.toBeVisible();
  await expect(page.getByText('Earlier message', { exact: true })).toBeVisible();
  await openAgentRoster(page);
  await page.getByRole('button', { name: 'Settings for Researcher' }).click();
  await expect(page.getByText('Shared views enabled', { exact: true })).toBeVisible();
  await page.getByRole('button', { name: 'Disable shared views and stop' }).click();
  await expect(page.getByRole('dialog')).not.toBeVisible();
});
