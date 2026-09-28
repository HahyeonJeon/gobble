import { Type, type Static } from '@sinclair/typebox';
import {
  closed,
  ProjectIdSchema,
  ResourceIdSchema,
  RevisionSchema,
  SurfaceIdSchema,
} from './identity';

export const NOTEBOOK_PROFILE = 'notebook-passive-1';
const index = Type.Integer({ minimum: 0, maximum: 999 });
const unit = Type.Integer({ minimum: 0, maximum: 1048576 });
const dimension = Type.Integer({ minimum: 1, maximum: 4000000 });
export const NotebookCellAddressSchema = Type.Union([
  Type.Object(
    { kind: Type.Literal('id'), id: Type.String({ pattern: '^[a-zA-Z0-9_-]{1,64}$' }) },
    closed,
  ),
  Type.Object({ kind: Type.Literal('ordinal'), index }, closed),
]);
export const NotebookPartAddressSchema = Type.Union([
  Type.Object({ kind: Type.Literal('source') }, closed),
  Type.Object(
    {
      kind: Type.Literal('output'),
      index,
      mime: Type.String({ minLength: 1, maxLength: 100 }),
      digest: RevisionSchema,
    },
    closed,
  ),
]);
export const NotebookRectangleSchema = Type.Object(
  {
    x: Type.Integer({ minimum: 0, maximum: 4000000 }),
    y: Type.Integer({ minimum: 0, maximum: 4000000 }),
    width: dimension,
    height: dimension,
  },
  closed,
);
export const NotebookSelectorSchema = Type.Union([
  Type.Object(
    {
      kind: Type.Literal('text'),
      coordinateSpace: Type.Literal('notebook-display-utf16'),
      start: unit,
      end: unit,
    },
    closed,
  ),
  Type.Object(
    {
      kind: Type.Literal('image'),
      coordinateSpace: Type.Literal('natural-image-pixels'),
      rect: NotebookRectangleSchema,
    },
    closed,
  ),
]);
export const NotebookSelectionSchema = Type.Object(
  {
    kind: Type.Literal('notebook'),
    profile: Type.Literal(NOTEBOOK_PROFILE),
    cell: NotebookCellAddressSchema,
    part: NotebookPartAddressSchema,
    selector: NotebookSelectorSchema,
  },
  closed,
);
export const NotebookTargetSchema = Type.Object(
  {
    schemaVersion: Type.Literal(6),
    projectId: ProjectIdSchema,
    resource: Type.Object({ kind: Type.Literal('file'), resourceId: ResourceIdSchema }, closed),
    dataRevision: RevisionSchema,
    origin: Type.Optional(Type.Object({ surfaceId: SurfaceIdSchema }, closed)),
    selection: NotebookSelectionSchema,
  },
  closed,
);
export const NotebookSourceSchema = Type.Object(
  {
    kind: Type.Literal('notebook'),
    base64: Type.String({ maxLength: 11184812 }),
  },
  closed,
);
const text = Type.Object(
  { kind: Type.Literal('text'), text: Type.String({ maxLength: 1048576 }), digest: RevisionSchema },
  closed,
);
const image = Type.Object(
  {
    kind: Type.Literal('image'),
    mime: Type.Union([Type.Literal('image/png'), Type.Literal('image/jpeg')]),
    width: dimension,
    height: dimension,
    digest: RevisionSchema,
  },
  closed,
);
const output = Type.Object(
  {
    index,
    type: Type.String({ maxLength: 8388608 }),
    mime: Type.String({ maxLength: 100 }),
    alternatives: Type.Array(Type.String({ maxLength: 8388608 }), { maxItems: 100000 }),
    notice: Type.String({ maxLength: 1000 }),
    part: Type.Union([
      text,
      image,
      Type.Object(
        { kind: Type.Literal('unavailable'), reason: Type.String({ maxLength: 1000 }) },
        closed,
      ),
    ]),
  },
  closed,
);
export const NotebookDocumentSchema = Type.Object(
  {
    profile: Type.Literal(NOTEBOOK_PROFILE),
    revision: RevisionSchema,
    bytes: Type.Integer({ minimum: 0, maximum: 8388608 }),
    minor: Type.Integer({ minimum: 0, maximum: 5 }),
    language: Type.String({ maxLength: 80 }),
    textBytes: unit,
    cells: Type.Array(
      Type.Object(
        {
          index,
          address: NotebookCellAddressSchema,
          type: Type.Union([
            Type.Literal('code'),
            Type.Literal('markdown'),
            Type.Literal('raw'),
            Type.Literal('unknown'),
          ]),
          source: text,
          outputs: Type.Array(output, { maxItems: 100 }),
          notice: Type.String({ maxLength: 1000 }),
        },
        closed,
      ),
      { maxItems: 1000 },
    ),
  },
  closed,
);
export const NotebookReferenceExcerptSchema = Type.Object(
  {
    target: NotebookTargetSchema,
    content: Type.Union([
      Type.Object({ kind: Type.Literal('text'), text: Type.String({ maxLength: 65536 }) }, closed),
      Type.Object(
        {
          kind: Type.Literal('image'),
          base64: Type.String({ minLength: 1, maxLength: 1048576 }),
          width: Type.Integer({ minimum: 1, maximum: 1536 }),
          height: Type.Integer({ minimum: 1, maximum: 1536 }),
        },
        closed,
      ),
    ]),
  },
  closed,
);
export type NotebookReferenceExcerpt = Static<typeof NotebookReferenceExcerptSchema>;
export const NotebookContentSchema = Type.Object(
  {
    kind: Type.Literal('notebook'),
    document: NotebookDocumentSchema,
    reference: Type.Optional(NotebookReferenceExcerptSchema),
  },
  closed,
);
export const NotebookNavigationSchema = Type.Object(
  { page: Type.Integer({ minimum: 0, maximum: 49 }) },
  closed,
);
export const NotebookImageSchema = Type.Object(
  {
    base64: Type.String({ minLength: 1, maxLength: 11184812 }),
    width: dimension,
    height: dimension,
  },
  closed,
);
export type NotebookSelection = Static<typeof NotebookSelectionSchema>;
export type NotebookTarget = Static<typeof NotebookTargetSchema>;
export type NotebookDocument = Static<typeof NotebookDocumentSchema>;
export type NotebookContent = Static<typeof NotebookContentSchema>;
export type NotebookImage = Static<typeof NotebookImageSchema>;

