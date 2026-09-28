// Frozen reference v2 storage and shared-views-v4 boundary.
import { Type, type Static } from '@sinclair/typebox';
import { closed, ProjectIdSchema, RevisionSchema, SurfaceIdSchema } from './identity';
import { DiscussionResourceRefSchema as ResourceRefSchema } from './resource';
import { SelectionV2Schema as SelectionSchema } from './selection-v2';

/** The durable address. No open Surface is required to retain or resolve it. */
export const ReferenceTargetV2Schema = Type.Object(
  {
    schemaVersion: Type.Literal(2),
    projectId: ProjectIdSchema,
    resource: ResourceRefSchema,
    dataRevision: RevisionSchema,
    selection: Type.Optional(SelectionSchema),
  },
  closed,
);
/** Origin is presentation provenance only: never identity, freshness or access authority. */
export const EvidenceRefV2Schema = Type.Composite(
  [
    ReferenceTargetV2Schema,
    Type.Object({ origin: Type.Optional(Type.Object({ surfaceId: SurfaceIdSchema }, closed)) }),
  ],
  closed,
);
/** A saved local gesture belongs to a specific open Surface, unlike shared references. */
export const LocalSelectionV2Schema = Type.Object(
  {
    surfaceId: SurfaceIdSchema,
    evidence: EvidenceRefV2Schema,
  },
  closed,
);
export type ReferenceTargetV2 = Static<typeof ReferenceTargetV2Schema>;
export type EvidenceRefV2 = Static<typeof EvidenceRefV2Schema>;
export type LocalSelectionV2 = Static<typeof LocalSelectionV2Schema>;
