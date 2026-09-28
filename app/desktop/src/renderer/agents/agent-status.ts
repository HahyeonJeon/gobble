import { needsToolsetRenewal } from '@gobble/contracts';
import type { AgentAttachment, CollaborationStatus, WorkspaceDocument } from '@gobble/contracts';

/** Presentation only; delivery and compatibility remain host decisions. */
export function agentStatus(
  agent: AgentAttachment,
  account: CollaborationStatus['account'] | undefined,
  submissions: NonNullable<WorkspaceDocument['collaboration']>['submissions'],
): string {
  const pending = submissions.find(
    (item) =>
      item.agentId === agent.agentId && ['submitting', 'running', 'uncertain'].includes(item.state),
  );
  if (pending) return pending.state === 'uncertain' ? 'Check status' : 'Working';
  if (account?.runtime !== 'ready' || account.status !== 'signedIn') return 'Not connected';
  if (!agent.configuration) return 'Needs settings';
  if (agent.provider.threadId && agent.provider.accountSessionId !== account.sessionId)
    return 'Earlier sign-in';
  if (needsToolsetRenewal(agent)) return 'New conversation needed';
  return 'Ready';
}
