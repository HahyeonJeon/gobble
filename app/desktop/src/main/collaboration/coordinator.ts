import { evidenceTextBytes } from '@gobble/contracts';
import type { PipelineReviewHost } from '../pipeline-review/host';
import type { QuestionReplies } from '../questions/ports';
import { SHARED_TOOLSET, needsToolsetRenewal } from '@gobble/contracts';
import type { SharedToolExecutor, ToolCall, ToolResult } from '../shared-context/ports';
import type {
  AgentAction,
  AgentAttachment,
  ConfigureAgent,
  SendMessage,
  Submission,
} from '@gobble/contracts';
import { AppProblem } from '../problem';
import type { AccountAccess, ConversationProvider, ProviderEvent, ProviderTurn } from './provider';
import type { CollaborationStore } from './store';
import type { AddressedEvidence } from '../evidence/ports';

export const isPending = (item: Submission) =>
  ['submitting', 'running', 'uncertain'].includes(item.state);
type Active = {
  projectId: string;
  agentId: string;
  submission: Submission;
  messages: Map<string, string>;
  agent?: AgentAttachment;
  sessionId?: string;
  started?: Promise<ProviderTurn>;
  tools?: AbortController;
  queuedEvents?: number;
  overflowing?: boolean;
};
const key = (projectId: string, agentId: string) => projectId + '/' + agentId;
const uncertain =
  'Delivery is uncertain. Check status to reconcile provider history; this message will not be resent.';

