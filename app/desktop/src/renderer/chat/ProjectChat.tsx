import { LaunchActionCards } from '../run-launch/LaunchActionCards';
import { useCallback, useEffect, useRef, useState } from 'react';
import type { WorkspaceDocument } from '@gobble/contracts';
import { Icon } from '../workspace/Icon';
import type { Command } from '../workspace/useWorkspace';
import { AgentMenu } from '../agents/AgentMenu';
import type { AgentsModel } from '../agents/useAgents';
import { ChatComposer } from './ChatComposer';
import { ChatTimeline } from './ChatTimeline';
import '../styles/chat.css';

export function ProjectChat({
  document,
  command,
  agents,
  hidden,
  compact,
  saving,
  busy,
  updateDraft,
  onCollapse,
  composerFocusRequest = 0,
}: {
  document: WorkspaceDocument;
  command: Command;
  agents: AgentsModel;
  hidden: boolean;
  compact: boolean;
  saving: boolean;
  busy: boolean;
  updateDraft: (projectId: string, value: string) => Promise<boolean>;
  onCollapse: () => void;
  composerFocusRequest?: number;
}) {
  const [pendingRecipient, setPendingRecipient] = useState<string | null>(null);
  const recipientChange = useRef(false);
  const draftRef = useRef<HTMLTextAreaElement>(null);
  const focusComposer = useCallback(() => draftRef.current?.focus(), []);
  const consumedFocus = useRef(composerFocusRequest);
  useEffect(() => {
    if (!hidden && composerFocusRequest !== consumedFocus.current) {
      consumedFocus.current = composerFocusRequest;
      focusComposer();
    }
  }, [composerFocusRequest, hidden, focusComposer]);
  async function selectRecipient(agentId: string | null): Promise<boolean> {
    if (busy || recipientChange.current || document.chat.replyToQuestionId) return false;
    recipientChange.current = true;
    setPendingRecipient(agentId ?? '');
    try {
      return await command({ kind: 'recipient', agentId });
    } finally {
      recipientChange.current = false;
      setPendingRecipient(null);
    }
  }
  const activeAgents = new Set(
    (document.collaboration?.submissions ?? [])
      .filter((item) => item.state === 'submitting' || item.state === 'running')
      .map((item) => item.agentId),
  ).size;
  const uncertain = document.collaboration?.submissions.some((item) => item.state === 'uncertain');
  return (
    <section id="project-chat" className="project-chat" aria-label="Project chat" hidden={hidden}>
      <header className="chat-header">
        <h2>Chat</h2>
        <small className="chat-working">
          {activeAgents ? activeAgents + ' working' : uncertain ? 'Check status' : ' '}
        </small>
        <AgentMenu
          document={document}
          model={agents}
          changingRecipient={pendingRecipient !== null}
          onMessage={selectRecipient}
          onFocusComposer={focusComposer}
        />
        {!compact && (
          <button
            className="icon-button"
            aria-label="Collapse chat"
            title="Collapse chat"
            aria-expanded="true"
            aria-controls="project-chat"
            onClick={onCollapse}
          >
            <Icon name="sidebar" />
          </button>
        )}
      </header>
      <ChatTimeline document={document} model={agents} command={command} />
      <LaunchActionCards key={document.workspace.projectId} document={document} command={command} />
      <ChatComposer
        document={document}
        command={command}
        saving={saving}
        busy={busy}
        updateDraft={updateDraft}
        agents={agents}
        draftRef={draftRef}
        pendingRecipient={pendingRecipient}
        onRecipient={selectRecipient}
      />
    </section>
  );
}
