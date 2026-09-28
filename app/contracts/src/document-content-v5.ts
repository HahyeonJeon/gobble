import { SurfaceV7Schema } from './surface-v7';
// Frozen storage components shared by v2–v5 readers. Keep original field order and closed shapes.
import { Type } from '@sinclair/typebox';
import {
  closed,
  TimestampSchema,
  AgentIdSchema,
  LabelSchema,
  RequestIdSchema,
  ProjectIdSchema,
  DecisionIdSchema,
} from './identity';
import { EvidenceRefV2Schema as EvidenceRefSchema } from './reference-v2';
import {
  AttachmentIdSchema,
  EvidenceHashSchema,
  EvidenceRepresentationV10Schema as EvidenceRepresentationSchema,
  EVIDENCE_IMAGE_BYTES,
  PreparedEvidenceIdSchema,
} from './evidence';
import { SharedReferenceIdSchema } from './shared-context';
import { ProviderIdSchema, SubmissionStateSchema } from './collaboration';
import { LegacyDecisionSchema as CurrentLegacyDecisionSchema } from './decision';
const LegacyDecisionSchema = Type.Object(
  {
    ...CurrentLegacyDecisionSchema.properties,
    evidence: Type.Array(EvidenceRefSchema, { minItems: 1, maxItems: 16 }),
  },
  closed,
);
import { WorkspaceSchema } from './workspace';
export const DraftAttachmentV5Schema = Type.Object(
  {
    attachmentId: AttachmentIdSchema,
    evidence: EvidenceRefSchema,
    label: Type.String({ minLength: 1, maxLength: 512 }),
    createdAt: TimestampSchema,
  },
  closed,
);

export const EvidenceManifestV5Schema = Type.Composite(
  [
    DraftAttachmentV5Schema,
    Type.Object({
      capturedAt: TimestampSchema,
      asset: Type.Object(
        {
          hash: EvidenceHashSchema,
          byteLength: Type.Integer({ minimum: 1, maximum: EVIDENCE_IMAGE_BYTES }),
          mediaType: Type.Union([Type.Literal('application/json'), Type.Literal('image/png')]),
        },
        closed,
      ),
      representation: EvidenceRepresentationSchema,
    }),
  ],
  closed,
);

export const SharedReferenceV5Schema = Type.Object(
  {
    referenceId: SharedReferenceIdSchema,
    label: Type.String({ minLength: 1, maxLength: 512 }),
    evidence: EvidenceRefSchema,
    author: Type.Union([
      Type.Object({ kind: Type.Literal('user') }, closed),
      Type.Object(
        { kind: Type.Literal('agent'), agentId: AgentIdSchema, name: LabelSchema },
        closed,
      ),
    ]),
    createdAt: TimestampSchema,
    note: Type.String({ maxLength: 500 }),
    retracted: Type.Boolean(),
    originSubmissionId: Type.Optional(RequestIdSchema),
  },
  closed,
);

export const QuestionV5Schema = Type.Object(
  {
    kind: Type.Literal('question'),
    projectId: ProjectIdSchema,
    decisionId: DecisionIdSchema,
    requestedBy: AgentIdSchema,
    question: Type.String({ minLength: 1, maxLength: 8000 }),
    options: Type.Array(Type.String({ minLength: 1, maxLength: 500 }), {
      maxItems: 6,
      uniqueItems: true,
    }),
    evidence: Type.Array(EvidenceManifestV5Schema, { minItems: 1, maxItems: 16 }),
    createdAt: TimestampSchema,
    originSubmissionId: RequestIdSchema,
    invocation: Type.Object(
      {
        identity: Type.String({ pattern: '^[a-f0-9]{64}$' }),
        digest: Type.String({ pattern: '^[a-f0-9]{64}$' }),
      },
      closed,
    ),
    state: Type.Union([
      Type.Object({ kind: Type.Literal('pending') }, closed),
      Type.Object(
        {
          kind: Type.Literal('answered'),
          submissionId: RequestIdSchema,
          answeredAt: TimestampSchema,
        },
        closed,
      ),
      Type.Object({ kind: Type.Literal('dismissed'), dismissedAt: TimestampSchema }, closed),
      Type.Object(
        {
          kind: Type.Literal('invalidated'),
          reason: Type.Literal('evidence_changed'),
          invalidatedAt: TimestampSchema,
        },
        closed,
      ),
    ]),
  },
  closed,
);

export const SubmissionV5Schema = Type.Object(
  {
    requestId: RequestIdSchema,
    agentId: AgentIdSchema,
    text: Type.String({ maxLength: 16000 }),
    evidence: Type.Optional(Type.Array(EvidenceManifestV5Schema, { minItems: 1, maxItems: 16 })),
    preparedEvidenceId: Type.Optional(PreparedEvidenceIdSchema),
    replyToQuestionId: Type.Optional(DecisionIdSchema),
    model: ProviderIdSchema,
    effort: Type.String({ minLength: 1, maxLength: 32 }),
    createdAt: TimestampSchema,
    // Absent in legacy records; null until the first response is durably received.
    responseStartedAt: Type.Optional(Type.Union([TimestampSchema, Type.Null()])),
    state: SubmissionStateSchema,
    threadId: ProviderIdSchema,
    turnId: Type.Union([ProviderIdSchema, Type.Null()]),
    response: Type.String({ maxLength: 32000 }),
    problem: Type.Union([Type.String({ maxLength: 500 }), Type.Null()]),
  },
  closed,
);

export const CollaborationHistoryV5Schema = Type.Object(
  {
    submissions: Type.Array(SubmissionV5Schema, { maxItems: 40 }),
  },
  closed,
);
export const WorkspaceV5Schema = Type.Object(
  {
    ...WorkspaceSchema.properties,
    surfaces: Type.Array(SurfaceV7Schema, { maxItems: 64 }),
    decisions: Type.Array(Type.Union([LegacyDecisionSchema, QuestionV5Schema]), { maxItems: 1000 }),
  },
  closed,
);
