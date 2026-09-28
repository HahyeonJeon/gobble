import {
  sameResource,
  type ObservedReferenceView,
  type SharedReference,
  type SurfaceData,
  type WorkspaceAction,
  type WorkspaceDocument,
} from '@gobble/contracts';
import { AppProblem } from '../problem';
import { checkSelection } from './selection';
import { focusSurface, transition } from './model';

type ReturnState = {
  activePane: WorkspaceDocument['activePane'];
  maximizedPane: WorkspaceDocument['maximizedPane'];
  primary: string | null;
  secondary: string | null;
};
export type ObservedReferenceSession = {
  projectId: string;
  surfaceId: string;
  state: ObservedReferenceView;
  data: SurfaceData;
  returnTo: ReturnState;
};

export function prepareObservedReference(
  doc: WorkspaceDocument,
  reference: SharedReference,
  data: SurfaceData,
  requestId: string,
): ObservedReferenceSession {
  const selection = reference.evidence.selection;
  if (
    reference.evidence.schemaVersion < 3 ||
    !selection ||
    ![
      'run-task',
      'log-text',
      'run-group',
      'run-dependency',
      'pdf',
      'notebook',
      'pipeline',
    ].includes(selection.kind) ||
    (data.kind === 'file' && !['pdf', 'notebook', 'pipeline'].includes(data.value.content.kind))
  )
    throw new AppProblem('unsupported', 'This pointer has no supported observed target.');
  try {
    checkSelection(reference.evidence, data);
  } catch {
    throw new AppProblem(
      'stale_revision',
      'This pointer targets an older observation. No highlight was applied. Open matching captured evidence when available; a pointer does not store source bytes.',
    );
  }
  if (Buffer.byteLength(JSON.stringify(data)) > (data.kind === 'file' ? 8 : 1) * 1024 * 1024)
    throw new AppProblem('unsupported', 'This reference exceeds the view memory limit.');
  const returnTo = {
    activePane: doc.activePane,
    maximizedPane: doc.maximizedPane,
    primary: doc.workspace.layout.primary.activeSurfaceId,
    secondary:
      doc.workspace.layout.kind === 'split' ? doc.workspace.layout.secondary.activeSurfaceId : null,
  };
  const origin = 'origin' in reference.evidence ? reference.evidence.origin?.surfaceId : undefined;
  let surface =
    doc.workspace.surfaces.find(
      (item) =>
        item.surfaceId === origin && sameResource(item.resource, reference.evidence.resource),
    ) ??
    doc.workspace.surfaces.find((item) => sameResource(item.resource, reference.evidence.resource));
  if (!surface) {
    const id = 'srf_ref_' + requestId.slice(4);
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
        {
          surfaceId: id,
          view:
            data.kind === 'file'
              ? data.value.content.kind === 'notebook'
                ? 'notebook'
                : 'pdf'
              : data.kind,
          title: reference.label,
        },
      ),
    );
    surface = doc.workspace.surfaces.find((item) => item.surfaceId === id)!;
  }
  focusSurface(doc, surface.surfaceId);
  return {
    projectId: doc.workspace.projectId,
    surfaceId: surface.surfaceId,
    state: { requestId, referenceId: reference.referenceId, evidence: reference.evidence },
    data,
    returnTo,
  };
}

/** At most one reference observation (1 MiB semantic or 8 MiB PDF raster). Durable settings and capture storage remain separate owners. */
export class ObservedReferenceViews {
  private session: ObservedReferenceSession | undefined;
  set(session: ObservedReferenceSession) {
    const returnTo =
      this.session?.projectId === session.projectId ? this.session.returnTo : session.returnTo;
    this.session = structuredClone({ ...session, returnTo });
  }
  clear() {
    this.session = undefined;
  }
  forSurface(id: string) {
    return this.session?.surfaceId === id ? this.session.state : undefined;
  }
  dataFor(id: string) {
    return this.session?.surfaceId === id ? structuredClone(this.session.data) : undefined;
  }
  has(requestId: string) {
    return this.session?.state.requestId === requestId;
  }
  returnTo(doc: WorkspaceDocument, requestId: string) {
    if (!this.session || !this.has(requestId))
      throw new AppProblem('stale_revision', 'This reference view is no longer active.');
    const prior = this.session.returnTo,
      layout = doc.workspace.layout;
    if (prior.primary && layout.primary.tabs.includes(prior.primary))
      layout.primary.activeSurfaceId = prior.primary;
    if (
      layout.kind === 'split' &&
      prior.secondary &&
      layout.secondary.tabs.includes(prior.secondary)
    )
      layout.secondary.activeSurfaceId = prior.secondary;
    doc.activePane =
      prior.activePane === 'secondary' && layout.kind !== 'split' ? 'primary' : prior.activePane;
    doc.maximizedPane =
      prior.maximizedPane === 'secondary' && layout.kind !== 'split' ? null : prior.maximizedPane;
  }
  reconcile(doc: WorkspaceDocument) {
    if (
      this.session &&
      (doc.workspace.projectId !== this.session.projectId ||
        !doc.workspace.surfaces.some((surface) => surface.surfaceId === this.session!.surfaceId))
    )
      this.clear();
  }
  assertBaseAction(action: WorkspaceAction) {
    if (
      'surfaceId' in action &&
      this.forSurface(action.surfaceId) &&
      [
        'pdfNavigate',
        'notebookNavigate',
        'runFilter',
        'logStream',
        'runMode',
        'dependencyNavigation',
        'dependencyCamera',
        'select',
        'attach',
        'share',
      ].includes(action.kind)
    )
      throw new AppProblem(
        'invalid_request',
        'Return to your view before changing or discussing your selection.',
      );
  }
}
