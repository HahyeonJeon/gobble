import {
  RunModeSchema,
  DependencyNavigationSchema,
  DependencyNavigationIntentSchema,
  matchingDependencyGroups,
  matchingDependencies,
} from './dependency-navigation';
import { Type, type Static } from '@sinclair/typebox';
import { closed, CounterSchema, SurfaceIdSchema } from './identity';
import type { RunPresentation } from './run-presentation';
import type { LogPresentation } from './log-presentation';
import type { Surface } from './surface';
import type { EvidenceRef } from './reference-target';
import type { SurfaceData } from './surface-data';

export const RunFilterSchema = Type.Object(
  {
    query: Type.String({ maxLength: 200 }),
    status: Type.Union([Type.String({ maxLength: 256 }), Type.Null()]),
  },
  closed,
);
export const RunViewStateSchema = Type.Object(
  {
    ...RunFilterSchema.properties,
    viewRevision: CounterSchema,
    mode: Type.Optional(RunModeSchema),
    dependencies: Type.Optional(DependencyNavigationSchema),
  },
  closed,
);
export const LogViewStateSchema = Type.Object(
  {
    stream: Type.Union([
      Type.Literal('auto'),
      Type.Literal('stdout'),
      Type.Literal('stderr'),
      Type.Literal('legacy'),
    ]),
    viewRevision: CounterSchema,
  },
  closed,
);
export type RunFilter = Static<typeof RunFilterSchema>;
export type LogStreamChoice = Static<typeof LogViewStateSchema>['stream'];

export const RunNavigationIntentSchema = Type.Union([
  ...DependencyNavigationIntentSchema.anyOf,
  Type.Object(
    { kind: Type.Literal('runFilter'), surfaceId: SurfaceIdSchema, filter: RunFilterSchema },
    closed,
  ),
  Type.Object(
    {
      kind: Type.Literal('logStream'),
      surfaceId: SurfaceIdSchema,
      stream: LogViewStateSchema.properties.stream,
    },
    closed,
  ),
]);

/** Filtering is scoped to the returned preview; task identity and runtime order are preserved. */
export function visibleRunTasks(
  value: RunPresentation,
  filter: RunFilter = { query: '', status: null },
) {
  const query = filter.query.trim().toLocaleLowerCase('en');
  return value.tasks.filter(
    (task) =>
      (filter.status === null || (task.status ?? '') === filter.status) &&
      (!query ||
        [task.instanceId, task.name, task.taskId].some((text) =>
          text?.toLocaleLowerCase('en').includes(query),
        )),
  );
}
export function activeLogStream(
  choice: LogStreamChoice | undefined,
  value: LogPresentation,
): 'stdout' | 'stderr' | 'legacy' {
  if (!choice || choice === 'legacy') return 'legacy';
  return choice === 'auto' ? (value.streams.stderr.text ? 'stderr' : 'stdout') : choice;
}
export function logUnavailableReason(task: RunPresentation['tasks'][number]): string | null {
  return task.template === true
    ? 'Template task · No attempt logs'
    : task.attempt < 1
      ? 'No attempt has started'
      : null;
}
/** Main applies this in addition to revision/coordinate validation; hidden targets are not attachable. */
export function observedTargetVisible(
  surface: Surface,
  data: SurfaceData,
  ref: EvidenceRef,
): boolean {
  if (ref.schemaVersion === 4) {
    if (
      surface.view !== 'run' ||
      surface.runView?.mode !== 'dependencies' ||
      data.kind !== 'run' ||
      !data.dependencies
    )
      return false;
    const query =
      surface.runView.dependencies?.representation !== 'list' &&
      data.dependencies.display === 'graph'
        ? ''
        : (surface.runView.dependencies?.query ?? '');
    const target = ref.selection;
    return target.kind === 'run-group'
      ? matchingDependencyGroups(data.dependencies, query).some((g) => g.taskId === target.taskId)
      : matchingDependencies(data.dependencies, query).some(
          (e) => e.fromTaskId === target.fromTaskId && e.toTaskId === target.toTaskId,
        );
  }
  if (ref.schemaVersion !== 3) return true;
  if (surface.view === 'run' && surface.runView?.mode === 'dependencies' && data.kind === 'run') {
    const target = ref.selection;
    return (
      !!data.dependencies &&
      target?.kind === 'run-task' &&
      data.dependencies.groups.some((g) =>
        g.members.some((m) => m.instanceId === target.instanceId && m.attempt === target.attempt),
      )
    );
  }
  const selection = ref.selection;
  if (surface.view === 'run' && data.kind === 'run')
    return (
      !selection ||
      (selection.kind === 'run-task' &&
        visibleRunTasks(data.value, surface.runView).some(
          (task) => task.instanceId === selection.instanceId && task.attempt === selection.attempt,
        ))
    );
  if (surface.view === 'log' && data.kind === 'log' && 'streams' in data.value)
    return (
      selection?.kind === 'log-text' &&
      activeLogStream(surface.logView?.stream, data.value) === selection.stream
    );
  return false;
}

/** Camera is local presentation geometry; it cannot invalidate a semantic render receipt. */
export function runPresentationState(state: Extract<Surface, { view: 'run' }>['runView']) {
  if (!state?.dependencies) return state;
  const { camera: _camera, ...dependencies } = state.dependencies;
  return { ...state, dependencies };
}
