import { useEffect, useState } from 'react';
import type { LaunchReview, PipelineInspection } from '@gobble/contracts';
import { FollowUpOrigin } from './FollowUpOrigin';
export function RunFeedbackControls({
  review,
  ready,
  onOpenCurrent,
}: {
  review: LaunchReview;
  ready: boolean;
  onOpenCurrent: (pipelineId: string) => Promise<boolean>;
}) {
  const [current, setCurrent] = useState<PipelineInspection | null>(null);
  const [issue, setIssue] = useState('');
  const [busy, setBusy] = useState(false);
  useEffect(() => {
    let active = true;
    setCurrent(null);
    setIssue('');
    void window.gobble.pipelines
      .inspection({ projectId: review.projectId, pipelineId: review.pipelineId })
      .then((r) => {
        if (active) {
          if (r.ok) setCurrent(r.value);
          else setIssue(r.error.message);
        }
      })
      .catch(() => {
        if (active) setIssue('Current design is unavailable. Reopen this analysis to check again.');
      });
    return () => {
      active = false;
    };
  }, [review.projectId, review.pipelineId, review.requestId]);
  const earlier =
    current?.artifact && current.artifact.artifactId !== review.preparation.artifactId;
  return (
    <div className="run-feedback-controls">
      {review.followUp && <FollowUpOrigin value={review.followUp} />}
      {earlier && (
        <p>This analysis used an earlier design. Its saved Flow and evidence remain unchanged.</p>
      )}
      {review.operation?.runStatus === 'failed' && (
        <p>
          Observed failure does not establish its cause. Share the step or a specific log excerpt
          for investigation.
        </p>
      )}
      <button
        disabled={!ready || busy}
        onClick={() => {
          setBusy(true);
          void onOpenCurrent(review.pipelineId)
            .catch(() => setIssue('Current could not be opened. Try again.'))
            .finally(() => setBusy(false));
        }}
      >
        Open current design
      </button>
      {issue && <p role="status">{issue}</p>}
    </div>
  );
}
