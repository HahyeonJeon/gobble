import { PipelineSelectionSchema } from './pipeline-reference';
import { NotebookSelectionSchema } from './notebook';
import { PdfSelectionSchema } from './pdf';
import { Type, type Static } from '@sinclair/typebox';
import { closed } from './identity';
import { DependencySelectionSchema } from './run-dependencies';
import { SelectionV2Schema } from './selection-v2';

/** Text: 1-based lines, 0-based UTF-16 columns, half-open range.
 * Image: normalized original-image coordinates, top-left origin, independent of viewport.
 * Table: row/column keys are exact within the resource revision, never biological IDs. */
const key = Type.String({ minLength: 1, maxLength: 256 });
const position = Type.Object(
  {
    line: Type.Integer({ minimum: 1, maximum: Number.MAX_SAFE_INTEGER }),
    column: Type.Integer({ minimum: 0, maximum: Number.MAX_SAFE_INTEGER }),
  },
  closed,
);

export const ObservedSelectionSchema = Type.Union([
  Type.Object(
    {
      kind: Type.Literal('run-task'),
      coordinateSpace: Type.Literal('observed-instance-attempt'),
      instanceId: key,
      attempt: Type.Integer({ minimum: 0, maximum: Number.MAX_SAFE_INTEGER }),
    },
    closed,
  ),
  Type.Object(
    {
      kind: Type.Literal('log-text'),
      coordinateSpace: Type.Literal('decoded-preview-utf16-line-column'),
      stream: Type.Union([Type.Literal('stdout'), Type.Literal('stderr')]),
      start: position,
      end: position,
    },
    closed,
  ),
]);
export const SelectionV3Schema = Type.Union([
  ...SelectionV2Schema.anyOf,
  ...ObservedSelectionSchema.anyOf,
]);

export const SelectionV4Schema = Type.Union([
  ...SelectionV3Schema.anyOf,
  ...DependencySelectionSchema.anyOf,
]);
export const SelectionV5Schema = Type.Union([PdfSelectionSchema, ...SelectionV4Schema.anyOf]);
export const SelectionV6Schema = Type.Union([NotebookSelectionSchema, ...SelectionV5Schema.anyOf]);
export const SelectionSchema = Type.Union([PipelineSelectionSchema, ...SelectionV6Schema.anyOf]);
export type Selection = Static<typeof SelectionSchema>;
