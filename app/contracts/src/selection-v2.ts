// Frozen selectors for reference v2 and workspace v3–v6.
import { Type, type Static } from '@sinclair/typebox';
import { closed } from './identity';

/** Text: 1-based lines, 0-based UTF-16 columns, half-open range.
 * Image: normalized original-image coordinates, top-left origin, independent of viewport.
 * Table: row/column keys are exact within the resource revision, never biological IDs. */
const key = Type.String({ minLength: 1, maxLength: 256 });
const coordinate = Type.Number({ minimum: 0, maximum: 1 });
const dimension = Type.Integer({ minimum: 1, maximum: 100_000 });
const position = Type.Object(
  {
    line: Type.Integer({ minimum: 1, maximum: Number.MAX_SAFE_INTEGER }),
    column: Type.Integer({ minimum: 0, maximum: Number.MAX_SAFE_INTEGER }),
  },
  closed,
);

export const SelectionV2Schema = Type.Union([
  Type.Object(
    {
      kind: Type.Literal('table'),
      coordinateSpace: Type.Literal('revision-row-column-keys'),
      rowKeys: Type.Array(key, { minItems: 1, maxItems: 500, uniqueItems: true }),
      columns: Type.Array(key, { minItems: 1, maxItems: 100, uniqueItems: true }),
    },
    closed,
  ),
  Type.Object(
    {
      kind: Type.Literal('text'),
      coordinateSpace: Type.Literal('utf16-line-column'),
      start: position,
      end: position,
    },
    closed,
  ),
  Type.Object(
    {
      kind: Type.Literal('image'),
      coordinateSpace: Type.Literal('normalized-original-image'),
      x: coordinate,
      y: coordinate,
      width: Type.Number({ exclusiveMinimum: 0, maximum: 1 }),
      height: Type.Number({ exclusiveMinimum: 0, maximum: 1 }),
      originalWidth: dimension,
      originalHeight: dimension,
      contentHash: Type.String({ pattern: '^sha256:[a-f0-9]{64}$' }),
    },
    closed,
  ),
]);

export type SelectionV2 = Static<typeof SelectionV2Schema>;
