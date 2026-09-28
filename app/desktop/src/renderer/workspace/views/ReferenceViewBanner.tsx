import { referenceAuthor, type ReferenceView, type SharedReference } from '@gobble/contracts';
import type { Command } from '../useWorkspace';

export function ReferenceViewBanner({
  view,
  reference,
  command,
}: {
  view: ReferenceView;
  reference: SharedReference | undefined;
  command: Command;
}) {
  return (
    <div className="reference-view-banner" role="status">
      <span>
        <strong>Reference view{reference ? ` · ${referenceAuthor(reference)}` : ''}</strong>
        <small>
          {view.adjustments.length ? 'Expanded to show the referenced rows. ' : ''}Your view
          settings are saved.
        </small>
      </span>
      <button
        onClick={(event) => {
          const surface = event.currentTarget.closest('.surface-view');
          void command({ kind: 'returnReferenceView', referenceRequestId: view.requestId }).then(
            (ok) => {
              if (ok) surface?.querySelector<HTMLElement>('.table-scroll')?.focus();
            },
          );
        }}
      >
        Return to my view
      </button>
    </div>
  );
}
