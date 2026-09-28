import { test, expect } from '@playwright/test';
import { mkdtemp, cp, copyFile, chmod, readFile, writeFile } from 'node:fs/promises';
import { execFileSync } from 'node:child_process';
import { tmpdir } from 'node:os';
import { join, resolve } from 'node:path';
import { launch, chooseProject, readyView, openAgentRoster } from './support';

test('B comparison: scoped Agent proposal, exact chat reference, adoption and restart', async ({}, info) => {
  test.skip(
    process.env.GOBBLE_PROPOSAL_LIVE !== '1',
    'Requires the owned pinned Linux review runtime.',
  );
  test.setTimeout(420_000);
  const base = await mkdtemp(join(tmpdir(), 'gobble-p2b1-live-'));
  const project = join(base, 'Read quality study'),
    profile = join(base, 'profile'),
    executable = join(base, 'codex');
  await cp(resolve('qualification/pipeline-flow/project'), project, { recursive: true });
  const source = await readFile(join(project, 'trim-review/pipeline.go'), 'utf8');
  const image = execFileSync(
    'docker',
    [
      'image',
      'inspect',
      '--format',
      '{{.Id}}',
      process.env.GOBBLE_REVIEW_IMAGE ?? 'gobble-p2b1-review:local',
    ],
    { encoding: 'utf8' },
  ).trim();
  const daemon = execFileSync('docker', ['info', '--format', '{{.ID}}'], {
    encoding: 'utf8',
  }).trim();
  await writeFile(
    join(project, '.gobble-runtime.json'),
    JSON.stringify({ format: 1, image, daemon }),
  );
  await copyFile(resolve('desktop/tests/fixtures/codex.cjs'), executable);
  await chmod(executable, 0o700);
  await copyFile(
    resolve('desktop/tests/fixtures/pipeline-proposal-tools.cjs'),
    join(base, 'pipeline-tools.cjs'),
  );
  await writeFile(
    info.outputPath('fixture.json'),
    JSON.stringify({ base, project, profile, image, daemon }, null, 2),
  );
  let { application, page } = await launch(profile, { GOBBLE_CODEX_EXECUTABLE: executable });
  try {
    page.setDefaultTimeout(15000);
    await chooseProject(application, page, project);
    await application.evaluate(
      ({ dialog }, folder) => {
        dialog.showOpenDialog = async () => ({ canceled: false, filePaths: [folder] });
      },
      join(project, 'trim-review'),
    );
    await page.getByRole('button', { name: 'Import pipeline', exact: true }).click();
    let primary = await readyView(page);
    await primary.getByRole('button', { name: 'Check flow', exact: true }).click();
    await expect(primary.getByText('2 steps · 1 input · 2 connections')).toBeVisible({
      timeout: 135_000,
    });
    await readyView(page);
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
    await page
      .getByRole('combobox', { name: 'Conversation recipient' })
      .selectOption({ label: 'To Researcher' });
    await primary.getByRole('button', { name: 'Discuss changes', exact: true }).click();
    await page
      .getByRole('checkbox', { name: 'Allow a Pipeline proposal for this message' })
      .check();
    await page.getByRole('textbox', { name: 'Message draft' }).fill('pipeline-proposal');
    await page.getByRole('button', { name: 'Send', exact: true }).click();
    const home = join(profile, 'codex/home');
    await expect
      .poll(async () => readFile(join(home, 'proposal-delivery.json'), 'utf8').catch(() => ''), {
        timeout: 30_000,
      })
      .not.toBe('');
    await primary.getByRole('button', { name: 'Changes', exact: true }).click();
    const review = page.getByRole('region', { name: 'Pipeline change review' });
    await expect(review.getByText('30 Phred', { exact: true })).toBeVisible({ timeout: 260_000 });
    await expect(review.getByText('25 Phred', { exact: true })).toBeVisible();
    await expect(review.locator('.pipeline-node[data-change="setting"]')).toHaveCount(1);
    await expect(review.locator('.pipeline-node[data-change="unchanged"]')).toHaveCount(2);
    for (const node of await review.locator('.pipeline-node[data-change="unchanged"]').all())
      await expect(node).toBeDisabled();
    await expect(review.locator('.pipeline-node[data-change="setting"]')).toHaveCSS(
      'background-color',
      'rgb(255, 249, 233)',
    );
    const addedEdge = review.locator('g[data-added-edge="true"] [role="button"]');
    await addedEdge.focus();
    await page.keyboard.press('Enter');
    await expect(review.getByText('Quality check added', { exact: true })).toBeVisible();
    await review.getByRole('button', { name: '1 Quality threshold' }).click();
    await page.screenshot({ path: info.outputPath('b-setting-review.png') });
    await review.getByRole('button', { name: '2 Add quality check' }).click();
    await expect(review.getByText('No quality check here', { exact: true })).toBeVisible();
    await expect(review.getByText('Quality check added', { exact: true })).toBeVisible();
    await page.screenshot({ path: info.outputPath('b-added-step-review.png') });
    await review.getByRole('button', { name: '1 Quality threshold' }).click();
    await review.getByRole('button', { name: 'Discuss this change' }).click();
    // Engine checking can finish while another app is foreground. Shared pointing
    // requires the actual Project window, unlike retained comparison reads.
    await application.evaluate(({ BrowserWindow }) => {
      const window = BrowserWindow.getAllWindows()[0]!;
      window.show();
      window.focus();
    });
    await expect
      .poll(() =>
        application.evaluate(({ BrowserWindow }) => BrowserWindow.getAllWindows()[0]!.isFocused()),
      )
      .toBe(true);
    await page.getByRole('textbox', { name: 'Message draft' }).fill('pipeline-review');
    await page.getByRole('button', { name: 'Send', exact: true }).click();
    await expect
      .poll(async () => readFile(join(home, 'review-delivery.json'), 'utf8').catch(() => ''), {
        timeout: 30_000,
      })
      .not.toBe('');
    await expect(review.getByRole('button', { name: /Agent reference/ })).toBeVisible();
    await page
      .getByRole('textbox', { name: 'Message draft' })
      .fill('Keep this unsent note while adopting.');
    await review.getByRole('button', { name: 'Adopt proposal', exact: true }).click();
    await expect(review).toContainText('app-managed source copy');
    await review.getByRole('button', { name: 'Confirm adoption' }).click();
    await expect(
      page.getByText('Adopted as current. No Run was started.', { exact: true }),
    ).toBeVisible();
    expect(await readFile(join(project, 'trim-review/pipeline.go'), 'utf8')).toBe(source);
    await page.screenshot({ path: info.outputPath('b-adopted-review.png') });
    await application.close();
    ({ application, page } = await launch(profile, { GOBBLE_CODEX_EXECUTABLE: executable }));
    primary = await readyView(page);
    await expect(page.getByRole('textbox', { name: 'Message draft' })).toHaveValue(
      'Keep this unsent note while adopting.',
    );
    await primary.getByRole('button', { name: 'Changes', exact: true }).click();
    await expect(page.getByRole('region', { name: 'Pipeline change review' })).toContainText(
      'This is the current version. No Run was started.',
    );
    await expect(page.getByText('25 Phred', { exact: true })).toBeVisible();
    await expect(page.getByText('30 Phred', { exact: true })).toBeVisible();
    const catalog = JSON.parse(await readFile(join(profile, 'service/catalog.json'), 'utf8'));
    expect(catalog.schemaVersion).toBe(5);
    expect(catalog.runs).toHaveLength(0);
    expect(Object.keys(catalog.revisions)).toEqual([catalog.pipelines[0].pipelineId]);
    expect(catalog.drafts).toEqual({});
    await page.setViewportSize({ width: 850, height: 900 });
    await page.screenshot({ path: info.outputPath('b-compact-review.png') });
  } finally {
    await application.close();
  }
});
