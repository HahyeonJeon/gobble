import { PreparationReferenceLabel } from './PreparationReferenceLabel';
import { CreationReferenceLabel } from './CreationReferenceLabel';
import type { Submission, WorkspaceDocument } from '@gobble/contracts';
import type { AgentsModel } from '../agents/useAgents';
import { AttachmentList } from '../evidence/AttachmentList';

/** One submission remains the source of input, response, evidence and delivery controls. */
export function SubmissionMessage({
  item,
  part,
  document,
  model,
  expanded,
  onExpand,
  onReveal,
}: {
  item: Submission;
  part: 'submission' | 'response';
  document: WorkspaceDocument;
  model: AgentsModel;
  expanded: string | null;
  onExpand: (id: string | null) => void;
  onReveal: (id: string) => void;
}) {
  const agent = document.workspace.agents.find((agent) => agent.agentId === item.agentId);
  const stream = model.status?.streams.find(
    (stream) =>
      stream.projectId === document.workspace.projectId && stream.requestId === item.requestId,
  );
  const pending = ['submitting', 'running', 'uncertain'].includes(item.state);
  return (
    <article className={part === 'submission' ? 'message-exchange' : 'chat-agent-response'}>
      {part === 'submission' && (
        <div className="chat-user-message">
          <header>
            <strong>You → {agent?.name ?? 'Agent'}</strong>
            <time dateTime={new Date(item.createdAt).toISOString()}>
              {new Intl.DateTimeFormat('en', { hour: '2-digit', minute: '2-digit' }).format(
                item.createdAt,
              )}
            </time>
          </header>
          {item.replyToQuestionId && (
            <blockquote className="reply-history">
              Reply to:{' '}
              {
                document.workspace.decisions.find(
                  (question) => question.decisionId === item.replyToQuestionId,
                )?.question
              }
            </blockquote>
          )}
          {item.pipelineCreation && (
            <div className="review-context">
              <span>
                <CreationReferenceLabel
                  projectId={document.workspace.projectId}
                  context={item.pipelineCreation}
                />
                {item.allowPipelineCreation ? ' · Proposal allowed' : ' · Discussion only'}
              </span>
            </div>
          )}
          {item.pipelineReview && (
            <div className="review-context">
              <span>
                {item.pipelineReview.preparationId ? (
                  <PreparationReferenceLabel
                    projectId={document.workspace.projectId}
                    context={item.pipelineReview}
                  />
                ) : item.pipelineReview.proposalId ? (
                  'Pipeline comparison'
                ) : (
                  'Current Pipeline'
                )}
                {item.allowPipelineProposal ? ' · Proposal allowed' : ' · Discussion only'}
              </span>
              <button
                onClick={() =>
                  void window.gobble.pipelineReviews.select({
                    projectId: document.workspace.projectId,
                    context: item.pipelineReview!,
                  })
                }
              >
                Discuss this version
              </button>
            </div>
          )}
          {item.text && <p className="message-text user-bubble">{item.text}</p>}
          {!!item.evidence?.length && (
            <AttachmentList
              items={item.evidence}
              manifests={item.evidence}
              expanded={expanded}
              onExpand={onExpand}
              request={(attachmentId) => ({
                kind: 'sent',
                projectId: document.workspace.projectId,
                requestId: item.requestId,
                attachmentId,
              })}
            />
          )}
        </div>
      )}
      {(part === 'response' || item.responseStartedAt === undefined) && (
        <div className="agent-answer">
          <header>
            <span className="avatar" aria-hidden="true">
              {agent?.name.slice(0, 1) ?? 'A'}
            </span>
            <strong>{agent?.name ?? 'Agent'}</strong>
          </header>
          {part === 'response' && (
            <button
              className="response-context text-button"
              onClick={() => onReveal(item.requestId)}
            >
              In reply to: {item.text.slice(0, 80) || 'Attached evidence'}
              {item.text.length > 80 ? '…' : ''}
            </button>
          )}
          <p className="message-text">
            {stream?.text ||
              item.response ||
              (pending ? 'Waiting for response…' : 'No response text.')}
          </p>
        </div>
      )}
      {(part === 'response' || typeof item.responseStartedAt !== 'number') && (
        <>
          <div className="message-state">
            <small>
              {item.state === 'uncertain'
                ? 'Delivery uncertain'
                : pending && !stream
                  ? 'Awaiting status'
                  : item.state === 'running'
                    ? 'Responding…'
                    : item.state === 'submitting'
                      ? 'Sending…'
                      : item.state === 'abandoned'
                        ? 'Earlier delivery unconfirmed'
                        : item.state}
            </small>
            {pending && (
              <button
                onClick={() =>
                  void model.agent({
                    projectId: document.workspace.projectId,
                    agentId: item.agentId,
                    action: 'reconcile',
                  })
                }
              >
                Check status
              </button>
            )}
            {pending && item.turnId && item.state !== 'uncertain' && (
              <button
                onClick={() =>
                  void model.agent({
                    projectId: document.workspace.projectId,
                    agentId: item.agentId,
                    action: 'interrupt',
                  })
                }
              >
                Stop {agent?.name ?? 'agent'}
              </button>
            )}
          </div>
          {item.problem && <p className="message-problem">{item.problem}</p>}
        </>
      )}
    </article>
  );
}
