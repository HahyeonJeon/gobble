import { toolSchemasV4 } from '../../contracts/src/shared-tools-v4';
import { mkdtemp, mkdir, readFile, writeFile, rm } from 'node:fs/promises';
import { join } from 'node:path';
import { tmpdir } from 'node:os';
import { afterEach, describe, expect, it, vi } from 'vitest';
import {
  EvidenceRefSchema,
  parse,
  readStoredWorkspace,
  logResource,
  type EvidenceRef,
  type SurfaceData,
  type WorkspaceAction,
  type RunLogs,
  type RunSnapshot,
} from '@gobble/contracts';
import { WorkspaceController } from '../src/main/workspace/controller';
import { WorkspaceStorage } from '../src/main/workspace/storage';
import { RenderSession } from '../src/main/workspace/render-session';
import { emptyWorkspace } from '../src/main/workspace/model';
import { checkSelection, observedDataRevision } from '../src/main/workspace/selection';
import { EvidenceStorage } from '../src/main/evidence/storage';
import { EvidenceService } from '../src/main/evidence/service';
import {
  EvidenceCapture,
  materializeObserved,
  capturedManifest,
} from '../src/main/evidence/capture';
import { evidencePreview } from '../src/main/evidence/materialize';
import { retainedEvidenceHashes } from '../src/main/evidence/references';
import { presentLog } from '../src/main/service/log-presentation';
import { presentRun } from '../src/main/service/run-presentation';
import { CollaborationCoordinator } from '../src/main/collaboration/coordinator';
import type { WorkspaceResources } from '../src/main/workspace/service';
import type { ConversationProvider, ProviderInput } from '../src/main/collaboration/provider';

const projectId = 'prj_observed';
const resource = logResource({ runRef: 'run_test', instanceId: 'align[S03]', attempt: 2 });
function logs(): RunLogs {
  return {
    projectId,
    runRef: 'run_test',
    engineRevision: 'checkpoint-one',
    observedAt: 1000,
    instance: 'align[S03]',
    attempt: 2,
    tailLimitBytes: 4096,
    logs: [
      {
        identity: 'align[S03]',
        stdout: 'private/native/path',
        stdout_tail: 'unselected stdout secret',
        stdout_size: 9000,
        stderr_tail: 'unselected prefix\nERROR 😀 exact\nunselected suffix',
        stderr_size: 70,
      },
    ],
  };
}
const logData = (): Extract<SurfaceData, { kind: 'log' }> => ({
  kind: 'log',
  value: presentLog(logs()),
});
function target(data: SurfaceData = logData()): Extract<EvidenceRef, { schemaVersion: 3 }> {
  return {
    schemaVersion: 3,
    projectId,
    resource,
    dataRevision: observedDataRevision(data),
    selection: {
      kind: 'log-text',
      coordinateSpace: 'decoded-preview-utf16-line-column',
      stream: 'stderr',
      start: { line: 2, column: 0 },
      end: { line: 2, column: 14 },
    },
  };
}
const attachment = (data = logData()) => ({
  attachmentId: 'att_capture',
  label: 'Align S03',
  createdAt: 2000,
  evidence: target(data),
});
const paths: string[] = [];
afterEach(async () => {
  for (const path of paths.splice(0)) await rm(path, { recursive: true, force: true });
});
async function temporary() {
  const path = await mkdtemp(join(tmpdir(), 'gobble-observed-'));
  paths.push(path);
  return path;
}

