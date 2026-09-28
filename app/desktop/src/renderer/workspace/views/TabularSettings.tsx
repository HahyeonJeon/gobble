import { useEffect, useRef, useState, type ReactNode } from 'react';
import {
  MAX_TABLE_FILTER_VALUE_LENGTH,
  numericColumns,
  type TableContent,
  type TableFilter,
  type TableState,
} from '@gobble/contracts';
import { ModalDialog } from '../../ui/ModalDialog';

function SettingsForm({
  children,
  apply,
  onClose,
  label = 'Apply',
  valid = true,
}: {
  children: ReactNode;
  apply: () => Promise<boolean>;
  onClose: () => void;
  label?: string;
  valid?: boolean;
}) {
  const [pending, setPending] = useState(false),
    [failed, setFailed] = useState(false);
  const working = useRef(false),
    mounted = useRef(true);
  useEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
    };
  }, []);
  return (
    <form
      className="tabular-settings"
      onSubmit={(event) => {
        event.preventDefault();
        if (working.current || !valid) return;
        working.current = true;
        setPending(true);
        setFailed(false);
        void apply()
          .then((ok) => {
            if (mounted.current) {
              if (ok) onClose();
              else setFailed(true);
            }
          })
          .catch(() => {
            if (mounted.current) setFailed(true);
          })
          .finally(() => {
            working.current = false;
            if (mounted.current) setPending(false);
          });
      }}
    >
      <fieldset disabled={pending}>{children}</fieldset>
      {failed && (
        <p role="alert">
          Settings were not applied. Close this dialog and try from the current view.
        </p>
      )}
      <footer>
        <button type="button" onClick={onClose}>
          Cancel
        </button>
        <button type="submit" disabled={pending || !valid}>
          {pending ? 'Applying…' : label}
        </button>
      </footer>
    </form>
  );
}
export function FilterSettings({
  content,
  initial,
  onApply,
  onClose,
}: {
  content: TableContent;
  initial: TableFilter;
  onApply: (filter: TableFilter) => Promise<boolean>;
  onClose: () => void;
}) {
  const [columnId, setColumnId] = useState(initial.kind === 'equals' ? initial.columnId : '');
  const [value, setValue] = useState(initial.kind === 'equals' ? initial.value : '');
  const index = content.columns.findIndex((column) => column.id === columnId);
  const values = [...new Set(content.rows.map((row) => row.cells[index] ?? ''))];
  return (
    <ModalDialog title="Filter linked views" onClose={onClose}>
      <SettingsForm
        onClose={onClose}
        label="Apply filter"
        valid={!columnId || value.length <= MAX_TABLE_FILTER_VALUE_LENGTH}
        apply={() => onApply(columnId ? { kind: 'equals', columnId, value } : { kind: 'all' })}
      >
        <p>
          Apply the same exact-value filter to table and plot. Selected rows outside the filter stay
          selected.
        </p>
        <label>
          Filter column
          <select
            value={columnId}
            onChange={(event) => {
              const id = event.currentTarget.value;
              setColumnId(id);
              setValue(content.rows[0]?.cells[content.columns.findIndex((c) => c.id === id)] ?? '');
            }}
          >
            <option value="">All rows</option>
            {content.columns.map((column) => (
              <option key={column.id} value={column.id}>
                {column.name || 'Unnamed column'}
              </option>
            ))}
          </select>
        </label>
        {columnId && (
          <label>
            Equals
            <select value={value} onChange={(event) => setValue(event.currentTarget.value)}>
              {!values.includes(value) && <option value={value}>{value || '(blank)'}</option>}
              {values.map((item) => (
                <option
                  key={item}
                  value={item}
                  disabled={item.length > MAX_TABLE_FILTER_VALUE_LENGTH}
                >
                  {item.length > MAX_TABLE_FILTER_VALUE_LENGTH
                    ? item.slice(0, 80) + '… (too long to filter)'
                    : item || '(blank)'}
                </option>
              ))}
            </select>
          </label>
        )}
        {columnId && values.some((item) => item.length > MAX_TABLE_FILTER_VALUE_LENGTH) && (
          <small>
            Values longer than 4,096 characters cannot be used as filters. Choose another value or
            column.
          </small>
        )}
      </SettingsForm>
    </ModalDialog>
  );
}
export function TableSettings({
  content,
  initial,
  onApply,
  onClose,
}: {
  content: TableContent;
  initial: Pick<TableState, 'sort' | 'columns'>;
  onApply: (settings: Pick<TableState, 'sort' | 'columns'>) => Promise<boolean>;
  onClose: () => void;
}) {
  const [sort, setSort] = useState(initial.sort);
  const [columns, setColumns] = useState(
    initial.columns ?? content.columns.map((column) => column.id),
  );
  const numeric = new Set(numericColumns(content).map((column) => column.id));
  return (
    <ModalDialog title="Table settings" onClose={onClose}>
      <SettingsForm onClose={onClose} apply={() => onApply({ sort, columns })}>
        <label>
          Sort by
          <select
            value={sort?.columnId ?? ''}
            onChange={(event) => {
              const id = event.currentTarget.value;
              setSort(
                id ? { columnId: id, direction: 'ascending', numeric: numeric.has(id) } : null,
              );
            }}
          >
            <option value="">Source order</option>
            {content.columns.map((column) => (
              <option key={column.id} value={column.id}>
                {column.name || 'Unnamed column'}
              </option>
            ))}
          </select>
        </label>
        {sort && (
          <div className="settings-pair">
            <label>
              Direction
              <select
                value={sort.direction}
                onChange={(event) =>
                  setSort({
                    ...sort,
                    direction: event.currentTarget.value as 'ascending' | 'descending',
                  })
                }
              >
                <option value="ascending">Ascending</option>
                <option value="descending">Descending</option>
              </select>
            </label>
            <label>
              Compare as
              <select
                value={sort.numeric ? 'numeric' : 'text'}
                onChange={(event) =>
                  setSort({ ...sort, numeric: event.currentTarget.value === 'numeric' })
                }
              >
                <option value="numeric">Numbers</option>
                <option value="text">Text</option>
              </select>
            </label>
          </div>
        )}
        <p>
          Columns to include when discussing a table selection. All source columns remain visible.
        </p>
        <div className="settings-columns">
          {content.columns.map((column) => (
            <label key={column.id}>
              <input
                type="checkbox"
                checked={columns.includes(column.id)}
                disabled={columns.length === 1 && columns.includes(column.id)}
                onChange={(event) =>
                  setColumns(
                    event.currentTarget.checked
                      ? [...columns, column.id]
                      : columns.filter((id) => id !== column.id),
                  )
                }
              />
              {column.name || 'Unnamed column'}
            </label>
          ))}
        </div>
        <small>Sorting changes display order only. Invalid numbers stay last.</small>
      </SettingsForm>
    </ModalDialog>
  );
}
