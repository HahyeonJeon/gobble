import {
  parse,
  LaunchProjectInputSchema,
  LaunchReviewInputSchema,
  LaunchCheckInputSchema,
  LaunchStartInputSchema,
  LaunchStopInputSchema,
  LaunchResultSchema,
  LaunchReviewsResultSchema,
  validateLaunchReview,
  type LaunchReview,
  type LaunchReviewInput,
  type LaunchCheckInput,
  type LaunchStartInput,
  type LaunchStopInput,
  type ProjectInput,
} from '@gobble/contracts';
import type { ProjectServiceClient } from './client';
import { AppProblem } from '../problem';
/** Trusted native-service transport. No Agent tool calls this execution API. */
export class RunLaunchService {
  constructor(private readonly transport: Pick<ProjectServiceClient, 'request'>) {}
  private path(project: string) {
    return `/v1/projects/${project}/launch-reviews`;
  }
  private associated(v: LaunchReview, project: string, review?: string) {
    validateLaunchReview(v);
    if (v.projectId !== project || (review && v.requestId !== review))
      throw new AppProblem('internal', 'The launch response belongs to another review.');
  }
  async list(raw: ProjectInput) {
    const input = parse(LaunchProjectInputSchema, raw),
      result = parse(
        LaunchReviewsResultSchema,
        await this.transport.request(this.path(input.projectId)),
      );
    if (!result.ok) throw new AppProblem(result.error.code, result.error.message);
    const ids = new Set<string>();
    for (const v of result.value) {
      this.associated(v, input.projectId);
      if (ids.has(v.requestId)) throw new AppProblem('internal', 'Duplicate launch review.');
      ids.add(v.requestId);
    }
    return result.value;
  }
  private async operation(
    v: LaunchReviewInput,
    verb: 'read' | 'cancel' | 'start' | 'refresh' | 'stop',
    body?: unknown,
  ) {
    const result = parse(
      LaunchResultSchema,
      await this.transport.request(
        this.path(v.projectId) + `/${v.reviewId}` + (verb === 'read' ? '' : `/${verb}`),
        verb === 'read' ? 'GET' : 'POST',
        verb === 'read' ? undefined : (body ?? {}),
      ),
    );
    if (!result.ok) throw new AppProblem(result.error.code, result.error.message);
    this.associated(result.value, v.projectId, v.reviewId);
    return result.value;
  }
  read(v: LaunchReviewInput) {
    return this.operation(parse(LaunchReviewInputSchema, v), 'read');
  }
  cancel(v: LaunchReviewInput) {
    return this.operation(parse(LaunchReviewInputSchema, v), 'cancel');
  }
  refresh(v: LaunchReviewInput) {
    return this.operation(parse(LaunchReviewInputSchema, v), 'refresh');
  }
  async start(raw: LaunchStartInput) {
    const v = parse(LaunchStartInputSchema, raw),
      result = await this.operation(v, 'start', { requestId: v.requestId });
    if (result.operation?.requestId !== v.requestId)
      throw new AppProblem('internal', 'Start acknowledgement belongs to another request.');
    return result;
  }
  async stop(raw: LaunchStopInput) {
    const v = parse(LaunchStopInputSchema, raw),
      result = await this.operation(v, 'stop', {
        requestId: v.requestId,
        expectedLease: v.expectedLease,
      });
    if (
      result.operation?.stopRequestId !== v.requestId ||
      result.operation.stopLease !== v.expectedLease
    )
      throw new AppProblem('internal', 'Stop acknowledgement belongs to another request.');
    return result;
  }
  async check(raw: LaunchCheckInput) {
    const v = parse(LaunchCheckInputSchema, raw),
      result = parse(
        LaunchResultSchema,
        await this.transport.request(
          `/v1/projects/${v.projectId}/pipelines/${v.pipelineId}/launch-reviews`,
          'POST',
          { requestId: v.requestId, preparationId: v.preparationId },
        ),
      );
    if (!result.ok) throw new AppProblem(result.error.code, result.error.message);
    this.associated(result.value, v.projectId, v.requestId);
    if (result.value.pipelineId !== v.pipelineId || result.value.preparationId !== v.preparationId)
      throw new AppProblem('internal', 'The launch review belongs to another preparation.');
    return result.value;
  }
}
