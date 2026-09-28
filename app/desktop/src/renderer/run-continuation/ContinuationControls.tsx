import { useEffect, useRef, useState } from 'react';
import type { Continuation, LaunchReview } from '@gobble/contracts';
import { requestId } from '../workspace/useWorkspace';

export function ContinuationControls({
  launch,
  latest,
  ready,
  status,
  onRefresh,
}: {
  launch: LaunchReview;
  latest?: Continuation | undefined;
  ready: boolean;
  status: string | null;
  onRefresh: () => void;
}) {
  const [support, setSupport] = useState<{ supported: boolean; reason: string } | null>(null);
  const [issue, setIssue] = useState('');
  const [busy, setBusy] = useState(false);
  const flight = useRef(false);
  useEffect(() => {
    let active = true;
    void window.gobble.continuations
      .support({ projectId: launch.projectId, launchReviewId: launch.requestId })
      .then((r) => {
        if (active) {
          if (r.ok) setSupport(r.value);
          else setIssue(r.error.message);
        }
      })
      .catch(() => {
        if (active) setIssue('Engine support could not be checked. Reopen this Run to retry.');
      });
    return () => {
      active = false;
    };
  }, [launch.projectId, launch.requestId]);
  const reviewKey = `${latest?.requestId ?? ''}:${latest?.state ?? ''}:${latest?.operation?.state ?? ''}`;
  const observedKey = useRef(reviewKey);
  useEffect(() => {
    if (observedKey.current === reviewKey) return;
    observedKey.current = reviewKey;
    if (latest?.state === 'ready' || latest?.state === 'accepted' || latest?.state === 'blocked')
      onRefresh();
  }, [reviewKey, latest?.state, onRefresh]);
  const unresolved =
    latest?.operation && ['accepted', 'dispatching', 'unknown'].includes(latest.operation.state);
  const checking = latest?.state === 'checking';
  const canCheck =
    ready && support?.supported && status === 'stopped' && !checking && !unresolved && !busy;
  async function check() {
    if (!canCheck || flight.current || !launch.operation?.runRef) return;
    flight.current = true;
    setBusy(true);
    setIssue('');
    try {
      const result = await window.gobble.continuations.check({
        projectId: launch.projectId,
        launchReviewId: launch.requestId,
        runRef: launch.operation.runRef,
        requestId: requestId(),
      });
      if (!result.ok) setIssue(result.error.message);
    } catch {
      setIssue('Check acknowledgement unavailable. Wait for the saved review to reconnect.');
    } finally {
      flight.current = false;
      setBusy(false);
    }
  }
  function guidance(): string {
    if (issue) return issue;
    if (support && !support.supported)
      return support.reason || 'This analysis uses an earlier engine.';
    if (checking || busy) return 'Checking saved data, results and execution state…';
    if (unresolved) return 'Confirmation outcome is unresolved. Check the same request in Chat.';
    if (status === 'succeeded')
      return 'Execution is complete. This saved continuation review is no longer actionable.';
    if (status === 'stopping')
      return 'Stop is still settling. Refresh before checking continuation.';
    if (status !== 'stopped') return 'Continuation can be checked after this analysis has stopped.';
    if (latest?.state === 'blocked' || latest?.state === 'interrupted')
      return latest.issue || 'Review unavailable. Check again.';
    if (latest?.state === 'ready')
      return 'Review the kept and restarted steps. Confirm in Chat when ready.';
    if (!support) return 'Checking engine support…';
    return 'Check which results can be kept. Nothing runs until you confirm in Chat.';
  }
  // Keep the observation effects mounted even when settled completion needs no controls.
  if (
    status === 'succeeded' &&
    !unresolved &&
    !checking &&
    !busy &&
    !issue &&
    !latest?.issue &&
    latest?.state !== 'ready'
  )
    return null;
  return (
    <div className="continuation-controls">
      <div>
        <strong>Continue this analysis</strong>
        <p role="status">{guidance()}</p>
      </div>
      <button disabled={!canCheck} onClick={() => void check()}>
        {latest ? 'Recheck continuation' : 'Check continuation'}
      </button>
    </div>
  );
}
