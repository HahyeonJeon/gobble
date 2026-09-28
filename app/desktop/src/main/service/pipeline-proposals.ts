import {
  parse,
  PipelineProposalResultSchema,
  PipelineProposalListResultSchema,
  PipelineProposalSourceResultSchema,
  PipelineAdoptionResultSchema,
  validatePipelineProposal,
  type PipelineInspectionInput,
  type PipelineProposeInput,
  type PipelineProposalSourceInput,
  type PipelineAdoptInput,
  type PipelineAdoptionInput,
} from '@gobble/contracts';
import { AppProblem } from '../problem';
import type { ProjectServiceClient } from './client';

/** Typed wire association for the source/review lifecycle. No renderer source API. */
export class PipelineProposalService {
  constructor(private readonly transport: Pick<ProjectServiceClient, 'request'>) {}
  private path(input: PipelineInspectionInput, operation: string) {
    return `/v1/projects/${input.projectId}/pipelines/${input.pipelineId}/${operation}`;
  }
  private associated<T extends PipelineInspectionInput>(
    input: PipelineInspectionInput,
    value: T,
  ): T {
    if (input.projectId !== value.projectId || input.pipelineId !== value.pipelineId)
      throw new AppProblem('internal', 'The service returned another Pipeline.');
    return value;
  }
  async list(input: PipelineInspectionInput) {
    const result = parse(
      PipelineProposalListResultSchema,
      await this.transport.request(this.path(input, 'proposals')),
    );
    if (!result.ok) throw new AppProblem(result.error.code, result.error.message);
    const value = this.associated(input, result.value);
    const ids = new Set<string>();
    for (const p of value.proposals) {
      this.associated(input, p);
      validatePipelineProposal(p);
      if (ids.has(p.proposalId)) throw new AppProblem('internal', 'Duplicate proposal identity.');
      ids.add(p.proposalId);
    }
    return value;
  }
  async source(input: PipelineProposalSourceInput) {
    const result = parse(
      PipelineProposalSourceResultSchema,
      await this.transport.request(
        this.path(input, 'proposal-source') +
          '?' +
          new URLSearchParams({ baseArtifactId: input.baseArtifactId }),
      ),
    );
    if (!result.ok) throw new AppProblem(result.error.code, result.error.message);
    if (result.value.baseArtifactId !== input.baseArtifactId)
      throw new AppProblem('stale_revision', 'The source belongs to another checked version.');
    return this.associated(input, result.value);
  }
  async propose(input: PipelineProposeInput) {
    const { projectId, pipelineId, ...payload } = input;
    const result = parse(
      PipelineProposalResultSchema,
      await this.transport.request(this.path(input, 'proposals'), 'POST', payload),
    );
    if (!result.ok) throw new AppProblem(result.error.code, result.error.message);
    const value = this.associated(input, result.value);
    validatePipelineProposal(value);
    if (
      value.proposalId !== input.requestId ||
      value.base.artifactId !== input.baseArtifactId ||
      !!value.followUp !== !!input.followUp ||
      (input.followUp &&
        (value.followUp?.launchReviewId !== input.followUp.launchReviewId ||
          value.followUp?.submissionId !== input.followUp.submissionId ||
          JSON.stringify(value.followUp?.evidence) !== JSON.stringify(input.followUp.evidence)))
    )
      throw new AppProblem('internal', 'Proposal identity mismatch.');
    return value;
  }
  async adopt(input: PipelineAdoptInput) {
    const { projectId, pipelineId, ...payload } = input;
    const result = parse(
      PipelineAdoptionResultSchema,
      await this.transport.request(this.path(input, 'adopt'), 'POST', payload),
    );
    if (!result.ok) throw new AppProblem(result.error.code, result.error.message);
    const value = this.associated(input, result.value);
    if (
      value.requestId !== input.requestId ||
      value.proposalId !== input.proposalId ||
      value.artifactId !== input.artifactId ||
      value.state !== 'adopted'
    )
      throw new AppProblem('internal', 'Adoption outcome does not match this request.');
    return value;
  }
  async outcome(input: PipelineAdoptionInput) {
    const result = parse(
      PipelineAdoptionResultSchema,
      await this.transport.request(this.path(input, 'adoptions/' + input.requestId)),
    );
    if (!result.ok) throw new AppProblem(result.error.code, result.error.message);
    const value = this.associated(input, result.value);
    if (value.requestId !== input.requestId)
      throw new AppProblem('internal', 'Adoption operation mismatch.');
    return value;
  }
}
