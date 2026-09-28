import { test, expect, type ElectronApplication, type Locator, type Page } from '@playwright/test';
import { mkdtemp, rm, copyFile, chmod, readFile, writeFile } from 'node:fs/promises';
import { join, resolve } from 'node:path';
import { tmpdir } from 'node:os';
import { openAgentRoster, chooseProject, launch, projectFixture, readyView } from './support';

let application: ElectronApplication;
let page: Page;
let base: string;
let profile: string;
let executable: string;
test.beforeEach(async () => {
  base = await mkdtemp(join(tmpdir(), 'gobble-agents-electron-'));
  profile = join(base, 'profile');
  executable = join(base, 'codex');
  await copyFile(resolve('desktop/tests/fixtures/codex.cjs'), executable);
  await chmod(executable, 0o700);
  ({ application, page } = await launch(profile, { GOBBLE_CODEX_EXECUTABLE: executable }));
  await chooseProject(application, page, await projectFixture(base));
  // The UI, Electron host, stdio transport, storage and Go service are real.
  // Only the protocol peer and official browser ceremony are test substitutes.
  await application.evaluate(({ shell }) => {
    shell.openExternal = async () => {};
  });
  await page.getByRole('button', { name: 'Account', exact: true }).click();
});
async function signIn() {
  await page.getByRole('button', { name: 'Sign in with ChatGPT' }).click();
  await expect(page.getByText('ChatGPT connected', { exact: true })).toBeVisible();
  await page.getByRole('button', { name: 'Close Codex account' }).click();
}
test.afterEach(async () => {
  await application?.close();
  if (base) await rm(base, { recursive: true, force: true });
});
async function addAgent(name: string) {
  await openAgentRoster(page);
  await page.getByRole('button', { name: 'Add agent', exact: true }).click();
  const dialog = page.getByRole('dialog', { name: 'Add agent', exact: true });
  await dialog.getByLabel('Name', { exact: true }).fill(name);
  await dialog.getByLabel('Role instructions').fill('Help discuss the study.');
  await dialog.getByRole('button', { name: 'Add agent', exact: true }).click();
  await expect(dialog).not.toBeVisible();
}
async function send(name: string, text: string) {
  await page
    .getByRole('combobox', { name: 'Conversation recipient' })
    .selectOption({ label: 'To ' + name });
  await page.getByRole('textbox', { name: 'Message draft' }).fill(text);
  await page.getByRole('button', { name: 'Send', exact: true }).click();
  await expect(page.getByRole('textbox', { name: 'Message draft' })).toHaveValue('');
}
test('two peer agents share panes but keep their streams and Stop controls separate', async ({}, testInfo) => {
  await signIn();
  await page.getByRole('button', { name: 'samples.csv', exact: true }).click();
  await readyView(page);
  await addAgent('Researcher');
  await addAgent('Reviewer');
  await send('Researcher', 'Please wait for Researcher.');
  await expect(page.getByText('Fixture response from Researcher', { exact: true })).toBeVisible();
  await send('Reviewer', 'Please wait for Reviewer.');
  await expect(page.getByText('Fixture response from Reviewer', { exact: true })).toBeVisible();
  await expect(page.getByRole('tab', { name: /samples.csv/ })).toBeVisible();
  await expect(page.getByRole('combobox', { name: 'Message model' })).toHaveValue('fixture-model');
  await page.getByRole('button', { name: 'Stop Researcher', exact: true }).click();
  await expect(
    page.getByRole('button', { name: 'Stop Researcher', exact: true }),
  ).not.toBeVisible();
  await expect(page.getByRole('button', { name: 'Stop Reviewer', exact: true })).toBeVisible();
  await page.screenshot({ path: testInfo.outputPath('agents.png') });
  await page.getByRole('button', { name: 'Stop Reviewer', exact: true }).click();
  await expect(
    page.locator('.message-state').getByText('interrupted', { exact: true }),
  ).toHaveCount(2);
});
test('lost acknowledgment survives restart and Check status recovers the original response without resending', async () => {
  await signIn();
  await addAgent('Researcher');
  await send('Researcher', 'disconnect-after-accept');
  await expect(page.getByText('Delivery uncertain', { exact: true })).toBeVisible();
  await application.close();
  ({ application, page } = await launch(profile, { GOBBLE_CODEX_EXECUTABLE: executable }));
  await expect(page.getByText('disconnect-after-accept', { exact: true })).toBeVisible();
  await page.getByRole('button', { name: 'Account', exact: true }).click();
  await page.getByRole('button', { name: 'Refresh connection' }).click();
  await expect(page.getByText('ChatGPT connected', { exact: true })).toBeVisible();
  await page.getByRole('button', { name: 'Close Codex account' }).click();
  await page.getByRole('button', { name: 'Check status', exact: true }).click();
  await expect(page.getByText('Recovered fixture response.', { exact: true })).toBeVisible();
  const state = JSON.parse(
    await readFile(join(profile, 'codex/home/fixture-state.json'), 'utf8'),
  ) as { threads: Record<string, { turns: unknown[] }> };
  expect(Object.values(state.threads)).toHaveLength(1);
  expect(Object.values(state.threads)[0]?.turns).toHaveLength(1);
});