async function setup(quota?: number) {
  const path = await temporary();
  const storage = new EvidenceStorage(join(path, 'evidence'), quota);
  const state = new WorkspaceStorage(join(path, 'workspace'));
  const capture = new EvidenceCapture(storage);
  let data: SurfaceData = logData(),
    available = true;
  const resources: WorkspaceResources = {
    projects: async () => [{ projectId, rootResourceId: 'res_root', name: 'Observed' }],
    describe: async () => ({ title: 'Align S03', view: 'log' }),
    read: vi.fn(async () => {
      if (!available) throw new Error('Attempt no longer current');
      return structuredClone(data);
    }),
  };
  const workspace = new WorkspaceController(state, resources, () => true, capture);
  await workspace.initialize();
  const session = await workspace.connect();
  await workspace.openProject(projectId);
  const account = { require: () => 'session-one', supportsImages: () => false };
  const evidence = new EvidenceService(
    workspace,
    resources,
    storage,
    async () => {
      throw new Error('No images');
    },
    account,
  );
  const delivered: ProviderInput[][] = [];
  const provider: ConversationProvider = {
    bind: async () => 'thread-one',
    start: async (_s, input) => {
      delivered.push(input ?? []);
      return { id: 'turn-one', status: 'inProgress', clientIds: [], messages: [] };
    },
    interrupt: async () => {},
    read: async () => [],
    onEvent: () => () => {},
    onDisconnected: () => () => {},
    disconnect: async () => {},
  };
  const coordinator = new CollaborationCoordinator(
    workspace,
    provider,
    account,
    () => {},
    () => {},
    undefined,
    evidence,
  );
  await coordinator.configure({
    projectId,
    agentId: null,
    requestId: 'req_researcher',
    name: 'Researcher',
    configuration: { model: 'fixture', effort: 'low', instructions: '' },
  });
  const agentId = (await workspace.read(projectId)).workspace.agents[0]!.agentId;
  let sequence = 0;
  const command = async (action: WorkspaceAction) =>
    workspace.command({
      projectId,
      requestId: 'req_action_' + ++sequence,
      expectedRevision: (await workspace.read(projectId)).workspace.revision,
      action,
    });
  await command({ kind: 'recipient', agentId });
  const opened = await command({ kind: 'open', resource, pane: 'primary', duplicate: false });
  const surfaceId = opened.workspace.layout.primary.activeSurfaceId!;
  await workspace.present({
    projectId,
    rendererSessionId: session.rendererSessionId,
    visiblePanes: ['primary'],
  });
  const load = await workspace.loadSurface({
    projectId,
    surfaceId,
    rendererSessionId: session.rendererSessionId,
  });
  await workspace.acknowledge(load.acknowledgment);
  const ref = target(load.data);
  await command({ kind: 'select', surfaceId, evidence: ref, acknowledgment: load.acknowledgment });
  const attach = () =>
    command({ kind: 'attach', surfaceId, evidence: ref, acknowledgment: load.acknowledgment });
  const prepare = async () =>
    evidence.prepare({
      projectId,
      agentId,
      attachmentRevision: (await workspace.read(projectId)).chat.attachmentRevision ?? 0,
    });
  return {
    path,
    state,
    storage,
    capture,
    workspace,
    session,
    resources,
    evidence,
    coordinator,
    command,
    load,
    ref,
    attach,
    prepare,
    delivered,
    agentId,
    surfaceId,
    unavailable: () => {
      available = false;
    },
    change: (value: SurfaceData) => {
      data = value;
    },
  };
}

