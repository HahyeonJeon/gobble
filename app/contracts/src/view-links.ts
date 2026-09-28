import type { WorkspaceDocument } from './workspace-document';
import type { Surface } from './surface';
import type { PresentationRevision, ViewLink } from './tabular-view';
import { sameResource } from './resource';
import { ContractValidationError } from './validation-error';

export function viewLinkFor(doc: WorkspaceDocument, surfaceId: string): ViewLink | undefined {
  return doc.viewLinks.find((link) => link.surfaceIds.includes(surfaceId));
}
export function presentationRevision(surface: Surface, link?: ViewLink): PresentationRevision {
  return {
    spec: surface.view === 'scatter' ? surface.scatter.specRevision : 0,
    view:
      surface.view === 'scatter'
        ? surface.scatter.viewRevision
        : surface.view === 'table'
          ? (surface.table?.viewRevision ?? 0)
          : surface.view === 'run'
            ? (surface.runView?.viewRevision ?? 0)
            : surface.view === 'log'
              ? (surface.logView?.viewRevision ?? 0)
              : 0,
    filter: link?.filterRevision ?? 0,
  };
}
export function samePresentation(a: PresentationRevision, b: PresentationRevision): boolean {
  return a.spec === b.spec && a.view === b.view && a.filter === b.filter;
}
export function validateViewLinks(doc: WorkspaceDocument): void {
  const ids = new Set<string>(),
    linked = new Set<string>();
  for (const link of doc.viewLinks) {
    if (ids.has(link.linkId)) throw new ContractValidationError('View link IDs must be unique.');
    ids.add(link.linkId);
    if (link.selectionSourceId && !link.surfaceIds.includes(link.selectionSourceId))
      throw new ContractValidationError('Selection origin must belong to its linked group.');
    for (const id of link.surfaceIds) {
      const surface = doc.workspace.surfaces.find((item) => item.surfaceId === id);
      if (
        !surface ||
        !['table', 'scatter'].includes(surface.view) ||
        !sameResource(surface.resource, link.resource)
      )
        throw new ContractValidationError('Linked views must address the same tabular resource.');
      if (linked.has(id) || doc.selections.some((item) => item.surfaceId === id))
        throw new ContractValidationError('A linked view must have exactly one membership owner.');
      linked.add(id);
    }
  }
}
