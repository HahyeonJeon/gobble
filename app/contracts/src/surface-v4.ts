import { Type } from '@sinclair/typebox';
import { closed } from './identity';
import { ScatterStateSchema } from './tabular-view';
import { SurfaceV3Schema } from './surface-v3';

/** A Project-owned opened view. The resource is its subject; view is its presentation kind. */
export const SurfaceV4Schema = Type.Union([
  SurfaceV3Schema,
  Type.Composite(
    [
      Type.Omit(SurfaceV3Schema, ['view']),
      Type.Object({ view: Type.Literal('scatter'), scatter: ScatterStateSchema }),
    ],
    closed,
  ),
]);
