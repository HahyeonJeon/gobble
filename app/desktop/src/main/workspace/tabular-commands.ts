import {
  filterTable,
  linkedSelectionColumns,
  sameResource,
  validateTableSettings,
  viewLinkFor,
  type Surface,
  type SurfaceData,
  type TableContent,
  type TabularAction,
  type ViewLink,
  type WorkspaceDocument,
} from '@gobble/contracts';
import { AppProblem } from '../problem';
import { attachEvidence } from '../evidence/draft';
import { checkSelection } from './selection';

function tableData(doc: WorkspaceDocument, surface: Surface, data: SurfaceData): TableContent {
  if (
    data.kind !== 'file' ||
    data.value.projectId !== doc.workspace.projectId ||
    !sameResource(surface.resource, { kind: 'file', resourceId: data.value.resourceId }) ||
    data.value.content.kind !== 'table'
  )
    throw new AppProblem('unsupported', 'This view requires a CSV table from the current Project.');
  return data.value.content;
}
function requireLink(doc: WorkspaceDocument, surfaceId: string): ViewLink {
  const link = viewLinkFor(doc, surfaceId);
  if (!link) throw new AppProblem('invalid_request', 'Open a linked view first.');
  return link;
}
/** Mutates only the controller's private draft; no I/O, persistence or render-state ownership. */
export function applyTabularAction(
  doc: WorkspaceDocument,
  action: TabularAction,
  data: SurfaceData,
  requestId: string,
): void {
  const surface = doc.workspace.surfaces.find((item) => item.surfaceId === action.surfaceId);
  if (!surface) throw new AppProblem('not_found', 'This view is no longer open.');
  const content = tableData(doc, surface, data);
  if (data.kind !== 'file') return; // narrowed by tableData; retained for the discriminated type.
  const sourceRevision = data.value.revision;
  const linked = viewLinkFor(doc, surface.surfaceId);
  if (action.kind !== 'refreshLinked' && linked && linked.dataRevision !== sourceRevision)
    throw new AppProblem('stale_revision', 'Refresh the linked source before changing its views.');
  try {
    switch (action.kind) {
      case 'openScatter':
      case 'openLinkedTable':
      case 'scatterSettings':
      case 'scatterViewport':
        throw new AppProblem(
          'unsupported',
          'CSV chart creation has been removed. Open the source table.',
        );
      case 'tableSettings': {
        if (surface.view !== 'table')
          throw new AppProblem('invalid_request', 'Table settings require a table view.');
        validateTableSettings(content, action.settings);
        const columnsChanged =
          JSON.stringify(surface.table?.columns ?? content.columns.map((column) => column.id)) !==
          JSON.stringify(action.settings.columns ?? content.columns.map((column) => column.id));
        surface.table = {
          ...action.settings,
          viewRevision: (surface.table?.viewRevision ?? 0) + 1,
        };
        if (linked && columnsChanged) linked.selectionSourceId = surface.surfaceId;
        const local = doc.selections.find((item) => item.surfaceId === surface.surfaceId);
        if (local?.evidence.selection?.kind === 'table')
          local.evidence.selection.columns =
            action.settings.columns ?? content.columns.map((column) => column.id);
        break;
      }
      case 'linkedFilter': {
        const link = requireLink(doc, surface.surfaceId);
        filterTable(content, action.filter);
        link.filter = action.filter;
        link.filterRevision++;
        break;
      }
      case 'discussSelection': {
        const link = requireLink(doc, surface.surfaceId);
        if (!link.rowKeys.length)
          throw new AppProblem('invalid_request', 'Select rows to discuss first.');
        const evidence = {
          schemaVersion: 2 as const,
          projectId: doc.workspace.projectId,
          resource: link.resource,
          dataRevision: link.dataRevision,
          origin: { surfaceId: surface.surfaceId },
          selection: {
            kind: 'table' as const,
            coordinateSpace: 'revision-row-column-keys' as const,
            rowKeys: [...link.rowKeys],
            columns: linkedSelectionColumns(doc.workspace.surfaces, link, content),
          },
        };
        checkSelection(evidence, data);
        attachEvidence(doc, {
          attachmentId: 'att_' + requestId.slice(4),
          evidence,
          label: data.value.name + ' · selected rows',
          createdAt: Date.now(),
        });
        doc.chat.collapsed = false;
        break;
      }
      case 'refreshLinked': {
        const link = requireLink(doc, surface.surfaceId);
        if (link.dataRevision !== action.dataRevision)
          throw new AppProblem('stale_revision', 'This linked source has already been refreshed.');
        if (sourceRevision === link.dataRevision) break;
        link.dataRevision = sourceRevision;
        link.rowKeys = [];
        delete link.selectionSourceId;
        link.filter = { kind: 'all' };
        link.filterRevision++;
        for (const member of doc.workspace.surfaces.filter((item) =>
          link.surfaceIds.includes(item.surfaceId),
        )) {
          if (member.view === 'table') delete member.table;
        }
        break;
      }
    }
  } catch (error) {
    if (error instanceof AppProblem) throw error;
    throw new AppProblem(
      'invalid_request',
      error instanceof Error ? error.message : 'These view settings are invalid.',
    );
  }
  const claimed = doc.workspace.surfaces.find((item) => item.surfaceId === action.surfaceId);
  if (claimed?.openedBy.kind === 'agent') claimed.userClaimed = true;
}
