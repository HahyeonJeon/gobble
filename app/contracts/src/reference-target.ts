import { ReportTargetSchema } from './run-report';
import { PipelineTargetSchema } from './pipeline-reference';
import { NotebookTargetSchema } from './notebook';
import { PdfTargetSchema } from './pdf';
import { Type, type Static } from '@sinclair/typebox';
import { closed, SurfaceIdSchema } from './identity';
import { ReferenceTargetV3Schema, EvidenceRefV3Schema } from './reference-v3';
import { DependencyTargetSchema } from './run-dependencies';
export const ReferenceTargetSchema = Type.Union([
  ReferenceTargetV3Schema,
  DependencyTargetSchema,
  PdfTargetSchema,
  NotebookTargetSchema,
  PipelineTargetSchema,
  ReportTargetSchema,
]);
export const EvidenceRefSchema = Type.Union([
  EvidenceRefV3Schema,
  DependencyTargetSchema,
  PdfTargetSchema,
  NotebookTargetSchema,
  PipelineTargetSchema,
  ReportTargetSchema,
]);
export const LocalSelectionSchema = Type.Object(
  { surfaceId: SurfaceIdSchema, evidence: EvidenceRefSchema },
  closed,
);
export type ReferenceTarget = Static<typeof ReferenceTargetSchema>;
export type EvidenceRef = Static<typeof EvidenceRefSchema>;
export type LocalSelection = Static<typeof LocalSelectionSchema>;