describe('current-attempt normalized observations', () => {
  it('keeps source bounds honest and removes native log paths', () => {
    const value = presentLog(logs());
    expect(value.streams.stdout).toMatchObject({
      sourceBytesReported: 9000,
      earlierBytesOmitted: true,
      completeness: 'unknown',
    });
    expect(value.streams.stderr.completeness).toBe('unknown');
    expect(JSON.stringify(value)).not.toContain('private/native/path');
    const empty = logs();
    empty.logs[0] = { identity: empty.instance, stderr_size: 0 };
    expect(presentLog(empty).streams.stderr).toMatchObject({
      text: '',
      availability: 'no-text-returned',
      completeness: 'unknown',
    });
  });
  it.each(['wrong-instance', 'duplicate', 'missing', 'bad-size', 'oversized'])(
    'rejects %s log metadata before rendering',
    (kind) => {
      const raw = logs();
      if (kind === 'wrong-instance') raw.logs[0]!.identity = 'align[S04]';
      if (kind === 'duplicate') raw.logs.push(raw.logs[0]!);
      if (kind === 'missing') raw.logs = [];
      if (kind === 'bad-size') raw.logs[0]!.stdout_size = -1;
      if (kind === 'oversized') raw.logs[0]!.stdout_tail = 'x'.repeat(4097);
      expect(() => presentLog(raw)).toThrow('invalid log metadata');
    },
  );
  it('bounds Run instances without merging identically named tasks or inventing attempts', () => {
    const run = { id: 'engine', status: 'custom-state' };
    const raw: RunSnapshot = {
      projectId,
      runRef: 'run_test',
      engineRevision: 'checkpoint-one',
      observedAt: 1000,
      availability: 'available',
      runtimeBinding: {
        endpoint: 'unix:///fixture',
        daemonId: 'fixture',
        imageId: 'sha256:' + 'a'.repeat(64),
        platform: 'linux/amd64',
        projectPath: '/gobble/project',
        workspacePath: '/gobble/project/run',
      },
      snapshot: {
        schema_version: 2,
        snapshot: 'checkpoint-one',
        run,
        tasks: Array.from({ length: 1001 }, (_, index) => ({
          identity: 'align[' + index + ']',
          attempt: index ? 2 : 0,
          task_id: 'align',
          name: 'Align',
          template: index === 0,
        })),
        logs: [],
      },
    };
    const value = presentRun(raw);
    expect(value.preview).toEqual({ availableTasks: 1001, returnedTasks: 1000, truncated: true });
    expect(value.status).toBe('custom-state');
    expect(value.tasks[0]).toMatchObject({
      attempt: 0,
      template: true,
      taskId: 'align',
      instanceId: 'align[0]',
    });
    raw.snapshot.tasks[1]!.identity = raw.snapshot.tasks[0]!.identity;
    expect(() => presentRun(raw)).toThrow('invalid task identities');
  });
});

