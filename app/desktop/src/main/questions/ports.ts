import type {
  EvidenceManifest,
  Question,
  Submission,
  SendMessage,
  AgentAttachment,
} from '@gobble/contracts';
import type { ProviderInput } from '../collaboration/provider';
import type { EvidenceWorkspace } from '../evidence/ports';

export type ReceivedSubmission = Pick<
  Submission,
  'requestId' | 'agentId' | 'threadId' | 'state' | 'evidence' | 'replyToQuestionId'
>;
export interface QuestionWorkspace extends Pick<
  EvidenceWorkspace,
  'readEvidenceDraft' | 'assertEvidenceSession'
> {
  readQuestions(
    projectId: string,
  ): Promise<{ questions: Question[]; submissions: ReceivedSubmission[] }>;
  changeQuestions(
    projectId: string,
    guard: () => void,
    change: (questions: Question[]) => Question[],
  ): Promise<void>;
}
export type ReplyCommit = {
  questionId: string;
  requestId: string;
  agentId: string;
  text: string;
  attachmentRevision: number;
  assertCurrent: () => void;
};
export interface QuestionReplies {
  prepareReply(
    input: SendMessage,
    agent: AgentAttachment,
    assertCurrent: () => void,
  ): Promise<{ commit: ReplyCommit; input: ProviderInput[]; evidence: EvidenceManifest[] }>;
}

/** Provider-neutral tool port; transport and rendering adapters do not own question policy. */
export interface QuestionTools {
  begin(projectId: string, agentId: string, submissionId: string): void;
  release(projectId: string, agentId: string, submissionId: string): void;
  observe(
    context: import('../shared-context/ports').ToolContext,
    evidence: import('@gobble/contracts').EvidenceRef,
    data: import('@gobble/contracts').SurfaceData,
    renderImage: import('../shared-context/observation').ImageRenderer,
    assertCurrent: () => void,
    presentation?: import('@gobble/contracts').ReferencePresentation,
  ): Promise<string>;
  create(
    context: import('../shared-context/ports').ToolContext,
    call: import('../shared-context/ports').ToolCall,
    guard: () => void,
  ): Promise<{ questionId: string; state: Question['state']['kind'] }>;
}
