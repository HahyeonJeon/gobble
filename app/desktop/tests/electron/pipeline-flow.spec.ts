import { expect, test } from '@playwright/test';
import { cp, mkdtemp, readFile, rm, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join, resolve } from 'node:path';
import { chooseProject, launch, readyView } from './support';

test('real checked branching flow, keyboard details, preserved draft and historical restart', async ({}, info) => {
  test.skip(
    process.env.GOBBLE_FLOW_LIVE !== '1',
    'Requires the owned pinned Linux qualification runtime.',
  );
  test.setTimeout(180_000);
  const directory = await mkdtemp(join(tmpdir(), 'gobble-flow-native-'));
  const project = join(directory, 'Flow review');
  const profile = join(directory, 'profile');
  await cp(resolve('qualification/pipeline-flow/project'), project, { recursive: true });
  await cp(resolve('desktop/tests/electron/fixtures/quality.png'), join(project, 'quality.png'));
  const source = await readFile(join(project, 'rnaseq/pipeline.go'), 'utf8');
  let { application, page } = await launch(profile);
  try {
    await chooseProject(application, page, project);
    const draft = page.getByRole('textbox', { name: 'Message draft' });
    await draft.fill('Can we compare quality checking with alignment?');
    await page.getByRole('button', { name: 'Open quality.png in the other pane' }).click();
    await readyView(page, 'Secondary pane');
    await page.getByRole('region', { name: 'Primary pane', exact: true }).click();
    await application.evaluate(
      ({ dialog }, folder) => {
        dialog.showOpenDialog = async () => ({ canceled: false, filePaths: [folder] });
      },
      join(project, 'rnaseq'),
    );
    await page.getByRole('button', { name: 'Import pipeline', exact: true }).click();
    let primary = await readyView(page);
    await primary.getByRole('button', { name: 'Check flow', exact: true }).click();
    await expect(primary.getByText('5 steps · 2 inputs · 7 connections')).toBeVisible({
      timeout: 135_000,
    });
    await readyView(page);
    await expect(primary.getByLabel('Flow colors', { exact: true })).toContainText('Selected');
    expect(await primary.locator('.pipeline-node-icon svg').count()).toBe(7);
    const paths = await primary
      .locator('.pipeline-edge')
      .evaluateAll((items) => items.map((item) => item.getAttribute('d') ?? ''));
    expect(paths).toHaveLength(7);
    expect(paths.some((path) => path.includes('Q'))).toBe(true);
    expect(paths.every((path) => !path.includes('C'))).toBe(true);
    await page.screenshot({ path: info.outputPath('pipeline-flow.png') });
    const trim = primary.getByRole('button', {
      name: 'Inspect Trim adapters · 1 input · 1 output',
      exact: true,
    });
    await primary.getByRole('button', { name: 'Maximize primary pane', exact: true }).click();
    await expect(primary.getByRole('button', { name: 'Restore panes', exact: true })).toBeVisible();
    primary = await readyView(page);
    await trim.focus();
    await expect(trim).toBeFocused();
    await trim.press('Enter');
    await expect(trim).toHaveAttribute('aria-pressed', 'true');
    await expect(primary.getByRole('region', { name: 'Details for Trim adapters' })).toContainText(
      '4g memory',
    );
    await expect(primary.getByRole('region', { name: 'Details for Trim adapters' })).toContainText(
      'trimmed',
    );
    await page.screenshot({ path: info.outputPath('pipeline-flow-selected.png') });
    await primary
      .getByRole('button', {
        name: 'Trim adapters · trimmed → Check read quality · reads',
        exact: true,
      })
      .last()
      .click();
    await expect(primary.getByRole('region', { name: 'Details for Connection' })).toBeVisible();
    await primary.getByRole('button', { name: 'Step list', exact: true }).click();
    await expect(
      primary.getByRole('button', {
        name: 'Trim adapters · trimmed → Align reads · reads',
        exact: true,
      }),
    ).toBeVisible();
    await primary.getByRole('button', { name: 'Flow', exact: true }).click();
    await primary.getByRole('button', { name: 'Restore panes', exact: true }).click();
    await expect(
      primary.getByRole('button', { name: 'Maximize primary pane', exact: true }),
    ).toBeVisible();
    primary = await readyView(page);
    await expect(draft).toHaveValue('Can we compare quality checking with alignment?');
    const catalog = JSON.parse(await readFile(join(profile, 'service/catalog.json'), 'utf8'));
    expect(catalog.runs).toHaveLength(0);
    const statePath = join(
      profile,
      'service/pipeline-inspections',
      catalog.pipelines[0].pipelineId,
      'state.json',
    );
    const checked = JSON.parse(await readFile(statePath, 'utf8'));
    expect(checked.artifact.flow.connections).toHaveLength(7);
    await page.screenshot({ path: info.outputPath('pipeline-flow.png') });
    await page.getByRole('button', { name: 'Open rnaseq pipeline in the other pane' }).click();
    const secondary = await readyView(page, 'Secondary pane');
    await expect(secondary.getByText('5 steps · 2 inputs · 7 connections')).toBeVisible();
    await expect(primary.getByText('5 steps · 2 inputs · 7 connections')).toBeVisible();
    await expect(secondary.getByRole('tab', { name: 'quality.png', exact: true })).toBeVisible();
    await secondary.getByRole('button', { name: 'Close rnaseq', exact: true }).click();
    await readyView(page, 'Secondary pane');
    await expect(draft).toHaveValue('Can we compare quality checking with alignment?');
    // A later failed candidate must retain the previous flow and source identity.
    await writeFile(join(project, 'rnaseq/.gobble-inspection.json'), '{}');
    await primary.getByRole('button', { name: 'Check flow', exact: true }).click();
    await expect(primary.getByRole('status', { name: 'Pipeline check status' })).toContainText(
      'inspection setup is unsupported',
    );
    await expect(trim).toBeVisible();
    expect(JSON.parse(await readFile(statePath, 'utf8')).artifact.artifactId).toBe(
      checked.artifact.artifactId,
    );
    await application.close();
    ({ application, page } = await launch(profile));
    primary = await readyView(page);
    await expect(
      primary.getByRole('button', {
        name: 'Inspect Align reads · 2 inputs · 1 output',
        exact: true,
      }),
    ).toBeVisible();
    await expect(page.getByRole('textbox', { name: 'Message draft' })).toHaveValue(
      'Can we compare quality checking with alignment?',
    );
    await readyView(page, 'Secondary pane');
    await application.evaluate(({ BrowserWindow }) =>
      BrowserWindow.getAllWindows()[0]!.setBounds({ width: 940, height: 760 }),
    );
    await primary
      .getByRole('button', { name: 'Inspect Trim adapters · 1 input · 1 output', exact: true })
      .click();
    await expect(primary.getByRole('region', { name: 'Details for Trim adapters' })).toBeVisible();
    await page.screenshot({ path: info.outputPath('pipeline-flow-compact.png') });
    expect(await readFile(join(project, 'rnaseq/pipeline.go'), 'utf8')).toBe(source);
  } finally {
    await application
      .evaluate(({ dialog }) => {
        dialog.showMessageBox = async () => ({ response: 1, checkboxChecked: false });
      })
      .catch(() => {});
    await application.close();
    await rm(directory, { recursive: true, force: true });
  }
});
