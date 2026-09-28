import { Type, type Static } from '@sinclair/typebox';
import { closed, ProjectIdSchema, RunRefSchema, RevisionSchema, TimestampSchema } from './identity';

const nullableText = Type.Union([Type.String(), Type.Null()]);
/** Read-only display facts. A pipeline label is not a registered Pipeline definition or Plan. */
export const RunPresentationSchema = Type.Object(
  {
    projectId: ProjectIdSchema,
    runRef: RunRefSchema,
    engineRevision: RevisionSchema,
    observedAt: TimestampSchema,
    imageId: Type.String(),
    runId: Type.String(),
    pipelineName: nullableText,
    status: nullableText,
    preview: Type.Optional(
      Type.Object(
        {
          availableTasks: Type.Integer({ minimum: 0, maximum: Number.MAX_SAFE_INTEGER }),
          returnedTasks: Type.Integer({ minimum: 0, maximum: 1000 }),
          truncated: Type.Boolean(),
        },
        closed,
      ),
    ),
    tasks: Type.Array(
      Type.Object(
        {
          instanceId: Type.String(),
          taskId: nullableText,
          name: nullableText,
          status: nullableText,
          reason: nullableText,
          attempt: Type.Integer(),
          template: Type.Optional(Type.Union([Type.Boolean(), Type.Null()])),
          expanded: Type.Optional(Type.Union([Type.Boolean(), Type.Null()])),
        },
        closed,
      ),
    ),
    dependencies: Type.Array(
      Type.Object({ fromTaskId: Type.String(), toTaskId: Type.String() }, closed),
    ),
  },
  closed,
);
export type RunPresentation = Static<typeof RunPresentationSchema>;
