import { ObservedReads } from '../src/main/shared-context/observed';
import { RenderSession } from '../src/main/workspace/render-session';
import type { Surface } from '@gobble/contracts';
import { describe, it, expect } from 'vitest';
import {
  parse,
  ContinuationSchema,
  type RunPresentationV2,
  type EvidenceRef,
  type SurfaceData,
} from '@gobble/contracts';
import { continuationContext } from '../src/main/service/continuation-context';
import { observedDataRevision, checkSelection } from '../src/main/workspace/selection';
import { materializeObserved } from '../src/main/evidence/capture';
import { evidencePreview } from '../src/main/evidence/materialize';
import { runObservation } from '../src/main/shared-context/run-observation';
const hash = 'sha256:' + 'a'.repeat(64),
  snapshot = 'b'.repeat(32);
function fixture() {
  const review = parse(ContinuationSchema, {
    projectId: 'prj_one',
    pipelineId: 'pip_one',
    launchReviewId: 'req_launch',
    runRef: 'run_one',
    requestId: 'req_review',
    createdAt: '2026-09-27T00:00:00Z',
    state: 'ready',
    issue: '',
    review: {
      schemaVersion: 1,
      workspaceId: 'req_launch',
      originDigest: hash,
      previousHead: hash,
      snapshot,
      stateDigest: hash,
      digest: hash,
      steps: [
        { taskId: 'trim', priorAttempt: 1, plannedAttempt: 1, action: 'reuse' },
        { taskId: 'qc', priorAttempt: 1, plannedAttempt: 2, action: 'restart' },
      ],
      files: [{ path: 'inputs/reads.fastq.gz', sha256: 'c'.repeat(64) }],
    },
  });
  const run: RunPresentationV2 = {
    projectId: 'prj_one',
    runRef: 'run_one',
    engineRevision: snapshot,
    observedAt: 1,
    imageId: hash,
    runId: 'analysis',
    pipelineName: 'RNA study',
    status: 'stopped',
    tasks: ['trim', 'qc'].map((id) => ({
      taskId: id,
      instanceId: id,
      attempt: 1,
      name: id,
      status: id === 'trim' ? 'succeeded' : 'canceled',
      reason: null,
    })),
    dependencies: [],
  };
  return { run, review };
}
describe('shared continuation context', () => {
  it('joins only the exact stopped snapshot and task attempts, never an old ready review under a newer check', () => {
    const { run, review } = fixture();
    expect(continuationContext(run, snapshot, [review])?.reviewId).toBe(review.requestId);
    expect(continuationContext(run, 'other', [review])).toBeUndefined();
    expect(
      continuationContext({ ...run, projectId: 'prj_other' }, snapshot, [review]),
    ).toBeUndefined();
    expect(
      continuationContext({ ...run, runRef: 'run_other' }, snapshot, [review]),
    ).toBeUndefined();
    expect(continuationContext({ ...run, status: 'stopping' }, snapshot, [review])).toBeUndefined();
    const { review: _evidence, ...checking } = review;
    expect(
      continuationContext(run, snapshot, [
        review,
        { ...checking, requestId: 'req_new', createdAt: '2026-09-28T00:00:00Z', state: 'checking' },
      ]),
    ).toBeUndefined();
    run.tasks[1]!.attempt = 2;
    expect(continuationContext(run, snapshot, [review])).toBeUndefined();
  });
  it('captures the exact plan independently of recheck, includes Agent facts and refuses retargeting', () => {
    const { run, review } = fixture();
    run.continuation = continuationContext(run, snapshot, [review])!;
    const data: SurfaceData = { kind: 'run', value: run };
    const evidence: EvidenceRef = {
      schemaVersion: 3,
      projectId: run.projectId,
      resource: { kind: 'run', runRef: run.runRef },
      dataRevision: observedDataRevision(data),
      selection: {
        kind: 'run-task',
        coordinateSpace: 'observed-instance-attempt',
        instanceId: 'qc',
        attempt: 1,
      },
    };
    const asset = materializeObserved(
      { attachmentId: 'att_one', label: 'Quality check', createdAt: 1, evidence },
      data,
    );
    const preview = evidencePreview(asset);
    expect(preview).toMatchObject({
      kind: 'run',
      observation: {
        schemaVersion: 2,
        source: {
          value: {
            continuation: { reviewId: 'req_review' },
            tasks: [{ taskId: 'qc', attempt: 1 }],
          },
        },
      },
    });
    expect(runObservation(run).continuation).toEqual(run.continuation);
    run.continuation.reviewId = 'req_new';
    run.continuation.review.digest = 'sha256:' + 'f'.repeat(64);
    expect(() => checkSelection(evidence, data)).toThrow();
    expect(evidencePreview(asset)).toEqual(preview);
    expect(asset.bytes.toString()).toContain('req_review');
    expect(asset.bytes.toString()).not.toContain('req_new');
  });
  it('authorizes an Agent mark only for returned facts from the exact shared review', () => {
    const { run, review } = fixture();
    run.continuation = continuationContext(run, snapshot, [review])!;
    const surface: Surface = {
      projectId: run.projectId,
      surfaceId: 'srf_run',
      resource: { kind: 'run', runRef: run.runRef },
      view: 'run',
      pinned: false,
      openedBy: { kind: 'user' },
    };
    const data: SurfaceData = { kind: 'run', value: run };
    const session = new RenderSession();
    session.reveal([surface.surfaceId]);
    const loaded = session.complete(
      session.begin(
        { projectId: run.projectId, surfaceId: surface.surfaceId, rendererSessionId: session.id },
        surface,
      ),
      data,
    );
    session.acknowledge(loaded.acknowledgment);
    const reads = new ObservedReads();
    const response = reads.observe(
      { surface, data, acknowledgment: loaded.acknowledgment },
      {
        kind: 'run-task',
        coordinateSpace: 'observed-instance-attempt',
        instanceId: 'qc',
        attempt: 1,
      },
      'view',
    )!;
    expect(response.content).toMatchObject({ continuation: { reviewId: 'req_review' } });
    const receipt = response.receipt;
    expect(() =>
      reads.assertPoint(receipt.observedReadId, receipt.evidence, receipt.observation),
    ).not.toThrow();
    expect(() =>
      reads.assertPoint(
        receipt.observedReadId,
        { ...receipt.evidence, dataRevision: 'new-review' },
        receipt.observation,
      ),
    ).toThrow();
    expect(() =>
      reads.assertPoint(
        receipt.observedReadId,
        { ...receipt.evidence, projectId: 'prj_other' },
        receipt.observation,
      ),
    ).toThrow();
  });
  it('keeps ordinary Run captures on v1', () => {
    const { run } = fixture(),
      data: SurfaceData = { kind: 'run', value: run };
    const asset = materializeObserved(
      {
        attachmentId: 'att_one',
        label: 'Run',
        createdAt: 1,
        evidence: {
          schemaVersion: 3,
          projectId: run.projectId,
          resource: { kind: 'run', runRef: run.runRef },
          dataRevision: observedDataRevision(data),
        },
      },
      data,
    );
    expect(evidencePreview(asset)).toMatchObject({ observation: { schemaVersion: 1 } });
  });
});