describe('exact target and captured content', () => {
  it('binds stream, revision and Unicode range while excluding unselected log text', () => {
    const data = logData(),
      ref = target(data);
    expect(parse(EvidenceRefSchema, ref)).toEqual(ref);
    expect(() => checkSelection(ref, data)).not.toThrow();
    const asset = materializeObserved(attachment(data), data);
    expect(evidencePreview(asset)).toMatchObject({
      kind: 'log',
      text: 'ERROR 😀 exact',
      observation: { target: ref, source: { selectedStream: 'stderr' } },
    });
    const encoded = asset.bytes.toString();
    for (const secret of [
      'unselected stdout secret',
      'unselected prefix',
      'unselected suffix',
      'private/native/path',
    ])
      expect(encoded).not.toContain(secret);
    expect(() => checkSelection({ ...ref, resource: { ...resource, attempt: 3 } }, data)).toThrow();
    const split = {
      ...ref,
      selection: {
        ...ref.selection!,
        kind: 'log-text' as const,
        coordinateSpace: 'decoded-preview-utf16-line-column' as const,
        stream: 'stderr' as const,
        start: { line: 2, column: 7 },
        end: { line: 2, column: 8 },
      },
    };
    expect(() => checkSelection(split, data)).toThrow();
    expect(() => parse(toolSchemasV4.workspace_point, { evidence: ref, note: '' })).toThrow();
  });
  it('uses observation facts rather than timestamps or engine checkpoint alone as v3 identity', () => {
    const one = logData(),
      two = structuredClone(one);
    two.value.observedAt++;
    expect(observedDataRevision(one)).toBe(observedDataRevision(two));
    if (!('streams' in two.value)) throw new Error('Expected projection');
    two.value.streams.stderr.text = 'different text';
    expect(observedDataRevision(one)).not.toBe(observedDataRevision(two));
    expect(() => checkSelection(target(one), two)).toThrow();
  });
  it('rejects altered target, capture time and preview text even with untouched asset bytes', () => {
    const asset = materializeObserved(attachment(), logData());
    const changed = structuredClone(asset.manifest);
    changed.evidence.resource = { ...resource, attempt: 8 };
    expect(() => evidencePreview({ ...asset, manifest: changed })).toThrow();
    const later = structuredClone(asset.manifest);
    later.capturedAt++;
    expect(() => evidencePreview({ ...asset, manifest: later })).toThrow();
    const content = JSON.parse(asset.bytes.toString());
    content.text = 'invented result';
    expect(() =>
      evidencePreview({ ...asset, bytes: Buffer.from(JSON.stringify(content)) }),
    ).toThrow();
  });
  it('captures one task observation without adjacent task facts', () => {
    const data: SurfaceData = {
      kind: 'run',
      value: {
        projectId,
        runRef: 'run_test',
        engineRevision: 'checkpoint',
        observedAt: 1000,
        runId: 'engine',
        imageId: 'image',
        pipelineName: 'Pipeline',
        status: 'failed',
        tasks: [
          {
            instanceId: 'align[S03]',
            taskId: 'align',
            name: 'Align',
            status: 'failed',
            reason: 'Input error',
            attempt: 2,
          },
          {
            instanceId: 'align[S04]',
            taskId: 'align',
            name: 'Other secret',
            status: 'succeeded',
            reason: null,
            attempt: 1,
          },
        ],
        dependencies: [],
      },
    };
    const ref: EvidenceRef = {
      schemaVersion: 3,
      projectId,
      resource: { kind: 'run', runRef: 'run_test' },
      dataRevision: observedDataRevision(data),
      selection: {
        kind: 'run-task',
        coordinateSpace: 'observed-instance-attempt',
        instanceId: 'align[S03]',
        attempt: 2,
      },
    };
    const asset = materializeObserved({ ...attachment(), evidence: ref }, data);
    expect(asset.bytes.toString()).not.toContain('Other secret');
    expect(evidencePreview(asset)).toMatchObject({
      kind: 'run',
      observation: {
        source: {
          observedTaskCount: 2,
          value: { tasks: [{ instanceId: 'align[S03]', attempt: 2 }] },
        },
      },
    });
    expect(() =>
      checkSelection(
        {
          ...ref,
          selection: {
            ...ref.selection!,
            kind: 'run-task',
            coordinateSpace: 'observed-instance-attempt',
            instanceId: 'align[S03]',
            attempt: 1,
          },
        },
        data,
      ),
    ).toThrow();
  });
});

