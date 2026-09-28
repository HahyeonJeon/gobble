import { useRef, useState } from 'react';
import type { Continuation, LaunchReview } from '@gobble/contracts';
import { continuationStepLabel, pipelineStepLabel } from '@gobble/contracts';
import { requestId } from '../workspace/useWorkspace';

/** Only an explicit User gesture reaches confirm. A review ID also keys its one confirmation. */
export function ContinuationActionCard({
  value: v,
  launch,
  available,
  onOpen,
}: {
  value: Continuation;
  launch: LaunchReview;
  available: boolean;
  onOpen: () => Promise<void>;
}) {
  const [busy, setBusy] = useState(false),
    [issue, setIssue] = useState(''),
    [uncertain, setUncertain] = useState(false);
  const flight = useRef(false),
    stopId = useRef('');
  const op = v.operation,
    review = v.review;
  const labels = new Map(
    launch.preparation.state === 'ready'
      ? launch.preparation.prepared.flow.steps.map((s) => [s.id, pipelineStepLabel(s)])
      : [],
  );
  async function act(kind: 'confirm' | 'refresh' | 'stop') {
    if (flight.current || !available) return;
    flight.current = true;
    setBusy(true);
    setIssue('');
    try {
      const base = { projectId: v.projectId, reviewId: v.requestId };
      const r =
        kind === 'confirm' && review
          ? await window.gobble.continuations.confirm({
              ...base,
              requestId: v.requestId,
              reviewDigest: review.digest,
            })
          : kind === 'stop'
            ? await window.gobble.continuations.stop({
                ...base,
                requestId: op?.stopRequestId || (stopId.current ||= requestId()),
                expectedLease: op?.receipt?.lease ?? '',
              })
            : await window.gobble.continuations.refresh(base);
      if (kind === 'confirm') setUncertain(true);
      if (!r.ok) setIssue(r.error.message);
    } catch {
      setIssue(
        'Acknowledgement unavailable. Check this request; do not submit another confirmation.',
      );
      if (kind === 'confirm') setUncertain(true);
    } finally {
      flight.current = false;
      setBusy(false);
    }
  }
  const kept = review?.steps.filter((s) => s.action === 'reuse').length ?? 0;
  const restarted = review?.steps.filter((s) => s.action === 'restart').length ?? 0;
  const started = review?.steps.filter((s) => s.action === 'start').length ?? 0;
  const summary = [
    kept ? `Keep ${kept} result${kept === 1 ? '' : 's'}` : '',
    restarted ? `Restart ${restarted} step${restarted === 1 ? '' : 's'}` : '',
    started ? `Start ${started} step${started === 1 ? '' : 's'}` : '',
  ]
    .filter(Boolean)
    .join(' · ');
  const observation = op?.observation;
  const owns = op?.receipt && observation?.epoch.lease === op.receipt.lease;
  const canStop = owns && observation?.status === 'running' && !op?.stopRequestId;
  const canConfirm = v.state === 'ready' && !op && !uncertain && available;
  const title = op
    ? op.state === 'admitted'
      ? 'Continuation confirmed'
      : op.state === 'rejected'
        ? 'Continuation was not admitted'
        : 'Checking confirmation outcome…'
    : v.state === 'checking'
      ? 'Checking continuation…'
      : v.state === 'ready'
        ? 'Resume this analysis?'
        : 'Continuation needs review';
  return (
    <article className="launch-action-card continuation-action-card" aria-label="Continue analysis">
      <header>
        <strong>{title}</strong>
      </header>
      {review && (
        <>
          <p>
            <strong>{summary}</strong>
          </p>
          {op && <small>Saved confirmation plan · Execution status is shown below.</small>}
          <ul>
            {review.steps.map((s) => (
              <li key={s.taskId}>
                {labels.get(s.taskId) ?? s.taskId} · {continuationStepLabel(s)}
              </li>
            ))}
          </ul>
          <p>
            Uses this Run’s saved design and checked data. Restarted steps run from their beginning.
          </p>
          {!op && <small>Gobble checks this plan again before starting.</small>}
          <p className="launch-path">
            {launch.preparation.state === 'ready'
              ? launch.preparation.prepared.input.relativePath
              : ''}
          </p>
        </>
      )}
      {observation && (
        <p>
          Last checked execution: {observation.status}
          {!owns ? ' · Execution owner changed' : ''}
        </p>
      )}
      {op?.stopState && (
        <p role="status">
          {
            {
              requested: 'Stop requested · Check status to confirm',
              settled: 'Stop settled',
              'owner-changed': 'Execution owner changed · Stop not sent',
              'recovery-required': 'Stop needs recovery review',
            }[op.stopState]
          }
        </p>
      )}
      {(issue || op?.issue || v.issue) && <p role="status">{issue || op?.issue || v.issue}</p>}
      {uncertain && !op && (
        <p role="status">Confirmation may have been saved. Refresh this request to reconnect.</p>
      )}
      {!available && <p role="status">Reconnect to the saved review before confirming.</p>}
      <footer>
        {!op && v.state === 'ready' && (
          <button
            className="primary"
            disabled={busy || !canConfirm}
            onClick={() => void act('confirm')}
          >
            Resume analysis
          </button>
        )}
        <button disabled={busy} onClick={() => void onOpen()}>
          Open Run
        </button>
        {(op || uncertain) && (
          <button disabled={busy || !available} onClick={() => void act('refresh')}>
            Check continuation status
          </button>
        )}
        {canStop && (
          <button disabled={busy || !available} onClick={() => void act('stop')}>
            Stop analysis
          </button>
        )}
      </footer>
    </article>
  );
}
