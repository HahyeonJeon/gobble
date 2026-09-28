import { test, expect } from '@playwright/test';
import { readFile, rename } from 'node:fs/promises';
import { join } from 'node:path';
import { launch } from './support';

// Opt-in continuation of the owned synthetic two-Run acceptance profile.
test('Follow-up: historical design, missing capture and compact restored UI', async ({}, info) => {
  test.skip(!process.env.GOBBLE_FEEDBACK_FIXTURE, 'Requires the completed synthetic P5A fixture');
  test.setTimeout(90000);
  const { profile, base } = JSON.parse(
    await readFile(process.env.GOBBLE_FEEDBACK_FIXTURE!, 'utf8'),
  );
  const catalog = JSON.parse(await readFile(join(profile, 'service/catalog.json'), 'utf8'));
  expect(catalog.runs).toHaveLength(2);
  const { application, page } = await launch(profile, {
    GOBBLE_CODEX_EXECUTABLE: join(base, 'codex'),
  });
  try {
    await application.evaluate(({ BrowserWindow }) =>
      BrowserWindow.getAllWindows()[0]!.setSize(1600, 1000),
    );
    const flow = page.getByRole('region', { name: 'Run flow', exact: true });
    await expect(flow).toContainText('Following Analysis 1');
    const reviews = await page.evaluate(
      async (projectId) => window.gobble.launches.list({ projectId }),
      catalog.runs[0].projectId,
    );
    if (!reviews.ok) throw Error(reviews.error.message);
    const origin = reviews.value.find((v) => v.followUp)!.followUp!;
    const details = flow.locator('.follow-up-origin');
    if (!(await details.evaluate((el) => (el as HTMLDetailsElement).open)))
      await details.locator('summary').click();
    const capture = join(
      profile,
      'workspace/evidence',
      origin.projectId,
      origin.evidence[0]!.asset.hash.slice(7) + '.blob',
    );
    await rename(capture, capture + '.p5a-test-held');
    try {
      await details.getByRole('button', { name: /Task.*Attempt 1/ }).click();
      const preview = page.getByRole('dialog', { name: /Attachment:/ });
      await expect(preview.getByRole('alert')).toContainText('saved content is missing or corrupt');
      await expect(preview.locator('pre')).toHaveCount(0);
      await page.screenshot({ path: info.outputPath('feedback-missing.png') });
    } finally {
      await rename(capture + '.p5a-test-held', capture);
    }
    await page.getByRole('button', { name: 'Retry preview', exact: true }).click();
    await expect(page.getByRole('dialog', { name: /Attachment:/ })).toContainText(
      'Captured Run facts',
    );
    await expect(page.getByRole('dialog', { name: /Attachment:/ }).locator('pre')).toContainText(
      'trim_galore',
    );
    await page.keyboard.press('Escape');
    await expect
      .poll(() =>
        page.evaluate(() => {
          const flow = document.querySelector<HTMLElement>('.retained-run-flow')!;
          const tasks = document.querySelector<HTMLElement>('.run-view')!;
          return (
            tasks.getBoundingClientRect().top >= flow.getBoundingClientRect().bottom - 1 &&
            (flow.scrollHeight <= flow.clientHeight + 1 ||
              ['auto', 'scroll'].includes(getComputedStyle(flow).overflowY))
          );
        }),
      )
      .toBe(true);
    await page.screenshot({ path: info.outputPath('feedback-origin-expanded.png') });
    await page.getByRole('tab', { name: 'Analysis 1', exact: true }).click();
    await expect(flow).toContainText('used an earlier design');
    await expect(flow.locator('.pipeline-node[data-run-status="succeeded"]')).toHaveCount(2);
    await page.screenshot({ path: info.outputPath('feedback-earlier.png') });
    await page.getByRole('button', { name: 'Open current design', exact: true }).click();
    await page.getByRole('button', { name: 'Changes', exact: true }).click();
    const comparison = page.getByRole('region', { name: 'Pipeline change review' });
    await expect(comparison.getByText('20 bp', { exact: true })).toBeVisible();
    await page.screenshot({ path: info.outputPath('feedback-comparison.png') });
    await page.getByRole('tab', { name: 'Analysis 2', exact: true }).click();
    await expect(flow).toContainText('Following Analysis 1');
    await application.evaluate(({ BrowserWindow }) =>
      BrowserWindow.getAllWindows()[0]!.setSize(1100, 800),
    );
    await expect(
      page.getByRole('button', { name: 'Open current design', exact: true }),
    ).toBeVisible();
    await expect
      .poll(() => page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth))
      .toBe(true);
    await page.screenshot({ path: info.outputPath('feedback-compact.png') });
    await page
      .getByRole('navigation', { name: 'Project regions' })
      .getByRole('button', { name: 'Chat', exact: true })
      .click();
    await expect(page.getByRole('textbox', { name: 'Message draft' })).toBeVisible();
    await page.screenshot({ path: info.outputPath('feedback-compact-chat.png') });
  } catch (error) {
    await page.screenshot({ path: info.outputPath('failure.png') }).catch(() => {});
    throw error;
  } finally {
    await application.close();
  }
});
