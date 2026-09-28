import { continuationStepLabel, type RunContinuationContext } from '@gobble/contracts';

/** Read-only content shared by the Flow and historical evidence; never renders actions. */
export function ContinuationContext({
  context,
  taskId,
  labels = {},
}: {
  context: RunContinuationContext;
  taskId?: string | null | undefined;
  labels?: Record<string, string>;
}) {
  const steps = taskId
    ? context.review.steps.filter((s) => s.taskId === taskId)
    : context.review.steps;
  return (
    <div className="continuation-context" aria-label="Saved continuation review">
      {steps.map((step) => (
        <p key={step.taskId} data-continuation={step.action}>
          <strong>
            {labels[step.taskId] ?? step.taskId} · {continuationStepLabel(step)}
          </strong>
          <br />
          {step.action === 'reuse'
            ? 'The checked result is kept.'
            : step.action === 'restart'
              ? `Attempt ${step.priorAttempt} remains in history. This step starts again from its beginning.`
              : 'This step has not run yet.'}
        </p>
      ))}
      <small>
        Saved design and checked data · Review {context.reviewId.slice(-8)} · Captured plan, not
        live status
      </small>
    </div>
  );
}
