import { Type } from '@sinclair/typebox';
import {
  closed,
  CounterSchema,
  ResourceIdSchema,
  RevisionSchema,
  SurfaceIdSchema,
} from './identity';
import { TableFilterSchema } from './tabular-view';
const key = Type.String({ minLength: 1, maxLength: 256 });

/** One membership owner for explicitly linked views of the same source revision. */
export const ViewLinkV4Schema = Type.Object(
  {
    linkId: Type.String({ pattern: '^lnk_[A-Za-z0-9_-]{1,80}$', maxLength: 84 }),
    resource: Type.Object({ kind: Type.Literal('file'), resourceId: ResourceIdSchema }, closed),
    dataRevision: RevisionSchema,
    surfaceIds: Type.Array(SurfaceIdSchema, { minItems: 1, maxItems: 64, uniqueItems: true }),
    filterRevision: CounterSchema,
    filter: TableFilterSchema,
    rowKeys: Type.Array(key, { maxItems: 500, uniqueItems: true }),
  },
  closed,
);
