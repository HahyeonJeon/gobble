import { RegisterPipeline } from './RegisterPipeline';
import { useEffect, useState } from 'react';
import type { DesktopBridge, ProjectInfo } from '@gobble/contracts';
import { Icon } from '../Icon';
import type { OpenResource, ReportError } from './types';
type FilesResult = Awaited<ReturnType<DesktopBridge['files']['list']>>;

export function FilesBrowser({
  project,
  onOpen,
  report,
  onPipelineRegistered,
}: {
  project: ProjectInfo;
  onOpen: OpenResource;
  report: ReportError;
  onPipelineRegistered: () => void;
}) {
  const [path, setPath] = useState([{ id: project.rootResourceId, name: project.name }]);
  const [result, setResult] = useState<FilesResult | null>(null);
  const [refresh, setRefresh] = useState(0);
  const directory = path.at(-1)?.id ?? project.rootResourceId;
  useEffect(() => {
    let active = true;
    setResult(null);
    void window.gobble.files
      .list({ projectId: project.projectId, directoryId: directory })
      .then((value) => {
        if (active) setResult(value);
      })
      .catch(() => {
        if (active) report('Files could not be loaded. Refresh this folder to try again.');
      });
    return () => {
      active = false;
    };
  }, [project.projectId, directory, refresh, report]);
  return (
    <details open className="resource-section">
      <summary>
        <Icon name="folder" />
        Files
      </summary>
      <div className="browser-toolbar">
        <button
          className="text-button truncate"
          disabled={path.length === 1}
          title={path.map((item) => item.name).join(' / ')}
          onClick={() => setPath((items) => items.slice(0, -1))}
        >
          {path.length > 1 ? '← ' + path.at(-1)?.name : 'Project files'}
        </button>
        <button
          className="icon-button small"
          aria-label="Refresh files"
          onClick={() => setRefresh((value) => value + 1)}
        >
          <Icon name="refresh" />
        </button>
      </div>
      {!result ? (
        <p className="sidebar-note muted">Loading files…</p>
      ) : !result.ok ? (
        <div className="sidebar-note inline-error">
          <p>{result.error.message}</p>
          <button onClick={() => setRefresh((value) => value + 1)}>Retry</button>
          <p>Use Open folder to locate a moved Project.</p>
        </div>
      ) : (
        <>
          {result.value.entries.some(
            (entry) =>
              entry.kind === 'file' &&
              entry.name.endsWith('.go') &&
              !entry.name.endsWith('_test.go'),
          ) && (
            <RegisterPipeline
              key={directory}
              projectId={project.projectId}
              packageResourceId={directory}
              name={path.at(-1)?.name ?? project.name}
              onRegistered={onPipelineRegistered}
            />
          )}
          <ul className="file-list">
            {result.value.entries.map((entry) => (
              <li key={entry.resourceId}>
                <button
                  title={entry.name}
                  disabled={entry.kind === 'unsupported'}
                  onClick={() =>
                    entry.kind === 'directory'
                      ? setPath((items) => [...items, { id: entry.resourceId, name: entry.name }])
                      : onOpen({ kind: 'file', resourceId: entry.resourceId }, 'active')
                  }
                >
                  <Icon
                    name={
                      entry.kind === 'directory'
                        ? 'folder'
                        : entry.name.toLowerCase().endsWith('.csv')
                          ? 'table'
                          : /\.(png|jpe?g)$/i.test(entry.name)
                            ? 'image'
                            : 'file'
                    }
                  />
                  <span className="truncate">{entry.name}</span>
                </button>
                {entry.kind === 'file' && (
                  <button
                    className="icon-button small file-side-action"
                    aria-label={'Open ' + entry.name + ' in the other pane'}
                    title="Open in the other pane"
                    onClick={() => onOpen({ kind: 'file', resourceId: entry.resourceId }, 'other')}
                  >
                    <Icon name="split" />
                  </button>
                )}
              </li>
            ))}
          </ul>
          {result.value.entries.length === 0 && (
            <p className="sidebar-note muted">This folder is empty.</p>
          )}
          {result.value.truncated && (
            <p className="sidebar-note muted">
              Showing up to 500 entries. More files are not shown.
            </p>
          )}
        </>
      )}
    </details>
  );
}
