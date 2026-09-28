import { CreationDrafts } from './CreationDrafts';
import { useCallback, useEffect, useRef, useState } from 'react';
import type { DesktopBridge } from '@gobble/contracts';
import { requestId } from '../useWorkspace';
import { Icon } from '../Icon';
import type { OpenResource } from './types';

type PipelinesResult = Awaited<ReturnType<DesktopBridge['pipelines']['list']>>;

export function PipelinesBrowser({
  projectId,
  revision,
  onOpen,
}: {
  projectId: string;
  revision: number;
  onOpen: OpenResource;
}) {
  const [result, setResult] = useState<PipelinesResult | null>(null);
  const [retry, setRetry] = useState(0);
  const onDraftsChanged = useCallback(() => setRetry((n) => n + 1), []);
  const [importing, setImporting] = useState(false);
  const [importIssue, setImportIssue] = useState<string | null>(null);
  const activeProject = useRef<string | null>(projectId);
  useEffect(() => {
    activeProject.current = projectId;
    setImporting(false);
    setImportIssue(null);
    return () => {
      activeProject.current = null;
    };
  }, [projectId]);
  async function importFolder() {
    if (importing) return;
    setImporting(true);
    setImportIssue(null);
    try {
      const result = await window.gobble.pipelines.importFolder({
        projectId,
        requestId: requestId(),
      });
      if (activeProject.current !== projectId) return;
      if (!result.ok) setImportIssue(result.error.message);
      else if (result.value.kind === 'selected') {
        setRetry((value) => value + 1);
        onOpen({ kind: 'pipeline', pipelineId: result.value.pipeline.pipelineId }, 'active');
      }
    } catch {
      if (activeProject.current === projectId)
        setImportIssue(
          'The import could not be confirmed. Refresh the pipeline list before trying again.',
        );
    } finally {
      if (activeProject.current === projectId) setImporting(false);
    }
  }
  useEffect(() => {
    let active = true;
    setResult(null);
    void window.gobble.pipelines.list({ projectId }).then(
      (value) => {
        if (active) setResult(value);
      },
      () => {
        if (active)
          setResult({
            schemaVersion: 1,
            ok: false,
            error: {
              code: 'runtime_unavailable',
              message: 'Pipelines could not be loaded. Try again after reconnecting.',
              retry: 'after_reconnect',
            },
          });
      },
    );
    return () => {
      active = false;
    };
  }, [projectId, revision, retry]);

  return (
    <details open className="resource-section" aria-label="Pipelines">
      <summary>
        <Icon name="pipeline" />
        Pipelines
        {result?.ok && result.value.pipelines.length > 0 && (
          <span className="count muted">{result.value.pipelines.length}</span>
        )}
      </summary>
      <CreationDrafts
        key={projectId}
        projectId={projectId}
        revision={revision}
        onOpen={onOpen}
        onChanged={onDraftsChanged}
      />
      {!result ? (
        <p className="sidebar-note muted">Loading pipelines…</p>
      ) : !result.ok ? (
        <div className="sidebar-note inline-error">
          <p>{result.error.message}</p>
          <button onClick={() => setRetry((value) => value + 1)}>Retry</button>
        </div>
      ) : result.value.pipelines.length === 0 ? (
        <p className="sidebar-note muted">Import an existing analysis to explore its flow.</p>
      ) : (
        <ul className="file-list">
          {result.value.pipelines.map((pipeline) => (
            <li key={pipeline.pipelineId}>
              <button
                aria-label={'Open ' + pipeline.name + ' pipeline'}
                title={'Open ' + pipeline.name + ' flow'}
                onClick={() =>
                  onOpen({ kind: 'pipeline', pipelineId: pipeline.pipelineId }, 'active')
                }
              >
                <Icon name="pipeline" />
                <span className="truncate">{pipeline.name}</span>
              </button>
              <button
                className="icon-button small file-side-action"
                aria-label={'Open ' + pipeline.name + ' pipeline in the other pane'}
                title="Open flow in the other pane"
                onClick={() =>
                  onOpen({ kind: 'pipeline', pipelineId: pipeline.pipelineId }, 'other')
                }
              >
                <Icon name="split" />
              </button>
            </li>
          ))}
        </ul>
      )}
      <button
        className="text-button pipeline-import"
        disabled={importing}
        onClick={() => void importFolder()}
      >
        <Icon name="plus" />
        {importing ? 'Importing…' : 'Import pipeline'}
      </button>
      {importIssue && (
        <p className="sidebar-note inline-error" role="alert">
          {importIssue}
        </p>
      )}
    </details>
  );
}
