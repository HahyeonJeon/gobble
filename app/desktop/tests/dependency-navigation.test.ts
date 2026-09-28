import { afterEach, expect, it, vi } from 'vitest';
import { mkdtemp, mkdir, readFile, writeFile, rm } from 'node:fs/promises';
import { join } from 'node:path';
import { tmpdir } from 'node:os';
import {
  parse,
  WorkspaceDocumentV8Schema,
  projectDependencies,
  readStoredWorkspace,
  observedTargetVisible,
  type SurfaceData,
  type DependencyTarget,
  type WorkspaceAction,
} from '@gobble/contracts';
import { toolSchemasV6 as toolSchemas } from '../../contracts/src/shared-tools-v6';
import { dependencySnapshot } from './fixtures/dependency-snapshot';
import { presentDependencies } from '../src/main/service/dependency-presentation';
import { presentRun } from '../src/main/service/run-presentation';
import { dependencyRevision } from '../src/main/evidence/dependency-capture';
import { EvidenceCapture, materializeObserved } from '../src/main/evidence/capture';
import { evidencePreview, evidenceInput } from '../src/main/evidence/materialize';
import { EvidenceStorage } from '../src/main/evidence/storage';
import { WorkspaceStorage } from '../src/main/workspace/storage';
import { WorkspaceController } from '../src/main/workspace/controller';
import { RenderSession } from '../src/main/workspace/render-session';
import { emptyWorkspace, transition } from '../src/main/workspace/model';
import { observedDataRevision } from '../src/main/workspace/selection';
import { ObservedReads } from '../src/main/shared-context/observed';

const paths: string[] = [];
afterEach(async () => {
  for (const p of paths.splice(0)) await rm(p, { recursive: true, force: true });
});
async function directory() {
  const p = await mkdtemp(join(tmpdir(), 'gobble-dependency-'));
  paths.push(p);
  return p;
}
function source(): Extract<SurfaceData, { kind: 'run' }> {
  const raw = dependencySnapshot();
  return {
    kind: 'run',
    value: presentRun(raw),
    dependencies: projectDependencies(presentDependencies(raw)),
  };
}
function target(data = source()): DependencyTarget {
  return {
    schemaVersion: 4,
    projectId: data.value.projectId,
    resource: { kind: 'run', runRef: data.value.runRef },
    dataRevision: dependencyRevision(data.dependencies!),
    selection: {
      kind: 'run-dependency',
      coordinateSpace: 'observed-authored-task-pair',
      fromTaskId: 'prepare',
      toTaskId: 'align',
    },
  };
}
function opened() {
  const data = source();
  return transition(
    emptyWorkspace(data.value.projectId),
    { kind: 'open', resource: target(data).resource, pane: 'primary', duplicate: false },
    { surfaceId: 'srf_run', title: 'Run', view: 'run' },
  );
}

