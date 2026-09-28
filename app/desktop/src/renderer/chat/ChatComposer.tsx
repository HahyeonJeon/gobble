import { FollowUpDraftHint } from '../run-feedback/FollowUpDraftHint';
import { PreparationReferenceLabel } from './PreparationReferenceLabel';
import { CreationReferenceLabel } from './CreationReferenceLabel';
import { useEffect, useRef, useState, type Ref } from 'react';
import { isQuestion, needsToolsetRenewal, draftCaptureManifest } from '@gobble/contracts';
import { ReplyTarget } from '../questions/ReplyTarget';
import type { WorkspaceDocument } from '@gobble/contracts';
import { Icon } from '../workspace/Icon';
import { requestId, type Command } from '../workspace/useWorkspace';
import type { AgentsModel } from '../agents/useAgents';
import { useEvidence } from '../evidence/useEvidence';
import { AttachmentList } from '../evidence/AttachmentList';

type Props = {
  document: WorkspaceDocument;
  command: Command;
  saving: boolean;
  busy: boolean;
  updateDraft: (projectId: string, value: string) => Promise<boolean>;
  agents: AgentsModel;
  draftRef: Ref<HTMLTextAreaElement>;
  pendingRecipient: string | null;
  onRecipient: (agentId: string | null) => Promise<boolean>;
};

