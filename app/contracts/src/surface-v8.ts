import { Type } from '@sinclair/typebox';
import { closed, CounterSchema } from './identity';
import { ScatterStateSchema, TableStateSchema } from './tabular-view';
import { SurfaceV3Schema } from './surface-v3';
import { RunFilterSchema, LogViewStateSchema } from './run-navigation';
const RunViewStateSchema = Type.Object(
  { ...RunFilterSchema.properties, viewRevision: CounterSchema },
  closed,
);
/** A Project-owned opened view. The resource is its subject; view is its presentation kind. */
export const SurfaceV8Schema = Type.Union([
  Type.Composite(
    [
      Type.Omit(SurfaceV3Schema, ['view']),
      Type.Object({
        view: Type.Union([Type.Literal('text'), Type.Literal('image')]),
      }),
    ],
    closed,
  ),
  Type.Composite(
    [
      Type.Omit(SurfaceV3Schema, ['view']),
      Type.Object({ view: Type.Literal('run'), runView: Type.Optional(RunViewStateSchema) }),
    ],
    closed,
  ),
  Type.Composite(
    [
      Type.Omit(SurfaceV3Schema, ['view']),
      Type.Object({ view: Type.Literal('log'), logView: Type.Optional(LogViewStateSchema) }),
    ],
    closed,
  ),
  Type.Composite(
    [
      Type.Omit(SurfaceV3Schema, ['view']),
      Type.Object({ view: Type.Literal('table'), table: Type.Optional(TableStateSchema) }),
    ],
    closed,
  ),
  Type.Composite(
    [
      Type.Omit(SurfaceV3Schema, ['view']),
      Type.Object({ view: Type.Literal('scatter'), scatter: ScatterStateSchema }),
    ],
    closed,
  ),
]);
