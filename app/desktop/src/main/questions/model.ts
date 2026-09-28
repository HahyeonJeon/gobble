import { isQuestion, type Question, type WorkspaceDocument } from '@gobble/contracts';
import { AppProblem } from '../problem';
import { invalidateAttachments } from '../evidence/draft';
import type { ReplyCommit } from './ports';

export function requireQuestion(questions: Question[], id: string): Question {
  const question = questions.find((item) => item.decisionId === id);
  if (!question) throw new AppProblem('not_found', 'This question is unavailable.');
  return question;
}
export function requirePending(question: Question): void {
  if (question.state.kind !== 'pending')
    throw new AppProblem(
      'request_conflict',
      question.state.kind === 'invalidated'
        ? 'The question’s evidence changed. Your draft is preserved; cancel the reply and ask for a new question.'
        : 'This question was already answered or dismissed. Your draft is preserved; cancel the reply to continue.',
    );
}
export function selectReply(doc: WorkspaceDocument, questionId: string): void {
  const question = requireQuestion(doc.workspace.decisions.filter(isQuestion), questionId);
  requirePending(question);
  doc.chat.replyToQuestionId = questionId;
  doc.chat.recipientAgentId = question.requestedBy;
  invalidateAttachments(doc);
}
export function cancelReply(doc: WorkspaceDocument): void {
  if (doc.chat.replyToQuestionId) {
    delete doc.chat.replyToQuestionId;
    invalidateAttachments(doc);
  }
}
export function dismissQuestion(doc: WorkspaceDocument, questionId: string): void {
  const question = requireQuestion(doc.workspace.decisions.filter(isQuestion), questionId);
  if (question.state.kind === 'dismissed') return;
  requirePending(question);
  question.state = { kind: 'dismissed', dismissedAt: Date.now() };
  // Keep any reply target visible as unavailable until the user explicitly cancels it.
}
/** Called inside the sole Project writer, before draft consumption or provider input. */
export function validateReplyCommit(doc: WorkspaceDocument, reply?: ReplyCommit): void {
  if (!reply) {
    if (doc.chat.replyToQuestionId)
      throw new AppProblem(
        'invalid_request',
        'Send this draft with its reply target, or cancel the reply first.',
      );
    return;
  }
  reply.assertCurrent();
  const question = requireQuestion(doc.workspace.decisions.filter(isQuestion), reply.questionId);
  requirePending(question);
  if (
    doc.chat.replyToQuestionId !== reply.questionId ||
    doc.chat.recipientAgentId !== reply.agentId ||
    question.requestedBy !== reply.agentId ||
    doc.chat.draft !== reply.text ||
    (doc.chat.attachmentRevision ?? 0) !== reply.attachmentRevision
  )
    throw new AppProblem(
      'stale_revision',
      'The reply draft or recipient changed. Review it before sending.',
    );
}
export function recordReply(doc: WorkspaceDocument, reply: ReplyCommit): void {
  const question = requireQuestion(doc.workspace.decisions.filter(isQuestion), reply.questionId);
  question.state = { kind: 'answered', submissionId: reply.requestId, answeredAt: Date.now() };
  cancelReply(doc);
}
