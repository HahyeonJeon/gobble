import { Type, type Static, type TSchema } from '@sinclair/typebox';
import { closed } from './identity';

export const ContractErrorSchema = Type.Object(
  {
    code: Type.Union([
      Type.Literal('invalid_request'),
      Type.Literal('forbidden'),
      Type.Literal('not_found'),
      Type.Literal('outside_project'),
      Type.Literal('runtime_unavailable'),
      Type.Literal('incompatible_runtime'),
      Type.Literal('stale_revision'),
      Type.Literal('unsupported'),
      Type.Literal('request_conflict'),
      Type.Literal('internal'),
    ]),
    message: Type.String({ minLength: 1, maxLength: 500 }),
    retry: Type.Union([
      Type.Literal('never'),
      Type.Literal('after_refresh'),
      Type.Literal('after_reconnect'),
    ]),
  },
  closed,
);

export type ContractError = Static<typeof ContractErrorSchema>;

export function serviceResult<T extends TSchema>(schema: T) {
  return Type.Union([
    Type.Object({ schemaVersion: Type.Literal(1), ok: Type.Literal(true), value: schema }, closed),
    Type.Object(
      { schemaVersion: Type.Literal(1), ok: Type.Literal(false), error: ContractErrorSchema },
      closed,
    ),
  ]);
}
