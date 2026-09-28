import { Type, type Static } from '@sinclair/typebox';
import { AgentIdSchema, closed, ProjectIdSchema } from './identity';
import { EvidenceRefSchema } from './reference-target';

// Compatibility exports for existing selection/capture consumers.
export { SelectionSchema, type Selection } from './selection';
export { EvidenceRefSchema, type EvidenceRef } from './reference-target';

export const AddressedContextSchema = Type.Object(
  {
    schemaVersion: Type.Literal(2),
    projectId: ProjectIdSchema,
    recipientAgentId: AgentIdSchema,
    evidence: Type.Array(EvidenceRefSchema, { minItems: 1, maxItems: 16 }),
  },
  closed,
);
export type AddressedContext = Static<typeof AddressedContextSchema>;