describe('durable observed drafts', () => {
  it('freezes at attach and sends the original excerpt after its attempt becomes unavailable', async () => {
    const s = await setup();
    s.unavailable();
    const saved = await s.attach();
    const draft = saved.chat.attachments![0]!;
    expect(draft.capture).toBeDefined();
    const reads = vi.mocked(s.resources.read).mock.calls.length;
    const prepared = await s.prepare();
    await s.workspace.updateDraft(projectId, 'Explain this observation');
    await s.coordinator.send({
      projectId,
      agentId: s.agentId,
      requestId: 'req_send',
      text: 'Explain this observation',
      preparedEvidenceId: prepared.preparedId,
    });
    expect(s.delivered).toHaveLength(1);
    const sentText = JSON.stringify(s.delivered);
    expect(sentText).toContain('ERROR 😀 exact');
    expect(sentText).not.toContain('unselected stdout secret');
    expect(vi.mocked(s.resources.read).mock.calls.length).toBe(reads);
    const document = await s.workspace.read(projectId);
    expect(document.collaboration!.submissions[0]!.evidence![0]!.asset.hash).toBe(
      draft.capture!.asset.hash,
    );
  });
  it('restores a frozen draft across controller restart without reading the current Run', async () => {
    const s = await setup();
    const before = await s.attach();
    const draft = before.chat.attachments![0]!;
    await s.workspace.stop();
    s.unavailable();
    const controller = new WorkspaceController(
      new WorkspaceStorage(join(s.path, 'workspace')),
      s.resources,
      () => true,
      s.capture,
    );
    await controller.initialize();
    await controller.connect();
    await controller.openProject(projectId);
    const after = await controller.read(projectId);
    expect(after.chat.attachments).toEqual(before.chat.attachments);
    const account = { require: () => 'session-one', supportsImages: () => false };
    const evidence = new EvidenceService(
      controller,
      s.resources,
      s.storage,
      async () => {
        throw new Error();
      },
      account,
    );
    const prepared = await evidence.prepare({
      projectId,
      agentId: s.agentId,
      attachmentRevision: after.chat.attachmentRevision!,
    });
    expect(prepared.items[0]!.asset.hash).toBe(draft.capture!.asset.hash);
    await expect(controller.captureShared(projectId, s.surfaceId, () => {})).rejects.toThrow();
  });
  it('fails closed for quota, stale receipt, cross-Project capture and missing saved content', async () => {
    const limited = await setup(64);
    await expect(limited.attach()).rejects.toThrow('quota');
    expect((await limited.workspace.read(projectId)).chat.attachments ?? []).toEqual([]);
    const s = await setup();
    await expect(
      s.command({
        kind: 'attach',
        surfaceId: s.surfaceId,
        evidence: s.ref,
        acknowledgment: { ...s.load.acknowledgment, generation: 999 },
      }),
    ).rejects.toThrow();
    await expect(
      s.command({
        kind: 'attach',
        surfaceId: s.surfaceId,
        evidence: { ...s.ref, projectId: 'prj_other' },
        acknowledgment: s.load.acknowledgment,
      }),
    ).rejects.toThrow();
    const saved = await s.attach(),
      draft = saved.chat.attachments![0]!;
    await rm(join(s.path, 'evidence', projectId, draft.capture!.asset.hash.slice(7) + '.blob'));
    s.unavailable();
    await expect(s.prepare()).rejects.toThrow('missing or corrupt');
  });
  it('does not attach if the window disconnects during asset publication', async () => {
    const s = await setup();
    const write = s.storage.putCaptured.bind(s.storage);
    vi.spyOn(s.storage, 'putCaptured').mockImplementation(async (project, asset) => {
      await write(project, asset);
      s.workspace.disconnect();
    });
    await expect(s.attach()).rejects.toThrow();
    expect((await s.workspace.read(projectId)).chat.attachments ?? []).toEqual([]);
  });
  it('keeps the draft unchanged when durable workspace commit fails after publishing the asset', async () => {
    const s = await setup();
    const before = await s.workspace.read(projectId);
    vi.spyOn(s.state.project(projectId), 'write').mockRejectedValueOnce(
      new Error('Disk save failed'),
    );
    await expect(s.attach()).rejects.toThrow('Disk save failed');
    expect(await s.workspace.read(projectId)).toEqual(before);
    const persisted = await new WorkspaceStorage(join(s.path, 'workspace'))
      .project(projectId)
      .read();
    expect(persisted).toEqual(before);
  });
  it('reclaims only tracked unreferenced captures and preserves all recovery roots', async () => {
    const s = await setup();
    const saved = await s.attach(),
      draft = saved.chat.attachments![0]!;
    const blob = join(s.path, 'evidence', projectId, draft.capture!.asset.hash.slice(7) + '.blob');
    await s.command({ kind: 'detach', attachmentId: draft.attachmentId });
    expect((await readFile(blob)).length).toBeGreaterThan(0); // previous workspace backup still refers to it
    await s.workspace.updateDraft(projectId, 'advance backup');
    await expect(readFile(blob)).rejects.toMatchObject({ code: 'ENOENT' });
    const asset = materializeObserved(attachment(), logData());
    await s.storage.put(projectId, [asset]); // pre-existing untracked history is never deleted
    await s.storage.reclaimCaptured(projectId, new Set());
    expect(
      await readFile(
        join(s.path, 'evidence', projectId, asset.manifest.asset.hash.slice(7) + '.blob'),
      ),
    ).toEqual(asset.bytes);
  });
  it('does not rescan recovery roots on each ordinary draft keystroke', async () => {
    const s = await setup();
    await s.attach();
    await s.workspace.updateDraft(projectId, 'advance backup once');
    const reads = vi.spyOn(s.state, 'retainedDocuments');
    await s.workspace.updateDraft(projectId, 'first edit');
    await s.workspace.updateDraft(projectId, 'second edit');
    expect(reads).not.toHaveBeenCalled();
  });
  it('preserves orphan captures when a recovery document has an unknown version', async () => {
    const s = await setup();
    const asset = materializeObserved(attachment(), logData());
    await s.storage.putCaptured(projectId, asset);
    const foreignBackup = join(s.path, 'workspace/projects', projectId + '.json.v99.backup');
    await writeFile(foreignBackup, JSON.stringify({ schemaVersion: 99 }));
    await s.attach(); // capture admission attempts reclamation, but this root forbids deletion
    await s.workspace.updateDraft(projectId, 'save safely');
    expect(
      await readFile(
        join(s.path, 'evidence', projectId, asset.manifest.asset.hash.slice(7) + '.blob'),
      ),
    ).toEqual(asset.bytes);
    await expect(s.state.retainedDocuments(projectId)).rejects.toThrow();
  });
});

