import { Type, type TSchema } from '@sinclair/typebox';
import { closed, ResourceIdSchema, SurfaceIdSchema, RunRefSchema } from './identity';
import { ResourceRefSchema } from './resource';
import { EvidenceRefSchema } from './reference-target';
import { SelectionSchema } from './selection';
import { ObservedReadIdSchema } from './observed-reference';
import { QuestionInputSchema } from './question';
import { PaneIdSchema } from './workspace';
import { RenderAcknowledgmentSchema } from './surface';
export type ToolInputSchema = TSchema;
const empty = Type.Object({}, closed);
export const toolSchemasV10 = {
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

export const SharedToolRequestV10Schema = Type.Union(
  Object.entries(toolSchemasV10).map(([tool, argumentsSchema]) =>
    Type.Object({ tool: Type.Literal(tool), arguments: argumentsSchema }, closed),
  ),
);
