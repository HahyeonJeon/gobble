import { describe, expect, it } from 'vitest';
import {
  containsObservedTarget,
  type Surface,
  type SurfaceData,
  type EvidenceRef,
  type RenderAcknowledgment,
} from '@gobble/contracts';
import { ObservedReads } from '../src/main/shared-context/observed';
import { presentLog } from '../src/main/service/log-presentation';
import { RenderSession } from '../src/main/workspace/render-session';
import { emptyWorkspace } from '../src/main/workspace/model';
import {
  ObservedReferenceViews,
  prepareObservedReference,
} from '../src/main/workspace/observed-reference-views';

function logFixture(stdout = 'same 😀\nother line', stderr = 'same 😀\nstderr only') {
  const surface: Surface = {
    projectId: 'prj_one',
    surfaceId: 'srf_log',
    resource: { kind: 'log', runRef: 'run_one', taskId: 'align:S03', attempt: 2 },
    view: 'log',
    pinned: false,
    openedBy: { kind: 'user' },
    logView: { stream: 'stderr', viewRevision: 1 },
  };
  const data: SurfaceData = {
    kind: 'log',
    value: presentLog({
      projectId: 'prj_one',
      runRef: 'run_one',
      engineRevision: 'checkpoint',
      observedAt: 1,
      instance: 'align:S03',
      attempt: 2,
      tailLimitBytes: 4096,
      logs: [{ identity: 'align:S03', stdout_tail: stdout, stderr_tail: stderr }],
    }),
  };
  const session = new RenderSession();
  session.reveal(['srf_log']);
  const load = session.complete(
    session.begin(
      { projectId: 'prj_one', surfaceId: 'srf_log', rendererSessionId: session.id },
      surface,
    ),
    data,
  );
  session.acknowledge(load.acknowledgment);
  return { surface, data, acknowledgment: load.acknowledgment };
}
const range = (stream: 'stdout' | 'stderr', end = 7) => ({
  kind: 'log-text' as const,
  coordinateSpace: 'decoded-preview-utf16-line-column' as const,
  stream,
  start: { line: 1, column: 0 },
  end: { line: 1, column: end },
});

