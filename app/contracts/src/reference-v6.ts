import { NotebookTargetSchema } from './notebook';
import { PdfTargetSchema } from './pdf';
import { Type, type Static } from '@sinclair/typebox';
import { closed, SurfaceIdSchema } from './identity';
import { ReferenceTargetV3Schema, EvidenceRefV3Schema } from './reference-v3';
import { DependencyTargetSchema } from './run-dependencies';
export const ReferenceTargetV6Schema = Type.Union([
  ReferenceTargetV3Schema,
  DependencyTargetSchema,
  PdfTargetSchema,
  NotebookTargetSchema,
]);
export const EvidenceRefV6Schema = Type.Union([
  EvidenceRefV3Schema,
  DependencyTargetSchema,
  PdfTargetSchema,
  NotebookTargetSchema,
]);
export const LocalSelectionV6Schema = Type.Object(
  { surfaceId: SurfaceIdSchema, evidence: EvidenceRefV6Schema },
  closed,
);
export type ReferenceTargetV6 = Static<typeof ReferenceTargetV6Schema>;
export type EvidenceRefV6 = Static<typeof EvidenceRefV6Schema>;
export type LocalSelectionV6 = Static<typeof LocalSelectionV6Schema>;
