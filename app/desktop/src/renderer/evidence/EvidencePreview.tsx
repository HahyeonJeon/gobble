import { ReportView } from '../workspace/views/ReportView';
import type { SavedReport } from '@gobble/contracts';
import { ContinuationContext } from '../run-continuation/ContinuationContext';
import { PipelineEvidence } from './PipelineEvidence';
import { DependencyEvidence } from './DependencyEvidence';
import { useEffect, useState } from 'react';
import type {
  EvidenceManifest,
  EvidencePreview as Content,
  PreviewEvidence,
} from '@gobble/contracts';

function Image({ base64, label }: { base64: string; label: string }) {
  const [url, setUrl] = useState<string>();
  useEffect(() => {
    const bytes = Uint8Array.from(atob(base64), (value) => value.charCodeAt(0));
    const next = URL.createObjectURL(new Blob([bytes], { type: 'image/png' }));
    setUrl(next);
    return () => URL.revokeObjectURL(next);
  }, [base64]);
  return url ? <img src={url} alt={'Attached preview of ' + label} /> : <p>Loading image…</p>;
}
export function EvidencePreview({
  request,
  manifest,
}: {
  request: PreviewEvidence;
  manifest: EvidenceManifest;
}) {
  const [state, setState] = useState<{ content: Content } | { error: string } | null>(null);
  const [retry, setRetry] = useState(0);
  const requestKey = JSON.stringify(request);
  useEffect(() => {
    let active = true;
    setState(null);
    void window.gobble.evidence
      .preview(JSON.parse(requestKey) as PreviewEvidence)
      .then((result) => {
        if (active)
          setState(result.ok ? { content: result.value } : { error: result.error.message });
      })
      .catch(() => {
        if (active) setState({ error: 'Evidence unavailable. Try opening this preview again.' });
      });
    return () => {
      active = false;
    };
  }, [requestKey, retry]);
  const rep = manifest.representation;
  const table =
    state && 'content' in state && state.content.kind === 'table' ? state.content : undefined;
  const plot = table?.presentation;
  return (
    <div className="evidence-preview" aria-label={'Attachment preview: ' + manifest.label}>
      <small>
        {manifest.capture
          ? manifest.capture.kind === 'report'
            ? 'Saved report'
            : manifest.capture.kind === 'notebook'
              ? 'Captured Notebook content'
              : manifest.capture.kind === 'pdf'
                ? 'Captured PDF page'
                : 'Captured observation'
          : request.kind === 'question'
            ? 'Saved question evidence'
            : request.kind === 'sent'
              ? 'Saved content at send time'
              : 'Content to send'}{' '}
        · {new Date(manifest.capturedAt).toLocaleTimeString('en')}
      </small>
      <div className="evidence-preview-body" tabIndex={0}>
        {!state ? (
          <p>Loading attachment…</p>
        ) : 'error' in state ? (
          <div>
            <p role="alert">{state.error}</p>
            <button onClick={() => setRetry((value) => value + 1)}>Retry preview</button>
          </div>
        ) : state.content.kind === 'report' ? (
          <ReportPreview report={state.content.report} />
        ) : state.content.kind === 'image' ? (
          <Image base64={state.content.base64} label={manifest.label} />
        ) : state.content.kind === 'pipeline' ? (
          <PipelineEvidence capture={state.content.pipeline} />
        ) : 'dependency' in state.content ? (
          <DependencyEvidence capture={state.content.dependency} />
        ) : state.content.kind === 'table' ? (
          <>
            {state.content.presentation && (
              <div className="evidence-plot-context">
                <strong>Plot context</strong>
                <p>
                  {table?.columns.find((column) => column.id === plot?.spec.xColumnId)?.name} ×{' '}
                  {table?.columns.find((column) => column.id === plot?.spec.yColumnId)?.name}
                </p>
                <small>Semantic rows and view settings · Plot pixels are not included</small>
                <details>
                  <summary>View settings and scope</summary>
                  <p>
                    Filter:{' '}
                    {state.content.presentation.filter.kind === 'all'
                      ? 'All rows'
                      : `${state.content.presentation.filter.columnId} = ${state.content.presentation.filter.value}`}
                  </p>
                  <p>
                    X: {state.content.presentation.viewport.x.join(' to ')} · Y:{' '}
                    {state.content.presentation.viewport.y.join(' to ')}
                  </p>
                  {state.content.scope && (
                    <p>
                      {state.content.scope.returnedRows} of {state.content.scope.requestedRows}{' '}
                      requested rows · {state.content.scope.previewRows} source preview rows
                      {state.content.scope.sourceTruncated ? ' · Source preview truncated' : ''}
                    </p>
                  )}
                </details>
              </div>
            )}
            <table>
              <thead>
                <tr>
                  <th scope="col">Row key</th>
                  {state.content.columns.map((column) => (
                    <th key={column.id} scope="col">
                      {column.name}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {state.content.rows.map((row) => (
                  <tr key={row.key}>
                    <th scope="row">{row.key}</th>
                    {row.cells.map((cell, index) => (
                      <td key={index}>{cell}</td>
                    ))}
                  </tr>
                ))}
              </tbody>
            </table>
          </>
        ) : (
          <>
            {'observation' in state.content &&
            state.content.observation?.schemaVersion === 2 &&
            state.content.observation.source.value.continuation ? (
              <ContinuationContext
                context={state.content.observation.source.value.continuation}
                labels={Object.fromEntries(
                  state.content.observation.source.value.tasks.map((t) => [
                    t.taskId ?? t.instanceId,
                    t.name ?? t.instanceId,
                  ]),
                )}
                taskId={
                  state.content.observation.source.value.tasks.length === 1
                    ? state.content.observation.source.value.tasks[0]?.taskId
                    : undefined
                }
              />
            ) : (
              <pre>{state.content.text}</pre>
            )}
          </>
        )}
      </div>
      <small>
        {manifest.capture?.kind === 'report'
          ? 'Whole saved report · Original charts available to the addressed agent'
          : manifest.capture?.kind === 'pipeline'
            ? 'Exact checked pipeline facts · Saved independently of later changes'
            : manifest.capture?.kind === 'notebook'
              ? 'Exact saved selection · Independent of later Notebook changes'
              : manifest.capture?.kind === 'pdf'
                ? 'PDF page ' +
                  (manifest.capture.representation.pdf.pageIndex + 1) +
                  ' · ' +
                  manifest.capture.representation.width +
                  ' × ' +
                  manifest.capture.representation.height +
                  ' pixels · Saved independently of the source'
                : manifest.capture
                  ? rep.kind === 'log'
                    ? 'Captured log excerpt · Source completeness unknown'
                    : 'Captured Run facts · Observed preview'
                  : rep.kind === 'image'
                    ? `${rep.width} × ${rep.height} · Crop ${rep.crop.x}, ${rep.crop.y}, ${rep.crop.width} × ${rep.crop.height} from ${rep.originalWidth} × ${rep.originalHeight}`
                    : rep.truncated
                      ? 'Bounded preview · source content is truncated'
                      : 'Exact selected content or complete available preview'}
      </small>
      <details className="evidence-provenance">
        <summary>Source version</summary>
        {state &&
          'content' in state &&
          'observation' in state.content &&
          state.content.observation && (
            <p>
              Observed{' '}
              {new Date(state.content.observation.source.value.observedAt).toLocaleString('en')} ·
              Saved independently of later runtime changes
            </p>
          )}
        {state && 'content' in state && 'notebook' in state.content && (
          <details>
            <summary>Original source quote</summary>
            <pre>{state.content.notebook.rawQuote}</pre>
          </details>
        )}
        <code>{manifest.evidence.dataRevision}</code>
      </details>
    </div>
  );
}

const reportReady = () => {};
function ReportPreview({ report }: { report: SavedReport }) {
  const [moduleId, setModuleId] = useState<string>();
  const [error, setError] = useState<string>();
  return (
    <>
      <p>Complete text and tables are included. The agent reads original charts on request.</p>
      {error && <p role="alert">{error}</p>}
      <ReportView
        report={report}
        moduleId={moduleId}
        ready={!error}
        onReady={reportReady}
        onFailure={setError}
        onNavigate={async (id) => {
          setModuleId(id);
          return true;
        }}
      />
    </>
  );
}