describe('v5 observed scopes', () => {
  it('returns only the displayed stream and requires explicit scope for another stream', () => {
    const view = logFixture(),
      reads = new ObservedReads();
    const output = reads.observe(view, undefined, 'view')!;
    expect(output.content).toMatchObject({ stream: 'stderr', text: 'same 😀\nstderr only' });
    expect(() => reads.observe(view, undefined, 'view', 'stdout')).toThrow('not displayed');
    const other = reads.observe(view, range('stdout'), 'source-preview', 'stdout')!;
    expect(other.content).toMatchObject({
      displayedStream: 'stderr',
      stream: 'stdout',
      text: 'same 😀',
    });
    expect(() =>
      reads.assertPoint(
        other.receipt.observedReadId,
        other.receipt.evidence,
        other.receipt.observation,
      ),
    ).toThrow('Observe this exact target');
    expect(other.receipt.pointable).toBe(false);
    expect(view.surface).toMatchObject({ logView: { stream: 'stderr' } });
  });
  it('rejects identical text in another stream, wider coordinates, split Unicode and foreign receipts', () => {
    const view = logFixture(),
      reads = new ObservedReads();
    const result = reads.observe(view, range('stderr'), 'view')!;
    const ref = result.receipt.evidence;
    const assert = (target: EvidenceRef, ack: RenderAcknowledgment = result.receipt.observation) =>
      reads.assertPoint(result.receipt.observedReadId, target, ack);
    for (const target of [
      { ...ref, selection: range('stdout') },
      { ...ref, selection: { ...range('stderr'), end: { line: 2, column: 1 } } },
      { ...ref, projectId: 'prj_two' },
      { ...ref, dataRevision: 'different' },
    ])
      expect(() => assert(target)).toThrow();
    expect(() => reads.observe(view, range('stderr', 6), 'view')).toThrow();
    expect(() => assert(ref, { ...result.receipt.observation, surfaceId: 'srf_other' })).toThrow();
    expect(() =>
      new ObservedReads().assertPoint(
        result.receipt.observedReadId,
        ref,
        result.receipt.observation,
      ),
    ).toThrow();
    expect(containsObservedTarget(ref, { ...ref, selection: range('stderr', 4) })).toBe(true);
  });
  it('describes an empty stream honestly and bounds each turn', () => {
    const view = logFixture('', ''),
      reads = new ObservedReads();
    const output = reads.observe(view, undefined, 'view')!;
    expect(output.receipt.pointable).toBe(false);
    expect(() =>
      reads.assertPoint(
        output.receipt.observedReadId,
        output.receipt.evidence,
        output.receipt.observation,
      ),
    ).toThrow();
    for (let i = 1; i < 32; i++) reads.observe(view, undefined, 'view');
    expect(() => reads.observe(view, undefined, 'view')).toThrow('limit');
  });
  it('bounds returned task JSON and only authorizes tasks actually returned', () => {
    const view = logFixture();
    const surface: Surface = {
      ...view.surface,
      resource: { kind: 'run', runRef: 'run_one' },
      view: 'run',
    };
    const data: SurfaceData = {
      kind: 'run',
      value: {
        projectId: 'prj_one',
        runRef: 'run_one',
        runId: 'engine',
        imageId: 'image',
        observedAt: 1,
        engineRevision: 'revision',
        pipelineName: null,
        status: 'failed',
        dependencies: [],
        tasks: Array.from({ length: 1000 }, (_, i) => ({
          instanceId: String(i),
          taskId: 'align',
          attempt: 2,
          name: 'a'.repeat(400),
          reason: null,
          status: 'failed',
        })),
      },
    };
    const reads = new ObservedReads();
    const output = reads.observe({ ...view, surface, data }, undefined, 'view')!;
    const encoded = JSON.stringify(output);
    expect(Buffer.byteLength(encoded)).toBeLessThan(64 * 1024);
    expect(JSON.parse(encoded).content.truncated).toBe(true);
    expect(() =>
      reads.assertPoint(
        output.receipt.observedReadId,
        {
          ...output.receipt.evidence,
          selection: {
            kind: 'run-task',
            coordinateSpace: 'observed-instance-attempt',
            instanceId: '999',
            attempt: 2,
          },
        },
        output.receipt.observation,
      ),
    ).toThrow();
  });
});

it('retains the original return destination across repeated Show and clears a closed reference', () => {
  const view = logFixture(),
    reads = new ObservedReads(),
    owner = new ObservedReferenceViews();
  const evidence = reads.observe(view, range('stdout'), 'source-preview')!.receipt.evidence;
  const doc = emptyWorkspace('prj_one');
  doc.workspace.surfaces = [view.surface, { ...view.surface, surfaceId: 'srf_previous' }];
  doc.workspace.layout.primary = {
    tabs: ['srf_previous', 'srf_log'],
    activeSurfaceId: 'srf_previous',
  };
  const reference = {
    referenceId: 'ref_one',
    evidence,
    label: 'stdout',
    note: '',
    author: { kind: 'agent' as const, agentId: 'agt_one', name: 'Researcher' },
    createdAt: 1,
    retracted: false,
  };
  owner.set(prepareObservedReference(doc, reference, view.data, 'req_first'));
  owner.set(prepareObservedReference(doc, reference, view.data, 'req_second'));
  owner.returnTo(doc, 'req_second');
  expect(doc.workspace.layout.primary.activeSurfaceId).toBe('srf_previous');
  expect(doc.workspace.surfaces[0]).toMatchObject({ logView: { stream: 'stderr' } });
  doc.workspace.surfaces = [];
  owner.reconcile(doc);
  expect(owner.forSurface('srf_log')).toBeUndefined();
});