it('freezes v8 nested readers and backs up exact prior bytes before storing v10', async () => {
  const published = JSON.parse(
    await readFile(new URL('../../contracts/schema/v10.json', import.meta.url), 'utf8'),
  );
  expect(JSON.parse(JSON.stringify(WorkspaceDocumentV8Schema))).toEqual(
    published.definitions.WorkspaceDocumentSchema,
  );
  const old = { ...opened(), schemaVersion: 8 },
    p = await directory(),
    id = old.workspace.projectId;
  await mkdir(join(p, 'projects'));
  const file = join(p, 'projects', id + '.json'),
    bytes = JSON.stringify(old, null, 2) + '\n';
  await writeFile(file, bytes);
  const store = new WorkspaceStorage(p).project(id),
    current = (await store.read())!;
  expect(current).toEqual({ ...old, schemaVersion: 21 });
  expect(await readFile(file, 'utf8')).toBe(bytes);
  await store.write(current);
  expect(await readFile(file + '.v8.backup', 'utf8')).toBe(bytes);
  expect(() =>
    readStoredWorkspace({ ...old, selections: [{ surfaceId: 'srf_run', evidence: target() }] }),
  ).toThrow();
});
it('keeps one semantic selection through mode and camera changes while preserving task filters', () => {
  let doc = opened();
  doc = transition(doc, {
    kind: 'runFilter',
    surfaceId: 'srf_run',
    filter: { query: 'S03', status: 'failed' },
  });
  doc = transition(doc, { kind: 'runMode', surfaceId: 'srf_run', mode: 'dependencies' });
  doc = transition(doc, { kind: 'select', surfaceId: 'srf_run', evidence: target() });
  const state = doc.workspace.surfaces[0]!;
  if (state.view !== 'run') throw new Error();
  const revision = state.runView!.viewRevision;
  const camera = { zoom: 1.5, x: 180, y: 90 };
  doc = transition(doc, { kind: 'dependencyCamera', surfaceId: 'srf_run', camera });
  expect(doc.workspace.surfaces[0]).toMatchObject({
    runView: {
      query: 'S03',
      status: 'failed',
      mode: 'dependencies',
      viewRevision: revision,
      dependencies: { camera },
    },
  });
  // Independent updates commute: delayed camera persistence cannot overwrite search or list mode.
  doc = transition(doc, {
    kind: 'dependencyNavigation',
    surfaceId: 'srf_run',
    navigation: { query: 'align', representation: 'list' },
  });
  doc = transition(doc, {
    kind: 'dependencyCamera',
    surfaceId: 'srf_run',
    camera: { ...camera, x: 240 },
  });
  expect(doc.workspace.surfaces[0]).toMatchObject({
    runView: {
      dependencies: { query: 'align', representation: 'list', camera: { ...camera, x: 240 } },
    },
  });
  doc = transition(doc, {
    kind: 'dependencyNavigation',
    surfaceId: 'srf_run',
    navigation: { query: 'prepare', representation: 'graph' },
  });
  expect(doc.workspace.surfaces[0]).toMatchObject({
    runView: { dependencies: { query: 'prepare', camera: { ...camera, x: 240 } } },
  });
  doc = transition(doc, { kind: 'runMode', surfaceId: 'srf_run', mode: 'tasks' });
  expect(doc.selections).toEqual([{ surfaceId: 'srf_run', evidence: target() }]);
  expect(observedTargetVisible(doc.workspace.surfaces[0]!, source(), target())).toBe(false);
  const data = source();
  expect(observedDataRevision(data)).toBe(observedDataRevision({ kind: 'run', value: data.value }));
});
it('publishes exact dependency capture through existing evidence bytes and rejects manifest substitution', () => {
  const data = source(),
    ref = target(data),
    attachment = { attachmentId: 'att_dependency', label: 'Run', createdAt: 2000, evidence: ref };
  const asset = materializeObserved(attachment, data),
    preview = evidencePreview(asset);
  expect(preview.kind).toBe('run');
  expect('dependency' in preview).toBe(true);
  expect(asset.manifest.capture?.asset.hash).toBe(asset.manifest.asset.hash);
  const sent = JSON.stringify(evidenceInput([asset]));
  expect(sent).toContain('prepare');
  expect(sent).toContain('align:S03');
  expect(sent).toContain('observed-authored-task-pair');
  expect(sent).not.toContain('NEVER INCLUDE');
  const frozen = structuredClone(preview);
  data.dependencies!.groups[0]!.members = [];
  expect(evidencePreview(asset)).toEqual(frozen);
  const wrong = structuredClone(asset);
  wrong.manifest.evidence = target();
  wrong.manifest.evidence.selection = {
    kind: 'run-group',
    coordinateSpace: 'observed-authored-task-group',
    taskId: 'prepare',
  };
  expect(() => evidencePreview(wrong)).toThrow('does not match');
  expect(() =>
    materializeObserved({ ...attachment, evidence: { ...ref, dataRevision: 'changed' } }, source()),
  ).toThrow('does not match');
});
it('uses retained coherent data for navigation/capture and requires exact current render authority', async () => {
  const data = source(),
    projectId = data.value.projectId,
    base = await directory();
  const read = vi.fn(async () => structuredClone(data));
  const evidenceStorage = new EvidenceStorage(join(base, 'evidence'));
  const workspace = new WorkspaceController(
    new WorkspaceStorage(join(base, 'workspace')),
    {
      projects: async () => [{ projectId, name: 'Dependency', rootResourceId: 'res_root' }],
      describe: async () => ({ view: 'run', title: 'Run' }),
      read,
    },
    () => true,
    new EvidenceCapture(evidenceStorage),
  );
  await workspace.initialize();
  const session = await workspace.connect();
  await workspace.openProject(projectId);
  let counter = 0;
  const command = async (action: WorkspaceAction) =>
    workspace.command({
      projectId,
      requestId: 'req_dep_' + ++counter,
      expectedRevision: (await workspace.read(projectId)).workspace.revision,
      action,
    });
  const doc = await command({
      kind: 'open',
      resource: target(data).resource,
      pane: 'primary',
      duplicate: false,
    }),
    surfaceId = doc.workspace.layout.primary.activeSurfaceId!;
  await workspace.present({
    projectId,
    rendererSessionId: session.rendererSessionId,
    visiblePanes: ['primary'],
  });
  const load = () =>
    workspace.loadSurface({ projectId, surfaceId, rendererSessionId: session.rendererSessionId });
  let shown = await load();
  await workspace.acknowledge(shown.acknowledgment);
  await command({ kind: 'runMode', surfaceId, mode: 'dependencies' });
  shown = await load();
  await workspace.acknowledge(shown.acknowledgment);
  const ref = target(data);
  await expect(command({ kind: 'select', surfaceId, evidence: ref })).rejects.toMatchObject({
    code: 'stale_revision',
  });
  await expect(
    command({
      kind: 'select',
      surfaceId,
      evidence: { ...ref, dataRevision: 'stale' },
      acknowledgment: shown.acknowledgment,
    }),
  ).rejects.toMatchObject({ code: 'stale_revision' });
  await command({ kind: 'dependencyCamera', surfaceId, camera: { zoom: 1.5, x: 50, y: 20 } });
  await command({ kind: 'select', surfaceId, evidence: ref, acknowledgment: shown.acknowledgment });
  const attached = await command({
    kind: 'attach',
    surfaceId,
    evidence: ref,
    acknowledgment: shown.acknowledgment,
  });
  expect(attached.chat.attachments).toHaveLength(1);
  expect(attached.chat.attachments![0]!.capture).toBeDefined();
  expect(read).toHaveBeenCalledTimes(1);
  await command({ kind: 'runMode', surfaceId, mode: 'tasks' });
  await expect(
    command({ kind: 'attach', surfaceId, evidence: ref, acknowledgment: shown.acknowledgment }),
  ).rejects.toThrow();
  expect((await workspace.read(projectId)).chat.attachments).toEqual(attached.chat.attachments);
});
it('keeps Agent v6 inputs closed and does not describe the dependency display as observed tasks', () => {
  const ref = target();
  expect(() => parse(toolSchemas.workspace_point, { evidence: ref, note: '' })).toThrow();
  expect(() =>
    parse(toolSchemas.workspace_observe, { surfaceId: 'srf_run', selection: ref.selection }),
  ).toThrow();
  const doc = transition(opened(), { kind: 'runMode', surfaceId: 'srf_run', mode: 'dependencies' }),
    data = source();
  const view = {
    surface: doc.workspace.surfaces[0]!,
    data,
    acknowledgment: {
      schemaVersion: 3 as const,
      projectId: data.value.projectId,
      surfaceId: 'srf_run',
      rendererSessionId: 'rnd_dep',
      requestId: 'req_dep',
      generation: 1,
      dataRevision: data.value.engineRevision,
      presentation: { spec: 0, view: 1, filter: 0 },
    },
  };
  expect(() => new ObservedReads().observe(view, undefined, 'view')).toThrow(
    'representation is Dependencies',
  );
  expect(new ObservedReads().observe(view, undefined, 'source-preview')).toBeDefined();
});