export function sameNotebookTarget(a: NotebookTarget, b: NotebookTarget): boolean {
  const key = (t: NotebookTarget) => {
    const s = t.selection,
      c = s.cell,
      p = s.part,
      v = s.selector;
    return JSON.stringify([
      t.projectId,
      t.resource.resourceId,
      t.dataRevision,
      t.origin?.surfaceId,
      s.profile,
      c.kind,
      c.kind === 'id' ? c.id : c.index,
      p.kind,
      ...(p.kind === 'output' ? [p.index, p.mime, p.digest] : []),
      v.kind,
      v.coordinateSpace,
      ...(v.kind === 'text' ? [v.start, v.end] : [v.rect.x, v.rect.y, v.rect.width, v.rect.height]),
    ]);
  };
  return key(a) === key(b);
}

/** Shape/order checks independent of the loaded document; all ranges are half-open. */
export function validateNotebookSelection(selection: NotebookSelection): void {
  const s = selection.selector;
  if (s.kind === 'text' && s.start >= s.end)
    throw new Error('Select a nonempty Notebook text range.');
  if (s.kind === 'image' && selection.part.kind !== 'output')
    throw new Error('A saved image output is required.');
}
/** Resolve only this exact saved representation. No quote search or identity repair. */
export function notebookPart(document: NotebookDocument, selection: NotebookSelection) {
  validateNotebookSelection(selection);
  if (selection.profile !== document.profile) throw new Error('Notebook reading profile changed.');
  const address = selection.cell;
  const cell =
    address.kind === 'id'
      ? document.cells.find((c) => c.address.kind === 'id' && c.address.id === address.id)
      : document.cells[address.index];
  if (
    !cell ||
    (address.kind === 'ordinal' &&
      (cell.address.kind !== 'ordinal' || cell.address.index !== address.index))
  )
    throw new Error('Notebook cell is unavailable.');
  const output = selection.part.kind === 'output' ? cell.outputs[selection.part.index] : undefined;
  const part = selection.part.kind === 'source' ? cell.source : output?.part;
  if (
    !part ||
    part.kind === 'unavailable' ||
    (selection.part.kind === 'output' &&
      (output?.mime !== selection.part.mime || part.digest !== selection.part.digest))
  )
    throw new Error('Notebook output representation changed.');
  const s = selection.selector;
  if (s.kind === 'text' && part.kind === 'text') {
    if (s.end > part.text.length) throw new Error('Notebook text range is outside this part.');
    for (const at of [s.start, s.end]) {
      if (
        /[\uDC00-\uDFFF]/.test(part.text[at] ?? '') &&
        /[\uD800-\uDBFF]/.test(part.text[at - 1] ?? '')
      )
        throw new Error('Notebook range splits a Unicode character.');
    }
    return part;
  }
  if (
    s.kind === 'image' &&
    part.kind === 'image' &&
    s.rect.x + s.rect.width <= part.width &&
    s.rect.y + s.rect.height <= part.height
  )
    return part;
  throw new Error('Notebook selection does not match this part.');
}
