import { Type } from '@sinclair/typebox';
import { AgentIdSchema, closed, ProjectIdSchema, SurfaceIdSchema } from './identity';
import { DiscussionResourceRefSchema as ResourceRefSchema } from './resource';

/** A Project-owned opened view. The resource is its subject; view is its presentation kind. */
export const SurfaceV3Schema = Type.Object(
  {
    projectId: ProjectIdSchema,
    surfaceId: SurfaceIdSchema,
    resource: ResourceRefSchema,
    view: Type.Union([
      Type.Literal('text'),
      Type.Literal('table'),
      Type.Literal('image'),
      Type.Literal('run'),
      Type.Literal('log'),
    ]),
    pinned: Type.Boolean(),
    userClaimed: Type.Optional(Type.Boolean()),
    openedBy: Type.Union([
      Type.Object({ kind: Type.Literal('user') }, closed),
      Type.Object({ kind: Type.Literal('agent'), agentId: AgentIdSchema }, closed),
    ]),
  },
  closed,
);