/** One composer owns the unsent text for this Project, including while its region is hidden. */
export function ChatComposer({
  document,
  command,
  saving,
  busy,
  updateDraft,
  agents,
  draftRef,
  pendingRecipient,
  onRecipient,
}: Props) {
  const [draft, setDraft] = useState(document.chat.draft);
  const [allowProposal, setAllowProposal] = useState(false);
  const [allowCreation, setAllowCreation] = useState(false);
  const creationIdentity = JSON.stringify(document.chat.pipelineCreation);
  useEffect(() => setAllowCreation(false), [creationIdentity]);
  useEffect(
    () => setAllowProposal(false),
    [
      document.chat.pipelineReview?.pipelineId,
      document.chat.pipelineReview?.baseArtifactId,
      document.chat.pipelineReview?.proposalId,
    ],
  );
  const [draftError, setDraftError] = useState(false);
  const [sending, setSending] = useState(false);
  const [pendingModel, setPendingModel] = useState<string | null>(null);
  const [expanded, setExpanded] = useState<string | null>(null);
  const evidence = useEvidence(document, agents.status);
  const attachments = document.chat.attachments ?? [];
  const localManifests = attachments.flatMap((item) => {
    const manifest = draftCaptureManifest(item);
    return manifest ? [manifest] : [];
  });
  const prepared = evidence.state.kind === 'ready' ? evidence.state.value : null;
  const sent = useRef<{ id: string; text: string } | null>(null);

  const reply = document.workspace.decisions
    .filter(isQuestion)
    .find((item) => item.decisionId === document.chat.replyToQuestionId);
  const projectId = document.workspace.projectId;
  const recipient = document.workspace.agents.find(
    (agent) => agent.agentId === document.chat.recipientAgentId,
  );
  const pending = document.collaboration?.submissions.some(
    (item) =>
      item.agentId === recipient?.agentId &&
      ['submitting', 'running', 'uncertain'].includes(item.state),
  );
  useEffect(() => {
    const last = sent.current;
    if (last && document.collaboration?.submissions.some((item) => item.requestId === last.id)) {
      setDraft((current) => (current === last.text ? '' : current));
      sent.current = null;
    }
  }, [document.collaboration]);

  const replyImagesUnsupported =
    !!reply?.evidence.some((item) => item.representation.kind === 'image') &&
    !agents.status?.models
      .find((item) => item.id === recipient?.configuration?.model)
      ?.inputModalities?.includes('image');
  const toolsChanged = !!recipient && needsToolsetRenewal(recipient);
  const canSend =
    (!reply || reply.state.kind === 'pending') &&
    !replyImagesUnsupported &&
    !toolsChanged &&
    !busy &&
    !sending &&
    pendingRecipient === null &&
    pendingModel === null &&
    !saving &&
    !draftError &&
    (!!draft.trim() || attachments.length > 0) &&
    (attachments.length === 0 || !!prepared) &&
    !!recipient?.configuration &&
    !pending &&
    agents.status?.account.status === 'signedIn' &&
    agents.status.account.runtime === 'ready';
  function send() {
    if (!canSend || !recipient || sent.current) return;
    const id = requestId();
    sent.current = { id, text: draft };
    setSending(true);
    agents
      .send({
        projectId,
        agentId: recipient.agentId,
        requestId: id,
        text: draft,
        ...(document.chat.pipelineReview
          ? { pipelineReview: document.chat.pipelineReview, allowPipelineProposal: allowProposal }
          : {}),
        ...(document.chat.pipelineCreation
          ? {
              pipelineCreation: document.chat.pipelineCreation,
              allowPipelineCreation: allowCreation,
            }
          : {}),
        ...(reply ? { replyToQuestionId: reply.decisionId } : {}),
        ...(prepared ? { preparedEvidenceId: prepared.preparedId } : {}),
      })
      .then((ok) => {
        if (ok) setDraft((current) => (current === draft ? '' : current));
        else if (attachments.length) evidence.retry();
      })
      .finally(() => {
        setSending(false);
        if (sent.current?.id === id) sent.current = null;
      })
      .catch(() => {});
  }
  return (
    <div className="chat-composer">
      <div className="composer-context">
        {document.chat.pipelineCreation && (
          <div className="review-context">
            <strong>
              <CreationReferenceLabel
                projectId={projectId}
                context={document.chat.pipelineCreation}
              />
            </strong>
            <button
              disabled={sending}
              onClick={() => void window.gobble.creation.select({ projectId, context: null })}
            >
              Remove
            </button>
            {document.chat.pipelineCreation.kind === 'draft' && (
              <>
                <label>
                  <input
                    type="checkbox"
                    checked={allowCreation}
                    disabled={sending || recipient?.access !== 'sharedViews'}
                    onChange={(e) => setAllowCreation(e.target.checked)}
                  />
                  Allow a new Pipeline proposal for this message
                </label>
                <small>
                  {allowCreation
                    ? 'The Agent can propose Trim + FastQC for your selected data. Review the checked flow here.'
                    : 'Discuss your goal. Enable proposals when you are ready.'}
                </small>
              </>
            )}
          </div>
        )}
        {document.chat.pipelineReview && (
          <div className="review-context">
            <strong>
              {document.chat.pipelineReview.preparationId ? (
                <PreparationReferenceLabel
                  projectId={projectId}
                  context={document.chat.pipelineReview}
                />
              ) : document.chat.pipelineReview.proposalId ? (
                'Pipeline change · ' + (document.chat.pipelineReview.side ?? 'Both versions')
              ) : (
                'Current Pipeline'
              )}
            </strong>
            <button
              disabled={sending}
              onClick={() =>
                void window.gobble.pipelineReviews.select({ projectId, context: null })
              }
            >
              Remove
            </button>
            {!document.chat.pipelineReview.proposalId && (
              <>
                <label>
                  <input
                    type="checkbox"
                    checked={allowProposal}
                    disabled={sending || recipient?.access !== 'sharedViews'}
                    onChange={(e) => setAllowProposal(e.target.checked)}
                  />
                  Allow a Pipeline proposal for this message
                </label>
                <small>
                  {allowProposal
                    ? 'The Agent can propose a supported design change. You review and adopt the result.'
                    : 'Discussion only. Your analysis remains unchanged.'}
                </small>
              </>
            )}
          </div>
        )}
        {document.chat.pipelineReview && (
          <FollowUpDraftHint
            projectId={projectId}
            attachments={attachments}
            context={document.chat.pipelineReview}
          />
        )}
        {reply && (
          <ReplyTarget
            question={reply}
            agentName={recipient?.name ?? 'Agent'}
            disabled={sending || busy}
            onCancel={() => void command({ kind: 'cancelReply' })}
            expanded={expanded}
            onExpand={setExpanded}
          />
        )}
        {attachments.length > 0 && (
          <>
            {evidence.state.kind !== 'error' && (
              <div className="attachment-status" role="status">
                {evidence.state.kind === 'ready'
                  ? `${attachments.length} attachment${attachments.length === 1 ? '' : 's'} · Ready for ${recipient?.name ?? 'recipient'}`
                  : 'Preparing attachments…'}
              </div>
            )}
            <AttachmentList
              items={attachments}
              manifests={prepared?.items ?? localManifests}
              expanded={expanded}
              onExpand={setExpanded}
              request={(attachmentId) =>
                prepared
                  ? { kind: 'prepared', projectId, preparedId: prepared.preparedId, attachmentId }
                  : localManifests.some((item) => item.attachmentId === attachmentId)
                    ? { kind: 'draft', projectId, attachmentId }
                    : null
              }
              disabled={sending || busy}
              onRemove={(attachmentId) => void command({ kind: 'detach', attachmentId })}
            />
          </>
        )}
      </div>
      {replyImagesUnsupported ? (
        <p className="composer-problem message-problem" role="status">
          This model cannot receive the question’s images. Choose a supported model.
        </p>
      ) : toolsChanged ? (
        <p className="composer-problem message-problem" role="status">
          Shared-view tools have changed. In Agent settings, start a new conversation to continue.
          Earlier messages stay in Project history.
        </p>
      ) : attachments.length > 0 && evidence.state.kind === 'error' ? (
        <div className="composer-problem message-problem attachment-status" role="status">
          {evidence.state.message}
          <button disabled={sending} onClick={evidence.retry}>
            Prepare again
          </button>
        </div>
      ) : null}
      <textarea
        ref={draftRef}
        aria-label="Message draft"
        placeholder={reply ? 'Write your reply…' : 'Write a message for your agent…'}
        value={draft}
        disabled={busy || sending}
        maxLength={16000}
        rows={3}
        onKeyDown={(event) => {
          if (
            event.key === 'Enter' &&
            !event.shiftKey &&
            !event.nativeEvent.isComposing &&
            event.keyCode !== 229
          ) {
            event.preventDefault();
            send();
          }
        }}
        onChange={(event) => {
          const value = event.currentTarget.value;
          setDraft(value);
          setDraftError(false);
          void updateDraft(projectId, value).then((ok) => {
            if (!ok) setDraftError(true);
          });
        }}
      />
      <div className="composer-controls">
        <label className="recipient">
          <span className="sr-only">Conversation recipient</span>
          <Icon name="agents" />
          <select
            aria-label="Conversation recipient"
            value={pendingRecipient ?? document.chat.recipientAgentId ?? ''}
            disabled={
              sending ||
              pendingRecipient !== null ||
              pendingModel !== null ||
              !!reply ||
              document.workspace.agents.length === 0
            }
            onChange={(event) => {
              const agentId = event.currentTarget.value;
              void onRecipient(agentId || null);
            }}
          >
            <option value="">
              {document.workspace.agents.length ? 'Choose recipient' : 'No agent connected'}
            </option>
            {document.workspace.agents.map((agent) => (
              <option key={agent.agentId} value={agent.agentId}>
                To {agent.name}
              </option>
            ))}
          </select>
        </label>
        {recipient?.configuration && (
          <label className="composer-model">
            <span className="sr-only">Message model</span>
            <select
              aria-label="Message model"
              value={pendingModel ?? recipient.configuration.model}
              disabled={
                sending ||
                pending ||
                pendingModel !== null ||
                pendingRecipient !== null ||
                agents.status?.account.status !== 'signedIn'
              }
              onChange={(event) => {
                const selected = agents.status?.models.find(
                  (item) => item.id === event.currentTarget.value,
                );
                if (selected && recipient.configuration) {
                  setPendingModel(selected.id);
                  void agents
                    .configure({
                      projectId,
                      agentId: recipient.agentId,
                      requestId: requestId(),
                      name: recipient.name,
                      configuration: {
                        ...recipient.configuration,
                        model: selected.id,
                        effort: selected.defaultEffort,
                      },
                    })
                    .finally(() => setPendingModel(null));
                }
              }}
            >
              {!agents.status?.models.some(
                (item) => item.id === recipient.configuration?.model,
              ) && (
                <option value={pendingModel ?? recipient.configuration.model}>
                  {recipient.configuration.model} · unavailable
                </option>
              )}
              {agents.status?.models.map((item) => (
                <option key={item.id} value={item.id}>
                  {item.name}
                </option>
              ))}
            </select>
          </label>
        )}

        <button className="primary-button send-button" disabled={!canSend} onClick={send}>
          {sending ? 'Sending…' : 'Send'} <Icon name="arrow" />
        </button>
      </div>
      <div className="composer-footer">
        <small>
          {draftError ? 'Draft not saved' : saving ? 'Saving draft…' : 'Draft saved locally'}
        </small>
        <small>Enter to send · Shift+Enter for a new line</small>
        {draftError && (
          <button
            className="text-button"
            onClick={() => void updateDraft(projectId, draft).then((ok) => setDraftError(!ok))}
          >
            Retry saving
          </button>
        )}
      </div>
    </div>
  );
}
