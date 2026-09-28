import { ContinuationContext } from '../../run-continuation/ContinuationContext';
import { pipelineSubject } from '@gobble/contracts';
import { PipelineSubjectContent } from '../../evidence/PipelineEvidence';
import { NotebookReferenceContent } from './notebook/NotebookReferenceContent';
import { PdfReferenceContent } from './pdf/PdfReferenceContent';
import type { RenderAcknowledgment } from '@gobble/contracts';
import { DependencyReferenceContent } from './dependencies/DependencyReferenceContent';
import { useEffect, useRef } from 'react';
import {
  referenceAuthor,
  activeLogStream,
  type SurfaceData,
  type ObservedReferenceView,
  type SharedReference,
} from '@gobble/contracts';
import type { Command } from '../useWorkspace';
import { TextHighlights, markStyle, evidenceLabel } from '../../shared-context/marks';

/** Temporary read-only presentation; the base presenter stays mounted with its native gesture/scroll. */
export function ObservedReferencePanel({
  view,
  data,
  reference,
  onReady,
  command,
  acknowledgment,
  ready = false,
  onFailure,
}: {
  view: ObservedReferenceView;
  data: SurfaceData;
  reference: SharedReference;
  onReady: () => void;
  command: Command;
  acknowledgment?: RenderAcknowledgment;
  ready?: boolean;
  onFailure?: (message: string) => void;
}) {
  const scroll = useRef<HTMLDivElement>(null);
  useEffect(() => {
    if (data.kind !== 'file') onReady();
  }, [onReady, data.kind]);
  useEffect(() => {
    const root = scroll.current,
      mark = root?.querySelector<HTMLElement>('[data-reference-ids], [data-target="true"]');
    if (root && mark) root.scrollTop = Math.max(0, mark.offsetTop - root.offsetTop - 40);
  }, [view.requestId]);
  if (data.kind === 'report') return <p role="alert">Report discussion is not available yet.</p>;
  if (data.kind === 'pipeline') {
    if (!data.value.artifact || view.evidence.schemaVersion !== 7)
      return <p role="alert">Pipeline reference unavailable.</p>;
    return (
      <div
        className="observed-reference-panel pipeline-reference-panel"
        style={markStyle(reference)}
      >
        <div className="reference-view-banner">
          <span>
            Reference view · {referenceAuthor(reference)}
            <small>Your selection and view are saved.</small>
          </span>
          <button
            onClick={() =>
              void command({ kind: 'returnReferenceView', referenceRequestId: view.requestId })
            }
          >
            Return to my view
          </button>
        </div>
        <PipelineSubjectContent
          subject={pipelineSubject(data.value.artifact.flow, view.evidence.selection.subject)}
        />
        {reference.note && <p>{reference.note}</p>}
      </div>
    );
  }
  const target = view.evidence.selection;
  const stream =
    data.kind === 'log' && 'streams' in data.value
      ? activeLogStream(target?.kind === 'log-text' ? target.stream : undefined, data.value)
      : 'legacy';
  return (
    <div className="data-view observed-reference-panel" style={markStyle(reference)}>
      <div className="reference-view-banner" role="status">
        <span>
          <strong>Reference view · {referenceAuthor(reference)}</strong>
          <small>{evidenceLabel(view.evidence)} · Your selection and view are saved.</small>
        </span>
        <button
          onClick={() =>
            void command({ kind: 'returnReferenceView', referenceRequestId: view.requestId })
          }
        >
          Return to my view
        </button>
      </div>
      {data.kind === 'file' &&
      data.value.content.kind === 'notebook' &&
      data.value.content.reference &&
      onFailure ? (
        <NotebookReferenceContent
          key={view.requestId}
          excerpt={data.value.content.reference}
          onReady={onReady}
          onFailure={onFailure}
        />
      ) : data.kind === 'file' &&
        data.value.content.kind === 'pdf' &&
        acknowledgment &&
        onFailure ? (
        <PdfReferenceContent
          key={view.requestId}
          content={data.value.content}
          reference={reference}
          acknowledgment={acknowledgment}
          ready={ready}
          onReady={onReady}
          onFailure={onFailure}
        />
      ) : (
        <div
          ref={scroll}
          className="observed-reference-scroll"
          tabIndex={0}
          aria-label="Agent reference content"
        >
          {data.kind === 'run' && data.dependencies && view.evidence.schemaVersion === 4 ? (
            <DependencyReferenceContent
              key={view.requestId}
              data={data}
              observation={data.dependencies}
              reference={reference}
              target={view.evidence.selection}
            />
          ) : data.kind === 'run' ? (
            <>
              {data.value.continuation && (
                <ContinuationContext
                  context={data.value.continuation}
                  labels={Object.fromEntries(
                    data.value.tasks.map((t) => [t.taskId ?? t.instanceId, t.name ?? t.instanceId]),
                  )}
                  taskId={
                    target?.kind === 'run-task'
                      ? data.value.tasks.find((t) => t.instanceId === target.instanceId)?.taskId
                      : undefined
                  }
                />
              )}
              <table className="task-table">
                <thead>
                  <tr>
                    <th>Task instance</th>
                    <th>State</th>
                    <th>Attempt</th>
                  </tr>
                </thead>
                <tbody>
                  {data.value.tasks.map((task) => (
                    <tr
                      key={task.instanceId}
                      data-target={
                        target?.kind === 'run-task' &&
                        task.instanceId === target.instanceId &&
                        task.attempt === target.attempt
                      }
                    >
                      <td>
                        {task.name || task.instanceId}
                        <small>{task.instanceId}</small>
                      </td>
                      <td>{task.status || 'Unavailable'}</td>
                      <td>{task.attempt}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </>
          ) : data.kind === 'log' && 'streams' in data.value && stream !== 'legacy' ? (
            <>
              <p className="observation-notice">
                {stream} · {data.value.instance} · Attempt {data.value.attempt}
              </p>
              <pre className="observed-reference-text">
                <TextHighlights text={data.value.streams[stream].text} marks={[reference]} />
              </pre>
            </>
          ) : null}
        </div>
      )}
      <div className="observation-notice">
        {data.kind === 'file'
          ? data.value.content.kind === 'notebook'
            ? 'Exact Notebook version · Authored pointer'
            : 'Exact PDF version · Authored pointer'
          : 'Observed ' +
            new Date(data.value.observedAt).toLocaleTimeString('en') +
            ' · Pointer to this observation'}
      </div>
    </div>
  );
}
