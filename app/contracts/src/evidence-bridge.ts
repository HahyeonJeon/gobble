import { Type, type Static } from '@sinclair/typebox';
import {
  DecisionIdSchema,
  AgentIdSchema,
  CounterSchema,
  ProjectIdSchema,
  RequestIdSchema,
  TimestampSchema,
  closed,
} from './identity';
import {
  AttachmentIdSchema,
  EvidenceManifestSchema,
  EvidencePreviewSchema,
  PreparedEvidenceIdSchema,
} from './evidence';
import { serviceResult } from './result';

export const PrepareEvidenceSchema = Type.Object(
  {
    projectId: ProjectIdSchema,
    agentId: AgentIdSchema,
    attachmentRevision: CounterSchema,
  },
  closed,
);
export const PreparedEvidenceSchema = Type.Object(
  {
    preparedId: PreparedEvidenceIdSchema,
    projectId: ProjectIdSchema,
    agentId: AgentIdSchema,
    attachmentRevision: CounterSchema,
    model: Type.String({ minLength: 1, maxLength: 256 }),
    expiresAt: TimestampSchema,
    items: Type.Array(EvidenceManifestSchema, { minItems: 1, maxItems: 16 }),
  },
  closed,
);
export const PreviewEvidenceSchema = Type.Union([
  Type.Object(
    { projectId: ProjectIdSchema, kind: Type.Literal('draft'), attachmentId: AttachmentIdSchema },
    closed,
  ),
  Type.Object(
    {
      projectId: ProjectIdSchema,
      kind: Type.Literal('question'),
      questionId: DecisionIdSchema,
      attachmentId: AttachmentIdSchema,
    },
    closed,
  ),
  Type.Object(
    {
      projectId: ProjectIdSchema,
      kind: Type.Literal('prepared'),
      preparedId: PreparedEvidenceIdSchema,
      attachmentId: AttachmentIdSchema,
    },
    closed,
  ),
  Type.Object(
    {
      projectId: ProjectIdSchema,
      kind: Type.Literal('sent'),
      requestId: RequestIdSchema,
      attachmentId: AttachmentIdSchema,
    },
    closed,
  ),
]);
export const PreparedEvidenceResultSchema = serviceResult(PreparedEvidenceSchema);
export const EvidencePreviewResultSchema = serviceResult(EvidencePreviewSchema);
export const EVIDENCE_CHANNELS = {
  prepare: 'gobble:evidence:prepare:v1',
  preview: 'gobble:evidence:preview:v1',
} as const;
export type PrepareEvidence = Static<typeof PrepareEvidenceSchema>;
export type PreparedEvidence = Static<typeof PreparedEvidenceSchema>;
export type PreviewEvidence = Static<typeof PreviewEvidenceSchema>;
export type EvidenceBridge = Readonly<{
  evidence: Readonly<{
    prepare: (input: PrepareEvidence) => Promise<Static<typeof PreparedEvidenceResultSchema>>;
    preview: (input: PreviewEvidence) => Promise<Static<typeof EvidencePreviewResultSchema>>;
  }>;
}>;