describe('bounded render ownership and migration', () => {
  it('isolates retained loads from caller mutation, hides and late completions, with an explicit memory limit', () => {
    const rendering = new RenderSession();
    const surface = {
      projectId,
      surfaceId: 'srf_log',
      resource,
      view: 'log' as const,
      pinned: false,
      openedBy: { kind: 'user' as const },
    };
    rendering.reveal(['srf_log']);
    const input = { projectId, surfaceId: 'srf_log', rendererSessionId: rendering.id };
    const first = rendering.begin(input, surface),
      second = rendering.begin(input, surface);
    expect(() => rendering.complete(first, logData())).toThrow('newer view');
    const load = rendering.complete(second, logData());
    rendering.acknowledge(load.acknowledgment);
    if (load.data.kind !== 'log') throw new Error();
    load.data.value.instance = 'forged';
    expect(rendering.snapshot('srf_log').data).toMatchObject({ value: { instance: 'align[S03]' } });
    const oversized = logData();
    if (!('streams' in oversized.value)) throw new Error();
    oversized.value.streams.stdout.text = 'x'.repeat(1024 * 1024);
    expect(() => rendering.complete(rendering.begin(input, surface), oversized)).toThrow(
      'memory limit',
    );
    rendering.reveal([]);
    expect(() => rendering.selectableData('srf_log')).toThrow();
    rendering.release(first);
    rendering.release(second);
  });
  it('backs up exact v6 bytes and rejects new reference/capture shapes nested inside old versions', async () => {
    const path = await temporary();
    await mkdir(join(path, 'projects'));
    const old = { ...emptyWorkspace(projectId), schemaVersion: 6 };
    const raw = JSON.stringify(old, null, 2) + '\n';
    const file = join(path, 'projects', projectId + '.json');
    await writeFile(file, raw);
    const store = new WorkspaceStorage(path).project(projectId),
      next = (await store.read())!;
    expect(next.schemaVersion).toBe(21);
    expect(await readFile(file, 'utf8')).toBe(raw);
    await store.write(next);
    expect(await readFile(file + '.v6.backup', 'utf8')).toBe(raw);
    const asset = materializeObserved(attachment(), logData());
    const invalid = {
      ...old,
      chat: { ...old.chat, attachments: [{ ...attachment(), capture: asset.manifest.capture }] },
    };
    expect(() => readStoredWorkspace(invalid)).toThrow();
    expect(() => readStoredWorkspace({ ...invalid, schemaVersion: 5 })).toThrow();
    expect(() => readStoredWorkspace({ ...old, schemaVersion: 999 })).toThrow();
    expect(
      retainedEvidenceHashes([
        {
          ...next,
          chat: {
            ...next.chat,
            attachments: [{ ...attachment(), capture: asset.manifest.capture! }],
          },
        },
      ]),
    ).toEqual(new Set([asset.manifest.asset.hash]));
    expect(capturedManifest({ ...attachment(), capture: asset.manifest.capture! }).asset.hash).toBe(
      asset.manifest.asset.hash,
    );
  });
});