export class CollaborationCoordinator {
  private readonly queues = new Map<string, Promise<unknown>>();
  private readonly active = new Map<string, Active>();
  private reservations = new Set<string>();
  private stopped = false;
  private readonly unsubscribe: (() => void)[];
  constructor(
    private readonly store: CollaborationStore,
    private readonly provider: ConversationProvider,
    private readonly account: AccountAccess,
    private readonly changed: () => void,
    private readonly report: (message: string) => void,
    private readonly sharedTools?: SharedToolExecutor,
    private readonly evidence?: AddressedEvidence,
    private readonly questions?: QuestionReplies,
    private readonly pipelineReview?: Pick<PipelineReviewHost, 'prepare'>,
  ) {
    this.unsubscribe = [
      provider.onTool?.((call, signal) => this.tool(call, signal)) ?? (() => {}),
      provider.onEvent((event) => this.receive(event)),
      provider.onDisconnected(() => {
        for (const [agentKey, active] of this.active) {
          this.revokeTools(active);
          this.schedule(agentKey, async () => {
            if (this.active.get(agentKey) === active) await this.markUncertain(active);
          }).catch(() =>
            this.report('Pending message status could not be saved. Keep the app open.'),
          );
        }
      }),
    ];
  }
  private revokeTools(active: Active): void {
    active.tools?.abort();
    this.sharedTools?.release(active.projectId, active.agentId, active.submission.requestId);
  }
  private async tool(call: ToolCall, signal: AbortSignal): Promise<ToolResult> {
    try {
      const active = [...this.active.values()].find(
        (item) => item.submission.threadId === call.threadId,
      );
      if (!active?.agent || !active.tools || !this.sharedTools)
        throw new AppProblem('forbidden', 'This conversation has no shared-view access.');
      // Wait for the authoritative turn/start response outside the per-Agent and Project queues.
      const turn = active.started
        ? await active.started
        : { id: active.submission.turnId, status: 'inProgress' };
      const combined = AbortSignal.any([signal, active.tools.signal]);
      const assert = () => {
        this.requireOpen();
        combined.throwIfAborted();
        if (
          this.active.get(key(active.projectId, active.agentId)) !== active ||
          active.agent?.access !== 'sharedViews' ||
          active.agent.provider.toolsetVersion !== SHARED_TOOLSET ||
          call.turnId !== turn.id ||
          turn.status !== 'inProgress' ||
          !['submitting', 'running'].includes(active.submission.state) ||
          this.account.require(active.submission.model, active.submission.effort) !==
            active.sessionId
        )
          throw new AppProblem('forbidden', 'This shared-tool caller is no longer authorized.');
      };
      assert();
      const result = await this.sharedTools.execute(
        { agent: active.agent, submission: active.submission, signal: combined, assert },
        call,
      );
      assert();
      result.assertCurrent?.();
      return result;
    } catch {
      return {
        success: false,
        content: [
          {
            type: 'text',
            text: '{"error":{"code":"forbidden","message":"This shared-tool caller is unknown, expired or unavailable."}}',
          },
        ],
      };
    }
  }
  streams() {
    return [...this.active.values()].map((active) => ({
      projectId: active.projectId,
      agentId: active.agentId,
      requestId: active.submission.requestId,
      text: [...active.messages.values()].join('\n\n').slice(0, 32000),
    }));
  }
  hasActive(): boolean {
    return this.reservations.size > 0 || this.active.size > 0;
  }
  private schedule<T>(agentKey: string, work: () => Promise<T>): Promise<T> {
    const result = (this.queues.get(agentKey) ?? Promise.resolve()).then(work);
    const observed = result.catch(() => {});
    this.queues.set(agentKey, observed);
    observed
      .then(() => {
        if (this.queues.get(agentKey) === observed) this.queues.delete(agentKey);
      })
      .catch(() => {});
    return result;
  }
  private requireOpen(): void {
    if (this.stopped) throw new AppProblem('runtime_unavailable', 'Agents are shutting down.');
  }
  private agent(agents: AgentAttachment[], agentId: string): AgentAttachment {
    const result = agents.find((agent) => agent.agentId === agentId);
    if (!result) throw new AppProblem('not_found', 'This agent is not attached to the Project.');
    return result;
  }
  configure(input: ConfigureAgent): Promise<void> {
    this.requireOpen();
    const agentId = input.agentId ?? 'agt_' + input.requestId.slice(4);
    return this.schedule(key(input.projectId, agentId), async () => {
      this.account.require(input.configuration.model, input.configuration.effort);
      await this.store.changeCollaboration(input.projectId, (current) => {
        if (current.history.submissions.some((item) => item.agentId === agentId && isPending(item)))
          throw new AppProblem(
            'request_conflict',
            'Finish or reconcile this agent’s pending message before changing its settings.',
          );
        const existing = current.agents.find((agent) => agent.agentId === agentId);
        if (input.agentId && !existing)
          throw new AppProblem('not_found', 'This agent is no longer attached.');
        if (!existing && current.agents.length >= 32)
          throw new AppProblem('unsupported', 'This Project has reached its agent limit.');
        if (
          existing &&
          input.access !== undefined &&
          input.access !== (existing.access ?? 'messages')
        )
          throw new AppProblem(
            'invalid_request',
            'Use the shared-view access action to change an existing conversation.',
          );
        const agent: AgentAttachment = {
          projectId: input.projectId,
          agentId,
          name: input.name.trim(),
          instructionProfile: 'discussion-v1',
          access: existing?.access ?? input.access ?? 'messages',
          configuration: input.configuration,
          provider: existing?.provider ?? { kind: 'codex', threadId: null, accountSessionId: null },
        };
        if (!agent.name) throw new AppProblem('invalid_request', 'Enter an agent name.');
        return {
          ...current,
          agents: existing
            ? current.agents.map((item) => (item.agentId === agentId ? agent : item))
            : [...current.agents, agent],
        };
      });
    });
  }
  send(input: SendMessage): Promise<void> {
    this.requireOpen();
    const agentKey = key(input.projectId, input.agentId);
    return this.schedule(agentKey, async () => {
      const current = await this.store.readCollaboration(input.projectId);
      const prior = current.history.submissions.find((item) => item.requestId === input.requestId);
      if (prior) {
        if (
          prior.agentId !== input.agentId ||
          prior.text !== input.text ||
          prior.preparedEvidenceId !== input.preparedEvidenceId ||
          prior.replyToQuestionId !== input.replyToQuestionId ||
          JSON.stringify(prior.pipelineReview) !== JSON.stringify(input.pipelineReview) ||
          prior.allowPipelineProposal !== input.allowPipelineProposal ||
          JSON.stringify(prior.pipelineCreation) !== JSON.stringify(input.pipelineCreation) ||
          prior.allowPipelineCreation !== input.allowPipelineCreation
        )
          throw new AppProblem(
            'request_conflict',
            'This submission ID belongs to another message.',
          );
        return;
      }
      const agent = this.agent(current.agents, input.agentId);
      if (needsToolsetRenewal(agent))
        throw new AppProblem(
          'unsupported',
          'Shared-view tools have changed. In Agent settings, start a new conversation to continue. Earlier messages stay in Project history.',
        );
      const config = agent.configuration;
      if (!config) throw new AppProblem('invalid_request', 'Configure this agent before sending.');
      const sessionId = this.account.require(config.model, config.effort);
      if (agent.provider.threadId && agent.provider.accountSessionId !== sessionId)
        throw new AppProblem(
          'forbidden',
          'This conversation belongs to an earlier sign-in. Start a new conversation for this agent.',
        );
      if (!input.text.trim() && !input.preparedEvidenceId)
        throw new AppProblem('invalid_request', 'Write a message before sending.');
      if (
        current.history.submissions.some(
          (item) => item.agentId === input.agentId && isPending(item),
        )
      )
        throw new AppProblem(
          'request_conflict',
          'This agent has a pending message. Stop it or check its status first.',
        );
      if (this.reservations.size >= 2)
        throw new AppProblem(
          'runtime_unavailable',
          'Two agents are working. Wait for one to finish before sending.',
        );
      if (current.history.submissions.length >= 40)
        throw new AppProblem(
          'unsupported',
          'This Project has reached the current discussion limit of 40 messages. Its history has been preserved.',
        );
      this.reservations.add(agentKey);
      let active: Active | undefined;
      try {
        const threadId = await this.provider.bind(agent);
        const assertCurrent = () => {
          this.requireOpen();
          if (this.account.require(config.model, config.effort) !== sessionId)
            throw new AppProblem('forbidden', 'Account changed before this message was sent.');
        };
        assertCurrent();
        if (input.preparedEvidenceId && !this.evidence)
          throw new AppProblem('unsupported', 'Attachment delivery is unavailable.');
        const delivery = input.preparedEvidenceId
          ? await this.evidence!.accept(input, agent, sessionId, assertCurrent)
          : undefined;
        if (input.replyToQuestionId && !this.questions)
          throw new AppProblem('unsupported', 'Question replies are unavailable.');
        const reply = input.replyToQuestionId
          ? await this.questions!.prepareReply(input, agent, assertCurrent)
          : undefined;
        if (
          (input.pipelineReview ||
            input.allowPipelineProposal ||
            input.pipelineCreation ||
            input.allowPipelineCreation) &&
          !this.pipelineReview
        )
          throw new AppProblem('unsupported', 'Pipeline discussion is unavailable.');
        const reviewInput = await this.pipelineReview?.prepare(input, agent, assertCurrent);
        const providerInput = [
          ...(reply?.input ?? []),
          ...(delivery?.input ?? []),
          ...(reviewInput ?? []),
        ];
        const combinedEvidence = [...(reply?.evidence ?? []), ...(delivery?.manifests ?? [])];
        if (
          combinedEvidence.length > 16 ||
          combinedEvidence.reduce((sum, item) => sum + evidenceTextBytes(item), 0) > 65536
        )
          throw new AppProblem(
            'unsupported',
            'The question and attachments exceed 16 items or 64 KiB of text. Remove an attachment before sending.',
          );
        const images = providerInput.filter((item) => item.type === 'image').length;
        if (
          images > 2 ||
          Buffer.byteLength(
            JSON.stringify({ input: [{ type: 'text', text: input.text }, ...providerInput] }),
          ) +
            65536 >
            4 * 1024 * 1024
        )
          throw new AppProblem(
            'unsupported',
            'The question and attachments exceed the combined delivery limit. Remove an attachment before sending.',
          );
        const submission: Submission = {
          requestId: input.requestId,
          agentId: input.agentId,
          text: input.text,
          ...(input.pipelineReview ? { pipelineReview: input.pipelineReview } : {}),
          ...(input.pipelineCreation ? { pipelineCreation: input.pipelineCreation } : {}),
          ...(input.allowPipelineCreation !== undefined
            ? { allowPipelineCreation: input.allowPipelineCreation }
            : {}),
          ...(input.allowPipelineProposal !== undefined
            ? { allowPipelineProposal: input.allowPipelineProposal }
            : {}),
          ...(input.replyToQuestionId ? { replyToQuestionId: input.replyToQuestionId } : {}),
          model: config.model,
          effort: config.effort,
          createdAt: Date.now(),
          responseStartedAt: null,
          state: 'submitting',
          threadId,
          turnId: null,
          response: '',
          problem: null,
          ...(delivery
            ? { evidence: delivery.manifests, preparedEvidenceId: input.preparedEvidenceId }
            : {}),
        };
        await this.store.changeCollaboration(
          input.projectId,
          (latest) => {
            assertCurrent();
            if (latest.history.submissions.length >= 40)
              throw new AppProblem(
                'unsupported',
                'This Project has reached its discussion limit. Its draft has been preserved.',
              );
            const recipient = this.agent(latest.agents, input.agentId);
            recipient.provider = {
              kind: 'codex',
              threadId,
              accountSessionId: sessionId,
              toolsetVersion: agent.provider.threadId
                ? (agent.provider.toolsetVersion ?? null)
                : agent.access === 'sharedViews'
                  ? SHARED_TOOLSET
                  : null,
            };
            agent.provider = recipient.provider;
            latest.history.submissions.push(submission);
            return latest;
          },
          delivery?.consumption ?? input.text,
          reply?.commit,
        );
        if (input.preparedEvidenceId) this.evidence!.consume(input.preparedEvidenceId);
        active = {
          projectId: input.projectId,
          agentId: input.agentId,
          submission,
          messages: new Map(),
          agent,
          sessionId,
          tools: new AbortController(),
        };
        this.active.set(agentKey, active);
        this.changed();
        if (agent.access === 'sharedViews')
          this.sharedTools?.begin(input.projectId, input.agentId, submission.requestId);
        active.started = this.provider.start(submission, providerInput);
        const turn = await active.started;
        await this.applyTurn(active, turn);
      } catch (error) {
        if (active) await this.markUncertain(active);
        else this.reservations.delete(agentKey);
        throw error;
      }
    });
  }
  agentAction(input: AgentAction): Promise<void> {
    this.requireOpen();
    if (input.action === 'interrupt' || input.action === 'disableSharedViews') {
      const active = this.active.get(key(input.projectId, input.agentId));
      if (active) this.revokeTools(active);
    }
    return this.schedule(key(input.projectId, input.agentId), async () => {
      const current = await this.store.readCollaboration(input.projectId);
      const agent = this.agent(current.agents, input.agentId);
      const submission = current.history.submissions.find(
        (item) => item.agentId === input.agentId && isPending(item),
      );
      if (input.action === 'disableSharedViews') {
        await this.store.changeCollaboration(input.projectId, (latest) => {
          this.agent(latest.agents, input.agentId).access = 'messages';
          return latest;
        });
        if (submission?.turnId)
          await this.provider.interrupt(submission.threadId, submission.turnId);
        return;
      }
      if (input.action === 'newConversation' || input.action === 'enableSharedViews') {
        if (
          this.active.has(key(input.projectId, input.agentId)) ||
          (submission && submission.state !== 'uncertain')
        )
          throw new AppProblem(
            'request_conflict',
            'Stop or check the pending message before starting a new conversation.',
          );
        await this.store.changeCollaboration(input.projectId, (latest) => {
          if (submission) {
            const pending = latest.history.submissions.find(
              (item) => item.requestId === submission.requestId,
            )!;
            pending.state = 'abandoned';
            pending.problem =
              'You started a new conversation. Earlier delivery remains unconfirmed and has not been resent.';
          }
          if (input.action === 'enableSharedViews')
            this.agent(latest.agents, input.agentId).access = 'sharedViews';
          this.agent(latest.agents, input.agentId).provider = {
            kind: 'codex',
            threadId: null,
            accountSessionId: null,
          };
          return latest;
        });
        return;
      }
      const sessionId = this.account.require();
      if (agent.provider.accountSessionId !== sessionId)
        throw new AppProblem(
          'forbidden',
          'Reconnect the original account to check this conversation.',
        );
      if (!submission) return;
      if (input.action === 'interrupt') {
        if (!submission.turnId)
          throw new AppProblem(
            'request_conflict',
            'Check status to identify the pending turn before stopping it.',
          );
        await this.provider.interrupt(submission.threadId, submission.turnId);
        return; // An interrupt acknowledgment is not a terminal outcome.
      }
      const turns = await this.provider.read(submission.threadId);
      const matches = turns.filter((turn) => turn.clientIds.includes(submission.requestId));
      if (matches.length !== 1 || (submission.turnId && matches[0]?.id !== submission.turnId)) {
        await this.save(input.projectId, submission.requestId, {
          state: 'uncertain',
          problem: uncertain,
        });
        return;
      }
      const turn = matches[0]!;
      const active = this.active.get(key(input.projectId, input.agentId)) ?? {
        projectId: input.projectId,
        agentId: input.agentId,
        submission,
        messages: new Map<string, string>(),
      };
      if (turn.status === 'inProgress') {
        if (!this.active.has(key(input.projectId, input.agentId)) && this.reservations.size >= 2)
          throw new AppProblem(
            'runtime_unavailable',
            'Wait for an active agent before reconnecting this turn.',
          );
        await this.provider.bind(agent);
        this.active.set(key(input.projectId, input.agentId), active);
        this.reservations.add(key(input.projectId, input.agentId));
      }
      await this.applyTurn(active, turn);
    });
  }
  private receive(event: ProviderEvent): void {
    const active = [...this.active.values()].find(
      (item) => item.submission.threadId === event.threadId,
    );
    if (!active) return;
    if (event.kind === 'turn' && event.turn.status !== 'inProgress') {
      if (active.submission.turnId === event.turn.id) this.revokeTools(active);
      else
        void active.started
          ?.then((turn) => {
            if (turn.id === event.turn.id) this.revokeTools(active);
          })
          .catch(() => {});
    }
    if (active.overflowing) return;
    if ((active.queuedEvents ?? 0) >= 256) {
      active.overflowing = true;
      this.report(
        'Codex sent updates faster than they could be saved. Reconnect and check pending message status.',
      );
      this.provider
        .disconnect()
        .catch(() => this.report('Codex could not disconnect. Keep this app open.'));
      return;
    }
    active.queuedEvents = (active.queuedEvents ?? 0) + 1;
    this.schedule(key(active.projectId, active.agentId), async () => {
      if (this.active.get(key(active.projectId, active.agentId)) !== active) return;
      const turnId = event.kind === 'turn' ? event.turn.id : event.turnId;
      if (active.submission.turnId !== turnId) return;
      if (event.kind === 'turn') {
        await this.applyTurn(active, event.turn);
        return;
      }
      const text =
        event.kind === 'delta'
          ? (active.messages.get(event.itemId) ?? '') + event.text
          : event.text;
      active.messages.set(event.itemId, text);
      const response = [...active.messages.values()].join('\n\n');
      if (response.length > 32000) {
        await this.provider.interrupt(active.submission.threadId, turnId);
        await this.markUncertain(
          active,
          'The response exceeded the display limit. Check status to recover its outcome.',
        );
        return;
      }
      if (event.kind === 'message' || (response && active.submission.responseStartedAt === null)) {
        const received = await this.save(active.projectId, active.submission.requestId, {
          response,
        });
        if (received !== undefined) active.submission.responseStartedAt = received;
      }
      this.changed();
    })
      .catch(() =>
        this.report('An agent update could not be saved. Check its status before continuing.'),
      )
      .finally(() => {
        active.queuedEvents = (active.queuedEvents ?? 1) - 1;
      })
      .catch(() => {});
  }
  private async applyTurn(active: Active, turn: ProviderTurn): Promise<void> {
    if (active.submission.turnId && active.submission.turnId !== turn.id)
      throw new AppProblem('forbidden', 'Codex returned a different turn.');
    for (const message of turn.messages) active.messages.set(message.id, message.text);
    const response = [...active.messages.values()].join('\n\n');
    if (response.length > 32000)
      throw new AppProblem('unsupported', 'This response exceeds the display limit.');
    const state = turn.status === 'inProgress' ? 'running' : turn.status;
    const update = {
      turnId: turn.id,
      state,
      response,
      problem:
        state === 'failed'
          ? 'Codex could not finish this message. Check your account before trying a new message.'
          : null,
    } satisfies Partial<Submission>;
    Object.assign(active.submission, update);
    const received = await this.save(active.projectId, active.submission.requestId, update);
    if (received !== undefined) active.submission.responseStartedAt = received;
    if (state !== 'running') this.release(active);
    this.changed();
  }
  private async save(
    projectId: string,
    requestId: string,
    update: Partial<Submission>,
  ): Promise<Submission['responseStartedAt']> {
    let responseStartedAt: Submission['responseStartedAt'];
    await this.store.changeCollaboration(projectId, (current) => {
      const item = current.history.submissions.find((item) => item.requestId === requestId);
      if (!item) throw new AppProblem('not_found', 'The saved message is unavailable.');
      if (item.responseStartedAt === null && update.response) {
        item.responseStartedAt = Math.max(
          Date.now(),
          item.createdAt,
          ...current.history.submissions.map((entry) => (entry.responseStartedAt ?? 0) + 1),
        );
      }
      Object.assign(item, update);
      responseStartedAt = item.responseStartedAt;
      return current;
    });
    return responseStartedAt;
  }
  private async markUncertain(active: Active, problem = uncertain): Promise<void> {
    this.revokeTools(active);
    active.submission.state = 'uncertain';
    try {
      await this.save(active.projectId, active.submission.requestId, {
        state: 'uncertain',
        problem,
      });
      this.release(active);
    } finally {
      this.changed();
    }
  }
  private release(active: Active): void {
    this.revokeTools(active);
    const agentKey = key(active.projectId, active.agentId);
    this.active.delete(agentKey);
    this.reservations.delete(agentKey);
  }
  async stop(): Promise<void> {
    this.stopped = true;
    for (const active of this.active.values()) this.revokeTools(active);
    await Promise.all([...this.queues.values()]);
    await Promise.allSettled(
      [...this.active.values()].map(async (active) => {
        if (active.submission.turnId)
          await this.provider
            .interrupt(active.submission.threadId, active.submission.turnId)
            .catch(() => {});
      }),
    );
    await Promise.all([...this.queues.values()]);
    try {
      for (const active of [...this.active.values()]) await this.markUncertain(active);
    } finally {
      for (const unsubscribe of this.unsubscribe) unsubscribe();
    }
  }
}
