import { addLegacyChart } from './fixtures/legacy-chart';
import { describe, expect, it } from 'vitest';
import { mkdtemp, mkdir, readFile, writeFile, rm } from 'node:fs/promises';
import { join } from 'node:path';
import { tmpdir } from 'node:os';
import {
  captureScatterPresentation,
  readStoredWorkspace,
  type DraftAttachment,
  type SurfaceData,
  type TableContent,
} from '@gobble/contracts';
import { WorkspaceDocumentV2Schema } from '../../contracts/src/workspace-document-v2';
import { WorkspaceDocumentV3Schema } from '../../contracts/src/workspace-document-v3';
import { WorkspaceDocumentV4Schema } from '../../contracts/src/workspace-document-v4';
import { WorkspaceDocumentV5Schema } from '../../contracts/src/workspace-document-v5';
import { WorkspaceDocumentV7Schema } from '../../contracts/src/workspace-document-v7';
import { WorkspaceDocumentV6Schema } from '../../contracts/src/workspace-document-v6';
import {
  materializeEvidence,
  evidencePreview,
  evidenceInput,
  contentHash,
} from '../src/main/evidence/materialize';
import { emptyWorkspace, transition } from '../src/main/workspace/model';
import { WorkspaceStorage } from '../src/main/workspace/storage';

const content: TableContent = {
  kind: 'table',
  columns: [
    { id: 'x', name: 'X' },
    { id: 'y', name: 'Y' },
    { id: 'label', name: 'Sample' },
  ],
  rows: [
    { key: 'one', cells: ['1', '2', 'Same'] },
    { key: 'two', cells: ['3', '4', 'Same'] },
  ],
  truncated: false,
};
const data: SurfaceData = {
  kind: 'file',
  value: {
    projectId: 'prj_test',
    resourceId: 'res_csv',
    name: 'test.csv',
    size: 50,
    revision: 'version-1',
    content,
  },
};
function linked() {
  const doc = transition(
    emptyWorkspace('prj_test'),
    {
      kind: 'open',
      resource: { kind: 'file', resourceId: 'res_csv' },
      pane: 'primary',
      duplicate: false,
    },
    { surfaceId: 'srf_table', view: 'table', title: 'test.csv' },
  );
  addLegacyChart(
    doc,
    { xColumnId: 'x', yColumnId: 'y', labelColumnId: 'label', xScale: 'linear', yScale: 'linear' },
    'version-1',
  );
  return doc;
}
const noImage = async () => {
  throw new Error('No image rendering expected');
};
it('preserves frozen legacy row/column identity and plot context independently of later view changes; binds context into evidence bytes', async () => {
  const doc = linked(),
    plot = doc.workspace.surfaces.find((s) => s.view === 'scatter')!;
  const presentation = captureScatterPresentation(plot, doc.viewLinks[0], content)!;
  doc.viewLinks[0]!.rowKeys = ['two'];
  const table = doc.workspace.surfaces.find((s) => s.view === 'table')!;
  if (table.view !== 'table' || plot.view !== 'scatter') throw new Error();
  table.table = { columns: ['x'], sort: null, viewRevision: 1 };
  // Archived draft fixture: no active chart command constructs new plot context.
  doc.chat.attachments = [
    {
      attachmentId: 'att_legacy',
      evidence: {
        schemaVersion: 2,
        projectId: 'prj_test',
        resource: { kind: 'file', resourceId: 'res_csv' },
        dataRevision: 'version-1',
        origin: { surfaceId: 'srf_plot' },
        selection: {
          kind: 'table',
          coordinateSpace: 'revision-row-column-keys',
          rowKeys: ['two'],
          columns: ['x', 'y', 'label'],
        },
      },
      presentation,
      label: 'test.csv · selected rows',
      createdAt: 2000,
    },
  ];
  const attachment = structuredClone(doc.chat.attachments![0]!);
  expect(attachment.evidence.selection).toMatchObject({
    rowKeys: ['two'],
    columns: ['x', 'y', 'label'],
  });
  plot.scatter.spec.xColumnId = 'y';
  doc.viewLinks[0]!.filter = { kind: 'equals', columnId: 'label', value: 'Absent' };
  expect(doc.chat.attachments![0]).toEqual(attachment);
  const asset = await materializeEvidence(attachment, data, noImage);
  expect(evidencePreview(asset)).toMatchObject({
    presentation,
    rows: [{ key: 'two', cells: ['3', '4', 'Same'] }],
    scope: { requestedRows: 1, returnedRows: 1, previewRows: 2 },
  });
  expect(evidenceInput([asset])[0]).toMatchObject({
    type: 'text',
    text: expect.stringContaining('"plotPixelsIncluded":false'),
  });
  const altered = structuredClone(attachment);
  altered.presentation!.viewport.x = [0, 100];
  const changed = await materializeEvidence(altered, data, noImage);
  expect(changed.manifest.asset.hash).not.toBe(asset.manifest.asset.hash);
  expect(() =>
    evidencePreview({
      ...asset,
      manifest: { ...asset.manifest, presentation: altered.presentation! },
    }),
  ).toThrow('does not match');
  const old: DraftAttachment = { ...attachment };
  delete old.presentation;
  const legacy = await materializeEvidence(old, data, noImage);
  expect(legacy.manifest.asset.hash).toBe(
    contentHash(
      Buffer.from(
        JSON.stringify({ kind: 'table', columns: content.columns, rows: [content.rows[1]] }),
      ),
    ),
  );
  const stale = structuredClone(data);
  if (stale.kind !== 'file') throw new Error();
  stale.value.revision = 'version-2';
  await expect(materializeEvidence(attachment, stale, noImage)).rejects.toMatchObject({
    code: 'stale_revision',
  });
});

