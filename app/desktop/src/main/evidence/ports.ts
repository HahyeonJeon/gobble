import type {
  AgentAttachment,
  DraftAttachment,
  EvidenceManifest,
  PreparedEvidence,
  PrepareEvidence,
  PreviewEvidence,
  EvidencePreview,
  SendMessage,
} from '@gobble/contracts';
import type { ProviderInput } from '../collaboration/provider';

export type EvidenceDraft = {
  rendererSessionId: string;
  attachmentRevision: number;
  attachments: DraftAttachment[];
  agent: AgentAttachment | null;
};
export interface EvidenceWorkspace {
  assertEvidenceSession(projectId: string, rendererSessionId: string): void;
  readEvidenceDraft(projectId: string): Promise<EvidenceDraft>;
  readQuestionEvidence?(
    projectId: string,
    questionId: string,
    attachmentId: string,
  ): Promise<EvidenceManifest>;
  readSentEvidence(
    projectId: string,
    requestId: string,
    attachmentId: string,
  ): Promise<EvidenceManifest>;
}
export interface EvidenceAccount {
  require(model?: string, effort?: string): string;
  supportsImages(model: string): boolean;
}
export type DraftConsumption = {
  text: string;
  agentId: string;
  attachmentRevision: number;
  attachmentIds: string[];
  assertCurrent: () => void;
};
export type EvidenceDelivery = {
  manifests: EvidenceManifest[];
  input: ProviderInput[];
  consumption: DraftConsumption;
};
export interface AddressedEvidence {
  prepare(input: PrepareEvidence): Promise<PreparedEvidence>;
  preview(input: PreviewEvidence): Promise<EvidencePreview>;
  accept(
    input: SendMessage,
    agent: AgentAttachment,
    sessionId: string,
    assertCurrent: () => void,
  ): Promise<EvidenceDelivery>;
  consume(preparedId: string): void;
  stop(): void;
}
