import { SavedReportSchema, SavedReportRecordSchema } from './run-report';
import { PipelineInspectionSchema } from './pipeline-inspection';
import { DependencyObservationSchema } from './run-dependencies';
import { Type, type Static } from '@sinclair/typebox';
import { closed } from './identity';
import { FileContentSchema, FilePreviewContentSchema } from './file';
import { NotebookContentSchema } from './notebook';
import { PdfContentSchema } from './pdf';
const SurfaceFileSchema = Type.Object(
  {
    ...FileContentSchema.properties,
    content: Type.Union([
      NotebookContentSchema,
      PdfContentSchema,
      ...FilePreviewContentSchema.anyOf,
    ]),
  },
  closed,
);
import { RunLogsSchema } from './run';
import { RunPresentationV2Schema as RunPresentationSchema } from './run-presentation-v2';
import { LogPresentationSchema } from './log-presentation';

/** Validated resource preview data shared by rendering, references and capture. */
export const SurfaceDataSchema = Type.Union([
  Type.Object(
    { kind: Type.Literal('report'), value: SavedReportSchema, saved: SavedReportRecordSchema },
    closed,
  ),
  Type.Object({ kind: Type.Literal('pipeline'), value: PipelineInspectionSchema }, closed),
  Type.Object({ kind: Type.Literal('file'), value: SurfaceFileSchema }, closed),
  Type.Object(
    {
      kind: Type.Literal('run'),
      value: RunPresentationSchema,
      dependencies: Type.Optional(DependencyObservationSchema),
      dependencyProblem: Type.Optional(Type.String({ maxLength: 1000 })),
    },
    closed,
  ),
  Type.Object(
    { kind: Type.Literal('log'), value: Type.Union([LogPresentationSchema, RunLogsSchema]) },
    closed,
  ),
]);
export type SurfaceData = Static<typeof SurfaceDataSchema>;
