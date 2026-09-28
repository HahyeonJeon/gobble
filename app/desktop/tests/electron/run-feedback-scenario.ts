import { expect, type ElectronApplication, type Page, type TestInfo } from '@playwright/test';
import { readFile, writeFile } from 'node:fs/promises';
import { join } from 'node:path';
import { launch } from './support';

/** Extends the actual synthetic P4 launch scenario; source/engine/evidence remain real. */
export async function verifyRunFollowUp({
  application,
  page,
  profile,
  base,
  info,
}: {
  application: ElectronApplication;
  page: Page;
  profile: string;
  base: string;
  info: TestInfo;
}) {
  const before = JSON.parse(await readFile(join(profile, 'service/catalog.json'), 'utf8'));
  expect(before.runs).toHaveLength(1);
  await page.getByRole('button', { name: 'Account', exact: true }).click();
  await page.getByRole('button', { name: 'Refresh connection', exact: true }).click();
  await expect(page.getByText('ChatGPT connected', { exact: true })).toBeVisible();
  await page.getByRole('button', { name: 'Close Codex account', exact: true }).click();
  await page.getByRole('button', { name: 'Open current design', exact: true }).click();
  await page.getByRole('button', { name: 'Discuss changes', exact: true }).click();
  await expect(page.locator('.chat-composer')).toContainText('proposal will retain a link');
  await page.getByRole('checkbox', { name: 'Allow a Pipeline proposal for this message' }).check();
  await page.getByRole('textbox', { name: 'Message draft' }).fill('pipeline-feedback');
  await page.getByRole('button', { name: 'Send', exact: true }).click();
  const deliveryPath = join(profile, 'codex/home/feedback-delivery.json');
  await expect
    .poll(() => readFile(deliveryPath, 'utf8').catch(() => ''), { timeout: 60000 })
    .not.toBe('');
  const delivery = JSON.parse(await readFile(deliveryPath, 'utf8'));
  expect(delivery.proposed.followUp.runRef).toBe(before.runs[0].runRef);
  expect(delivery.proposed.followUp.evidence[0].evidence.selection.attempt).toBe(1);
  await page.getByRole('button', { name: 'Changes', exact: true }).click();
  const review = page.getByRole('region', { name: 'Pipeline change review' });
  await expect(review.getByText('20 bp', { exact: true })).toBeVisible({ timeout: 180000 });
  await expect(review.getByText('40 bp', { exact: true })).toBeVisible();
  await review.locator('.follow-up-origin summary').click();
  await review.getByRole('button', { name: /Task.*Attempt 1/ }).click();
  await expect(page.getByRole('dialog', { name: /Attachment:/ })).toContainText(
    'Captured Run facts',
  );
  await expect(page.getByRole('dialog', { name: /Attachment:/ }).locator('pre')).toContainText(
    'trim_galore',
  );
  await page.keyboard.press('Escape');
  await page.screenshot({ path: info.outputPath('feedback-review.png') });
  await review.getByRole('button', { name: 'Adopt proposal', exact: true }).click();
  await review.getByRole('button', { name: 'Confirm adoption' }).click();
  await expect(review).toContainText('Adopted as current. No Run was started.');
  expect(
    JSON.parse(await readFile(join(profile, 'service/catalog.json'), 'utf8')).runs,
  ).toHaveLength(1);
  await review.getByRole('button', { name: 'Review a new analysis →', exact: true }).click();
  const prep = page.getByRole('region', { name: 'Run preparation' });
  await prep.getByRole('button', { name: 'Prepare again', exact: true }).click();
  await expect(prep.getByRole('button', { name: 'Check data and tools', exact: true })).toBeVisible(
    { timeout: 150000 },
  );
  await prep.getByRole('button', { name: 'Check data and tools', exact: true }).click();
  const card = page.getByRole('article', { name: 'Confirm analysis', exact: true });
  await expect(card.getByRole('button', { name: 'Start new analysis', exact: true })).toBeEnabled({
    timeout: 90000,
  });
  await expect(card).toContainText('All steps run again');
  await expect(card).toContainText('Same input content');
  await page.screenshot({ path: info.outputPath('feedback-ready.png') });
  await card.getByRole('button', { name: 'Start new analysis', exact: true }).click();
  const run = page.getByRole('article', { name: 'Analysis run', exact: true }).last();
  await expect(run).toContainText('succeeded', { timeout: 180000 });
  await run.getByRole('button', { name: 'Open Run', exact: true }).click();
  const flow = page.getByRole('region', { name: 'Run flow', exact: true });
  await expect(flow.locator('.pipeline-node[data-run-status="succeeded"]')).toHaveCount(2, {
    timeout: 30000,
  });
  await expect(flow).toContainText('Following Analysis 1');
  const after = JSON.parse(await readFile(join(profile, 'service/catalog.json'), 'utf8'));
  expect(after.runs).toHaveLength(2);
  expect(after.runs[0]).toEqual(before.runs[0]);
  const reviews = await page.evaluate(
    async (projectId) => window.gobble.launches.list({ projectId }),
    before.runs[0].projectId,
  );
  expect(reviews.ok).toBe(true);
  if (!reviews.ok) throw Error(reviews.error.message);
  const following = reviews.value.find((v) => v.followUp);
  expect(following?.operation?.runRef).not.toBe(before.runs[0].runRef);
  expect(following?.followUp?.runRef).toBe(before.runs[0].runRef);
  await writeFile(
    info.outputPath('feedback-result.json'),
    JSON.stringify({ fixture: { profile, base }, reviews }, null, 2),
  );
  await page.screenshot({ path: info.outputPath('feedback-run.png') });
  await application.close();
  const restored = await launch(profile, { GOBBLE_CODEX_EXECUTABLE: join(base, 'codex') });
  try {
    await expect(
      restored.page.getByRole('region', { name: 'Run flow', exact: true }),
    ).toContainText('Following Analysis 1');
    await restored.page
      .getByRole('region', { name: 'Run flow', exact: true })
      .locator('.follow-up-origin summary')
      .click();
    await restored.page
      .getByRole('region', { name: 'Run flow', exact: true })
      .getByRole('button', { name: /Task.*Attempt 1/ })
      .click();
    await expect(restored.page.getByRole('dialog', { name: /Attachment:/ })).toContainText(
      'Captured Run facts',
    );
    await expect(
      restored.page.getByRole('dialog', { name: /Attachment:/ }).locator('pre'),
    ).toContainText('trim_galore');
    await restored.page.keyboard.press('Escape');
    await restored.page.screenshot({ path: info.outputPath('feedback-restored.png') });
    await restored.application.evaluate(({ BrowserWindow }) => {
      BrowserWindow.getAllWindows()[0]!.setSize(1100, 800);
    });
    await expect(
      restored.page.getByRole('button', { name: 'Open current design', exact: true }),
    ).toBeVisible();
    await expect
      .poll(() =>
        restored.page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth),
      )
      .toBe(true);
    await restored.page.screenshot({ path: info.outputPath('feedback-compact.png') });
    await restored.page
      .getByRole('navigation', { name: 'Project regions' })
      .getByRole('button', { name: 'Chat', exact: true })
      .click();
    await expect(restored.page.getByRole('textbox', { name: 'Message draft' })).toBeVisible();
    await restored.page.screenshot({ path: info.outputPath('feedback-compact-chat.png') });
  } catch (error) {
    await restored.page
      .screenshot({ path: info.outputPath('restore-failure.png') })
      .catch(() => {});
    await writeFile(info.outputPath('restore-failure.txt'), String(error));
    throw error;
  } finally {
    await restored.application.close();
  }
}
