import { buildRunFollowUp } from './run-followup';
import { PipelineCreationHost } from './creation-host';
import { createHash } from 'node:crypto';
import {
  parse,
  validatePipelineReviewContext,
  reviewChange,
  toolSchemas,
  type PipelineReviewContext,
  type PipelineReviewSelect,
  type PipelineInspectionInput,
  type SendMessage,
  type AgentAttachment,
  type EvidenceManifest,
} from '@gobble/contracts';
import type { ProjectService } from '../service/project-service';
import type { WorkspaceController } from '../workspace/controller';
import type { ToolContext, ToolCall } from '../shared-context/ports';
import { AppProblem } from '../problem';
const turnKey = (p: string, a: string, s: string) => `${p}/${a}/${s}`;

/** Owns only per-turn source authority and exact comparison discussion. Gobble
 * owns checked facts; the service owns current source; WorkspaceController owns saved chat and references.
 * Review references explicitly do not assert viewport observation. */
export class PipelineReviewHost {
  readonly creation: PipelineCreationHost;
  private readonly reads = new Map<string, Set<string>>();
  private readonly sourceReads = new Set<string>();
  constructor(
    private readonly workspace: WorkspaceController,
    private readonly service: ProjectService,
    private readonly verifyEvidence?: (
      projectId: string,
      manifest: EvidenceManifest,
    ) => Promise<void>,
  ) {
    this.creation = new PipelineCreationHost(workspace, service);
  }
  release(projectId: string, agentId: string, submissionId: string) {
    this.creation.release(projectId, agentId, submissionId);
    const key = turnKey(projectId, agentId, submissionId);
    this.reads.delete(key);
    this.sourceReads.delete(key);
  }
  async list(input: PipelineInspectionInput) {
    const value = await this.service.proposals.list(input);
    const doc = await this.workspace.readShared(input.projectId);
    return {
      ...value,
      marks: (doc.pipelineReviewMarks ?? []).filter(
        (m) => m.context.pipelineId === input.pipelineId,
      ),
    };
  }
  private async resolve(projectId: string, context: PipelineReviewContext) {
    validatePipelineReviewContext(context);
    if (context.preparationId) {
      const preparation = await this.service.preparations.read({
        projectId,
        pipelineId: context.pipelineId,
        requestId: context.preparationId,
      });
      if (preparation.artifactId !== context.baseArtifactId || preparation.state !== 'ready')
        throw new AppProblem('stale_revision', 'Choose an exact completed run review.');
      return {
        kind: 'preparation' as const,
        preparation,
        section: context.preparationSection ?? null,
      };
    }
    if (!context.proposalId) {
      if (context.proposedArtifactId || context.changeId || context.side)
        throw new AppProblem('invalid_request', 'A change requires an exact comparison.');
      const current = await this.service.pipelineInspection({
        projectId,
        pipelineId: context.pipelineId,
      });
      if (current.state !== 'ready' || current.artifact?.artifactId !== context.baseArtifactId)
        throw new AppProblem('stale_revision', 'The current Pipeline changed. Select it again.');
      return { kind: 'current' as const, artifact: current.artifact };
    }
    const list = await this.service.proposals.list({ projectId, pipelineId: context.pipelineId });
    const proposal = list.proposals.find((p) => p.proposalId === context.proposalId);
    if (!proposal || proposal.state !== 'ready')
      throw new AppProblem('not_found', 'The checked comparison is unavailable.');
    const change = reviewChange(proposal, context);
    return { kind: 'comparison' as const, proposal, change };
  }
  async select(input: PipelineReviewSelect) {
    if (input.context) await this.resolve(input.projectId, input.context);
    await this.workspace.selectPipelineReview(input.projectId, input.context);
    return { selected: true as const };
  }
  async prepare(input: SendMessage, agent: AgentAttachment, guard: () => void) {
    if (input.pipelineCreation || input.allowPipelineCreation)
      return this.creation.prepare(input, agent, guard);
    if (!input.pipelineReview) {
      if (input.allowPipelineProposal)
        throw new AppProblem('forbidden', 'Select a current Pipeline before allowing a proposal.');
      return [];
    }
    if (agent.access !== 'sharedViews')
      throw new AppProblem('forbidden', 'Enable shared views for Pipeline discussion.');
    const doc = await this.workspace.readShared(input.projectId);
    guard();
    if (JSON.stringify(doc.chat.pipelineReview) !== JSON.stringify(input.pipelineReview))
      throw new AppProblem(
        'stale_revision',
        'The selected discussion context changed. Your draft is preserved.',
      );
    const facts = await this.resolve(input.projectId, input.pipelineReview);
    guard();
    if (input.allowPipelineProposal) {
      if (facts.kind !== 'current' && !(facts.kind === 'preparation' && facts.preparation.fresh))
        throw new AppProblem('forbidden', 'Request a new proposal from the current Pipeline.');
      await this.service.proposals.source({
        projectId: input.projectId,
        pipelineId: input.pipelineReview.pipelineId,
        baseArtifactId: input.pipelineReview.baseArtifactId,
      });
      guard();
    }
    return [
      {
        type: 'text' as const,
        text: JSON.stringify({
          kind: 'pipeline-discussion',
          reference: input.pipelineReview,
          allowPipelineProposal: input.allowPipelineProposal === true,
          facts,
          scope:
            'Exact checked source facts. Not a viewport observation. Authoring is allowed only for this message when explicitly enabled; User alone can adopt. No Run is authorized.',
        }),
      },
    ];
  }
  async execute(context: ToolContext, call: ToolCall, guard: () => void): Promise<unknown> {
    guard();
    if (call.tool.startsWith('gobble_creation_'))
      return this.creation.execute(context, call, guard);
    const projectId = context.agent.projectId;
    const key = turnKey(projectId, context.agent.agentId, context.submission.requestId);
    if (call.tool === 'gobble_pipeline_proposals') {
      const input = parse(toolSchemas.gobble_pipeline_proposals, call.arguments);
      const list = await this.service.proposals.list({ projectId, ...input });
      guard();
      return {
        ...list,
        proposals: list.proposals.map((p) => ({
          proposalId: p.proposalId,
          state: p.state,
          summary: p.summary,
          issue: p.issue,
          context: {
            pipelineId: p.pipelineId,
            proposalId: p.proposalId,
            baseArtifactId: p.base.artifactId,
            ...(p.proposed ? { proposedArtifactId: p.proposed.artifactId } : {}),
          },
        })),
      };
    }
    if (call.tool === 'gobble_pipeline_source' || call.tool === 'gobble_pipeline_propose') {
      const scope = context.submission.pipelineReview;
      if (!context.submission.allowPipelineProposal || !scope || scope.proposalId)
        throw new AppProblem(
          'forbidden',
          'This message does not authorize a Pipeline source proposal.',
        );
      const input = {
        projectId,
        pipelineId: scope.pipelineId,
        baseArtifactId: scope.baseArtifactId,
      };
      if (call.tool === 'gobble_pipeline_source') {
        parse(toolSchemas.gobble_pipeline_source, call.arguments);
        const value = await this.service.proposals.source(input);
        guard();
        this.sourceReads.add(key);
        return value;
      }
      const candidate = parse(toolSchemas.gobble_pipeline_propose, call.arguments);
      if (!this.sourceReads.has(key))
        throw new AppProblem('forbidden', 'Read the scoped source before proposing a replacement.');
      guard();
      // Stable for this provider turn/call; late retries can only address the same
      // immutable candidate, and the service rejects changed content under this ID.
      const requestId =
        'req_' + createHash('sha256').update(`${key}/${call.turnId}/${call.callId}`).digest('hex');
      const followUp = await buildRunFollowUp(
        context,
        this.service,
        this.workspace,
        this.verifyEvidence,
        guard,
      );
      guard();
      const value = await this.service.proposals.propose({
        ...input,
        ...candidate,
        requestId,
        ...(followUp ? { followUp } : {}),
      });
      guard();
      return value;
    }
    const input =
      call.tool === 'gobble_pipeline_point'
        ? parse(toolSchemas.gobble_pipeline_point, call.arguments)
        : parse(toolSchemas.gobble_pipeline_review, call.arguments);
    const facts = await this.resolve(projectId, input.context);
    guard();
    if (facts.kind === 'preparation') {
      const identity = JSON.stringify([
        'preparation',
        facts.preparation.requestId,
        facts.preparation.prepared.digest,
      ]);
      if (call.tool === 'gobble_pipeline_review') {
        const read = this.reads.get(key) ?? new Set<string>();
        read.add(identity);
        this.reads.set(key, read);
        return { scope: 'retained-preparation', ...facts };
      }
      if (!this.reads.get(key)?.has(identity) || !input.context.preparationSection)
        throw new AppProblem(
          'forbidden',
          'Read this exact run review and choose data, settings or environment before pointing.',
        );
      const note = 'note' in input && typeof input.note === 'string' ? input.note : '';
      await this.workspace.changeShared(projectId, guard, (doc) => {
        const marks = (doc.pipelineReviewMarks ??= []);
        if (marks.length >= 64)
          throw new AppProblem(
            'unsupported',
            'The retained Pipeline reference limit has been reached.',
          );
        marks.push({
          context: input.context,
          agentId: context.agent.agentId,
          submissionId: context.submission.requestId,
          note,
        });
      });
      return { published: true, context: input.context, note, scope: 'preparation-reference' };
    }
    if (facts.kind !== 'comparison')
      throw new AppProblem('invalid_request', 'Choose an exact checked comparison.');
    const identity = JSON.stringify([
      facts.proposal.proposalId,
      facts.proposal.base.artifactId,
      facts.proposal.proposed?.artifactId,
    ]);
    if (call.tool === 'gobble_pipeline_review') {
      const read = this.reads.get(key) ?? new Set<string>();
      read.add(identity);
      this.reads.set(key, read);
      return { scope: 'retained-comparison', ...facts };
    }
    if (!this.reads.get(key)?.has(identity) || !facts.change)
      throw new AppProblem(
        'forbidden',
        'Read this comparison and select a valid change before publishing a reference.',
      );
    const note = 'note' in input && typeof input.note === 'string' ? input.note : '';
    await this.workspace.changeShared(projectId, guard, (doc) => {
      const marks = (doc.pipelineReviewMarks ??= []);
      if (marks.length >= 64)
        throw new AppProblem(
          'unsupported',
          'The retained comparison reference limit has been reached.',
        );
      marks.push({
        context: input.context,
        agentId: context.agent.agentId,
        submissionId: context.submission.requestId,
        note,
      });
    });
    return { published: true, context: input.context, note, scope: 'comparison-reference' };
  }
}
