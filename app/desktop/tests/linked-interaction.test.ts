import { addLegacyChart } from './fixtures/legacy-chart';
import { describe, expect, it } from 'vitest';
import { mkdtemp, mkdir, readFile, writeFile, rm } from 'node:fs/promises';
import { join } from 'node:path';
import { tmpdir } from 'node:os';
import { WorkspaceStorage } from '../src/main/workspace/storage';
import { WorkspaceController } from '../src/main/workspace/controller';
import { RenderSession } from '../src/main/workspace/render-session';
import {
  defaultScatterSpec,
  parse,
  parseWorkspaceDocument,
  readStoredWorkspace,
  sortTable,
  numericColumns,
  scatterViewport,
  validateViewport,
  type SurfaceData,
  type TableContent,
} from '@gobble/contracts';
import { WorkspaceDocumentV4Schema } from '../../contracts/src/workspace-document-v4';
import { emptyWorkspace, transition } from '../src/main/workspace/model';
import { applyTabularAction } from '../src/main/workspace/tabular-commands';

const content: TableContent = {
  kind: 'table',
  columns: [
    { id: 'sample', name: 'Sample' },
    { id: 'x', name: 'X' },
    { id: 'y', name: 'Y' },
  ],
  rows: [
    { key: 'one', cells: ['same', '0', '10'] },
    { key: 'two', cells: ['same', '20', '0'] },
    { key: 'three', cells: ['C', '2', 'NaN'] },
    { key: 'four', cells: ['D', '', '1'] },
  ],
  truncated: false,
};
const data: SurfaceData = {
  kind: 'file',
  value: {
    projectId: 'prj_test',
    resourceId: 'res_csv',
    revision: 'revision-1',
    name: 'data.csv',
    size: 100,
    content,
  },
};
function initial() {
  return transition(
    emptyWorkspace('prj_test'),
    {
      kind: 'open',
      resource: { kind: 'file', resourceId: 'res_csv' },
      pane: 'primary',
      duplicate: false,
    },
    { surfaceId: 'srf_table', view: 'table', title: 'data.csv' },
  );
}
const acknowledgment = {
  schemaVersion: 3 as const,
  projectId: 'prj_test',
  surfaceId: 'srf_table',
  rendererSessionId: 'rnd_test',
  requestId: 'req_load',
  generation: 1,
  dataRevision: 'revision-1',
  presentation: { spec: 0, view: 0, filter: 0 },
};
const spec = defaultScatterSpec(content)!;
function open(doc = initial()) {
  return parseWorkspaceDocument(addLegacyChart(doc, spec, 'revision-1'));
}

describe('linked User intent semantics', () => {
  it('rejects retired chart actions without mutating a table or its selection', () => {
    const doc = initial(),
      before = structuredClone(doc);
    for (const value of [
      { kind: 'openScatter', spec },
      { kind: 'openLinkedTable' },
      { kind: 'scatterSettings', spec },
      { kind: 'scatterViewport', viewport: null },
    ] as const)
      expect(() =>
        applyTabularAction(
          doc,
          { ...value, surfaceId: 'srf_table', acknowledgment },
          data,
          'req_removed',
        ),
      ).toThrow('removed');
    expect(doc).toEqual(before);
    applyTabularAction(
      doc,
      {
        kind: 'tableSettings',
        surfaceId: 'srf_table',
        acknowledgment,
        settings: {
          columns: ['sample', 'x'],
          sort: { columnId: 'x', direction: 'ascending', numeric: true },
        },
      },
      data,
      'req_table',
    );
    expect(doc.workspace.surfaces[0]).toMatchObject({
      view: 'table',
      table: { columns: ['sample', 'x'] },
    });
  });
  it('renews the whole source without relocating old members or draft targets', () => {
    const doc = open();
    doc.viewLinks[0]!.rowKeys = ['one'];
    applyTabularAction(
      doc,
      { kind: 'discussSelection', surfaceId: 'srf_table', acknowledgment },
      data,
      'req_discuss',
    );
    const next = structuredClone(data);
    if (next.kind !== 'file') throw new Error();
    next.value.revision = 'revision-2';
    applyTabularAction(
      doc,
      { kind: 'refreshLinked', surfaceId: 'srf_table', dataRevision: 'revision-1' },
      next,
      'req_refresh',
    );
    expect(doc.viewLinks[0]).toMatchObject({
      dataRevision: 'revision-2',
      rowKeys: [],
      filter: { kind: 'all' },
    });
    expect(doc.chat.attachments![0]!.evidence.dataRevision).toBe('revision-1');
    expect(() =>
      applyTabularAction(
        doc,
        { kind: 'refreshLinked', surfaceId: 'srf_table', dataRevision: 'revision-1' },
        next,
        'req_old',
      ),
    ).toThrow('already been refreshed');
  });
  it('preserves v4 schema identity and migrates without inventing table options or selection origins', async () => {
    const published = JSON.parse(
      await readFile(new URL('../../contracts/schema/v4.json', import.meta.url), 'utf8'),
    );
    expect(JSON.parse(JSON.stringify(WorkspaceDocumentV4Schema))).toEqual(
      published.definitions.WorkspaceDocumentSchema,
    );
    const old = { ...open(), schemaVersion: 4 };
    delete old.viewLinks[0]!.selectionSourceId;
    parse(WorkspaceDocumentV4Schema, old);
    const migrated = readStoredWorkspace(old);
    expect(migrated).toEqual({ ...old, schemaVersion: 21 });
    expect(old.schemaVersion).toBe(4);
    expect(() => readStoredWorkspace({ ...old, schemaVersion: 999 })).toThrow();
  });
});

