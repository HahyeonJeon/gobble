import { useState } from 'react';
import {
  NOTEBOOK_PROFILE,
  notebookSelectionContains,
  type SharedReference,
  type NotebookDocument,
  type NotebookTarget,
  type NotebookSelection,
  type RenderAcknowledgment,
} from '@gobble/contracts';
import { NotebookText } from './NotebookText';
import { NotebookImage } from './NotebookImage';
import { textWindow } from './text-window';

export function NotebookCell({
  cell,
  marks,
  target,
  acknowledgment,
  ready,
  selected,
  imageKey,
  onImage,
  onSelect,
}: {
  cell: NotebookDocument['cells'][number];
  marks: SharedReference[];
  target: Omit<NotebookTarget, 'selection'>;
  acknowledgment: RenderAcknowledgment;
  ready: boolean;
  selected: NotebookSelection | undefined;
  imageKey: string | undefined;
  onImage: (key: string | undefined) => void;
  onSelect: (selection: NotebookSelection) => Promise<boolean>;
}) {
  const [source, setSource] = useState(cell.type !== 'markdown');
  const [outputs, setOutputs] = useState(true);
  const selection = (
    part: NotebookSelection['part'],
    selector: NotebookSelection['selector'],
  ): NotebookSelection => ({
    kind: 'notebook',
    profile: NOTEBOOK_PROFILE,
    cell: cell.address,
    part,
    selector,
  });
  const marksFor = (part: NotebookSelection['part'], selector: NotebookSelection['selector']) =>
    marks.filter(
      (mark) =>
        mark.evidence.schemaVersion === 6 &&
        notebookSelectionContains(selection(part, selector), mark.evidence.selection),
    );
  const codeLabel = 'Cell ' + (cell.index + 1) + ' source';
  const selectedCell =
    selected &&
    (cell.address.kind === 'id'
      ? selected.cell.kind === 'id' && selected.cell.id === cell.address.id
      : selected.cell.kind === 'ordinal' && selected.cell.index === cell.index);
  const selectedText =
    selectedCell && selected.part.kind === 'source' && selected.selector.kind === 'text'
      ? selected.selector
      : undefined;
  return (
    <article className="notebook-cell" aria-label={'Notebook cell ' + (cell.index + 1)}>
      <div className="notebook-cell-header">
        <strong>Cell {cell.index + 1}</strong>
        <span>
          {cell.type === 'code' ? 'Code' : cell.type === 'markdown' ? 'Markdown' : 'Source'}
        </span>
        {cell.type === 'markdown' && (
          <button disabled={!ready} aria-pressed={source} onClick={() => setSource(!source)}>
            {source ? 'Read Markdown' : 'View source'}
          </button>
        )}
      </div>
      {source ? (
        <NotebookText
          text={cell.source.text}
          address={selection(
            { kind: 'source' },
            {
              kind: 'text',
              coordinateSpace: 'notebook-display-utf16',
              start: 0,
              end: cell.source.text.length,
            },
          )}
          marks={marksFor(
            { kind: 'source' },
            {
              kind: 'text',
              coordinateSpace: 'notebook-display-utf16',
              start: 0,
              end: cell.source.text.length,
            },
          )}
          label={codeLabel}
          ready={ready}
          selected={selectedText}
          onSelect={(selector) => onSelect(selection({ kind: 'source' }, selector))}
        />
      ) : (
        <div className="notebook-markdown">
          {textWindow(cell.source.text, 0)
            .text.split('\n')
            .map((line, i) =>
              line.startsWith('# ') ? (
                <h3 key={i}>{line.slice(2)}</h3>
              ) : (
                <p key={i}>{line || '\u00a0'}</p>
              ),
            )}
          <small>Basic saved Markdown · Use Source for exact text and continuation</small>
        </div>
      )}
      {cell.notice && <p className="notebook-notice">{cell.notice}</p>}
      {cell.outputs.length > 0 && (
        <>
          <div className="notebook-output-header">
            <span>Saved outputs · {cell.outputs.length}</span>
            <button
              disabled={!ready}
              onClick={() => {
                setOutputs(!outputs);
                if (outputs) onImage(undefined);
              }}
            >
              {outputs ? 'Collapse outputs' : 'Show outputs'}
            </button>
          </div>
          {outputs &&
            cell.outputs.map((output) => {
              const part = output.part,
                key = cell.index + ':' + output.index;
              const address: NotebookSelection['part'] = {
                kind: 'output',
                index: output.index,
                mime: output.mime,
                digest: part.kind === 'unavailable' ? target.dataRevision : part.digest,
              };
              const chosen =
                selectedCell &&
                selected.part.kind === 'output' &&
                selected.part.index === output.index &&
                selected.selector.kind === 'text'
                  ? selected.selector
                  : undefined;
              return (
                <section
                  className="notebook-output"
                  key={output.index}
                  aria-label={'Cell ' + (cell.index + 1) + ' output ' + (output.index + 1)}
                >
                  <small>
                    Output {output.index + 1} · {output.mime || 'Unavailable'}
                  </small>
                  {output.notice && <p className="notebook-notice">{output.notice}</p>}
                  {part.kind === 'text' ? (
                    <NotebookText
                      text={part.text}
                      address={selection(address, {
                        kind: 'text',
                        coordinateSpace: 'notebook-display-utf16',
                        start: 0,
                        end: part.text.length,
                      })}
                      marks={marksFor(address, {
                        kind: 'text',
                        coordinateSpace: 'notebook-display-utf16',
                        start: 0,
                        end: part.text.length,
                      })}
                      label={'Cell ' + (cell.index + 1) + ' output ' + (output.index + 1) + ' text'}
                      selected={chosen}
                      ready={ready}
                      onSelect={(selector) => onSelect(selection(address, selector))}
                    />
                  ) : part.kind === 'image' ? (
                    <NotebookImage
                      marks={marksFor(address, {
                        kind: 'image',
                        coordinateSpace: 'natural-image-pixels',
                        rect: { x: 0, y: 0, width: part.width, height: part.height },
                      })}
                      target={{
                        ...target,
                        selection: selection(address, {
                          kind: 'image',
                          coordinateSpace: 'natural-image-pixels',
                          rect: { x: 0, y: 0, width: part.width, height: part.height },
                        }),
                      }}
                      acknowledgment={acknowledgment}
                      width={part.width}
                      height={part.height}
                      active={imageKey === key}
                      onActivate={(active) => onImage(active ? key : undefined)}
                      ready={ready}
                      onSelect={(selector) => onSelect(selection(address, selector))}
                    />
                  ) : (
                    <p className="notebook-notice">{part.reason}</p>
                  )}
                </section>
              );
            })}
        </>
      )}
    </article>
  );
}
