import { Type, type Static } from '@sinclair/typebox';
import { closed, TimestampSchema } from './identity';
import { NotebookRectangleSchema, NotebookTargetSchema, type NotebookTarget } from './notebook';
export const NotebookRepresentationSchema = Type.Union([
  Type.Object(
    { kind: Type.Literal('text'), truncated: Type.Literal(false), notebook: Type.Literal(true) },
    closed,
  ),
  Type.Object(
    {
      kind: Type.Literal('image'),
      notebook: Type.Literal(true),
      width: Type.Integer({ minimum: 1, maximum: 1536 }),
      height: Type.Integer({ minimum: 1, maximum: 1536 }),
      originalWidth: Type.Integer({ minimum: 1, maximum: 4000000 }),
      originalHeight: Type.Integer({ minimum: 1, maximum: 4000000 }),
      crop: NotebookRectangleSchema,
    },
    closed,
  ),
]);
export const NotebookTextEvidenceSchema = Type.Object(
  {
    kind: Type.Literal('text'),
    text: Type.String({ maxLength: 65536 }),
    notebook: Type.Object(
      {
        target: NotebookTargetSchema,
        capturedAt: TimestampSchema,
        rawQuote: Type.String({ maxLength: 65536 }),
      },
      closed,
    ),
  },
  closed,
);
export type NotebookRepresentation = Static<typeof NotebookRepresentationSchema>;
export function validateNotebookRepresentation(
  target: NotebookTarget,
  representation: NotebookRepresentation,
): void {
  const s = target.selection.selector;
  if (s.kind === 'text' && representation.kind === 'text') return;
  if (
    s.kind === 'image' &&
    representation.kind === 'image' &&
    representation.originalWidth * representation.originalHeight <= 4000000 &&
    s.rect.x === representation.crop.x &&
    s.rect.y === representation.crop.y &&
    s.rect.width === representation.width &&
    s.rect.height === representation.height &&
    representation.width === representation.crop.width &&
    representation.height === representation.crop.height &&
    s.rect.x + s.rect.width <= representation.originalWidth &&
    s.rect.y + s.rect.height <= representation.originalHeight
  )
    return;
  throw new Error('Notebook capture does not match its exact selection.');
}
