import { afterEach, expect, it, vi } from 'vitest';
import { mkdtemp, mkdir, readFile, rm, writeFile } from 'node:fs/promises';
import { join } from 'node:path';
import { tmpdir } from 'node:os';
import {
  containsObservedTarget,
  defaultDependencyNavigation,
  observedReferenceSurface,
  parse,
  projectDependencies,
  readStoredWorkspace,
  SharedToolRequestV6Schema,
  toolSchemas,
  WorkspaceDocumentV9Schema,
  type SharedReference,
  type Surface,
  type SurfaceData,
} from '@gobble/contracts';
import { dependencySnapshot } from './fixtures/dependency-snapshot';
import { presentDependencies } from '../src/main/service/dependency-presentation';
import { presentRun } from '../src/main/service/run-presentation';
import { dependencyObservation } from '../src/main/shared-context/dependency-observation';
import { ObservedReads } from '../src/main/shared-context/observed';
import { RenderSession } from '../src/main/workspace/render-session';
import { emptyWorkspace, transition } from '../src/main/workspace/model';
import {
  ObservedReferenceViews,
  prepareObservedReference,
} from '../src/main/workspace/observed-reference-views';
import { findCurrentDependency } from '../src/main/workspace/current-dependency';
import { WorkspaceStorage } from '../src/main/workspace/storage';

