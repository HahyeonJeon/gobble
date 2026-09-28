import { mkdtemp, rm, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { afterEach, expect, it, vi } from 'vitest';
import {
  reportTarget,
  reportReading,
  evidenceTextBytes,
  validateManifest,
  SHARED_TOOLSET,
  type SavedReport,
  type SavedReportRecord,
  type AgentAttachment,
  type DraftAttachment,
  type Submission,
  type EvidenceManifest,
  type SurfaceData,
} from '@gobble/contracts';
import { reportAsset } from '../src/main/evidence/report';
import { contentHash, evidenceInput, evidencePreview } from '../src/main/evidence/materialize';
import { EvidenceStorage } from '../src/main/evidence/storage';
import { EvidenceService } from '../src/main/evidence/service';
import type { EvidenceDraft } from '../src/main/evidence/ports';
import { SharedContextHost } from '../src/main/shared-context/host';
import type { ToolContext } from '../src/main/shared-context/ports';
import type { WorkspaceController } from '../src/main/workspace/controller';
import type { WorkspaceResources } from '../src/main/workspace/service';
import { ReportEvidence } from '../src/main/evidence/report';

const dirs: string[] = [];
afterEach(async () => {
  vi.restoreAllMocks();
  await Promise.all(dirs.splice(0).map((p) => rm(p, { recursive: true, force: true })));
});
const projectId = 'prj_reports';
const png =
  'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+aMZkAAAAASUVORK5CYII=';
const report: SavedReport = {
  schemaVersion: 1,
  projectId,
  runRef: 'run_quality',
  capturedAt: 1,
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
  content: {
    profile: 'fastqc-0.12.1-v1',
    title: 'Quality report',
    logoLabel: 'FastQC',
    headerTitle: 'FastQC Report',
    headerFilename: 'reads.fq',
    summaryHeading: 'Summary',
    summary: [{ moduleId: 'M0', title: 'Quality', status: '[PASS]' }],
    modules: [
      {
        id: 'M0',
        title: 'Quality',
        status: '[OK]',
        blocks: [
          { kind: 'table', headers: ['Measure', 'Value'], rows: [['Reads', '100']] },
          { kind: 'text', text: 'Original source statement' },
          {
            kind: 'image',
            id: 'image-0',
            alt: 'Original quality chart',
            width: 1,
            height: 1,
            base64: png,
          },
        ],
      },
    ],
    footer: 'Produced by FastQC',
  },
};
function saved(value: SavedReport): SavedReportRecord {
  const bytes = Buffer.from(JSON.stringify(value));
  return {
    projectId: value.projectId,
    runRef: value.runRef,
    producer: value.producer,
    capturedAt: value.capturedAt,
    title: value.content.title,
    profile: value.content.profile,
    asset: { hash: contentHash(bytes), byteLength: bytes.length },
  };
}
async function setup(value = structuredClone(report)) {
  const base = await mkdtemp(join(tmpdir(), 'gobble-report-sharing-'));
  dirs.push(base);
  const storage = new EvidenceStorage(join(base, 'evidence'));
  const record = saved(value);
  const target = reportTarget(record, 'srf_report');
  const draftAttachment: DraftAttachment = {
    attachmentId: 'att_report',
    evidence: target,
    label: 'Quality report',
    createdAt: 2,
  };
  const data: SurfaceData = { kind: 'report', saved: record, value };
  const asset = reportAsset(draftAttachment, data);
  await storage.putCaptured(projectId, asset);
  const reports = new ReportEvidence(storage, async () => {
    throw Error('Source decoding is forbidden');
  });
  const agent: AgentAttachment = {
    projectId,
    agentId: 'agt_research',
    name: 'Researcher',
    instructionProfile: 'discussion-v1',
    configuration: { model: 'vision', effort: 'medium', instructions: '' },
    access: 'sharedViews',
    provider: {
      kind: 'codex',
      threadId: 'thread_report',
      accountSessionId: 'session',
      toolsetVersion: SHARED_TOOLSET,
    },
  };
  const draft: EvidenceDraft = {
    rendererSessionId: 'renderer',
    attachmentRevision: 1,
    agent,
    attachments: [{ ...draftAttachment, capture: asset.manifest.capture! }],
  };
  let active = true,
    imageCapable = true;
  const submission: Submission = {
    requestId: 'req_message',
    agentId: agent.agentId,
    text: 'Explain this result',
    evidence: [asset.manifest],
    preparedEvidenceId: 'prep_message',
    model: 'vision',
    effort: 'medium',
    createdAt: 3,
    state: 'running',
    threadId: 'thread_report',
    turnId: 'turn_report',
    response: '',
    problem: null,
  };
  const guard = () => {
    if (!active) throw Error('Revoked');
  };
  const lookup = vi.fn(
    async (
      _project: string,
      requestId: string,
      attachmentId: string,
    ): Promise<EvidenceManifest> => {
      guard();
      if (
        _project !== projectId ||
        requestId !== submission.requestId ||
        attachmentId !== 'att_report'
      )
        throw Error('Unknown message');
      return structuredClone(asset.manifest);
    },
  );
  const workspace = {
    readEvidenceDraft: async () => structuredClone(draft),
    assertEvidenceSession: guard,
    readSentEvidence: lookup,
    sentEvidenceGuard: () => guard,
  };
  const resources = {
    read: vi.fn(async (id: string, ref: import('@gobble/contracts').ResourceRef) => {
      if (ref.kind !== 'report') throw Error('Unexpected resource');
      return reports.read(id, ref.saved);
    }),
  } as unknown as WorkspaceResources;
  const supportsImages = () => imageCapable;
  const evidence = new EvidenceService(
    workspace,
    resources,
    storage,
    async () => {
      throw Error('Never resize report images');
    },
    {
      require: () => {
        guard();
        return 'session';
      },
      supportsImages,
    },
  );
  const host = new SharedContextHost(
    workspace as unknown as WorkspaceController,
    resources,
    {} as never,
    async () => {
      throw Error('Never resize');
    },
    supportsImages,
  );
  const context: ToolContext = {
    agent,
    submission,
    signal: new AbortController().signal,
    assert: guard,
  };
  host.begin(projectId, agent.agentId, submission.requestId);
  let sequence = 0;
  const invoke = (args: unknown, ctx = context, callId = 'call_' + ++sequence) =>
    host.execute(ctx, {
      threadId: 'thread_report',
      turnId: 'turn_report',
      callId,
      tool: 'read_report',
      arguments: args,
    });
  return {
    base,
    record,
    data,
    asset,
    draft,
    evidence,
    host,
    context,
    invoke,
    lookup,
    resources,
    storage,
    revoke: () => {
      active = false;
    },
    capability: (v: boolean) => {
      imageCapable = v;
    },
  };
}

it('uses one saved blob for previews and complete message content, excluding original PNGs from initial delivery', async () => {
  const s = await setup();
  const receipt = await s.evidence.prepare({
    projectId,
    agentId: s.context.agent.agentId,
    attachmentRevision: 1,
  });
  const preview = await s.evidence.preview({
    kind: 'prepared',
    projectId,
    preparedId: receipt.preparedId,
    attachmentId: 'att_report',
  });
  expect(preview).toEqual({ kind: 'report', report });
  const delivery = await s.evidence.accept(
    {
      projectId,
      agentId: s.context.agent.agentId,
      requestId: 'req_send',
      text: 'Explain',
      preparedEvidenceId: receipt.preparedId,
    },
    s.context.agent,
    'session',
    () => {},
  );
  expect(delivery.manifests[0]!.asset.hash).toBe(s.record.asset.hash);
  expect(delivery.input).toHaveLength(1);
  const message = delivery.input[0]!;
  expect(message.type).toBe('text');
  if (message.type !== 'text') throw Error();
  expect(message.text).not.toContain(png);
  expect(message.text).not.toContain('base64');
  expect(message.text).toContain('Original source statement');
  expect(message.text).toContain('image-0');
  expect(message.text).toContain('read_report');
  expect(evidenceTextBytes(s.asset.manifest)).toBe(
    Buffer.byteLength(JSON.stringify(reportReading(report))),
  );
  const forged = structuredClone(s.asset);
  forged.manifest.representation = {
    kind: 'report',
    truncated: false,
    textByteLength: 1,
    imageCount: 0,
  };
  expect(() => evidencePreview(forged)).toThrow();
});

it('reads only the current addressed report and returns each original PNG without a pointing receipt', async () => {
  const s = await setup();
  const reading = await s.invoke({ attachmentId: 'att_report' });
  expect(reading.success).toBe(true);
  const text = reading.content[0]!;
  if (text.type !== 'text') throw Error();
  expect(JSON.parse(text.text).report).toEqual(reportReading(report));
  expect(text.text).not.toContain(png);
  const image = await s.invoke({ attachmentId: 'att_report', imageId: 'image-0' });
  expect(image.success).toBe(true);
  expect(image.content[1]).toEqual({ type: 'image', url: 'data:image/png;base64,' + png });
  expect(JSON.stringify(image)).not.toContain('observedReadId');
  const identity = image.content[0]!;
  if (identity.type !== 'text') throw Error();
  expect(JSON.parse(identity.text).pointable).toBe(false);
  expect((await s.invoke({ attachmentId: 'att_report', imageId: 'image-99' })).success).toBe(false);
  expect((await s.invoke({ attachmentId: 'att_old' })).success).toBe(false);
  expect((await s.invoke({ attachmentId: 'att_report', submissionId: 'req_other' })).success).toBe(
    false,
  );
  for (const changed of [
    { ...s.context, submission: { ...s.context.submission, evidence: [] } },
    { ...s.context, submission: { ...s.context.submission, agentId: 'agt_other' } },
    { ...s.context, agent: { ...s.context.agent, access: 'messages' as const } },
  ])
    expect((await s.invoke({ attachmentId: 'att_report' }, changed)).success).toBe(false);
  const foreign = { ...s.context, agent: { ...s.context.agent, projectId: 'prj_other' } };
  s.host.begin('prj_other', foreign.agent.agentId, foreign.submission.requestId);
  expect((await s.invoke({ attachmentId: 'att_report' }, foreign)).success).toBe(false);
  s.host.release(projectId, s.context.agent.agentId, s.context.submission.requestId);
  expect((await s.invoke({ attachmentId: 'att_report' })).success).toBe(false);
});

it('rechecks recipient capability, manifest, cancellation and account authorization after I/O and cached replay', async () => {
  const s = await setup();
  expect((await s.invoke({ attachmentId: 'att_report' }, s.context, 'cached')).success).toBe(true);
  s.capability(false);
  expect((await s.invoke({ attachmentId: 'att_report' }, s.context, 'cached')).success).toBe(false);
  await expect(
    s.evidence.prepare({ projectId, agentId: s.context.agent.agentId, attachmentRevision: 1 }),
  ).rejects.toThrow('cannot receive images');
  s.capability(true);
  s.draft.agent!.access = 'messages';
  await expect(
    s.evidence.prepare({ projectId, agentId: s.context.agent.agentId, attachmentRevision: 1 }),
  ).rejects.toThrow('Shared views');
  s.draft.agent!.access = 'sharedViews';
  const receipt = await s.evidence.prepare({
    projectId,
    agentId: s.context.agent.agentId,
    attachmentRevision: 1,
  });
  s.draft.agent!.access = 'messages';
  await expect(
    s.evidence.accept(
      {
        projectId,
        agentId: s.context.agent.agentId,
        requestId: 'req_send',
        text: 'Explain',
        preparedEvidenceId: receipt.preparedId,
      },
      s.context.agent,
      'session',
      () => {},
    ),
  ).rejects.toThrow();
  const t = await setup();
  const original = t.resources.read;
  vi.mocked(t.resources.read).mockImplementationOnce(async (...args) => {
    const value = await original(...args);
    t.revoke();
    return value;
  });
  expect((await t.invoke({ attachmentId: 'att_report', imageId: 'image-0' })).success).toBe(false);
  const u = await setup();
  u.lookup
    .mockImplementationOnce(async () => structuredClone(u.asset.manifest))
    .mockImplementationOnce(async () => ({ ...u.asset.manifest, label: 'Different' }));
  expect((await u.invoke({ attachmentId: 'att_report' })).success).toBe(false);
});

it('rejects corrupt retained bytes without reacquiring a live source', async () => {
  const s = await setup();
  await writeFile(
    join(s.base, 'evidence', projectId, s.record.asset.hash.slice(7) + '.blob'),
    'corrupt',
  );
  const result = await s.invoke({ attachmentId: 'att_report', imageId: 'image-0' });
  expect(result.success).toBe(false);
  expect(JSON.stringify(result)).toContain('missing or corrupt');
});

it('counts report delivery separately from retained bytes and rejects combined text overflow', async () => {
  const large = structuredClone(report);
  // Source decoder qualification belongs to report-reader.spec; this tests the delivery/storage boundary.
  large.content.modules[0]!.blocks.push({ kind: 'text', text: 'x'.repeat(40_000) });
  const s = await setup(large);
  s.draft.attachments.push({ ...s.draft.attachments[0]!, attachmentId: 'att_second' });
  await expect(
    s.evidence.prepare({ projectId, agentId: s.context.agent.agentId, attachmentRevision: 1 }),
  ).rejects.toThrow('combined text');
  const manifest = structuredClone(s.asset.manifest);
  manifest.asset.hash = 'sha256:' + 'd'.repeat(64);
  expect(() => validateManifest(manifest)).toThrow();
  expect(evidenceInput([s.asset])[0]!.type).toBe('text');
});
