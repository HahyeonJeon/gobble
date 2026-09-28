import type { Question, WorkspaceDocument } from '@gobble/contracts';
import type { Command } from '../workspace/useWorkspace';
import { AttachmentList } from '../evidence/AttachmentList';
import '../styles/questions.css';

export const questionStatus = (question: Question): string => {
  switch (question.state.kind) {
    case 'pending':
      return 'Awaiting your reply';
    case 'answered':
      return 'Answer recorded';
    case 'dismissed':
      return 'Dismissed';
    case 'invalidated':
      return 'Evidence changed · a new question is needed';
  }
};
export function QuestionMessage({
  question,
  document,
  command,
  expanded,
  onExpand,
}: {
  question: Question;
  document: WorkspaceDocument;
  command: Command;
  expanded: string | null;
  onExpand: (id: string | null) => void;
}) {
  const agent = document.workspace.agents.find((item) => item.agentId === question.requestedBy);
  return (
    <article
      className="question-message agent-answer"
      aria-label={'Question from ' + (agent?.name ?? 'Agent')}
    >
      <header>
        <span className="avatar" aria-hidden="true">
          {agent?.name.slice(0, 1) ?? 'A'}
        </span>
        <strong>{agent?.name ?? 'Agent'}</strong>
      </header>
      <p className="message-text">{question.question}</p>
      {question.options.length > 0 && (
        <p className="question-options">
          <span>Suggestions: </span>
          {question.options.join(' · ')}
        </p>
      )}
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
      <div className="question-actions">
        <small>{questionStatus(question)}</small>
        {question.state.kind === 'pending' && (
          <>
            <button
              onClick={() => void command({ kind: 'reply', questionId: question.decisionId })}
            >
              Reply
            </button>
            <button
              onClick={() =>
                void command({ kind: 'dismissQuestion', questionId: question.decisionId })
              }
            >
              Dismiss
            </button>
          </>
        )}
      </div>
    </article>
  );
}
