import { test, expect } from '@playwright/test';
import { mkdtemp, cp, copyFile, chmod, readFile, writeFile, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join, resolve } from 'node:path';
import {
  launch,
  chooseProject,
  readyView,
  openAgentRoster,
  attachmentPreview,
  closeAttachmentPreview,
} from './support';

test('real module flow, exact discussion, independent Agent marks, stale refusal and historical restart', async ({}, info) => {
  test.skip(
    process.env.GOBBLE_FLOW_LIVE !== '1',
    'Requires the owned pinned Linux inspection runtime.',
  );
  test.setTimeout(240_000);
  const base = await mkdtemp(join(tmpdir(), 'gobble-pipeline-discuss-')),
    profile = join(base, 'profile'),
    project = join(base, 'Read quality study'),
    executable = join(base, 'codex');
  await cp(resolve('qualification/pipeline-flow/project'), project, { recursive: true });
  await copyFile(resolve('desktop/tests/fixtures/codex.cjs'), executable);
  await chmod(executable, 0o700);
  await copyFile(
    resolve('desktop/tests/fixtures/pipeline-tools.cjs'),
    join(base, 'pipeline-tools.cjs'),
  );
  let { application, page } = await launch(profile, { GOBBLE_CODEX_EXECUTABLE: executable }).catch(
    async (error) => {
      await rm(base, { recursive: true, force: true });
      throw error;
    },
  );
  try {
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
    primary = await readyView(page);
    await primary.getByRole('button', { name: /^Inspect Trim adapters/ }).click();
    await primary.getByRole('button', { name: 'Quality threshold 25 Phred', exact: true }).focus();
    await primary
      .getByRole('button', { name: 'Quality threshold 25 Phred', exact: true })
      .press('Enter');
    await expect(
      primary.getByRole('region', { name: 'Details for Quality threshold', exact: true }),
    ).toBeVisible();
    await primary.getByRole('button', { name: 'Add to message', exact: true }).click();
    await primary
      .getByRole('button', { name: 'Select declared outputs port trimmed_read1', exact: true })
      .click();
    await expect(
      primary.getByRole('region', { name: 'Details for trimmed_read1', exact: true }),
    ).toBeVisible();
    await page
      .getByLabel('Message attachments', { exact: true })
      .getByRole('button', { name: /^Trim adapters/ })
      .click();
    await expect(attachmentPreview(page)).toContainText('25 Phred');
    await closeAttachmentPreview(page);
    await expect(
      primary.getByRole('region', { name: 'Details for trimmed_read1', exact: true }),
    ).toBeVisible();
    await primary.getByRole('button', { name: 'Zoom in pipeline flow' }).click();
    const zoom = await primary.getByLabel('Flow zoom level').textContent();
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
    const draft = page.getByRole('textbox', { name: 'Message draft' });
    await draft.fill('pipeline-point');
    await page.getByRole('button', { name: 'Send', exact: true }).click();
    await expect(draft).toHaveValue('');
    await draft.fill('Keep my output selection and this draft.');
    const home = join(profile, 'codex/home');
    await writeFile(join(home, 'pipeline-observe'), 'ready');
    await writeFile(join(home, 'pipeline-point'), 'ready');
    await expect(page.locator('.shared-reference-event')).toHaveCount(1);
    await expect(
      page.locator('.message-state').getByText('completed', { exact: true }),
    ).toHaveCount(1);
    await expect(
      primary.getByRole('region', { name: 'Details for trimmed_read1', exact: true }),
    ).toBeVisible();
    await expect(draft).toBeFocused();
    expect(await primary.getByLabel('Flow zoom level').textContent()).toBe(zoom);
    await expect(primary.locator('.pipeline-node[data-agent-mark="true"]')).toHaveCount(1);
    const delivery = JSON.parse(await readFile(join(home, 'pipeline-delivery.json'), 'utf8'));
    expect(
      delivery.observed.content.subjects.some(
        (s: { target: { selection: { subject: { kind: string } } } }) =>
          s.target.selection.subject.kind === 'connection',
      ),
    ).toBe(true);
    await page.screenshot({ path: info.outputPath('pipeline-shared-selection.png') });
    await page.getByRole('button', { name: 'Show in view', exact: true }).click();
    await expect(primary.getByText('Your selection and view are saved.')).toBeVisible();
    await primary.getByRole('button', { name: 'Return to my view', exact: true }).click();
    primary = await readyView(page);
    await expect(
      primary.getByRole('region', { name: 'Details for trimmed_read1', exact: true }),
    ).toBeVisible();
    expect(await primary.getByLabel('Flow zoom level').textContent()).toBe(zoom);
    // Hold an old observation while the User checks changed owned source.
    for (const name of ['pipeline-point', 'pipeline-observed.json', 'pipeline-delivery.json'])
      await rm(join(home, name));
    await draft.fill('pipeline-stale');
    await page.getByRole('button', { name: 'Send', exact: true }).click();
    await expect
      .poll(async () => readFile(join(home, 'pipeline-observed.json'), 'utf8').catch(() => ''))
      .not.toBe('');
    const sourcePath = join(project, 'trim-review/pipeline.go'),
      source = await readFile(sourcePath, 'utf8');
    await writeFile(sourcePath, source.replace('Quality: 25', 'Quality: 30'));
    await primary.getByRole('button', { name: 'Check flow', exact: true }).click();
    await expect(primary.getByText('Checking analysis…')).toBeVisible();
    await expect(primary.getByText('Checking analysis…')).not.toBeVisible({ timeout: 135_000 });
    primary = await readyView(page);
    await writeFile(join(home, 'pipeline-point'), 'ready');
    await expect(
      page.locator('.message-state').getByText('completed', { exact: true }),
    ).toHaveCount(2);
    expect(
      JSON.parse(await readFile(join(home, 'pipeline-delivery.json'), 'utf8')).results[0].success,
    ).toBe(false);
    await expect(page.locator('.shared-reference-event')).toHaveCount(1);
    await expect(primary.locator('.pipeline-node[data-agent-mark="true"]')).toHaveCount(0);
    await page.getByRole('button', { name: 'View captured evidence', exact: true }).click();
    await expect(page.getByRole('dialog', { name: 'Captured evidence' })).toContainText('25 Phred');
    await page.screenshot({ path: info.outputPath('pipeline-historical-evidence.png') });
    await page.keyboard.press('Escape');
    await application.close();
    ({ application, page } = await launch(profile, { GOBBLE_CODEX_EXECUTABLE: executable }));
    primary = await readyView(page);
    await primary.getByRole('button', { name: /^Inspect Trim adapters/ }).click();
    await expect(
      primary.getByRole('button', { name: 'Quality threshold 30 Phred', exact: true }),
    ).toBeVisible();
    await primary.getByRole('button', { name: 'Step list', exact: true }).click();
    await primary
      .getByRole('button', { name: /^Trim adapters/ })
      .first()
      .click();
    await primary.getByRole('button', { name: 'Minimum length 40 bp', exact: true }).click();
    await expect(
      primary.getByRole('region', { name: 'Details for Minimum length', exact: true }),
    ).toBeVisible();
    await application.evaluate(({ BrowserWindow }) =>
      BrowserWindow.getAllWindows()[0]!.setBounds({ width: 940, height: 760 }),
    );
    if (await page.getByRole('button', { name: 'Workspace', exact: true }).isVisible())
      await page.getByRole('button', { name: 'Workspace', exact: true }).click();
    await expect(
      primary.getByRole('region', { name: 'Details for Minimum length', exact: true }),
    ).toBeVisible();
    await page.screenshot({ path: info.outputPath('pipeline-discussion-compact.png') });
    await page.getByRole('button', { name: 'Chat', exact: true }).click();
    await page.getByRole('button', { name: 'View captured evidence', exact: true }).click();
    await expect(page.getByRole('dialog', { name: 'Captured evidence' })).toContainText('25 Phred');
    const catalog = JSON.parse(await readFile(join(profile, 'service/catalog.json'), 'utf8'));
    expect(catalog.runs).toHaveLength(0);
    await writeFile(
      info.outputPath('pipeline-tool-delivery.json'),
      JSON.stringify(delivery, null, 2),
    );
  } finally {
    await application
      .evaluate(({ dialog }) => {
        dialog.showMessageBox = async () => ({ response: 1, checkboxChecked: false });
      })
      .catch(() => {});
    await application.close();
    await rm(base, { recursive: true, force: true });
  }
});
