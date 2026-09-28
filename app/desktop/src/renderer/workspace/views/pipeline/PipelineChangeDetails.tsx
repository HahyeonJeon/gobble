import type { PipelineChange, PipelineFlow } from '@gobble/contracts';
import { stepLabel } from './flow-layout';

const setting = (value: number | null, unit: string) =>
  value === null ? 'Tool default (unknown)' : `${value}${unit ? ' ' + unit : ''}`;

/** Renders only checked comparison facts. It owns no adoption or Chat state. */
export function PipelineChangeDetails({
  flow,
  change,
  index,
  total,
  busy,
  onDiscuss,
}: {
  flow: PipelineFlow;
  change: PipelineChange;
  index: number;
  total: number;
  busy: boolean;
  onDiscuss: () => void;
}) {
  const selectedStep = flow.steps.find((step) => step.id === change.stepId);
  return (
    <section className="review-detail" aria-label="Selected change">
      <header>
        <div>
          <small>
            CHANGE {index + 1} OF {total}
          </small>
          <h3>{selectedStep ? stepLabel(selectedStep) : change.label}</h3>
        </div>
        <button disabled={busy} onClick={onDiscuss}>
          Discuss this change →
        </button>
      </header>
      <div className="review-pair">
        <article className="review-before">
          <h4>− Current</h4>
          {change.kind === 'setting' ? (
            <>
              <span>{change.label}</span>
              <strong>{setting(change.before, change.unit)}</strong>
            </>
          ) : (
            <>
              <strong>No quality check here</strong>
              <span>This branch does not exist in the current flow.</span>
            </>
          )}
        </article>
        <article className="review-after">
          <h4>＋ Proposed</h4>
          {change.kind === 'setting' ? (
            <>
              <span>{change.label}</span>
              <strong>{setting(change.after, change.unit)}</strong>
            </>
          ) : (
            <>
              <strong>Quality check added</strong>
              <span>
                {flow.connections
                  .filter((e) => e.toTask === change.stepId)
                  .map((e) => {
                    const source = flow.steps.find((s) => s.id === e.fromTask);
                    return `${source ? stepLabel(source) : e.fromTask} · ${e.fromPort} → reads`;
                  })
                  .join(', ')}
              </span>
              <span>Outputs: {selectedStep?.outputs.map((o) => o.name).join(', ')}</span>
            </>
          )}
        </article>
      </div>
      <small className="review-fact-note">
        Checked by Gobble · Design changes, not execution status
      </small>
    </section>
  );
}
