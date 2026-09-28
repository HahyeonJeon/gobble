import { useEffect, useState } from 'react';
import type { PipelineReviewContext } from '@gobble/contracts';
/** A human-readable label for an immutable saved review, never an alias for latest. */
export function PreparationReferenceLabel({
  projectId,
  context,
}: {
  projectId: string;
  context: PipelineReviewContext;
}) {
  const [time, setTime] = useState('');
  useEffect(() => {
    let stopped = false;
    setTime('');
    if (context.preparationId)
      void window.gobble.preparations
        .read({ projectId, pipelineId: context.pipelineId, requestId: context.preparationId })
        .then((result) => {
          if (stopped) return;
          if (result.ok && result.value.state === 'ready')
            setTime(
              new Date(result.value.prepared.checkedAt).toLocaleString('en-US', {
                month: 'short',
                day: 'numeric',
                hour: 'numeric',
                minute: '2-digit',
                second: '2-digit',
              }),
            );
          else setTime('Saved version unavailable');
        });
    return () => {
      stopped = true;
    };
  }, [projectId, context.pipelineId, context.preparationId]);
  return (
    <>
      Run review · {context.preparationSection ?? 'Overview'}
      {time ? ' · ' + time : ''}
    </>
  );
}
