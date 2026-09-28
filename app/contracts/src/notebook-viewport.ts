import { Type, type Static } from '@sinclair/typebox';
import {
  NotebookSelectionSchema,
  notebookPart,
  type NotebookSelection,
  type NotebookDocument,
} from './notebook';
import { closed } from './identity';
/** Ephemeral rendered ranges, not a document snapshot or a durable Workspace setting. */
export const NotebookViewportSchema = Type.Object(
  {
    parts: Type.Array(NotebookSelectionSchema, { maxItems: 64 }),
  },
  closed,
);
export type NotebookViewport = Static<typeof NotebookViewportSchema>;
export function notebookSelectionContains(
  outer: NotebookSelection,
  inner: NotebookSelection,
): boolean {
  const a = outer.cell,
    b = inner.cell,
    p = outer.part,
    q = inner.part;
  if (
    outer.profile !== inner.profile ||
    a.kind !== b.kind ||
    (a.kind === 'id' && b.kind === 'id'
      ? a.id !== b.id
      : a.kind === 'ordinal' && b.kind === 'ordinal' && a.index !== b.index) ||
    p.kind !== q.kind ||
    (p.kind === 'output' &&
      q.kind === 'output' &&
      (p.index !== q.index || p.mime !== q.mime || p.digest !== q.digest))
  )
    return false;
  const x = outer.selector,
    y = inner.selector;
  if (x.kind === 'text' && y.kind === 'text')
    return y.start >= x.start && y.end <= x.end && y.start < y.end;
  if (x.kind === 'image' && y.kind === 'image')
    return (
      y.rect.x >= x.rect.x &&
      y.rect.y >= x.rect.y &&
      y.rect.x + y.rect.width <= x.rect.x + x.rect.width &&
      y.rect.y + y.rect.height <= x.rect.y + x.rect.height
    );
  return false;
}
export function validateNotebookViewport(
  document: NotebookDocument,
  viewport: NotebookViewport,
): void {
  if (viewport.parts.length > 64) throw new Error('Notebook viewport exceeds its part limit.');
  let units = 0;
  for (const selection of viewport.parts) {
    notebookPart(document, selection);
    if (selection.selector.kind === 'text')
      units += selection.selector.end - selection.selector.start;
  }
  if (units > 4096) throw new Error('Notebook viewport exceeds its text limit.');
}
