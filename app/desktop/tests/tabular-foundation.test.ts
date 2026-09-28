import { describe, expect, it } from 'vitest';
import {
  captureScatterPresentation,
  type SharedReference,
  parseWorkspaceDocument,
  readStoredWorkspace,
  samePresentation,
  type Surface,
  type SurfaceData,
  type ViewLink,
} from '@gobble/contracts';
import { emptyWorkspace, transition } from '../src/main/workspace/model';
import { prepareReferenceView } from '../src/main/workspace/reference-views';
import { RenderSession } from '../src/main/workspace/render-session';
import { TabularSnapshots } from '../src/main/workspace/tabular-snapshots';

const resource = { kind: 'file' as const, resourceId: 'res_samples' };
const scatter = (): Extract<Surface, { view: 'scatter' }> => ({
  projectId: 'prj_atlas',
  surfaceId: 'srf_plot',
  resource,
  view: 'scatter',
  pinned: false,
  openedBy: { kind: 'user' },
  scatter: {
    spec: { xColumnId: 'x', yColumnId: 'y', xScale: 'linear', yScale: 'linear' },
    specRevision: 1,
    viewRevision: 0,
    viewport: null,
  },
});
const link = (): ViewLink => ({
  linkId: 'lnk_samples',
  resource,
  dataRevision: 'revision-1',
  surfaceIds: ['srf_plot', 'srf_table'],
  filterRevision: 0,
  filter: { kind: 'all' },
  rowKeys: [],
});
const data = (): SurfaceData => ({
  kind: 'file',
  value: {
    projectId: 'prj_atlas',
    resourceId: 'res_samples',
    name: 'samples.csv',
    revision: 'revision-1',
    size: 20,
    content: {
      kind: 'table',
      columns: [
        { id: 'x', name: 'X' },
        { id: 'y', name: 'Y' },
      ],
      rows: [{ key: 'row_1', cells: ['0', '1'] }],
      truncated: false,
    },
  },
});
function document() {
  const doc = emptyWorkspace('prj_atlas');
  doc.workspace.surfaces = [
    scatter(),
    {
      projectId: 'prj_atlas',
      surfaceId: 'srf_table',
      resource,
      view: 'table',
      pinned: false,
      openedBy: { kind: 'user' },
    },
  ];
  doc.workspace.layout = {
    kind: 'split',
    primary: { tabs: ['srf_plot'], activeSurfaceId: 'srf_plot' },
    secondary: { tabs: ['srf_table'], activeSurfaceId: 'srf_table' },
    primaryFraction: 0.5,
  };
  doc.titles = doc.workspace.surfaces.map((s) => ({ surfaceId: s.surfaceId, title: s.view }));
  doc.viewLinks = [link()];
  return parseWorkspaceDocument(doc);
}

describe('single linked membership owner', () => {
  it('rejects duplicated ownership, mismatched resources and missing surfaces', () => {
    for (const mutate of [
      (doc: ReturnType<typeof document>) => doc.viewLinks.push({ ...link(), linkId: 'lnk_other' }),
      (doc: ReturnType<typeof document>) => {
        doc.viewLinks[0]!.surfaceIds.push('srf_missing');
      },
      (doc: ReturnType<typeof document>) => {
        doc.workspace.surfaces[0]!.resource = { kind: 'file', resourceId: 'res_foreign' };
      },
      (doc: ReturnType<typeof document>) => {
        doc.selections.push({
          surfaceId: 'srf_table',
          evidence: {
            schemaVersion: 2,
            projectId: 'prj_atlas',
            resource,
            dataRevision: 'revision-1',
          },
        });
      },
    ]) {
      const doc = document();
      mutate(doc);
      expect(() => parseWorkspaceDocument(doc)).toThrow();
    }
  });
  it('commits membership once and keeps it until the last linked view closes', () => {
    const doc = document();
    const selected = transition(doc, {
      kind: 'select',
      surfaceId: 'srf_plot',
      evidence: {
        schemaVersion: 2,
        projectId: 'prj_atlas',
        resource,
        dataRevision: 'revision-1',
        selection: {
          kind: 'table',
          coordinateSpace: 'revision-row-column-keys',
          rowKeys: ['row_1'],
          columns: ['x', 'y'],
        },
      },
    });
    expect(selected.selections).toEqual([]);
    expect(selected.viewLinks[0]!.rowKeys).toEqual(['row_1']);
    expect(doc.viewLinks[0]!.rowKeys).toEqual([]);
    const one = transition(selected, { kind: 'close', surfaceId: 'srf_plot' });
    expect(one.viewLinks[0]!.rowKeys).toEqual(['row_1']);
    expect(transition(one, { kind: 'close', surfaceId: 'srf_table' }).viewLinks).toEqual([]);
  });
  it('migrates v3 without inferring links or accepting a future version', () => {
    const doc = emptyWorkspace('prj_atlas');
    const { viewLinks: _, ...old } = doc;
    const original = { ...old, schemaVersion: 3 };
    expect(readStoredWorkspace(original)).toEqual(doc);
    expect(original.schemaVersion).toBe(3);
    expect(() => readStoredWorkspace({ ...original, schemaVersion: 7 })).toThrow();
  });
});

