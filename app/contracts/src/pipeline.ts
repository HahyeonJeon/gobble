import { Type, type Static } from '@sinclair/typebox';
import {
  closed,
  LabelSchema,
  PipelineIdSchema,
  ProjectIdSchema,
  RequestIdSchema,
  ResourceIdSchema,
} from './identity';

/** Origin is source provenance, independent of the adopted Current pointer. */
export const PipelineOriginSchema = Type.Union([
  Type.Object(
    {
      kind: Type.Literal('imported'),
      packageResourceId: ResourceIdSchema,
      sourceResourceId: ResourceIdSchema,
    },
    closed,
  ),
  Type.Object({ kind: Type.Literal('managed') }, closed),
]);
export const PipelineDefinitionSchema = Type.Object(
  {
    projectId: ProjectIdSchema,
    pipelineId: PipelineIdSchema,
    name: LabelSchema,
    origin: PipelineOriginSchema,
  },
  closed,
);
export type PipelineDefinition = Static<typeof PipelineDefinitionSchema>;

export const PipelinesSchema = Type.Object(
  {
    projectId: ProjectIdSchema,
    pipelines: Type.Array(PipelineDefinitionSchema, { maxItems: 1000 }),
  },
  closed,
);

/** Resource IDs are resolved inside the Project service; paths and code are not request inputs. */
export const RegisterPipelineInputSchema = Type.Object(
  {
    projectId: ProjectIdSchema,
    requestId: RequestIdSchema,
    packageResourceId: ResourceIdSchema,
    name: LabelSchema,
  },
  closed,
);
export type RegisterPipelineInput = Static<typeof RegisterPipelineInputSchema>;
