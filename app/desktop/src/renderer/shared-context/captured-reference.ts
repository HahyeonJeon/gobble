import {
  containsObservedTarget,
  isQuestion,
  type EvidenceManifest,
  type EvidenceRef,
  type PreviewEvidence,
  type WorkspaceDocument,
} from '@gobble/contracts';

/** Locate an existing immutable capture; a pointer never creates or owns an evidence asset. */
export function capturedReference(
  document: WorkspaceDocument,
  evidence: EvidenceRef,
): { manifest: EvidenceManifest; request: PreviewEvidence } | undefined {
  const projectId = document.workspace.projectId;
  for (const submission of document.collaboration?.submissions ?? []) {
    const manifest = submission.evidence?.find(
      (item) =>
        item.capture && item.evidence.selection && containsObservedTarget(item.evidence, evidence),
    );
    if (manifest)
      return {
        manifest,
        request: {
          kind: 'sent',
          projectId,
          requestId: submission.requestId,
          attachmentId: manifest.attachmentId,
        },
      };
  }
  for (const question of document.workspace.decisions.filter(isQuestion)) {
    const manifest = question.evidence.find(
      (item) =>
        item.capture && item.evidence.selection && containsObservedTarget(item.evidence, evidence),
    );
    if (manifest)
      return {
        manifest,
        request: {
          kind: 'question',
          projectId,
          questionId: question.decisionId,
          attachmentId: manifest.attachmentId,
        },
      };
  }
  return undefined;
}
