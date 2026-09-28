import { Type, type Static } from '@sinclair/typebox';
import { DecisionSchema } from './decision';
import { closed, CounterSchema, ProjectIdSchema, SurfaceIdSchema } from './identity';
import { AgentAttachmentSchema } from './agent';
import { SurfaceSchema } from './surface';

/** Layout container: ordered surface tabs and one active surface; never owns source content. */
export const PaneSchema = Type.Object(
  {
    tabs: Type.Array(SurfaceIdSchema, { maxItems: 32, uniqueItems: true }),
    activeSurfaceId: Type.Union([SurfaceIdSchema, Type.Null()]),
  },
  closed,
);

export const PaneIdSchema = Type.Union([Type.Literal('primary'), Type.Literal('secondary')]);
export type PaneId = Static<typeof PaneIdSchema>;
export type Pane = Static<typeof PaneSchema>;

/** The single shared application workspace for a Project; unrelated to an engine execution directory. */
export const WorkspaceSchema = Type.Object(
  {
    schemaVersion: Type.Literal(1),
    projectId: ProjectIdSchema,
    revision: CounterSchema,
    agents: Type.Array(AgentAttachmentSchema, { maxItems: 32 }),
    surfaces: Type.Array(SurfaceSchema, { maxItems: 64 }),
    layout: Type.Union([
      Type.Object({ kind: Type.Literal('single'), primary: PaneSchema }, closed),
      Type.Object(
        {
          kind: Type.Literal('split'),
          primary: PaneSchema,
          secondary: PaneSchema,
          primaryFraction: Type.Number({ minimum: 0.2, maximum: 0.8 }),
        },
        closed,
      ),
    ]),
    decisions: Type.Array(DecisionSchema, { maxItems: 1000 }),
  },
  closed,
);

export type Workspace = Static<typeof WorkspaceSchema>;
