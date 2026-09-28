import type { SavedReportRecord } from '@gobble/contracts';
import '../styles/report.css';
import { ContinuationControls } from '../run-continuation/ContinuationControls';
import { ContinuationContext } from '../run-continuation/ContinuationContext';
import { latestContinuation, useContinuations } from '../run-continuation/useContinuations';
import { continuationStepLabel } from '@gobble/contracts';
import { RunFeedbackControls } from '../run-feedback/RunFeedbackControls';
import { useEffect, useState } from 'react';
import type { ComponentProps } from 'react';
import { referenceAuthor } from '@gobble/contracts';
import { PipelineFlow } from '../workspace/views/pipeline/PipelineFlow';
import type { RunTasksView } from '../workspace/views/RunTasksView';
import { useLaunchReviews } from './useLaunchReviews';
import '../styles/pipeline.css';
import '../styles/run-launch.css';
/** A Run uses its admitted prepared Flow, never the Pipeline's newer Current. */
export function RetainedRunFlow({
  value,
  savedReports,
  onOpenReport,
  onOpenSavedReport,
  selection,
  ready,
  onSelect,
  onDiscuss,
  onRefresh,
  onOpenCurrent,
  marks = [],
}: Pick<
  ComponentProps<typeof RunTasksView>,
  'value' | 'selection' | 'ready' | 'onSelect' | 'onDiscuss' | 'onRefresh' | 'marks'
> & {
  savedReports: SavedReportRecord[];
  onOpenReport: (instance: string, attempt: number) => Promise<boolean>;
  onOpenSavedReport: (saved: SavedReportRecord) => Promise<boolean>;
  onOpenCurrent: (pipelineId: string) => Promise<boolean>;
}) {
  const [opening, setOpening] = useState(false);
  const continuations = useContinuations(value.projectId);
  const latest = latestContinuation(continuations.values, value.runRef);
  const context = value.continuation;
  const planCurrent =
    context &&
    context.reviewId === latest?.requestId &&
    latest.state === 'ready' &&
    !continuations.issue;
  const { values } = useLaunchReviews(value.projectId);
  const review = values.find((v) => v.operation?.runRef === value.runRef);
  useEffect(() => {
    if (!review || !['running', 'stopping'].includes(value.status ?? '')) return;
    const timer = setInterval(onRefresh, 4000);
    return () => clearInterval(timer);
  }, [!!review, value.status, onRefresh]);
  if (!review || review.preparation.state !== 'ready') return null;
  const prepared = review.preparation.prepared;
  const flow = prepared.flow;
  const observed = (id: string) => {
    const tasks = value.tasks.filter((t) => t.taskId === id && !t.template);
    return tasks.length === 1 ? tasks[0] : undefined;
  };
  const selected =
    selection?.kind === 'run-task'
      ? value.tasks.find(
          (t) => t.instanceId === selection.instanceId && t.attempt === selection.attempt,
        )
      : undefined;
  const quality = value.tasks.find((t) => !t.template && t.taskId === prepared.steps[1]?.id);
  const saved = savedReports.filter((r) => r.runRef === value.runRef);
  const currentSaved =
    quality &&
    saved.find(
      (r) => r.producer.instance === quality.instanceId && r.producer.attempt === quality.attempt,
    );
  async function openReport() {
    if (!quality || opening) return;
    setOpening(true);
    try {
      await onOpenReport(quality.instanceId, quality.attempt);
    } finally {
      setOpening(false);
    }
  }
  const statuses = Object.fromEntries(
    flow.steps.map((s) => [s.id, observed(s.id)?.status ?? 'unobserved']),
  );
  return (
    <section className="retained-run-flow" aria-label="Run flow">
      <header>
        <strong>Run flow</strong>
        <span>
          {value.status === 'succeeded'
            ? 'Execution complete'
            : (value.status ?? 'Status unavailable')}{' '}
          · Saved design
        </span>
      </header>
      <PipelineFlow
        flow={flow}
        list={false}
        statuses={statuses}
        plan={
          context
            ? Object.fromEntries(
                context.review.steps.map((s) => [
                  s.taskId,
                  { action: s.action, label: continuationStepLabel(s) },
                ]),
              )
            : undefined
        }
        selected={selected?.taskId ? { kind: 'step', id: selected.taskId } : null}
        onSelect={(target) => {
          if (!ready || target.kind !== 'step') return;
          const t = observed(target.id);
          if (t)
            void onSelect({
              kind: 'run-task',
              coordinateSpace: 'observed-instance-attempt',
              instanceId: t.instanceId,
              attempt: t.attempt,
            });
        }}
        selectableSteps={ready ? flow.steps.filter((s) => observed(s.id)).map((s) => s.id) : []}
        marks={marks.flatMap((m) => {
          const s = m.evidence.selection;
          if (s?.kind !== 'run-task') return [];
          const t = value.tasks.find(
            (t) => t.instanceId === s.instanceId && t.attempt === s.attempt,
          );
          return t?.taskId
            ? [
                {
                  target: { kind: 'step' as const, id: t.taskId },
                  label: referenceAuthor(m),
                  author: m.author.kind === 'agent' ? ('agent' as const) : ('user' as const),
                },
              ]
            : [];
        })}
      />
      <div className="report-actions">
        {quality?.status === 'succeeded' && (
          <button disabled={!ready || opening} onClick={() => void openReport()}>
            {opening
              ? 'Opening report…'
              : currentSaved
                ? 'Open saved quality report'
                : 'Open quality report'}
          </button>
        )}
        {saved.length > 0 && (
          <details>
            <summary>Saved reports ({saved.length})</summary>
            {saved.map((r) => (
              <button
                key={r.asset.hash}
                disabled={!ready || opening}
                onClick={() => void onOpenSavedReport(r)}
              >
                {r.title} · attempt {r.producer.attempt}
              </button>
            ))}
          </details>
        )}
      </div>
      <RunFeedbackControls
        key={review.requestId}
        review={review}
        ready={ready}
        onOpenCurrent={onOpenCurrent}
      />
      <ContinuationControls
        key={review.requestId}
        launch={review}
        latest={latest}
        ready={ready && continuations.loaded && !continuations.issue}
        status={value.status}
        onRefresh={onRefresh}
      />
      {continuations.issue && (
        <p className="continuation-notice" role="status">
          {continuations.issue}
        </p>
      )}
      {context && !planCurrent && (
        <p className="continuation-notice" role="status">
          Saved review shown. Refresh the Run to see the latest review.
        </p>
      )}
      {context && selected && (
        <div className="continuation-selection">
          <ContinuationContext
            context={context}
            labels={Object.fromEntries(
              value.tasks.map((t) => [t.taskId ?? t.instanceId, t.name ?? t.instanceId]),
            )}
            taskId={selected.taskId}
          />
          <button
            disabled={!ready || !planCurrent}
            onClick={() => selection && void onDiscuss(selection)}
          >
            Discuss this step
          </button>
        </div>
      )}
    </section>
  );
}
