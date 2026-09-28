import { useState } from 'react';
import { capturedReference } from './captured-reference';
import { ModalDialog } from '../ui/ModalDialog';
import { EvidencePreview } from '../evidence/EvidencePreview';
import { referenceAuthor, type SharedReference, type WorkspaceDocument } from '@gobble/contracts';
import type { Command } from '../workspace/useWorkspace';
import { evidenceLabel, markStyle } from './marks';
export function ReferenceEvent({
  reference,
  document,
  command,
}: {
  reference: SharedReference;
  document: WorkspaceDocument;
  command: Command;
}) {
  const title = reference.label;
  const observed = reference.evidence.schemaVersion >= 3;
  const captured = capturedReference(document, reference.evidence);
  const [preview, setPreview] = useState(false);
  return (
    <article
      className="shared-reference-event"
      style={markStyle(reference)}
      aria-label={referenceAuthor(reference) + ' shared a reference'}
    >
      <div className="reference-heading">
        <div>
          <strong>
            {referenceAuthor(reference)} <span>pointed to {title}</span>
          </strong>
          <small>{evidenceLabel(reference.evidence)}</small>
        </div>
        {!reference.retracted && (
          <button
            onClick={() => void command({ kind: 'reveal', referenceId: reference.referenceId })}
          >
            {observed ? 'Show in view' : reference.presentation ? 'Show in views' : 'Reveal'}
          </button>
        )}
      </div>
      {reference.note && <p>{reference.note}</p>}
      {captured && (
        <button className="text-button" onClick={() => setPreview(true)}>
          View captured evidence
        </button>
      )}
      {preview && captured && (
        <ModalDialog title="Captured evidence" onClose={() => setPreview(false)}>
          <EvidencePreview request={captured.request} manifest={captured.manifest} />
        </ModalDialog>
      )}
      {reference.retracted ? (
        <small>Reference retracted · history preserved</small>
      ) : (
        <details className="reference-actions">
          <summary>Mark options</summary>
          {(reference.evidence.schemaVersion === 3 || reference.evidence.schemaVersion === 4) && (
            <button
              className="text-button"
              onClick={() =>
                void command({
                  kind:
                    reference.evidence.schemaVersion === 4 ? 'currentDependency' : 'currentTask',
                  referenceId: reference.referenceId,
                })
              }
            >
              {reference.evidence.schemaVersion === 4
                ? reference.evidence.selection.kind === 'run-group'
                  ? 'Find current group'
                  : 'Find current dependency'
                : 'Find current task'}
            </button>
          )}
          <button
            className="text-button"
            onClick={() => void command({ kind: 'retract', referenceId: reference.referenceId })}
          >
            Retract mark
          </button>
        </details>
      )}
    </article>
  );
}
