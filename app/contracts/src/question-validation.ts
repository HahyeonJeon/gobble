import type { WorkspaceDocument } from './workspace-document';
import { isQuestion } from './question';
import { validateManifest } from './evidence-validation';
import { ContractValidationError } from './validation';

export function validateQuestions(doc: WorkspaceDocument): void {
  const fail = (message: string): never => {
    throw new ContractValidationError(message);
  };
  const questions = doc.workspace.decisions.filter(isQuestion);
  if (questions.length > 100) fail('This Project has reached its question limit.');
  const invocations = new Set<string>();
  const submissions = doc.collaboration?.submissions ?? [];
  for (const question of questions) {
    if (!question.question.trim() || question.options.some((option) => !option.trim()))
      fail('Question text must be nonempty.');
    if (invocations.has(question.invocation.identity)) fail('Question invocations must be unique.');
    invocations.add(question.invocation.identity);
    if (
      !submissions.some(
        (item) =>
          item.requestId === question.originSubmissionId && item.agentId === question.requestedBy,
      )
    )
      fail('Question has no originating Agent submission.');
    if (
      new Set(question.evidence.map((item) => item.attachmentId)).size !== question.evidence.length
    )
      fail('Question evidence identities must be unique.');
    if (
      question.evidence.filter((item) => item.representation.kind === 'image').length > 2 ||
      question.evidence
        .filter((item) => item.asset.mediaType === 'application/json')
        .reduce((sum, item) => sum + item.asset.byteLength, 0) > 65536
    )
      fail('Question evidence exceeds its delivery limits.');
    for (const item of question.evidence) {
      if (item.evidence.schemaVersion === 8)
        fail('Report-linked questions are not supported. Attach reports directly to a message.');
      if (item.evidence.projectId !== doc.workspace.projectId)
        fail('Question evidence belongs to another Project.');
      validateManifest(item);
    }
    if (question.state.kind === 'answered') {
      const state = question.state;
      const answer = submissions.find((item) => item.requestId === state.submissionId);
      if (
        !answer ||
        answer.agentId !== question.requestedBy ||
        answer.replyToQuestionId !== question.decisionId
      )
        fail('Question answer must link its addressed submission.');
    }
  }
  for (const submission of submissions) {
    if (!submission.replyToQuestionId) continue;
    const question = questions.find((item) => item.decisionId === submission.replyToQuestionId);
    if (
      !question ||
      question.state.kind !== 'answered' ||
      question.state.submissionId !== submission.requestId
    )
      fail('A reply must be the question’s unique recorded answer.');
  }
  if (doc.chat.replyToQuestionId) {
    const question = questions.find((item) => item.decisionId === doc.chat.replyToQuestionId);
    if (!question || question.requestedBy !== doc.chat.recipientAgentId)
      fail('The reply recipient must be the requesting Agent.');
  }
}
