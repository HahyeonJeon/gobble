import { Type, type Static } from '@sinclair/typebox';
import { closed } from './identity';
import { ObservedEvidenceSchema } from './observed-evidence';
import { RunPresentationV2Schema } from './run-presentation-v2';

/** v1 stays frozen. v2 carries the checked continuation plan in captured Run facts. */
export const ObservedEvidenceV2Schema = Type.Object(
  {
    ...ObservedEvidenceSchema.properties,
    schemaVersion: Type.Literal(2),
    source: Type.Object(
      {
        kind: Type.Literal('run'),
        value: RunPresentationV2Schema,
        observedTaskCount: Type.Integer({ minimum: 0, maximum: 1000 }),
      },
      closed,
    ),
  },
  closed,
);
export type ObservedEvidenceV2 = Static<typeof ObservedEvidenceV2Schema>;