// Visibility alone cannot detect white text on a pale dialog surface.
async function expectReadable(button: Locator) {
  await expect(button).toBeVisible();
  const contrast = await button.evaluate((element) => {
    const rgb = (color: string) => color.match(/[\d.]+/g)!.map(Number);
    const luminance = (color: number[]) =>
      color.slice(0, 3).reduce((sum, component, index) => {
        const channel = component / 255;
        return (
          sum +
          (channel <= 0.04045 ? channel / 12.92 : ((channel + 0.055) / 1.055) ** 2.4) *
            [0.2126, 0.7152, 0.0722][index]!
        );
      }, 0);
    let background = [255, 255, 255];
    for (let parent: Element | null = element; parent; parent = parent.parentElement) {
      const color = rgb(getComputedStyle(parent).backgroundColor);
      if (color.length === 3 || color[3] === 1) {
        background = color;
        break;
      }
    }
    const a = luminance(rgb(getComputedStyle(element).color));
    const b = luminance(background);
    return (Math.max(a, b) + 0.05) / (Math.min(a, b) + 0.05);
  });
  expect(contrast).toBeGreaterThanOrEqual(4.5);
}
async function expectReadableButtonStates(button: Locator) {
  await page.mouse.move(0, 0);
  await expectReadable(button);
  await button.hover();
  await expectReadable(button);
  await page.mouse.move(0, 0);
  await page.keyboard.press('Tab');
  await button.focus();
  await expect(button).toBeFocused();
  await expectReadable(button);
  expect(
    await button.evaluate((element) => {
      const style = getComputedStyle(element);
      return (
        element.matches(':focus-visible') &&
        style.outlineStyle !== 'none' &&
        parseFloat(style.outlineWidth) >= 2
      );
    }),
  ).toBe(true);
}
test('account and agent actions are readable before hover, on hover and with keyboard focus', async () => {
  await expectReadableButtonStates(page.getByRole('button', { name: 'Sign in with ChatGPT' }));
  await page.keyboard.press('Escape');
  await expect(page.getByRole('dialog')).not.toBeVisible();
  await expect(page.getByRole('button', { name: 'Account', exact: true })).toBeFocused();
  await page.keyboard.press('Enter');
  await signIn();
  await openAgentRoster(page);
  await page.getByRole('button', { name: 'Add agent', exact: true }).click();
  const dialog = page.getByRole('dialog', { name: 'Add agent', exact: true });
  await dialog.getByLabel('Name', { exact: true }).fill('Reviewer');
  await expectReadableButtonStates(dialog.getByRole('button', { name: 'Add agent', exact: true }));
  await dialog.getByRole('button', { name: 'Add agent', exact: true }).click();
  await openAgentRoster(page);
  await page.getByRole('button', { name: 'Settings for Reviewer' }).click();
  await expectReadableButtonStates(page.getByRole('button', { name: 'Save settings' }));
  await page.keyboard.press('Escape');
  await page.getByRole('button', { name: 'Account', exact: true }).click();
  await expectReadableButtonStates(page.getByRole('button', { name: 'Sign out', exact: true }));
  await expectReadableButtonStates(page.getByRole('button', { name: 'Refresh connection' }));
});

