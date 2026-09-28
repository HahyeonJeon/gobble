import type { FileContent } from './file';
import type {
  PlotViewport,
  ScatterSpec,
  TableFilter,
  TableSort,
  TableState,
  ViewLink,
} from './tabular-view';
import type { Surface } from './surface';
import { ContractValidationError } from './validation-error';

export type TableContent = Extract<FileContent['content'], { kind: 'table' }>;
export type ScatterPoint = Readonly<{ rowKey: string; x: number; y: number; label: string }>;

/** ASCII decimal/scientific notation only. Empty, hex, locale grouping and non-finite values are not zero. */
export function numericCell(raw: string): number | null {
  const text = raw.trim();
  if (!/^[+-]?(?:\d+(?:\.\d*)?|\.\d+)(?:[eE][+-]?\d+)?$/.test(text)) return null;
  const value = Number(text);
  const underflow = value === 0 && /[1-9]/.test(text.split(/[eE]/)[0]!);
  return Number.isFinite(value) && !underflow ? value : null;
}

export function validateViewport(viewport: PlotViewport | null): void {
  if (
    viewport &&
    [viewport.x, viewport.y].some(
      ([lo, hi]) => !Number.isFinite(lo) || !Number.isFinite(hi) || lo >= hi,
    )
  )
    throw new ContractValidationError('Plot ranges must be finite and increasing.');
}

export function tableColumnIndex(content: TableContent, id: string): number {
  const matches = content.columns.flatMap((column, index) => (column.id === id ? [index] : []));
  if (matches.length !== 1)
    throw new ContractValidationError('A view column is missing or ambiguous.');
  return matches[0]!;
}

export function filterTable(content: TableContent, filter: TableFilter): TableContent['rows'] {
  if (filter.kind === 'all') return content.rows;
  const index = tableColumnIndex(content, filter.columnId);
  return content.rows.filter((row) => row.cells[index] === filter.value);
}

/** Pure, shared by the host and renderer. Raw source cells are never overwritten. */
export function projectScatter(
  content: TableContent,
  spec: ScatterSpec,
  filter: TableFilter = { kind: 'all' },
) {
  if (new Set(content.rows.map((row) => row.key)).size !== content.rows.length)
    throw new ContractValidationError('Source row keys must be unique within this revision.');
  const x = tableColumnIndex(content, spec.xColumnId);
  const y = tableColumnIndex(content, spec.yColumnId);
  const label =
    spec.labelColumnId === undefined ? undefined : tableColumnIndex(content, spec.labelColumnId);
  const rows = filterTable(content, filter);
  const points: ScatterPoint[] = [];
  for (const row of rows) {
    const px = numericCell(row.cells[x] ?? '');
    const py = numericCell(row.cells[y] ?? '');
    if (px !== null && py !== null)
      points.push({
        rowKey: row.key,
        x: px,
        y: py,
        label: label === undefined ? row.key : (row.cells[label] ?? ''),
      });
  }
  return {
    points,
    availableRows: content.rows.length,
    filteredRows: rows.length,
    excludedRows: rows.length - points.length,
    truncated: content.truncated,
  };
}

/** Bounds describe a completed gesture; exact members, not the predicate, become the selection. */
export function rowsInBox(points: readonly ScatterPoint[], bounds: PlotViewport): string[] {
  validateViewport(bounds);
  return points
    .filter(
      (p) => p.x >= bounds.x[0] && p.x <= bounds.x[1] && p.y >= bounds.y[0] && p.y <= bounds.y[1],
    )
    .map((p) => p.rowKey);
}

export function scatterColumns(spec: ScatterSpec): string[] {
  return [
    ...new Set([
      spec.xColumnId,
      spec.yColumnId,
      ...(spec.labelColumnId ? [spec.labelColumnId] : []),
    ]),
  ];
}

