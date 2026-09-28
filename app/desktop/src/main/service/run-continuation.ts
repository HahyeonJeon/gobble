import {
  parse,
  ContinuationProjectInputSchema,
  ContinuationSupportInputSchema,
  ContinuationInputSchema,
  ContinuationCheckInputSchema,
  ContinuationConfirmInputSchema,
  ContinuationStopInputSchema,
  ContinuationResultSchema,
  ContinuationsResultSchema,
  ContinuationSupportResultSchema,
  CapabilitiesResultSchema,
  validateContinuation,
  type Continuation,
  type ContinuationInput,
  type ContinuationCheckInput,
  type ContinuationConfirmInput,
  type ContinuationStopInput,
  type ContinuationSupportInput,
  type ProjectInput,
} from '@gobble/contracts';
import type { ProjectServiceClient } from './client';
import { AppProblem } from '../problem';

/** User-only continuation transport. Saved native records own intent restoration. */
export class RunContinuationService {
  constructor(private readonly transport: Pick<ProjectServiceClient, 'request'>) {}
  private path(project: string) {
    return `/v1/projects/${project}/continuations`;
  }
  private associated(v: Continuation, project: string, review?: string) {
    validateContinuation(v);
    if (v.projectId !== project || (review && v.requestId !== review))
      throw new AppProblem('internal', 'The continuation response belongs to another review.');
  }
  private async available() {
    const result = parse(
      CapabilitiesResultSchema,
      await this.transport.request('/v1/capabilities'),
    );
    if (!result.ok) throw new AppProblem(result.error.code, result.error.message);
    if (
      !result.value.queries.includes('continuations') ||
      !result.value.queries.includes('continuation_support') ||
      ![
        'check_continuation',
        'confirm_continuation',
        'refresh_continuation',
        'stop_continuation',
      ].every((v) => result.value.mutations.includes(v))
    )
      throw new AppProblem('unsupported', 'The native service does not support continuation.');
  }
  async support(raw: ContinuationSupportInput) {
    const input = parse(ContinuationSupportInputSchema, raw);
    await this.available();
    const result = parse(
      ContinuationSupportResultSchema,
      await this.transport.request(
        `/v1/projects/${input.projectId}/launch-reviews/${input.launchReviewId}/continuation-support`,
      ),
    );
    if (!result.ok) throw new AppProblem(result.error.code, result.error.message);
    if (
      result.value.projectId !== input.projectId ||
      result.value.launchReviewId !== input.launchReviewId
    )
      throw new AppProblem('internal', 'Engine support belongs to another Run.');
    return result.value;
  }
  /** Restoration is a read. It never resends a confirmation or dispatches work. */
  async list(raw: ProjectInput) {
    const input = parse(ContinuationProjectInputSchema, raw);
    await this.available();
    const result = parse(
      ContinuationsResultSchema,
      await this.transport.request(this.path(input.projectId)),
    );
    if (!result.ok) throw new AppProblem(result.error.code, result.error.message);
    const ids = new Set<string>();
    for (const v of result.value) {
      this.associated(v, input.projectId);
      if (ids.has(v.requestId)) throw new AppProblem('internal', 'Duplicate continuation review.');
      ids.add(v.requestId);
    }
    return result.value;
  }
  private async operation(
    input: ContinuationInput,
    verb: 'read' | 'confirm' | 'refresh' | 'stop',
    body?: unknown,
  ) {
    await this.available();
    const result = parse(
      ContinuationResultSchema,
      await this.transport.request(
        this.path(input.projectId) + `/${input.reviewId}` + (verb === 'read' ? '' : `/${verb}`),
        verb === 'read' ? 'GET' : 'POST',
        verb === 'read' ? undefined : (body ?? {}),
      ),
    );
    if (!result.ok) throw new AppProblem(result.error.code, result.error.message);
    this.associated(result.value, input.projectId, input.reviewId);
    return result.value;
  }
  read(raw: ContinuationInput) {
    return this.operation(parse(ContinuationInputSchema, raw), 'read');
  }
  refresh(raw: ContinuationInput) {
    return this.operation(parse(ContinuationInputSchema, raw), 'refresh');
  }
  async check(raw: ContinuationCheckInput) {
    const input = parse(ContinuationCheckInputSchema, raw);
    await this.available();
    const result = parse(
      ContinuationResultSchema,
      await this.transport.request(this.path(input.projectId), 'POST', {
        requestId: input.requestId,
        launchReviewId: input.launchReviewId,
        runRef: input.runRef,
      }),
    );
    if (!result.ok) throw new AppProblem(result.error.code, result.error.message);
    this.associated(result.value, input.projectId, input.requestId);
    if (
      result.value.launchReviewId !== input.launchReviewId ||
      result.value.runRef !== input.runRef
    )
      throw new AppProblem('internal', 'The review belongs to another Run.');
    return result.value;
  }
  async confirm(raw: ContinuationConfirmInput) {
    const input = parse(ContinuationConfirmInputSchema, raw);
    const result = await this.operation(input, 'confirm', {
      requestId: input.requestId,
      reviewDigest: input.reviewDigest,
    });
    if (
      result.operation?.requestId !== input.requestId ||
      result.review?.digest !== input.reviewDigest
    )
      throw new AppProblem(
        'internal',
        'Continuation acknowledgement belongs to another confirmation.',
      );
    return result;
  }
  async stop(raw: ContinuationStopInput) {
    const input = parse(ContinuationStopInputSchema, raw);
    const result = await this.operation(input, 'stop', {
      requestId: input.requestId,
      expectedLease: input.expectedLease,
    });
    if (
      result.operation?.stopRequestId !== input.requestId ||
      result.operation.stopLease !== input.expectedLease
    )
      throw new AppProblem('internal', 'Stop acknowledgement belongs to another execution.');
    return result;
  }
}
