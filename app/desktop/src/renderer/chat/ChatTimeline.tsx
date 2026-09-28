import { isQuestion } from '@gobble/contracts';
import { QuestionMessage } from '../questions/QuestionMessage';
import type { WorkspaceDocument } from '@gobble/contracts';
import { useState } from 'react';
import { SubmissionMessage } from './SubmissionMessage';
import { useChatScroll } from './useChatScroll';
import type { AgentsModel } from '../agents/useAgents';
import { Icon } from '../workspace/Icon';
import type { Command } from '../workspace/useWorkspace';
import { ReferenceEvent } from '../shared-context/ReferenceEvent';
import { chatItems, groupViewActivity, type TimelineEntry } from './timeline';

export function ChatTimeline({
  document,
  model,
  command,
}: {
  document: WorkspaceDocument;
  model: AgentsModel;
  command: Command;
}) {
  const items = chatItems(document);
  const [expandedEvidence, setExpandedEvidence] = useState<{
    requestId: string;
    attachmentId: string;
  } | null>(null);
  const version =
    items
      .map((item) =>
        item.kind === 'question'
          ? item.id + item.question.state.kind
          : item.kind === 'reference'
            ? item.id + item.reference.retracted
            : item.kind === 'activity'
              ? item.id
              : item.id + item.submission.state + item.submission.response,
      )
      .join('|') +
    model.status?.streams
      .filter((stream) => stream.projectId === document.workspace.projectId)
      .map((stream) => stream.requestId + stream.text)
      .join('|');
  const scroll = useChatScroll(version);
  function renderItem(entry: TimelineEntry) {
    if (entry.kind === 'activityGroup')
      return (
        <details className="chat-activity-group">
          <summary>{entry.items.length} view updates</summary>
          <ul>
            {entry.items.map((item) => (
              <li key={item.id}>{item.text}</li>
            ))}
          </ul>
        </details>
      );
    if (entry.kind === 'question')
      return (
        <QuestionMessage
          key={entry.id}
          question={entry.question}
          document={document}
          command={command}
          expanded={expandedEvidence?.requestId === entry.id ? expandedEvidence.attachmentId : null}
          onExpand={(attachmentId) =>
            setExpandedEvidence(attachmentId ? { requestId: entry.id, attachmentId } : null)
          }
        />
      );
    if (entry.kind === 'reference')
      return (
        <ReferenceEvent
          key={entry.id}
          reference={entry.reference}
          document={document}
          command={command}
        />
      );
    if (entry.kind === 'activity')
      return (
        <div className="chat-activity" key={entry.id}>
          <Icon name="grid" />
          <span>{entry.text}</span>
        </div>
      );
    return (
      <SubmissionMessage
        key={entry.id}
        item={entry.submission}
        part={entry.kind}
        document={document}
        model={model}
        expanded={expandedEvidence?.requestId === entry.id ? expandedEvidence.attachmentId : null}
        onExpand={(attachmentId) =>
          setExpandedEvidence(attachmentId ? { requestId: entry.id, attachmentId } : null)
        }
        onReveal={scroll.reveal}
      />
    );
  }
  return (
    <div className="chat-timeline">
      <div
        ref={scroll.list}
        className="message-list"
        aria-label="Project messages"
        tabIndex={0}
        onScroll={scroll.onScroll}
      >
        <div ref={scroll.content}>
          {!document.collaboration?.submissions.length && (
            <div className="chat-empty">
              <Icon name="chat" />
              <strong>Work together in this Project</strong>
              <p>Choose an agent and start a conversation about your work.</p>
            </div>
          )}
          {groupViewActivity(items).map((entry) => (
            <div key={entry.id} data-chat-item={entry.id} tabIndex={-1}>
              {renderItem(entry)}
            </div>
          ))}
          {document.workspace.decisions
            .filter((item) => !isQuestion(item))
            .map((decision) => (
              <article className="message-exchange" key={decision.decisionId}>
                <div className="agent-answer">
                  <strong>
                    {document.workspace.agents.find(
                      (agent) => agent.agentId === decision.requestedBy,
                    )?.name ?? 'Agent'}
                  </strong>
                  <p className="message-text">{decision.question}</p>
                  <small>{decision.state.kind}</small>
                </div>
              </article>
            ))}
        </div>
      </div>
      {scroll.unread && (
        <button className="new-messages" onClick={scroll.latest}>
          New messages <Icon name="arrow" />
        </button>
      )}
    </div>
  );
}
