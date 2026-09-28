import { mkdtemp, readFile, writeFile, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { afterEach, describe, expect, it } from 'vitest';
import {
  parseWorkspaceDocument,
  type ResourceRef,
  type SurfaceData,
  type WorkspaceAction,
} from '@gobble/contracts';
import { clampBounds } from '../src/main/workspace/bounds';
import { checkSelection, dataRevision } from '../src/main/workspace/selection';
import { WorkspaceController } from '../src/main/workspace/controller';
import { emptyWorkspace, transition } from '../src/main/workspace/model';
import { WorkspaceStorage, AtomicStateFile } from '../src/main/workspace/storage';
import type { WorkspaceResources } from '../src/main/workspace/service';

const directories: string[] = [];
afterEach(async () => {
  await Promise.all(
    directories.splice(0).map((path) => rm(path, { recursive: true, force: true })),
  );
});
async function directory() {
  const path = await mkdtemp(join(tmpdir(), 'gobble-workspace-'));
  directories.push(path);
  return path;
}
const resource: ResourceRef = { kind: 'file', resourceId: 'res_samples' };
const data = (projectId = 'prj_atlas'): SurfaceData => ({
  kind: 'file',
  value: {
    projectId,
    resourceId: 'res_samples',
    name: 'samples.csv',
    revision: 'sha256:' + 'a'.repeat(64),
    size: 22,
    content: {
      kind: 'table',
      columns: [{ id: 'col_0', name: 'Sample' }],
      rows: [{ key: 'row_1', cells: ['S03'] }],
      truncated: false,
    },
  },
});
class Resources implements WorkspaceResources {
  projects = async () => [
    { projectId: 'prj_atlas', name: 'Atlas', rootResourceId: 'res_atlas' },
    { projectId: 'prj_other', name: 'Other', rootResourceId: 'res_other' },
  ];
  read = async (projectId: string): Promise<SurfaceData> => data(projectId);
  describe = async () => ({ title: 'samples.csv', view: 'table' as const });
}
async function setup() {
  const path = await directory();
  const resources = new Resources();
  const controller = new WorkspaceController(new WorkspaceStorage(path), resources);
  await controller.initialize();
  await controller.connect();
  const bootstrap = await controller.openProject('prj_atlas');
  await controller.present({
    projectId: 'prj_atlas',
    rendererSessionId: bootstrap.rendererSessionId,
    visiblePanes: ['primary'],
  });
  let doc = bootstrap.document ?? emptyWorkspace('prj_atlas');
  let request = 0;
  const command = async (action: WorkspaceAction) => {
    doc = await controller.command({
      projectId: doc.workspace.projectId,
      expectedRevision: doc.workspace.revision,
      requestId: 'req_' + ++request,
      action,
    });
    return doc;
  };
  return { path, resources, controller, bootstrap, command };
}

describe('Project workspace ownership and persistence', () => {
  it('reuses an open resource and duplicates it only on explicit request', async () => {
    const { command } = await setup();
    const first = await command({ kind: 'open', resource, pane: 'primary', duplicate: false });
    const reused = await command({ kind: 'open', resource, pane: 'secondary', duplicate: false });
    expect(reused.workspace.surfaces).toEqual(first.workspace.surfaces);
    expect(reused.workspace.layout.kind).toBe('single');
    const compared = await command({ kind: 'open', resource, pane: 'secondary', duplicate: true });
    expect(compared.workspace.surfaces).toHaveLength(2);
    expect(compared.workspace.layout.kind).toBe('split');
    expect(new Set(compared.workspace.surfaces.map((surface) => surface.surfaceId)).size).toBe(2);
  });

  it('preserves pinned views while opening more views and combines tabs without dropping them', async () => {
    const { command } = await setup();
    const first = await command({ kind: 'open', resource, pane: 'primary', duplicate: false });
    const id = first.workspace.surfaces[0]?.surfaceId ?? '';
    await command({ kind: 'pin', surfaceId: id, pinned: true });
    await command({ kind: 'open', resource, pane: 'secondary', duplicate: true });
    await command({ kind: 'maximize', pane: 'secondary' });
    const single = await command({ kind: 'arrange', layout: 'single' });
    expect(single.workspace.surfaces.find((surface) => surface.surfaceId === id)?.pinned).toBe(
      true,
    );
    expect(single.workspace.layout.primary.tabs).toHaveLength(2);
    expect(single.maximizedPane).toBeNull();
    expect(parseWorkspaceDocument(single)).toEqual(single);
  });

  it('rejects foreign Project changes and stale revisions without changing the workspace', async () => {
    const { controller, command } = await setup();
    await command({ kind: 'open', resource, pane: 'primary', duplicate: false });
    await expect(
      controller.command({
        projectId: 'prj_other',
        expectedRevision: 0,
        requestId: 'req_foreign',
        action: { kind: 'arrange', layout: 'split' },
      }),
    ).rejects.toMatchObject({ code: 'forbidden' });
    await expect(
      controller.command({
        projectId: 'prj_atlas',
        expectedRevision: 0,
        requestId: 'req_old',
        action: { kind: 'arrange', layout: 'split' },
      }),
    ).rejects.toMatchObject({ code: 'stale_revision' });
    expect((await controller.openProject('prj_atlas')).document?.workspace.layout.kind).toBe(
      'single',
    );
  });

  it('makes duplicate request IDs a no-op and rejects a reused ID with a different action', async () => {
    const { controller } = await setup();
    const input = {
      projectId: 'prj_atlas',
      expectedRevision: 0,
      requestId: 'req_once',
      action: { kind: 'arrange', layout: 'split' },
    } as const;
    const first = await controller.command(input);
    expect(await controller.command(input)).toEqual(first);
    await expect(
      controller.command({ ...input, action: { kind: 'arrange', layout: 'single' } }),
    ).rejects.toMatchObject({ code: 'request_conflict' });
  });

  it('restores layout, pin and unsent drafts without replaying actions', async () => {
    const { path, controller, resources, command, bootstrap } = await setup();
    const first = await command({ kind: 'open', resource, pane: 'primary', duplicate: false });
    await command({
      kind: 'pin',
      surfaceId: first.workspace.surfaces[0]?.surfaceId ?? '',
      pinned: true,
    });
    await command({ kind: 'arrange', layout: 'split' });
    const saved = await controller.updateDraft('prj_atlas', 'Review this sample.');
    await controller.stop();
    const restored = new WorkspaceController(new WorkspaceStorage(path), resources);
    await restored.initialize();
    const result = await restored.connect();
    expect(result.document).toEqual(saved);
    expect(result.document?.chat.draft).toBe('Review this sample.');
    expect(result.rendererSessionId).not.toBe(bootstrap.rendererSessionId);
  });

  it('keeps Project drafts and views separate while switching', async () => {
    const { controller, command } = await setup();
    await command({ kind: 'open', resource, pane: 'primary', duplicate: false });
    await controller.updateDraft('prj_atlas', 'Atlas draft');
    const other = await controller.openProject('prj_other');
    expect(other.document?.workspace.surfaces).toEqual([]);
    expect(other.document?.chat.draft).toBe('');
    const atlas = await controller.openProject('prj_atlas');
    expect(atlas.document?.workspace.surfaces).toHaveLength(1);
    expect(atlas.document?.chat.draft).toBe('Atlas draft');
  });

  it('preserves pending decisions and agent references when a view closes', () => {
    const initial = emptyWorkspace('prj_atlas');
    const opened = transition(
      initial,
      { kind: 'open', resource, pane: 'primary', duplicate: false },
      { surfaceId: 'srf_first', title: 'samples.csv', view: 'table' },
    );
    opened.workspace.agents.push({
      projectId: 'prj_atlas',
      agentId: 'agt_later',
      name: 'Later agent',
      instructionProfile: 'Reader',
      provider: { kind: 'codex', threadId: null },
    });
    opened.workspace.decisions.push({
      projectId: 'prj_atlas',
      decisionId: 'dec_pending',
      requestedBy: 'agt_later',
      question: 'Keep this sample?',
      evidence: [
        {
          projectId: 'prj_atlas',
          schemaVersion: 2 as const,
          origin: { surfaceId: 'srf_first' },
          resource,
          dataRevision: 'revision-1',
        },
      ],
      state: { kind: 'pending' },
    });
    const closed = transition(opened, { kind: 'close', surfaceId: 'srf_first' });
    expect(closed.workspace.decisions[0]?.state.kind).toBe('pending');
    expect(closed.workspace.agents).toEqual(opened.workspace.agents);
  });
});

describe('rendered evidence lifetime', () => {
  it('revokes ready content in chat-only mode and requires a fresh load after revealing the workspace', async () => {
    const { controller, command, bootstrap } = await setup();
    const opened = await command({ kind: 'open', resource, pane: 'primary', duplicate: false });
    const surfaceId = opened.workspace.surfaces[0]!.surfaceId;
    const input = {
      projectId: 'prj_atlas',
      surfaceId,
      rendererSessionId: bootstrap.rendererSessionId,
    };
    const loaded = await controller.loadSurface(input);
    await controller.acknowledge(loaded.acknowledgment);
    await controller.present({ ...input, visiblePanes: [] });
    await expect(controller.loadSurface(input)).rejects.toMatchObject({ code: 'stale_revision' });
    await expect(controller.acknowledge(loaded.acknowledgment)).rejects.toMatchObject({
      code: 'stale_revision',
    });
    await controller.present({ ...input, visiblePanes: ['primary'] });
    await expect(controller.acknowledge(loaded.acknowledgment)).rejects.toMatchObject({
      code: 'stale_revision',
    });
    const fresh = await controller.loadSurface(input);
    await expect(controller.acknowledge(fresh.acknowledgment)).resolves.toEqual({ accepted: true });
  });
  it('rejects a foreign Project, obsolete window and impossible Pane presentation', async () => {
    const { controller, bootstrap } = await setup();
    const input = {
      projectId: 'prj_atlas',
      rendererSessionId: bootstrap.rendererSessionId,
      visiblePanes: ['primary'] as ('primary' | 'secondary')[],
    };
    await expect(controller.present({ ...input, projectId: 'prj_other' })).rejects.toMatchObject({
      code: 'forbidden',
    });
    await expect(
      controller.present({ ...input, rendererSessionId: 'rnd_old' }),
    ).rejects.toMatchObject({ code: 'stale_revision' });
    await expect(
      controller.present({ ...input, visiblePanes: ['secondary'] }),
    ).rejects.toMatchObject({ code: 'stale_revision' });
  });

  it('rejects stale render acknowledgments, closed views and unknown row selections', async () => {
    const { controller, command, bootstrap } = await setup();
    const opened = await command({ kind: 'open', resource, pane: 'primary', duplicate: false });
    const surfaceId = opened.workspace.surfaces[0]?.surfaceId ?? '';
    const input = {
      projectId: 'prj_atlas',
      surfaceId,
      rendererSessionId: bootstrap.rendererSessionId,
    };
    const first = await controller.loadSurface(input);
    const second = await controller.loadSurface(input);
    await expect(controller.acknowledge(first.acknowledgment)).rejects.toMatchObject({
      code: 'stale_revision',
    });
    await controller.acknowledge(second.acknowledgment);
    await expect(
      command({
        kind: 'select',
        surfaceId,
        evidence: {
          projectId: input.projectId,
          schemaVersion: 2 as const,
          origin: { surfaceId: surfaceId },
          resource,
          dataRevision: second.acknowledgment.dataRevision,
          selection: {
            coordinateSpace: 'revision-row-column-keys',
            kind: 'table',
            rowKeys: ['missing'],
            columns: ['col_0'],
          },
        },
      }),
    ).rejects.toMatchObject({ code: 'stale_revision' });
    await command({ kind: 'close', surfaceId });
    await expect(controller.acknowledge(second.acknowledgment)).rejects.toMatchObject({
      code: 'stale_revision',
    });
  });

  it('binds local selection to the action Surface while origin cannot confer authority', async () => {
    const { controller, command, bootstrap } = await setup();
    const first = await command({ kind: 'open', resource, pane: 'primary', duplicate: false });
    const surfaceId = first.workspace.surfaces[0]!.surfaceId;
    const duplicated = await command({
      kind: 'open',
      resource,
      pane: 'secondary',
      duplicate: true,
    });
    const loaded = await controller.loadSurface({
      projectId: 'prj_atlas',
      surfaceId,
      rendererSessionId: bootstrap.rendererSessionId,
    });
    await controller.acknowledge(loaded.acknowledgment);
    const evidence = {
      projectId: 'prj_atlas',
      schemaVersion: 2 as const,
      origin: { surfaceId: duplicated.workspace.surfaces[1]!.surfaceId },
      resource,
      dataRevision: loaded.acknowledgment.dataRevision,
      selection: {
        coordinateSpace: 'revision-row-column-keys' as const,
        kind: 'table' as const,
        rowKeys: ['row_1'],
        columns: ['col_0'],
      },
    };
    const selected = await command({ kind: 'select', surfaceId, evidence });
    expect(selected.selections).toEqual([{ surfaceId, evidence }]);
    for (const invalid of [
      { ...evidence, projectId: 'prj_other' },
      { ...evidence, resource: { kind: 'file' as const, resourceId: 'res_other' } },
    ])
      await expect(command({ kind: 'select', surfaceId, evidence: invalid })).rejects.toMatchObject(
        { code: 'forbidden' },
      );
  });

  it('invalidates a log selection when the tail changes without a new engine checkpoint', () => {
    const first: SurfaceData = {
      kind: 'log',
      value: {
        projectId: 'prj_atlas',
        runRef: 'run_first',
        engineRevision: 'same-checkpoint',
        observedAt: 1,
        instance: 'align',
        attempt: 1,
        tailLimitBytes: 4096,
        logs: [{ identity: 'align', stdout_tail: 'First output' }],
      },
    };
    const next = structuredClone(first);
    if (!('logs' in next.value)) throw new Error('Expected legacy log fixture');
    next.value.logs[0] = { identity: 'align', stdout_tail: 'Next output' };
    const evidence = {
      projectId: 'prj_atlas',
      schemaVersion: 2 as const,
      origin: { surfaceId: 'srf_logs' },
      resource: { kind: 'log' as const, runRef: 'run_first', taskId: 'align', attempt: 1 },
      dataRevision: dataRevision(first),
    };
    expect(() => checkSelection(evidence, first)).not.toThrow();
    expect(() => checkSelection(evidence, next)).toThrow('no longer matches');
  });

  it('does not let a late load replace a newer load', async () => {
    const { controller, command, bootstrap, resources } = await setup();
    const opened = await command({ kind: 'open', resource, pane: 'primary', duplicate: false });
    const input = {
      projectId: 'prj_atlas',
      surfaceId: opened.workspace.surfaces[0]?.surfaceId ?? '',
      rendererSessionId: bootstrap.rendererSessionId,
    };
    let complete: ((value: SurfaceData) => void) | undefined;
    const waiting = new Promise<SurfaceData>((resolve) => {
      complete = resolve;
    });
    resources.read = async () => waiting;
    const first = controller.loadSurface(input);
    const firstRejected = expect(first).rejects.toMatchObject({ code: 'stale_revision' });
    await controller.flush();
    resources.read = async () => data();
    const second = await controller.loadSurface(input);
    complete?.(data());
    await firstRejected;
    await expect(controller.acknowledge(second.acknowledgment)).resolves.toEqual({
      accepted: true,
    });
  });

  it('invalidates observations when the window closes or a pane becomes hidden', async () => {
    const { controller, command, bootstrap } = await setup();
    const opened = await command({ kind: 'open', resource, pane: 'primary', duplicate: false });
    const surfaceId = opened.workspace.surfaces[0]?.surfaceId ?? '';
    const loaded = await controller.loadSurface({
      projectId: 'prj_atlas',
      surfaceId,
      rendererSessionId: bootstrap.rendererSessionId,
    });
    await command({ kind: 'open', resource, pane: 'secondary', duplicate: true });
    await controller.present({
      projectId: 'prj_atlas',
      rendererSessionId: bootstrap.rendererSessionId,
      visiblePanes: ['secondary'],
    });
    await expect(controller.acknowledge(loaded.acknowledgment)).rejects.toMatchObject({
      code: 'stale_revision',
    });
    controller.disconnect();
    await expect(
      controller.loadSurface({
        projectId: 'prj_atlas',
        surfaceId,
        rendererSessionId: bootstrap.rendererSessionId,
      }),
    ).rejects.toMatchObject({ code: 'stale_revision' });
  });
});

describe('workspace storage recovery', () => {
  it('retains the previous valid file as a backup', async () => {
    const path = join(await directory(), 'state.json');
    const file = new AtomicStateFile(path, parseWorkspaceDocument);
    const first = emptyWorkspace('prj_atlas');
    await file.write(first);
    const next = { ...first, chat: { ...first.chat, draft: 'Unsent' } };
    await file.write(next);
    expect(JSON.parse(await readFile(path + '.backup', 'utf8'))).toEqual(first);
    expect(await new AtomicStateFile(path, parseWorkspaceDocument).read()).toEqual(next);
  });

  it('preserves corrupt and future state and refuses automatic overwrite', async () => {
    for (const source of [
      '{broken',
      JSON.stringify({ ...emptyWorkspace('prj_atlas'), schemaVersion: 999 }),
    ]) {
      const path = join(await directory(), 'state.json');
      await writeFile(path, source);
      const file = new AtomicStateFile(path, parseWorkspaceDocument);
      await expect(file.read()).rejects.toMatchObject({ code: 'internal' });
      await expect(file.write(emptyWorkspace('prj_atlas'))).rejects.toMatchObject({
        code: 'internal',
      });
      expect(await readFile(path, 'utf8')).toBe(source);
    }
  });

  it('refuses to discard a backup or overwrite external changes', async () => {
    const path = join(await directory(), 'state.json');
    const first = emptyWorkspace('prj_atlas');
    const file = new AtomicStateFile(path, parseWorkspaceDocument);
    await file.write(first);
    await writeFile(path, '{external change');
    await expect(
      file.write({ ...first, activity: [{ id: 'req_test', at: 1, text: 'A change' }] }),
    ).rejects.toMatchObject({ code: 'internal' });
    expect(await readFile(path, 'utf8')).toBe('{external change');
    await writeFile(path + '.backup', JSON.stringify(first));
    await rm(path);
    await expect(new AtomicStateFile(path, parseWorkspaceDocument).read()).rejects.toMatchObject({
      code: 'internal',
    });
  });
});

describe('native window recovery', () => {
  it('moves an offscreen window onto the available display without exceeding its work area', () => {
    expect(
      clampBounds(
        { x: 4000, y: -900, width: 1600, height: 1200 },
        { x: -1440, y: 25, width: 1440, height: 850 },
      ),
    ).toEqual({ x: -1440, y: 25, width: 1440, height: 850 });
  });
  it('centers a new window and expands an undersized saved window', () => {
    expect(clampBounds(null, { x: 0, y: 25, width: 1440, height: 900 })).toEqual({
      x: 80,
      y: 55,
      width: 1280,
      height: 840,
    });
    expect(
      clampBounds(
        { x: 0, y: 25, width: 640, height: 480 },
        { x: 0, y: 25, width: 1440, height: 900 },
      ),
    ).toEqual({ x: 0, y: 25, width: 720, height: 520 });
  });
});
