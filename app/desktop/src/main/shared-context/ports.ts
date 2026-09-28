import type { AgentAttachment, Submission, ToolInputSchema } from '@gobble/contracts';

/** Provider envelopes contain routing identity; model arguments never choose the caller. */
export type ToolCall = {
  threadId: string;
  turnId: string;
  callId: string;
  tool: string;
  arguments: unknown;
};
export type ToolResult = {
  assertCurrent?: () => void;
  success: boolean;
  content: ({ type: 'text'; text: string } | { type: 'image'; url: string })[];
};
export type ToolContext = {
  agent: AgentAttachment;
  submission: Submission;
  signal: AbortSignal;
  assert: () => void;
};
export type ToolDefinition = { name: string; description: string; schema: ToolInputSchema };
export interface SharedToolExecutor {
  begin(projectId: string, agentId: string, submissionId: string): void;
  execute(context: ToolContext, call: ToolCall): Promise<ToolResult>;
  release(projectId: string, agentId: string, submissionId: string): void;
}
export type ToolHandler = (call: ToolCall, signal: AbortSignal) => Promise<ToolResult>;
export const textResult = (value: unknown): ToolResult => ({
  success: true,
  content: [{ type: 'text', text: JSON.stringify(value) }],
});
