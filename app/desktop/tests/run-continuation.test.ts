import { describe, it, expect, vi } from 'vitest';
import {
  parse,
  ContinuationSchema,
  validateContinuation,
  type Continuation,
} from '@gobble/contracts';
import { RunContinuationService } from '../src/main/service/run-continuation';

const hash = 'sha256:' + 'a'.repeat(64),
  lease = 'b'.repeat(32);
const result = (value: unknown) => ({ schemaVersion: 1, ok: true, value });
const capabilities = result({
  protocolVersion: 1,
  queries: ['continuations', 'continuation_support'],
  mutations: [
    'check_continuation',
    'confirm_continuation',
    'refresh_continuation',
    'stop_continuation',
  ],
});
function review(): Continuation {
  return parse(ContinuationSchema, {
    projectId: 'prj_one',
    pipelineId: 'pip_one',
    launchReviewId: 'req_launch',
    runRef: 'run_saved',
    requestId: 'req_review',
    createdAt: '2026-09-27T00:00:00Z',
    state: 'ready',
    issue: '',
    review: {
      schemaVersion: 1,
      workspaceId: 'req_launch',
      originDigest: hash,
      previousHead: hash,
      stateDigest: hash,
      snapshot: lease,
      digest: hash,
      steps: [
        { taskId: 'trim', priorAttempt: 1, plannedAttempt: 1, action: 'reuse' },
        { taskId: 'qc', priorAttempt: 1, plannedAttempt: 2, action: 'restart' },
      ],
      files: [{ path: 'inputs/reads.fastq.gz', sha256: 'c'.repeat(64) }],
    },
  });
}
function accepted(): Continuation {
  return {
    ...review(),
    state: 'accepted',
    operation: {
      requestId: 'req_continue',
      state: 'unknown',
      issue: 'Check the same request.',
      stopRequestId: '',
      stopLease: '',
      stopState: '',
    },
  };
}
function transport(value: unknown) {
  return {
    request: vi.fn(async (path: string) =>
      path === '/v1/capabilities' ? capabilities : result(value),
    ),
  };
}
describe('continuation native/Main boundary', () => {
  it('accepts bounded saved evidence and rejects contradictory meaning or private fields', () => {
    validateContinuation(review());
    validateContinuation(accepted());
    expect(() => parse(ContinuationSchema, { ...review(), workspacePath: '/private' })).toThrow();
    for (const change of [
      (v: Continuation) => {
        v.review!.workspaceId = 'req_elsewhere';
      },
      (v: Continuation) => {
        v.review!.steps[1]!.plannedAttempt = 1;
      },
      (v: Continuation) => {
        v.review!.steps[1]!.taskId = 'trim';
      },
      (v: Continuation) => {
        v.review!.files[0]!.path = '../outside';
      },
      (v: Continuation) => {
        v.state = 'accepted';
      },
    ]) {
      const value = review();
      change(value);
      expect(() => validateContinuation(value)).toThrow();
    }
  });
  it('restores an uncertain request by reading; refresh never resends confirmation', async () => {
    const saved = accepted(),
      connection = transport([saved]),
      service = new RunContinuationService(connection);
    expect((await service.list({ projectId: 'prj_one' }))[0]?.operation?.requestId).toBe(
      'req_continue',
    );
    expect(
      connection.request.mock.calls.every(
        (c) => c[0] === '/v1/capabilities' || c[0] === '/v1/projects/prj_one/continuations',
      ),
    ).toBe(true);
    const next = transport(saved);
    await new RunContinuationService(next).refresh({
      projectId: 'prj_one',
      reviewId: 'req_review',
    });
    expect(next.request).toHaveBeenLastCalledWith(
      '/v1/projects/prj_one/continuations/req_review/refresh',
      'POST',
      {},
    );
  });
  it('requires negotiated service support before issuing any continuation operation', async () => {
    const connection = {
      request: vi.fn(async () => result({ protocolVersion: 1, queries: [], mutations: [] })),
    };
    await expect(
      new RunContinuationService(connection).confirm({
        projectId: 'prj_one',
        reviewId: 'req_review',
        requestId: 'req_continue',
        reviewDigest: hash,
      }),
    ).rejects.toThrow('does not support');
    expect(connection.request).toHaveBeenCalledTimes(1);
  });
  it('checks exact Project, Run, review and confirmation association', async () => {
    const input = {
      projectId: 'prj_one',
      launchReviewId: 'req_launch',
      runRef: 'run_saved',
      requestId: 'req_review',
    };
    for (const changed of [
      { projectId: 'prj_other' },
      { runRef: 'run_other' },
      { requestId: 'req_other' },
    ]) {
      await expect(
        new RunContinuationService(transport({ ...review(), ...changed })).check(input),
      ).rejects.toThrow();
    }
    const wrong = accepted();
    wrong.operation!.requestId = 'req_other';
    await expect(
      new RunContinuationService(transport(wrong)).confirm({
        projectId: 'prj_one',
        reviewId: 'req_review',
        requestId: 'req_continue',
        reviewDigest: hash,
      }),
    ).rejects.toThrow('another confirmation');
    await expect(
      new RunContinuationService(
        transport({
          ...accepted(),
          review: { ...review().review!, digest: 'sha256:' + 'd'.repeat(64) },
        }),
      ).confirm({
        projectId: 'prj_one',
        reviewId: 'req_review',
        requestId: 'req_continue',
        reviewDigest: hash,
      }),
    ).rejects.toThrow();
  });
  it('forwards only the exact confirmation fields and rejects executable parameters', async () => {
    const connection = transport(accepted()),
      service = new RunContinuationService(connection);
    const input = {
      projectId: 'prj_one',
      reviewId: 'req_review',
      requestId: 'req_continue',
      reviewDigest: hash,
    };
    await service.confirm(input);
    expect(connection.request).toHaveBeenLastCalledWith(
      '/v1/projects/prj_one/continuations/req_review/confirm',
      'POST',
      { requestId: 'req_continue', reviewDigest: hash },
    );
    await expect(service.confirm({ ...input, command: 'rerun' } as typeof input)).rejects.toThrow();
  });
  it('keeps old-engine support separate from inspection and rejects cross-Run support', async () => {
    const input = { projectId: 'prj_one', launchReviewId: 'req_launch' };
    expect(
      (
        await new RunContinuationService(
          transport({
            ...input,
            supported: false,
            reason: 'This analysis uses an earlier engine.',
          }),
        ).support(input)
      ).supported,
    ).toBe(false);
    await expect(
      new RunContinuationService(
        transport({ ...input, launchReviewId: 'req_other', supported: true, reason: '' }),
      ).support(input),
    ).rejects.toThrow();
  });
  it('rejects receipts for another review and Stop for another epoch', async () => {
    const value = accepted();
    value.operation!.state = 'admitted';
    value.operation!.receipt = {
      intent: {
        schemaVersion: 1,
        requestId: 'req_continue',
        workspaceId: 'req_launch',
        originDigest: hash,
        reviewDigest: hash,
        previousHead: hash,
      },
      intentDigest: hash,
      lease,
      snapshot: lease,
    };
    validateContinuation(value);
    const other = structuredClone(value);
    other.operation!.receipt!.intent.reviewDigest = 'sha256:' + 'd'.repeat(64);
    expect(() => validateContinuation(other)).toThrow();
    value.operation!.stopRequestId = 'req_stop';
    value.operation!.stopLease = lease;
    value.operation!.stopState = 'requested';
    await expect(
      new RunContinuationService(transport(value)).stop({
        projectId: 'prj_one',
        reviewId: 'req_review',
        requestId: 'req_stop',
        expectedLease: 'e'.repeat(32),
      }),
    ).rejects.toThrow('another execution');
  });
});
