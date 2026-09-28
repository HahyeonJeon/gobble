import type { FlowNavigation } from './views/pipeline/flow-navigation';
import { CreationView } from './views/creation/CreationView';
import { useLayoutEffect, useRef, useState } from 'react';
import {
  sameResource,
  viewLinkFor,
  logResource,
  type PaneId,
  type WorkspaceDocument,
} from '@gobble/contracts';
import { Icon } from './Icon';
import type { Command } from './useWorkspace';
import { SurfaceView } from './views/SurfaceView';
import { PaneActions } from './PaneActions';
import { EmptyPane } from './EmptyPane';
import { ErrorBoundary } from './ErrorBoundary';

export function Pane({
  document,
  visibleSurfaceIds,
  paneId,
  rendererSessionId,
  flowNavigation,
  command,
  onBrowse,
  focusSurfaceId,
  onFocusRestored,
}: {
  document: WorkspaceDocument;
  visibleSurfaceIds: string[];
  paneId: PaneId;
  rendererSessionId: string;
  flowNavigation: (surfaceId: string) => FlowNavigation;
  command: Command;
  onBrowse: (pane: PaneId) => void;
  focusSurfaceId: string | null;
  onFocusRestored: () => void;
}) {
  const [refresh, setRefresh] = useState(0);
  const root = useRef<HTMLElement>(null);
  const claiming = useRef(false);
  const pendingFocus = useRef<{ kind: 'activate' | 'close'; surfaceId: string } | null>(null);
  const layout = document.workspace.layout;
  const pane =
    paneId === 'primary' ? layout.primary : layout.kind === 'split' ? layout.secondary : null;
  useLayoutEffect(() => {
    if (focusSurfaceId && pane?.activeSurfaceId === focusSurfaceId) {
      root.current
        ?.querySelector<HTMLButtonElement>('button[role="tab"][aria-selected="true"]')
        ?.focus();
      onFocusRestored();
      return;
    }
    const pending = pendingFocus.current;
    if (!pending || !pane) return;
    const applied =
      pending.kind === 'activate'
        ? pane.activeSurfaceId === pending.surfaceId
        : !pane.tabs.includes(pending.surfaceId);
    if (!applied) return;
    pendingFocus.current = null;
    (
      root.current?.querySelector<HTMLButtonElement>('button[role="tab"][aria-selected="true"]') ??
      root.current
    )?.focus();
  }, [pane, focusSurfaceId, onFocusRestored]);
  if (!pane) return null;
  const active = document.workspace.surfaces.find(
    (surface) => surface.surfaceId === pane.activeSurfaceId,
  );
  const title =
    document.titles.find((item) => item.surfaceId === active?.surfaceId)?.title ?? 'View';
  const link = active ? viewLinkFor(document, active.surfaceId) : undefined;
  const other = paneId === 'primary' ? 'secondary' : 'primary';
  const titleFor = (id: string) =>
    document.titles.find((item) => item.surfaceId === id)?.title ?? 'View';
  function claimPane() {
    if (
      claiming.current ||
      (document.activePane === paneId &&
        !(active?.openedBy.kind === 'agent' && !active.userClaimed))
    )
      return;
    claiming.current = true;
    void command({ kind: 'focusPane', pane: paneId }).finally(() => {
      claiming.current = false;
    });
  }
  async function focusTab(id: string) {
    pendingFocus.current = { kind: 'activate', surfaceId: id };
    if (!(await command({ kind: 'activate', surfaceId: id }))) pendingFocus.current = null;
  }
  return (
    <section
      ref={root}
      tabIndex={-1}
      className={'work-pane ' + (document.activePane === paneId ? 'active-pane' : '')}
      id={paneId + '-pane'}
      aria-label={paneId === 'primary' ? 'Primary pane' : 'Secondary pane'}
      onFocusCapture={claimPane}
      onPointerDownCapture={claimPane}
    >
      <header className="pane-header">
        <div
          className="view-tabs"
          role="tablist"
          aria-label={paneId === 'primary' ? 'Primary views' : 'Secondary views'}
        >
          {pane.tabs.map((id) => (
            <button
              key={id}
              id={'tab-' + id}
              role="tab"
              aria-selected={id === pane.activeSurfaceId}
              aria-controls={'panel-' + paneId}
              tabIndex={id === pane.activeSurfaceId ? 0 : -1}
              title={titleFor(id)}
              onClick={() => void command({ kind: 'activate', surfaceId: id })}
              onKeyDown={(event) => {
                const index = pane.tabs.indexOf(id);
                const next =
                  event.key === 'ArrowRight'
                    ? pane.tabs[(index + 1) % pane.tabs.length]
                    : event.key === 'ArrowLeft'
                      ? pane.tabs[(index + pane.tabs.length - 1) % pane.tabs.length]
                      : event.key === 'Home'
                        ? pane.tabs[0]
                        : event.key === 'End'
                          ? pane.tabs.at(-1)
                          : undefined;
                if (next) {
                  event.preventDefault();
                  void focusTab(next);
                }
              }}
            >
              {document.workspace.surfaces.find((surface) => surface.surfaceId === id)?.pinned && (
                <Icon name="pin" />
              )}
              <span className="truncate">{titleFor(id)}</span>
            </button>
          ))}
          {pane.tabs.length === 0 && (
            <span className="pane-placeholder-title">
              {paneId === 'primary' ? 'Workspace' : 'Second pane'}
            </span>
          )}
        </div>
        <div className="pane-actions toolbar" aria-label="View actions">
          {active && (
            <PaneActions
              key={active.surfaceId}
              surface={active}
              title={title}
              linked={!!link}
              referenceActive={
                !!document.referenceReveal &&
                (document.sharedReferences ?? []).some(
                  (reference) =>
                    reference.referenceId === document.referenceReveal?.referenceId &&
                    reference.evidence.schemaVersion === 3 &&
                    sameResource(reference.evidence.resource, active.resource),
                )
              }
              otherPane={other}
              command={command}
              onRefresh={() => {
                if (link)
                  void command({
                    kind: 'refreshLinked',
                    surfaceId: active.surfaceId,
                    dataRevision: link.dataRevision,
                  }).then((ok) => {
                    if (ok) setRefresh((value) => value + 1);
                  });
                else setRefresh((value) => value + 1);
              }}
            />
          )}
          {layout.kind === 'split' && (
            <button
              className="icon-button"
              aria-label={
                document.maximizedPane === paneId ? 'Restore panes' : 'Maximize ' + paneId + ' pane'
              }
              title={document.maximizedPane === paneId ? 'Restore panes' : 'Maximize pane'}
              onClick={() =>
                void command({
                  kind: 'maximize',
                  pane: document.maximizedPane === paneId ? null : paneId,
                })
              }
            >
              <Icon name="expand" />
            </button>
          )}
          {active && (
            <button
              className="icon-button"
              aria-label={'Close ' + title}
              title="Close view"
              onClick={() => {
                pendingFocus.current = { kind: 'close', surfaceId: active.surfaceId };
                void command({ kind: 'close', surfaceId: active.surfaceId }).then((ok) => {
                  if (!ok) pendingFocus.current = null;
                });
              }}
            >
              <Icon name="close" />
            </button>
          )}
        </div>
      </header>
      <div
        className="pane-content"
        role={active ? 'tabpanel' : undefined}
        id={'panel-' + paneId}
        aria-labelledby={active ? 'tab-' + active.surfaceId : undefined}
      >
        {active ? (
          <ErrorBoundary
            key={
              active.surfaceId +
              ':' +
              (active.view === 'run' || active.view === 'log' ? 0 : refresh)
            }
          >
            {active.view === 'creation-draft' ? (
              <CreationView
                key={active.surfaceId}
                projectId={active.projectId}
                draftId={active.resource.draftId}
                document={document}
                onOpenPipeline={(pipelineId) =>
                  void command({
                    kind: 'open',
                    resource: { kind: 'pipeline', pipelineId },
                    pane: paneId,
                    duplicate: false,
                  })
                }
              />
            ) : (
              <SurfaceView
                flowNavigation={flowNavigation}
                savedReports={document.savedReports ?? []}
                surfaces={document.workspace.surfaces}
                key={
                  active.surfaceId +
                  ':' +
                  (active.view === 'run' || active.view === 'log' ? 0 : refresh)
                }
                refreshVersion={refresh}
                surface={active}
                link={link}
                showLinkedActions={
                  !!link &&
                  visibleSurfaceIds.find((id) => link.surfaceIds.includes(id)) === active.surfaceId
                }
                title={title}
                rendererSessionId={rendererSessionId}
                evidence={
                  document.selections.find((item) => item.surfaceId === active.surfaceId)
                    ?.evidence ?? null
                }
                command={command}
                references={document.sharedReferences ?? []}
                reveal={document.referenceReveal}
                onOpenLogs={(target) => {
                  const resource = logResource(target);
                  const existing = document.workspace.surfaces.find((item) =>
                    sameResource(item.resource, resource),
                  );
                  // Open logs is an explicit companion-pane intent, including an existing background tab.
                  void command(
                    existing
                      ? { kind: 'move', surfaceId: existing.surfaceId, pane: other }
                      : { kind: 'open', resource, pane: other, duplicate: false },
                  );
                }}
              />
            )}
          </ErrorBoundary>
        ) : (
          <EmptyPane
            views={document.workspace.surfaces
              .filter((surface) => !pane.tabs.includes(surface.surfaceId))
              .map((surface) => ({ id: surface.surfaceId, title: titleFor(surface.surfaceId) }))}
            onBrowse={() => onBrowse(paneId)}
            onMove={(surfaceId) => void command({ kind: 'move', surfaceId, pane: paneId })}
          />
        )}
      </div>
    </section>
  );
}
