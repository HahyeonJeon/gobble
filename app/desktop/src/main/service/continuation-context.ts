import type { Continuation, RunPresentation, RunContinuationContext } from '@gobble/contracts';

/** Join only the ready review of this exact stopped snapshot. Never overlay a proposal on a later attempt. */
export function continuationContext(
  raw: RunPresentation,
  snapshot: string,
  values: Continuation[],
): RunContinuationContext | undefined {
  const latest = values
    .filter((v) => v.projectId === raw.projectId && v.runRef === raw.runRef)
    .sort(
      (a, b) => b.createdAt.localeCompare(a.createdAt) || b.requestId.localeCompare(a.requestId),
    )[0];
  if (
    !latest ||
    latest.state !== 'ready' ||
    !latest.review ||
    raw.status !== 'stopped' ||
    latest.review.snapshot !== snapshot
  )
    return;
  if (
    !latest.review.steps.every((step) => {
      const tasks = raw.tasks.filter((t) => t.taskId === step.taskId && !t.template);
      return tasks.length === 1 && tasks[0]!.attempt === step.priorAttempt;
    })
  )
    return;
  return {
    schemaVersion: 1,
    reviewId: latest.requestId,
    checkedAt: latest.createdAt,
    review: structuredClone(latest.review),
  };
}
