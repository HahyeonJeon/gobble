import { focusWindow } from './support';
import { expect, test } from '@playwright/test';
import { readFile, rm, writeFile } from 'node:fs/promises';
import { join } from 'node:path';
import {
  launch,
  chooseProject,
  readyView,
  openAgentRoster,
  openPaneActions,
  captureWindow,
} from './support';
import { runtimeFixture } from './runtime-fixture';

test('Agent Run and stream references preserve User work and separate historical evidence from current tasks', async ({}, info) => {
  test.setTimeout(90_000);
  const fixture = await runtimeFixture();
  const { directory, root, profile, environment, monitor, monitorPath } = fixture;
  // A scrollable synthetic stream also includes identical text across stdout/stderr.
  monitor.logs[0]!.stderr_tail +=
    '\n' + Array.from({ length: 65 }, (_, i) => 'Diagnostic line ' + i).join('\n');
  await writeFile(monitorPath, JSON.stringify(monitor));
  let { application, page } = await launch(profile, environment);
  try {
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
    await page.getByRole('button', { name: 'Attach', exact: true }).click();
    await page.getByRole('button', { name: 'existing', exact: true }).click();
    const primary = await readyView(page);
    await primary.getByRole('searchbox', { name: 'Search tasks' }).fill('S03');
    await expect(primary.getByRole('radio')).toHaveCount(1);
    await primary.getByRole('radio').click();
    await primary.getByRole('button', { name: 'Open logs', exact: true }).click();
    const secondary = await readyView(page, 'Secondary pane');
    // Capture the displayed stderr range before selecting a different local range.
    await secondary.getByRole('button', { name: 'stderr', exact: true }).click();
    const stdout = secondary.getByRole('textbox', { name: 'stderr log text' });
    await stdout.focus();
    await stdout.press('ControlOrMeta+Home');
    await stdout.press('Home');
    await stdout.press('Shift+End');
    await secondary.getByRole('button', { name: 'Discuss selection', exact: true }).click();
    await secondary.getByRole('button', { name: 'stderr', exact: true }).click();
    const stderr = secondary.getByRole('textbox', { name: 'stderr log text' });
    await stderr.focus();
    await stderr.press('ControlOrMeta+End');
    await stderr.press('Home');
    await stderr.press('Shift+End');
    const position = () =>
      stderr.evaluate((el: HTMLTextAreaElement) => ({
        start: el.selectionStart,
        end: el.selectionEnd,
        top: el.scrollTop,
        left: el.scrollLeft,
      }));
    const before = await position();
    expect(before.top).toBeGreaterThan(0);
    await primary.getByRole('searchbox').fill('');
    await expect(primary.getByRole('radio')).toHaveCount(4);
    await expect(primary.getByRole('button', { name: 'Discuss task', exact: true })).toBeEnabled();
    const draft = page.getByRole('textbox', { name: 'Message draft' });
    await draft.fill('observed-point');
    await page.getByRole('button', { name: 'Send', exact: true }).click();
    await expect(draft).toHaveValue('');
    await draft.fill('Keep this draft local.');
    await focusWindow(application);
    await writeFile(join(profile, 'codex/home/observed-gate'), 'ready');
    await expect(page.locator('.shared-reference-event')).toHaveCount(2);
    await expect(
      page.locator('.message-state').getByText('completed', { exact: true }),
    ).toHaveCount(1);
    await expect(draft).toBeFocused();
    expect(await position()).toEqual(before);
    await expect(primary.getByRole('searchbox')).toHaveValue('');
    await primary.getByRole('searchbox').fill('S03');
    await expect(primary.getByRole('radio')).toHaveCount(1);
    const runEvent = page
      .locator('.shared-reference-event')
      .filter({ hasText: 'Check the preparation attempt.' });
    const logEvent = page
      .locator('.shared-reference-event')
      .filter({ hasText: 'The output says ERROR 😀 exact.' });
    await runEvent.getByRole('button', { name: 'Show in view', exact: true }).click();
    await readyView(page);
    await expect(primary.locator('[data-target="true"]')).toContainText('Prepare samples');
    await primary.getByRole('button', { name: 'Return to my view' }).click();
    await readyView(page);
    await expect(primary.getByRole('searchbox')).toHaveValue('S03');
    await expect(primary.getByRole('radio')).toBeChecked();
    await logEvent.getByRole('button', { name: 'Show in view', exact: true }).click();
    await readyView(page, 'Secondary pane');
    await expect(secondary.locator('.observed-reference-text mark')).toHaveText('ERROR 😀 exact');
    await openPaneActions(secondary, 'align:S03 · attempt 2');
    await expect(
      secondary.getByRole('button', { name: 'Refresh align:S03 · attempt 2', exact: true }),
    ).toBeDisabled();
    await page.keyboard.press('Escape');
    await captureWindow(application, info.outputPath('r3a3-reference.png'));
    await secondary.getByRole('button', { name: 'Return to my view' }).click();
    await readyView(page, 'Secondary pane');
    await expect(stderr).toBeFocused();
    expect(await position()).toEqual(before);
    await expect(draft).toHaveValue('Keep this draft local.');
    await captureWindow(application, info.outputPath('r3a3-return.png'));
    for (const size of [
      { width: 900, height: 650, zoom: 1, name: 'compact' },
      { width: 1280, height: 840, zoom: 1.5, name: 'zoom-150' },
    ]) {
      await logEvent.getByRole('button', { name: 'Show in view', exact: true }).click();
      await application.evaluate(({ BrowserWindow }, size) => {
        const window = BrowserWindow.getAllWindows()[0]!;
        window.setSize(size.width, size.height);
        window.webContents.setZoomFactor(size.zoom);
      }, size);
      if (await page.getByRole('button', { name: 'Workspace', exact: true }).isVisible())
        await page.getByRole('button', { name: 'Workspace', exact: true }).click();
      if (await page.getByRole('button', { name: 'Secondary pane', exact: true }).isVisible())
        await page.getByRole('button', { name: 'Secondary pane', exact: true }).click();
      await readyView(page, 'Secondary pane');
      await expect(secondary.getByRole('button', { name: 'Return to my view' })).toBeInViewport();
      await expect(secondary.locator('.observed-reference-text mark')).toBeInViewport();
      await captureWindow(application, info.outputPath('r3a3-' + size.name + '.png'));
      await secondary.getByRole('button', { name: 'Return to my view' }).click();
      await application.evaluate(({ BrowserWindow }) => {
        const window = BrowserWindow.getAllWindows()[0]!;
        window.webContents.setZoomFactor(1);
        window.setSize(1280, 840);
      });
      await readyView(page, 'Secondary pane');
    }
    // Refresh new bytes, refuse the old pointer, and open the existing sent snapshot.
    monitor.logs[0]!.stderr_tail = 'New output';
    await writeFile(monitorPath, JSON.stringify(monitor));
    await secondary.getByRole('button', { name: 'Refresh', exact: true }).click();
    await readyView(page, 'Secondary pane');
    await logEvent.getByRole('button', { name: 'Show in view', exact: true }).click();
    await expect(page.getByText(/This pointer targets an older observation/)).toBeVisible();
    await expect(secondary.getByRole('button', { name: 'Return to my view' })).toHaveCount(0);
    await logEvent.getByRole('button', { name: 'View captured evidence' }).click();
    const capture = page.getByRole('dialog', { name: 'Captured evidence', exact: true });
    await expect(capture.locator('pre')).toHaveText('ERROR 😀 exact');
    await captureWindow(application, info.outputPath('r3a3-historical.png'));
    await capture.getByRole('button', { name: 'Close Captured evidence' }).click();
    monitor.tasks[1]!.attempt = 3;
    await writeFile(monitorPath, JSON.stringify(monitor));
    await logEvent.getByText('Mark options', { exact: true }).click();
    await logEvent.getByRole('button', { name: 'Find current task' }).click();
    await readyView(page);
    await expect(primary.getByRole('radio')).toHaveAccessibleName(
      'Align reads · align:S03 · Attempt 3',
    );
    const delivery = JSON.parse(
      await readFile(join(profile, 'codex/home/observed-delivery.json'), 'utf8'),
    );
    expect(delivery.receipts[1].observed.content).toMatchObject({
      stream: 'stderr',
      displayedStream: 'stderr',
      text: 'ERROR 😀 exact',
    });
    await writeFile(
      info.outputPath('r3a3-scripted-delivery.json'),
      JSON.stringify(delivery, null, 2),
    );
    await application.close();
    ({ application, page } = await launch(profile, environment));
    await expect(page.locator('.shared-reference-event')).toHaveCount(2);
    await expect(page.getByRole('button', { name: 'Return to my view' })).toHaveCount(0);
    await expect(page.getByRole('textbox', { name: 'Message draft' })).toHaveValue(
      'Keep this draft local.',
    );
  } catch (error) {
    await writeFile(info.outputPath('failure-ui.txt'), await page.locator('body').innerText());
    throw error;
  } finally {
    await application.close();
    await rm(directory, { recursive: true, force: true });
  }
});
