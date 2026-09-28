import { Type, type Static } from '@sinclair/typebox';
import {
  closed,
  CounterSchema,
  ResourceIdSchema,
  RevisionSchema,
  SurfaceIdSchema,
} from './identity';

const key = Type.String({ minLength: 1, maxLength: 256 });
const range = Type.Tuple([Type.Number(), Type.Number()]);

/** R2 uses existing numeric coordinates; no implicit statistical transformation. */
export const ScatterSpecSchema = Type.Object(
  {
    xColumnId: key,
    yColumnId: key,
    labelColumnId: Type.Optional(key),
    xScale: Type.Literal('linear'),
    yScale: Type.Literal('linear'),
  },
  closed,
);
export const PlotViewportSchema = Type.Object({ x: range, y: range }, closed);
export const MAX_TABLE_FILTER_VALUE_LENGTH = 4096;
export const TableFilterSchema = Type.Union([
  Type.Object({ kind: Type.Literal('all') }, closed),
  Type.Object(
    {
      kind: Type.Literal('equals'),
      columnId: key,
      value: Type.String({ maxLength: MAX_TABLE_FILTER_VALUE_LENGTH }),
    },
    closed,
  ),
]);
export const ScatterStateSchema = Type.Object(
  {
    spec: ScatterSpecSchema,
    specRevision: CounterSchema,
    viewRevision: CounterSchema,
    viewport: Type.Union([PlotViewportSchema, Type.Null()]),
  },
  closed,
);
export const TableSortSchema = Type.Union([
  Type.Null(),
  Type.Object(
    {
      columnId: key,
      direction: Type.Union([Type.Literal('ascending'), Type.Literal('descending')]),
      numeric: Type.Boolean(),
    },
    closed,
  ),
]);
/** Missing settings mean source order and all columns included. These never own row membership. */
export const TableStateSchema = Type.Object(
  {
    sort: TableSortSchema,
    columns: Type.Union([
      Type.Null(),
      Type.Array(key, { minItems: 1, maxItems: 100, uniqueItems: true }),
    ]),
    viewRevision: CounterSchema,
  },
  closed,
);
/** One membership owner for explicitly linked views of the same source revision. */
export const ViewLinkSchema = Type.Object(
  {
    linkId: Type.String({ pattern: '^lnk_[A-Za-z0-9_-]{1,80}$', maxLength: 84 }),
    resource: Type.Object({ kind: Type.Literal('file'), resourceId: ResourceIdSchema }, closed),
    dataRevision: RevisionSchema,
    surfaceIds: Type.Array(SurfaceIdSchema, { minItems: 1, maxItems: 64, uniqueItems: true }),
    filterRevision: CounterSchema,
    filter: TableFilterSchema,
    rowKeys: Type.Array(key, { maxItems: 500, uniqueItems: true }),
    selectionSourceId: Type.Optional(SurfaceIdSchema),
  },
  closed,
);
export const PresentationRevisionSchema = Type.Object(
  {
    spec: CounterSchema,
    view: CounterSchema,
    filter: CounterSchema,
  },
  closed,
);
/** Presentation provenance, separate from ReferenceTarget identity. Bound into evidence in R2c. */
export const ReferencePresentationSchema = Type.Object(
  {
    schemaVersion: Type.Literal(1),
    kind: Type.Literal('scatter'),
    spec: ScatterSpecSchema,
    filter: TableFilterSchema,
    viewport: PlotViewportSchema,
    revision: PresentationRevisionSchema,
  },
  closed,
);
export type ScatterSpec = Static<typeof ScatterSpecSchema>;
export type PlotViewport = Static<typeof PlotViewportSchema>;
export type TableFilter = Static<typeof TableFilterSchema>;
export type ScatterState = Static<typeof ScatterStateSchema>;
export type ViewLink = Static<typeof ViewLinkSchema>;
export type PresentationRevision = Static<typeof PresentationRevisionSchema>;
export type ReferencePresentation = Static<typeof ReferencePresentationSchema>;
export type TableSort = Static<typeof TableSortSchema>;
export type TableState = Static<typeof TableStateSchema>;
