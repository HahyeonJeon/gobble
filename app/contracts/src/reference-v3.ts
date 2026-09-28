// Frozen v2/v3 evidence boundary for archived content and Agent toolset v6.
import { Type, type Static } from '@sinclair/typebox';
import { closed, ProjectIdSchema, RevisionSchema, SurfaceIdSchema } from './identity';
import { DiscussionResourceRefSchema as ResourceRefSchema } from './resource';
import { ObservedSelectionSchema } from './selection';
import { ReferenceTargetV2Schema, EvidenceRefV2Schema } from './reference-v2';

/** The durable address. No open Surface is required to retain or resolve it. */
const observedTarget = Type.Object(
  {
    schemaVersion: Type.Literal(3),
    projectId: ProjectIdSchema,
    resource: ResourceRefSchema,
    dataRevision: RevisionSchema,
    selection: Type.Optional(ObservedSelectionSchema),
  },
  closed,
);
export const ReferenceTargetV3Schema = Type.Union([ReferenceTargetV2Schema, observedTarget]);
/** Origin is presentation provenance only: never identity, freshness or access authority. */
const observedEvidence = Type.Composite(
  [
    observedTarget,
    Type.Object({ origin: Type.Optional(Type.Object({ surfaceId: SurfaceIdSchema }, closed)) }),
  ],
  closed,
);
export const EvidenceRefV3Schema = Type.Union([EvidenceRefV2Schema, observedEvidence]);
/** A saved local gesture belongs to a specific open Surface, unlike shared references. */
export const LocalSelectionV3Schema = Type.Object(
  {
    surfaceId: SurfaceIdSchema,
    evidence: EvidenceRefV3Schema,
  },
  closed,
);
export type ReferenceTargetV3 = Static<typeof ReferenceTargetV3Schema>;
export type EvidenceRefV3 = Static<typeof EvidenceRefV3Schema>;
export type LocalSelectionV3 = Static<typeof LocalSelectionV3Schema>;