describe('task and stream navigation ownership', () => {
  it('changes streams without rereading, rejects inactive targets and superseded acknowledgments', async () => {
    const s = await setup();
    const request = {
      projectId,
      surfaceId: s.surfaceId,
      rendererSessionId: s.session.rendererSessionId,
    };
    const reads = vi.mocked(s.resources.read).mock.calls.length;
    await s.command({ kind: 'logStream', surfaceId: s.surfaceId, stream: 'stdout' });
    await expect(s.attach()).rejects.toThrow();
    const stdout = await s.workspace.loadSurface(request);
    await s.workspace.acknowledge(stdout.acknowledgment);
    expect(stdout.observedRevision).toBe(s.load.observedRevision);
    expect(stdout.acknowledgment.presentation.view).toBe(1);
    await expect(
      s.command({
        kind: 'attach',
        surfaceId: s.surfaceId,
        evidence: s.ref,
        acknowledgment: stdout.acknowledgment,
      }),
    ).rejects.toThrow('Reveal this task or stream');
    expect((await s.workspace.read(projectId)).selections[0]!.evidence).toEqual(s.ref);
    await s.command({ kind: 'logStream', surfaceId: s.surfaceId, stream: 'stderr' });
    const stderr = await s.workspace.loadSurface(request);
    await s.workspace.acknowledge(stderr.acknowledgment);
    const saved = await s.command({
      kind: 'attach',
      surfaceId: s.surfaceId,
      evidence: s.ref,
      acknowledgment: stderr.acknowledgment,
    });
    expect(saved.chat.attachments).toHaveLength(1);
    expect(vi.mocked(s.resources.read).mock.calls.length).toBe(reads);
  });
  it('retains failed observations across navigation; a successful refresh clears only stale local selection', async () => {
    const s = await setup();
    const saved = await s.attach();
    const request = {
      projectId,
      surfaceId: s.surfaceId,
      rendererSessionId: s.session.rendererSessionId,
    };
    vi.mocked(s.resources.read).mockRejectedValueOnce(new Error('unavailable'));
    const failed = await s.workspace.loadSurface({ ...request, refresh: true });
    expect(failed.data).toEqual(s.load.data);
    expect(failed.refreshProblem).toContain('could not be refreshed');
    await s.workspace.acknowledge(failed.acknowledgment);
    await s.command({ kind: 'logStream', surfaceId: s.surfaceId, stream: 'stdout' });
    const switched = await s.workspace.loadSurface(request);
    expect(switched.refreshProblem).toBe(failed.refreshProblem);
    await s.workspace.acknowledge(switched.acknowledgment);
    const changed = logData();
    if (!('streams' in changed.value)) throw new Error();
    changed.value.streams.stderr.text = 'replacement';
    s.change(changed);
    const refreshed = await s.workspace.loadSurface({ ...request, refresh: true });
    expect(refreshed.refreshProblem).toBeUndefined();
    expect(refreshed.observedRevision).not.toBe(failed.observedRevision);
    await s.workspace.acknowledge(refreshed.acknowledgment);
    const document = await s.workspace.read(projectId);
    expect(document.selections).toEqual([]);
    expect(document.chat.attachments).toEqual(saved.chat.attachments);
    expect((await s.prepare()).items[0]!.asset.hash).toBe(
      saved.chat.attachments![0]!.capture!.asset.hash,
    );
  });
  it('retains data but rejects an in-flight completion after its presentation changes', () => {
    const rendering = new RenderSession();
    const surface = {
      projectId,
      surfaceId: 'srf_log',
      resource,
      view: 'log' as const,
      pinned: false,
      openedBy: { kind: 'user' as const },
      logView: { stream: 'stderr' as const, viewRevision: 0 },
    };
    rendering.reveal([surface.surfaceId]);
    const request = { projectId, surfaceId: surface.surfaceId, rendererSessionId: rendering.id };
    const initial = rendering.begin(request, surface);
    rendering.complete(initial, logData());
    rendering.release(initial);
    const pending = rendering.begin({ ...request, refresh: true }, surface);
    const changed = { ...surface, logView: { stream: 'stdout' as const, viewRevision: 1 } };
    rendering.reconcile([changed], []);
    expect(() => rendering.complete(pending, logData())).toThrow('newer view');
    rendering.release(pending);
    const next = rendering.begin(request, changed);
    expect(rendering.retainedData(next)).toEqual(logData());
    rendering.complete(next, logData());
    rendering.release(next);
  });
  it('preserves exact v7 captured drafts and old combined log state during v8 migration', async () => {
    const s = await setup();
    const saved = await s.attach();
    const old = { ...structuredClone(saved), schemaVersion: 7 };
    old.schemaVersion = 7;
    const logSurface = old.workspace.surfaces[0]!;
    if (logSurface.view !== 'log') throw new Error();
    delete logSurface.logView;
    const raw = JSON.stringify(old, null, 2) + '\n';
    const path = await temporary();
    await mkdir(join(path, 'projects'));
    const file = join(path, 'projects', projectId + '.json');
    await writeFile(file, raw);
    const store = new WorkspaceStorage(path).project(projectId);
    const migrated = (await store.read())!;
    expect(migrated.schemaVersion).toBe(21);
    expect(migrated.chat.attachments).toEqual(saved.chat.attachments);
    expect(migrated.workspace.surfaces[0]).not.toHaveProperty('logView');
    await store.write(migrated);
    expect(await readFile(file + '.v7.backup', 'utf8')).toBe(raw);
    logSurface.logView = { stream: 'stdout', viewRevision: 0 };
    expect(() => readStoredWorkspace(old)).toThrow();
  });
});

