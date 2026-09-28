import { Type, type Static } from '@sinclair/typebox';
import {
  AgentIdSchema,
  DecisionIdSchema,
  ProjectIdSchema,
  RequestIdSchema,
  TimestampSchema,
  closed,
} from './identity';
import { EvidenceManifestSchema, AttachmentIdSchema } from './evidence';

export const ObservationIdSchema = Type.String({
  pattern: '^obs_[A-Za-z0-9_-]{1,80}$',
  maxLength: 84,
});
export const QuestionInputSchema = Type.Object(
  {
    question: Type.String({ minLength: 1, maxLength: 8000 }),
    options: Type.Optional(
      Type.Array(Type.String({ minLength: 1, maxLength: 500 }), { maxItems: 6, uniqueItems: true }),
    ),
    evidenceIds: Type.Array(Type.Union([ObservationIdSchema, AttachmentIdSchema]), {
      minItems: 1,
      maxItems: 16,
      uniqueItems: true,
    }),
  },
  closed,
);
/** The answer body and delivery outcome belong to the linked Submission, not this Decision. */
export const QuestionSchema = Type.Object(
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
    evidence: Type.Array(EvidenceManifestSchema, { minItems: 1, maxItems: 16 }),
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
export type Question = Static<typeof QuestionSchema>;
export type QuestionInput = Static<typeof QuestionInputSchema>;
export const isQuestion = (value: object): value is Question =>
  'kind' in value && value.kind === 'question';
