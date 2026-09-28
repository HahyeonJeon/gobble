// Frozen before R3c2.
import { Type, type Static } from '@sinclair/typebox';
import { closed, SurfaceIdSchema } from './identity';
import { ReferenceTargetV3Schema, EvidenceRefV3Schema } from './reference-v3';
import { DependencyTargetSchema } from './run-dependencies';
export const ReferenceTargetV4Schema = Type.Union([
  ReferenceTargetV3Schema,
  DependencyTargetSchema,
]);
export const EvidenceRefV4Schema = Type.Union([EvidenceRefV3Schema, DependencyTargetSchema]);
export const LocalSelectionV4Schema = Type.Object(
  { surfaceId: SurfaceIdSchema, evidence: EvidenceRefV4Schema },
  closed,
);
export type ReferenceTargetV4 = Static<typeof ReferenceTargetV4Schema>;
export type EvidenceRefV4 = Static<typeof EvidenceRefV4Schema>;
export type LocalSelectionV4 = Static<typeof LocalSelectionV4Schema>;