test('the shared composer protects IME input and the timeline preserves the reading position', async ({}, testInfo) => {
  await signIn();
  await addAgent('Researcher');
  await addAgent('Reviewer');
  await page
    .getByRole('combobox', { name: 'Conversation recipient' })
    .selectOption({ label: 'To Researcher' });
  const draft = page.getByRole('textbox', { name: 'Message draft' });
  await draft.fill('Composing a message');
  await expect(page.getByRole('button', { name: 'Send', exact: true })).toBeEnabled();
  await draft.dispatchEvent('keydown', {
    key: 'Enter',
    code: 'Enter',
    keyCode: 229,
    isComposing: true,
    bubbles: true,
  });
  await expect(draft).toHaveValue('Composing a message');
  await expect(page.locator('.message-exchange')).toHaveCount(0);
  await draft.press('Shift+Enter');
  await expect(draft).toHaveValue('Composing a message\n');
  await draft.fill(
    'Please wait.\n' + Array.from({ length: 55 }, (_, index) => 'Review item ' + index).join('\n'),
  );
  await expect(page.getByRole('button', { name: 'Send', exact: true })).toBeEnabled();
  await draft.press('Enter');
  await expect(draft).toHaveValue('');
  await expect(page.getByRole('button', { name: 'Stop Researcher' })).toBeVisible();
  const timeline = page.getByLabel('Project messages', { exact: true });
  await timeline.evaluate((element) => {
    element.scrollTop = 0;
    element.dispatchEvent(new Event('scroll'));
  });
  await send('Reviewer', 'Please wait for my next message.');
  await expect(page.getByRole('button', { name: 'New messages' })).toBeVisible();
  expect(await timeline.evaluate((element) => element.scrollTop)).toBeLessThan(2);
  await page.getByRole('button', { name: 'New messages' }).click();
  await expect
    .poll(() =>
      timeline.evaluate(
        (element) => element.scrollHeight - element.scrollTop - element.clientHeight,
      ),
    )
    .toBeLessThan(48);
  await page.getByRole('button', { name: 'Stop Researcher' }).click();
  await page.getByRole('button', { name: 'Stop Reviewer' }).click();
  await page.getByRole('button', { name: 'samples.csv', exact: true }).click();
  await readyView(page);
  await page.getByRole('button', { name: 'Open quality.png in the other pane' }).click();
  await readyView(page, 'Secondary pane');
  await page.screenshot({ path: testInfo.outputPath('05-1-agent-chat.png') });
});

test('interleaved responses retain reading position through growth, collapse, compact mode and restoration', async ({}, info) => {
  await signIn();
  await page.getByRole('button', { name: 'samples.csv', exact: true }).click();
  await readyView(page);
  await page.getByRole('button', { name: 'Open quality.png in the other pane' }).click();
  await readyView(page, 'Secondary pane');
  await addAgent('Researcher');
  await addAgent('Reviewer');
  await send(
    'Researcher',
    'controlled: Review the curve across this deliberately long reference to preserve context while the reply streams.',
  );
  await send('Reviewer', 'controlled: Review the samples');
  const control = join(profile, 'codex/home/stream-control.json');
  const first = 'Reviewer arrived first.\n';
  await writeFile(control, JSON.stringify({ Reviewer: [first] }));
  const responses = page.locator('.chat-agent-response');
  await expect(responses).toHaveCount(1);
  const second =
    'Researcher arrived second.\n' +
    Array.from({ length: 70 }, (_, i) => 'Read this line ' + i).join('\n');
  await writeFile(control, JSON.stringify({ Reviewer: [first], Researcher: [second] }));
  await expect(responses.locator('header strong')).toHaveText(['Reviewer', 'Researcher']);
  const contextLink = responses.nth(1).getByRole('button', { name: /In reply to:/ });
  expect(await contextLink.evaluate((element) => element.scrollWidth <= element.clientWidth)).toBe(
    true,
  );

  const timeline = page.getByLabel('Project messages', { exact: true });
  const anchor = responses.nth(1).locator('..');
  await anchor.evaluate((element) => {
    const list = document.querySelector('.message-list')!;
    list.scrollTop += element.getBoundingClientRect().top - list.getBoundingClientRect().top + 100;
    list.dispatchEvent(new Event('scroll'));
  });
  const offset = () =>
    anchor.evaluate(
      (element) =>
        element.getBoundingClientRect().top -
        document.querySelector('.message-list')!.getBoundingClientRect().top,
    );
  const before = await offset();
  const stable = () => expect.poll(async () => Math.abs((await offset()) - before)).toBeLessThan(2);
  await writeFile(
    control,
    JSON.stringify({
      Reviewer: [first, '\nMore detail above your reading position.\n'.repeat(35)],
      Researcher: [second],
    }),
  );
  await expect(responses.nth(0)).toContainText('More detail above your reading position.');
  await expect(page.getByRole('button', { name: 'New messages' })).toBeVisible();
  await stable();
  await page.getByRole('textbox', { name: 'Message draft' }).fill('Keep this draft while I read.');
  await page.getByRole('button', { name: 'Collapse chat' }).click();
  await expect(page.getByRole('button', { name: 'Show chat' })).toBeFocused();
  await page.getByRole('button', { name: 'Show chat' }).click();
  await stable();
  await application.evaluate(({ BrowserWindow }) =>
    BrowserWindow.getAllWindows()[0]!.setSize(1000, 720),
  );
  await page.getByRole('button', { name: 'Chat', exact: true }).click();
  await stable();
  await expect(page.getByRole('textbox', { name: 'Message draft' })).toHaveValue(
    'Keep this draft while I read.',
  );
  await expect(page.locator('textarea')).toHaveCount(1);
  await page.getByRole('button', { name: 'Workspace', exact: true }).click();
  await page.getByRole('button', { name: 'Chat', exact: true }).click();
  await stable();
  await page.screenshot({ path: info.outputPath('05-5-compact-reading.png') });
  await page.getByRole('button', { name: 'New messages' }).click();
  await expect
    .poll(() =>
      timeline.evaluate(
        (element) => element.scrollHeight - element.scrollTop - element.clientHeight,
      ),
    )
    .toBeLessThan(2);
  await page.getByRole('button', { name: 'Stop Researcher', exact: true }).click();
  await page.getByRole('button', { name: 'Stop Reviewer', exact: true }).click();
  const order = await responses.locator('header strong').allTextContents();
  await application.close();
  ({ application, page } = await launch(profile, { GOBBLE_CODEX_EXECUTABLE: executable }));
  await page.getByRole('button', { name: 'Chat', exact: true }).click();
  await expect(page.locator('.chat-agent-response').locator('header strong')).toHaveText(order);
  await expect(page.getByRole('textbox', { name: 'Message draft' })).toHaveValue(
    'Keep this draft while I read.',
  );
  await page
    .locator('.chat-agent-response')
    .first()
    .getByRole('button', { name: /In reply to:/ })
    .click();
  await expect(
    page.locator('[data-chat-item]').filter({
      has: page.locator('.user-bubble').filter({ hasText: 'controlled: Review the samples' }),
    }),
  ).toBeFocused();
  const persisted = JSON.parse(
    await readFile(join(profile, 'codex/home/fixture-state.json'), 'utf8'),
  );
  expect(
    Object.values(persisted.threads).flatMap((thread) => (thread as { turns: unknown[] }).turns),
  ).toHaveLength(2);
});

