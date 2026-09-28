import { useLayoutEffect, useRef, useState } from 'react';
import type { PaneId, WorkspaceDocument } from '@gobble/contracts';
import type { Command } from './useWorkspace';
import { emptyFlowNavigation, type FlowNavigation } from './views/pipeline/flow-navigation';
import { Pane } from './Pane';
import { Splitter } from './Splitter';
import { PresentedViews } from './PresentedViews';

export function ResourceWorkspace({
  document,
  rendererSessionId,
  command,
  hidden,
  report,
  onBrowse,
  focusSurfaceId,
  onFocusRestored,
}: {
  document: WorkspaceDocument;
  rendererSessionId: string;
  command: Command;
  hidden: boolean;
  report: (message: string) => void;
  onBrowse: (pane: PaneId) => void;
  focusSurfaceId: string | null;
  onFocusRestored: () => void;
}) {
  const navigation = useRef(new Map<string, FlowNavigation>());
  const openPipelines = new Set(
    document.workspace.surfaces
      .filter((surface) => surface.view === 'pipeline')
      .map((surface) => surface.surfaceId),
  );
  for (const id of navigation.current.keys()) {
    if (!openPipelines.has(id)) navigation.current.delete(id);
  }
  function flowNavigation(surfaceId: string): FlowNavigation {
    let value = navigation.current.get(surfaceId);
    if (!value) {
      value = emptyFlowNavigation();
      navigation.current.set(surfaceId, value);
    }
    return value;
  }
  const region = useRef<HTMLDivElement>(null);
  const [short, setShort] = useState(false);
  const [preview, setPreview] = useState<number | null>(null);
  useLayoutEffect(() => {
    const element = region.current;
    if (!element) return;
    const observer = new ResizeObserver(([entry]) => {
      if (entry && entry.contentRect.height > 0) setShort(entry.contentRect.height < 520);
    });
    observer.observe(element);
    return () => observer.disconnect();
  }, []);
  const layout = document.workspace.layout;
  const shownPane = document.maximizedPane ?? (short ? document.activePane : null);
  const panes: PaneId[] = hidden
    ? []
    : layout.kind === 'single'
      ? ['primary']
      : shownPane
        ? [shownPane]
        : ['primary', 'secondary'];
  const fraction = preview ?? (layout.kind === 'split' ? layout.primaryFraction : 0.5);
  const visibleSurfaceIds = panes.flatMap((pane) => {
    const id =
      pane === 'primary'
        ? layout.primary.activeSurfaceId
        : layout.kind === 'split'
          ? layout.secondary.activeSurfaceId
          : null;
    return id ? [id] : [];
  });
  return (
    <div ref={region} className="resource-workspace">
      {layout.kind === 'split' && short && (
        <div className="narrow-pane-switch" aria-label="Visible pane">
          <button
            aria-pressed={document.activePane === 'primary'}
            onClick={() => void command({ kind: 'focusPane', pane: 'primary' })}
          >
            Primary pane
          </button>
          <button
            aria-pressed={document.activePane === 'secondary'}
            onClick={() => void command({ kind: 'focusPane', pane: 'secondary' })}
          >
            Secondary pane
          </button>
          <small>Both panes are kept</small>
        </div>
      )}
      <div
        className="workspace-panes"
        style={{
          gridTemplateRows:
            panes.length === 2
              ? `minmax(0, ${fraction}fr) 12px minmax(0, ${1 - fraction}fr)`
              : 'minmax(0, 1fr)',
        }}
      >
        <PresentedViews
          key={rendererSessionId + '/' + panes.join(',')}
          projectId={document.workspace.projectId}
          rendererSessionId={rendererSessionId}
          panes={panes}
          report={report}
        >
          {panes.includes('primary') && (
            <Pane
              document={document}
              flowNavigation={flowNavigation}
              visibleSurfaceIds={visibleSurfaceIds}
              paneId="primary"
              rendererSessionId={rendererSessionId}
              command={command}
              onBrowse={onBrowse}
              focusSurfaceId={focusSurfaceId}
              onFocusRestored={onFocusRestored}
            />
          )}
          {panes.length === 2 && layout.kind === 'split' && (
            <Splitter
              label="Resize panes"
              controls="primary-pane"
              orientation="horizontal"
              value={layout.primaryFraction}
              min={0.2}
              max={0.8}
              step={0.05}
              unit="fraction"
              onPreview={setPreview}
              onCommit={(primaryFraction) => command({ kind: 'resize', primaryFraction })}
            />
          )}
          {panes.includes('secondary') && (
            <Pane
              document={document}
              flowNavigation={flowNavigation}
              visibleSurfaceIds={visibleSurfaceIds}
              paneId="secondary"
              rendererSessionId={rendererSessionId}
              command={command}
              onBrowse={onBrowse}
              focusSurfaceId={focusSurfaceId}
              onFocusRestored={onFocusRestored}
            />
          )}
        </PresentedViews>
      </div>
    </div>
  );
}
