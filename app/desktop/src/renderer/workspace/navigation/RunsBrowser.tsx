import { useLaunchReviews } from '../../run-launch/useLaunchReviews';
import { useEffect, useState } from 'react';
import type { DesktopBridge } from '@gobble/contracts';
import { Icon } from '../Icon';
import { requestId } from '../useWorkspace';
import type { OpenResource, ReportError } from './types';
type RunsResult = Awaited<ReturnType<DesktopBridge['runs']['list']>>;

export function RunsBrowser({
  projectId,
  onOpen,
  report,
}: {
  projectId: string;
  onOpen: OpenResource;
  report: ReportError;
}) {
  const { values: reviews } = useLaunchReviews(projectId);
  // Stable across status polling; only confirmed Run membership invalidates the list.
  const admittedRuns = JSON.stringify(
    [
      ...new Set(
        reviews.flatMap(({ operation }) =>
          operation?.state === 'admitted' && operation.runRef ? [operation.runRef] : [],
        ),
      ),
    ].sort(),
  );
  const [result, setResult] = useState<RunsResult | null>(null);
  const [refresh, setRefresh] = useState(0);
  const [attaching, setAttaching] = useState(false);
  useEffect(() => {
    let active = true;
    setResult(null);
    void window.gobble.runs
      .list({ projectId })
      .then((value) => {
        if (active) setResult(value);
      })
      .catch(() => {
        if (active) report('Runs could not be loaded. Refresh to try again.');
      });
    return () => {
      active = false;
    };
  }, [projectId, refresh, report, admittedRuns]);
  async function attach(workspaceResourceId: string) {
    setAttaching(true);
    try {
      const value = await window.gobble.runs.attach({
        projectId,
        workspaceResourceId,
        requestId: requestId(),
      });
      if (!value.ok) report(value.error.message);
      else setRefresh((count) => count + 1);
    } catch {
      report('This Run could not be attached. Check its recorded runtime and try again.');
    } finally {
      setAttaching(false);
    }
  }
  return (
    <details open className="resource-section">
      <summary>
        <Icon name="run" />
        Runs
      </summary>
      <div className="browser-toolbar">
        <span className="muted">Existing analyses</span>
        <button
          className="icon-button small"
          aria-label="Refresh runs"
          onClick={() => setRefresh((value) => value + 1)}
        >
          <Icon name="refresh" />
        </button>
      </div>
      {!result ? (
        <p className="sidebar-note muted">Loading runs…</p>
      ) : !result.ok ? (
        <p className="sidebar-note inline-error">{result.error.message}</p>
      ) : (
        <>
          <ul className="file-list">
            {result.value.runs.map((run) => (
              <li key={run.runRef}>
                <button onClick={() => onOpen({ kind: 'run', runRef: run.runRef }, 'active')}>
                  <Icon name="run" />
                  <span className="truncate">{run.name}</span>
                </button>
              </li>
            ))}
          </ul>
          {result.value.runs.length === 0 && <p className="sidebar-note muted">No runs attached</p>}
          {result.value.candidates.map((candidate) => (
            <div className="candidate" key={candidate.workspaceResourceId}>
              <span className="truncate">{candidate.name}</span>
              <button
                disabled={attaching}
                onClick={() => void attach(candidate.workspaceResourceId)}
              >
                Attach
              </button>
            </div>
          ))}
          {attaching && <p className="sidebar-note muted">Checking recorded runtime…</p>}
          {result.value.truncated && (
            <p className="sidebar-note muted">More workspace candidates are not shown.</p>
          )}
        </>
      )}
    </details>
  );
}
