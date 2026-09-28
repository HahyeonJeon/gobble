import { test, expect } from '@playwright/test';
import { readFile, cp, mkdtemp } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { launch, readyView } from './support';

test('completed actual continuation restores its latest attempt and original review without executing', async ({}, info) => {
  test.skip(
    !process.env.GOBBLE_CONTINUATION_RESTORE_FIXTURE,
    'Requires the owned completed actual-tool fixture',
  );
  test.setTimeout(90000);
  const fixture = JSON.parse(
    await readFile(process.env.GOBBLE_CONTINUATION_RESTORE_FIXTURE!, 'utf8'),
  );
  const before = JSON.parse(await readFile(join(fixture.profile, 'service/catalog.json'), 'utf8'));
  const { application, page } = await launch(fixture.profile, {
    GOBBLE_CODEX_EXECUTABLE: join(fixture.base, 'codex'),
  });
  try {
    await application.evaluate(({ BrowserWindow }) =>
      BrowserWindow.getAllWindows()[0]!.setSize(1440, 950),
    );
    await readyView(page);
    const flow = page.getByRole('region', { name: 'Run flow', exact: true });
    await expect(flow).toContainText('Execution complete');
    await expect(flow.locator('.pipeline-node[data-run-status="succeeded"]')).toHaveCount(2);
    await expect(page.getByRole('row').filter({ hasText: 'fastqc' })).toContainText('3');
    await expect(page.getByRole('article', { name: 'Continue analysis' })).toContainText(
      'Saved confirmation plan',
    );
    await expect(page.getByRole('button', { name: 'Resume analysis', exact: true })).toHaveCount(0);
    await page.screenshot({ path: info.outputPath('completed-wide.png') });
    await page.locator('.attachment-chip').getByRole('button').first().click();
    const preview = page.getByRole('dialog', { name: /^Attachment:/ });
    await expect(preview).toContainText('Will restart · Attempt 2');
    await expect(preview).toContainText('Attempt 1 remains in history.');
    await page.screenshot({ path: info.outputPath('completed-original-reference.png') });
    await page.keyboard.press('Escape');
    await application.evaluate(({ BrowserWindow }) =>
      BrowserWindow.getAllWindows()[0]!.setSize(1100, 800),
    );
    await page.getByRole('button', { name: 'Workspace', exact: true }).click();
    await expect(flow).toContainText('Execution complete');
    await page.screenshot({ path: info.outputPath('completed-compact-flow.png') });
    await page.getByRole('button', { name: 'Chat', exact: true }).click();
    await expect(page.getByRole('textbox', { name: 'Message draft' })).toHaveValue(
      'Keep the trimmed reads. Why does the quality check restart?',
    );
    await page.screenshot({ path: info.outputPath('completed-compact-chat.png') });
    const records = await page.evaluate(
      async (projectId) => window.gobble.continuations.list({ projectId }),
      before.runs[0].projectId,
    );
    if (!records.ok) throw Error(records.error.message);
    const receipts = records.value.filter((v) => v.operation?.receipt);
    expect(receipts).toHaveLength(2);
    expect(receipts.at(-1)?.operation?.observation?.status).toBe('succeeded');
    expect(
      JSON.parse(await readFile(join(fixture.profile, 'service/catalog.json'), 'utf8')).runs,
    ).toEqual(before.runs);
  } finally {
    await application.close();
  }
});

test('completed Run keeps its Flow and exact task actions reachable in stacked compact Panes', async ({}, info) => {
  test.skip(
    !process.env.GOBBLE_CONTINUATION_RESTORE_FIXTURE,
    'Requires the owned completed actual-tool fixture',
  );
  test.setTimeout(90000);
  const fixture = JSON.parse(
    await readFile(process.env.GOBBLE_CONTINUATION_RESTORE_FIXTURE!, 'utf8'),
  );
  const profile = join(await mkdtemp(join(tmpdir(), 'gobble-completed-layout-')), 'profile');
  await cp(fixture.profile, profile, {
    recursive: true,
    filter: (source) => !source.split('/').at(-1)!.startsWith('Singleton'),
  });
  const before = JSON.parse(await readFile(join(profile, 'service/catalog.json'), 'utf8'));
  const { application, page } = await launch(profile, {
    GOBBLE_CODEX_EXECUTABLE: join(fixture.base, 'codex'),
  });
  try {
    await application.evaluate(({ BrowserWindow }) =>
      BrowserWindow.getAllWindows()[0]!.setSize(1440, 950),
    );
    await readyView(page);
    const flow = page.getByRole('region', { name: 'Run flow', exact: true });
    const node = flow.locator('.pipeline-node[data-run-status="succeeded"]').last();
    await expect(node).toBeEnabled();
    await node.focus();
    await page.keyboard.press('Enter');
    await expect(page.locator('.observation-actions')).toContainText('fastqc · Attempt 3');
    await page.getByRole('button', { name: 'Open logs', exact: true }).click();
    await expect(page.getByRole('textbox', { name: 'stderr log text', exact: true })).toContainText(
      'Started analysis',
    );
    await application.evaluate(({ BrowserWindow }) =>
      BrowserWindow.getAllWindows()[0]!.setSize(1100, 800),
    );
    await page.getByRole('button', { name: 'Workspace', exact: true }).click();
    await page.screenshot({ path: info.outputPath('completed-stacked-compact.png') });
    // Check clipping, allowing only subpixel rounding in Chromium's intersection ratio.
    await expect(node).toBeInViewport({ ratio: 0.999 });
    await expect(flow).toContainText('Execution complete');
    await expect(flow.getByText('Continue this analysis', { exact: true })).toHaveCount(0);
    await page.getByRole('button', { name: 'Discuss task', exact: true }).click();
    await expect(page.locator('.attachment-chip')).toHaveCount(2);
    await page.getByRole('button', { name: 'Chat', exact: true }).click();
    await expect(page.locator('.attachment-chip').last()).toContainText(
      'Task · fastqc · Attempt 3',
    );
    await expect(page.getByRole('textbox', { name: 'Message draft' })).toHaveValue(
      'Keep the trimmed reads. Why does the quality check restart?',
    );
    await page.screenshot({ path: info.outputPath('completed-stacked-chat.png') });
    await page.getByRole('button', { name: 'Workspace', exact: true }).click();
    await flow.getByRole('button', { name: 'Open current design', exact: true }).click();
    await expect(page.getByRole('region', { name: 'Run preparation' })).toContainText(
      'Prepared review',
    );
    await page.screenshot({ path: info.outputPath('current-prepared-review.png') });

    expect(JSON.parse(await readFile(join(profile, 'service/catalog.json'), 'utf8')).runs).toEqual(
      before.runs,
    );
  } finally {
    await application.close();
  }
});
