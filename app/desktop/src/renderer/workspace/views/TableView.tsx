import { useEffect, useRef, useState } from 'react';
import { markStyle, type MarkProps } from '../../shared-context/marks';
import { referenceAuthor, type FileContent, type Selection } from '@gobble/contracts';

type Table = Extract<FileContent['content'], { kind: 'table' }>;
export function TableView({
  content,
  selection,
  onReady,
  onSelect,
  canSelect,
  canChooseColumns = true,
  includedColumns,
  sourceRows,
  emptyLabel,
  onColumnsChange,
  marks = [],
  reveal,
  referenceRequestId,
}: MarkProps & {
  content: Table;
  referenceRequestId?: string | undefined;
  selection: Selection | undefined;
  onReady: () => void;
  onSelect: (selection: Selection | null) => Promise<boolean>;
  canSelect: boolean;
  canChooseColumns?: boolean;
  includedColumns?: string[] | undefined;
  sourceRows?: number;
  emptyLabel?: string | undefined;
  onColumnsChange?: ((columns: string[]) => Promise<boolean>) | undefined;
}) {
  const viewport = useRef<HTMLDivElement>(null);
  const [pending, setPending] = useState(false);
  const [pendingKeys, setPendingKeys] = useState<string[] | null>(null);
  const [pendingColumns, setPendingColumns] = useState<string[] | null>(null);
  useEffect(onReady, [onReady]);
  const selected = pendingKeys ?? (selection?.kind === 'table' ? selection.rowKeys : []);
  const columns =
    pendingColumns ??
    includedColumns ??
    (selection?.kind === 'table' ? selection.columns : content.columns.map((column) => column.id));
  const bookmark = useRef<{ top: number; left: number } | null>(null);
  const lastRevealed = useRef<string | undefined>(undefined);
  const rowIdentity = JSON.stringify(content.rows.map((row) => row.key));
  const markIdentity = JSON.stringify(marks.map((mark) => mark.referenceId));
  const plotReference = marks.some(
    (mark) => mark.referenceId === reveal?.referenceId && !!mark.presentation,
  );
  useEffect(() => {
    const element = viewport.current;
    if (!element) return;
    if (referenceRequestId && !bookmark.current)
      bookmark.current = { top: element.scrollTop, left: element.scrollLeft };
    if (!referenceRequestId && bookmark.current) {
      element.scrollTo(bookmark.current);
      bookmark.current = null;
    }
  }, [referenceRequestId]);
  useEffect(() => {
    if (plotReference && referenceRequestId !== reveal?.requestId) return;
    if (lastRevealed.current === reveal?.requestId) return;
    if (!reveal || !viewport.current) return;
    const cell = [...viewport.current.querySelectorAll<HTMLElement>('[data-reference-ids]')].find(
      (item) => item.dataset.referenceIds?.split(' ').includes(reveal.referenceId),
    );
    if (cell) {
      const element = viewport.current;
      const target = cell.getBoundingClientRect(),
        bounds = element.getBoundingClientRect();
      const header = element.querySelector('thead')?.getBoundingClientRect().height ?? 0;
      element.scrollTo({
        top: element.scrollTop + target.top - bounds.top - header,
        left: Math.max(0, element.scrollLeft + target.left - bounds.left - 40),
      });
      lastRevealed.current = reveal.requestId;
    }
  }, [reveal?.requestId, referenceRequestId, rowIdentity, markIdentity, plotReference]);
  return (
    <div className="data-view">
      <div
        ref={viewport}
        className="table-scroll"
        tabIndex={0}
        aria-label="CSV data, scroll to see more columns"
      >
        <table className="data-table">
          <thead>
            <tr>
              <th scope="col" className="selection-column">
                <span className="sr-only">Select row</span>
              </th>
              {content.columns.map((column) => (
                <th scope="col" key={column.id}>
                  {!canChooseColumns ? (
                    <span>{column.name || 'Unnamed column'}</span>
                  ) : (
                    <label className="column-choice">
                      <input
                        type="checkbox"
                        aria-label={
                          'Include column ' + (column.name || 'Unnamed column') + ' in selection'
                        }
                        checked={columns.includes(column.id)}
                        disabled={
                          pending ||
                          !canSelect ||
                          !canChooseColumns ||
                          !selected.length ||
                          (columns.length === 1 && columns.includes(column.id))
                        }
                        onChange={(event) => {
                          const next = event.currentTarget.checked
                            ? [...columns, column.id]
                            : columns.filter((id) => id !== column.id);
                          setPending(true);
                          setPendingColumns(next);
                          void (
                            onColumnsChange
                              ? onColumnsChange(next)
                              : onSelect({
                                  kind: 'table',
                                  coordinateSpace: 'revision-row-column-keys',
                                  rowKeys: selected,
                                  columns: next,
                                })
                          ).finally(() => {
                            setPending(false);
                            setPendingColumns(null);
                          });
                        }}
                      />
                      {column.name || 'Unnamed column'}
                    </label>
                  )}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {content.rows.map((row, index) => (
              <tr
                key={row.key}
                data-row-key={row.key}
                className={selected.includes(row.key) ? 'selected-row' : undefined}
              >
                <td className="selection-column">
                  <input
                    type="checkbox"
                    aria-label={'Select row ' + (index + 1) + ': ' + (row.cells[0] ?? '')}
                    checked={selected.includes(row.key)}
                    disabled={pending || !canSelect}
                    onChange={(event) => {
                      const keys = event.currentTarget.checked
                        ? [...selected, row.key]
                        : selected.filter((key) => key !== row.key);
                      setPending(true);
                      setPendingKeys(keys);
                      void onSelect(
                        keys.length
                          ? {
                              kind: 'table',
                              coordinateSpace: 'revision-row-column-keys',
                              rowKeys: keys,
                              columns,
                            }
                          : null,
                      ).finally(() => {
                        setPending(false);
                        setPendingKeys(null);
                      });
                    }}
                  />
                </td>
                {row.cells.map((cell, column) => {
                  const matches = marks.filter((item) => {
                    const range = item.evidence.selection;
                    return (
                      range?.kind === 'table' &&
                      range.rowKeys.includes(row.key) &&
                      range.columns.includes(content.columns[column]?.id ?? '')
                    );
                  });
                  return (
                    <td
                      key={content.columns[column]?.id ?? column}
                      className={matches.length ? 'shared-cell' : undefined}
                      style={matches[0] ? markStyle(matches[0]) : undefined}
                      data-reference-ids={matches.map((item) => item.referenceId).join(' ')}
                      title={
                        matches.length
                          ? matches.map(referenceAuthor).join(', ') + ': ' + cell
                          : undefined
                      }
                    >
                      {cell}
                    </td>
                  );
                })}
              </tr>
            ))}
          </tbody>
        </table>
        {content.rows.length === 0 && (
          <p className="muted data-empty">
            {emptyLabel ?? 'This CSV contains a header and no data rows.'}
          </p>
        )}
      </div>
      <footer className="view-footer">
        <span>
          {content.rows.length}
          {sourceRows !== undefined && sourceRows !== content.rows.length
            ? ' of ' + sourceRows
            : ''}{' '}
          rows · CSV preview
          {content.truncated ? ' · More rows not shown' : ''}
        </span>
        <span>
          {selected.length ? selected.length + ' selected' : 'Read only'}
          {selected.some((id) => !content.rows.some((row) => row.key === id))
            ? ' · ' +
              selected.filter((id) => !content.rows.some((row) => row.key === id)).length +
              ' not shown'
            : ''}
        </span>
      </footer>
    </div>
  );
}
