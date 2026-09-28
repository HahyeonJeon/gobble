import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import {
  PipelineFlowSchema,
  PipelineTargetSchema,
  WorkspaceDocumentV15Schema,
  SharedToolRequestV9Schema,
  parse,
  pipelineTarget,
  pipelineSubject,
  connectionSelector,
  resolveReferenceTarget,
  containsObservedTarget,
  readStoredWorkspace,
  type PipelineInspection,
  type Surface,
  type SurfaceData,
} from '@gobble/contracts';
import { RenderSession } from '../src/main/workspace/render-session';
import { ObservedReads } from '../src/main/shared-context/observed';
import { materializePipelineEvidence } from '../src/main/evidence/pipeline-evidence';
import { evidencePreview, contentHash } from '../src/main/evidence/materialize';
import { emptyWorkspace, transition } from '../src/main/workspace/model';

function fixture() {
  const flow = parse(
    PipelineFlowSchema,
    JSON.parse(readFileSync(new URL('./fixtures/pipeline-flow.json', import.meta.url), 'utf8')),
  );
  const value: PipelineInspection = {
    projectId: 'prj_one',
    pipelineId: 'pip_one',
    state: 'ready',
    artifact: {
      artifactId: 'sha256:' + 'a'.repeat(64),
      sourceRevision: 'sha256:' + 'b'.repeat(64),
      checkedAt: '2026-09-09T00:00:00Z',
      flow: {
        ...flow,
        schemaVersion: 2,
        steps: flow.steps.map((s) => ({
          ...s,
          settings:
            s.id === 'trim'
              ? [
                  { key: 'quality', label: 'Quality threshold', unit: 'Phred', value: 25 },
                  { key: 'length', label: 'Minimum length', unit: 'bp', value: null },
                ]
              : [],
        })),
      },
    },
  };
  const data: SurfaceData = { kind: 'pipeline', value };
  const surface: Surface = {
    surfaceId: 'srf_flow',
    projectId: 'prj_one',
    resource: { kind: 'pipeline', pipelineId: 'pip_one' },
    view: 'pipeline',
    pinned: false,
    openedBy: { kind: 'user' },
  };
  const session = new RenderSession();
  session.reveal([surface.surfaceId]);
  const load = session.complete(
    session.begin(
      { projectId: surface.projectId, surfaceId: surface.surfaceId, rendererSessionId: session.id },
      surface,
    ),
    data,
  );
  session.acknowledge(load.acknowledgment);
  return { value, data, surface, session, acknowledgment: load.acknowledgment };
}
describe('checked pipeline discussion', () => {
  it('bounds whole-artifact observations and never authorizes omitted settings', () => {
    const f = fixture();
    const base = f.value.artifact!.flow.steps[0]!;
    f.value.artifact!.flow = {
      ...f.value.artifact!.flow,
      schemaVersion: 2,
      steps: Array.from({ length: 100 }, (_, i) => ({
        ...base,
        id: 'step_' + i,
        settings: [{ key: 'quality', label: 'Quality threshold', unit: 'Phred', value: i }],
      })),
    };
    const reads = new ObservedReads();
    const out = reads.observePipeline(f, undefined, 'view');
    expect(Buffer.byteLength(JSON.stringify(out.content))).toBeLessThan(50 * 1024);
    expect(out.content.returnedSubjects).toBeLessThanOrEqual(128);
    expect(out.content.truncated).toBe(true);
    const omitted = pipelineTarget(
      f.value,
      { kind: 'setting', stepId: 'step_99', key: 'quality' },
      'srf_flow',
    );
    expect(() =>
      reads.assertPoint(out.receipt.observedReadId, omitted, out.receipt.observation),
    ).toThrow();
    const exact = reads.observePipeline(f, omitted.selection, 'view');
    expect(() =>
      reads.assertPoint(exact.receipt.observedReadId, omitted, exact.receipt.observation),
    ).not.toThrow();
  });
  it('resolves step, directional port, setting and exact edge only within the checked artifact', () => {
    const f = fixture(),
      flow = f.value.artifact!.flow;
    const source = {
      projectId: 'prj_one',
      resource: f.surface.resource,
      dataRevision: 'live-state-revision',
      data: f.data,
    };
    const selectors = [
      { kind: 'step' as const, stepId: 'trim' },
      { kind: 'port' as const, stepId: 'trim', direction: 'output' as const, portName: 'trimmed' },
      { kind: 'setting' as const, stepId: 'trim', key: 'quality' },
      connectionSelector(flow.connections[0]!),
    ];
    for (const selector of selectors) {
      const target = pipelineTarget(f.value, selector, 'srf_flow');
      parse(PipelineTargetSchema, target);
      expect(resolveReferenceTarget(target, source).kind).toBe('exact');
      expect(
        resolveReferenceTarget({ ...target, dataRevision: 'sha256:' + 'c'.repeat(64) }, source)
          .kind,
      ).toBe('historical');
      expect(
        resolveReferenceTarget({ ...target, sourceRevision: 'sha256:' + 'c'.repeat(64) }, source)
          .kind,
      ).toBe('unavailable');
      expect(resolveReferenceTarget({ ...target, projectId: 'prj_foreign' }, source).kind).toBe(
        'unavailable',
      );
    }
    expect(() =>
      pipelineTarget(f.value, {
        kind: 'port',
        stepId: 'trim',
        direction: 'input',
        portName: 'trimmed',
      }),
    ).toThrow();
    const edge = connectionSelector(flow.connections[0]!);
    if (edge.kind !== 'connection') throw new Error();
    expect(() =>
      pipelineTarget(f.value, { ...edge, to: { ...edge.to, portName: 'other' } }),
    ).toThrow();
    expect(() =>
      pipelineTarget(f.value, { kind: 'setting', stepId: 'quality', key: 'quality' }),
    ).toThrow();
    const defaults = pipelineSubject(flow, { kind: 'setting', stepId: 'trim', key: 'length' });
    expect(defaults).toMatchObject({ setting: { value: null } });
  });
  it('requires the returned subject and current render receipt; observes without changing User state', () => {
    const f = fixture(),
      reads = new ObservedReads(),
      before = JSON.stringify(f.surface);
    const target = pipelineTarget(
      f.value,
      { kind: 'setting', stepId: 'trim', key: 'quality' },
      'srf_flow',
    );
    const out = reads.observePipeline(f, target.selection, 'view');
    expect(() =>
      reads.assertPoint(out.receipt.observedReadId, target, out.receipt.observation),
    ).not.toThrow();
    const other = pipelineTarget(f.value, { kind: 'step', stepId: 'trim' }, 'srf_flow');
    expect(containsObservedTarget(target, other)).toBe(false);
    expect(() =>
      reads.assertPoint(out.receipt.observedReadId, other, out.receipt.observation),
    ).toThrow();
    const hidden = reads.observePipeline(f, target.selection, 'source-preview');
    expect(() =>
      reads.assertPoint(hidden.receipt.observedReadId, target, hidden.receipt.observation),
    ).toThrow();
    expect(JSON.stringify(f.surface)).toBe(before);
    const load = f.session.complete(
      f.session.begin(
        { projectId: 'prj_one', surfaceId: 'srf_flow', rendererSessionId: f.session.id },
        f.surface,
      ),
      { ...f.data, value: { ...f.value, state: 'checking', jobId: 'req_new' } },
    );
    f.session.acknowledge(load.acknowledgment);
    expect(() => f.session.assertAcknowledgment(out.receipt.observation)).toThrow();
    expect(() =>
      reads.assertPoint(out.receipt.observedReadId, target, load.acknowledgment),
    ).toThrow();
  });
  it('freezes readable facts independent of later source and rejects a relabelled capture', () => {
    const f = fixture();
    const evidence = pipelineTarget(
      f.value,
      { kind: 'setting', stepId: 'trim', key: 'quality' },
      'srf_flow',
    );
    const asset = materializePipelineEvidence(
      { attachmentId: 'att_one', evidence, label: 'Flow', createdAt: 1 },
      f.data,
    );
    expect(asset.manifest.label).toBe('Trim adapters · Quality threshold');
    f.value.artifact!.artifactId = 'sha256:' + 'c'.repeat(64);
    expect(evidencePreview(asset)).toMatchObject({
      kind: 'pipeline',
      pipeline: {
        subject: { setting: { value: 25 } },
        target: { dataRevision: evidence.dataRevision },
      },
    });
    const changed = JSON.parse(asset.bytes.toString());
    changed.pipeline.subject.setting.key = 'length';
    const bytes = Buffer.from(JSON.stringify(changed));
    expect(() =>
      evidencePreview({
        ...asset,
        bytes,
        manifest: {
          ...asset.manifest,
          asset: { ...asset.manifest.asset, hash: contentHash(bytes) },
        },
      }),
    ).toThrow();
    const doc = transition(
      emptyWorkspace('prj_one'),
      { kind: 'open', resource: f.surface.resource, pane: 'primary', duplicate: false },
      { surfaceId: 'srf_flow', title: 'Flow', view: 'pipeline' },
    );
    // Use the actual DraftAttachment shape, not the expanded delivery manifest.
    const { attachmentId, label, createdAt, capture } = asset.manifest;
    if (!capture) throw new Error('Capture missing');
    doc.chat.attachments = [{ attachmentId, label, createdAt, evidence, capture }];
    expect(readStoredWorkspace(doc).chat.attachments?.[0]?.capture?.kind).toBe('pipeline');
    expect(() => parse(WorkspaceDocumentV15Schema, { ...doc, schemaVersion: 15 })).toThrow();
    expect(() =>
      parse(SharedToolRequestV9Schema, {
        tool: 'workspace_point',
        arguments: { evidence, note: 'old toolset' },
      }),
    ).toThrow();
    const old = emptyWorkspace('prj_one');
    expect(readStoredWorkspace({ ...old, schemaVersion: 15 }).schemaVersion).toBe(21);
  });
});