it('keeps the last bounded observation when refreshed content exceeds render admission limits', async () => {
  const s = await setup();
  const enlarged = logData();
  if (!('streams' in enlarged.value)) throw new Error();
  enlarged.value.streams.stdout.text = 'x'.repeat(1024 * 1024);
  s.change(enlarged);
  const load = await s.workspace.loadSurface({
    projectId,
    surfaceId: s.surfaceId,
    rendererSessionId: s.session.rendererSessionId,
    refresh: true,
  });
  expect(load.data).toEqual(s.load.data);
  expect(load.refreshProblem).toContain('memory limit');
  await s.workspace.acknowledge(load.acknowledgment);
  expect((await s.workspace.read(projectId)).selections[0]!.evidence).toEqual(s.ref);
});

it('keeps identical text from different streams as distinct addressed captures', () => {
  const data = logData();
  if (!('streams' in data.value)) throw new Error();
  for (const stream of ['stdout', 'stderr'] as const) {
    data.value.streams[stream].text = 'Shared line';
    data.value.streams[stream].decodedUtf8Bytes = 11;
  }
  const ref = target(data);
  const captures = (['stdout', 'stderr'] as const).map((stream) =>
    materializeObserved(
      {
        ...attachment(data),
        evidence: {
          ...ref,
          selection: {
            kind: 'log-text',
            coordinateSpace: 'decoded-preview-utf16-line-column',
            stream,
            start: { line: 1, column: 0 },
            end: { line: 1, column: 11 },
          },
        },
      },
      data,
    ),
  );
  expect(captures[0]!.manifest.asset.hash).not.toBe(captures[1]!.manifest.asset.hash);
  for (const [index, stream] of ['stdout', 'stderr'].entries())
    expect(evidencePreview(captures[index]!)).toMatchObject({
      kind: 'log',
      text: 'Shared line',
      observation: { source: { selectedStream: stream } },
    });
});
