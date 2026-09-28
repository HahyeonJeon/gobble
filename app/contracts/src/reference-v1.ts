// Storage-only v1 reference shapes. Do not evolve these with the live contract.
import { Type, type Static } from '@sinclair/typebox';
import { closed, ProjectIdSchema, RevisionSchema, SurfaceIdSchema } from './identity';
import { DiscussionResourceRefSchema as ResourceRefSchema } from './resource';

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

export const SelectionV1Schema = Type.Union([
  Type.Object(
    {
      kind: Type.Literal('table'),
      rowKeys: Type.Array(key, { minItems: 1, maxItems: 500, uniqueItems: true }),
      columns: Type.Array(key, { minItems: 1, maxItems: 100, uniqueItems: true }),
    },
    closed,
  ),
  Type.Object({ kind: Type.Literal('text'), start: position, end: position }, closed),
  Type.Object(
    {
      kind: Type.Literal('image'),
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

export const EvidenceRefV1Schema = Type.Object(
  {
    projectId: ProjectIdSchema,
    surfaceId: SurfaceIdSchema,
    resource: ResourceRefSchema,
    dataRevision: RevisionSchema,
    selection: Type.Optional(SelectionV1Schema),
  },
  closed,
);

export type EvidenceRefV1 = Static<typeof EvidenceRefV1Schema>;
