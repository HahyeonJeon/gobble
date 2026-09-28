// Frozen authored-reference shape through Workspace v9 and Agent toolset v6.
import { Type, type Static } from '@sinclair/typebox';
import { AgentIdSchema, LabelSchema, RequestIdSchema, TimestampSchema, closed } from './identity';
import { EvidenceRefV3Schema as EvidenceRefSchema } from './reference-v3';
import { ReferencePresentationSchema } from './tabular-view';
export const SharedReferenceIdSchema = Type.String({
  pattern: '^ref_[A-Za-z0-9_-]{1,80}$',
  maxLength: 84,
});
/** An authored pointer, not a source snapshot or an addressed message attachment. */
export const SharedReferenceV3Schema = Type.Object(
  {
    referenceId: SharedReferenceIdSchema,
    label: Type.String({ minLength: 1, maxLength: 512 }),
    evidence: EvidenceRefSchema,
    presentation: Type.Optional(ReferencePresentationSchema),
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
export type SharedReferenceV3 = Static<typeof SharedReferenceV3Schema>;
