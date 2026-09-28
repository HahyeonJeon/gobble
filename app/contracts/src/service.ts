import type { ContinuationBridge } from './run-continuation';
import type { LaunchBridge } from './run-launch';
import type { PreparationBridge } from './run-preparation';
import type { CreationBridge } from './creation-bridge';
import {
  PipelineReviewStateResultSchema,
  PipelineReviewSelectSchema,
  PipelineReviewSelectedResultSchema,
  PipelineAdoptInputSchema,
  PipelineAdoptionInputSchema,
  PipelineAdoptionResultSchema,
} from './pipeline-proposal';
import {
  PipelineInspectionSchema,
  PipelineInspectionInputSchema,
  CheckPipelineInputSchema,
  CancelPipelineCheckInputSchema,
} from './pipeline-inspection';
import { PipelineDefinitionSchema, PipelinesSchema, RegisterPipelineInputSchema } from './pipeline';
import { Type, type Static } from '@sinclair/typebox';
import {
  closed,
  ProjectIdSchema,
  ResourceIdSchema,
  RevisionSchema,
  RunRefSchema,
  RequestIdSchema,
} from './identity';
import { ProjectInfoSchema } from './project';
import { DirectorySchema, FileContentSchema } from './file';
import { RunRegistrationSchema, RunsSchema, RunSnapshotSchema, RunLogsSchema } from './run';
import { serviceResult } from './result';

export const ServiceHandshakeSchema = Type.Object(
  {
    schemaVersion: Type.Literal(1),
    instanceId: Type.String({ pattern: '^svc_[A-Za-z0-9_]+$', maxLength: 100 }),
    port: Type.Integer({ minimum: 1, maximum: 65535 }),
  },
  closed,
);
export const ServiceCapabilitiesSchema = Type.Object(
  {
    protocolVersion: Type.Literal(1),
    queries: Type.Array(Type.String({ maxLength: 100 }), { maxItems: 20 }),
    mutations: Type.Array(Type.String({ maxLength: 100 }), { maxItems: 32 }),
  },
  closed,
);
export const ProjectsResultSchema = serviceResult(Type.Array(ProjectInfoSchema, { maxItems: 100 }));
export const ProjectResultSchema = serviceResult(ProjectInfoSchema);
export const DirectoryResultSchema = serviceResult(DirectorySchema);
export const FileResultSchema = serviceResult(FileContentSchema);
export const ImportPipelineInputSchema = Type.Object(
  { projectId: ProjectIdSchema, requestId: RequestIdSchema },
  closed,
);
export const ImportPipelineResultSchema = serviceResult(
  Type.Union([
    Type.Object({ kind: Type.Literal('cancelled') }, closed),
    Type.Object({ kind: Type.Literal('selected'), pipeline: PipelineDefinitionSchema }, closed),
  ]),
);
export const PipelineInspectionResultSchema = serviceResult(PipelineInspectionSchema);
export const PipelinesResultSchema = serviceResult(PipelinesSchema);
export const PipelineResultSchema = serviceResult(PipelineDefinitionSchema);
export const RunsResultSchema = serviceResult(RunsSchema);
export const RunResultSchema = serviceResult(RunRegistrationSchema);
export const SnapshotResultSchema = serviceResult(RunSnapshotSchema);
export const LogsResultSchema = serviceResult(RunLogsSchema);
export const CapabilitiesResultSchema = serviceResult(ServiceCapabilitiesSchema);
export const ChooseProjectResultSchema = serviceResult(
  Type.Union([
    Type.Object({ kind: Type.Literal('cancelled') }, closed),
    Type.Object({ kind: Type.Literal('selected'), project: ProjectInfoSchema }, closed),
  ]),
);

export const ChooseProjectInputSchema = Type.Object({ requestId: RequestIdSchema }, closed);
export const ProjectInputSchema = Type.Object({ projectId: ProjectIdSchema }, closed);
export const DirectoryInputSchema = Type.Object(
  { projectId: ProjectIdSchema, directoryId: Type.Optional(ResourceIdSchema) },
  closed,
);
export const FileInputSchema = Type.Object(
  {
    projectId: ProjectIdSchema,
    resourceId: ResourceIdSchema,
    expectedRevision: Type.Optional(RevisionSchema),
  },
  closed,
);
export const AttachRunInputSchema = Type.Object(
  { projectId: ProjectIdSchema, workspaceResourceId: ResourceIdSchema, requestId: RequestIdSchema },
  closed,
);
export const SnapshotInputSchema = Type.Object(
  { projectId: ProjectIdSchema, runRef: RunRefSchema },
  closed,
);
export const LogsInputSchema = Type.Object(
  {
    ...SnapshotInputSchema.properties,
    instance: Type.String({ minLength: 1, maxLength: 256 }),
    attempt: Type.Integer({ minimum: 1, maximum: Number.MAX_SAFE_INTEGER }),
  },
  closed,
);

