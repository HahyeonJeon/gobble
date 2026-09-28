import { useEffect, useState, type ReactNode } from 'react';
import type { PaneId } from '@gobble/contracts';

/** Load views only after this renderer session has reported its currently visible Pane set. */
function usePresentation(
  projectId: string,
  rendererSessionId: string,
  panes: PaneId[],
  report: (message: string) => void,
) {
  const paneKey = panes.join(',');
  const key = projectId + '/' + rendererSessionId + '/' + paneKey;
  const [accepted, setAccepted] = useState<string | null>(null);
  useEffect(() => {
    let current = true;
    const visiblePanes = paneKey ? (paneKey.split(',') as PaneId[]) : [];
    window.gobble.workspace
      .present({ projectId, rendererSessionId, visiblePanes })
      .then((result) => {
        if (!current) return;
        if (result.ok) setAccepted(key);
        else report(result.error.message);
      })
      .catch(() => {
        if (current) report('Workspace presentation is unavailable. Reload the workspace.');
      });
    return () => {
      current = false;
    };
  }, [projectId, rendererSessionId, paneKey, key, report]);
  return accepted === key;
}

/** Keyed by the visible Pane set: returning to an earlier layout always obtains a fresh receipt. */
export function PresentedViews({
  projectId,
  rendererSessionId,
  panes,
  report,
  children,
}: {
  projectId: string;
  rendererSessionId: string;
  panes: PaneId[];
  report: (message: string) => void;
  children: ReactNode;
}) {
  const ready = usePresentation(projectId, rendererSessionId, panes, report);
  return ready ? children : null;
}
