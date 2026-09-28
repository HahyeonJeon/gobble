import { AttachmentIdSchema } from './evidence';
import { CreationContextSchema } from './creation-review';
import { PipelineProposalFileSchema, PipelineReviewContextSchema } from './pipeline-proposal';
import { Type, type TSchema } from '@sinclair/typebox';
import {
  closed,
  ResourceIdSchema,
  SurfaceIdSchema,
  RunRefSchema,
  PipelineIdSchema,
} from './identity';
import { ResourceRefSchema } from './resource';
import { EvidenceRefSchema } from './reference-target';
import { SelectionSchema } from './selection';
import { ObservedReadIdSchema } from './observed-reference';
import { QuestionInputSchema } from './question';
import { PaneIdSchema } from './workspace';
import { RenderAcknowledgmentSchema } from './surface';
export type ToolInputSchema = TSchema;
const empty = Type.Object({}, closed);
export const toolSchemas = {
  read_report: Type.Object(
    {
      attachmentId: AttachmentIdSchema,
      imageId: Type.Optional(Type.String({ pattern: '^image-[0-9]+$' })),
    },
    closed,
  ),
  gobble_creation_source: empty,
  gobble_creation_propose: Type.Object(
    {
      summary: Type.String({ maxLength: 2000 }),
      files: Type.Array(PipelineProposalFileSchema, { minItems: 1, maxItems: 1 }),
    },
    closed,
  ),
  gobble_creation_status: empty,
  gobble_creation_review: Type.Object({ context: CreationContextSchema }, closed),
  gobble_creation_point: Type.Object(
    { context: CreationContextSchema, note: Type.String({ maxLength: 500 }) },
    closed,
  ),
  gobble_pipeline_source: empty,
  gobble_pipeline_proposals: Type.Object({ pipelineId: PipelineIdSchema }, closed),
  gobble_pipeline_propose: Type.Object(
    {
      summary: Type.String({ maxLength: 2000 }),
      files: Type.Array(PipelineProposalFileSchema, { minItems: 1, maxItems: 16 }),
    },
    closed,
  ),
  gobble_pipeline_review: Type.Object({ context: PipelineReviewContextSchema }, closed),
  gobble_pipeline_point: Type.Object(
    { context: PipelineReviewContextSchema, note: Type.String({ maxLength: 500 }) },
    closed,
  ),
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

export const SharedToolRequestSchema = Type.Union(
  Object.entries(toolSchemas).map(([tool, argumentsSchema]) =>
    Type.Object({ tool: Type.Literal(tool), arguments: argumentsSchema }, closed),
  ),
);
