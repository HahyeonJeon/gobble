import { ContinuationActionCard } from '../run-continuation/ContinuationActionCard';
import { latestContinuation, useContinuations } from '../run-continuation/useContinuations';
import type { Continuation } from '@gobble/contracts';
import { FollowUpOrigin } from '../run-feedback/FollowUpOrigin';
import { followUpDataRelation } from '@gobble/contracts';
import { useRef, useState } from 'react';
import type { LaunchReview, WorkspaceDocument } from '@gobble/contracts';
import type { Command } from '../workspace/useWorkspace';
import { requestId } from '../workspace/useWorkspace';
import { useLaunchReviews } from './useLaunchReviews';
import '../styles/run-launch.css';
export function LaunchActionCards({
  document,
  command,
}: {
  document: WorkspaceDocument;
  command: Command;
}) {
  const continuations = useContinuations(document.workspace.projectId);
  const { values, issue } = useLaunchReviews(document.workspace.projectId, true);
  const visible = values.filter((v) => v.state === 'ready' || v.state === 'accepted');
  const [history, setHistory] = useState(false);
  return (
    <div className="launch-action-list" aria-label="Run actions">
      {issue && <p role="status">{issue}</p>}
      {visible.length > 1 && (
        <button
          className="launch-history"
          aria-expanded={history}
          onClick={() => setHistory((v) => !v)}
        >
          {history ? 'Hide earlier runs' : `${visible.length - 1} earlier run reviews`}
        </button>
      )}
      {(history ? visible : visible.slice(-1)).map((v) => (
        <LaunchActionCard
          key={v.requestId}
          value={v}
          continuation={latestContinuation(continuations.values, v.operation?.runRef ?? '')}
          continuationAvailable={continuations.loaded && !continuations.issue}
          onOpen={async (kind) => {
            await command({
              kind: 'open',
              resource:
                kind === 'run' && v.operation?.runRef
                  ? { kind: 'run', runRef: v.operation.runRef }
                  : { kind: 'pipeline', pipelineId: v.pipelineId },
              pane: document.activePane,
              duplicate: false,
            });
          }}
        />
      ))}
    </div>
  );
}
function LaunchActionCard({
  value: v,
  continuation,
  continuationAvailable,
  onOpen,
}: {
  value: LaunchReview;
  continuation?: Continuation | undefined;
  continuationAvailable: boolean;
  onOpen: (kind: 'review' | 'run') => Promise<void>;
}) {
  const [busy, setBusy] = useState(false),
    [issue, setIssue] = useState('');
  const request = useRef<{ start?: string; stop?: string }>({});
  const op = v.operation;
  async function act(kind: 'start' | 'stop' | 'refresh') {
    if (busy) return;
    setBusy(true);
    setIssue('');
    const base = { projectId: v.projectId, reviewId: v.requestId };
    try {
      const result =
        kind === 'start'
          ? await window.gobble.launches.start({
              ...base,
              requestId: (request.current.start ??= requestId()),
            })
          : kind === 'stop'
            ? await window.gobble.launches.stop({
                ...base,
                requestId: op?.stopRequestId || (request.current.stop ??= requestId()),
                expectedLease: op?.ownerLease ?? '',
              })
            : await window.gobble.launches.refresh(base);
      if (!result.ok) setIssue(result.error.message);
    } catch {
      setIssue('Acknowledgement unavailable. Check status to reconnect to this exact Run.');
    } finally {
      setBusy(false);
    }
  }
  if (continuation)
    return (
      <ContinuationActionCard
        key={continuation.requestId}
        value={continuation}
        launch={v}
        available={continuationAvailable}
        onOpen={() => onOpen('run')}
      />
    );
  const running = op && ['running', 'stopping'].includes(op.runStatus);
  return (
    <article className="launch-action-card" aria-label={op ? 'Analysis run' : 'Confirm analysis'}>
      <header>
        <strong>
          {op
            ? op.state === 'admitted'
              ? `Analysis · ${op.runStatus}`
              : op.state === 'rejected'
                ? 'Start was not admitted'
                : op.state === 'recovery-required'
                  ? 'Run needs attention'
                  : 'Confirming this Run…'
            : v.followUp
              ? 'Start a new analysis?'
              : 'Start this analysis?'}
        </strong>
      </header>
      <p>{v.preparation.state === 'ready' ? v.preparation.prepared.input.relativePath : ''}</p>
      <p>
        {v.readCount.toLocaleString('en-US')} records checked ·{' '}
        {v.preparation.state === 'ready' ? v.preparation.prepared.steps.length : 0} steps
      </p>
      {v.followUp && (
        <>
          <FollowUpOrigin value={v.followUp} />
          <p>
            New analysis · All steps run again · Previous results kept.
            <br />
            {followUpDataRelation(v.inputSHA256, v.followUp)}
          </p>
        </>
      )}
      <p className="launch-path">Results: {v.outputPath}</p>
      {!op && <p>Runs locally using the checked data copy and reviewed settings.</p>}
      {(issue || op?.issue || v.issue) && <p role="status">{issue || op?.issue || v.issue}</p>}
      {op?.stopRequestId && (
        <p role="status">
          {op.stopState === 'settled'
            ? 'Stop settled'
            : op.stopState === 'owner-changed'
              ? 'Execution owner changed · Stop not sent'
              : op.stopState === 'recovery-required'
                ? 'Stop needs recovery review'
                : 'Stop requested · Waiting for confirmation'}
        </p>
      )}
      <footer>
        {!op ? (
          <>
            <button
              disabled={busy || !v.fresh}
              className="primary"
              onClick={() => void act('start')}
            >
              {v.followUp ? 'Start new analysis' : 'Start analysis'}
            </button>
            <button onClick={() => void onOpen('review')}>Back to review</button>
          </>
        ) : (
          <>
            {op.runRef && (
              <button className="primary" onClick={() => void onOpen('run')}>
                Open Run
              </button>
            )}
            {op.state !== 'rejected' && (
              <button disabled={busy} onClick={() => void act('refresh')}>
                Check status
              </button>
            )}
            {continuationAvailable && running && op.ownerLease && !op.stopRequestId && (
              <button disabled={busy} onClick={() => void act('stop')}>
                Stop analysis
              </button>
            )}
          </>
        )}
      </footer>
    </article>
  );
}
