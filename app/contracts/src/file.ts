import { NotebookSourceSchema } from './notebook';
import { PdfSourceSchema } from './pdf';
import { Type, type Static } from '@sinclair/typebox';
import { closed, ProjectIdSchema, ResourceIdSchema, RevisionSchema } from './identity';

export const DirectorySchema = Type.Object(
  {
    projectId: ProjectIdSchema,
    directoryId: ResourceIdSchema,
    entries: Type.Array(
      Type.Object(
        {
          resourceId: ResourceIdSchema,
          name: Type.String({ maxLength: 512 }),
          kind: Type.Union([
            Type.Literal('directory'),
            Type.Literal('file'),
            Type.Literal('unsupported'),
          ]),
          size: Type.Integer({ minimum: 0, maximum: Number.MAX_SAFE_INTEGER }),
        },
        closed,
      ),
      { maxItems: 500 },
    ),
    truncated: Type.Boolean(),
  },
  closed,
);
export const FilePreviewContentSchema = Type.Union([
  Type.Object(
    { kind: Type.Literal('text'), text: Type.String({ maxLength: 1024 * 1024 }) },
    closed,
  ),
  Type.Object(
    {
      kind: Type.Literal('table'),
      columns: Type.Array(Type.Object({ id: Type.String(), name: Type.String() }, closed), {
        maxItems: 100,
      }),
      rows: Type.Array(
        Type.Object(
          { key: Type.String(), cells: Type.Array(Type.String(), { maxItems: 100 }) },
          closed,
        ),
        { maxItems: 500 },
      ),
      truncated: Type.Boolean(),
    },
    closed,
  ),
  Type.Object(
    {
      kind: Type.Literal('image'),
      mediaType: Type.Union([Type.Literal('image/png'), Type.Literal('image/jpeg')]),
      width: Type.Integer({ minimum: 1, maximum: 20_000_000 }),
      height: Type.Integer({ minimum: 1, maximum: 20_000_000 }),
      base64: Type.String({ maxLength: 11_184_812 }),
    },
    closed,
  ),
]);

export const FileContentSchema = Type.Object(
  {
    projectId: ProjectIdSchema,
    resourceId: ResourceIdSchema,
    name: Type.String({ maxLength: 512 }),
    revision: RevisionSchema,
    size: Type.Integer({ minimum: 0, maximum: 8 * 1024 * 1024 }),
    content: Type.Union([NotebookSourceSchema, PdfSourceSchema, ...FilePreviewContentSchema.anyOf]),
  },
  closed,
);
export type FileContent = Static<typeof FileContentSchema>;
