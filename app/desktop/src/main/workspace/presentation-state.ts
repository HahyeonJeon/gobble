import type { PaneId, WorkspaceDocument } from '@gobble/contracts';
import { RenderSession } from './render-session';
import { ReferenceViews } from './reference-views';
import { ObservedReferenceViews } from './observed-reference-views';
import { TabularSnapshots } from './tabular-snapshots';
import type { WorkspaceResources } from './service';

/**
 * Transient view lifetime owned by one workspace controller. The controller
 * authorizes and serializes calls; this object never writes durable documents.
 * Specialized helpers retain their own reference/acknowledgment invariants.
 */
export class PresentationState {
  readonly rendering = new RenderSession();
  readonly referenceViews = new ReferenceViews();
  readonly observedReferences = new ObservedReferenceViews();
  readonly tabularSnapshots = new TabularSnapshots();

  constructor(private readonly resources: Pick<WorkspaceResources, 'retainViews' | 'clearViews'>) {}

  reconcile(doc: WorkspaceDocument, visiblePanes: readonly PaneId[]): void {
    const visible = new Set(visibleSurfaces(doc, visiblePanes));
    this.rendering.reveal([...visible]);
    this.referenceViews.reconcile(doc);
    this.observedReferences.reconcile(doc);
    this.rendering.reconcile(
      doc.workspace.surfaces,
      doc.viewLinks,
      (id) => this.referenceViews.forSurface(id),
      (id) => this.observedReferences.forSurface(id),
    );
    this.resources.retainViews?.(
      doc.workspace.projectId,
      doc.workspace.surfaces
        .filter((s) => (s.view === 'pdf' || s.view === 'notebook') && visible.has(s.surfaceId))
        .map((s) => s.surfaceId),
    );
    this.tabularSnapshots.retain(
      doc.workspace.projectId,
      doc.viewLinks.filter((link) => link.surfaceIds.some((id) => visible.has(id))),
    );
  }

  clear(): void {
    this.rendering.clear();
    this.resources.clearViews?.();
    this.clearRetainedReferences();
  }

  disconnect(): void {
    this.resources.clearViews?.();
    this.rendering.disconnect();
    this.clearRetainedReferences();
  }

  private clearRetainedReferences(): void {
    this.tabularSnapshots.clear();
    this.referenceViews.clear();
    this.observedReferences.clear();
  }
}

function visibleSurfaces(
  doc: WorkspaceDocument,
  visiblePanes: readonly PaneId[],
): (string | null)[] {
  const layout = doc.workspace.layout;
  return visiblePanes.flatMap((pane) => {
    if (doc.maximizedPane !== null && pane !== doc.maximizedPane) return [];
    if (pane === 'secondary')
      return layout.kind === 'split' ? [layout.secondary.activeSurfaceId] : [];
    return [layout.primary.activeSurfaceId];
  });
}