describe('frozen historical document contracts', () => {
  it.each([
    [2, WorkspaceDocumentV2Schema],
    [3, WorkspaceDocumentV3Schema],
    [4, WorkspaceDocumentV4Schema],
    [5, WorkspaceDocumentV5Schema],
    [6, WorkspaceDocumentV6Schema],
    [7, WorkspaceDocumentV7Schema],
  ])('preserves published v%s exactly', async (version, schema) => {
    const published = JSON.parse(
      await readFile(new URL(`../../contracts/schema/v${version}.json`, import.meta.url), 'utf8'),
    );
    const normalized = (value: unknown) =>
      JSON.parse(
        JSON.stringify(value, (key, item) =>
          key === 'required' && Array.isArray(item) ? [...item].sort() : item,
        ),
      );
    expect(normalized(schema)).toEqual(normalized(published.definitions.WorkspaceDocumentSchema));
  });
  it('backs up exact v5 bytes once and restores the current workspace without adding plot context to old records', async () => {
    const base = await mkdtemp(join(tmpdir(), 'gobble-v5-'));
    try {
      const old = { ...linked(), schemaVersion: 5 },
        raw = JSON.stringify(old, null, 2) + '\n';
      await mkdir(join(base, 'projects'));
      const path = join(base, 'projects/prj_test.json');
      await writeFile(path, raw);
      const storage = new WorkspaceStorage(base).project('prj_test');
      const next = (await storage.read())!;
      expect(next).toEqual({ ...old, schemaVersion: 21 });
      expect(await readFile(path, 'utf8')).toBe(raw);
      await storage.write(next);
      expect(await readFile(path + '.v5.backup', 'utf8')).toBe(raw);
      next.workspace.revision++;
      await storage.write(next);
      expect(await readFile(path + '.v5.backup', 'utf8')).toBe(raw);
      expect(() => readStoredWorkspace({ ...old, schemaVersion: 999 })).toThrow();
    } finally {
      await rm(base, { recursive: true, force: true });
    }
  });
});
