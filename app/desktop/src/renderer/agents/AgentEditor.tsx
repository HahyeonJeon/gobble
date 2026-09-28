import { useState } from 'react';
import { needsToolsetRenewal } from '@gobble/contracts';
import type { AgentAttachment } from '@gobble/contracts';
import { requestId } from '../workspace/useWorkspace';
import { ModalDialog } from '../ui/ModalDialog';
import type { AgentsModel } from './useAgents';

export function AgentEditor({
  projectId,
  agent,
  model,
  onClose,
}: {
  projectId: string;
  agent: AgentAttachment | null;
  model: AgentsModel;
  onClose: () => void;
}) {
  const models = model.status?.models ?? [];
  const initial =
    models.find((item) => item.id === agent?.configuration?.model) ??
    models.find((item) => item.isDefault) ??
    models[0];
  const [name, setName] = useState(agent?.name ?? '');
  const [instructions, setInstructions] = useState(agent?.configuration?.instructions ?? '');
  const [modelId, setModelId] = useState(initial?.id ?? '');
  const [effort, setEffort] = useState(
    agent?.configuration?.effort ?? initial?.defaultEffort ?? '',
  );
  const [access, setAccess] = useState<'messages' | 'sharedViews'>(agent?.access ?? 'messages');
  const [busy, setBusy] = useState(false);
  const [submissionId] = useState(requestId);
  const selected = models.find((item) => item.id === modelId);
  return (
    <ModalDialog title={agent ? 'Agent settings' : 'Add agent'} onClose={onClose}>
      <form
        onSubmit={(event) => {
          event.preventDefault();
          setBusy(true);
          model
            .configure({
              projectId,
              agentId: agent?.agentId ?? null,
              requestId: submissionId,
              name,
              access,
              configuration: { model: modelId, effort, instructions },
            })
            .then((ok) => {
              if (ok) onClose();
            })
            .finally(() => setBusy(false));
        }}
      >
        <label>
          Name
          <input
            autoFocus
            required
            maxLength={200}
            value={name}
            onChange={(event) => setName(event.currentTarget.value)}
            placeholder="e.g. Researcher"
          />
        </label>
        <label>
          Role instructions
          <textarea
            maxLength={4000}
            value={instructions}
            onChange={(event) => setInstructions(event.currentTarget.value)}
            placeholder="How should this agent help with the Project?"
            rows={4}
          />
        </label>
        <label>
          Model
          <select
            required
            value={modelId}
            onChange={(event) => {
              const next = models.find((item) => item.id === event.currentTarget.value);
              setModelId(next?.id ?? '');
              setEffort(next?.defaultEffort ?? '');
            }}
          >
            {!selected && <option value="">Connect your account to load models</option>}
            {models.map((item) => (
              <option value={item.id} key={item.id}>
                {item.name}
              </option>
            ))}
          </select>
        </label>
        <label>
          Reasoning effort
          <select value={effort} onChange={(event) => setEffort(event.currentTarget.value)}>
            {selected?.efforts.map((item) => (
              <option key={item} value={item}>
                {item}
              </option>
            ))}
          </select>
        </label>
        {!agent && (
          <label>
            Project access
            <select
              value={access}
              onChange={(event) =>
                setAccess(event.currentTarget.value as 'messages' | 'sharedViews')
              }
            >
              <option value="messages">Messages only</option>
              <option value="sharedViews">Shared views</option>
            </select>
          </label>
        )}
        {agent && (
          <div className="access-settings">
            <strong>
              {agent.access === 'sharedViews' ? 'Shared views enabled' : 'Messages only'}
            </strong>
            {needsToolsetRenewal(agent) && (
              <p className="message-problem">
                Questions need the updated shared-view tools. Use Start new conversation below. Your
                earlier messages and evidence stay in Project history.
              </p>
            )}
            <p className="muted">
              Shared views let this agent open Project resources, observe ready previews and publish
              pointers and ask evidence-backed questions in chat. Your local selections and draft
              stay private.
            </p>
            <button
              type="button"
              disabled={busy}
              onClick={() => {
                setBusy(true);
                void model
                  .agent({
                    projectId,
                    agentId: agent.agentId,
                    action:
                      agent.access === 'sharedViews' ? 'disableSharedViews' : 'enableSharedViews',
                  })
                  .then((ok) => {
                    if (ok) onClose();
                  })
                  .finally(() => setBusy(false));
              }}
            >
              {agent.access === 'sharedViews'
                ? 'Disable shared views and stop'
                : 'Enable shared views and start new conversation'}
            </button>
          </div>
        )}
        <p className="muted">
          Each agent keeps its own conversation. Shared views stay with the Project.
        </p>
        <button
          className="primary-button"
          disabled={busy || !selected || !selected.efforts.includes(effort)}
        >
          {busy ? 'Saving…' : agent ? 'Save settings' : 'Add agent'}
        </button>
        {agent && (
          <button
            type="button"
            disabled={busy}
            onClick={() => {
              setBusy(true);
              model
                .agent({ projectId, agentId: agent.agentId, action: 'newConversation' })
                .then((ok) => {
                  if (ok) onClose();
                })
                .finally(() => setBusy(false));
            }}
          >
            Start new conversation
          </button>
        )}
        {agent && (
          <small className="muted">
            Starting anew keeps the shared history and does not resend earlier messages.
          </small>
        )}
      </form>
    </ModalDialog>
  );
}
