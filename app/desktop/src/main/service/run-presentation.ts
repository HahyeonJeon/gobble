import { AppProblem } from '../problem';
import {
  parse,
  RunPresentationSchema,
  type RunPresentation,
  type RunSnapshot,
} from '@gobble/contracts';

function isRecord(value: unknown): value is Record<string, unknown> {
  return value !== null && typeof value === 'object' && !Array.isArray(value);
}
function text(record: unknown, key: string): string | null {
  if (!isRecord(record))
    throw new AppProblem('incompatible_runtime', 'The runtime returned invalid display metadata.');
  const value = record[key];
  if (value === undefined || value === '') return null;
  if (typeof value !== 'string')
    throw new AppProblem('incompatible_runtime', 'The runtime returned invalid display metadata.');
  return value;
}

/** Converts Monitor v2 facts once at the app boundary; does not derive execution state. */
export function presentRun(value: RunSnapshot): RunPresentation {
  const snapshot: unknown = value.snapshot;
  if (!isRecord(snapshot))
    throw new AppProblem('incompatible_runtime', 'The runtime returned invalid display metadata.');
  const edges = snapshot.edges;
  if (edges !== undefined && !Array.isArray(edges))
    throw new AppProblem('incompatible_runtime', 'The runtime returned invalid dependencies.');
  const identities = new Set<string>();
  for (const task of value.snapshot.tasks) {
    if (
      !task.identity ||
      task.identity.length > 256 ||
      identities.has(task.identity) ||
      !Number.isSafeInteger(task.attempt) ||
      task.attempt < 0
    )
      throw new AppProblem('incompatible_runtime', 'The runtime returned invalid task identities.');
    identities.add(task.identity);
  }
  const flag = (task: Record<string, unknown>, key: string): boolean | null => {
    if (task[key] === undefined) return null;
    if (typeof task[key] !== 'boolean')
      throw new AppProblem('incompatible_runtime', 'The runtime returned invalid task facts.');
    return task[key];
  };
  return parse(RunPresentationSchema, {
    projectId: value.projectId,
    runRef: value.runRef,
    engineRevision: value.engineRevision,
    observedAt: value.observedAt,
    imageId: value.runtimeBinding.imageId,
    runId: value.snapshot.run.id,
    pipelineName: text(snapshot, 'pipeline'),
    status: text(value.snapshot.run, 'status'),
    preview: {
      availableTasks: value.snapshot.tasks.length,
      returnedTasks: Math.min(1000, value.snapshot.tasks.length),
      truncated: value.snapshot.tasks.length > 1000,
    },
    tasks: value.snapshot.tasks.slice(0, 1000).map((task) => ({
      instanceId: task.identity,
      taskId: text(task, 'task_id'),
      name: text(task, 'name'),
      status: text(task, 'status'),
      reason: text(task, 'reason'),
      attempt: task.attempt,
      template: flag(task, 'template'),
      expanded: flag(task, 'expanded'),
    })),
    // Monitor edges link authored task IDs; runtime instance identities must not be substituted.
    dependencies: Array.isArray(edges)
      ? edges.flatMap((edge: unknown) => {
          const fromTaskId = text(edge, 'from');
          const toTaskId = text(edge, 'to');
          if (!fromTaskId || !toTaskId)
            throw new AppProblem(
              'incompatible_runtime',
              'The runtime returned invalid dependencies.',
            );
          return [{ fromTaskId, toTaskId }];
        })
      : [],
  });
}
