# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: shared-context.spec.ts >> User and Agent share table/image pointers while local selection and composer focus stay independent
- Location: desktop/tests/electron/shared-context.spec.ts:61:1

# Error details

```
Error: expect(locator).toHaveValue(expected) failed

Locator:  getByRole('textbox', { name: 'Message draft' })
Expected: ""
Received: "shared-pointt"
Timeout:  5000ms

Call log:
  - Expect "toHaveValue" getByRole('textbox', { name: 'Message draft' }) with timeout 5000ms
  - waiting for getByRole('textbox', { name: 'Message draft' })
    14 × locator resolved to <textarea rows="3" maxlength="16000" aria-label="Message draft" placeholder="Write a message for your agent…">shared-pointt</textarea>
       - unexpected value "shared-pointt"

```

```yaml
- textbox "Message draft":
  - /placeholder: Write a message for your agent…
  - text: shared-pointt
```

# Test source

```ts
  1   | import {
  2   |   openPaneActions,
  3   |   focusWindow,
  4   |   openAgentRoster,
  5   |   launch,
  6   |   chooseProject,
  7   |   projectFixture,
  8   |   readyView,
  9   | } from './support';
  10  | import { test, expect, type ElectronApplication, type Page } from '@playwright/test';
  11  | import { mkdtemp, rm, copyFile, chmod, writeFile } from 'node:fs/promises';
  12  | import { join, resolve } from 'node:path';
  13  | import { tmpdir } from 'node:os';
  14  | 
  15  | let application: ElectronApplication;
  16  | let page: Page;
  17  | let base: string;
  18  | let profile: string;
  19  | let executable: string;
  20  | let root: string;
  21  | test.beforeEach(async () => {
  22  |   base = await mkdtemp(join(tmpdir(), 'gobble-shared-electron-'));
  23  |   profile = join(base, 'profile');
  24  |   executable = join(base, 'codex');
  25  |   await copyFile(resolve('desktop/tests/fixtures/codex.cjs'), executable);
  26  |   await chmod(executable, 0o700);
  27  |   ({ application, page } = await launch(profile, { GOBBLE_CODEX_EXECUTABLE: executable }));
  28  |   root = await projectFixture(base);
  29  |   await chooseProject(application, page, root);
  30  |   await application.evaluate(({ shell }) => {
  31  |     shell.openExternal = async () => {};
  32  |   });
  33  |   await page.getByRole('button', { name: 'Account', exact: true }).click();
  34  |   await page.getByRole('button', { name: 'Sign in with ChatGPT' }).click();
  35  |   await expect(page.getByText('ChatGPT connected', { exact: true })).toBeVisible();
  36  |   await page.getByRole('button', { name: 'Close Codex account' }).click();
  37  | });
  38  | test.afterEach(async () => {
  39  |   await application?.close();
  40  |   if (base) await rm(base, { recursive: true, force: true });
  41  | });
  42  | async function agent(shared = true) {
  43  |   await openAgentRoster(page);
  44  |   await page.getByRole('button', { name: 'Add agent', exact: true }).click();
  45  |   const dialog = page.getByRole('dialog', { name: 'Add agent', exact: true });
  46  |   await dialog.getByLabel('Name', { exact: true }).fill('Researcher');
  47  |   if (shared) await dialog.getByLabel('Project access').selectOption('sharedViews');
  48  |   await dialog.getByRole('button', { name: 'Add agent', exact: true }).click();
  49  |   await expect(dialog).not.toBeVisible();
  50  |   await page
  51  |     .getByRole('combobox', { name: 'Conversation recipient' })
  52  |     .selectOption({ label: 'To Researcher' });
  53  | }
  54  | async function send(text: string) {
  55  |   await focusWindow(application);
  56  |   await expect(page.locator('dialog[open]')).toHaveCount(0);
  57  |   await page.getByRole('textbox', { name: 'Message draft' }).fill(text);
  58  |   await page.getByRole('button', { name: 'Send', exact: true }).click();
> 59  |   await expect(page.getByRole('textbox', { name: 'Message draft' })).toHaveValue('');
      |                                                                      ^ Error: expect(locator).toHaveValue(expected) failed
  60  | }
  61  | test('User and Agent share table/image pointers while local selection and composer focus stay independent', async ({}, info) => {
  62  |   await page.getByRole('button', { name: 'samples.csv', exact: true }).click();
  63  |   await readyView(page);
  64  |   await page.getByRole('checkbox', { name: /Select row 3:/ }).check();
  65  |   await page.getByRole('button', { name: 'Share mark from samples.csv' }).click();
  66  |   await expect(page.locator('.shared-reference-event')).toHaveCount(1);
  67  |   await expect(page.locator('.shared-cell')).toHaveCount(4);
  68  |   await page.getByRole('button', { name: 'quality.png', exact: true }).click();
  69  |   await readyView(page);
  70  |   await openPaneActions(page, 'quality.png');
  71  |   await page.getByRole('button', { name: 'Move quality.png to other pane' }).click();
  72  |   await readyView(page);
  73  |   await readyView(page, 'Secondary pane');
  74  |   await agent();
  75  |   await send('shared-point');
  76  |   await page.getByRole('textbox', { name: 'Message draft' }).fill('Keep this draft local.');
  77  |   await expect(page.locator('.shared-reference-event')).toHaveCount(3);
  78  |   await expect(page.locator('.message-state').getByText('completed', { exact: true })).toHaveCount(
  79  |     1,
  80  |   );
  81  |   await expect(page.locator('.shared-image-mark')).toHaveCount(1);
  82  |   await expect(page.getByRole('checkbox', { name: /Select row 3:/ })).toBeChecked();
  83  |   await expect(page.getByRole('textbox', { name: 'Message draft' })).toHaveValue(
  84  |     'Keep this draft local.',
  85  |   );
  86  |   await expect(page.getByRole('textbox', { name: 'Message draft' })).toBeFocused();
  87  |   const imageEvent = page
  88  |     .getByRole('article', { name: 'Researcher shared a reference' })
  89  |     .filter({ hasText: 'quality.png' });
  90  |   await page.getByRole('button', { name: 'Close quality.png', exact: true }).click();
  91  |   await expect(imageEvent).toContainText('quality.png');
  92  |   await imageEvent.getByRole('button', { name: 'Reveal', exact: true }).click();
  93  |   await readyView(page, 'Secondary pane');
  94  |   await expect(page.locator('.shared-image-mark')).toHaveCount(1);
  95  |   await page.screenshot({ path: info.outputPath('shared-pointing.png') });
  96  |   await application.close();
  97  |   ({ application, page } = await launch(profile, { GOBBLE_CODEX_EXECUTABLE: executable }));
  98  |   await expect(page.locator('.shared-reference-event')).toHaveCount(3);
  99  |   await expect(page.getByRole('textbox', { name: 'Message draft' })).toHaveValue(
  100 |     'Keep this draft local.',
  101 |   );
  102 |   const ownMark = page.getByRole('article', { name: 'You shared a reference', exact: true });
  103 |   await ownMark.getByText('Mark options', { exact: true }).click();
  104 |   await ownMark.getByRole('button', { name: 'Retract mark', exact: true }).click();
  105 |   await expect(ownMark).toContainText('Reference retracted · history preserved');
  106 |   await expect(page.locator('.shared-reference-event')).toHaveCount(3);
  107 | });
  108 | test('text pointing overlays a revision-bound range without replacing native local selection', async () => {
  109 |   await page.getByRole('button', { name: 'notes.txt', exact: true }).click();
  110 |   await readyView(page);
  111 |   const text = page.getByRole('textbox', { name: 'notes.txt text' });
  112 |   await text.focus();
  113 |   await text.press('ControlOrMeta+a');
  114 |   await text.press('ArrowLeft');
  115 |   for (let index = 0; index < 7; index++) await text.press('Shift+ArrowRight');
  116 |   expect(
  117 |     await text.evaluate((element: HTMLTextAreaElement) => [
  118 |       element.selectionStart,
  119 |       element.selectionEnd,
  120 |     ]),
  121 |   ).toEqual([0, 7]);
  122 |   await page.getByRole('button', { name: 'Use text selection' }).click();
  123 |   await page.getByRole('button', { name: 'Share mark from notes.txt' }).click();
  124 |   await expect(page.locator('.text-highlights mark')).toHaveText('Inspect');
  125 |   await agent();
  126 |   await send('shared-point');
  127 |   await expect(page.locator('.shared-reference-event')).toHaveCount(2);
  128 |   expect(
  129 |     await text.evaluate((element: HTMLTextAreaElement) => [
  130 |       element.selectionStart,
  131 |       element.selectionEnd,
  132 |     ]),
  133 |   ).toEqual([0, 7]);
  134 |   await page
  135 |     .locator('.shared-reference-event')
  136 |     .first()
  137 |     .getByRole('button', { name: 'Reveal', exact: true })
  138 |     .click();
  139 |   await writeFile(join(root, 'notes.txt'), 'A changed file.\nOriginal selection is stale.\n');
  140 |   await openPaneActions(page, 'notes.txt');
  141 |   await page.getByRole('button', { name: 'Refresh notes.txt' }).click();
  142 |   await readyView(page);
  143 |   await expect(page.locator('.text-highlights mark')).toHaveCount(0);
  144 |   await expect(
  145 |     page.getByText('This shared reference targets an older version.', { exact: false }),
  146 |   ).toBeVisible();
  147 | });
  148 | test('explicit shared-view opt-in starts a new conversation and retains earlier messages', async () => {
  149 |   await agent(false);
  150 |   await send('Earlier message');
  151 |   await expect(page.locator('.message-state').getByText('completed', { exact: true })).toHaveCount(
  152 |     1,
  153 |   );
  154 |   await openAgentRoster(page);
  155 |   await page.getByRole('button', { name: 'Settings for Researcher' }).click();
  156 |   await expect(page.getByText('Messages only', { exact: true })).toBeVisible();
  157 |   await page
  158 |     .getByRole('button', { name: 'Enable shared views and start new conversation', exact: true })
  159 |     .click();
```