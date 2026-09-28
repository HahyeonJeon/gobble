import {
  captureWindow,
  openPaneActions,
  openAgentRoster,
  chooseProject,
  launch,
  readyView,
  attachmentPreview,
  closeAttachmentPreview,
} from './support';
import { expect, test } from '@playwright/test';
import { readFile, rm, writeFile } from 'node:fs/promises';
import { join } from 'node:path';
import { runtimeFixture } from './runtime-fixture';

test('task navigation and stream selections survive refresh failures and normal restart', async ({}, info) => {
  test.setTimeout(90_000);
  const { directory, root, profile, monitorPath, monitor, environment } = await runtimeFixture();
  let { application, page } = await launch(profile, environment);
  try {
    await chooseProject(application, page, root);
    // Substitute only the provider executable; account UI and recipient configuration are real.
    await application.evaluate(({ shell }) => {
      shell.openExternal = async () => {};
    });
    await page.getByRole('button', { name: 'Account', exact: true }).click();
    await page.getByRole('button', { name: 'Sign in with ChatGPT' }).click();
    await expect(page.getByText('ChatGPT connected', { exact: true })).toBeVisible();
    await page.getByRole('button', { name: 'Close Codex account' }).click();
    await openAgentRoster(page);
    await page.getByRole('button', { name: 'Add agent', exact: true }).click();
    const agent = page.getByRole('dialog', { name: 'Add agent', exact: true });
    await agent.getByLabel('Name', { exact: true }).fill('Researcher');
    await agent.getByRole('button', { name: 'Add agent', exact: true }).click();
    await expect(agent).not.toBeVisible();
    await page.getByRole('button', { name: 'Attach', exact: true }).click();
    await page.getByRole('button', { name: 'existing', exact: true }).click();
    const primary = await readyView(page);
    await primary
      .getByRole('radio', { name: 'Align template · align-template · Attempt 0', exact: true })
      .click();
    await expect(primary.getByRole('button', { name: 'Open logs', exact: true })).toBeDisabled();
    await expect(
      primary
        .getByText('Template task · No attempt logs', { exact: true })
        .filter({ visible: true }),
    ).toBeVisible();
    await primary
      .getByRole('radio', { name: 'Align reads · align:S04 · Attempt 0', exact: true })
      .click();
    await expect(
      primary.getByText('No attempt has started', { exact: true }).filter({ visible: true }),
    ).toBeVisible();
    await primary.getByRole('searchbox', { name: 'Search tasks' }).fill('S03');
    await expect(primary.getByRole('radio')).toHaveCount(1);
    await primary
      .getByRole('radio', { name: 'Align reads · align:S03 · Attempt 2', exact: true })
      .click();
    await primary.getByRole('button', { name: 'Open logs', exact: true }).click();
    const secondary = await readyView(page, 'Secondary pane');
    const errorLog = secondary.getByRole('textbox', { name: 'stderr log text' });
    await expect(errorLog).toHaveValue('ERROR 😀 exact\nShared line\nstderr only');
    await errorLog.dblclick({ position: { x: 26, y: 17 } });
    expect(
      await errorLog.evaluate((input: HTMLTextAreaElement) =>
        input.value.slice(input.selectionStart, input.selectionEnd),
      ),
    ).toBe('ERROR');
    await errorLog.focus();
    await errorLog.press('ControlOrMeta+Home');
    await errorLog.press('Shift+End');
    await secondary.getByRole('button', { name: 'Discuss selection', exact: true }).click();
    await expect(page.getByRole('textbox', { name: 'Message draft' })).toBeFocused();
    await secondary.getByRole('button', { name: 'stdout', exact: true }).click();
    const log = secondary.getByRole('textbox', { name: 'stdout log text' });
    await expect(log).toHaveValue(/Read sample S03/);
    await log.focus();
    await log.press('ControlOrMeta+End');
    await log.press('Home');
    await log.press('Shift+End');
    expect(
      await log.evaluate((element: HTMLTextAreaElement) =>
        element.value.slice(element.selectionStart, element.selectionEnd),
      ),
    ).toBe('Alignment complete');
    await secondary.getByRole('button', { name: 'Discuss selection', exact: true }).click();
    await page
      .getByRole('combobox', { name: 'Conversation recipient' })
      .selectOption({ label: 'To Researcher' });
    await expect(
      page.getByText('2 attachments · Ready for Researcher', { exact: true }),
    ).toBeVisible();
    await page.screenshot({ path: info.outputPath('r3a2-task-log-chat.png') });
    await openPaneActions(secondary, 'align:S03 · attempt 2');
    await secondary
      .getByRole('button', { name: 'Move align:S03 · attempt 2 to other pane' })
      .click();
    await primary.getByRole('tab', { name: 'existing', exact: true }).click();
    await readyView(page);
    await primary.getByRole('button', { name: 'Open logs', exact: true }).click();
    await readyView(page, 'Secondary pane');
    await expect(primary.getByRole('tab', { name: 'existing', exact: true })).toBeVisible();
    await expect(secondary.getByRole('tab', { name: /align:S03/ })).toHaveCount(1);
    // Refresh failure keeps the prior text, stream and selection on screen.
    await writeFile(monitorPath, '{}');
    await secondary.getByRole('button', { name: 'Refresh', exact: true }).click();
    await expect(
      secondary.getByText(/Refresh failed · Showing previous observation/),
    ).toBeVisible();
    await expect(log).toHaveValue(/Alignment complete/);
    await secondary.getByRole('button', { name: 'stderr', exact: true }).click();
    await expect(
      secondary.getByText(/Refresh failed · Showing previous observation/),
    ).toBeVisible();
    await secondary.getByRole('button', { name: 'stdout', exact: true }).click();
    await expect(
      secondary.getByRole('button', { name: 'Discuss selection', exact: true }),
    ).toBeEnabled();
    await page.screenshot({ path: info.outputPath('r3a2-refresh-failed.png') });
    await writeFile(monitorPath, JSON.stringify(monitor));
    await secondary.getByRole('button', { name: 'Refresh', exact: true }).click();
    await expect(secondary.getByText(/Refresh failed/)).toHaveCount(0);
    // An updated stream invalidates local coordinates without changing either captured draft.
    monitor.logs[0]!.stdout_tail = 'New output from the same attempt';
    monitor.logs[0]!.stderr_tail = '';
    await writeFile(monitorPath, JSON.stringify(monitor));
    await secondary.getByRole('button', { name: 'Refresh', exact: true }).click();
    await expect(log).toHaveValue('New output from the same attempt');
    await expect(
      secondary.getByRole('button', { name: 'Discuss selection', exact: true }),
    ).toBeDisabled();
    await secondary.getByRole('button', { name: 'stderr · empty', exact: true }).click();
    await expect(
      secondary.getByText('No text returned for stderr. Completeness is unknown.', { exact: true }),
    ).toBeVisible();
    await secondary.getByRole('button', { name: 'stdout', exact: true }).click();
    for (const size of [
      { width: 900, height: 650, zoom: 1, name: 'compact' },
      { width: 1280, height: 840, zoom: 1.5, name: 'zoom-150' },
    ]) {
      await application.evaluate(({ BrowserWindow }, size) => {
        const window = BrowserWindow.getAllWindows()[0]!;
        window.setSize(size.width, size.height);
        window.webContents.setZoomFactor(size.zoom);
      }, size);
      await page.getByRole('button', { name: 'Workspace', exact: true }).click();
      await readyView(page, 'Secondary pane');
      await expect(
        secondary.getByRole('button', { name: 'Refresh', exact: true }),
      ).toBeInViewport();
      await expect(secondary.getByRole('textbox', { name: 'stdout log text' })).toBeInViewport();
      await captureWindow(application, info.outputPath('r3a2-' + size.name + '.png'));
      const primarySwitch = page.getByRole('button', { name: 'Primary pane', exact: true });
      if (await primarySwitch.isVisible()) await primarySwitch.click();
      await readyView(page);
      await expect(primary.getByRole('radio')).toHaveCount(1);
      await expect(
        primary.getByRole('button', { name: 'Discuss task', exact: true }),
      ).toBeInViewport();
      await captureWindow(application, info.outputPath('r3a2-tasks-' + size.name + '.png'));
      const secondarySwitch = page.getByRole('button', { name: 'Secondary pane', exact: true });
      if (await secondarySwitch.isVisible()) await secondarySwitch.click();
    }
    await application.evaluate(({ BrowserWindow }) => {
      const window = BrowserWindow.getAllWindows()[0]!;
      window.webContents.setZoomFactor(1);
      window.setSize(1280, 840);
    });
    const attached = await page.evaluate(async () => {
      const projects = await window.gobble.projects.list();
      if (!projects.ok) throw new Error(projects.error.message);
      const result = await window.gobble.workspace.read({
        projectId: projects.value[0]!.projectId,
      });
      if (!result.ok) throw new Error(result.error.message);
      return result.value.chat.attachments![1]!;
    });
    expect(attached.capture).toBeDefined();
    // Replace the source after capture: attempt 2 can no longer be queried from this runtime.
    monitor.snapshot = 'fixture-revision-5';
    monitor.tasks[1]!.attempt = 3;
    monitor.logs[0]!.stdout_tail = 'Different attempt, different output';
    await writeFile(monitorPath, JSON.stringify(monitor));
    await application.close();
    ({ application, page } = await launch(profile, environment));
    await page.getByRole('button', { name: 'Account', exact: true }).click();
    await page.getByRole('button', { name: 'Refresh connection', exact: true }).click();
    await page.getByRole('button', { name: 'Close Codex account' }).click();
    await expect(
      page.getByText('2 attachments · Ready for Researcher', { exact: true }),
    ).toBeVisible();
    await page
      .getByLabel('Message attachments', { exact: true })
      .getByRole('button', { name: /^align:S03/ })
      .nth(1)
      .click();
    await expect(attachmentPreview(page).getByText(/^Captured observation/)).toBeVisible();
    await expect(attachmentPreview(page).locator('pre')).toHaveText('Alignment complete');
    await page.screenshot({ path: info.outputPath('r3a2-frozen-log-after-restart.png') });
    await closeAttachmentPreview(page);
    await page
      .getByRole('textbox', { name: 'Message draft' })
      .fill('Explain this earlier observation.');
    await page.getByRole('button', { name: 'Send', exact: true }).click();
    await expect(
      page.locator('.message-state').getByText('completed', { exact: true }),
    ).toHaveCount(1);
    const state = JSON.parse(
      await readFile(join(profile, 'codex/home/fixture-state.json'), 'utf8'),
    );
    const threads = Object.values(state.threads) as {
      turns: { items: { content?: unknown[] }[] }[];
    }[];
    const sent = JSON.stringify(threads[0]!.turns[0]!.items[0]!.content);
    expect(sent).toContain('Alignment complete');
    expect(sent).toContain('ERROR 😀 exact');
    expect(sent).toContain('decoded-preview-utf16-line-column');
    expect(sent).toContain('stderr');
    expect(sent).toContain('stdout');
    expect(sent).toContain('fixture-revision-4');
    expect(sent).not.toContain('Read sample S03');
    expect(sent).not.toContain('Different attempt');
    expect(sent).toContain(attached.capture!.asset.hash);
  } finally {
    await application.close();
    await rm(directory, { recursive: true, force: true });
  }
});
