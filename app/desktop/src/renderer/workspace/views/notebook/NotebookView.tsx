import { useNotebookViewport } from './useNotebookViewport';
import { useEffect, useRef, useState } from 'react';
import {
  notebookPart,
  type NotebookDocument,
  type SharedReference,
  type NotebookSelection,
  type Surface,
  type EvidenceRef,
  type RenderAcknowledgment,
} from '@gobble/contracts';
import type { Command } from '../../useWorkspace';
import { SelectionToolbar } from '../../../selections/SelectionToolbar';
import { NotebookCell } from './NotebookCell';
import '../../../styles/notebook.css';

export function NotebookView({
  document,
  surface,
  title,
  acknowledgment,
  ready,
  onReady,
  evidence,
  command,
  onSelect,
  onRefresh,
  onNavigate,
  marks,
}: {
  document: NotebookDocument;
  marks: SharedReference[];
  surface: Extract<Surface, { view: 'notebook' }>;
  title: string;
  acknowledgment: RenderAcknowledgment;
  ready: boolean;
  onReady: () => void;
  evidence: EvidenceRef | null;
  command: Command;
  onSelect: (selection: NotebookSelection) => Promise<boolean>;
  onRefresh: () => void;
  onNavigate: (page: number) => Promise<boolean>;
}) {
  const scroll = useRef<HTMLDivElement>(null);
  useNotebookViewport(scroll, acknowledgment, ready);
  const [imageKey, setImageKey] = useState<string>();
  const [selecting, setSelecting] = useState(false);
  const selectionPending = useRef(false);
  const interactive = ready && !selecting;
  // The old selection must not remain attachable while a new intent is being saved.
  async function select(selection: NotebookSelection): Promise<boolean> {
    if (!ready || selectionPending.current) return false;
    selectionPending.current = true;
    setSelecting(true);
    try {
      return await onSelect(selection);
    } finally {
      selectionPending.current = false;
      setSelecting(false);
    }
  }
  useEffect(onReady, [onReady]);
  const pages = Math.max(1, Math.ceil(document.cells.length / 20)),
    storedPage = surface.notebook?.page ?? 0;
  const page = Math.min(storedPage, pages - 1);
  useEffect(() => setImageKey(undefined), [page]);
  let selected: NotebookSelection | undefined;
  if (evidence?.schemaVersion === 6 && evidence.dataRevision === document.revision) {
    try {
      notebookPart(document, evidence.selection);
      selected = evidence.selection;
    } catch {
      /* Stale local intent is not highlighted. */
    }
  }
  if (surface.resource.kind !== 'file') return <p role="alert">A Notebook file is required.</p>;
  const target = {
    schemaVersion: 6 as const,
    projectId: surface.projectId,
    resource: surface.resource,
    origin: { surfaceId: surface.surfaceId },
    dataRevision: document.revision,
  };
  return (
    <div
      className="notebook-surface"
      data-ready={interactive}
      aria-label={title + ' Notebook reader'}
    >
      <div className="notebook-toolbar">
        <span>
          {document.language} · {document.cells.length} cells
        </span>
        {pages > 1 && (
          <>
            <button
              aria-label="Previous Notebook cells"
              disabled={!interactive || page === 0}
              onClick={() => void onNavigate(page - 1)}
            >
              Previous
            </button>
            <span>
              Cells {page * 20 + 1}–{Math.min(document.cells.length, (page + 1) * 20)}
            </span>
            <button
              aria-label="Next Notebook cells"
              disabled={!interactive || page + 1 === pages}
              onClick={() => void onNavigate(page + 1)}
            >
              Next
            </button>
          </>
        )}
        <button disabled={!interactive} onClick={onRefresh}>
          Refresh Notebook
        </button>
      </div>
      <details className="notebook-support">
        <summary>Saved Notebook · Read only</summary>
        <p>
          Code and saved outputs from this file version. Execution is not verified. Basic Markdown,
          text, PNG and JPEG are supported; widgets, HTML and interactive content are unavailable.
          Large text is shown in bounded sections. Refresh reads the current file.
        </p>
      </details>
      {selected && evidence && (
        <SelectionToolbar
          key={JSON.stringify(evidence)}
          evidence={evidence}
          surfaceId={surface.surfaceId}
          title={title}
          ready={interactive}
          acknowledgment={acknowledgment}
          command={command}
        />
      )}
      {storedPage !== page && (
        <p className="notebook-notice">
          The saved cell page is no longer available. Showing the last available cells.
        </p>
      )}
      <div className="notebook-scroll" ref={scroll}>
        {!document.cells.length && <p className="empty-state">This saved Notebook has no cells.</p>}
        {document.cells.slice(page * 20, (page + 1) * 20).map((cell) => (
          <NotebookCell
            key={cell.index}
            cell={cell}
            marks={marks.filter(
              (mark) =>
                !mark.retracted &&
                mark.evidence.schemaVersion === 6 &&
                mark.evidence.dataRevision === document.revision &&
                mark.evidence.projectId === surface.projectId &&
                mark.evidence.resource.resourceId === target.resource.resourceId,
            )}
            target={target}
            acknowledgment={acknowledgment}
            ready={interactive}
            selected={selected}
            imageKey={imageKey}
            onImage={setImageKey}
            onSelect={select}
          />
        ))}
      </div>
    </div>
  );
}