const paths: string[] = [];
afterEach(async () => {
  for (const path of paths.splice(0)) await rm(path, { recursive: true, force: true });
});
function fixture() {
  const raw = dependencySnapshot();
  const data: Extract<SurfaceData, { kind: 'run' }> = {
    kind: 'run',
    value: presentRun(raw),
    dependencies: projectDependencies(presentDependencies(raw)),
  };
  const surface: Surface = {
    projectId: raw.projectId,
    surfaceId: 'srf_run',
    resource: { kind: 'run', runRef: raw.runRef },
    view: 'run',
    pinned: false,
    openedBy: { kind: 'user' },
    runView: {
      query: 'S03',
      status: 'failed',
      mode: 'dependencies',
      viewRevision: 2,
      dependencies: defaultDependencyNavigation(),
    },
  };
  const render = new RenderSession();
  render.reveal([surface.surfaceId]);
  const load = render.complete(
    render.begin(
      { projectId: raw.projectId, surfaceId: surface.surfaceId, rendererSessionId: render.id },
      surface,
    ),
    data,
  );
  render.acknowledge(load.acknowledgment);
  return { surface, data, acknowledgment: load.acknowledgment };
}
const group = (taskId = 'align') => ({
  kind: 'run-group' as const,
  coordinateSpace: 'observed-authored-task-group' as const,
  taskId,
});
const edge = () => ({
  kind: 'run-dependency' as const,
  coordinateSpace: 'observed-authored-task-pair' as const,
  fromTaskId: 'prepare',
  toTaskId: 'align',
});
function pointer(view = fixture()): SharedReference {
  const read = new ObservedReads().observeDependencies(view, edge(), 'view');
  return {
    referenceId: 'ref_agent',
    label: 'Pilot',
    evidence: read.receipt.evidence!,
    author: { kind: 'agent', agentId: 'agt_reviewer', name: 'Reviewer' },
    createdAt: 1001,
    note: 'Check this directed pair.',
    retracted: false,
  };
}
it('returns exact authored groups and directed pairs without inventing a whole-graph address', () => {
  const view = fixture(),
    reads = new ObservedReads();
  const result = reads.observeDependencies(view, undefined, 'view');
  expect(result.receipt).toMatchObject({ pointable: true, scope: 'view' });
  expect(result.receipt).not.toHaveProperty('evidence');
  expect(result.content).toMatchObject({ returnedGroups: 3, returnedEdges: 2, truncated: false });
  expect(result.content.groups.find((item) => item.taskId === 'align')).toMatchObject({
    taskId: 'align',
    observedMembers: 3,
  });
  expect(JSON.stringify(result)).not.toMatch(/NEVER INCLUDE LOGS|private\/source|runtime.sock/);
  for (const item of [...result.content.groups, ...result.content.edges])
    expect(() =>
      reads.assertPoint(result.receipt.observedReadId, item.target, result.receipt.observation),
    ).not.toThrow();
  expect(
    parse(toolSchemas.workspace_point, {
      evidence: result.content.edges[0]!.target,
      observedReadId: result.receipt.observedReadId,
      observation: result.receipt.observation,
      note: '',
    }),
  ).toBeDefined();
});
it('refuses reversed pairs, invented groups, foreign receipts, and a previous turn', () => {
  const view = fixture(),
    reads = new ObservedReads();
  const read = reads.observeDependencies(view, edge(), 'view'),
    target = read.receipt.evidence!;
  for (const changed of [
    { ...target, selection: { ...edge(), fromTaskId: 'align', toTaskId: 'prepare' } },
    { ...target, selection: group('invented') },
    { ...target, projectId: 'prj_other' },
    { ...target, dataRevision: 'changed' },
  ])
    expect(() =>
      reads.assertPoint(read.receipt.observedReadId, changed, read.receipt.observation),
    ).toThrow();
  expect(() =>
    reads.assertPoint(read.receipt.observedReadId, target, {
      ...read.receipt.observation,
      generation: 50,
    }),
  ).toThrow();
  expect(() =>
    new ObservedReads().assertPoint(read.receipt.observedReadId, target, read.receipt.observation),
  ).toThrow();
  expect(containsObservedTarget({ ...target, selection: group('prepare') }, target)).toBe(false);
});
it('distinguishes graph find, filtered list, and non-pointable source previews', () => {
  const view = fixture(),
    reads = new ObservedReads();
  view.surface.runView!.dependencies!.query = 'report';
  expect(reads.observeDependencies(view, undefined, 'view').content.groups).toHaveLength(3);
  view.surface.runView!.dependencies!.representation = 'list';
  expect(
    reads.observeDependencies(view, undefined, 'view').content.groups.map((item) => item.taskId),
  ).toEqual(['report']);
  expect(() => reads.observeDependencies(view, group(), 'view')).toThrow('outside');
  const source = reads.observeDependencies(view, group(), 'source-preview');
  expect(source.receipt.pointable).toBe(false);
  expect(() =>
    reads.assertPoint(
      source.receipt.observedReadId,
      source.receipt.evidence!,
      source.receipt.observation,
    ),
  ).toThrow();
  view.surface.runView!.mode = 'tasks';
  expect(() => reads.observeDependencies(view, undefined, 'view')).toThrow('not displayed');
  expect(reads.observeDependencies(view, undefined, 'source-preview').receipt.pointable).toBe(
    false,
  );
});
it('bounds context at whole records and never authorizes omitted groups', () => {
  const view = fixture(),
    value = view.data.dependencies!;
  const base = value.groups[1]!;
  value.groups = Array.from({ length: 80 }, (_, index) => ({
    ...base,
    taskId: 'group-' + index,
    counts: { ...base.counts, states: [{ status: 'S'.repeat(1600), count: 3 }] },
    members: Array.from({ length: 50 }, (_, i) => ({
      ...base.members[0]!,
      instanceId: 'sample-' + i,
    })),
  }));
  value.edges = [];
  const read = new ObservedReads().observeDependencies(view, undefined, 'view');
  expect(Buffer.byteLength(JSON.stringify(read))).toBeLessThan(64 * 1024);
  expect(read.content.returnedGroups).toBeLessThan(80);
  expect(read.content.truncated).toBe(true);
  expect(read.content.groups.every((item) => item.observedMembers === 50 && item.truncated)).toBe(
    true,
  );
  const observed = dependencyObservation(value, view.surface, 'view', group('group-79'));
  expect(observed.evidence!.selection).toEqual(group('group-79'));
  const reads = new ObservedReads(),
    all = reads.observeDependencies(view, undefined, 'view');
  expect(() =>
    reads.assertPoint(all.receipt.observedReadId, observed.evidence!, all.receipt.observation),
  ).toThrow();
});
it('keeps Show transient, protects base commands, and returns mode, camera, selection and draft', () => {
  const view = fixture(),
    reference = pointer(view);
  let document = transition(
    emptyWorkspace(view.surface.projectId),
    { kind: 'open', resource: view.surface.resource, pane: 'primary', duplicate: false },
    { surfaceId: view.surface.surfaceId, view: 'run', title: 'Pilot' },
  );
  document.workspace.surfaces = [structuredClone(view.surface)];
  const base = document.workspace.surfaces[0]! as Extract<Surface, { view: 'run' }>;
  base.runView!.mode = 'tasks';
  base.runView!.dependencies!.camera = { zoom: 1.5, x: 130, y: 20 };
  document.chat.draft = 'Keep my draft';
  if (reference.evidence.schemaVersion !== 4) throw new Error('Dependency fixture required');
  document.selections = [
    {
      surfaceId: base.surfaceId,
      evidence: { ...reference.evidence, schemaVersion: 4, selection: group('report') },
    },
  ];
  const before = structuredClone(document);
  const session = prepareObservedReference(document, reference, view.data, 'req_show');
  expect(document).toEqual(before);
  expect(observedReferenceSurface(base, session.state)).toMatchObject({
    runView: { mode: 'dependencies', dependencies: { query: '', camera: { zoom: 1, x: 0, y: 0 } } },
  });
  const views = new ObservedReferenceViews();
  views.set(session);
  for (const action of [
    { kind: 'runMode' as const, surfaceId: base.surfaceId, mode: 'dependencies' as const },
    {
      kind: 'dependencyCamera' as const,
      surfaceId: base.surfaceId,
      camera: { zoom: 2, x: 0, y: 0 },
    },
    {
      kind: 'dependencyNavigation' as const,
      surfaceId: base.surfaceId,
      navigation: { representation: 'list' as const, query: '' },
    },
  ])
    expect(() => views.assertBaseAction(action)).toThrow('Return');
  views.returnTo(document, 'req_show');
  expect(document).toEqual(before);
  const changed = structuredClone(view.data);
  changed.dependencies!.groups[1]!.members[0]!.attempt++;
  expect(() => prepareObservedReference(document, reference, changed, 'req_stale')).toThrow(
    'older observation',
  );
  expect(document).toEqual(before);
});
it('searches current source explicitly without replacing historical target or draft', async () => {
  const view = fixture(),
    reference = pointer(view),
    doc = emptyWorkspace(view.surface.projectId);
  reference.author = { kind: 'user' };
  doc.workspace.agents = [
    {
      projectId: view.surface.projectId,
      agentId: 'agt_reviewer',
      name: 'Reviewer',
      instructionProfile: 'discussion-v1',
      provider: { kind: 'codex', threadId: null },
    },
  ];
  doc.sharedReferences = [reference];
  doc.chat.draft = 'Unsent';
  const describe = vi.fn(async () => ({ view: 'run' as const, title: 'Current run' }));
  const next = await findCurrentDependency(doc, reference, { describe });
  expect(next.document.workspace.surfaces[0]).toMatchObject({
    runView: { mode: 'dependencies', dependencies: { representation: 'list', query: 'prepare' } },
  });
  expect(next.document.selections).toEqual([]);
  expect(next.document.sharedReferences).toEqual([reference]);
  expect(next.document.chat.draft).toBe('Unsent');
  expect(doc.workspace.surfaces).toEqual([]);
});
it('freezes v9/v6 readers and backs up exact bytes before v10 writes', async () => {
  const published = JSON.parse(
    await readFile(new URL('../../contracts/schema/v11.json', import.meta.url), 'utf8'),
  );
  expect(JSON.parse(JSON.stringify(WorkspaceDocumentV9Schema))).toEqual(
    published.definitions.WorkspaceDocumentSchema,
  );
  expect(JSON.parse(JSON.stringify(SharedToolRequestV6Schema))).toEqual(
    published.definitions.SharedToolRequestSchema,
  );
  const reference = pointer();
  reference.author = { kind: 'user' };
  const old = { ...emptyWorkspace(reference.evidence.projectId), schemaVersion: 9 };
  expect(() => readStoredWorkspace({ ...old, sharedReferences: [reference] })).toThrow();
  const directory = await mkdtemp(join(tmpdir(), 'gobble-dependency-v10-'));
  paths.push(directory);
  await mkdir(join(directory, 'projects'));
  const path = join(directory, 'projects', old.workspace.projectId + '.json'),
    bytes = JSON.stringify(old, null, 2) + '\n';
  await writeFile(path, bytes);
  const store = new WorkspaceStorage(directory).project(old.workspace.projectId),
    current = (await store.read())!;
  expect(current.schemaVersion).toBe(21);
  current.workspace.agents = [
    {
      projectId: reference.evidence.projectId,
      agentId: 'agt_reviewer',
      name: 'Reviewer',
      instructionProfile: 'discussion-v1',
      provider: { kind: 'codex', threadId: null },
    },
  ];
  await store.write({ ...current, sharedReferences: [reference] });
  expect(await readFile(path + '.v9.backup', 'utf8')).toBe(bytes);
  expect((await store.read())!.sharedReferences).toEqual([reference]);
});

it('resolves selector fields by value and does not authorize a hidden endpoint group from a visible pair', () => {
  const view = fixture(),
    reads = new ObservedReads();
  view.surface.runView!.dependencies = {
    ...defaultDependencyNavigation(),
    representation: 'list',
    query: 'align',
  };
  const selected = {
    toTaskId: 'align',
    fromTaskId: 'prepare',
    coordinateSpace: 'observed-authored-task-pair' as const,
    kind: 'run-dependency' as const,
  };
  const read = reads.observeDependencies(view, selected, 'view');
  expect(() =>
    reads.assertPoint(
      read.receipt.observedReadId,
      read.receipt.evidence!,
      read.receipt.observation,
    ),
  ).not.toThrow();
  const endpoint = read.content.groups.find((item) => item.taskId === 'prepare')!;
  expect(endpoint.members).toHaveLength(1);
  expect(() =>
    reads.assertPoint(read.receipt.observedReadId, endpoint.target, read.receipt.observation),
  ).toThrow();
});
