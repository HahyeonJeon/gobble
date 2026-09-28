import { expect, test, type ElectronApplication } from '@playwright/test';
import { mkdir, readFile, writeFile } from 'node:fs/promises';
import { join, resolve } from 'node:path';
import { captureWindow, chooseProject, launch, readyView } from '../electron/support';
import { createRuntimeFixture } from './runtime-fixture';

test('actual CLI Run matches Electron and its controller survives app quit', async ({}, info) => {
  const image = process.env.GOBBLE_LIVE_RUNTIME_IMAGE;
  if (!image)
    throw new Error(
      'Set GOBBLE_LIVE_RUNTIME_IMAGE to an existing exact Gobble runtime image. This live test cannot use a fixture or skip.',
    );
  const directory = resolve(
    'test-results/live-fixtures',
    new Date().toISOString().replaceAll(':', '-'),
  );
  const runtime = await createRuntimeFixture(directory, image);
  const profile = join(directory, 'profile');
  let application: ElectronApplication | undefined;
  try {
    const doctor = JSON.parse(await runtime.cli('doctor'));
    expect(doctor.checks.workspace).toContain('sibling container');
    await runtime.cli('plan', '.');
    await runtime.cli('run', '.', '--workspace', 'runs/hello');
    expect(
      (
        await readFile(join(runtime.project, 'runs/hello/results/sequence-count.txt'), 'utf8')
      ).trim(),
    ).toBe('2');
    const completed = JSON.parse(
      await runtime.cli('inspect', 'monitor', '--workspace', 'runs/hello'),
    );
    await writeFile(info.outputPath('cli-completed.json'), JSON.stringify(completed, null, 2));
    await runtime.startGated();
    await expect
      .poll(
        async () => {
          try {
            const monitor = JSON.parse(
              await runtime.cli(
                'inspect',
                'monitor',
                '--workspace',
                'runs/live',
                '--instance',
                'count-sequences',
              ),
            );
            return JSON.stringify(monitor.logs).includes('stage6-started');
          } catch {
            return false;
          }
        },
        { timeout: 90_000, intervals: [1000, 2000] },
      )
      .toBe(true);
    const running = JSON.parse(await runtime.cli('inspect', 'monitor', '--workspace', 'runs/live'));
    await writeFile(info.outputPath('cli-running.json'), JSON.stringify(running, null, 2));
    expect(running.run.status).toBe('running');
    expect(running.run.occupancy.live).toBe(true);
    const started = await launch(profile);
    application = started.application;
    let page = started.page;
    await chooseProject(application, page, runtime.project);
    await page
      .locator('.candidate')
      .filter({ has: page.getByText('live', { exact: true }) })
      .getByRole('button', { name: 'Attach', exact: true })
      .click();
    await page.getByRole('button', { name: 'live', exact: true }).click();
    const runPane = await readyView(page);
    await expect(runPane.getByRole('heading', { name: running.run.id, exact: true })).toBeVisible();
    await expect(runPane.locator('.status-badge')).toHaveText(running.run.status);
    await expect(runPane.getByRole('row').filter({ hasText: 'count-sequences' })).toContainText(
      running.tasks[0].status,
    );
    const projects = await page.evaluate(() => window.gobble.projects.list());
    if (!projects.ok) throw new Error(projects.error.message);
    const projectId = projects.value[0]!.projectId;
    const runs = await page.evaluate(
      (projectId) => window.gobble.runs.list({ projectId }),
      projectId,
    );
    if (!runs.ok) throw new Error(runs.error.message);
    const run = runs.value.runs.find((item) => item.name === 'live')!;
    const snapshot = await page.evaluate((input) => window.gobble.runs.snapshot(input), {
      projectId,
      runRef: run.runRef,
    });
    if (!snapshot.ok) throw new Error(snapshot.error.message);
    expect(snapshot.value.runtimeBinding.imageId).toBe(runtime.image);
    expect(snapshot.value.snapshot.run.id).toBe(running.run.id);
    expect(snapshot.value.snapshot.tasks[0]!.attempt).toBe(running.tasks[0].attempt);
    await captureWindow(application, info.outputPath('run-running.png'));
    await runPane.getByRole('button', { name: 'View logs', exact: true }).click();
    const logPane = await readyView(page, 'Secondary pane');
    await expect(logPane.getByRole('textbox')).toHaveValue(/stage6-started/);
    await page
      .getByRole('textbox', { name: 'Message draft' })
      .fill('Keep the current Run and log review after restart.');
    await application.close();
    application = undefined;
    expect(await runtime.controllerRunning()).toBe(true);
    await runtime.release();
    expect(await runtime.controllerExit()).toBe('0');
    expect(
      (
        await readFile(join(runtime.project, 'runs/live/results/sequence-count.txt'), 'utf8')
      ).trim(),
    ).toBe('2');
    const final = JSON.parse(await runtime.cli('inspect', 'monitor', '--workspace', 'runs/live'));
    expect(final.run.id).toBe(running.run.id);
    expect(final.run.status).toBe('succeeded');
    expect(final.tasks[0].status).toBe('succeeded');
    expect(final.tasks[0].reason).not.toBe('log-copy-failed');
    await writeFile(info.outputPath('cli-final.json'), JSON.stringify(final, null, 2));
    const restarted = await launch(profile);
    application = restarted.application;
    page = restarted.page;
    const restored = await readyView(page);
    await expect(restored.locator('.status-badge')).toHaveText(final.run.status);
    await expect(page.getByRole('textbox', { name: 'Message draft' })).toHaveValue(
      'Keep the current Run and log review after restart.',
    );
    await expect((await readyView(page, 'Secondary pane')).getByRole('textbox')).toHaveValue(
      /stage6-finished/,
    );
    await captureWindow(application, info.outputPath('run-completed-after-restart.png'));
    await expect(page.getByRole('button', { name: /^(Start|Stop|Resume)$/ })).toHaveCount(0);
    const repeated = await page.evaluate(
      async ({ projectId, run }) => {
        const request = {
          projectId,
          workspaceResourceId: run.workspaceResourceId,
          requestId: 'req_live_repeat',
        };
        return Promise.all([
          window.gobble.runs.attach(request),
          window.gobble.runs.attach(request),
        ]);
      },
      { projectId, run },
    );
    expect(repeated.every((result) => result.ok && result.value.runRef === run.runRef)).toBe(true);
    // No engine fallback: changed identity in an isolated invalid Project must be refused.
    const invalid = join(directory, 'invalid-project');
    await mkdir(join(invalid, 'runs/invalid/.gobble'), { recursive: true });
    const lock = JSON.parse(await readFile(join(runtime.project, '.gobble-runtime.json'), 'utf8'));
    await writeFile(
      join(invalid, '.gobble-runtime.json'),
      JSON.stringify({ ...lock, daemon: 'not-the-recorded-daemon' }),
    );
    await chooseProject(application, page, invalid);
    await page.getByRole('button', { name: 'Attach', exact: true }).click();
    await expect(page.getByRole('alert')).toContainText('Docker daemon differs');
    const foreign = await page.evaluate(async (runRef) => {
      const state = await window.gobble.workspace.connect();
      if (!state.ok || !state.value.document) throw new Error('Missing active Project');
      return window.gobble.runs.snapshot({
        projectId: state.value.document.workspace.projectId,
        runRef,
      });
    }, run.runRef);
    expect(foreign.ok).toBe(false);
    if (!foreign.ok) expect(foreign.error.code).toBe('not_found');
    await writeFile(
      info.outputPath('fixture-location.json'),
      JSON.stringify({ project: runtime.project, image: runtime.image, profile }, null, 2),
    );
  } finally {
    await application?.close();
    await runtime.close();
  }
});
