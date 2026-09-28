import { logPreviewText, type Selection, type SurfaceData } from '@gobble/contracts';
import { AppProblem } from '../problem';

export const TEXT_BYTES = 64 * 1024;
export const IMAGE_BYTES = 1024 * 1024;
export type ImageContent = Extract<
  Extract<SurfaceData, { kind: 'file' }>['value']['content'],
  { kind: 'image' }
>;
export type ImageRenderer = (
  content: ImageContent,
  selection?: Extract<Selection, { kind: 'image' }>,
) => Promise<{
  url: string;
  width: number;
  height: number;
  crop: { x: number; y: number; width: number; height: number };
}>;
export function boundedText(
  text: string,
  limit = TEXT_BYTES,
): { text: string; truncated: boolean } {
  const bytes = Buffer.from(text);
  let end = Math.min(limit, bytes.length);
  while (end > 0 && end < bytes.length && (bytes[end]! & 0xc0) === 0x80) end--;
  return { text: bytes.subarray(0, end).toString('utf8'), truncated: end < bytes.length };
}
function selectedText(text: string, selection?: Selection): string {
  if (selection?.kind !== 'text') return text;
  const lines = text.split('\n');
  const offset = (point: { line: number; column: number }) =>
    lines.slice(0, point.line - 1).reduce((sum, line) => sum + line.length + 1, 0) + point.column;
  return text.slice(offset(selection.start), offset(selection.end));
}
export function textObservation(data: SurfaceData, selection?: Selection) {
  if (data.kind === 'report')
    throw new AppProblem('unsupported', 'Report discussion is not available yet.');
  if (data.kind === 'pipeline')
    throw new AppProblem('unsupported', 'Pipeline discussion references are not available yet.');
  if (data.kind === 'file' && data.value.content.kind === 'notebook')
    throw new AppProblem('unsupported', 'Notebook content needs an exact captured selection.');
  if (data.kind === 'file' && data.value.content.kind === 'image')
    throw new AppProblem('unsupported', 'Image observation requires an image-capable model.');
  if (data.kind === 'file' && data.value.content.kind === 'table') {
    const content = data.value.content;
    const selected = selection?.kind === 'table' ? selection : undefined;
    const indices = content.columns.flatMap((column, index) =>
      !selected || selected.columns.includes(column.id) ? [index] : [],
    );
    const columns = indices.map((index) => content.columns[index]!);
    const sourceRows = content.rows.filter(
      (row) => !selected || selected.rowKeys.includes(row.key),
    );
    const rows: { key: string; cells: string[] }[] = [];
    let bytes = Buffer.byteLength(JSON.stringify(columns)) + 512;
    if (bytes > TEXT_BYTES)
      throw new AppProblem(
        'unsupported',
        'The table headers exceed the observation limit. Select fewer columns.',
      );
    for (const row of sourceRows) {
      const projected = { key: row.key, cells: indices.map((index) => row.cells[index] ?? '') };
      bytes += Buffer.byteLength(JSON.stringify(projected)) + 1;
      if (bytes > TEXT_BYTES) break;
      rows.push(projected);
    }
    return {
      kind: 'table' as const,
      columns,
      rows,
      availableRows: sourceRows.length,
      truncated: content.truncated || rows.length < sourceRows.length,
    };
  }
  const source =
    data.kind === 'log'
      ? logPreviewText(data.value)
      : data.kind === 'run'
        ? JSON.stringify(data.value)
        : data.value.content.kind === 'text'
          ? data.value.content.text
          : '';
  const selected = selectedText(source, selection);
  return {
    kind: data.kind === 'run' ? ('run' as const) : ('text' as const),
    ...boundedText(selected),
    availableUtf16: selected.length,
    selection: selection ?? null,
  };
}
