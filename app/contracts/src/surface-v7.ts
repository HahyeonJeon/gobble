import { Type } from '@sinclair/typebox';
import { closed } from './identity';
import { ScatterStateSchema, TableStateSchema } from './tabular-view';
import { SurfaceV3Schema } from './surface-v3';

/** A Project-owned opened view. The resource is its subject; view is its presentation kind. */
export const SurfaceV7Schema = Type.Union([
  Type.Composite(
    [
      Type.Omit(SurfaceV3Schema, ['view']),
      Type.Object({
        view: Type.Union([
          Type.Literal('text'),
          Type.Literal('image'),
          Type.Literal('run'),
          Type.Literal('log'),
        ]),
      }),
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
