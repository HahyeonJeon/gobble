import { type ScatterSpec, type WorkspaceDocument } from '@gobble/contracts';
import { ensureSplit, getPane } from '../../src/main/workspace/model';

/** Stored pre-retirement fixture only. Product code cannot create these views. */
export function addLegacyChart(
  doc: WorkspaceDocument,
  spec: ScatterSpec,
  revision: string,
  id = 'srf_plot',
) {
  const source = doc.workspace.surfaces.find((s) => s.view === 'table')!;
  ensureSplit(doc);
  doc.workspace.surfaces.push({
    ...source,
    surfaceId: id,
    view: 'scatter',
    scatter: { spec, specRevision: 1, viewRevision: 0, viewport: null },
  });
  doc.titles.push({ surfaceId: id, title: 'Archived chart' });
  getPane(doc, 'primary').tabs = [id];
  getPane(doc, 'primary').activeSurfaceId = id;
  getPane(doc, 'secondary').tabs = [source.surfaceId];
  getPane(doc, 'secondary').activeSurfaceId = source.surfaceId;
  const selection = doc.selections.find((s) => s.surfaceId === source.surfaceId)?.evidence
    .selection;
  doc.viewLinks.push({
    linkId: 'lnk_plot',
    resource: source.resource as { kind: 'file'; resourceId: string },
    dataRevision: revision,
    surfaceIds: [source.surfaceId, id],
    filterRevision: 0,
    filter: { kind: 'all' },
    rowKeys: selection?.kind === 'table' ? [...selection.rowKeys] : [],
    selectionSourceId: source.surfaceId,
  });
  return doc;
}