export type DirectoryInput = Static<typeof DirectoryInputSchema>;
export type FileInput = Static<typeof FileInputSchema>;
export type ProjectInput = Static<typeof ProjectInputSchema>;
export type AttachRunInput = Static<typeof AttachRunInputSchema>;
export type SnapshotInput = Static<typeof SnapshotInputSchema>;
export type LogsInput = Static<typeof LogsInputSchema>;

export type ServiceBridge = Readonly<{
  creation: CreationBridge;
  preparations: PreparationBridge;
  launches: LaunchBridge;
  continuations: ContinuationBridge;
  pipelineReviews: Readonly<{
    list: (
      input: Static<typeof PipelineInspectionInputSchema>,
    ) => Promise<Static<typeof PipelineReviewStateResultSchema>>;
    select: (
      input: Static<typeof PipelineReviewSelectSchema>,
    ) => Promise<Static<typeof PipelineReviewSelectedResultSchema>>;
    adopt: (
      input: Static<typeof PipelineAdoptInputSchema>,
    ) => Promise<Static<typeof PipelineAdoptionResultSchema>>;
    outcome: (
      input: Static<typeof PipelineAdoptionInputSchema>,
    ) => Promise<Static<typeof PipelineAdoptionResultSchema>>;
  }>;
  projects: Readonly<{
    list: () => Promise<Static<typeof ProjectsResultSchema>>;
    chooseFolder: (
      input: Static<typeof ChooseProjectInputSchema>,
    ) => Promise<Static<typeof ChooseProjectResultSchema>>;
  }>;
  files: Readonly<{
    list: (
      input: Static<typeof DirectoryInputSchema>,
    ) => Promise<Static<typeof DirectoryResultSchema>>;
    read: (input: Static<typeof FileInputSchema>) => Promise<Static<typeof FileResultSchema>>;
  }>;
  pipelines: Readonly<{
    importFolder: (
      input: Static<typeof ImportPipelineInputSchema>,
    ) => Promise<Static<typeof ImportPipelineResultSchema>>;
    inspection: (
      input: Static<typeof PipelineInspectionInputSchema>,
    ) => Promise<Static<typeof PipelineInspectionResultSchema>>;
    check: (
      input: Static<typeof CheckPipelineInputSchema>,
    ) => Promise<Static<typeof PipelineInspectionResultSchema>>;
    cancelCheck: (
      input: Static<typeof CancelPipelineCheckInputSchema>,
    ) => Promise<Static<typeof PipelineInspectionResultSchema>>;
    list: (input: ProjectInput) => Promise<Static<typeof PipelinesResultSchema>>;
    register: (
      input: Static<typeof RegisterPipelineInputSchema>,
    ) => Promise<Static<typeof PipelineResultSchema>>;
  }>;
  runs: Readonly<{
    list: (input: Static<typeof ProjectInputSchema>) => Promise<Static<typeof RunsResultSchema>>;
    attach: (input: Static<typeof AttachRunInputSchema>) => Promise<Static<typeof RunResultSchema>>;
    snapshot: (
      input: Static<typeof SnapshotInputSchema>,
    ) => Promise<Static<typeof SnapshotResultSchema>>;
    logs: (input: Static<typeof LogsInputSchema>) => Promise<Static<typeof LogsResultSchema>>;
  }>;
}>;

export const SERVICE_CHANNELS = {
  pipelineReviews: 'gobble:pipeline-reviews:list:v1',
  pipelineReviewSelect: 'gobble:pipeline-reviews:select:v1',
  pipelineAdopt: 'gobble:pipeline-reviews:adopt:v1',
  pipelineAdoption: 'gobble:pipeline-reviews:outcome:v1',
  projectsList: 'gobble:projects:list:v1',
  projectsChoose: 'gobble:projects:choose:v1',
  filesList: 'gobble:files:list:v1',
  filesRead: 'gobble:files:read:v1',
  pipelineImport: 'gobble:pipelines:import:v1',
  pipelineInspection: 'gobble:pipelines:inspection:v1',
  pipelineCheck: 'gobble:pipelines:check:v1',
  pipelineCancelCheck: 'gobble:pipelines:cancel-check:v1',
  pipelinesList: 'gobble:pipelines:list:v1',
  pipelinesRegister: 'gobble:pipelines:register:v1',
  runsList: 'gobble:runs:list:v1',
  runsAttach: 'gobble:runs:attach:v1',
  runsSnapshot: 'gobble:runs:snapshot:v1',
  runsLogs: 'gobble:runs:logs:v1',
} as const;
