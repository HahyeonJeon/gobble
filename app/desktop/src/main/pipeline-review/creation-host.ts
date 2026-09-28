import { createHash } from 'node:crypto';
import {
  parse,
  creationFacts,
  toolSchemas,
  type CreationContext,
  type CreationSelect,
  type SendMessage,
  type AgentAttachment,
  type PipelineDraftInput,
} from '@gobble/contracts';
import type { ProjectService } from '../service/project-service';
import type { WorkspaceController } from '../workspace/controller';
import type { ToolContext, ToolCall } from '../shared-context/ports';
import { AppProblem } from '../problem';
const turnKey = (p: string, a: string, s: string) => `${p}/${a}/${s}`;
/** Per-message creation authority and retained facts. Never adopts or owns Views. */
export class PipelineCreationHost {
  private readonly sourceReads = new Map<string, string>();
  private readonly reads = new Map<string, Set<string>>();
  constructor(
    private readonly workspace: WorkspaceController,
    private readonly service: ProjectService,
  ) {}
  release(p: string, a: string, s: string) {
    const key = turnKey(p, a, s);
    this.sourceReads.delete(key);
    this.reads.delete(key);
  }
  async state(input: PipelineDraftInput) {
    const draft = await this.service.drafts.read(input);
    const candidates = await this.service.creation.list(input);
    const doc = await this.workspace.readShared(input.projectId);
    return {
      draft,
      candidates,
      marks: (doc.creationMarks ?? []).filter((m) => m.context.draftId === input.draftId),
    };
  }
  private async resolve(projectId: string, context: CreationContext) {
    if (context.kind === 'draft') {
      const draft = await this.service.drafts.read({ projectId, draftId: context.draftId });
      if (draft.state !== 'draft' || draft.generation !== context.generation || !draft.input)
        throw new AppProblem('stale_revision', 'The draft changed. Select its data again.');
      return { kind: 'draft' as const, draft };
    }
    const candidate = await this.service.creation.read({
      projectId,
      draftId: context.draftId,
      candidateId: context.candidateId,
    });
    return { kind: 'candidate' as const, candidate, facts: creationFacts(candidate, context) };
  }
  async select(input: CreationSelect) {
    if (input.context) await this.resolve(input.projectId, input.context);
    await this.workspace.selectPipelineCreation(input.projectId, input.context);
    return { selected: true as const };
  }
  async prepare(input: SendMessage, agent: AgentAttachment, guard: () => void) {
    if (!input.pipelineCreation) {
      if (input.allowPipelineCreation)
        throw new AppProblem('forbidden', 'Select a creation draft first.');
      return [];
    }
    if (input.pipelineReview || input.allowPipelineProposal || agent.access !== 'sharedViews')
      throw new AppProblem('forbidden', 'Creation needs one shared-view discussion context.');
    const doc = await this.workspace.readShared(input.projectId);
    guard();
    if (JSON.stringify(doc.chat.pipelineCreation) !== JSON.stringify(input.pipelineCreation))
      throw new AppProblem(
        'stale_revision',
        'The selected creation reference changed. Your text is preserved.',
      );
    const facts = await this.resolve(input.projectId, input.pipelineCreation);
    guard();
    if (input.allowPipelineCreation) {
      if (input.pipelineCreation.kind !== 'draft')
        throw new AppProblem('forbidden', 'Select the active draft before requesting a proposal.');
      await this.service.creation.source({
        projectId: input.projectId,
        draftId: input.pipelineCreation.draftId,
        generation: input.pipelineCreation.generation,
      });
      guard();
    }
    return [
      {
        type: 'text' as const,
        text: JSON.stringify({
          kind: 'pipeline-creation-discussion',
          reference: input.pipelineCreation,
          allowPipelineCreation: input.allowPipelineCreation === true,
          facts,
          scope:
            'Exact retained creation facts, not viewport observation. The selected input remains bound independently of this attachment. Source authoring is allowed only with this message opt-in. No adoption or Run is authorized.',
        }),
      },
    ];
  }
  async execute(context: ToolContext, call: ToolCall, guard: () => void): Promise<unknown> {
    guard();
    const projectId = context.agent.projectId,
      key = turnKey(projectId, context.agent.agentId, context.submission.requestId),
      scope = context.submission.pipelineCreation;
    if (!scope) throw new AppProblem('forbidden', 'This message has no creation scope.');
    const base = { projectId, draftId: scope.draftId };
    if (call.tool === 'gobble_creation_source' || call.tool === 'gobble_creation_propose') {
      if (!context.submission.allowPipelineCreation || scope.kind !== 'draft')
        throw new AppProblem('forbidden', 'This message does not allow creation authoring.');
      if (call.tool === 'gobble_creation_source') {
        parse(toolSchemas.gobble_creation_source, call.arguments);
        const value = await this.service.creation.source({ ...base, generation: scope.generation });
        guard();
        this.sourceReads.set(key, value.scopeId);
        return value;
      }
      const input = parse(toolSchemas.gobble_creation_propose, call.arguments),
        scopeId = this.sourceReads.get(key);
      if (!scopeId) throw new AppProblem('forbidden', 'Read the scoped creation source first.');
      const requestId =
        'req_' + createHash('sha256').update(`${key}/${call.turnId}/${call.callId}`).digest('hex');
      const value = await this.service.creation.propose({
        ...base,
        ...input,
        requestId,
        expectedGeneration: scope.generation,
        scopeId,
      });
      guard();
      return value;
    }
    if (call.tool === 'gobble_creation_status') {
      parse(toolSchemas.gobble_creation_status, call.arguments);
      const candidates = await this.service.creation.list(base);
      guard();
      return candidates.map((c) => ({
        candidateId: c.candidateId,
        state: c.state,
        summary: c.summary,
        issue: c.issue,
        ...(c.artifact
          ? {
              context: {
                kind: 'candidate',
                base: 'none',
                draftId: c.draftId,
                generation: c.generation,
                candidateId: c.candidateId,
                artifactId: c.artifact.artifactId,
              },
            }
          : {}),
      }));
    }
    const input =
      call.tool === 'gobble_creation_point'
        ? parse(toolSchemas.gobble_creation_point, call.arguments)
        : parse(toolSchemas.gobble_creation_review, call.arguments);
    if (input.context.kind !== 'candidate' || input.context.draftId !== scope.draftId)
      throw new AppProblem('forbidden', 'Choose a checked candidate in this message’s draft.');
    const facts = await this.resolve(projectId, input.context);
    guard();
    const identity = JSON.stringify([
      input.context.draftId,
      input.context.candidateId,
      input.context.artifactId,
    ]);
    if (call.tool === 'gobble_creation_review') {
      const reads = this.reads.get(key) ?? new Set<string>();
      reads.add(identity);
      this.reads.set(key, reads);
      return { scope: 'retained-creation', ...facts };
    }
    if (!this.reads.get(key)?.has(identity) || !input.context.target)
      throw new AppProblem(
        'forbidden',
        'Read the checked candidate and choose an exact addition first.',
      );
    const note = 'note' in input && typeof input.note === 'string' ? input.note : '';
    await this.workspace.changeShared(projectId, guard, (doc) => {
      const marks = (doc.creationMarks ??= []);
      if (marks.length >= 64)
        throw new AppProblem('unsupported', 'The creation reference limit has been reached.');
      marks.push({
        context: input.context,
        agentId: context.agent.agentId,
        submissionId: context.submission.requestId,
        note,
      });
    });
    return { published: true, context: input.context, note, scope: 'creation-reference' };
  }
}
