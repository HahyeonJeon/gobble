import { readFileSync } from 'node:fs';
import { parse, PipelineFlowSchema, pipelineTarget } from '@gobble/contracts';
import { notebookPart, projectDependencies } from '@gobble/contracts';
import { dependencySnapshot } from './fixtures/dependency-snapshot';
import { presentDependencies } from '../src/main/service/dependency-presentation';
import { presentRun } from '../src/main/service/run-presentation';
import { mkdtemp, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { afterEach, describe, expect, it } from 'vitest';
import {
  parseWorkspaceDocument,
  SHARED_TOOLSET,
  type AgentAttachment,
  type EvidenceRef,
  type SurfaceData,
  type WorkspaceAction,
} from '@gobble/contracts';
import { WorkspaceStorage } from '../src/main/workspace/storage';
import { WorkspaceController } from '../src/main/workspace/controller';
import { SharedContextHost, type ResourceCatalog } from '../src/main/shared-context/host';
import type { WorkspaceResources } from '../src/main/workspace/service';
import type { ImageRenderer } from '../src/main/shared-context/observation';
import { moveTextRange } from '../src/renderer/shared-context/text-range';
import { boundedText, textObservation } from '../src/main/shared-context/observation';

const paths: string[] = [];
afterEach(async () => {
  for (const path of paths.splice(0)) await rm(path, { recursive: true, force: true });
});
const revision = 'sha256:' + 'a'.repeat(64);
const textData = (id = 'res_text', text = 'Alpha\nBeta 😀 gamma\n'): SurfaceData => ({
  kind: 'file',
  value: {
    projectId: 'prj_one',
    resourceId: id,
    name: 'notes.txt',
    revision,
    size: text.length,
    content: { kind: 'text', text },
  },
});
async function setup() {
  const path = await mkdtemp(join(tmpdir(), 'gobble-shared-'));
  paths.push(path);
  let available = true;
  const data = new Map<string, SurfaceData>([
    ['res_text', textData()],
    ['res_other', textData('res_other')],
    [
      'res_image',
      {
        kind: 'file',
        value: {
          projectId: 'prj_one',
          resourceId: 'res_image',
          name: 'plot.png',
          revision,
          size: 10,
          content: {
            kind: 'image',
            width: 100,
            height: 80,
            mediaType: 'image/png',
            base64: 'fixture',
          },
        },
      },
    ],
  ]);
  const resources: WorkspaceResources = {
    validateNotebook: (_id, target, data) => {
      if (data.kind !== 'file' || data.value.content.kind !== 'notebook')
        throw new Error('Notebook required');
      notebookPart(data.value.content.document, target.selection);
    },
    projects: async () =>
      ['one', 'two'].map((id) => ({
        projectId: 'prj_' + id,
        rootResourceId: 'res_' + id,
        name: id,
      })),
    read: async (_project, ref) => {
      if (ref.kind === 'report' || ref.kind === 'creation-draft')
        throw new Error('Use creation review tools');
      const found = data.get(
        ref.kind === 'pipeline'
          ? ref.pipelineId
          : ref.kind === 'file'
            ? ref.resourceId
            : ref.runRef,
      );
      if (!found) throw new Error('Unknown resource');
      return found;
    },
    describe: async (project, ref) => {
      const value = await resources.read(project, ref);
      return {
        title: 'Preview',
        view: value.kind === 'file' ? value.value.content.kind : value.kind,
      };
    },
  };
  const storage = new WorkspaceStorage(path);
  const workspace = new WorkspaceController(storage, resources, () => available);
  await workspace.initialize();
  let session = await workspace.connect();
  await workspace.openProject('prj_one');
  const agent: AgentAttachment = {
    projectId: 'prj_one',
    agentId: 'agt_one',
    name: 'Researcher',
    instructionProfile: 'discussion-v1',
    access: 'sharedViews',
    configuration: { model: 'fixture', effort: 'low', instructions: '' },
    provider: {
      kind: 'codex',
      threadId: 'thread_one',
      accountSessionId: 'session',
      toolsetVersion: SHARED_TOOLSET,
    },
  };
  const submission = {
    requestId: 'req_submission',
    agentId: agent.agentId,
    text: 'Inspect',
    model: 'fixture',
    effort: 'low',
    createdAt: 1,
    state: 'running' as const,
    threadId: 'thread_one',
    turnId: 'turn_one',
    response: '',
    problem: null,
  };
  await workspace.changeCollaboration('prj_one', () => ({
    agents: [agent],
    history: { submissions: [submission] },
  }));
  const catalog: ResourceCatalog = {
    listFiles: async () => ({
      projectId: 'prj_one',
      directoryId: 'res_one',
      entries: [],
      truncated: false,
    }),
    listRuns: async () => ({ projectId: 'prj_one', runs: [], candidates: [], truncated: false }),
    readRun: async () => {
      throw new Error('Unused');
    },
    readLogs: async () => {
      throw new Error('Unused');
    },
  };
  let image: ImageRenderer = async () => ({
    url: 'data:image/png;base64,AA==',
    width: 100,
    height: 80,
    crop: { x: 0, y: 0, width: 100, height: 80 },
  });
  let images = true;
  let authorized = true;
  const signal = new AbortController();
  const host = new SharedContextHost(
    workspace,
    resources,
    catalog,
    (...args) => image(...args),
    () => images,
  );
  host.begin(agent.projectId, agent.agentId, submission.requestId);
  let serial = 0;
  const invoke = (tool: string, args: unknown = {}, callId = 'call_' + ++serial) =>
    host.execute(
      {
        agent,
        submission,
        signal: signal.signal,
        assert: () => {
          if (!authorized) throw new Error('Revoked');
        },
      },
      { threadId: 'thread_one', turnId: 'turn_one', callId, tool, arguments: args },
    );
  const command = async (action: WorkspaceAction) =>
    workspace.command({
      projectId: 'prj_one',
      expectedRevision: (await workspace.read('prj_one')).workspace.revision,
      requestId: 'req_' + ++serial,
      action,
    });
  const open = async (resourceId = 'res_text') => {
    const doc = await command({
      kind: 'open',
      resource: { kind: 'file', resourceId },
      pane: 'primary',
      duplicate: false,
    });
    const surface = doc.workspace.surfaces.find(
      (item) => item.resource.kind === 'file' && item.resource.resourceId === resourceId,
    )!;
    if (
      surface.view === 'report' ||
      surface.view === 'pipeline' ||
      surface.view === 'creation-draft'
    )
      throw new Error('Expected file fixture');
    return surface;
  };
  const ready = async (
    surfaceId: string,
    visiblePanes: ('primary' | 'secondary')[] = ['primary'],
  ) => {
    session = { ...session, rendererSessionId: session.rendererSessionId };
    await workspace.present({
      projectId: 'prj_one',
      rendererSessionId: session.rendererSessionId,
      visiblePanes,
    });
    const load = await workspace.loadSurface({
      projectId: 'prj_one',
      surfaceId,
      rendererSessionId: session.rendererSessionId,
    });
    await workspace.acknowledge(load.acknowledgment);
    return load;
  };
  return {
    path,
    storage,
    workspace,
    host,
    invoke,
    command,
    open,
    ready,
    data,
    agent,
    submission,
    signal,
    setImage: (value: ImageRenderer) => {
      image = value;
    },
    setAvailable: (value: boolean) => {
      available = value;
      workspace.invalidatePresentation();
    },
    setImages: (value: boolean) => {
      images = value;
    },
    revoke: () => {
      authorized = false;
    },
  };
}
function body(result: Awaited<ReturnType<SharedContextHost['execute']>>) {
  return JSON.parse(result.content.find((item) => item.type === 'text')!.text) as Record<
    string,
    any
  >;
}

describe('shared Project tools and authored references', () => {
  it('keeps User selection/draft private while publishing the same authored contract for both peers', async () => {
    const s = await setup();
    const surface = await s.open();
    await s.ready(surface.surfaceId);
    const evidence: EvidenceRef = {
      projectId: 'prj_one',
      schemaVersion: 2 as const,
      origin: { surfaceId: surface.surfaceId },
      resource: surface.resource,
      dataRevision: revision,
      selection: {
        coordinateSpace: 'utf16-line-column',
        kind: 'text',
        start: { line: 2, column: 0 },
        end: { line: 2, column: 7 },
      },
    };
    await s.command({ kind: 'select', surfaceId: surface.surfaceId, evidence });
    await s.workspace.updateDraft('prj_one', 'private draft');
    await s.command({
      surfaceId: evidence.origin!.surfaceId,
      kind: 'share',
      evidence,
      note: 'User pointer',
    });
    expect((await s.invoke('workspace_point', { evidence, note: 'Agent pointer' })).success).toBe(
      true,
    );
    const doc = await s.workspace.read('prj_one');
    expect(doc.selections).toEqual([{ surfaceId: surface.surfaceId, evidence }]);
    expect(doc.chat.draft).toBe('private draft');
    expect(doc.sharedReferences?.map((item) => item.author.kind)).toEqual(['user', 'agent']);
    const listed = body(await s.invoke('workspace_list'));
    expect(listed.references).toHaveLength(2);
    expect(JSON.stringify(listed)).not.toContain('private draft');
    expect(listed.selections).toBeUndefined();
    expect(listed.collaboration).toBeUndefined();
    const referenceId = doc.sharedReferences![0]!.referenceId;
    await s.command({ kind: 'retract', referenceId });
    expect((await s.workspace.read('prj_one')).sharedReferences![0]!.retracted).toBe(true);
    expect((await s.storage.project('prj_one').read())?.sharedReferences).toHaveLength(2);
  });
  it('opens two temporary views atomically in separate Panes and protects user claims', async () => {
    const s = await setup();
    const result = body(
      await s.invoke('workspace_open', {
        resources: [
          { kind: 'file', resourceId: 'res_text' },
          { kind: 'file', resourceId: 'res_image' },
        ],
      }),
    );
    expect(result.views.map((view: any) => view.pane)).toEqual(['primary', 'secondary']);
    expect(result.views.every((view: any) => view.visibility === 'opening')).toBe(true);
    expect(result.observed).toBe(false);
    const first = result.views[0].surfaceId;
    await s.command({ kind: 'activate', surfaceId: first });
    expect((await s.invoke('workspace_release', { surfaceId: first })).success).toBe(false);
    expect(
      (await s.invoke('workspace_release', { surfaceId: result.views[1].surfaceId })).success,
    ).toBe(true);
    const before = await s.workspace.read('prj_one');
    expect(
      (
        await s.invoke('workspace_open', {
          resources: [
            { kind: 'file', resourceId: 'res_other' },
            { kind: 'file', resourceId: 'res_missing' },
          ],
        })
      ).success,
    ).toBe(false);
    expect(await s.workspace.read('prj_one')).toEqual(before);
  });
  it('preserves protected foreground views and honors user dismissal for the entire submission', async () => {
    const s = await setup();
    const first = await s.open();
    const second = await s.open('res_image');
    await s.command({ kind: 'move', surfaceId: second.surfaceId, pane: 'secondary' });
    const result = body(
      await s.invoke('workspace_open', { resources: [{ kind: 'file', resourceId: 'res_other' }] }),
    );
    expect(result.views[0].visibility).toBe('background');
    expect((await s.workspace.read('prj_one')).workspace.layout.primary.activeSurfaceId).toBe(
      first.surfaceId,
    );
    await s.command({ kind: 'close', surfaceId: result.views[0].surfaceId });
    expect(
      (await s.invoke('workspace_open', { resources: [{ kind: 'file', resourceId: 'res_other' }] }))
        .success,
    ).toBe(false);
  });
  it('deduplicates a mutation, rejects conflicting arguments and caps distinct calls', async () => {
    const s = await setup();
    const args = { resources: [{ kind: 'file', resourceId: 'res_text' }] };
    const first = await s.invoke('workspace_open', args, 'same');
    expect(await s.invoke('workspace_open', args, 'same')).toEqual(first);
    expect((await s.workspace.read('prj_one')).workspace.surfaces).toHaveLength(1);
    expect((await s.invoke('workspace_list', {}, 'same')).success).toBe(false);
    for (let i = 0; i < 31; i++)
      expect((await s.invoke('workspace_list', {}, 'list_' + i)).success).toBe(true);
    expect(body(await s.invoke('workspace_list', {}, 'limit')).error.code).toBe('unsupported');
  });
  it('requires ready visible observations, and revalidates cached reads after navigation', async () => {
    const s = await setup();
    const surface = await s.open();
    const args = { surfaceId: surface.surfaceId };
    expect((await s.invoke('workspace_observe', args)).success).toBe(false);
    await s.ready(surface.surfaceId);
    const seen = await s.invoke('workspace_observe', args, 'observe');
    expect(body(seen).content.text).toContain('Beta 😀');
    await s.workspace.openProject('prj_two');
    expect((await s.invoke('workspace_observe', args, 'observe')).success).toBe(false);
    expect((await s.invoke('workspace_list')).success).toBe(true);
    expect(
      (await s.invoke('workspace_open', { resources: [{ kind: 'file', resourceId: 'res_other' }] }))
        .success,
    ).toBe(false);
  });
  it.each(['modal', 'blur', 'cancel', 'refresh', 'revoke'] as const)(
    'discards an image result invalidated by %s during materialization',
    async (reason) => {
      const s = await setup();
      const surface = await s.open('res_image');
      await s.ready(surface.surfaceId);
      let complete!: (value: Awaited<ReturnType<ImageRenderer>>) => void;
      let started!: () => void;
      const began = new Promise<void>((resolve) => {
        started = resolve;
      });
      s.setImage(async () => {
        started();
        return new Promise((resolve) => {
          complete = resolve;
        });
      });
      const pending = s.invoke('workspace_observe', { surfaceId: surface.surfaceId });
      await began;
      if (reason === 'modal') {
        await s.workspace.interaction({ token: 'req_modal', blocked: true });
        await s.workspace.interaction({ token: 'req_modal', blocked: false });
      }
      if (reason === 'blur') {
        s.setAvailable(false);
        s.setAvailable(true);
      }
      if (reason === 'cancel') s.signal.abort();
      if (reason === 'refresh') await s.ready(surface.surfaceId);
      if (reason === 'revoke') s.revoke();
      complete({
        url: 'data:image/png;base64,AA==',
        width: 100,
        height: 80,
        crop: { x: 0, y: 0, width: 100, height: 80 },
      });
      const result = await pending;
      expect(result.success).toBe(false);
      expect(result.content.some((item) => item.type === 'image')).toBe(false);
    },
  );
  it('rejects foreign or stale evidence, unsupported image models and forged authority arguments', async () => {
    const s = await setup();
    const surface = await s.open('res_image');
    await s.ready(surface.surfaceId);
    s.setImages(false);
    expect((await s.invoke('workspace_observe', { surfaceId: surface.surfaceId })).success).toBe(
      false,
    );
    const evidence = {
      projectId: 'prj_two',
      schemaVersion: 2 as const,
      origin: { surfaceId: surface.surfaceId },
      resource: surface.resource,
      dataRevision: revision,
    };
    expect((await s.invoke('workspace_point', { evidence, note: 'Foreign' })).success).toBe(false);
    evidence.projectId = 'prj_one';
    evidence.dataRevision = 'sha256:' + 'b'.repeat(64);
    expect((await s.invoke('workspace_point', { evidence, note: 'Stale' })).success).toBe(false);
    expect((await s.invoke('workspace_list', { projectId: 'prj_two' })).success).toBe(false);
    expect((await s.invoke('command_exec', { command: 'anything' })).success).toBe(false);
    expect((await s.workspace.read('prj_one')).sharedReferences).toBeUndefined();
  });
  it('retains reference history after closing resources, and rejects foreign authors on restoration', async () => {
    const s = await setup();
    const surface = await s.open();
    await s.ready(surface.surfaceId);
    await s.command({
      kind: 'share',
      surfaceId: surface.surfaceId,
      evidence: {
        projectId: 'prj_one',
        schemaVersion: 2 as const,
        origin: { surfaceId: surface.surfaceId },
        resource: surface.resource,
        dataRevision: revision,
      },
      note: '',
    });
    await s.command({ kind: 'close', surfaceId: surface.surfaceId });
    const doc = await s.workspace.read('prj_one');
    expect(parseWorkspaceDocument(doc).sharedReferences).toHaveLength(1);
    doc.sharedReferences![0]!.evidence.projectId = 'prj_two';
    expect(() => parseWorkspaceDocument(doc)).toThrow();
  });
  it('publishes a portable target after closing and reopening its original Surface, without origin metadata', async () => {
    const s = await setup();
    const original = await s.open();
    await s.ready(original.surfaceId);
    const observed = body(await s.invoke('workspace_observe', { surfaceId: original.surfaceId }));
    const { origin: _origin, ...evidence } = observed.receipt.evidence;
    await s.command({ kind: 'close', surfaceId: original.surfaceId });
    expect((await s.invoke('workspace_point', { evidence, note: 'No ready source' })).success).toBe(
      false,
    );
    const reopened = await s.open();
    expect(reopened.surfaceId).not.toBe(original.surfaceId);
    await s.ready(reopened.surfaceId);
    expect(
      (await s.invoke('workspace_point', { evidence, note: 'Portable reference' })).success,
    ).toBe(true);
    const doc = await s.workspace.read('prj_one');
    const reference = doc.sharedReferences![0]!;
    expect(reference.evidence).not.toHaveProperty('origin');
    await s.command({ kind: 'close', surfaceId: reopened.surfaceId });
    const revealed = await s.command({ kind: 'reveal', referenceId: reference.referenceId });
    expect(revealed.workspace.surfaces).toHaveLength(1);
    expect(revealed.workspace.surfaces[0]!.resource).toEqual(evidence.resource);
    expect(revealed.sharedReferences![0]!.evidence).toEqual(evidence);
  });
  it('bounds preview text without cutting UTF-8 characters and returns exact UTF-16 ranges', () => {
    const value = boundedText('😀'.repeat(30000));
    expect(Buffer.byteLength(value.text)).toBeLessThanOrEqual(65536);
    expect(value.truncated).toBe(true);
    expect(value.text).not.toContain('\ufffd');
    expect(
      textObservation(textData(), {
        coordinateSpace: 'utf16-line-column',
        kind: 'text',
        start: { line: 2, column: 5 },
        end: { line: 2, column: 7 },
      }),
    ).toMatchObject({ text: '😀', truncated: false });
  });
});

it('bounds retained observation results per active submission without evicting retry receipts', async () => {
  const s = await setup();
  const surface = await s.open('res_image');
  await s.ready(surface.surfaceId);
  s.setImage(async () => ({
    url: 'data:image/png;base64,' + 'A'.repeat(1024 * 1024),
    width: 100,
    height: 80,
    crop: { x: 0, y: 0, width: 100, height: 80 },
  }));
  for (let index = 0; index < 7; index++)
    expect(
      (await s.invoke('workspace_observe', { surfaceId: surface.surfaceId }, 'image_' + index))
        .success,
    ).toBe(true);
  expect(
    body(await s.invoke('workspace_observe', { surfaceId: surface.surfaceId }, 'image_overflow'))
      .error.message,
  ).toContain('cache limit');
  expect(
    (await s.invoke('workspace_observe', { surfaceId: surface.surfaceId }, 'image_0')).success,
  ).toBe(true);
  expect((await s.invoke('workspace_list')).success).toBe(true);
});

it('supports read-only keyboard range extension across lines and Unicode without splitting characters', () => {
  expect(moveTextRange('\nABC', 0, 0, 'none', 'Home', false, false)).toEqual({
    start: 0,
    end: 0,
    direction: 'none',
  });
  expect(moveTextRange('A😀\nBC', 1, 1, 'none', 'ArrowRight', true, false)).toEqual({
    start: 1,
    end: 3,
    direction: 'forward',
  });
  expect(moveTextRange('A😀\nBC', 1, 3, 'forward', 'ArrowLeft', false, false)).toEqual({
    start: 1,
    end: 1,
    direction: 'none',
  });
  expect(moveTextRange('ABC\nD', 2, 2, 'none', 'ArrowDown', true, false)).toEqual({
    start: 2,
    end: 5,
    direction: 'forward',
  });
  expect(moveTextRange('ABC', 1, 2, 'backward', 'Home', true, false)).toEqual({
    start: 0,
    end: 2,
    direction: 'backward',
  });
});

it('rejects removed Agent chart-open arguments without opening any view', async () => {
  const s = await setup();
  const before = await s.workspace.read('prj_one');
  const result = await s.invoke('workspace_open', {
    tableSurfaceId: 'srf_table',
    scatter: {
      xColumnId: 'x',
      yColumnId: 'y',
      xScale: 'linear',
      yScale: 'linear',
    },
  });
  expect(result.success).toBe(false);
  expect(await s.workspace.read('prj_one')).toEqual(before);
});

it('observes an empty modern Run with a nonpointable v5 receipt', async () => {
  const s = await setup();
  s.data.set('run_test', {
    kind: 'run',
    value: {
      projectId: 'prj_one',
      runRef: 'run_test',
      runId: 'engine',
      engineRevision: 'revision',
      observedAt: 1,
      imageId: 'image',
      pipelineName: null,
      status: 'failed',
      tasks: [],
      dependencies: [],
    },
  });
  const document = await s.command({
    kind: 'open',
    resource: { kind: 'run', runRef: 'run_test' },
    pane: 'primary',
    duplicate: false,
  });
  const surfaceId = document.workspace.layout.primary.activeSurfaceId!;
  await s.ready(surfaceId);
  const result = await s.invoke('workspace_observe', { surfaceId });
  expect(result.success).toBe(true);
  expect(body(result)).toMatchObject({
    receipt: {
      evidence: { schemaVersion: 3 },
      observedReadId: expect.stringMatching(/^obs_/),
      pointable: false,
    },
    content: { kind: 'run', tasks: [] },
  });
});

it('binds Agent task pointers to a returned scope and preserves User filters through Show and Return', async () => {
  const s = await setup();
  const data: Extract<SurfaceData, { kind: 'run' }> = {
    kind: 'run',
    value: {
      projectId: 'prj_one',
      runRef: 'run_test',
      runId: 'engine',
      engineRevision: 'revision',
      observedAt: 1,
      imageId: 'image',
      pipelineName: null,
      status: 'failed',
      dependencies: [],
      tasks: ['one', 'two'].map((instanceId) => ({
        instanceId,
        taskId: instanceId,
        name: instanceId,
        status: 'failed',
        attempt: 2,
        reason: null,
      })),
    },
  };
  s.data.set('run_test', data);
  const opened = await s.command({
    kind: 'open',
    resource: { kind: 'run', runRef: 'run_test' },
    pane: 'primary',
    duplicate: false,
  });
  const surfaceId = opened.workspace.layout.primary.activeSurfaceId!;
  await s.command({ kind: 'runFilter', surfaceId, filter: { query: 'one', status: null } });
  await s.ready(surfaceId);
  const displayed = body(await s.invoke('workspace_observe', { surfaceId }));
  expect(displayed.content.tasks.map((task: { instanceId: string }) => task.instanceId)).toEqual([
    'one',
  ]);
  const selection = {
    kind: 'run-task',
    coordinateSpace: 'observed-instance-attempt',
    instanceId: 'two',
    attempt: 2,
  };
  const pointer = {
    evidence: { ...displayed.receipt.evidence, selection },
    observedReadId: displayed.receipt.observedReadId,
    observation: displayed.receipt.observation,
    note: 'Other task',
  };
  expect((await s.invoke('workspace_point', pointer)).success).toBe(false);
  expect((await s.invoke('workspace_observe', { surfaceId, selection })).success).toBe(false);
  const source = body(
    await s.invoke('workspace_observe', { surfaceId, selection, scope: 'source-preview' }),
  );
  pointer.observedReadId = source.receipt.observedReadId;
  expect(
    (await s.invoke('workspace_point', { ...pointer, observedReadId: undefined })).success,
  ).toBe(false);
  expect((await s.invoke('workspace_point', pointer)).success).toBe(false);
  await s.command({ kind: 'runFilter', surfaceId, filter: { query: '', status: null } });
  await s.ready(surfaceId);
  const visible = body(await s.invoke('workspace_observe', { surfaceId, selection }));
  pointer.observedReadId = visible.receipt.observedReadId;
  pointer.observation = visible.receipt.observation;
  const result = await s.invoke('workspace_point', pointer);
  expect(result.success).toBe(true);
  await s.command({ kind: 'runFilter', surfaceId, filter: { query: 'one', status: null } });
  await s.ready(surfaceId);
  const referenceId = body(result).referenceId;
  const before = await s.workspace.read('prj_one');
  expect(before.referenceReveal).toBeUndefined();
  const shown = await s.command({ kind: 'reveal', referenceId });
  const referenceLoad = await s.ready(surfaceId);
  expect(referenceLoad.observedReferenceView?.referenceId).toBe(referenceId);
  expect(shown.workspace.surfaces).toEqual(before.workspace.surfaces);
  expect((await s.workspace.captureShared('prj_one', surfaceId, () => {})).surface).toMatchObject({
    runView: { query: '' },
  });
  await expect(
    s.command({ kind: 'runFilter', surfaceId, filter: { query: 'two', status: null } }),
  ).rejects.toThrow('Return to your view');
  expect((await s.invoke('workspace_point', pointer)).success).toBe(false);
  await s.command({
    kind: 'returnReferenceView',
    referenceRequestId: shown.referenceReveal!.requestId,
  });
  await s.ready(surfaceId);
  expect((await s.workspace.read('prj_one')).workspace.surfaces).toEqual(before.workspace.surfaces);
  // A current-task search explicitly reads new data but never rewrites the old pointer.
  data.value.tasks[1]!.attempt = 3;
  data.value.engineRevision = 'new';
  await s.command({ kind: 'currentTask', referenceId });
  const current = await s.ready(surfaceId);
  expect(current.data.kind === 'run' && current.data.value.tasks[1]!.attempt).toBe(3);
  await expect(s.command({ kind: 'reveal', referenceId })).rejects.toThrow('older observation');
  const history = (await s.workspace.read('prj_one')).sharedReferences![0]!;
  expect(history.evidence.selection).toMatchObject({ attempt: 2 });
});

it('publishes dependency references only from a current displayed and returned scope', async () => {
  const s = await setup(),
    raw = dependencySnapshot();
  raw.projectId = 'prj_one';
  s.data.set(raw.runRef, {
    kind: 'run',
    value: presentRun(raw),
    dependencies: projectDependencies(presentDependencies(raw)),
  });
  const doc = await s.command({
    kind: 'open',
    resource: { kind: 'run', runRef: raw.runRef },
    pane: 'primary',
    duplicate: false,
  });
  const surfaceId = doc.workspace.surfaces[0]!.surfaceId;
  await s.ready(surfaceId);
  const source = body(
    await s.invoke('workspace_observe', {
      surfaceId,
      representation: 'dependencies',
      scope: 'source-preview',
    }),
  );
  expect(source.receipt.pointable).toBe(false);
  const sourceTarget = source.content.edges[0].target;
  const point = (read: typeof source, evidence = read.content.edges[0].target) =>
    s.invoke('workspace_point', {
      evidence,
      observedReadId: read.receipt.observedReadId,
      observation: read.receipt.observation,
      note: 'Review this directed dependency.',
    });
  expect(body(await point(source)).error.code).toBe('stale_revision');
  expect(
    body(await s.invoke('workspace_observe', { surfaceId, representation: 'dependencies' })).error
      .code,
  ).toBe('invalid_request');
  await s.command({ kind: 'runMode', surfaceId, mode: 'dependencies' });
  await s.ready(surfaceId);
  const read = body(await s.invoke('workspace_observe', { surfaceId }));
  expect(read.content.kind).toBe('run-dependencies');
  expect(read.content.edges[0].target).toEqual(sourceTarget);
  const before = await s.workspace.read('prj_one');
  await s.command({ kind: 'dependencyCamera', surfaceId, camera: { zoom: 1.5, x: 20, y: 10 } });
  expect(body(await point(read)).published).toBe(true);
  const marked = await s.workspace.read('prj_one');
  expect(marked.selections).toEqual(before.selections);
  expect(marked.chat).toEqual(before.chat);
  expect(marked.sharedReferences![0]!.evidence).toEqual(sourceTarget);
  expect(
    body(
      await point(read, {
        ...sourceTarget,
        selection: { ...sourceTarget.selection, fromTaskId: 'invented' },
      }),
    ).error.code,
  ).toBe('stale_revision');
  await s.command({ kind: 'runMode', surfaceId, mode: 'tasks' });
  await s.ready(surfaceId);
  expect(body(await point(read)).error.code).toBe('stale_revision');
  expect((await s.workspace.read('prj_one')).sharedReferences).toHaveLength(1);
});

it('Notebook tools bind current viewport receipts and preserve private User intent', async () => {
  const h = await setup();
  const text = 'visible first line\nhidden second line';
  h.data.set('res_notebook', {
    kind: 'file',
    value: {
      projectId: 'prj_one',
      resourceId: 'res_notebook',
      name: 'study.ipynb',
      revision,
      size: 100,
      content: {
        kind: 'notebook',
        document: {
          profile: 'notebook-passive-1',
          revision,
          bytes: 100,
          minor: 5,
          language: 'python',
          textBytes: text.length,
          cells: [
            {
              index: 0,
              address: { kind: 'id', id: 'cell' },
              type: 'code',
              source: { kind: 'text', text, digest: revision },
              outputs: [],
              notice: '',
            },
          ],
        },
      },
    },
  });
  const surface = await h.open('res_notebook'),
    load = await h.ready(surface.surfaceId);
  const selection: import('@gobble/contracts').NotebookSelection = {
    kind: 'notebook',
    profile: 'notebook-passive-1',
    cell: { kind: 'id', id: 'cell' },
    part: { kind: 'source' },
    selector: { kind: 'text', coordinateSpace: 'notebook-display-utf16', start: 0, end: 18 },
  };
  await h.workspace.notebookViewport({
    acknowledgment: load.acknowledgment,
    viewport: { parts: [selection] },
  });
  const observed = body(await h.invoke('workspace_observe', { surfaceId: surface.surfaceId }));
  const receipt = observed.receipt as {
    evidence: EvidenceRef;
    observedReadId: string;
    observation: typeof load.acknowledgment;
  };
  const prior = await h.workspace.read('prj_one');
  expect((await h.invoke('workspace_point', { ...receipt, note: 'This line' })).success).toBe(
    false,
  ); // closed inputs: receipt extras are not tool arguments
  expect(
    (
      await h.invoke('workspace_point', {
        evidence: receipt.evidence,
        observedReadId: receipt.observedReadId,
        observation: receipt.observation,
        note: 'This line',
      })
    ).success,
  ).toBe(true);
  const after = await h.workspace.read('prj_one');
  expect(after.selections).toEqual(prior.selections);
  expect(after.chat).toEqual(prior.chat);
  expect(after.workspace.surfaces).toEqual(prior.workspace.surfaces);
  await h.workspace.notebookViewport({
    acknowledgment: load.acknowledgment,
    viewport: { parts: [] },
  });
  expect(
    (
      await h.invoke('workspace_point', {
        evidence: receipt.evidence,
        observedReadId: receipt.observedReadId,
        observation: receipt.observation,
        note: 'Stale',
      })
    ).success,
  ).toBe(false);
  expect(
    (await h.invoke('workspace_observe', { surfaceId: surface.surfaceId, selection })).success,
  ).toBe(false);
  expect(
    (
      await h.invoke('workspace_observe', {
        surfaceId: surface.surfaceId,
        selection,
        scope: 'source-preview',
      })
    ).success,
  ).toBe(true);
  h.host.release(h.agent.projectId, h.agent.agentId, h.submission.requestId);
  expect(
    (
      await h.invoke('workspace_point', {
        evidence: receipt.evidence,
        observedReadId: receipt.observedReadId,
        observation: receipt.observation,
        note: 'Ended',
      })
    ).success,
  ).toBe(false);
});

it('pipeline Agent pointers commit exact subjects while preserving User selection and rejecting stale observations', async () => {
  const h = await setup();
  const flow = parse(
    PipelineFlowSchema,
    JSON.parse(readFileSync(new URL('./fixtures/pipeline-flow.json', import.meta.url), 'utf8')),
  );
  const value = {
    projectId: 'prj_one',
    pipelineId: 'pip_one',
    state: 'ready' as const,
    artifact: {
      artifactId: revision,
      sourceRevision: 'sha256:' + 'b'.repeat(64),
      checkedAt: '2026-09-09T00:00:00Z',
      flow,
    },
  };
  h.data.set('pip_one', { kind: 'pipeline', value });
  const opened = await h.command({
    kind: 'open',
    resource: { kind: 'pipeline', pipelineId: 'pip_one' },
    pane: 'primary',
    duplicate: false,
  });
  const surfaceId = opened.workspace.surfaces[0]!.surfaceId,
    load = await h.ready(surfaceId);
  const local = pipelineTarget(
    value,
    { kind: 'port', stepId: 'trim', direction: 'output', portName: 'trimmed' },
    surfaceId,
  );
  await h.command({
    kind: 'select',
    surfaceId,
    evidence: local,
    acknowledgment: load.acknowledgment,
  });
  const before = await h.workspace.read('prj_one');
  const observed = body(await h.invoke('workspace_observe', { surfaceId }));
  const target = observed.content.subjects.find(
    (s: { target: { selection: { subject: { kind: string; stepId?: string } } } }) =>
      s.target.selection.subject.kind === 'step' && s.target.selection.subject.stepId === 'quality',
  ).target;
  const args = {
    evidence: target,
    observedReadId: observed.receipt.observedReadId,
    observation: observed.receipt.observation,
    note: 'Compare this quality step.',
  };
  expect(body(await h.invoke('workspace_point', args))).toMatchObject({ published: true });
  const after = await h.workspace.read('prj_one');
  expect(after.selections).toEqual(before.selections);
  expect(after.chat).toEqual(before.chat);
  expect(after.activePane).toBe(before.activePane);
  expect(after.workspace.layout).toEqual(before.workspace.layout);
  expect(after.sharedReferences![0]!.label).toBe('Check read quality');
  expect(after.sharedReferences![0]!.evidence).toEqual(target);
  h.data.set('pip_one', {
    kind: 'pipeline',
    value: { ...value, artifact: { ...value.artifact, artifactId: 'sha256:' + 'c'.repeat(64) } },
  });
  await h.ready(surfaceId);
  expect(body(await h.invoke('workspace_point', args)).error.code).toBe('stale_revision');
  expect((await h.workspace.read('prj_one')).sharedReferences).toHaveLength(1);
});

it('rejects Agent opening of a report not retained in this Project', async () => {
  const s = await setup();
  const result = await s.invoke('workspace_open', {
    resources: [
      {
        kind: 'report',
        saved: {
          projectId: 'prj_one',
          runRef: 'run_one',
          capturedAt: 1,
          profile: 'fastqc-0.12.1-v1',
          title: 'Quality report',
          asset: { hash: 'sha256:' + 'a'.repeat(64), byteLength: 100 },
          producer: {
            schemaVersion: 1,
            runId: 'quality',
            snapshot: 'a'.repeat(32),
            originDigest: 'sha256:' + 'b'.repeat(64),
            instance: 'qc',
            attempt: 1,
            port: 'html',
            recipe: 'fastqc-v1',
            sha256: 'sha256:' + 'c'.repeat(64),
            size: 100,
          },
        },
      },
    ],
  });
  expect(body(result).error).toMatchObject({
    code: 'not_found',
    message: 'This saved report is not retained in this Project.',
  });
  expect((await s.workspace.read('prj_one')).workspace.surfaces).toEqual([]);
});