test('a refused Send stays visible in compact Chat and preserves its recipient and draft', async ({}, info) => {
  await signIn();
  await addAgent('Researcher');
  await addAgent('Reviewer');
  await addAgent('Editor');
  await send('Researcher', 'Please wait for Researcher.');
  await send('Reviewer', 'Please wait for Reviewer.');
  await application.evaluate(({ BrowserWindow }) =>
    BrowserWindow.getAllWindows()[0]!.setSize(1000, 640),
  );
  await page.getByRole('button', { name: 'Chat', exact: true }).click();
  await page
    .getByRole('combobox', { name: 'Conversation recipient' })
    .selectOption({ label: 'To Editor' });
  const draft = page.getByRole('textbox', { name: 'Message draft' });
  await draft.fill('Keep this unsent review request.');
  await page.getByRole('button', { name: 'Send', exact: true }).click();
  await expect(page.getByRole('alert')).toContainText('Two agents are working.');
  await expect(page.getByRole('alert')).toBeVisible();
  await expect(draft).toHaveValue('Keep this unsent review request.');
  await expect(page.locator('textarea')).toHaveCount(1);
  await expect(page.locator('.message-exchange')).toHaveCount(2);
  await page.screenshot({ path: info.outputPath('05-5-compact-error.png') });
  await page.getByRole('button', { name: 'Dismiss error' }).click();
  await page.getByRole('button', { name: 'Stop Researcher', exact: true }).click();
  await page.getByRole('button', { name: 'Stop Reviewer', exact: true }).click();
  await page.getByRole('button', { name: 'Send', exact: true }).click();
  await expect(
    page.locator('.chat-agent-response').filter({ hasText: 'Fixture response from Editor.' }),
  ).toBeVisible();
  await expect(draft).toHaveValue('');
});

