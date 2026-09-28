import { Type, type Static } from '@sinclair/typebox';
import { QuestionSchema } from './question';
import { EvidenceRefV3Schema as EvidenceRefSchema } from './reference-v3';
import {
  AgentIdSchema,
  closed,
  DecisionIdSchema,
  ProjectIdSchema,
  TimestampSchema,
} from './identity';

export const LegacyDecisionSchema = Type.Object(
  {
    projectId: ProjectIdSchema,
    decisionId: DecisionIdSchema,
    requestedBy: AgentIdSchema,
    question: Type.String({ minLength: 1, maxLength: 8000 }),
    evidence: Type.Array(EvidenceRefSchema, { minItems: 1, maxItems: 16 }),
    state: Type.Union([
      Type.Object({ kind: Type.Literal('pending') }, closed),
      Type.Object(
        {
          kind: Type.Literal('answered'),
          answer: Type.String({ minLength: 1, maxLength: 8000 }),
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

export const DecisionSchema = Type.Union([LegacyDecisionSchema, QuestionSchema]);
export type Decision = Static<typeof DecisionSchema>;
