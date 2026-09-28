import { mkdtemp, rm, writeFile, readdir, readFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { createHash } from 'node:crypto';
import { afterEach, expect, it, vi } from 'vitest';
import {
  parseWorkspaceDocument,
  readStoredWorkspace,
  type FastqcContent,
  type RunReportSource,
  type WorkspaceAction,
} from '@gobble/contracts';
import { ReportEvidence } from '../src/main/evidence/report';
import { EvidenceStorage } from '../src/main/evidence/storage';
import { EvidenceCapture } from '../src/main/evidence/capture';
import { retainedEvidenceHashes } from '../src/main/evidence/references';
import { WorkspaceStorage } from '../src/main/workspace/storage';
import { WorkspaceController } from '../src/main/workspace/controller';
import { emptyWorkspace } from '../src/main/workspace/model';
import type { WorkspaceResources } from '../src/main/workspace/service';
import { ProjectService } from '../src/main/service/project-service';
import { textObservation } from '../src/main/shared-context/observation';

const directories: string[] = [];
afterEach(async () => {
  vi.restoreAllMocks();
  await Promise.all(directories.splice(0).map((p) => rm(p, { recursive: true, force: true })));
});
const content: FastqcContent = {
  profile: 'fastqc-0.12.1-v1',
  title: 'reads.fq FastQC Report',
  logoLabel: 'FastQC',
  headerTitle: 'FastQC Report',
  headerFilename: 'Today\nreads.fq',
  summaryHeading: 'Summary',
  summary: [{ moduleId: 'M0', title: 'Basic Statistics', status: '[PASS]' }],
  modules: [
    {
      id: 'M0',
      title: 'Basic Statistics',
      status: '[OK]',
      blocks: [
        { kind: 'table', headers: ['Measure', 'Value'], rows: [['Total Sequences', '100']] },
      ],
    },
  ],
  footer: 'Produced by FastQC  (version 0.12.1)',
};
const bytes = Buffer.from('<verified source>');
const source: RunReportSource = {
  projectId: 'prj_reports',
  runRef: 'run_reads',
  observedAt: 1,
  evidence: {
    schemaVersion: 1,
    runId: 'quality',
    snapshot: 'a'.repeat(32),
    originDigest: 'sha256:' + 'b'.repeat(64),
    instance: 'fastqc',
    attempt: 1,
    port: 'html',
    recipe: 'fastqc-v1',
    path: 'work/fastqc/report.html',
    size: bytes.length,
    sha256: 'sha256:' + createHash('sha256').update(bytes).digest('hex'),
  },
  base64: bytes.toString('base64'),
};
async function setup() {
  const base = await mkdtemp(join(tmpdir(), 'gobble-saved-report-'));
  directories.push(base);
  const blobs = new EvidenceStorage(join(base, 'evidence'));
  const decode = vi.fn(async () => structuredClone(content));
  const reports = new ReportEvidence(blobs, decode);
  const state = new WorkspaceStorage(base);
  const capture = vi.fn(async () => reports.capture(source));
  const resources: WorkspaceResources = {
    projects: async () => [
      { projectId: source.projectId, name: 'Reports', rootResourceId: 'res_root' },
    ],
    captureReport: capture,
    read: async (project, resource) => {
      if (resource.kind !== 'report') throw new Error('Unexpected resource');
      return reports.read(project, resource.saved);
    },
    describe: async (project, resource) => {
      if (resource.kind !== 'report') throw new Error('Unexpected resource');
      await reports.read(project, resource.saved);
      return { view: 'report', title: resource.saved.title };
    },
  };
  let controller = new WorkspaceController(
    state,
    resources,
    () => true,
    new EvidenceCapture(blobs),
  );
  await controller.initialize();
  await controller.connect();
  await controller.openProject(source.projectId);
  let counter = 0;
  const command = async (action: WorkspaceAction) =>
    controller.command({
      projectId: source.projectId,
      expectedRevision: (await controller.read(source.projectId)).workspace.revision,
      requestId: 'req_report_' + ++counter,
      action,
    });
  const restart = async () => {
    controller = new WorkspaceController(
      new WorkspaceStorage(base),
      resources,
      () => true,
      new EvidenceCapture(blobs),
    );
    await controller.initialize();
    await controller.connect();
    await controller.openProject(source.projectId);
    return controller;
  };
  return { base, blobs, reports, decode, resources, capture, controller, command, restart };
}
const open = {
  kind: 'openReport',
  runRef: source.runRef,
  instance: source.evidence.instance,
  attempt: 1,
} as const;
it('retains exact captured content after close, restart and loss of the original source; rejects corrupt saved bytes without replacement', async () => {
  const s = await setup();
  let doc = await s.command(open);
  const saved = doc.savedReports![0]!;
  expect(doc.workspace.layout.kind).toBe('split');
  expect(doc.workspace.surfaces[0]!.view).toBe('report');
  expect(retainedEvidenceHashes([doc])).toEqual(new Set([saved.asset.hash]));
  expect(saved.producer).not.toHaveProperty('path');
  await s.command({
    kind: 'duplicateView',
    surfaceId: doc.workspace.surfaces[0]!.surfaceId,
    pane: 'primary',
  });
  doc = await s.controller.read(source.projectId);
  for (const view of doc.workspace.surfaces)
    await s.command({ kind: 'close', surfaceId: view.surfaceId });
  s.capture.mockRejectedValue(new Error('Source was deleted; it must never be read again'));
  const restarted = await s.restart();
  doc = await s.command(open);
  expect(doc.savedReports).toEqual([saved]);
  expect(s.capture).toHaveBeenCalledTimes(1);
  expect((await s.reports.read(source.projectId, saved)).value.content).toEqual(content);
  expect((await restarted.read(source.projectId)).workspace.surfaces).toHaveLength(1);
  const path = join(s.base, 'evidence', source.projectId, saved.asset.hash.slice(7) + '.blob');
  await writeFile(path, 'corrupt');
  await expect(s.reports.read(source.projectId, saved)).rejects.toThrow('missing or corrupt');
  await expect(s.command(open)).rejects.toThrow('missing or corrupt');
  expect(s.capture).toHaveBeenCalledTimes(1);
});
it('rejects mismatched source identity, foreign saved records, quota overflow and complete text overflow before publication', async () => {
  const s = await setup();
  await expect(
    s.reports.capture({ ...source, base64: Buffer.from('different').toString('base64') }),
  ).rejects.toThrow('recorded output');
  expect(s.decode).not.toHaveBeenCalled();
  const saved = await s.reports.capture(source);
  await expect(s.reports.read('prj_other', saved)).rejects.toThrow('another Project');
  await expect(
    s.command({
      kind: 'open',
      resource: { kind: 'report', saved },
      pane: 'secondary',
      duplicate: false,
    }),
  ).rejects.toThrow('not retained');
  const limited = new ReportEvidence(new EvidenceStorage(join(s.base, 'tiny'), 1), s.decode);
  await expect(limited.capture(source)).rejects.toThrow('quota');
  s.decode.mockResolvedValue({ ...content, footer: 'x'.repeat(64 * 1024) });
  await expect(s.reports.capture(source)).rejects.toThrow('limits');
});
it('failed publication is not retained and its captured orphan is reclaimed on restart, while valid saved records survive', async () => {
  const s = await setup();
  const original = s.resources.describe;
  s.resources.describe = async () => {
    throw new Error('Publication cannot proceed');
  };
  await expect(s.command(open)).rejects.toThrow('Publication cannot proceed');
  expect((await s.controller.read(source.projectId)).savedReports).toBeUndefined();
  const directory = join(s.base, 'evidence', source.projectId);
  expect((await readdir(directory)).some((n) => n.endsWith('.blob'))).toBe(true);
  s.resources.describe = original;
  await s.command({ kind: 'chat', collapsed: true });
  expect((await readdir(directory)).filter((n) => n.endsWith('.blob'))).toEqual([]);
  const saved = (await s.command(open)).savedReports![0]!;
  await s.restart();
  await expect(s.reports.read(source.projectId, saved)).resolves.toMatchObject({ kind: 'report' });
});
it('report Views reject unrelated targets and generic text observations', async () => {
  const s = await setup();
  const doc = await s.command(open);
  const view = doc.workspace.surfaces[0]!;
  for (const kind of ['select', 'attach', 'share'] as const) {
    // Supplying another valid target must not turn a Report View into a sharing authority.
    const evidence = {
      schemaVersion: 2 as const,
      projectId: source.projectId,
      resource: { kind: 'run' as const, runRef: source.runRef },
      dataRevision: 'revision',
    };
    await expect(
      s.command({
        kind,
        surfaceId: view.surfaceId,
        evidence,
        ...(kind === 'share' ? { note: '' } : {}),
      } as WorkspaceAction),
    ).rejects.toThrow(kind === 'select' ? 'whole-report' : 'another Project view');
  }
  expect(() =>
    textObservation({
      kind: 'report',
      saved: doc.savedReports![0]!,
      value: {
        schemaVersion: 1,
        projectId: source.projectId,
        runRef: source.runRef,
        producer: doc.savedReports![0]!.producer,
        capturedAt: doc.savedReports![0]!.capturedAt,
        content,
      },
    }),
  ).toThrow('not available');
});
it('v19 migration preserves history and freezes backup before publishing saved reports', async () => {
  const s = await setup();
  const old = { ...emptyWorkspace(source.projectId), schemaVersion: 19 };
  expect(readStoredWorkspace(old)).toEqual({ ...old, schemaVersion: 21 });
  const path = join(s.base, 'projects', source.projectId + '.json');
  await writeFile(path, JSON.stringify(old));
  const controller = await s.restart();
  expect((await controller.read(source.projectId)).schemaVersion).toBe(21);
  expect(JSON.parse(await readFile(path + '.v19.backup', 'utf8'))).toEqual(old);
  const doc = await s.command(open);
  const altered = structuredClone(doc);
  altered.savedReports = [];
  expect(() => parseWorkspaceDocument(altered)).toThrow('retained');
});
it('native report route verifies exact project, Run, instance and attempt associations', async () => {
  const request = vi.fn(async () => ({ schemaVersion: 1, ok: true, value: source }));
  const service = new ProjectService({ request });
  await expect(
    service.readReport({
      projectId: source.projectId,
      runRef: source.runRef,
      instance: source.evidence.instance,
      attempt: 1,
    }),
  ).resolves.toEqual(source);
  expect(request).toHaveBeenCalledWith(
    '/v1/projects/prj_reports/runs/run_reads/report?instance=fastqc&attempt=1',
  );
  request.mockResolvedValue({
    schemaVersion: 1,
    ok: true,
    value: { ...source, evidence: { ...source.evidence, attempt: 2 } },
  });
  await expect(
    service.readReport({
      projectId: source.projectId,
      runRef: source.runRef,
      instance: source.evidence.instance,
      attempt: 1,
    }),
  ).rejects.toThrow('another report attempt');
});
it('report navigation validates the displayed saved modules and persists only View state', async () => {
  const s = await setup();
  const doc = await s.command(open);
  const surface = doc.workspace.surfaces[0]!;
  const session = await s.controller.connect();
  await s.controller.present({
    projectId: source.projectId,
    rendererSessionId: session.rendererSessionId,
    visiblePanes: ['primary', 'secondary'],
  });
  const load = await s.controller.loadSurface({
    projectId: source.projectId,
    rendererSessionId: session.rendererSessionId,
    surfaceId: surface.surfaceId,
  });
  await s.controller.acknowledge(load.acknowledgment);
  await expect(
    s.command({
      kind: 'reportNavigate',
      surfaceId: surface.surfaceId,
      moduleId: 'M404',
      acknowledgment: load.acknowledgment,
    }),
  ).rejects.toThrow('outside');
  const navigated = await s.command({
    kind: 'reportNavigate',
    surfaceId: surface.surfaceId,
    moduleId: 'M0',
    acknowledgment: load.acknowledgment,
  });
  expect(navigated.workspace.surfaces[0]).toMatchObject({ reportModuleId: 'M0' });
  expect(navigated.savedReports).toEqual(doc.savedReports);
  const restarted = await s.restart();
  expect((await restarted.read(source.projectId)).workspace.surfaces[0]).toMatchObject({
    reportModuleId: 'M0',
  });
});

it('whole-report attachment and pointers retain exact identity, and a new presentation invalidates Agent pointing authority', async () => {
  const { reportTarget, SHARED_TOOLSET } = await import('@gobble/contracts');
  const { SharedContextHost } = await import('../src/main/shared-context/host');
  const s = await setup();
  const doc = await s.command(open),
    surface = doc.workspace.surfaces[0]!;
  const agent = {
    projectId: source.projectId,
    agentId: 'agt_report',
    name: 'Researcher',
    instructionProfile: 'discussion-v1',
    access: 'sharedViews' as const,
    configuration: { model: 'vision', effort: 'low', instructions: '' },
    provider: {
      kind: 'codex' as const,
      threadId: 'thread_report',
      accountSessionId: 'session',
      toolsetVersion: SHARED_TOOLSET,
    },
  };
  const submission = {
    requestId: 'req_reportturn',
    agentId: agent.agentId,
    text: 'Discuss',
    model: 'vision',
    effort: 'low',
    createdAt: 1,
    state: 'running' as const,
    threadId: 'thread_report',
    turnId: 'turn_report',
    response: '',
    problem: null,
  };
  await s.controller.changeCollaboration(source.projectId, () => ({
    agents: [agent],
    history: { submissions: [submission] },
  }));
  const session = await s.controller.connect();
  await s.controller.present({
    projectId: source.projectId,
    rendererSessionId: session.rendererSessionId,
    visiblePanes: ['primary', 'secondary'],
  });
  const load = await s.controller.loadSurface({
    projectId: source.projectId,
    rendererSessionId: session.rendererSessionId,
    surfaceId: surface.surfaceId,
  });
  await s.controller.acknowledge(load.acknowledgment);
  const evidence = reportTarget(doc.savedReports![0]!, surface.surfaceId);
  await expect(
    s.command({ kind: 'attach', surfaceId: surface.surfaceId, evidence }),
  ).rejects.toThrow('observation');
  const attached = await s.command({
    kind: 'attach',
    surfaceId: surface.surfaceId,
    evidence,
    acknowledgment: load.acknowledgment,
  });
  expect(attached.chat.attachments![0]!.capture!.asset.hash).toBe(evidence.dataRevision);
  await s.command({
    kind: 'share',
    surfaceId: surface.surfaceId,
    evidence,
    note: 'Review this report',
  });
  const host = new SharedContextHost(
    s.controller,
    s.resources,
    {} as never,
    async () => {
      throw Error('No rasterization');
    },
    () => true,
  );
  host.begin(source.projectId, agent.agentId, submission.requestId);
  let serial = 0;
  const invoke = (tool: string, args: unknown) =>
    host.execute(
      { agent, submission, signal: new AbortController().signal, assert: () => {} },
      {
        tool,
        arguments: args,
        callId: 'call_' + ++serial,
        threadId: submission.threadId,
        turnId: submission.turnId,
      },
    );
  expect((await invoke('workspace_point', { evidence, note: 'No receipt' })).success).toBe(false);
  const result = await invoke('workspace_observe', { surfaceId: surface.surfaceId });
  expect(result.success).toBe(true);
  const first = result.content[0]!;
  if (first.type !== 'text') throw Error();
  const observed = JSON.parse(first.text);
  expect(observed.content.imagesReturned).toEqual([]);
  const point = {
    evidence,
    observedReadId: observed.receipt.observedReadId,
    observation: observed.receipt.observation,
    note: 'Whole result',
  };
  expect((await invoke('workspace_point', point)).success).toBe(true);
  const marks = (await s.controller.read(source.projectId)).sharedReferences!;
  expect(marks.map((m) => m.author.kind)).toEqual(['user', 'agent']);
  await s.command({
    kind: 'reportNavigate',
    surfaceId: surface.surfaceId,
    moduleId: 'M0',
    acknowledgment: load.acknowledgment,
  });
  expect((await invoke('workspace_point', point)).success).toBe(false);
  await s.command({ kind: 'close', surfaceId: surface.surfaceId });
  const closed = await s.controller.read(source.projectId);
  expect(retainedEvidenceHashes([closed])).toContain(evidence.dataRevision);
  const restarted = await s.restart();
  const reopened = await restarted.read(source.projectId);
  expect(reopened.chat.attachments![0]!.evidence).toEqual(evidence);
  expect(reopened.sharedReferences!.map((m) => m.evidence)).toEqual([evidence, evidence]);
});

it('migrates the frozen v20 report registry without changing saved evidence or its backup', async () => {
  const s = await setup();
  const current = await s.command(open);
  const old = { ...current, schemaVersion: 20 };
  const path = join(s.base, 'projects', source.projectId + '.json');
  await writeFile(path, JSON.stringify(old));
  const controller = await s.restart();
  expect((await controller.read(source.projectId)).savedReports).toEqual(current.savedReports);
  expect(JSON.parse(await readFile(path + '.v20.backup', 'utf8'))).toEqual(old);
});

it('saved evidence read guards survive Pane closure but expire across Project sessions', async () => {
  const s = await setup();
  const doc = await s.command(open);
  const guard = s.controller.sentEvidenceGuard(source.projectId);
  await s.command({ kind: 'close', surfaceId: doc.workspace.surfaces[0]!.surfaceId });
  expect(guard).not.toThrow();
  s.controller.disconnect();
  expect(guard).toThrow();
  await s.controller.connect();
  await s.controller.openProject(source.projectId);
  expect(guard).toThrow();
  expect(s.controller.sentEvidenceGuard(source.projectId)).not.toThrow();
});