it('sorts decimal values stably, retains invalids last, and never changes the source order or identity', () => {
  const original = structuredClone(content);
  expect(numericColumns(content).map((c) => c.id)).toEqual(['x', 'y']);
  expect(
    sortTable(content, content.rows, { columnId: 'x', numeric: true, direction: 'ascending' }).map(
      (r) => r.key,
    ),
  ).toEqual(['one', 'three', 'two', 'four']);
  expect(
    sortTable(content, content.rows, { columnId: 'x', numeric: true, direction: 'descending' }).map(
      (r) => r.key,
    ),
  ).toEqual(['two', 'three', 'one', 'four']);
  expect(
    sortTable(content, content.rows, { columnId: 'sample', numeric: false, direction: 'ascending' })
      .filter((r) => r.cells[0] === 'same')
      .map((r) => r.key),
  ).toEqual(['one', 'two']);
  expect(content).toEqual(original);
});

it('backs up the exact v4 archive only on mutation and keeps it immutable after later saves', async () => {
  const base = await mkdtemp(join(tmpdir(), 'gobble-v4-'));
  try {
    const old = { ...open(), schemaVersion: 4 };
    delete old.viewLinks[0]!.selectionSourceId;
    const raw = JSON.stringify(old, null, 2) + '\n';
    await mkdir(join(base, 'projects'));
    const path = join(base, 'projects/prj_test.json');
    await writeFile(path, raw);
    const file = new WorkspaceStorage(base).project('prj_test');
    const migrated = (await file.read())!;
    expect(await readFile(path, 'utf8')).toBe(raw);
    migrated.workspace.revision++;
    await file.write(migrated);
    expect(await readFile(path + '.v4.backup', 'utf8')).toBe(raw);
    migrated.workspace.revision++;
    await file.write(migrated);
    expect(await readFile(path + '.v4.backup', 'utf8')).toBe(raw);
    expect(JSON.parse(await readFile(path, 'utf8')).schemaVersion).toBe(21);
  } finally {
    await rm(base, { recursive: true, force: true });
  }
});

it('retires old table settings receipts and keeps empty/constant/extreme plot bounds finite', () => {
  const doc = open();
  const table = doc.workspace.surfaces.find((s) => s.view === 'table')!;
  const render = new RenderSession();
  render.reveal([table.surfaceId]);
  const ticket = render.begin(
    { projectId: 'prj_test', surfaceId: table.surfaceId, rendererSessionId: render.id },
    table,
    doc.viewLinks[0],
  );
  const load = render.complete(ticket, data);
  render.release(ticket);
  render.acknowledge(load.acknowledgment);
  if (table.view !== 'table') throw new Error();
  table.table = { sort: null, columns: ['x'], viewRevision: 1 };
  render.reconcile(doc.workspace.surfaces, doc.viewLinks);
  expect(() => render.assertAcknowledgment(load.acknowledgment)).toThrow();
  for (const points of [
    [],
    [{ rowKey: 'one', x: 0, y: 0, label: 'same' }],
    [{ rowKey: 'one', x: Number.MAX_VALUE, y: -Number.MAX_VALUE, label: 'edge' }],
  ]) {
    expect(() => validateViewport(scatterViewport(points))).not.toThrow();
  }
});

it('the Main host rejects retired requests before source I/O or workspace mutation', async () => {
  const base = await mkdtemp(join(tmpdir(), 'gobble-retirement-'));
  try {
    let reads = 0;
    const controller = new WorkspaceController(new WorkspaceStorage(base), {
      projects: async () => [{ projectId: 'prj_test', name: 'Test', rootResourceId: 'res_root' }],
      describe: async () => ({ title: 'data.csv', view: 'table' }),
      read: async () => {
        reads++;
        return structuredClone(data);
      },
    });
    await controller.initialize();
    await controller.connect();
    await controller.openProject('prj_test');
    const before = await controller.read('prj_test');
    await expect(
      controller.command({
        projectId: 'prj_test',
        expectedRevision: before.workspace.revision,
        requestId: 'req_removed',
        action: { kind: 'openScatter', surfaceId: 'srf_table', acknowledgment, spec },
      }),
    ).rejects.toMatchObject({ code: 'unsupported' });
    expect(reads).toBe(0);
    expect(await controller.read('prj_test')).toEqual(before);
  } finally {
    await rm(base, { recursive: true, force: true });
  }
});