export function numericColumns(content: TableContent) {
  return content.columns.flatMap((column, index) => {
    const valid = content.rows.filter((row) => numericCell(row.cells[index] ?? '') !== null).length;
    return valid ? [{ ...column, valid, excluded: content.rows.length - valid }] : [];
  });
}
export function defaultScatterSpec(content: TableContent): ScatterSpec | null {
  const columns = numericColumns(content);
  if (columns.length < 2) return null;
  const label = content.columns.find(
    (column) => !columns.some((numeric) => numeric.id === column.id),
  );
  return {
    xColumnId: columns[0]!.id,
    yColumnId: columns[1]!.id,
    xScale: 'linear',
    yScale: 'linear',
    ...(label ? { labelColumnId: label.id } : {}),
  };
}
export function validateScatterSpec(content: TableContent, spec: ScatterSpec): void {
  projectScatter(content, spec);
  const numeric = new Set(numericColumns(content).map((column) => column.id));
  if (
    spec.xColumnId === spec.yColumnId ||
    !numeric.has(spec.xColumnId) ||
    !numeric.has(spec.yColumnId)
  )
    throw new ContractValidationError('Choose two different columns with finite numeric values.');
}
export function validateTableSettings(
  content: TableContent,
  settings: Pick<TableState, 'sort' | 'columns'>,
): void {
  if (settings.sort) tableColumnIndex(content, settings.sort.columnId);
  for (const id of settings.columns ?? []) tableColumnIndex(content, id);
}
/** Sort only presentation; invalid numeric values stay last in either direction. Equal values retain source order. */
export function sortTable(
  content: TableContent,
  rows: TableContent['rows'],
  sort: TableSort,
): TableContent['rows'] {
  if (!sort) return rows;
  const index = tableColumnIndex(content, sort.columnId);
  const direction = sort.direction === 'ascending' ? 1 : -1;
  return [...rows].sort((a, b) => {
    const left = a.cells[index] ?? '',
      right = b.cells[index] ?? '';
    if (!sort.numeric) return (left < right ? -1 : left > right ? 1 : 0) * direction;
    const x = numericCell(left),
      y = numericCell(right);
    if (x === null) return y === null ? 0 : 1;
    if (y === null) return -1;
    return (x < y ? -1 : x > y ? 1 : 0) * direction;
  });
}
export function linkedSelectionColumns(
  surfaces: Surface[],
  link: ViewLink,
  content: TableContent,
): string[] {
  const source =
    surfaces.find((surface) => surface.surfaceId === link.selectionSourceId) ??
    surfaces.find(
      (surface) => link.surfaceIds.includes(surface.surfaceId) && surface.view === 'scatter',
    ) ??
    surfaces.find((surface) => link.surfaceIds.includes(surface.surfaceId));
  return source?.view === 'scatter'
    ? scatterColumns(source.scatter.spec)
    : source?.view === 'table' && source.table?.columns
      ? source.table.columns
      : content.columns.map((column) => column.id);
}

/** A finite default viewport, including empty/constant axes and extreme finite coordinates. */
export function scatterViewport(points: readonly ScatterPoint[]): PlotViewport {
  const range = (values: number[]): [number, number] => {
    if (!values.length) return [-1, 1];
    const lo = Math.min(...values),
      hi = Math.max(...values);
    const span = hi - lo;
    const pad =
      span > 0 && Number.isFinite(span)
        ? span * 0.08
        : Math.max(Math.abs(lo), Math.abs(hi), 1) * 0.08;
    let a = Number.isFinite(lo - pad) ? lo - pad : lo;
    let b = Number.isFinite(hi + pad) ? hi + pad : hi;
    if (a === b) {
      a = lo === 0 ? -1 : lo > 0 ? lo * 0.9 : lo;
      b = hi === 0 ? 1 : hi < 0 ? hi * 0.9 : hi;
    }
    return [a, b];
  };
  return { x: range(points.map((point) => point.x)), y: range(points.map((point) => point.y)) };
}