test('compact project chrome and chat-side Agents preserve draft, focus and work across navigation', async ({}, info) => {
  await signIn();
  for (const name of [
    'Researcher',
    'Reviewer',
    'Workspace guide',
    'Evidence reviewer',
    'Question reviewer',
  ])
    await addAgent(name);
  await page.getByRole('button', { name: 'samples.csv', exact: true }).click();
  await readyView(page);
  await page.getByRole('button', { name: 'Open quality.png in the other pane' }).click();
  await readyView(page, 'Secondary pane');
  const draft = page.getByRole('textbox', { name: 'Message draft' });
  const recipient = page.getByRole('combobox', { name: 'Conversation recipient' });
  await recipient.selectOption({ label: 'To Reviewer' });
  await draft.fill('Preserve my review while I inspect the agents.');
  const recipientId = await recipient.inputValue();
  await expect(page.getByText('Shared workspace', { exact: true })).toHaveCount(0);
  await expect(page.locator('.sidebar .agent-row')).toHaveCount(0);
  await expect(page.locator('h1')).toHaveCount(1);
  expect((await page.locator('.project-bar').boundingBox())!.height).toBeLessThanOrEqual(46);
  expect((await page.locator('.chat-header').boundingBox())!.height).toBeLessThanOrEqual(46);
  expect((await page.locator('.sidebar').boundingBox())!.width).toBe(180);
  await page.screenshot({ path: info.outputPath('05-6a-desktop.png') });
  await openAgentRoster(page);
  const roster = page.getByRole('dialog', { name: 'Project agents', exact: true });
  await expect(roster.locator('.agent-row')).toHaveCount(5);
  const rows = await roster.locator('.agent-row').evaluateAll((elements) =>
    elements.map((element) => ({
      height: element.getBoundingClientRect().height,
      labelX: element.querySelector('.agent-identity')!.getBoundingClientRect().left,
      fits: element.scrollWidth <= element.clientWidth,
    })),
  );
  expect(rows.every((row) => row.height === 46 && row.fits)).toBe(true);
  expect(new Set(rows.map((row) => row.labelX)).size).toBe(1);
  await page.screenshot({ path: info.outputPath('05-6a-agents.png') });
  await roster.getByRole('button', { name: 'Settings for Question reviewer', exact: true }).click();
  await page.getByRole('button', { name: 'Close Agent settings', exact: true }).click();
  await expect(page.getByRole('button', { name: 'Project agents', exact: true })).toBeFocused();
  await expect(draft).toHaveValue('Preserve my review while I inspect the agents.');
  await expect(recipient).toHaveValue(recipientId);
  await openAgentRoster(page);
  await roster.getByRole('button', { name: 'Message Evidence reviewer', exact: true }).click();
  await expect(draft).toBeFocused();
  await expect(recipient.locator('option:checked')).toHaveText('To Evidence reviewer');
  await expect(draft).toHaveValue('Preserve my review while I inspect the agents.');
  await expect(page.locator('.message-exchange')).toHaveCount(0);
  await page.getByRole('button', { name: 'Toggle files' }).click();
  await expect(page.getByRole('complementary', { name: 'Project navigation' })).not.toBeVisible();
  await page.getByRole('button', { name: 'Toggle files' }).click();
  await expect(page.getByRole('button', { name: 'samples.csv', exact: true })).toBeVisible();
  await application.evaluate(({ BrowserWindow }) =>
    BrowserWindow.getAllWindows()[0]!.setContentSize(900, 600),
  );
  await page.getByRole('button', { name: 'Chat', exact: true }).click();
  await expect(page.getByRole('button', { name: 'Toggle files' })).toHaveAttribute(
    'aria-expanded',
    'false',
  );
  await page.getByRole('button', { name: 'Toggle files' }).click();
  await expect(page.getByRole('button', { name: 'quality.png', exact: true })).toBeVisible();
  await page.keyboard.press('Escape');
  await expect(page.getByRole('button', { name: 'Toggle files' })).toBeFocused();
  await page.getByRole('button', { name: 'Toggle files' }).click();
  await page.getByRole('button', { name: 'quality.png', exact: true }).click();
  await expect(page.getByRole('button', { name: 'Workspace', exact: true })).toHaveAttribute(
    'aria-pressed',
    'true',
  );
  await expect(page.getByRole('button', { name: 'Toggle files' })).toHaveAttribute(
    'aria-expanded',
    'false',
  );
  await page.getByRole('button', { name: 'Chat', exact: true }).click();
  await expect(draft).toHaveValue('Preserve my review while I inspect the agents.');
  await expect(page.locator('textarea')).toHaveCount(1);
  await expect(page.getByRole('button', { name: 'Account', exact: true })).toBeVisible();
  await page.screenshot({ path: info.outputPath('05-6a-compact.png') });
});

test('empty Agent roster explains connection and returns keyboard focus', async () => {
  await page.getByRole('button', { name: 'Close Codex account' }).click();
  await openAgentRoster(page);
  await expect(
    page.getByText('Sign in through Account to add agents.', { exact: true }),
  ).toBeVisible();
  await expect(page.getByRole('button', { name: 'Add agent', exact: true })).toBeDisabled();
  await page.keyboard.press('Escape');
  await expect(page.getByRole('button', { name: 'Project agents', exact: true })).toBeFocused();
});
