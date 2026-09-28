import { test, expect } from '@playwright/test';
import { readFile } from 'node:fs/promises';
import { join } from 'node:path';
import { launch } from './support';

test('Run launch: retained Flow, exact task discussion and compact Chat survive App restart', async ({}, info) => {
  test.skip(
    !process.env.GOBBLE_LAUNCH_RESTORE_FIXTURE,
    'Requires an owned completed synthetic Run fixture',
  );
  const fixture = JSON.parse(await readFile(process.env.GOBBLE_LAUNCH_RESTORE_FIXTURE!, 'utf8'));
  const { application, page } = await launch(fixture.profile, {
    GOBBLE_CODEX_EXECUTABLE: join(fixture.base, 'codex'),
  });
  try {
    await application.evaluate(({ BrowserWindow }) =>
      BrowserWindow.getAllWindows()[0]!.setSize(1400, 900),
    );
    const flow = page.getByRole('region', { name: 'Run flow', exact: true });
    await expect(flow).toBeVisible();
    await expect(flow.locator('.pipeline-node[data-run-status="succeeded"]')).toHaveCount(2, {
      timeout: 20000,
    });
    await expect(page.getByRole('article', { name: 'Analysis run' })).toContainText('succeeded');
    await expect(page.getByRole('button', { name: 'Start analysis', exact: true })).toHaveCount(0);
    const node = flow.locator('.pipeline-node[data-run-status="succeeded"]').last();
    await expect(node).toBeEnabled();
    await node.click();
    await expect(page.locator('.observation-actions')).toContainText('fastqc · Attempt 1');
    await expect(page.getByRole('button', { name: 'Discuss task', exact: true })).toBeEnabled();
    await page.getByRole('button', { name: 'Discuss task', exact: true }).click();
    await expect(page.locator('.chat-composer .attachment-chip').last()).toContainText(
      'Task · fastqc · Attempt 1',
    );
    await expect(page.locator('.observation-actions')).toContainText('fastqc · Attempt 1');
    await expect(page.getByRole('button', { name: 'Discuss task', exact: true })).toBeEnabled();
    await page.screenshot({ path: info.outputPath('run-flow-restored.png') });
    await flow.screenshot({ path: info.outputPath('flow-detail.png') });
    await application.evaluate(({ BrowserWindow }) =>
      BrowserWindow.getAllWindows()[0]!.setSize(1100, 800),
    );
    await page.getByRole('button', { name: 'Workspace', exact: true }).click();
    await expect(node).toBeVisible();
    await page.screenshot({ path: info.outputPath('run-flow-compact.png') });
    await page.getByRole('button', { name: 'Chat', exact: true }).click();
    await expect(page.getByRole('textbox', { name: 'Message draft' })).toBeVisible();
    await page.screenshot({ path: info.outputPath('run-chat-compact.png') });
  } finally {
    await application.close();
  }
});
