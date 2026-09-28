import { Type } from '@sinclair/typebox';
import { closed, ResourceIdSchema, SurfaceIdSchema, RunRefSchema } from './identity';
import { DiscussionResourceRefSchema as ResourceRefSchema } from './resource';
import { EvidenceRefV4Schema as EvidenceRefSchema } from './reference-v4';
import { SelectionV4Schema as SelectionSchema } from './selection';
import { ObservedReadIdSchema } from './observed-reference';
import { QuestionInputSchema } from './question';
import { PaneIdSchema } from './workspace';
import { RenderAcknowledgmentSchema } from './surface';
const empty = Type.Object({}, closed);
export const toolSchemasV7 = {
  workspace_question: QuestionInputSchema,
  workspace_list: empty,
  workspace_resources: Type.Object({ directoryId: Type.Optional(ResourceIdSchema) }, closed),
  workspace_open: Type.Object(
    {
      resources: Type.Array(ResourceRefSchema, { minItems: 1, maxItems: 2 }),
      preferredPane: Type.Optional(PaneIdSchema),
    },
    closed,
  ),
  workspace_arrange: Type.Object({ surfaceId: SurfaceIdSchema, pane: PaneIdSchema }, closed),
  workspace_observe: Type.Object(
    {
      surfaceId: SurfaceIdSchema,
      representation: Type.Optional(
        Type.Union([Type.Literal('tasks'), Type.Literal('dependencies')]),
      ),
      stream: Type.Optional(Type.Union([Type.Literal('stdout'), Type.Literal('stderr')])),
      selection: Type.Optional(SelectionSchema),
      scope: Type.Optional(Type.Union([Type.Literal('view'), Type.Literal('source-preview')])),
    },
    closed,
  ),
  workspace_point: Type.Object(
    {
      evidence: EvidenceRefSchema,
      observedReadId: Type.Optional(ObservedReadIdSchema),
      observation: Type.Optional(RenderAcknowledgmentSchema),
      note: Type.String({ maxLength: 500 }),
    },
    closed,
  ),
  workspace_release: Type.Object({ surfaceId: SurfaceIdSchema }, closed),
  gobble_runs: empty,
  gobble_run: Type.Object({ runRef: RunRefSchema }, closed),
  gobble_logs: Type.Object(
    {
      runRef: RunRefSchema,
      instance: Type.String({ minLength: 1, maxLength: 256 }),
      attempt: Type.Integer({ minimum: 1, maximum: Number.MAX_SAFE_INTEGER }),
    },
    closed,
  ),
};

export const SharedToolRequestV7Schema = Type.Union(
  Object.entries(toolSchemasV7).map(([tool, argumentsSchema]) =>
    Type.Object({ tool: Type.Literal(tool), arguments: argumentsSchema }, closed),
  ),
);
