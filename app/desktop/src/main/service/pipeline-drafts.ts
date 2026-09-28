import {
  parse,
  validateCreationDraft,
  CreatePipelineDraftInputSchema,
  UpdatePipelineDraftInputSchema,
  DiscardPipelineDraftInputSchema,
  PipelineDraftInputSchema,
  ProjectInputSchema,
  CreationDraftResultSchema,
  CreationDraftsResultSchema,
  type CreationDraft,
  type CreatePipelineDraftInput,
  type UpdatePipelineDraftInput,
  type DiscardPipelineDraftInput,
  type PipelineDraftInput,
  type ProjectInput,
} from '@gobble/contracts';
import { AppProblem } from '../problem';
import type { ProjectServiceClient } from './client';

/** Native draft storage API. Does not authorize Agent authoring or open a Pane. */
export class PipelineDraftService {
  constructor(private readonly transport: Pick<ProjectServiceClient, 'request'>) {}
  private path(projectId: string) {
    return `/v1/projects/${projectId}/pipeline-drafts`;
  }
  private associated(input: { projectId: string; draftId?: string }, draft: CreationDraft) {
    if (
      input.projectId !== draft.projectId ||
      (input.draftId !== undefined && input.draftId !== draft.draftId)
    )
      throw new AppProblem('internal', 'The service returned a different creation draft.');
    validateCreationDraft(draft);
    return draft;
  }
  private async result(
    input: { projectId: string; draftId?: string },
    path: string,
    payload?: unknown,
  ) {
    const result = parse(
      CreationDraftResultSchema,
      await this.transport.request(path, payload === undefined ? 'GET' : 'POST', payload),
    );
    if (!result.ok) throw new AppProblem(result.error.code, result.error.message);
    return this.associated(input, result.value);
  }
  async list(raw: ProjectInput) {
    const input = parse(ProjectInputSchema, raw);
    const result = parse(
      CreationDraftsResultSchema,
      await this.transport.request(this.path(input.projectId)),
    );
    if (!result.ok) throw new AppProblem(result.error.code, result.error.message);
    if (result.value.projectId !== input.projectId)
      throw new AppProblem('internal', 'The service returned another Project.');
    const ids = new Set<string>();
    for (const draft of result.value.drafts) {
      this.associated(input, draft);
      if (ids.has(draft.draftId) || draft.state !== 'draft')
        throw new AppProblem('internal', 'The active draft list is inconsistent.');
      ids.add(draft.draftId);
    }
    return result.value;
  }
  async read(raw: PipelineDraftInput) {
    const input = parse(PipelineDraftInputSchema, raw);
    return this.result(input, `${this.path(input.projectId)}/${input.draftId}`);
  }
  async create(raw: CreatePipelineDraftInput) {
    const input = parse(CreatePipelineDraftInputSchema, raw);
    const { projectId, ...payload } = input;
    return this.result(input, this.path(projectId), payload);
  }
  async update(raw: UpdatePipelineDraftInput) {
    const input = parse(UpdatePipelineDraftInputSchema, raw);
    const { projectId, draftId, ...payload } = input;
    return this.result(input, `${this.path(projectId)}/${draftId}/update`, payload);
  }
  async discard(raw: DiscardPipelineDraftInput) {
    const input = parse(DiscardPipelineDraftInputSchema, raw);
    const { projectId, draftId, ...payload } = input;
    return this.result(input, `${this.path(projectId)}/${draftId}/discard`, payload);
  }
}
