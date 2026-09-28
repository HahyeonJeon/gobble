import { parse, SurfaceSchema } from '@gobble/contracts';
import {
  sameResource,
  type AgentAttachment,
  type PaneId,
  type ResourceRef,
  type Surface,
  type WorkspaceDocument,
} from '@gobble/contracts';
import { AppProblem } from '../problem';
import { getPane, paneOf, transition } from '../workspace/model';

export type PreparedView = {
  resource: ResourceRef;
  surfaceId: string;
  title: string;
  reuseSurfaceId?: string;
} & (
  | { view: Exclude<Surface['view'], 'scatter'> }
  | { view: 'scatter'; scatter: Extract<Surface, { view: 'scatter' }>['scatter'] }
);
export function protectedView(surface: Surface, agentId: string): boolean {
  return (
    surface.pinned ||
    !!surface.userClaimed ||
    surface.openedBy.kind === 'user' ||
    surface.openedBy.agentId !== agentId
  );
}
function split(doc: WorkspaceDocument): void {
  if (doc.workspace.layout.kind === 'single')
    doc.workspace.layout = {
      kind: 'split',
      primary: doc.workspace.layout.primary,
      secondary: { tabs: [], activeSurfaceId: null },
      primaryFraction: 0.5,
    };
}
function occupied(doc: WorkspaceDocument, paneId: PaneId, agentId: string): boolean {
  if (paneId === 'secondary' && doc.workspace.layout.kind === 'single') return false;
  const id = getPane(doc, paneId).activeSurfaceId;
  const surface = doc.workspace.surfaces.find((item) => item.surfaceId === id);
  return !!surface && protectedView(surface, agentId);
}
function outcome(doc: WorkspaceDocument, surfaceId: string) {
  const pane = paneOf(doc, surfaceId);
  return {
    surfaceId,
    pane,
    visibility:
      getPane(doc, pane).activeSurfaceId === surfaceId &&
      (doc.maximizedPane === null || doc.maximizedPane === pane)
        ? 'revealed'
        : 'background',
  };
}
/** Applies to a cloned document; the host commits all requested opens together or none. */
export function openViews(
  doc: WorkspaceDocument,
  prepared: PreparedView[],
  agent: AgentAttachment,
  preferred: PaneId,
  dismissed: (resource: ResourceRef) => boolean,
) {
  return prepared.map((item, index) => {
    if (dismissed(item.resource))
      throw new AppProblem(
        'forbidden',
        'The user closed this resource during your current message. Do not reopen it.',
      );
    const existing = doc.workspace.surfaces.find(
      (surface) =>
        surface.view === item.view &&
        sameResource(surface.resource, item.resource) &&
        (item.view !== 'scatter' || surface.surfaceId === item.reuseSurfaceId),
    );
    if (existing) {
      const pane = paneOf(doc, existing.surfaceId);
      if (!occupied(doc, pane, agent.agentId))
        getPane(doc, pane).activeSurfaceId = existing.surfaceId;
      return { ...outcome(doc, existing.surfaceId), reused: true };
    }
    let target = index === 0 ? preferred : preferred === 'primary' ? 'secondary' : 'primary';
    if (occupied(doc, target, agent.agentId)) {
      const other = target === 'primary' ? 'secondary' : 'primary';
      if (!occupied(doc, other, agent.agentId)) target = other;
    }
    if (target === 'secondary') split(doc);
    const pane = getPane(doc, target);
    if (pane.tabs.length >= 32 || doc.workspace.surfaces.length >= 64)
      throw new AppProblem('unsupported', 'The workspace has no room for another view.');
    const protect = occupied(doc, target, agent.agentId);
    doc.workspace.surfaces.push(
      parse(SurfaceSchema, {
        projectId: doc.workspace.projectId,
        surfaceId: item.surfaceId,
        resource: item.resource,
        ...(item.view === 'scatter'
          ? { view: 'scatter' as const, scatter: item.scatter }
          : { view: item.view }),
        ...(item.view === 'run' ? { runView: { query: '', status: null, viewRevision: 0 } } : {}),
        ...(item.view === 'log' ? { logView: { stream: 'auto' as const, viewRevision: 0 } } : {}),
        pinned: false,
        openedBy: { kind: 'agent', agentId: agent.agentId },
      }),
    );
    doc.titles.push({ surfaceId: item.surfaceId, title: item.title });
    pane.tabs.push(item.surfaceId);
    if (!protect) pane.activeSurfaceId = item.surfaceId;
    return { ...outcome(doc, item.surfaceId), reused: false };
  });
}
export function releaseView(doc: WorkspaceDocument, surfaceId: string, agentId: string): void {
  const surface = doc.workspace.surfaces.find((item) => item.surfaceId === surfaceId);
  if (!surface) throw new AppProblem('not_found', 'This view is no longer open.');
  if (protectedView(surface, agentId))
    throw new AppProblem(
      'forbidden',
      'This view belongs to the user or another Agent, or has been pinned or claimed.',
    );
  Object.assign(doc, transition(doc, { kind: 'close', surfaceId }));
}
export function arrangeView(
  doc: WorkspaceDocument,
  surfaceId: string,
  target: PaneId,
  agentId: string,
) {
  const surface = doc.workspace.surfaces.find((item) => item.surfaceId === surfaceId);
  if (!surface) throw new AppProblem('not_found', 'This view is no longer open.');
  if (protectedView(surface, agentId) || occupied(doc, target, agentId))
    throw new AppProblem('forbidden', 'The requested placement would replace protected user work.');
  const activePane = doc.activePane;
  const maximizedPane = doc.maximizedPane;
  const next = transition(doc, { kind: 'move', surfaceId, pane: target });
  next.workspace.surfaces.find((item) => item.surfaceId === surfaceId)!.userClaimed = false;
  next.activePane = activePane;
  next.maximizedPane = maximizedPane;
  Object.assign(doc, next);
  return outcome(doc, surfaceId);
}
