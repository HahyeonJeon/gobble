import { useEffect, useRef, useState } from 'react';
import type { AgentAttachment, WorkspaceDocument } from '@gobble/contracts';
import { Icon } from '../workspace/Icon';
import { ModalDialog } from '../ui/ModalDialog';
import { AgentEditor } from './AgentEditor';
import { agentStatus } from './agent-status';
import type { AgentsModel } from './useAgents';

type Menu =
  { kind: 'closed' } | { kind: 'roster' } | { kind: 'editor'; agent: AgentAttachment | null };

export function AgentMenu({
  document,
  model,
  changingRecipient,
  onMessage,
  onFocusComposer,
}: {
  document: WorkspaceDocument;
  model: AgentsModel;
  changingRecipient: boolean;
  onMessage: (agentId: string) => Promise<boolean>;
  onFocusComposer: () => void;
}) {
  const [menu, setMenu] = useState<Menu>({ kind: 'closed' });
  const [failed, setFailed] = useState(false);
  const focusComposer = useRef(false);
  const generation = useRef(0);
  useEffect(
    () => () => {
      generation.current++;
    },
    [],
  );
  const trigger = useRef<HTMLButtonElement>(null);
  const account = model.status?.account;
  const agents = document.workspace.agents;
  const submissions = document.collaboration?.submissions ?? [];
  useEffect(() => {
    if (menu.kind === 'closed' && focusComposer.current) {
      focusComposer.current = false;
      onFocusComposer();
    }
  }, [menu, onFocusComposer]);
  function close() {
    generation.current++;
    setMenu({ kind: 'closed' });
    trigger.current?.focus();
  }
  return (
    <>
      <button
        ref={trigger}
        className="agents-trigger"
        aria-label="Project agents"
        aria-description={agents.length + ' attached to this Project'}
        aria-haspopup="dialog"
        aria-expanded={menu.kind !== 'closed'}
        onClick={() => {
          setFailed(false);
          setMenu({ kind: 'roster' });
        }}
      >
        <Icon name="agents" /> Agents <span>{agents.length}</span>
      </button>
      {menu.kind === 'roster' && (
        <ModalDialog title="Project agents" className="agent-roster" onClose={close}>
          <p className="roster-help">
            Agents share this Project chat. Message chooses your next recipient.
          </p>
          {document.chat.replyToQuestionId && (
            <p className="roster-help">
              Cancel your reply in the composer before choosing another recipient.
            </p>
          )}
          {failed && (
            <p role="alert" className="inline-error">
              Recipient could not be changed. Close this list to review the error, then try again.
            </p>
          )}
          <div className="roster-list">
            {agents.length === 0 && (
              <p className="roster-empty">
                {account?.status === 'signedIn'
                  ? 'Add an agent to this Project.'
                  : 'Sign in through Account to add agents.'}
              </p>
            )}
            {agents.map((agent) => (
              <div className="agent-row" key={agent.agentId}>
                <span className="avatar" aria-hidden="true">
                  {agent.name.slice(0, 1)}
                </span>
                <div className="agent-identity">
                  <strong>{agent.name}</strong>
                  <small>{agentStatus(agent, account, submissions)}</small>
                </div>
                <button
                  className="agent-message"
                  aria-label={'Message ' + agent.name}
                  aria-pressed={document.chat.recipientAgentId === agent.agentId}
                  disabled={changingRecipient || !!document.chat.replyToQuestionId}
                  onClick={() => {
                    setFailed(false);
                    const current = generation.current;
                    void onMessage(agent.agentId).then((ok) => {
                      if (current !== generation.current) return;
                      if (ok) {
                        focusComposer.current = true;
                        close();
                      } else setFailed(true);
                    });
                  }}
                >
                  Message
                </button>
                <button
                  className="icon-button"
                  aria-label={'Settings for ' + agent.name}
                  onClick={() => {
                    generation.current++;
                    setMenu({ kind: 'editor', agent });
                  }}
                >
                  ···
                </button>
              </div>
            ))}
          </div>
          <footer>
            <button
              disabled={account?.status !== 'signedIn' || account.runtime !== 'ready'}
              onClick={() => {
                generation.current++;
                setMenu({ kind: 'editor', agent: null });
              }}
            >
              <Icon name="plus" /> Add agent
            </button>
            <small>{agents.length} attached to this Project</small>
          </footer>
        </ModalDialog>
      )}
      {menu.kind === 'editor' && (
        <AgentEditor
          projectId={document.workspace.projectId}
          agent={menu.agent}
          model={model}
          onClose={close}
        />
      )}
    </>
  );
}
