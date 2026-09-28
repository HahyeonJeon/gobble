import { ResourceRefSchema } from './resource';
import { NotebookNavigationSchema } from './notebook';
import { PdfNavigationSchema } from './pdf';
import { Type, type Static } from '@sinclair/typebox';
import {
  closed,
  CounterSchema,
  ProjectIdSchema,
  RendererSessionIdSchema,
  RequestIdSchema,
  RevisionSchema,
  SurfaceIdSchema,
} from './identity';
import { ScatterStateSchema, TableStateSchema, PresentationRevisionSchema } from './tabular-view';
import { SurfaceV3Schema } from './surface-v3';
import { RunViewStateSchema, LogViewStateSchema } from './run-navigation';

/** A Project-owned opened view. The resource is its subject; view is its presentation kind. */
export const SurfaceSchema = Type.Union([
  Type.Composite(
    [
      Type.Omit(SurfaceV3Schema, ['view', 'resource']),
      Type.Object({
        view: Type.Literal('report'),
        resource: ResourceRefSchema.anyOf[5],
        reportModuleId: Type.Optional(Type.String({ pattern: '^M[0-9]+$', maxLength: 16 })),
      }),
    ],
    closed,
  ),
  Type.Composite(
    [
      Type.Omit(SurfaceV3Schema, ['view', 'resource']),
      Type.Object({ view: Type.Literal('creation-draft'), resource: ResourceRefSchema.anyOf[4] }),
    ],
    closed,
  ),
  Type.Composite(
    [
      Type.Omit(SurfaceV3Schema, ['view', 'resource']),
      Type.Object({ view: Type.Literal('pipeline'), resource: ResourceRefSchema.anyOf[3] }),
    ],
    closed,
  ),
  Type.Composite(
    [
      Type.Omit(SurfaceV3Schema, ['view']),
      Type.Object({
        view: Type.Literal('notebook'),
        notebook: Type.Optional(NotebookNavigationSchema),
      }),
    ],
    closed,
  ),
  Type.Composite(
    [
      Type.Omit(SurfaceV3Schema, ['view']),
      Type.Object({ view: Type.Literal('pdf'), pdf: Type.Optional(PdfNavigationSchema) }),
    ],
    closed,
  ),
  Type.Composite(
    [
      Type.Omit(SurfaceV3Schema, ['view']),
      Type.Object({
        view: Type.Union([Type.Literal('text'), Type.Literal('image')]),
      }),
    ],
    closed,
  ),
  Type.Composite(
    [
      Type.Omit(SurfaceV3Schema, ['view']),
      Type.Object({ view: Type.Literal('run'), runView: Type.Optional(RunViewStateSchema) }),
    ],
    closed,
  ),
  Type.Composite(
    [
      Type.Omit(SurfaceV3Schema, ['view']),
      Type.Object({ view: Type.Literal('log'), logView: Type.Optional(LogViewStateSchema) }),
    ],
    closed,
  ),
  Type.Composite(
    [
      Type.Omit(SurfaceV3Schema, ['view']),
      Type.Object({ view: Type.Literal('table'), table: Type.Optional(TableStateSchema) }),
    ],
    closed,
  ),
  Type.Composite(
    [
      Type.Omit(SurfaceV3Schema, ['view']),
      Type.Object({ view: Type.Literal('scatter'), scatter: ScatterStateSchema }),
    ],
    closed,
  ),
]);

export const RenderAcknowledgmentSchema = Type.Object(
  {
    schemaVersion: Type.Literal(3),
    projectId: ProjectIdSchema,
    surfaceId: SurfaceIdSchema,
    rendererSessionId: RendererSessionIdSchema,
    requestId: RequestIdSchema,
    generation: CounterSchema,
    dataRevision: RevisionSchema,
    presentation: PresentationRevisionSchema,
    referenceViewRequestId: Type.Optional(RequestIdSchema),
  },
  closed,
);

export type Surface = Static<typeof SurfaceSchema>;
export type RenderAcknowledgment = Static<typeof RenderAcknowledgmentSchema>;
