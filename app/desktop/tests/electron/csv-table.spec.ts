import { expect, test, type ElectronApplication, type Page } from '@playwright/test';
import { mkdtemp, mkdir, readFile, rm, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { parseWorkspaceDocument } from '@gobble/contracts';
import { addLegacyChart } from '../fixtures/legacy-chart';
import { launch, chooseProject, readyView, captureWindow } from './support';

let application: ElectronApplication, page: Page, base: string, profile: string, projectId: string;
async function document() {
  const result = await page.evaluate(
    (projectId) => window.gobble.workspace.read({ projectId }),
    projectId,
  );
  if (!result.ok) throw new Error(result.error.message);
  return result.value;
}
test.beforeEach(async () => {
  base = await mkdtemp(join(tmpdir(), 'gobble-csv-table-'));
  profile = join(base, 'profile');
  const project = join(base, 'CSV study');
  await mkdir(project);
  await writeFile(join(project, 'samples.csv'), 'Sample,X,Y\nSame,1,4\nSame,3,2\nS03,5,6\n');
  ({ application, page } = await launch(profile));
  await chooseProject(application, page, project);
  projectId = JSON.parse(
    await readFile(join(profile, 'workspace/window.json'), 'utf8'),
  ).activeProjectId;
  await page.getByRole('button', { name: 'samples.csv', exact: true }).click();
  await readyView(page);
});
test.afterEach(async () => {
  await application?.close();
  if (base) await rm(base, { recursive: true, force: true });
});

test('CSV stays a table: exact row selection, settings and attachment without chart creation', async ({}, info) => {
  await page
    .getByRole('textbox', { name: 'Message draft', exact: true })
    .fill('Compare these samples.');
  await expect(
    page.getByRole('button', { name: /scatter|linked plot|plot settings/i }),
  ).toHaveCount(0);
  await page.locator('[data-row-key="row_2"] input[type="checkbox"]').check();
  const before = await document();
  expect(before.selections[0]?.evidence.selection).toMatchObject({
    kind: 'table',
    rowKeys: ['row_2'],
  });
  await page.getByRole('button', { name: 'Table settings', exact: true }).click();
  const dialog = page.getByRole('dialog', { name: 'Table settings', exact: true });
  const x = await dialog
    .getByRole('combobox', { name: 'Sort by', exact: true })
    .locator('option')
    .filter({ hasText: /^X$/ })
    .getAttribute('value');
  await dialog.getByRole('combobox', { name: 'Sort by', exact: true }).selectOption(x!);
  await dialog.getByRole('combobox', { name: 'Direction', exact: true }).selectOption('descending');
  await dialog.getByRole('button', { name: 'Apply', exact: true }).click();
  await readyView(page);
  expect((await document()).selections[0]?.evidence.selection).toMatchObject({
    rowKeys: ['row_2'],
  });
  await page.getByRole('button', { name: 'Add to message from samples.csv', exact: true }).click();
  await expect.poll(async () => (await document()).chat.attachments?.length).toBe(1);
  const saved = await document();
  expect(saved.chat.attachments?.[0]?.evidence.selection).toMatchObject({
    kind: 'table',
    rowKeys: ['row_2'],
  });
  expect(saved.chat.draft).toBe('Compare these samples.');
  expect(saved.collaboration?.submissions ?? []).toHaveLength(0);
  expect(saved.workspace.surfaces.every((s) => s.view === 'table')).toBe(true);
  await expect(page.getByLabel('Message attachments', { exact: true })).toBeVisible();
  await expect(
    page.getByRole('button', { name: 'Add to message from samples.csv', exact: true }),
  ).toBeEnabled();
  await page.evaluate(
    () =>
      new Promise<void>((resolve) =>
        requestAnimationFrame(() => requestAnimationFrame(() => resolve())),
      ),
  );
  await captureWindow(application, info.outputPath('csv-table.png'));
});

test('retired renderer requests are refused before changing the workspace', async () => {
  const before = await document();
  const result = await page.evaluate(
    async (doc) =>
      window.gobble.workspace.command({
        projectId: doc.workspace.projectId,
        expectedRevision: doc.workspace.revision,
        requestId: 'req_removed',
        action: {
          kind: 'openScatter',
          surfaceId: doc.workspace.surfaces[0]!.surfaceId,
          spec: { xColumnId: 'col_2', yColumnId: 'col_3', xScale: 'linear', yScale: 'linear' },
          acknowledgment: {
            schemaVersion: 3,
            projectId: doc.workspace.projectId,
            surfaceId: doc.workspace.surfaces[0]!.surfaceId,
            rendererSessionId: 'rnd_old',
            requestId: 'req_old',
            generation: 1,
            dataRevision: 'old',
            presentation: { spec: 0, view: 0, filter: 0 },
          },
        },
      }),
    before,
  );
  expect(result.ok).toBe(false);
  if (!result.ok) expect(result.error.code).toBe('unsupported');
  expect(await document()).toEqual(before);
});

test('old chart tabs retain metadata on restart and offer their source table', async ({}, info) => {
  const doc = await document(),
    resource = doc.workspace.surfaces[0]!.resource;
  if (resource.kind !== 'file') throw new Error('CSV required');
  const read = await page.evaluate((input) => window.gobble.files.read(input), {
    projectId,
    resourceId: resource.resourceId,
  });
  if (!read.ok || read.value.content.kind !== 'table') throw new Error('CSV required');
  const columns = read.value.content.columns;
  addLegacyChart(
    doc,
    { xColumnId: columns[1]!.id, yColumnId: columns[2]!.id, xScale: 'linear', yScale: 'linear' },
    read.value.revision,
  );
  const legacy = parseWorkspaceDocument(doc);
  await application.close();
  await writeFile(
    join(profile, 'workspace/projects', projectId + '.json'),
    JSON.stringify(legacy) + '\n',
  );
  ({ application, page } = await launch(profile));
  await expect(
    page.getByRole('heading', { name: 'Chart view retired', exact: true }),
  ).toBeVisible();
  await expect(page.getByRole('button', { name: 'Open source table', exact: true })).toBeVisible();
  await readyView(page, 'Secondary pane');
  await captureWindow(application, info.outputPath('legacy-chart-recovery.png'));
  await page.getByRole('button', { name: 'Open source table', exact: true }).click();
  const after = await document();
  expect(after.workspace.surfaces.filter((s) => s.view === 'scatter')).toEqual(
    legacy.workspace.surfaces.filter((s) => s.view === 'scatter'),
  );
  expect(after.workspace.surfaces.filter((s) => s.view === 'table')).toHaveLength(1);
  expect(await page.locator('.scatter-chart').count()).toBe(0);
});