it('keeps Tasks usable when optional dependency context exceeds the retained view bound', () => {
  const raw = dependencySnapshot();
  raw.snapshot.tasks = Array.from({ length: 1000 }, (_, i) => ({
    ...raw.snapshot.tasks[2]!,
    identity: 'align:' + i,
    status: 'state'.repeat(120),
  }));
  const data: SurfaceData = {
    kind: 'run',
    value: presentRun(raw),
    dependencies: projectDependencies(presentDependencies(raw)),
  };
  expect(Buffer.byteLength(JSON.stringify(data))).toBeGreaterThan(1024 * 1024);
  expect(Buffer.byteLength(JSON.stringify({ kind: 'run', value: data.value }))).toBeLessThan(
    1024 * 1024,
  );
  const render = new RenderSession(),
    surface = opened().workspace.surfaces[0]!;
  const request = {
    projectId: data.value.projectId,
    surfaceId: surface.surfaceId,
    rendererSessionId: render.id,
  };
  render.reveal([surface.surfaceId]);
  const loaded = render.complete(render.begin(request, surface), data);
  render.acknowledge(loaded.acknowledgment);
  expect(loaded.data).toMatchObject({
    kind: 'run',
    value: data.value,
    dependencyProblem: expect.stringContaining('memory limit'),
  });
  expect(loaded.dependencyRevision).toBeUndefined();
  expect(loaded.observedRevision).toBe(observedDataRevision(data));
  expect(render.snapshot(surface.surfaceId).data).not.toHaveProperty('dependencies');
  expect(data.dependencies).toBeDefined();
});
