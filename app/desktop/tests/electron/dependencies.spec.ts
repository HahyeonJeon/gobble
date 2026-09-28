import { expect, test, type Page, type ElectronApplication } from '@playwright/test';
import { readFile, rm, writeFile } from 'node:fs/promises';
import { join } from 'node:path';
import { runtimeFixture } from './runtime-fixture';
import {
  launch,
  chooseProject,
  readyView,
  openAgentRoster,
  attachmentPreview,
  closeAttachmentPreview,
  captureWindow,
} from './support';
async function addResearcher(application: ElectronApplication, page: Page) {
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
  await dialog.getByLabel('Name', { exact: true }).fill('Researcher');
  await dialog.getByRole('button', { name: 'Add agent', exact: true }).click();
  await expect(dialog).not.toBeVisible();
}
async function document(page: Page) {
  return page.evaluate(async () => {
    const projects = await window.gobble.projects.list();
    if (!projects.ok) throw new Error(projects.error.message);
    const state = await window.gobble.workspace.read({ projectId: projects.value[0]!.projectId });
    if (!state.ok) throw new Error(state.error.message);
    return state.value;
  });
}
async function paint(page: Page) {
  await page.evaluate(
    () =>
      new Promise<void>((resolve) =>
        requestAnimationFrame(() => requestAnimationFrame(() => resolve())),
      ),
  );
}

