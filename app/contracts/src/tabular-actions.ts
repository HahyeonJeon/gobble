import { Type, type Static } from '@sinclair/typebox';
import type { WorkspaceAction } from './workspace-bridge';
import { closed, SurfaceIdSchema, RevisionSchema } from './identity';
import { RenderAcknowledgmentSchema } from './surface';
import {
  ScatterSpecSchema,
  TableFilterSchema,
  TableStateSchema,
  PlotViewportSchema,
} from './tabular-view';

const observed = { surfaceId: SurfaceIdSchema, acknowledgment: RenderAcknowledgmentSchema };
export const TabularActionSchema = Type.Union([
  Type.Object({ kind: Type.Literal('openScatter'), ...observed, spec: ScatterSpecSchema }, closed),
  Type.Object({ kind: Type.Literal('openLinkedTable'), ...observed }, closed),
  Type.Object(
    {
      kind: Type.Literal('tableSettings'),
      ...observed,
      settings: Type.Omit(TableStateSchema, ['viewRevision']),
    },
    closed,
  ),
  Type.Object(
    { kind: Type.Literal('scatterSettings'), ...observed, spec: ScatterSpecSchema },
    closed,
  ),
  Type.Object(
    { kind: Type.Literal('linkedFilter'), ...observed, filter: TableFilterSchema },
    closed,
  ),
  Type.Object(
    {
      kind: Type.Literal('scatterViewport'),
      ...observed,
      viewport: Type.Union([PlotViewportSchema, Type.Null()]),
    },
    closed,
  ),
  Type.Object({ kind: Type.Literal('discussSelection'), ...observed }, closed),
  Type.Object(
    {
      kind: Type.Literal('refreshLinked'),
      surfaceId: SurfaceIdSchema,
      dataRevision: RevisionSchema,
    },
    closed,
  ),
]);
export type TabularAction = Static<typeof TabularActionSchema>;
const kinds = new Set([
  'openScatter',
  'openLinkedTable',
  'tableSettings',
  'scatterSettings',
  'linkedFilter',
  'scatterViewport',
  'discussSelection',
  'refreshLinked',
]);
/** Discriminator for an already schema-validated Workspace action, not an input parser. */
export function isTabularAction(action: WorkspaceAction): action is TabularAction {
  return kinds.has(action.kind);
}
