import { FollowUpOrigin } from '../run-feedback/FollowUpOrigin';
import { followUpDataRelation } from '@gobble/contracts';
import { useRef, useState } from 'react';
import type { PipelinePreparation } from '@gobble/contracts';
import { requestId } from '../workspace/useWorkspace';
import { useLaunchReviews } from './useLaunchReviews';
import '../styles/run-launch.css';
export function LaunchReviewControls({
  preparation,
}: {
  preparation: Extract<PipelinePreparation, { state: 'ready' }>;
}) {
  const { projectId, pipelineId } = preparation;
  const { values, issue: loadIssue } = useLaunchReviews(projectId);
  const value = values.filter((v) => v.preparationId === preparation.requestId).at(-1);
  const [busy, setBusy] = useState(false),
    [issue, setIssue] = useState('');
  const pending = useRef<string | null>(null);
  async function check() {
    if (busy) return;
    setBusy(true);
    setIssue('');
    pending.current ??= requestId();
    try {
      const result = await window.gobble.launches.check({
        projectId,
        pipelineId,
        requestId: pending.current,
        preparationId: preparation.requestId,
      });
      if (result.ok) pending.current = null;
      else setIssue(result.error.message);
    } catch {
      setIssue('Check acknowledgement unavailable. Retry to reconnect to the same check.');
    } finally {
      setBusy(false);
    }
  }
  async function cancel() {
    if (!value || busy) return;
    setBusy(true);
    try {
      const result = await window.gobble.launches.cancel({ projectId, reviewId: value.requestId });
      if (!result.ok) setIssue(result.error.message);
    } finally {
      setBusy(false);
    }
  }
  const checking = value?.state === 'checking';
  return (
    <section className="launch-review-controls" aria-label="Execution check">
      <header>
        <strong>Ready to run?</strong>
        <button
          className="primary"
          disabled={
            busy ||
            checking ||
            !preparation.fresh ||
            (value?.state === 'ready' && value.fresh) ||
            (!!value?.operation &&
              !(
                value.operation.state === 'admitted' &&
                ['succeeded', 'failed', 'stopped'].includes(value.operation.runStatus)
              ))
          }
          onClick={() => void check()}
        >
          {busy ? 'Connecting…' : value ? 'Check again' : 'Check data and tools'}
        </button>
      </header>
      <p>
        Validate a separate data copy and reserve a new result folder. Your original file stays
        available.
      </p>
      {(issue || loadIssue || value?.issue) && (
        <p role="status">{issue || loadIssue || value?.issue}</p>
      )}
      {value && (
        <>
          <p>
            <strong>
              {value.state === 'ready' && value.fresh
                ? 'Checked · Confirm in Chat'
                : value.state === 'accepted'
                  ? 'Start recorded · Follow this Run in Chat'
                  : value.state === 'checking'
                    ? 'Checking data…'
                    : value.state === 'ready'
                      ? 'Review changed · Check again'
                      : value.state}
            </strong>
          </p>
          {checking && (
            <>
              <progress max={value.totalBytes} value={value.copiedBytes} />
              <span> {Math.round((value.copiedBytes / value.totalBytes) * 100)}% copied</span>
            </>
          )}
          {!!value.readCount && (
            <p>
              {value.readCount.toLocaleString('en-US')} FASTQ records checked ·{' '}
              {(value.totalBytes / 1024 / 1024).toFixed(1)} MB copied
            </p>
          )}
          {value.followUp && (
            <>
              <FollowUpOrigin value={value.followUp} />
              <p>
                New analysis · All steps run again · Previous results kept.
                <br />
                {followUpDataRelation(value.inputSHA256, value.followUp)}
              </p>
            </>
          )}
          <p className="launch-path">Results: {value.outputPath}</p>
          {(checking || value.state === 'ready') && (
            <button disabled={busy} onClick={() => void cancel()}>
              Cancel check
            </button>
          )}
        </>
      )}
    </section>
  );
}
