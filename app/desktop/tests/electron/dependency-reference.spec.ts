import { focusWindow } from './support';
import { expect, test, type Page } from '@playwright/test';
import { readFile, rm, writeFile } from 'node:fs/promises';
import { join } from 'node:path';
import { launch, chooseProject, readyView, openAgentRoster, captureWindow } from './support';
import { runtimeFixture } from './runtime-fixture';
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
  const state = await document(page);
  return { surfaces: state.workspace.surfaces, selections: state.selections, chat: state.chat };
};
test('Agent dependency marks preserve User work; Show/Return, historical capture, current search and restart stay distinct', async ({}, info) => {
  test.setTimeout(100_000);
  const fixture = await runtimeFixture();
  let { application, page } = await launch(fixture.profile, fixture.environment);
  try {
    await chooseProject(application, page, fixture.root);
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
    let primary = await readyView(page);
    await primary.getByRole('button', { name: 'Dependencies', exact: true }).click();
    await readyView(page);
    await primary.getByRole('button', { name: 'Task group align', exact: true }).click();
    await expect(primary.getByRole('button', { name: 'Discuss group', exact: true })).toBeEnabled();
    await primary.getByRole('button', { name: 'Discuss group', exact: true }).click();
    await primary.getByText('Dependency list · 1', { exact: true }).click();
    await primary.getByRole('button', { name: 'prepare → align', exact: true }).click();
    await expect(
      primary.getByRole('button', { name: 'Discuss dependency', exact: true }),
    ).toBeEnabled();
    await primary.getByRole('button', { name: 'Discuss dependency', exact: true }).click();
    await primary.getByRole('button', { name: 'Task group prepare', exact: true }).click();
    await expect(
      primary.getByRole('button', { name: 'Task group prepare', exact: true }),
    ).toHaveAttribute('aria-pressed', 'true');
    await primary.getByRole('button', { name: 'Zoom in dependencies' }).click();
    await expect(primary.getByLabel('Dependency zoom')).toHaveText('125%');
    const draft = page.getByRole('textbox', { name: 'Message draft' });
    await draft.fill('dependency-point');
    await page.getByRole('button', { name: 'Send', exact: true }).click();
    await expect(draft).toHaveValue('');
    await draft.fill('My next instruction stays local.');
    const before = await userState(page);
    await focusWindow(application);
    await writeFile(join(fixture.profile, 'codex/home/dependency-gate'), 'ready');
    await expect(page.locator('.shared-reference-event')).toHaveCount(2);
    await expect(
      page.locator('.message-state').getByText('completed', { exact: true }),
    ).toHaveCount(1);
    expect(await userState(page)).toEqual(before);
    await expect(draft).toBeFocused();
    await expect(primary.locator('.dependency-node[data-agent-mark="true"]')).toContainText(
      'Researcher',
    );
    await expect(primary.locator('g[data-agent-mark="true"]')).toHaveCount(1);
    await captureWindow(application, info.outputPath('dependency-agent-marks.png'));
    const groupEvent = page
      .locator('.shared-reference-event')
      .filter({ hasText: 'Review the align group.' });
    const edgeEvent = page
      .locator('.shared-reference-event')
      .filter({ hasText: 'Review prepare to align.' });
    await primary.getByRole('button', { name: 'Tasks', exact: true }).click();
    await readyView(page);
    await primary.getByRole('searchbox', { name: 'Search tasks' }).fill('S03');
    await expect(primary.getByRole('radio')).toHaveCount(1);
    const base = await userState(page);
    await edgeEvent.getByRole('button', { name: 'Show in view', exact: true }).click();
    await readyView(page);
    const reference = primary.locator('.observed-reference-panel');
    await expect(reference.getByText('prepare → align', { exact: true }).first()).toBeVisible();
    await expect(reference.getByRole('button', { name: 'Discuss dependency' })).toHaveCount(0);
    await expect(reference.locator('g[data-agent-mark="true"]')).toHaveCount(1);
    await reference.getByRole('button', { name: 'Zoom in dependencies' }).click();
    await expect(reference.getByLabel('Dependency zoom')).toHaveText('125%');
    expect(await userState(page)).toEqual(base);
    await captureWindow(application, info.outputPath('dependency-agent-show.png'));
    await reference.getByRole('button', { name: 'Return to my view' }).click();
    await readyView(page);
    expect(await userState(page)).toEqual(base);
    await expect(primary.getByRole('searchbox')).toHaveValue('S03');
    await expect(draft).toHaveValue('My next instruction stays local.');
    await groupEvent.getByRole('button', { name: 'Show in view', exact: true }).click();
    await readyView(page);
    await expect(
      primary.locator('.observed-reference-panel .dependency-node[data-agent-mark="true"]'),
    ).toHaveAttribute('data-group-id', 'align');
    await expect(
      primary.locator('.observed-reference-panel').getByRole('radio').first(),
    ).toBeDisabled();
    await application.evaluate(({ BrowserWindow }) => {
      const window = BrowserWindow.getAllWindows()[0]!;
      window.setSize(900, 650);
      window.webContents.setZoomFactor(1.5);
    });
    if (await page.getByRole('button', { name: 'Workspace', exact: true }).isVisible())
      await page.getByRole('button', { name: 'Workspace', exact: true }).click();
    await readyView(page);
    await expect(primary.getByRole('button', { name: 'Return to my view' })).toBeInViewport();
    await captureWindow(application, info.outputPath('dependency-agent-compact.png'));
    await primary.getByRole('button', { name: 'Return to my view' }).click();
    await application.evaluate(({ BrowserWindow }) => {
      const window = BrowserWindow.getAllWindows()[0]!;
      window.webContents.setZoomFactor(1);
      window.setSize(1280, 840);
    });
    await readyView(page);
    // A changed observation must never silently bind the old pair to current facts.
    fixture.monitor.tasks[1]!.attempt = 3;
    await writeFile(fixture.monitorPath, JSON.stringify(fixture.monitor));
    await primary.getByRole('button', { name: 'Refresh', exact: true }).click();
    await readyView(page);
    await edgeEvent.getByRole('button', { name: 'Show in view', exact: true }).click();
    await expect(page.getByText(/This pointer targets an older observation/)).toBeVisible();
    await expect(primary.getByRole('button', { name: 'Return to my view' })).toHaveCount(0);
    await edgeEvent.getByRole('button', { name: 'View captured evidence', exact: true }).click();
    const capture = page.getByRole('dialog', { name: 'Captured evidence', exact: true });
    await expect(capture.getByText('prepare → align', { exact: true })).toBeVisible();
    await captureWindow(application, info.outputPath('dependency-agent-capture.png'));
    await capture.getByRole('button', { name: 'Close Captured evidence' }).click();
    const saved = (await document(page)).sharedReferences;
    await edgeEvent.getByText('Mark options', { exact: true }).click();
    await edgeEvent.getByRole('button', { name: 'Find current dependency' }).click();
    await readyView(page);
    await expect(primary.getByRole('combobox', { name: 'Dependency representation' })).toHaveValue(
      'list',
    );
    await expect(primary.getByRole('searchbox', { name: 'Find task group' })).toHaveValue(
      'prepare',
    );
    expect((await document(page)).selections).toHaveLength(0);
    expect((await document(page)).sharedReferences).toEqual(saved);
    await expect(primary.locator('.dependency-list [data-agent-mark="true"]')).toHaveCount(0);
    const delivery = JSON.parse(
      await readFile(join(fixture.profile, 'codex/home/dependency-delivery.json'), 'utf8'),
    );
    expect(delivery.observed.content.kind).toBe('run-dependencies');
    expect(delivery.observed.receipt).not.toHaveProperty('evidence');
    expect(delivery.selected.receipt.evidenceId).toBeTruthy();
    await writeFile(
      info.outputPath('dependency-scripted-delivery.json'),
      JSON.stringify(delivery, null, 2),
    );
    // Restore the synthetic bytes only, then restart while Show is open.
    fixture.monitor.tasks[1]!.attempt = 2;
    await writeFile(fixture.monitorPath, JSON.stringify(fixture.monitor));
    await primary.getByRole('button', { name: 'Refresh', exact: true }).click();
    await readyView(page);
    await edgeEvent.getByRole('button', { name: 'Show in view', exact: true }).click();
    await expect(primary.getByRole('button', { name: 'Return to my view' })).toBeVisible();
    await application.close();
    ({ application, page } = await launch(fixture.profile, fixture.environment));
    primary = await readyView(page);
    await expect(page.getByRole('button', { name: 'Return to my view' })).toHaveCount(0);
    await expect(primary.getByRole('searchbox', { name: 'Find task group' })).toHaveValue(
      'prepare',
    );
    await expect(page.getByRole('textbox', { name: 'Message draft' })).toHaveValue(
      'My next instruction stays local.',
    );
  } finally {
    await application.close();
    await rm(fixture.directory, { recursive: true, force: true });
  }
});
