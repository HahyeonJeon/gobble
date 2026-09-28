import type { RunPresentationV2 } from '@gobble/contracts';

/** Whole task records only. Message bounds never cut JSON midway or imply a full Run. */
export function runObservation(value: RunPresentationV2, tasks = value.tasks) {
  const returned: RunPresentationV2['tasks'] = [];
  let bytes = Buffer.byteLength(JSON.stringify(value.continuation ?? null));
  for (const task of tasks) {
    const size = Buffer.byteLength(JSON.stringify(task));
    if (bytes + size > 48 * 1024) break;
    returned.push(task);
    bytes += size;
  }
  return {
    kind: 'run' as const,
    runRef: value.runRef,
    runId: value.runId,
    engineRevision: value.engineRevision,
    observedAt: value.observedAt,
    status: value.status,
    ...(value.continuation ? { continuation: value.continuation } : {}),
    tasks: returned,
    requestedTasks: tasks.length,
    previewTasks: value.tasks.length,
    availableTasks: value.preview?.availableTasks ?? value.tasks.length,
    truncated: returned.length < tasks.length || !!value.preview?.truncated,
  };
}
