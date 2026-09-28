import {
  parse,
  PreparationEnginesResultSchema,
  PipelineInspectionInputSchema,
  PreparationInputSchema,
  PreparePipelineInputSchema,
  PreparationResultSchema,
  PreparationsResultSchema,
  validatePreparation,
  type PipelinePreparation,
  type PreparationInput,
  type PreparePipelineInput,
  type PipelineInspectionInput,
} from '@gobble/contracts';
import type { ProjectServiceClient } from './client';
import { AppProblem } from '../problem';
export class RunPreparationService {
  constructor(private readonly transport: Pick<ProjectServiceClient, 'request'>) {}
  async engines() {
    const result = parse(
      PreparationEnginesResultSchema,
      await this.transport.request('/v1/preparation-engines'),
    );
    if (!result.ok) throw new AppProblem(result.error.code, result.error.message);
    return result.value;
  }
  private path(v: PipelineInspectionInput) {
    return `/v1/projects/${v.projectId}/pipelines/${v.pipelineId}/preparations`;
  }
  private associated(input: PipelineInspectionInput, v: PipelinePreparation) {
    if (v.projectId !== input.projectId || v.pipelineId !== input.pipelineId)
      throw new AppProblem('internal', 'The service returned a different Pipeline review.');
    validatePreparation(v);
  }
  async list(raw: PipelineInspectionInput) {
    const input = parse(PipelineInspectionInputSchema, raw),
      result = parse(PreparationsResultSchema, await this.transport.request(this.path(input)));
    if (!result.ok) throw new AppProblem(result.error.code, result.error.message);
    const ids = new Set<string>();
    for (const v of result.value) {
      this.associated(input, v);
      if (ids.has(v.requestId)) throw new AppProblem('internal', 'Duplicate run review identity.');
      ids.add(v.requestId);
    }
    return result.value;
  }
  private async operation(
    input: PreparationInput,
    kind: 'read' | 'prepare' | 'cancel',
    artifactId?: string,
    engineId?: string,
  ) {
    const route =
      this.path(input) +
      (kind === 'prepare' ? '' : `/${input.requestId}`) +
      (kind === 'cancel' ? '/cancel' : '');
    const result = parse(
      PreparationResultSchema,
      await this.transport.request(
        route,
        kind === 'read' ? 'GET' : 'POST',
        kind === 'prepare'
          ? { requestId: input.requestId, artifactId, ...(engineId ? { engineId } : {}) }
          : kind === 'cancel'
            ? {}
            : undefined,
      ),
    );
    if (!result.ok) throw new AppProblem(result.error.code, result.error.message);
    this.associated(input, result.value);
    if (
      result.value.requestId !== input.requestId ||
      (artifactId && result.value.artifactId !== artifactId) ||
      (engineId && result.value.engineId !== engineId)
    )
      throw new AppProblem('internal', 'The run review does not match this request.');
    return result.value;
  }
  read(raw: PreparationInput) {
    return this.operation(parse(PreparationInputSchema, raw), 'read');
  }
  cancel(raw: PreparationInput) {
    return this.operation(parse(PreparationInputSchema, raw), 'cancel');
  }
  prepare(raw: PreparePipelineInput) {
    const input = parse(PreparePipelineInputSchema, raw);
    return this.operation(input, 'prepare', input.artifactId, input.engineId);
  }
}
