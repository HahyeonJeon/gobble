import { codexToolSchema } from './tool-schema';
import { needsToolsetRenewal, type AgentAttachment, type Submission } from '@gobble/contracts';
import type { RpcTransport } from './transport';
import { array, decodeEvent, decodeTurn, id, object, malformed } from './protocol';
import type {
  ConversationProvider,
  ProviderTurn,
  ProviderEvent,
  ProviderInput,
} from '../collaboration/provider';
import { discussionInstructions, sharedInstructions, discussionPolicy } from './policy';
import { sharedToolDefinitions } from '../shared-context/catalog';
import type { ToolHandler } from '../shared-context/ports';
import type {
  DynamicToolCallParams,
  DynamicToolCallResponse,
  DynamicToolSpec,
  ThreadResumeParams,
  ThreadStartParams,
  TurnStartParams,
} from './generated';

export class CodexConversations implements ConversationProvider {
  private readonly bindings = new Map<string, string>();
  constructor(
    private readonly rpc: RpcTransport,
    private readonly cwd: string,
  ) {}
  onTool(handler: ToolHandler): () => void {
    return (
      this.rpc.onRequest?.('item/tool/call', async (raw, signal) => {
        const value = object(raw);
        if (value.namespace !== null) throw malformed();
        const envelope = {
          threadId: id(value.threadId),
          turnId: id(value.turnId),
          callId: id(value.callId),
          namespace: null,
          tool: id(value.tool),
          arguments: value.arguments as DynamicToolCallParams['arguments'],
        } satisfies DynamicToolCallParams;
        const result = await handler(envelope, signal);
        return {
          success: result.success,
          contentItems: result.content.map((item) =>
            item.type === 'text'
              ? { type: 'inputText' as const, text: item.text }
              : { type: 'inputImage' as const, imageUrl: item.url },
          ),
        } satisfies DynamicToolCallResponse;
      }) ?? (() => {})
    );
  }
  async bind(agent: AgentAttachment): Promise<string> {
    if (!agent.configuration) throw malformed();
    const shared = agent.access === 'sharedViews';
    if (needsToolsetRenewal(agent)) throw malformed();
    const dynamicTools: DynamicToolSpec[] = shared
      ? sharedToolDefinitions.map((tool) => ({
          type: 'function',
          name: tool.name,
          description: tool.description,
          inputSchema: codexToolSchema(tool.schema),
        }))
      : [];
    const params = {
      model: agent.configuration.model,
      modelProvider: 'openai',
      cwd: this.cwd,
      sandbox: 'read-only',
      approvalPolicy: 'never',
      config: shared
        ? { ...discussionPolicy, 'features.code_mode.enabled': true }
        : discussionPolicy,
      developerInstructions:
        (shared ? sharedInstructions : discussionInstructions) +
        '\nYour name: ' +
        agent.name +
        '\nRole instructions: ' +
        agent.configuration.instructions,
    } satisfies ThreadStartParams;
    const response = object(
      await this.rpc.request(
        agent.provider.threadId ? 'thread/resume' : 'thread/start',
        agent.provider.threadId
          ? ({ ...params, threadId: agent.provider.threadId } satisfies ThreadResumeParams)
          : ({
              ...params,
              environments: [],
              dynamicTools,
              ephemeral: false,
            } satisfies ThreadStartParams),
      ),
    );
    const sandbox = object(response.sandbox);
    if (
      response.approvalPolicy !== 'never' ||
      sandbox.type !== 'readOnly' ||
      sandbox.networkAccess !== false ||
      response.model !== agent.configuration.model ||
      response.modelProvider !== 'openai'
    )
      throw malformed();
    const threadId = id(object(response.thread).id);
    if (agent.provider.threadId && threadId !== agent.provider.threadId) throw malformed();
    const owner = agent.projectId + '/' + agent.agentId;
    if (this.bindings.has(threadId) && this.bindings.get(threadId) !== owner) throw malformed();
    this.bindings.set(threadId, owner);
    return threadId;
  }
  async start(submission: Submission, evidence: ProviderInput[] = []): Promise<ProviderTurn> {
    return decodeTurn(
      object(
        await this.rpc.request('turn/start', {
          threadId: submission.threadId,
          clientUserMessageId: submission.requestId,
          input: [
            ...(submission.text
              ? [{ type: 'text' as const, text: submission.text, text_elements: [] }]
              : []),
            ...evidence.map((item) =>
              item.type === 'text'
                ? { type: 'text' as const, text: item.text, text_elements: [] }
                : { type: 'image' as const, url: item.url },
            ),
          ],
          model: submission.model,
          effort: submission.effort,
          environments: [],
          sandboxPolicy: { type: 'readOnly', networkAccess: false },
          approvalPolicy: 'never',
        } satisfies TurnStartParams),
      ).turn,
    );
  }
  async interrupt(threadId: string, turnId: string): Promise<void> {
    await this.rpc.request('turn/interrupt', { threadId, turnId });
  }
  async read(threadId: string): Promise<ProviderTurn[]> {
    const thread = object(
      object(await this.rpc.request('thread/read', { threadId, includeTurns: true })).thread,
    );
    if (thread.id !== threadId) throw malformed();
    return array(thread.turns, 2000).map(decodeTurn);
  }
  onEvent(listener: (event: ProviderEvent) => void): () => void {
    return this.rpc.onNotification((event) => {
      const decoded = decodeEvent(event.method, event.params);
      if (decoded) listener(decoded);
    });
  }
  onDisconnected(listener: () => void): () => void {
    return this.rpc.onDisconnected(listener);
  }
  disconnect(): Promise<void> {
    return this.rpc.stop();
  }
}
