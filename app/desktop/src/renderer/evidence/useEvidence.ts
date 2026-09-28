import { useEffect, useState } from 'react';
import type { CollaborationStatus, PreparedEvidence, WorkspaceDocument } from '@gobble/contracts';

type State =
  | { key: string; kind: 'empty' | 'preparing' | 'error'; message?: string }
  | { key: string; kind: 'ready'; value: PreparedEvidence };

/** Preparation follows attachment intent, never local selection or message typing. */
export function useEvidence(document: WorkspaceDocument, status: CollaborationStatus | null) {
  const [retry, setRetry] = useState(0);
  const recipient = document.workspace.agents.find(
    (item) => item.agentId === document.chat.recipientAgentId,
  );
  const projectId = document.workspace.projectId;
  const revision = document.chat.attachmentRevision ?? 0;
  const hasAttachments = !!document.chat.attachments?.length;
  const agentId = recipient?.agentId;
  const key = JSON.stringify([
    projectId,
    agentId,
    revision,
    recipient?.configuration,
    status?.account.sessionId,
    status?.account.status,
    status?.account.runtime,
    status?.models.find((item) => item.id === recipient?.configuration?.model)?.inputModalities,
  ]);
  const [state, setState] = useState<State>({ key, kind: 'empty' });
  useEffect(() => {
    if (!hasAttachments) {
      setState({ key, kind: 'empty' });
      return;
    }
    if (!agentId) {
      setState({ key, kind: 'error', message: 'Choose a recipient to prepare these attachments.' });
      return;
    }
    let active = true;
    let expiry: ReturnType<typeof setTimeout> | undefined;
    setState({ key, kind: 'preparing' });
    const timer = setTimeout(() => {
      void window.gobble.evidence
        .prepare({ projectId, agentId, attachmentRevision: revision })
        .then((result) => {
          if (!active) return;
          if (!result.ok) {
            setState({ key, kind: 'error', message: result.error.message });
            return;
          }
          setState({ key, kind: 'ready', value: result.value });
          expiry = setTimeout(
            () => {
              if (active)
                setState({
                  key,
                  kind: 'error',
                  message: 'The attachment preview expired. Prepare it again before sending.',
                });
            },
            Math.max(0, result.value.expiresAt - Date.now()),
          );
        })
        .catch(() => {
          if (active)
            setState({
              key,
              kind: 'error',
              message: 'Attachments could not be prepared. Try again.',
            });
        });
    }, 180);
    return () => {
      active = false;
      clearTimeout(timer);
      clearTimeout(expiry);
    };
  }, [key, projectId, agentId, revision, hasAttachments, retry]);
  const current: State =
    state.key === key ? state : { key, kind: hasAttachments ? 'preparing' : 'empty' };
  return { state: current, retry: () => setRetry((value) => value + 1) };
}
