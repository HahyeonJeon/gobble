import {
  revealPlotTarget,
  viewLinkFor,
  sameResource,
  validateReferencePresentation,
  type ReferenceView,
  type SharedReference,
  type SurfaceData,
  type WorkspaceAction,
  type WorkspaceDocument,
} from '@gobble/contracts';
import { AppProblem } from '../problem';
import { checkSelection } from './selection';
import { focusSurface, transition } from './model';

export type ReferenceSession = {
  projectId: string;
  dataRevision: string;
  surfaceIds: string[];
  state: ReferenceView;
};

/** Prepare only an explicit User reveal. It preserves base settings and exact row membership. */
export function prepareReferenceView(
  doc: WorkspaceDocument,
  reference: SharedReference,
  data: SurfaceData,
  requestId: string,
): ReferenceSession {
  const p = reference.presentation;
  if (!p || data.kind !== 'file' || data.value.content.kind !== 'table')
    throw new AppProblem('unsupported', 'This reference has no supported plot context.');
  if (data.value.revision !== reference.evidence.dataRevision)
    throw new AppProblem(
      'stale_revision',
      'This reference is from an older source version. Its rows have not been highlighted on the changed source. Saved message evidence remains available.',
    );
  checkSelection(reference.evidence, data);
  validateReferencePresentation(reference.evidence, p);
  // Legacy chart references retain their captured context but reveal exact rows in a table.
  let table = doc.workspace.surfaces.find(
    (surface) =>
      surface.view === 'table' && sameResource(surface.resource, reference.evidence.resource),
  );
  if (!table) {
    const id = 'srf_ref_table_' + requestId.slice(4);
    Object.assign(
      doc,
      transition(
        doc,
        {
          kind: 'open',
          resource: reference.evidence.resource,
          pane: doc.activePane,
          duplicate: false,
        },
        { surfaceId: id, view: 'table', title: data.value.name },
      ),
    );
    table = doc.workspace.surfaces.find((surface) => surface.surfaceId === id)!;
  }
  const link = viewLinkFor(doc, table.surfaceId);
  if (link && link.dataRevision !== data.value.revision)
    throw new AppProblem(
      'stale_revision',
      'Refresh the linked source before showing this reference. Your view settings have been retained.',
    );
  focusSurface(doc, table.surfaceId);
  const rows =
    reference.evidence.selection?.kind === 'table' ? reference.evidence.selection.rowKeys : [];
  return {
    projectId: doc.workspace.projectId,
    dataRevision: data.value.revision,
    surfaceIds: [table.surfaceId],
    state: {
      referenceId: reference.referenceId,
      requestId,
      ...revealPlotTarget(data.value.content, p, rows),
    },
  };
}

/** Main-process, session-only state. Never persisted or writable by a renderer/provider payload. */
export class ReferenceViews {
  private session: ReferenceSession | undefined;
  set(session: ReferenceSession): void {
    this.session = structuredClone(session);
  }
  clear(): void {
    this.session = undefined;
  }
  forSurface(surfaceId: string): ReferenceView | undefined {
    return this.session?.surfaceIds.includes(surfaceId) ? this.session.state : undefined;
  }
  requireReturn(requestId: string): void {
    if (!this.session || this.session.state.requestId !== requestId)
      throw new AppProblem('stale_revision', 'This reference view is no longer active.');
  }
  reconcile(doc: WorkspaceDocument): void {
    if (!this.session) return;
    if (doc.workspace.projectId !== this.session.projectId) {
      this.clear();
      return;
    }
    this.session.surfaceIds = this.session.surfaceIds.filter((id) =>
      doc.workspace.surfaces.some((surface) => surface.surfaceId === id),
    );
    if (
      !this.session.surfaceIds.length ||
      this.session.surfaceIds.some((id) => {
        const link = viewLinkFor(doc, id);
        return link && link.dataRevision !== this.session!.dataRevision;
      })
    )
      this.clear();
  }
  assertBaseAction(action: WorkspaceAction): void {
    if (
      'surfaceId' in action &&
      this.forSurface(action.surfaceId) &&
      [
        'tableSettings',
        'scatterSettings',
        'linkedFilter',
        'scatterViewport',
        'refreshLinked',
        'duplicateView',
        'openScatter',
        'openLinkedTable',
      ].includes(action.kind)
    )
      throw new AppProblem(
        'invalid_request',
        'Return to your view before changing its settings. Your row selection can still be edited.',
      );
  }
}
