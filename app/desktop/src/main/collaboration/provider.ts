import type { ToolHandler } from '../shared-context/ports';
import type { AgentAttachment, Submission } from '@gobble/contracts';

// Application-owned ports. Provider schemas and wire details stay in adapters.
export interface AccountAccess {
  require(model?: string, effort?: string): string;
}
export type ProviderInput = { type: 'text'; text: string } | { type: 'image'; url: string };
export type ProviderTurn = {
  id: string;
  status: 'inProgress' | 'completed' | 'interrupted' | 'failed';
  clientIds: string[];
  messages: { id: string; text: string }[];
};
export type ProviderEvent =
  | { kind: 'turn'; threadId: string; turn: ProviderTurn }
  | { kind: 'delta'; threadId: string; turnId: string; itemId: string; text: string }
  | { kind: 'message'; threadId: string; turnId: string; itemId: string; text: string };
export interface ConversationProvider {
  onTool?(handler: ToolHandler): () => void;
  bind(agent: AgentAttachment): Promise<string>;
  start(submission: Submission, evidence?: ProviderInput[]): Promise<ProviderTurn>;
  interrupt(threadId: string, turnId: string): Promise<void>;
  read(threadId: string): Promise<ProviderTurn[]>;
  onEvent(listener: (event: ProviderEvent) => void): () => void;
  onDisconnected(listener: () => void): () => void;
  disconnect(): Promise<void>;
}
