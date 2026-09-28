import { useRef, useState } from 'react';
import { evidenceLabel } from '../shared-context/marks';
import type { Command } from '../workspace/useWorkspace';
import '../styles/selection.css';
import type { WorkspaceDocument } from '@gobble/contracts';
import { ModalDialog } from '../ui/ModalDialog';

/** The overview locates saved selections, including views that are not currently presented. */
export function SelectionMenu({
  document,
  command,
  onReveal,
}: {
  document: Pick<WorkspaceDocument, 'selections' | 'titles' | 'viewLinks'>;
  command: Command;
  onReveal: (surfaceId: string) => void;
}) {
  const [open, setOpen] = useState(false);
  const [pending, setPending] = useState(false);
  const working = useRef(false);
  const generation = useRef(0);
  const revealed = useRef<string | null>(null);
  const selections = [
    ...document.selections.map((item) => ({
      id: item.surfaceId,
      surfaceId: item.surfaceId,
      label: evidenceLabel(item.evidence),
    })),
    ...document.viewLinks
      .filter((link) => link.rowKeys.length)
      .map((link) => ({
        id: link.linkId,
        surfaceId: link.surfaceIds[0]!,
        label: `${link.rowKeys.length} rows · Linked views`,
      })),
  ];
  function dismiss() {
    generation.current += 1;
    setOpen(false);
  }
  async function reveal(surfaceId: string) {
    if (working.current) return;
    working.current = true;
    const issued = generation.current;
    setPending(true);
    try {
      if ((await command({ kind: 'activate', surfaceId })) && issued === generation.current) {
        revealed.current = surfaceId;
        setOpen(false);
      }
    } finally {
      working.current = false;
      setPending(false);
    }
  }
  return (
    <>
      <button
        aria-label="Project selections"
        aria-haspopup="dialog"
        aria-expanded={open}
        onClick={() => {
          generation.current += 1;
          revealed.current = null;
          setOpen(true);
        }}
      >
        Selections <span className="selection-count">{selections.length}</span>
      </button>
      {open && (
        <ModalDialog
          title="Project selections"
          className="selection-dialog"
          onClose={dismiss}
          onAfterClose={() => {
            if (revealed.current) onReveal(revealed.current);
          }}
        >
          <p className="muted">
            Selections stay local until you add them to a message or share a pointer. Return to a
            source to use its selection.
          </p>
          {selections.length === 0 && (
            <p>No selections yet. Select rows, text or an image region in a view.</p>
          )}
          <ul className="selection-list">
            {selections.map((item) => {
              const title =
                document.titles.find((entry) => entry.surfaceId === item.surfaceId)?.title ??
                'View';
              return (
                <li key={item.id}>
                  <div>
                    <strong>{title}</strong>
                    <small>{item.label}</small>
                  </div>
                  <button
                    disabled={pending}
                    aria-label={'Return to selection in ' + title}
                    onClick={() => void reveal(item.surfaceId)}
                  >
                    Show in pane
                  </button>
                </li>
              );
            })}
          </ul>
        </ModalDialog>
      )}
    </>
  );
}
