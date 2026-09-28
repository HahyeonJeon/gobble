import { expect, type ElectronApplication, type Page, type TestInfo } from '@playwright/test';
import type { Continuation } from '@gobble/contracts';
import { readFile, writeFile, rename, stat, chmod, readdir } from 'node:fs/promises';
import { createHash } from 'node:crypto';
import { join } from 'node:path';
import { launch, readyView } from './support';

/** Opt-in actual Docker/tools qualification, extending the normal creation and Start UI. */
export async function verifyLiveContinuation(input: {
  application: ElectronApplication;
  page: Page;
  profile: string;
  base: string;
  project: string;
  info: TestInfo;
}) {
  let { application, page } = input;
  const { profile, base, project, info } = input;
  const catalog = JSON.parse(await readFile(join(profile, 'service/catalog.json'), 'utf8'));
  const projectId: string = catalog.runs[0].projectId;
  const runRef: string = catalog.runs[0].runRef;
  const initial = await page.evaluate(
    async (projectId) => window.gobble.launches.list({ projectId }),
    projectId,
  );
  if (!initial.ok || !initial.value[0]) throw Error('Initial admission unavailable');
  const original = initial.value[0];
  expect(original.operation?.runRef).toBe(runRef);
  const output = join(project, original.outputPath);
  const records = async () => {
    const result = await page.evaluate(
      async (projectId) => window.gobble.continuations.list({ projectId }),
      projectId,
    );
    if (!result.ok) throw Error(result.error.message);
    return result.value;
  };
  const checkpoint = async () => {
    const pointer = JSON.parse(await readFile(join(output, '.gobble/current.json'), 'utf8'));
    const dir = join(output, '.gobble/checkpoints', pointer.current);
    return {
      pointer,
      run: JSON.parse(await readFile(join(dir, 'run.json'), 'utf8')),
      tasks: JSON.parse(await readFile(join(dir, 'tasks.json'), 'utf8')),
    };
  };
  const flow = () => page.getByRole('region', { name: 'Run flow', exact: true });
  async function review(): Promise<Continuation> {
    const before = (await records()).length;
    const check = flow().getByRole('button', { name: /^(Recheck|Check) continuation$/ });
    await expect(check).toBeEnabled({ timeout: 30000 });
    await check.click();
    await expect
      .poll(
        async () => {
          const all = await records();
          return all.length > before && all.at(-1)?.state !== 'checking';
        },
        { timeout: 60000 },
      )
      .toBe(true);
    const value = (await records()).at(-1)!;
    await writeFile(info.outputPath(`review-${before + 1}.json`), JSON.stringify(value, null, 2));
    return value;
  }
  async function confirm(): Promise<void> {
    await page.getByRole('button', { name: 'Resume analysis', exact: true }).click();
    await expect
      .poll(
        async () => {
          const state = (await records()).at(-1)?.operation?.state;
          // Admission can follow the first receipt query. Reconcile the same intent;
          // never click Resume again or submit a replacement request.
          if (state === 'unknown') {
            const refresh = page.getByRole('button', { name: 'Check continuation status' });
            if (await refresh.isEnabled()) await refresh.click();
          }
          return state;
        },
        { timeout: 60000, intervals: [500, 1000, 2000] },
      )
      .toBe('admitted');
  }
  async function stop(): Promise<void> {
    await page.getByRole('button', { name: 'Stop analysis', exact: true }).click();
    await expect
      .poll(async () => (await checkpoint()).run.status, { timeout: 60000 })
      .toBe('stopped');
    await page.getByRole('button', { name: 'Refresh', exact: true }).click();
    await expect(flow().locator('header').first()).toContainText('stopped', { timeout: 30000 });
  }
  try {
    await application.evaluate(({ BrowserWindow }) =>
      BrowserWindow.getAllWindows()[0]!.setSize(1440, 950),
    );
    // Real FastQC must have started; the test does not pause, wrap or replace a tool.
    await expect(flow().locator('.pipeline-node[data-run-status="succeeded"]')).toHaveCount(1, {
      timeout: 240000,
    });
    await expect(flow().locator('.pipeline-node[data-run-status="running"]')).toHaveCount(1, {
      timeout: 30000,
    });
    await stop();
    const stopped = await checkpoint();
    await writeFile(info.outputPath('initial-stopped.json'), JSON.stringify(stopped, null, 2));
    expect(stopped.pointer.format).toBe(3);
    const attemptLogs = async () => {
      const directory = join(output, '.gobble/tasks');
      const files = await readdir(directory, { recursive: true });
      const hashes: Record<string, string> = {};
      for (const file of files.filter((p) => /\/(stdout|stderr)$/.test(p)))
        hashes[file] = createHash('sha256')
          .update(await readFile(join(directory, file)))
          .digest('hex');
      return hashes;
    };
    const originalLogs = await attemptLogs();
    expect(Object.keys(originalLogs).length).toBeGreaterThanOrEqual(4);
    const first = await review();
    expect(first.state, first.issue).toBe('ready');
    expect(first.review?.steps.map((s) => s.action)).toEqual(['reuse', 'restart']);
    expect(first.review?.steps[1]?.plannedAttempt).toBe(2);
    await expect(flow().locator('[data-continuation="restart"]')).toHaveCount(1, {
      timeout: 20000,
    });
    await flow().locator('[data-continuation="restart"]').click();
    await flow().getByRole('button', { name: 'Discuss this step' }).click();
    await page
      .getByRole('textbox', { name: 'Message draft' })
      .fill('Keep the trimmed reads. Why does the quality check restart?');
    await readyView(page);
    await page.screenshot({ path: info.outputPath('live-continuation-ready.png') });

    const completed = first.review!.files[1]!;
    const resultPath = join(output, completed.path);
    const saved = await readFile(resultPath);
    expect(createHash('sha256').update(saved).digest('hex')).toBe(completed.sha256);
    // Only this disposable stopped Run is changed. Always restore the exact bytes/path.
    await rename(resultPath, resultPath + '.qualification-held');
    try {
      expect(await review()).toMatchObject({
        state: 'blocked',
        issue: expect.stringContaining('A completed result is missing.'),
      });
    } finally {
      await rename(resultPath + '.qualification-held', resultPath);
    }
    await page.locator('.attachment-chip').getByRole('button').first().click();
    await expect(page.getByRole('dialog', { name: /^Attachment:/ })).toContainText(
      'Will restart · Attempt 2',
    );
    await page.screenshot({ path: info.outputPath('live-historical-reference.png') });
    await page.keyboard.press('Escape');
    await writeFile(resultPath, Buffer.concat([saved, Buffer.from('changed')]));
    try {
      expect(await review()).toMatchObject({
        state: 'blocked',
        issue: expect.stringContaining('A completed result has changed.'),
      });
    } finally {
      await writeFile(resultPath, saved);
    }
    const inputPath = join(output, first.review!.files[0]!.path);
    const savedInput = await readFile(inputPath);
    const inputMode = (await stat(inputPath)).mode & 0o777;
    await chmod(inputPath, inputMode | 0o200);
    try {
      await writeFile(inputPath, Buffer.concat([savedInput, Buffer.from('changed')]));
      expect(await review()).toMatchObject({
        state: 'blocked',
        issue: expect.stringContaining('The checked input copy has changed.'),
      });
    } finally {
      await writeFile(inputPath, savedInput);
      await chmod(inputPath, inputMode);
    }
    const checked = await review();
    expect(checked.state, checked.issue).toBe('ready');
    expect(checked.review?.digest).toBe(first.review?.digest);
    await application.close();
    ({ application, page } = await launch(profile, {
      GOBBLE_CODEX_EXECUTABLE: join(base, 'codex'),
    }));
    await application.evaluate(({ BrowserWindow }) =>
      BrowserWindow.getAllWindows()[0]!.setSize(1440, 950),
    );
    expect((await checkpoint()).run.executionHistory.continuations).toHaveLength(0);
    await expect(page.getByRole('textbox', { name: 'Message draft' })).toHaveValue(
      'Keep the trimmed reads. Why does the quality check restart?',
    );
    await confirm();
    await page.getByRole('button', { name: 'Check continuation status' }).click();
    await expect(page.getByRole('button', { name: 'Stop analysis', exact: true })).toBeVisible({
      timeout: 30000,
    });
    await expect
      .poll(
        async () =>
          (await checkpoint()).tasks.tasks.find(
            (t: { id: string; attempt: number }) => t.id === 'fastqc' && t.attempt === 2,
          )?.status,
        { timeout: 30000 },
      )
      .toBe('running');
    await stop();
    const repeated = await checkpoint();
    await writeFile(
      info.outputPath('continuation-stopped.json'),
      JSON.stringify(repeated, null, 2),
    );
    expect(repeated.run.admission).toEqual(stopped.run.admission);
    expect(repeated.run.executionHistory.continuations).toHaveLength(1);
    const second = await review();
    expect(second.state, second.issue).toBe('ready');
    expect(second.review?.steps[0]).toMatchObject({
      action: 'reuse',
      priorAttempt: 1,
      plannedAttempt: 1,
    });
    expect(second.review?.steps[1]).toMatchObject({
      action: 'restart',
      priorAttempt: 2,
      plannedAttempt: 3,
    });
    await confirm();
    const accepted = (await records()).at(-1)!;
    const replay = await page.evaluate(
      async (v) =>
        window.gobble.continuations.confirm({
          projectId: v.projectId,
          reviewId: v.requestId,
          requestId: v.operation!.requestId,
          reviewDigest: v.review!.digest,
        }),
      accepted,
    );
    expect(replay.ok).toBe(true);
    await application.close();
    ({ application, page } = await launch(profile, {
      GOBBLE_CODEX_EXECUTABLE: join(base, 'codex'),
    }));
    await application.evaluate(({ BrowserWindow }) =>
      BrowserWindow.getAllWindows()[0]!.setSize(1440, 950),
    );
    await expect
      .poll(async () => (await checkpoint()).run.status, { timeout: 240000 })
      .toBe('succeeded');
    await page.getByRole('button', { name: 'Check continuation status' }).click();
    await expect(page.getByRole('article', { name: 'Continue analysis' })).toContainText(
      'Last checked execution: succeeded',
      { timeout: 30000 },
    );
    await page.getByRole('button', { name: 'Refresh', exact: true }).click();
    await expect(flow().locator('.pipeline-node[data-run-status="succeeded"]')).toHaveCount(2, {
      timeout: 30000,
    });
    const final = await checkpoint();
    expect(final.run.admission).toEqual(stopped.run.admission);
    expect(final.run.executionHistory.continuations).toHaveLength(2);
    expect(await readFile(resultPath)).toEqual(saved);
    expect(
      JSON.parse(await readFile(join(profile, 'service/catalog.json'), 'utf8')).runs,
    ).toHaveLength(1);
    const finalLogs = await attemptLogs();
    for (const [path, hash] of Object.entries(originalLogs)) expect(finalLogs[path]).toBe(hash);
    expect(
      final.tasks.tasks.map((t: { id: string; attempt: number; status: string }) => [
        t.id,
        t.attempt,
        t.status,
      ]),
    ).toEqual(
      expect.arrayContaining([
        ['trim_galore', 1, 'succeeded'],
        ['fastqc', 1, 'incomplete'],
        ['fastqc', 2, 'incomplete'],
        ['fastqc', 3, 'succeeded'],
      ]),
    );
    await writeFile(
      info.outputPath('attempt-logs.json'),
      JSON.stringify({ originalLogs, finalLogs }, null, 2),
    );
    await writeFile(
      info.outputPath('final.json'),
      JSON.stringify(
        { initial: original, stopped, repeated, final, continuations: await records() },
        null,
        2,
      ),
    );
    await page.screenshot({ path: info.outputPath('live-continuation-completed.png') });
    await application.evaluate(({ BrowserWindow }) =>
      BrowserWindow.getAllWindows()[0]!.setSize(1100, 800),
    );
    await page.getByRole('button', { name: 'Workspace', exact: true }).click();
    await page.screenshot({ path: info.outputPath('live-compact-flow.png') });
    await page.getByRole('button', { name: 'Chat', exact: true }).click();
    await page.screenshot({ path: info.outputPath('live-compact-chat.png') });
  } catch (error) {
    await page.screenshot({ path: info.outputPath('continuation-failure.png') }).catch(() => {});
    await writeFile(info.outputPath('continuation-failure.txt'), String(error));
    throw error;
  } finally {
    await application.close().catch(() => {});
  }
}
