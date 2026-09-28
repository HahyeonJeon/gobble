import { PdfTargetSchema } from './pdf';
import { Type } from '@sinclair/typebox';
import { ReferenceTargetV3Schema, EvidenceRefV3Schema } from './reference-v3';
import { DependencyTargetSchema } from './run-dependencies';
export const ReferenceTargetV5Schema = Type.Union([
  ReferenceTargetV3Schema,
  DependencyTargetSchema,
  PdfTargetSchema,
]);
export const EvidenceRefV5Schema = Type.Union([
  EvidenceRefV3Schema,
  DependencyTargetSchema,
  PdfTargetSchema,
]);
