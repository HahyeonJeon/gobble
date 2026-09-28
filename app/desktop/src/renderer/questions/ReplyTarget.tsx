import type { Question } from '@gobble/contracts';
import { AttachmentList } from '../evidence/AttachmentList';
import { questionStatus } from './QuestionMessage';

/** Read-only context for the single composer, with no input, options control or submit. */
export function ReplyTarget({
  question,
  agentName,
  disabled,
  onCancel,
  expanded,
  onExpand,
}: {
  question: Question;
  agentName: string;
  disabled: boolean;
  onCancel: () => void;
  expanded: string | null;
  onExpand: (id: string | null) => void;
}) {
  return (
    <div className="reply-target" role="region" aria-label="Reply target">
      <div className="reply-target-header">
        <strong>Replying to {agentName}</strong>
        <button disabled={disabled} onClick={onCancel}>
          Cancel reply
        </button>
      </div>
      <blockquote>{question.question}</blockquote>
      {question.options.length > 0 && (
        <p className="question-options">Suggestions: {question.options.join(' · ')}</p>
      )}
      {question.state.kind !== 'pending' && (
        <p role="status">
          {questionStatus(question)}. Cancel the reply to continue; your draft will be kept.
        </p>
      )}
      <small>Question evidence · included with your reply</small>
      <AttachmentList
        label="Question evidence"
        items={question.evidence}
        manifests={question.evidence}
        expanded={expanded}
        onExpand={onExpand}
        request={(attachmentId) => ({
          kind: 'question',
          projectId: question.projectId,
          questionId: question.decisionId,
          attachmentId,
        })}
      />
    </div>
  );
}
