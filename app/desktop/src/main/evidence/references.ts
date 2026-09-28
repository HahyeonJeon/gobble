import { isQuestion, type WorkspaceDocument } from '@gobble/contracts';

/** Explicit ownership traversal; unknown document versions must fail before reaching this function. */
export function retainedEvidenceHashes(documents: WorkspaceDocument[]): Set<string> {
  const hashes = new Set<string>();
  for (const doc of documents) {
    for (const report of doc.savedReports ?? []) hashes.add(report.asset.hash);
    for (const ref of doc.sharedReferences ?? [])
      if (ref.evidence.schemaVersion === 8) hashes.add(ref.evidence.resource.saved.asset.hash);
    for (const surface of doc.workspace.surfaces)
      if (surface.resource.kind === 'report') hashes.add(surface.resource.saved.asset.hash);
    for (const draft of doc.chat.attachments ?? [])
      if (draft.capture) hashes.add(draft.capture.asset.hash);
    for (const submission of doc.collaboration?.submissions ?? [])
      for (const evidence of submission.evidence ?? []) hashes.add(evidence.asset.hash);
    for (const decision of doc.workspace.decisions)
      if (isQuestion(decision))
        for (const evidence of decision.evidence) hashes.add(evidence.asset.hash);
  }
  return hashes;
}
