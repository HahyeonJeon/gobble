import { parse, SurfaceSchema } from '@gobble/contracts';
import { defaultDependencyNavigation } from '@gobble/contracts';
import {
  parseWorkspaceDocument,
  isTabularAction,
  viewLinkFor,
  resourceKey,
  type PaneId,
  type Surface,
  type WorkspaceAction,
  type WorkspaceDocument,
} from '@gobble/contracts';
import { AppProblem } from '../problem';
import { invalidateAttachments } from '../evidence/draft';

export function emptyWorkspace(projectId: string): WorkspaceDocument {
  return {
    schemaVersion: 21,
    viewLinks: [],
    paneOrientation: 'vertical',
    workspace: {
      schemaVersion: 1,
      projectId,
      revision: 0,
      agents: [],
      surfaces: [],
      layout: { kind: 'single', primary: { tabs: [], activeSurfaceId: null } },
      decisions: [],
    },
    titles: [],
    activePane: 'primary',
    maximizedPane: null,
    chat: { collapsed: false, width: 420, draft: '', recipientAgentId: null },
    selections: [],
    activity: [],
    receipts: [],
  };
}

/** Finds the containing pane; an absent surface is a not_found error, never a default pane. */
export function paneOf(doc: WorkspaceDocument, surfaceId: string): PaneId {
  if (doc.workspace.layout.primary.tabs.includes(surfaceId)) return 'primary';
  if (
    doc.workspace.layout.kind === 'split' &&
    doc.workspace.layout.secondary.tabs.includes(surfaceId)
  )
    return 'secondary';
  throw new AppProblem('not_found', 'This view is not placed in any pane.');
}
export function getPane(doc: WorkspaceDocument, id: PaneId) {
  if (id === 'primary') return doc.workspace.layout.primary;
  if (doc.workspace.layout.kind !== 'split')
    throw new AppProblem('invalid_request', 'The second pane is not open.');
  return doc.workspace.layout.secondary;
}
export function ensureSplit(doc: WorkspaceDocument): void {
  if (doc.workspace.layout.kind === 'single') {
    doc.workspace.layout = {
      kind: 'split',
      primary: doc.workspace.layout.primary,
      secondary: { tabs: [], activeSurfaceId: null },
      primaryFraction: 0.5,
    };
  }
}
function removeTab(doc: WorkspaceDocument, surfaceId: string): void {
  const pane = getPane(doc, paneOf(doc, surfaceId));
  const index = pane.tabs.indexOf(surfaceId);
  pane.tabs = pane.tabs.filter((id) => id !== surfaceId);
  if (pane.activeSurfaceId === surfaceId)
    pane.activeSurfaceId = pane.tabs[Math.min(index, pane.tabs.length - 1)] ?? null;
}
export function focusSurface(doc: WorkspaceDocument, surfaceId: string): void {
  const id = paneOf(doc, surfaceId);
  getPane(doc, id).activeSurfaceId = surfaceId;
  doc.activePane = id;
  if (doc.maximizedPane !== null) doc.maximizedPane = id;
}

export type OpenedResource = {
  surfaceId: string;
  title: string;
  view: Exclude<Surface['view'], 'scatter'>;
};

/** User pipeline navigation reuses a view in the requested Pane. Other resource
 * navigation retains its existing workspace-wide reuse policy. */
export function reusableSurface(
  doc: WorkspaceDocument,
  action: Extract<WorkspaceAction, { kind: 'open' }>,
): Surface | undefined {
  if (action.duplicate) return undefined;
  return doc.workspace.surfaces.find(
    (surface) =>
      surface.view !== 'scatter' &&
      (surface.view !== 'pipeline' || paneOf(doc, surface.surfaceId) === action.pane) &&
      resourceKey(surface.resource) === resourceKey(action.resource),
  );
}