test('User selects a group, exact attempt and directed dependency; frozen evidence and navigation survive restart and source changes', async ({}, info) => {
  test.setTimeout(90_000);
  const fixture = await runtimeFixture();
  let { application, page } = await launch(fixture.profile, fixture.environment);
  try {
    await chooseProject(application, page, fixture.root);
    await addResearcher(application, page);
    await page.getByRole('button', { name: 'Attach', exact: true }).click();
    await page.getByRole('button', { name: 'existing', exact: true }).click();
    let primary = await readyView(page);
    await primary.getByRole('searchbox', { name: 'Search tasks' }).fill('S03');
    await expect(primary.getByRole('radio')).toHaveCount(1);
    await primary.getByRole('button', { name: 'Dependencies', exact: true }).click();
    await readyView(page);
    const group = primary.getByRole('button', { name: 'Task group align', exact: true });
    await group.focus();
    await group.press('Enter');
    await expect(primary.getByRole('button', { name: 'Discuss group', exact: true })).toBeEnabled();
    await primary
      .getByRole('radio', { name: 'Instance align-template · Attempt 0', exact: true })
      .click();
    await expect(
      primary.getByRole('radio', { name: 'Instance align-template · Attempt 0', exact: true }),
    ).toBeChecked();
    await expect(primary.getByRole('button', { name: 'Open logs', exact: true })).toBeDisabled();
    await primary.getByRole('button', { name: 'Tasks', exact: true }).click();
    await readyView(page);
    await primary.getByRole('button', { name: 'Show selected task', exact: true }).click();
    await expect(primary.getByRole('searchbox', { name: 'Search tasks' })).toHaveValue(
      'align-template',
    );
    await expect(primary.getByRole('radio')).toBeChecked();
    await primary.getByRole('searchbox', { name: 'Search tasks' }).fill('S03');
    await expect(primary.getByRole('radio')).toHaveCount(1);
    await primary.getByRole('button', { name: 'Dependencies', exact: true }).click();
    await readyView(page);

    await primary
      .getByRole('radio', { name: 'Instance align:S03 · Attempt 2', exact: true })
      .click();
    await expect(
      primary.getByRole('radio', { name: 'Instance align:S03 · Attempt 2', exact: true }),
    ).toBeChecked();
    await primary.getByRole('button', { name: 'Open logs', exact: true }).click();
    const secondary = await readyView(page, 'Secondary pane');
    await expect(secondary.getByRole('textbox', { name: 'stderr log text' })).toHaveValue(
      /ERROR 😀 exact/,
    );
    await primary.getByRole('button', { name: 'Zoom in dependencies' }).click();
    await expect(primary.getByLabel('Dependency zoom')).toHaveText('125%');
    await readyView(page);
    await primary.getByText('Dependency list · 1', { exact: true }).click();
    const pair = primary.getByRole('button', { name: 'prepare → align', exact: true });
    await pair.focus();
    await pair.press('Enter');
    // Revealing an already unfiltered target changes camera only; it must not strand readiness.
    await primary.getByRole('button', { name: 'Show selected target', exact: true }).click();
    await expect(
      primary.getByRole('button', { name: 'Discuss dependency', exact: true }),
    ).toBeEnabled();

    await primary.getByRole('button', { name: 'Discuss dependency', exact: true }).click();
    await expect(page.getByRole('textbox', { name: 'Message draft' })).toBeFocused();
    await page
      .getByRole('textbox', { name: 'Message draft' })
      .fill('Explain this observed dependency.');
    await page
      .getByRole('combobox', { name: 'Conversation recipient' })
      .selectOption({ label: 'To Researcher' });
    await expect(
      page.getByText('1 attachment · Ready for Researcher', { exact: true }),
    ).toBeVisible();
    const captured = (await document(page)).chat.attachments![0]!;
    expect(captured.evidence).toMatchObject({
      schemaVersion: 4,
      selection: { kind: 'run-dependency', fromTaskId: 'prepare', toTaskId: 'align' },
    });
    expect((await document(page)).collaboration?.submissions ?? []).toHaveLength(0);
    await paint(page);
    await captureWindow(application, info.outputPath('dependency-discussion.png'));
    await primary.getByRole('button', { name: 'Tasks', exact: true }).click();
    await readyView(page);
    await expect(primary.getByRole('searchbox', { name: 'Search tasks' })).toHaveValue('S03');
    expect(
      (await document(page)).selections.find((s) => s.evidence.schemaVersion === 4)?.evidence,
    ).toEqual(captured.evidence);
    await primary.getByRole('button', { name: 'Show in Dependencies', exact: true }).click();
    await readyView(page);
    await application.close();
    ({ application, page } = await launch(fixture.profile, fixture.environment));
    primary = await readyView(page);
    await expect(
      primary.getByRole('button', { name: 'Dependencies', exact: true }),
    ).toHaveAttribute('aria-pressed', 'true');
    await expect(primary.getByLabel('Dependency zoom')).toHaveText('125%');
    await expect(page.getByRole('textbox', { name: 'Message draft' })).toHaveValue(
      'Explain this observed dependency.',
    );
    await expect(
      primary.getByRole('button', { name: 'Discuss dependency', exact: true }),
    ).toBeEnabled();
    await writeFile(fixture.monitorPath, '{}');
    await primary.getByRole('button', { name: 'Refresh', exact: true }).click();
    await expect(primary.getByText(/Refresh failed · Showing previous observation/)).toBeVisible();
    await expect(
      primary.getByRole('button', { name: 'Discuss dependency', exact: true }),
    ).toBeEnabled();
    fixture.monitor.snapshot = 'fixture-revision-5';
    fixture.monitor.tasks[1]!.attempt = 3;
    fixture.monitor.edges = [];
    await writeFile(fixture.monitorPath, JSON.stringify(fixture.monitor));
    await primary.getByRole('button', { name: 'Refresh', exact: true }).click();
    await readyView(page);
    await expect(primary.locator('.dependency-scope')).toContainText(
      '0 of 0 observed dependencies',
    );
    await expect(
      primary.getByRole('button', { name: 'Discuss dependency', exact: true }),
    ).toHaveCount(0);
    expect((await document(page)).chat.attachments![0]).toEqual(captured);
    await page.getByRole('button', { name: 'Account', exact: true }).click();
    await page.getByRole('button', { name: 'Refresh connection', exact: true }).click();
    await page.getByRole('button', { name: 'Close Codex account' }).click();
    await expect(
      page.getByText('1 attachment · Ready for Researcher', { exact: true }),
    ).toBeVisible();
    await page
      .getByLabel('Message attachments', { exact: true })
      .locator('.attachment-toggle')
      .click();
    await expect(
      attachmentPreview(page).getByText('prepare → align', { exact: true }),
    ).toBeVisible();
    await expect(
      attachmentPreview(page).getByRole('row').filter({ hasText: 'align:S03' }),
    ).toContainText('2');
    await paint(page);
    await captureWindow(application, info.outputPath('dependency-frozen-preview.png'));
    await closeAttachmentPreview(page);
    await page.getByRole('button', { name: 'Send', exact: true }).click();
    await expect(
      page.locator('.message-state').getByText('completed', { exact: true }),
    ).toHaveCount(1);
    const state = JSON.parse(
      await readFile(join(fixture.profile, 'codex/home/fixture-state.json'), 'utf8'),
    );
    const threads = Object.values(state.threads) as {
      turns: { items: { content?: unknown[] }[] }[];
    }[];
    const sent = JSON.stringify(threads[0]!.turns[0]!.items[0]!.content);
    expect(sent).toContain('observed-authored-task-pair');
    expect(sent).toContain('fixture-revision-4');
    expect(sent).not.toContain('Read sample S03');
    expect(sent).not.toContain('fixture-revision-5');
  } finally {
    await application.close();
    await rm(fixture.directory, { recursive: true, force: true });
  }
});

