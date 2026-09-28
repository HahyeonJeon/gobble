import { SavedReportRecordSchema } from './run-report';
import { CreationDraftIdSchema } from './pipeline-creation';
import { Type, type Static } from '@sinclair/typebox';
import { closed, ResourceIdSchema, RunRefSchema, PipelineIdSchema } from './identity';

// Paths and runtime bindings stay behind the service, not inside view authority.
export const DiscussionResourceRefSchema = Type.Union([
  Type.Object({ kind: Type.Literal('file'), resourceId: ResourceIdSchema }, closed),
  Type.Object({ kind: Type.Literal('run'), runRef: RunRefSchema }, closed),
  Type.Object(
    {
      kind: Type.Literal('log'),
      runRef: RunRefSchema,
      taskId: Type.String({ minLength: 1, maxLength: 256 }),
      attempt: Type.Integer({ minimum: 1, maximum: Number.MAX_SAFE_INTEGER }),
    },
    closed,
  ),
]);

export const ResourceRefSchema = Type.Union([
  ...DiscussionResourceRefSchema.anyOf,
  Type.Object({ kind: Type.Literal('pipeline'), pipelineId: PipelineIdSchema }, closed),
  Type.Object({ kind: Type.Literal('creation-draft'), draftId: CreationDraftIdSchema }, closed),
  Type.Object({ kind: Type.Literal('report'), saved: SavedReportRecordSchema }, closed),
]);
export type ResourceRef = Static<typeof ResourceRefSchema>;
export type LogTarget = { runRef: string; instanceId: string; attempt: number };

/** Builds an attempt-log resource. v1 persists instance identity under the legacy taskId key. */
export function logResource(target: LogTarget): Extract<ResourceRef, { kind: 'log' }> {
  return { kind: 'log', runRef: target.runRef, taskId: target.instanceId, attempt: target.attempt };
}
/** Returns the runtime instance, never the authored task_id. Preserves existing v1 profiles. */
export function logTarget(resource: Extract<ResourceRef, { kind: 'log' }>): LogTarget {
  return { runRef: resource.runRef, instanceId: resource.taskId, attempt: resource.attempt };
}
export const resourceKey = (resource: ResourceRef): string =>
  resource.kind === 'report'
    ? 'report:' + resource.saved.asset.hash
    : resource.kind === 'creation-draft'
      ? 'creation-draft:' + resource.draftId
      : resource.kind === 'pipeline'
        ? 'pipeline:' + resource.pipelineId
        : resource.kind === 'file'
          ? 'file:' + resource.resourceId
          : resource.kind === 'run'
            ? 'run:' + resource.runRef
            : 'log:' +
              resource.runRef +
              ':' +
              logTarget(resource).instanceId +
              ':' +
              resource.attempt;
export const sameResource = (a: ResourceRef, b: ResourceRef): boolean =>
  resourceKey(a) === resourceKey(b);
