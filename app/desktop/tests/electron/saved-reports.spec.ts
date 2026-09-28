import { test, expect, type ElectronApplication } from '@playwright/test';
import { readFile, writeFile, mkdir, unlink } from 'node:fs/promises';
import { join, dirname, resolve } from 'node:path';
import { createHash } from 'node:crypto';
import { continuationFixture } from './continuation-fixture';
import {
  launch,
  chooseProject,
  openAgentRoster,
  attachmentPreview,
  closeAttachmentPreview,
  focusWindow,
} from './support';

// Real App/Main/native file verification and original FastQC bytes; the engine query peer is isolated.
// The preceding output-evidence slice separately qualifies an actual engine/tool Run.
test('Flow opens a complete saved report; compact/focus/close/restart preserve its original result', async ({}, info) => {
  test.setTimeout(120_000);
  const fixture = await continuationFixture(info.outputPath('fixture'));
  const html = await readFile(resolve('desktop/tests/fixtures/fastqc-0.12.1.html'));
  const source = join(fixture.root, fixture.original.value.outputPath, 'results/quality.html');
  await mkdir(dirname(source), { recursive: true });
  await writeFile(source, html);
  const evidence = {
    schemaVersion: 1,
    runId: 'fixture-run',
    snapshot: fixture.review.snapshot,
    originDigest: fixture.review.originDigest,
    instance: 'qc',
    attempt: 1,
    port: 'html',
    recipe: 'fastqc-v1',
    path: 'results/quality.html',
    size: html.length,
    sha256: 'sha256:' + createHash('sha256').update(html).digest('hex'),
  };
  const peerPath = join(fixture.base, 'bin/docker');
  let peer = await readFile(peerPath, 'utf8');
  peer = peer
    .replace(
      "if (args[0] === 'context')",
      `if (args[0] === 'image' && joined.includes('output-evidence.version')) out('1');
else if (joined.includes('output-capabilities')) out({schemaVersion:1,outputEvidenceVersion:1});
else if (joined.includes('output-evidence')) out(${JSON.stringify(evidence)});
else if (args[0] === 'context')`,
    )
    .replaceAll("receipt && !exists('stopped') ? 'running' : 'stopped'", "'succeeded'")
    .replace("status: receipt ? 'running' : 'canceled'", "status: 'succeeded'")
    .replace('attempt: receipt ? 2 : 1', "attempt: exists('new-attempt') ? 2 : 1");
  await writeFile(peerPath, peer);
  let application: ElectronApplication | undefined;
  try {
    let launched = await launch(fixture.profile, fixture.environment);
    application = launched.application;
    let page = launched.page;
    await application.evaluate(({ BrowserWindow }) =>
      BrowserWindow.getAllWindows()[0]!.setSize(1600, 1000),
    );
    await chooseProject(application, page, fixture.root);
    await page.getByRole('button', { name: 'Analysis 1', exact: true }).click();
    const flow = page.getByRole('region', { name: 'Run flow', exact: true });
    await expect(
      flow.getByRole('button', { name: 'Open quality report', exact: true }),
    ).toBeEnabled();
    await flow.getByRole('button', { name: 'Open quality report', exact: true }).click();
    const report = page.locator('.report-view');
    await expect(report).toBeVisible();
    await expect(report.locator('img')).toHaveCount(8);
    await expect
      .poll(() =>
        report
          .locator('img')
          .evaluateAll((imgs) =>
            imgs.every(
              (i) => (i as HTMLImageElement).complete && (i as HTMLImageElement).naturalWidth > 0,
            ),
          ),
      )
      .toBe(true);
    await expect(report.getByRole('cell', { name: '100', exact: true })).toHaveCount(2);
    await expect(
      report.getByRole('button', { name: 'Producer logs · attempt 1', exact: true }),
    ).toBeVisible();
    await expect(report.getByRole('button', { name: 'Attach report', exact: true })).toBeEnabled();
    expect(await report.locator('a,script,iframe').count()).toBe(0);
    await page.screenshot({ path: info.outputPath('report-stacked.png') });
    await report.getByRole('combobox', { name: 'Report section' }).selectOption('M1');
    await expect(
      report.getByRole('img', { name: 'Per base quality graph', exact: true }),
    ).toBeInViewport();
    await page.screenshot({ path: info.outputPath('report-chart.png') });
    const pane = page.locator('.work-pane').filter({ has: report });
    await pane.getByRole('button', { name: 'Maximize secondary pane', exact: true }).click();
    await expect(page.getByRole('button', { name: 'Restore panes', exact: true })).toBeVisible();
    await expect(page.locator('#primary-pane')).toHaveCount(0);
    await expect(report.getByRole('combobox', { name: 'Report section' })).toHaveValue('M1');
    await expect(
      report.getByRole('img', { name: 'Per base quality graph', exact: true }),
    ).toBeInViewport();
    await page.screenshot({ path: info.outputPath('report-focused.png') });
    await application.evaluate(({ BrowserWindow }) =>
      BrowserWindow.getAllWindows()[0]!.setSize(1000, 760),
    );
    await expect(report.getByRole('combobox', { name: 'Report section' })).toBeVisible();
    await page.screenshot({ path: info.outputPath('report-compact.png') });
    await expect
      .poll(() => report.evaluate((el) => el.scrollWidth <= el.clientWidth + 1))
      .toBe(true);
    const state = await page.evaluate(async (projectId) => {
      const s = await window.gobble.workspace.read({ projectId });
      if (!s.ok) throw Error(s.error.message);
      return s.value;
    }, fixture.projectId);
    const saved = state.savedReports![0]!;
    const savedBytes = await readFile(
      join(
        fixture.profile,
        'workspace/evidence',
        fixture.projectId,
        saved.asset.hash.slice(7) + '.blob',
      ),
    );
    expect(createHash('sha256').update(savedBytes).digest('hex')).toBe(saved.asset.hash.slice(7));
    expect(JSON.parse(savedBytes.toString()).content.modules).toHaveLength(10);
    await pane
      .getByRole('button', { name: 'Close sample_trimmed.fq.gz FastQC Report', exact: true })
      .click();
    await unlink(source);
    await writeFile(join(fixture.base, 'new-attempt'), 'yes');
    await application.close();
    application = undefined;
    launched = await launch(fixture.profile, fixture.environment);
    application = launched.application;
    page = launched.page;
    await application.evaluate(({ BrowserWindow }) =>
      BrowserWindow.getAllWindows()[0]!.setSize(1600, 1000),
    );
    await page.getByRole('button', { name: 'Analysis 1', exact: true }).click();
    const restored = page.getByRole('region', { name: 'Run flow', exact: true });
    await restored.getByText('Saved reports (1)', { exact: true }).click();
    await restored
      .getByRole('button', { name: 'sample_trimmed.fq.gz FastQC Report · attempt 1', exact: true })
      .click();
    await expect(page.locator('.report-view img')).toHaveCount(8);
    const doc = await page.evaluate(async (projectId) => {
      const s = await window.gobble.workspace.read({ projectId });
      if (!s.ok) throw Error(s.error.message);
      return s.value;
    }, fixture.projectId);
    expect(doc.savedReports).toEqual([saved]);
    await page.screenshot({ path: info.outputPath('report-restored.png') });
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
    await dialog.getByLabel('Project access').selectOption('sharedViews');
    await dialog.getByRole('button', { name: 'Add agent', exact: true }).click();
    await page
      .getByRole('combobox', { name: 'Conversation recipient' })
      .selectOption({ label: 'To Researcher' });
    await page.getByRole('button', { name: 'Attach report', exact: true }).click();
    await expect(
      page.getByText('1 attachment · Ready for Researcher', { exact: true }),
    ).toBeVisible();
    await page
      .getByLabel('Message attachments', { exact: true })
      .getByRole('button', { name: /^sample_trimmed/ })
      .click();
    await expect(attachmentPreview(page).locator('.report-view img')).toHaveCount(8);
    await page.screenshot({ path: info.outputPath('report-attachment.png') });
    await closeAttachmentPreview(page);
    await page.getByRole('button', { name: 'Point to report', exact: true }).click();
    await expect(page.getByLabel('Report pointers')).toContainText('You · Whole report');
    await focusWindow(application);
    await expect(page.locator('dialog[open]')).toHaveCount(0);
    await page.getByRole('textbox', { name: 'Message draft' }).fill('report-read');
    await page.getByRole('button', { name: 'Send', exact: true }).click();
    await expect(
      page.locator('.message-state').getByText('completed', { exact: true }),
    ).toHaveCount(1);
    await expect(page.getByLabel('Report pointers')).toContainText('Researcher · Whole report');
    const delivery = JSON.parse(
      await readFile(join(fixture.profile, 'codex/home/report-delivery.json'), 'utf8'),
    );
    const retained = JSON.parse(savedBytes.toString('utf8'));
    const originals = retained.content.modules
      .flatMap((m: { blocks: { kind: string; id?: string; base64?: string }[] }) => m.blocks)
      .filter((b: { kind: string }) => b.kind === 'image');
    expect(delivery.charts).toHaveLength(8);
    expect(JSON.stringify(delivery.attached)).not.toContain('base64');
    expect(delivery.reading.report.content.modules).toHaveLength(10);
    for (const chart of delivery.charts) {
      const original = originals.find((b: { id: string }) => b.id === chart.id);
      expect(
        chart.result.contentItems.find((i: { type: string }) => i.type === 'inputImage').imageUrl,
      ).toBe('data:image/png;base64,' + original.base64);
    }
    expect(delivery.reading.pointable).toBe(false);
    expect(delivery.observed.receipt.pointable).toBe(true);
    expect(delivery.observed.content.imagesReturned).toEqual([]);
    await page.screenshot({ path: info.outputPath('report-discussion.png') });

    await page.getByRole('button', { name: 'Producer logs · attempt 1', exact: true }).click();
    await expect(
      page
        .getByText(/attempt.*changed|attempt.*unavailable|no longer.*attempt|requested attempt/i)
        .first(),
    ).toBeVisible();
    expect(
      (
        await readFile(
          join(
            fixture.profile,
            'workspace/evidence',
            fixture.projectId,
            saved.asset.hash.slice(7) + '.blob',
          ),
        )
      ).equals(savedBytes),
    ).toBe(true);
  } catch (error) {
    const page = application && (await application.windows())[0];
    await page?.screenshot({ path: info.outputPath('failure.png') }).catch(() => {});
    throw error;
  } finally {
    await application?.close();
  }
});