test('missing, cyclic and bounded topology stays navigable by list and at compact native zoom', async ({}, info) => {
  test.setTimeout(90_000);
  const fixture = await runtimeFixture();
  const { application, page } = await launch(fixture.profile, fixture.environment);
  try {
    await chooseProject(application, page, fixture.root);
    await page.getByRole('button', { name: 'Attach', exact: true }).click();
    await page.getByRole('button', { name: 'existing', exact: true }).click();
    const primary = await readyView(page);
    await primary.getByRole('button', { name: 'Dependencies', exact: true }).click();
    await readyView(page);
    const raw: Record<string, unknown> = fixture.monitor;
    delete raw.edges;
    await writeFile(fixture.monitorPath, JSON.stringify(raw));
    await primary.getByRole('button', { name: 'Refresh', exact: true }).click();
    await readyView(page);
    await expect(
      primary.getByText('Dependencies unavailable. Known task groups are listed below.', {
        exact: true,
      }),
    ).toBeVisible();
    fixture.monitor.edges = [
      { from: 'prepare', to: 'align' },
      { from: 'align', to: 'prepare' },
    ];
    await writeFile(fixture.monitorPath, JSON.stringify(fixture.monitor));
    await primary.getByRole('button', { name: 'Refresh', exact: true }).click();
    await readyView(page);
    await expect(primary.getByLabel('Dependency overview list')).toBeVisible();
    await primary.getByRole('button', { name: 'prepare → align', exact: true }).click();
    await expect(
      primary.getByRole('button', { name: 'Discuss dependency', exact: true }),
    ).toBeEnabled();
    for (const count of [80, 81]) {
      fixture.monitor.tasks = Array.from({ length: count }, (_, i) => ({
        identity: 'instance-' + i,
        task_id: 'task-' + String(i).padStart(2, '0'),
        name: 'Repeated label',
        status: i % 2 ? 'completed' : 'failed',
        attempt: 1,
      }));
      fixture.monitor.edges = Array.from({ length: count - 1 }, (_, i) => ({
        from: 'task-' + String(i).padStart(2, '0'),
        to: 'task-' + String(i + 1).padStart(2, '0'),
      }));
      await writeFile(fixture.monitorPath, JSON.stringify(fixture.monitor));
      await primary.getByRole('button', { name: 'Refresh', exact: true }).click();
      await readyView(page);
      await expect(primary.locator('.dependency-scope')).toContainText(
        `80 of ${count} observed groups`,
      );
      if (count === 80) {
        await expect(primary.locator('.dependency-node')).toHaveCount(80);
        await primary.locator('.dependency-viewport').hover();
        await page.mouse.wheel(300, 0);
        await expect
          .poll(async () => {
            const s = (await document(page)).workspace.surfaces.find((s) => s.view === 'run');
            return s?.view === 'run' ? s.runView?.dependencies?.camera.x : undefined;
          })
          .toBeGreaterThan(0);
        const pan = await primary.locator('.dependency-viewport').evaluate((e) => e.scrollLeft);
        await primary.getByRole('button', { name: 'Tasks', exact: true }).click();
        await readyView(page);
        await primary.getByRole('button', { name: 'Dependencies', exact: true }).click();
        await readyView(page);
        await expect
          .poll(() => primary.locator('.dependency-viewport').evaluate((e) => e.scrollLeft))
          .toBeCloseTo(pan, 0);
        await primary.getByRole('button', { name: 'Fit graph', exact: true }).click();
        await readyView(page);
        await application.evaluate(({ BrowserWindow }) => {
          const w = BrowserWindow.getAllWindows()[0]!;
          w.setSize(900, 650);
          w.webContents.setZoomFactor(1.5);
        });
        const workspace = page.getByRole('button', { name: 'Workspace', exact: true });
        if (await workspace.isVisible()) await workspace.click();
        await readyView(page);
        await primary
          .getByRole('combobox', { name: 'Dependency representation' })
          .selectOption('list');
        await readyView(page);
        const first = primary.getByRole('button', { name: 'task-00 1 observed', exact: true });
        await first.focus();
        await first.press('Enter');
        await primary.getByRole('searchbox', { name: 'Find task group' }).fill('unreturned-group');
        await expect(
          primary.getByText('Selection is outside this search.', { exact: true }),
        ).toBeVisible();
        await expect(
          primary.getByRole('button', { name: 'Discuss group', exact: true }),
        ).toBeDisabled();
        await primary
          .locator('.dependency-selection-actions')
          .getByRole('button', { name: 'Show selected target', exact: true })
          .click();
        await expect(primary.getByRole('searchbox', { name: 'Find task group' })).toHaveValue('');

        await expect(
          primary.getByRole('button', { name: 'Discuss group', exact: true }),
        ).toBeEnabled();
        await primary
          .getByRole('button', { name: 'Discuss group', exact: true })
          .scrollIntoViewIfNeeded();
        await expect(
          primary.getByRole('button', { name: 'Discuss group', exact: true }),
        ).toBeInViewport();
        await paint(page);
        await captureWindow(application, info.outputPath('dependency-compact-150.png'));
        await application.evaluate(({ BrowserWindow }) => {
          const w = BrowserWindow.getAllWindows()[0]!;
          w.webContents.setZoomFactor(1);
          w.setSize(1280, 840);
        });
        await primary
          .getByRole('combobox', { name: 'Dependency representation' })
          .selectOption('graph');
        await readyView(page);
      } else {
        await expect(primary.getByLabel('Dependency overview list')).toBeVisible();
        await expect(primary.locator('.dependency-node')).toHaveCount(0);
        await paint(page);
        await captureWindow(application, info.outputPath('dependency-bounded-list.png'));
      }
    }
  } finally {
    await application.close();
    await rm(fixture.directory, { recursive: true, force: true });
  }
});
