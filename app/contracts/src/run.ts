import { Type, type Static } from '@sinclair/typebox';
import type { LogPresentation } from './log-presentation';
import {
  closed,
  ProjectIdSchema,
  ResourceIdSchema,
  RevisionSchema,
  RunRefSchema,
  TimestampSchema,
} from './identity';

export const RunRegistrationSchema = Type.Object(
  {
    projectId: ProjectIdSchema,
    runRef: RunRefSchema,
    name: Type.String({ maxLength: 512 }),
    workspaceResourceId: ResourceIdSchema,
    engineRunId: Type.String({ minLength: 1, maxLength: 256 }),
  },
  closed,
);
export const RunsSchema = Type.Object(
  {
    projectId: ProjectIdSchema,
    runs: Type.Array(RunRegistrationSchema, { maxItems: 1000 }),
    candidates: Type.Array(
      Type.Object(
        { workspaceResourceId: ResourceIdSchema, name: Type.String({ maxLength: 512 }) },
        closed,
      ),
      { maxItems: 501 },
    ),
    truncated: Type.Boolean(),
  },
  closed,
);
export const RuntimeBindingSchema = Type.Object(
  {
    endpoint: Type.String({ pattern: '^unix:///', maxLength: 4096 }),
    daemonId: Type.String({ minLength: 1, maxLength: 256 }),
    imageId: Type.String({ pattern: '^sha256:[a-f0-9]{64}$' }),
    platform: Type.Literal('linux/amd64'),
    projectPath: Type.Literal('/gobble/project'),
    workspacePath: Type.String({ pattern: '^/gobble/project/', maxLength: 8192 }),
  },
  closed,
);

// Engine fields remain engine-owned. Validate the consumed header; retain other JSON facts.
const engineObject = Type.Record(Type.String(), Type.Unknown());
export const RunSnapshotSchema = Type.Object(
  {
    projectId: ProjectIdSchema,
    runRef: RunRefSchema,
    engineRevision: RevisionSchema,
    observedAt: TimestampSchema,
    runtimeBinding: RuntimeBindingSchema,
    availability: Type.Literal('available'),
    snapshot: Type.Object(
      {
        schema_version: Type.Literal(2),
        snapshot: RevisionSchema,
        run: Type.Object(
          { id: Type.String({ minLength: 1, maxLength: 256 }) },
          { additionalProperties: true },
        ),
        tasks: Type.Array(
          Type.Object(
            { identity: Type.String(), attempt: Type.Integer() },
            { additionalProperties: true },
          ),
        ),
        logs: Type.Array(engineObject),
      },
      { additionalProperties: true },
    ),
  },
  closed,
);
export const RunLogsSchema = Type.Object(
  {
    projectId: ProjectIdSchema,
    runRef: RunRefSchema,
    engineRevision: RevisionSchema,
    observedAt: TimestampSchema,
    instance: Type.String({ minLength: 1, maxLength: 256 }),
    attempt: Type.Integer({ minimum: 1 }),
    tailLimitBytes: Type.Literal(4096),
    logs: Type.Array(engineObject),
  },
  closed,
);

export type RunSnapshot = Static<typeof RunSnapshotSchema>;
export type RunLogs = Static<typeof RunLogsSchema>;

// Canonical displayed log text is shared with selection validation; source paths are not preview text.
export function logPreviewText(value: RunLogs | LogPresentation): string {
  if ('schemaVersion' in value) {
    return [
      'Task: ' + value.instance,
      value.error,
      'Standard output',
      value.streams.stdout.text,
      'Standard error',
      value.streams.stderr.text,
    ]
      .filter(Boolean)
      .join('\n');
  }
  return value.logs
    .map((log) => {
      const text = (key: string) => (typeof log[key] === 'string' ? log[key] : '');
      return [
        'Task: ' + text('identity'),
        text('error'),
        'Standard output',
        text('stdout_tail'),
        'Standard error',
        text('stderr_tail'),
      ]
        .filter(Boolean)
        .join('\n');
    })
    .join('\n\n');
}