describe('render identity and snapshot coherence', () => {
  it('invalidates a failed render permanently without retiring another view', () => {
    const session = new RenderSession();
    session.reveal(['srf_plot']);
    const request = {
      projectId: 'prj_atlas',
      surfaceId: 'srf_plot',
      rendererSessionId: session.id,
    };
    const ticket = session.begin(request, { ...scatter(), view: 'table' }, link());
    const load = session.complete(ticket, data());
    session.release(ticket);
    session.acknowledge(load.acknowledgment);
    expect(() => session.invalidate({ ...load.acknowledgment, surfaceId: 'srf_other' })).toThrow();
    expect(session.presentation('srf_plot')).toBe('ready');
    session.invalidate(load.acknowledgment);
    expect(session.presentation('srf_plot')).toBe('loading');
    expect(() => session.snapshot('srf_plot')).toThrow();
    expect(() => session.acknowledge(load.acknowledgment)).toThrow();
    const retry = session.begin(request, { ...scatter(), view: 'table' }, link());
    const next = session.complete(retry, data());
    session.release(retry);
    expect(session.acknowledge(next.acknowledgment)).toEqual({ accepted: true });
  });
  it('rejects old axes/filter acknowledgments, cross-view replay and stale completion', () => {
    const session = new RenderSession(),
      surface = {
        ...scatter(),
        view: 'table' as const,
        table: { sort: null, columns: null, viewRevision: 0 },
      },
      linked = link();
    session.reveal(['srf_plot']);
    const request = {
      projectId: 'prj_atlas',
      surfaceId: 'srf_plot',
      rendererSessionId: session.id,
    };
    const ticket = session.begin(request, surface, linked);
    const first = session.complete(ticket, data());
    session.release(ticket);
    session.acknowledge(first.acknowledgment);
    expect(() =>
      session.assertAcknowledgment({ ...first.acknowledgment, surfaceId: 'srf_table' }),
    ).toThrow();
    surface.table.viewRevision++;
    session.reconcile([surface], [linked]);
    expect(() => session.acknowledge(first.acknowledgment)).toThrow();
    const pending = session.begin(request, surface, linked);
    linked.filterRevision++;
    session.reconcile([surface], [linked]);
    expect(() => session.complete(pending, data())).toThrow();
    session.release(pending);
    expect(
      samePresentation(first.acknowledgment.presentation, { spec: 2, view: 0, filter: 0 }),
    ).toBe(false);
  });
  it('refuses a retired chart before creating a render acknowledgment', () => {
    const session = new RenderSession();
    session.reveal(['srf_plot']);
    expect(() =>
      session.begin(
        { projectId: 'prj_atlas', surfaceId: 'srf_plot', rendererSessionId: session.id },
        scatter(),
        link(),
      ),
    ).toThrow('retired');
    expect(() => session.snapshot('srf_plot')).toThrow();
  });
  it('shares one source load across linked views and discards rejected or released leases', async () => {
    const snapshots = new TabularSnapshots();
    let calls = 0;
    const load = async () => {
      calls++;
      return data();
    };
    const [a, b] = await Promise.all([
      snapshots.read('prj_atlas', link(), load),
      snapshots.read('prj_atlas', link(), load),
    ]);
    expect(calls).toBe(1);
    expect(a).toEqual(b);
    expect(a).not.toBe(b);
    snapshots.retain('prj_atlas', []);
    await snapshots.read('prj_atlas', link(), load);
    expect(calls).toBe(2);
    snapshots.clear();
    const changed = { ...link(), dataRevision: 'revision-2' };
    await expect(snapshots.read('prj_atlas', changed, load)).rejects.toMatchObject({
      code: 'stale_revision',
    });
    await expect(snapshots.read('prj_atlas', changed, load)).rejects.toMatchObject({
      code: 'stale_revision',
    });
    expect(calls).toBe(4);
  });
  it('rejects a missing filter column before a linked table can be acknowledged', () => {
    const session = new RenderSession();
    session.reveal(['srf_table']);
    const ticket = session.begin(
      { projectId: 'prj_atlas', surfaceId: 'srf_table', rendererSessionId: session.id },
      document().workspace.surfaces[1]!,
      { ...link(), filter: { kind: 'equals', columnId: 'missing', value: '1' } },
    );
    expect(() => session.complete(ticket, data())).toThrow('column');
    expect(() => session.snapshot('srf_table')).toThrow();
    session.release(ticket);
  });
});

it('reveals an archived chart pointer in a table without changing its captured target or recreating a chart', () => {
  const source = data();
  if (source.kind !== 'file' || source.value.content.kind !== 'table') throw new Error();
  const reference: SharedReference = {
    referenceId: 'ref_legacy',
    label: 'Saved rows',
    note: '',
    author: { kind: 'user' },
    createdAt: 2000,
    retracted: false,
    evidence: {
      schemaVersion: 2,
      projectId: 'prj_atlas',
      resource,
      dataRevision: 'revision-1',
      selection: {
        kind: 'table',
        coordinateSpace: 'revision-row-column-keys',
        rowKeys: ['row_1'],
        columns: ['x', 'y'],
      },
    },
    presentation: captureScatterPresentation(scatter(), link(), source.value.content)!,
  };
  const archived = structuredClone(reference);
  const existing = document();
  const originalSurfaces = structuredClone(existing.workspace.surfaces);
  const revealed = prepareReferenceView(existing, reference, source, 'req_reveal');
  expect(revealed.surfaceIds).toEqual(['srf_table']);
  expect(existing.workspace.surfaces).toEqual(originalSurfaces);
  expect(reference).toEqual(archived);
  const empty = emptyWorkspace('prj_atlas');
  const opened = prepareReferenceView(empty, reference, source, 'req_new');
  expect(empty.workspace.surfaces).toHaveLength(1);
  expect(empty.workspace.surfaces[0]).toMatchObject({
    view: 'table',
    surfaceId: opened.surfaceIds[0],
  });
  expect(empty.viewLinks).toEqual([]);
  const stale = structuredClone(source);
  stale.value.revision = 'revision-2';
  const before = structuredClone(empty);
  expect(() => prepareReferenceView(empty, reference, stale, 'req_stale')).toThrow('older source');
  expect(empty).toEqual(before);
  expect(reference).toEqual(archived);
});
