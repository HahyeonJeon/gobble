import {
  parse,
  AdoptCreationInputSchema,
  CreationAdoptionInputSchema,
  CreationAdoptionResultSchema,
  type AdoptCreationInput,
  type CreationAdoptionInput,
  CreationSourceInputSchema,
  CreationScopeResultSchema,
  CreationCandidateInputSchema,
  CreationCandidatesResultSchema,
  CreationCandidateResultSchema,
  CreationProposeInputSchema,
  CreationEnginesResultSchema,
  CreationConnectSchema,
  PipelineDraftInputSchema,
  validateCreationCandidate,
  type CreationSourceInput,
  type CreationCandidateInput,
  type CreationProposeInput,
  type PipelineDraftInput,
} from '@gobble/contracts';
import type { ProjectServiceClient } from './client';
import { AppProblem } from '../problem';
const unwrap = <T>(
  result:
    | { ok: true; value: T }
    | {
        ok: false;
        error: { code: import('@gobble/contracts').ContractError['code']; message: string };
      },
): T => {
  if (!result.ok) throw new AppProblem(result.error.code, result.error.message);
  return result.value;
};
export class PipelineCreationService {
  constructor(private readonly transport: Pick<ProjectServiceClient, 'request'>) {}
  private path(input: PipelineDraftInput) {
    return `/v1/projects/${input.projectId}/pipeline-drafts/${input.draftId}`;
  }
  private associated(input: PipelineDraftInput, value: { projectId: string; draftId: string }) {
    if (value.projectId !== input.projectId || value.draftId !== input.draftId)
      throw new AppProblem('internal', 'The service returned a different creation draft.');
  }
  async source(raw: CreationSourceInput) {
    const input = parse(CreationSourceInputSchema, raw);
    const value = unwrap(
      parse(
        CreationScopeResultSchema,
        await this.transport.request(`${this.path(input)}/source?generation=${input.generation}`),
      ),
    );
    this.associated(input, value.draft);
    if (value.draft.generation !== input.generation)
      throw new AppProblem('stale_revision', 'The draft changed.');
    return value;
  }
  async list(raw: PipelineDraftInput) {
    const input = parse(PipelineDraftInputSchema, raw);
    const values = unwrap(
      parse(
        CreationCandidatesResultSchema,
        await this.transport.request(`${this.path(input)}/candidates`),
      ),
    );
    const ids = new Set<string>();
    for (const value of values) {
      this.associated(input, value);
      validateCreationCandidate(value);
      if (ids.has(value.candidateId))
        throw new AppProblem('internal', 'Duplicate candidate identity.');
      ids.add(value.candidateId);
    }
    return values;
  }
  async read(raw: CreationCandidateInput) {
    const input = parse(CreationCandidateInputSchema, raw);
    const value = unwrap(
      parse(
        CreationCandidateResultSchema,
        await this.transport.request(`${this.path(input)}/candidates/${input.candidateId}`),
      ),
    );
    this.associated(input, value);
    if (value.candidateId !== input.candidateId)
      throw new AppProblem('internal', 'Candidate identity mismatch.');
    validateCreationCandidate(value);
    return value;
  }
  async propose(raw: CreationProposeInput) {
    const input = parse(CreationProposeInputSchema, raw);
    const { projectId, draftId, ...payload } = input;
    const value = unwrap(
      parse(
        CreationCandidateResultSchema,
        await this.transport.request(`${this.path(input)}/candidates`, 'POST', payload),
      ),
    );
    this.associated({ projectId, draftId }, value);
    if (value.candidateId !== input.requestId || value.generation !== input.expectedGeneration)
      throw new AppProblem('internal', 'Candidate scope mismatch.');
    validateCreationCandidate(value);
    return value;
  }
  async cancel(raw: CreationCandidateInput) {
    const input = parse(CreationCandidateInputSchema, raw);
    const value = unwrap(
      parse(
        CreationCandidateResultSchema,
        await this.transport.request(
          `${this.path(input)}/candidates/${input.candidateId}/cancel`,
          'POST',
          {},
        ),
      ),
    );
    this.associated(input, value);
    if (value.candidateId !== input.candidateId)
      throw new AppProblem('internal', 'Candidate identity mismatch.');
    return value;
  }
  async adopt(raw: AdoptCreationInput) {
    const input = parse(AdoptCreationInputSchema, raw);
    const { projectId, draftId, ...payload } = input;
    const value = unwrap(
      parse(
        CreationAdoptionResultSchema,
        await this.transport.request(`${this.path(input)}/adopt`, 'POST', payload),
      ),
    );
    this.associated({ projectId, draftId }, value);
    if (
      value.requestId !== input.requestId ||
      value.state !== 'adopted' ||
      value.adoption.candidateId !== input.candidateId ||
      value.adoption.artifactId !== input.artifactId ||
      value.adoption.generation !== input.expectedGeneration ||
      value.adoption.name !== input.name
    )
      throw new AppProblem(
        'internal',
        'The adoption result does not match this intent. Check its saved outcome.',
      );
    return value;
  }
  async outcome(raw: CreationAdoptionInput) {
    const input = parse(CreationAdoptionInputSchema, raw);
    const value = unwrap(
      parse(
        CreationAdoptionResultSchema,
        await this.transport.request(`${this.path(input)}/adoptions/${input.requestId}`),
      ),
    );
    this.associated(input, value);
    if (value.requestId !== input.requestId)
      throw new AppProblem('internal', 'Adoption identity mismatch.');
    return value;
  }
  async engines() {
    return unwrap(
      parse(CreationEnginesResultSchema, await this.transport.request('/v1/creation-engines')),
    );
  }
  async connect(raw: { engineId: string }) {
    const input = parse(CreationConnectSchema, raw);
    return unwrap(
      parse(
        CreationEnginesResultSchema,
        await this.transport.request('/v1/creation-runtime/connect', 'POST', input),
      ),
    );
  }
}