// Pure user-action transitions. Provider policy is a separate future entry point, not renderer input.
export function transition(
  current: WorkspaceDocument,
  action: WorkspaceAction,
  opened?: OpenedResource | { surfaceId: string },
): WorkspaceDocument {
  if (isTabularAction(action))
    throw new AppProblem(
      'invalid_request',
      'Tabular actions require host source and render validation.',
    );
  const doc = structuredClone(current);
  if (
    'surfaceId' in action &&
    !doc.workspace.surfaces.some((surface) => surface.surfaceId === action.surfaceId)
  )
    throw new AppProblem('not_found', 'This view is no longer open.');
  switch (action.kind) {
    case 'runMode':
    case 'dependencyCamera':
    case 'dependencyNavigation': {
      const surface = doc.workspace.surfaces.find((item) => item.surfaceId === action.surfaceId)!;
      if (surface.view !== 'run')
        throw new AppProblem('invalid_request', 'Dependency navigation requires a Run view.');
      const state = surface.runView ?? { query: '', status: null, viewRevision: 0 };
      if (action.kind === 'runMode')
        surface.runView = {
          ...state,
          mode: action.mode,
          ...(action.mode === 'dependencies'
            ? { dependencies: state.dependencies ?? defaultDependencyNavigation() }
            : {}),
          viewRevision: state.viewRevision + 1,
        };
      else {
        const previous = state.dependencies ?? defaultDependencyNavigation();
        if (action.kind === 'dependencyCamera') {
          surface.runView = {
            ...state,
            dependencies: { ...previous, camera: structuredClone(action.camera) },
          };
          break;
        }
        const changed =
          previous.query !== action.navigation.query ||
          previous.representation !== action.navigation.representation;
        surface.runView = {
          ...state,
          dependencies: { ...previous, ...action.navigation },
          viewRevision: state.viewRevision + (changed ? 1 : 0),
        };
      }
      break;
    }
    case 'reportNavigate': {
      const surface = doc.workspace.surfaces.find((s) => s.surfaceId === action.surfaceId)!;
      if (surface.view !== 'report')
        throw new AppProblem('invalid_request', 'Report navigation requires a Report View.');
      surface.reportModuleId = action.moduleId;
      break;
    }
    case 'notebookNavigate': {
      const surface = doc.workspace.surfaces.find((item) => item.surfaceId === action.surfaceId);
      if (!surface || surface.view !== 'notebook')
        throw new AppProblem('invalid_request', 'A Notebook view is required.');
      surface.notebook = structuredClone(action.navigation);
      doc.selections = doc.selections.filter((item) => item.surfaceId !== action.surfaceId);
      break;
    }
    case 'pdfNavigate': {
      const surface = doc.workspace.surfaces.find((item) => item.surfaceId === action.surfaceId);
      if (!surface || surface.view !== 'pdf')
        throw new AppProblem('invalid_request', 'A PDF view is required.');
      surface.pdf = structuredClone(action.navigation);
      doc.selections = doc.selections.filter((item) => item.surfaceId !== action.surfaceId);
      break;
    }
    case 'runFilter': {
      const surface = doc.workspace.surfaces.find((item) => item.surfaceId === action.surfaceId)!;
      if (surface.view !== 'run')
        throw new AppProblem('invalid_request', 'Task filters require a Run view.');
      surface.runView = {
        ...surface.runView,
        ...action.filter,
        viewRevision: (surface.runView?.viewRevision ?? 0) + 1,
      };
      break;
    }
    case 'logStream': {
      const surface = doc.workspace.surfaces.find((item) => item.surfaceId === action.surfaceId)!;
      if (surface.view !== 'log')
        throw new AppProblem('invalid_request', 'Stream choice requires a log view.');
      surface.logView = {
        stream: action.stream,
        viewRevision: (surface.logView?.viewRevision ?? 0) + 1,
      };
      break;
    }
    case 'reply':
    case 'cancelReply':
    case 'dismissQuestion':
    case 'attach':
    case 'detach':
    case 'openReport':
    case 'share':
    case 'retract':
    case 'returnReferenceView':
    case 'reveal':
    case 'currentTask':
      throw new AppProblem('invalid_request', 'Shared references require host validation.');
    case 'open': {
      const found = reusableSurface(doc, action);
      if (found) {
        focusSurface(doc, found.surfaceId);
        break;
      }
      if (!opened || !('view' in opened))
        throw new AppProblem('invalid_request', 'The resource has not been validated.');
      if (action.pane === 'secondary') ensureSplit(doc);
      const pane = getPane(doc, action.pane);
      if (pane.tabs.length >= 32)
        throw new AppProblem('unsupported', 'A pane can hold up to 32 views. Close a view first.');
      doc.workspace.surfaces.push(
        parse(SurfaceSchema, {
          projectId: doc.workspace.projectId,
          surfaceId: opened.surfaceId,
          resource: action.resource,
          view: opened.view,
          pinned: false,
          openedBy: { kind: 'user' },
          ...(opened.view === 'run'
            ? { runView: { query: '', status: null, viewRevision: 0 } }
            : {}),
          ...(opened.view === 'log'
            ? { logView: { stream: 'auto' as const, viewRevision: 0 } }
            : {}),
        }),
      );
      doc.titles.push({ surfaceId: opened.surfaceId, title: opened.title });
      pane.tabs.push(opened.surfaceId);
      focusSurface(doc, opened.surfaceId);
      break;
    }
    case 'duplicateView': {
      if (!opened) throw new AppProblem('invalid_request', 'The duplicate has no host identity.');
      if (action.pane === 'secondary') ensureSplit(doc);
      const pane = getPane(doc, action.pane);
      if (pane.tabs.length >= 32 || doc.workspace.surfaces.length >= 64)
        throw new AppProblem('unsupported', 'Close a view before opening another.');
      const source = doc.workspace.surfaces.find((item) => item.surfaceId === action.surfaceId)!;
      const copy = {
        ...structuredClone(source),
        surfaceId: opened.surfaceId,
        pinned: false,
        openedBy: { kind: 'user' as const },
      };
      delete copy.userClaimed;
      doc.workspace.surfaces.push(copy);
      doc.titles.push({
        surfaceId: copy.surfaceId,
        title: doc.titles.find((item) => item.surfaceId === source.surfaceId)!.title,
      });
      pane.tabs.push(copy.surfaceId);
      focusSurface(doc, copy.surfaceId);
      break;
    }
    case 'activate':
      focusSurface(doc, action.surfaceId);
      break;
    case 'focusPane':
      getPane(doc, action.pane);
      doc.activePane = action.pane;
      if (doc.maximizedPane !== null) doc.maximizedPane = action.pane;
      break;
    case 'close': {
      removeTab(doc, action.surfaceId);
      doc.workspace.surfaces = doc.workspace.surfaces.filter(
        (surface) => surface.surfaceId !== action.surfaceId,
      );
      doc.viewLinks = doc.viewLinks
        .map((link) => {
          const next = {
            ...link,
            surfaceIds: link.surfaceIds.filter((id) => id !== action.surfaceId),
          };
          if (next.selectionSourceId === action.surfaceId) delete next.selectionSourceId;
          return next;
        })
        .filter((link) => link.surfaceIds.length);
      doc.titles = doc.titles.filter((item) => item.surfaceId !== action.surfaceId);
      doc.selections = doc.selections.filter((item) => item.surfaceId !== action.surfaceId);
      if (getPane(doc, doc.activePane).tabs.length === 0 && doc.workspace.layout.kind === 'split') {
        const other = doc.activePane === 'primary' ? 'secondary' : 'primary';
        if (getPane(doc, other).tabs.length) {
          doc.activePane = other;
          if (doc.maximizedPane !== null) doc.maximizedPane = other;
        }
      }
      break;
    }
    case 'pin': {
      const surface = doc.workspace.surfaces.find((item) => item.surfaceId === action.surfaceId);
      if (surface) surface.pinned = action.pinned;
      break;
    }
    case 'move': {
      if (action.pane === 'secondary') ensureSplit(doc);
      const target = getPane(doc, action.pane);
      if (!target.tabs.includes(action.surfaceId)) {
        if (target.tabs.length >= 32)
          throw new AppProblem('unsupported', 'The destination pane is full.');
        removeTab(doc, action.surfaceId);
        target.tabs.push(action.surfaceId);
      }
      focusSurface(doc, action.surfaceId);
      break;
    }
    case 'arrange':
      if (action.layout === 'split') ensureSplit(doc);
      else if (doc.workspace.layout.kind === 'split') {
        const { primary, secondary } = doc.workspace.layout;
        if (primary.tabs.length + secondary.tabs.length > 32)
          throw new AppProblem(
            'unsupported',
            'Close some views before combining these panes. A single pane holds 32 views.',
          );
        doc.workspace.layout = {
          kind: 'single',
          primary: {
            tabs: [...primary.tabs, ...secondary.tabs],
            activeSurfaceId:
              getPane(doc, doc.activePane).activeSurfaceId ??
              primary.activeSurfaceId ??
              secondary.activeSurfaceId,
          },
        };
        doc.activePane = 'primary';
        doc.maximizedPane = null;
      }
      break;
    case 'resize':
      if (doc.workspace.layout.kind === 'split')
        doc.workspace.layout.primaryFraction = action.primaryFraction;
      break;
    case 'maximize':
      if (action.pane !== null) getPane(doc, action.pane);
      doc.maximizedPane = doc.workspace.layout.kind === 'split' ? action.pane : null;
      if (action.pane !== null) doc.activePane = action.pane;
      break;
    case 'chat':
      doc.chat.collapsed = action.collapsed;
      break;
    case 'resizeChat':
      doc.chat.width = action.width;
      break;
    case 'recipient':
      if (doc.chat.replyToQuestionId && doc.chat.recipientAgentId !== action.agentId)
        throw new AppProblem(
          'invalid_request',
          'Cancel the reply before choosing another recipient.',
        );
      if (doc.chat.recipientAgentId !== action.agentId) invalidateAttachments(doc);
      doc.chat.recipientAgentId = action.agentId;
      break;
    case 'select': {
      const link = viewLinkFor(doc, action.surfaceId);
      if (link) {
        if (
          action.evidence &&
          (action.evidence.dataRevision !== link.dataRevision ||
            action.evidence.selection?.kind !== 'table')
        )
          throw new AppProblem(
            'stale_revision',
            'Selection does not address the linked source revision.',
          );
        link.rowKeys =
          action.evidence?.selection?.kind === 'table'
            ? [...action.evidence.selection.rowKeys]
            : [];
        link.selectionSourceId = action.surfaceId;
        break;
      }
      doc.selections = doc.selections.filter((item) => item.surfaceId !== action.surfaceId);
      if (action.evidence)
        doc.selections.push({ surfaceId: action.surfaceId, evidence: action.evidence });
      break;
    }
  }
  const claimed =
    'surfaceId' in action && action.kind !== 'close'
      ? action.surfaceId
      : action.kind === 'focusPane'
        ? getPane(doc, action.pane).activeSurfaceId
        : action.kind === 'open'
          ? getPane(doc, doc.activePane).activeSurfaceId
          : null;
  const surface = doc.workspace.surfaces.find((item) => item.surfaceId === claimed);
  if (surface?.openedBy.kind === 'agent') surface.userClaimed = true;
  return parseWorkspaceDocument(doc);
}
