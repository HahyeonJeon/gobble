import {
  type SharedReference,
  type WorkspaceDocument,
  type EvidenceRef,
  sameResource,
} from '@gobble/contracts';
import { AppProblem } from '../problem';

export function requireEvidence(
  doc: WorkspaceDocument,
  evidence: EvidenceRef,
  surfaceId: string,
): void {
  const surface = doc.workspace.surfaces.find((item) => item.surfaceId === surfaceId);
  if (
    evidence.projectId !== doc.workspace.projectId ||
    !surface ||
    !sameResource(surface.resource, evidence.resource) ||
    ((evidence.schemaVersion === 5 || evidence.schemaVersion === 6) &&
      evidence.origin?.surfaceId !== surfaceId)
  )
    throw new AppProblem('forbidden', 'This reference belongs to another Project view.');
}
export function publishReference(
  doc: WorkspaceDocument,
  reference: Omit<SharedReference, 'label'> & { label?: string },
  sourceSurfaceId: string,
): void {
  const references = doc.sharedReferences ?? [];
  if (references.length >= 128)
    throw new AppProblem(
      'unsupported',
      'This Project has reached its shared reference limit. Existing references have been preserved.',
    );
  requireEvidence(doc, reference.evidence, sourceSurfaceId);
  doc.sharedReferences = [
    ...references,
    {
      ...reference,
      label:
        reference.label ??
        doc.titles.find((item) => item.surfaceId === sourceSurfaceId)?.title ??
        'Project resource',
    },
  ];
}
export function sharedReference(doc: WorkspaceDocument, id: string): SharedReference {
  const found = doc.sharedReferences?.find((item) => item.referenceId === id);
  if (!found) throw new AppProblem('not_found', 'This shared reference is unavailable.');
  return found;
}
