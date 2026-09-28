import { expect, test, type ElectronApplication, type Page } from '@playwright/test';
import { mkdtemp, mkdir, readFile, rm, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { chooseProject, launch, projectFixture, readyView } from './support';

let application: ElectronApplication;
let page: Page;
let directory: string;
let profile: string;
let project: string;

const source = `package rnaseq

import gobble "github.com/HahyeonJeon/gobble"

// The Agent authors this pipeline; the App shows its checked flow.
func Pipeline() *gobble.Pipeline {
    return gobble.NewPipeline("RNA analysis")
}
`;

test.beforeEach(async () => {
  directory = await mkdtemp(join(tmpdir(), 'gobble-pipeline-e2e-'));
  profile = join(directory, 'profile');
  project = await projectFixture(directory);
  await mkdir(join(project, 'RNA analysis'));
  await writeFile(join(project, 'RNA analysis/pipeline.go'), source);
  ({ application, page } = await launch(profile));
  await chooseProject(application, page, project);
});
test.afterEach(async () => {
  await application?.evaluate(({ dialog }) => {
    dialog.showMessageBox = async () => ({ response: 1, checkboxChecked: false });
  });
  await application?.close();
  await rm(directory, { recursive: true, force: true });
});

test('import opens a flow, preserves the other pane and draft, and restores Pipeline identity', async ({}, info) => {
  const draft = page.getByRole('textbox', { name: 'Message draft' });
  await draft.fill('Review the RNA pipeline structure with me.');
  await page.getByRole('button', { name: 'Open quality.png in the other pane' }).click();
  await readyView(page, 'Secondary pane');
  await page.getByRole('region', { name: 'Primary pane', exact: true }).click();
  await application.evaluate(
    ({ dialog }, folder) => {
      dialog.showOpenDialog = async () => ({ canceled: false, filePaths: [folder] });
    },
    join(project, 'RNA analysis'),
  );
  const importButton = page.getByRole('button', { name: 'Import pipeline', exact: true });
  await importButton.focus();
  await page.keyboard.press('Enter');
  await expect(
    page.getByRole('button', { name: 'Open RNA analysis pipeline', exact: true }),
  ).toBeVisible();
  const primary = await readyView(page);
  await expect(primary.getByRole('heading', { name: 'See how this analysis works' })).toBeVisible();
  await expect(primary.getByRole('textbox', { name: 'pipeline.go text' })).toHaveCount(0);
  await primary.getByRole('button', { name: 'Check flow', exact: true }).click();
  await expect(primary.getByRole('status', { name: 'Pipeline check status' })).toContainText(
    'analysis runtime is not ready',
  );
  await expect(draft).toHaveValue('Review the RNA pipeline structure with me.');
  await expect(page.getByRole('button', { name: 'Start run', exact: true })).toHaveCount(0);
  const catalogPath = join(profile, 'service/catalog.json');
  const before = JSON.parse(await readFile(catalogPath, 'utf8'));
  expect(before.pipelines).toHaveLength(1);
  expect(before.runs).toHaveLength(0);
  await page.screenshot({ path: info.outputPath('pipeline-setup.png') });
  await application.close();
  ({ application, page } = await launch(profile));
  await readyView(page);
  await readyView(page, 'Secondary pane');
  await expect(page.getByRole('textbox', { name: 'Message draft' })).toHaveValue(
    'Review the RNA pipeline structure with me.',
  );
  const restored = JSON.parse(await readFile(catalogPath, 'utf8'));
  expect(restored.pipelines).toEqual(before.pipelines);
  expect(await readFile(join(project, 'RNA analysis/pipeline.go'), 'utf8')).toBe(source);
  await application.evaluate(({ BrowserWindow }) =>
    BrowserWindow.getAllWindows()[0]!.setBounds({ width: 900, height: 700 }),
  );
  await page.screenshot({ path: info.outputPath('pipeline-compact.png') });
});

test('registration errors stay visible and retries do not create incomplete definitions', async () => {
  await writeFile(join(project, 'RNA analysis/pipeline.go'), 'package rnaseq\n');
  await page.getByRole('button', { name: 'RNA analysis', exact: true }).click();
  const register = page.getByRole('button', { name: 'Register pipeline', exact: true });
  await register.click();
  await expect(page.getByRole('alert')).toContainText('No Pipeline() entry point');
  await expect(register).toBeEnabled();
  await writeFile(join(project, 'RNA analysis/pipeline.go'), source);
  await register.click();
  await expect(
    page.getByRole('button', { name: 'Open RNA analysis pipeline', exact: true }),
  ).toBeVisible();
  const catalog = JSON.parse(await readFile(join(profile, 'service/catalog.json'), 'utf8'));
  expect(catalog.pipelines).toHaveLength(1);
  expect(catalog.runs).toHaveLength(0);
});
