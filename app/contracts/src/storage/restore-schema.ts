import { Kind, type SchemaOptions } from '@sinclair/typebox';

/** Restore TypeBox's non-JSON kind tags on trusted, frozen Draft 7 storage schemas.
 * This interprets no user schema, generates no code, and preserves every historical constraint. */
export function restoreSchema(input: Record<string, unknown>): SchemaOptions {
  const result: SchemaOptions & { [Kind]?: string } = {};
  for (const [key, value] of Object.entries(input)) {
    result[key] = Array.isArray(value)
      ? value.map((item) =>
          item !== null && typeof item === 'object' ? restoreSchema(item) : item,
        )
      : value !== null && typeof value === 'object'
        ? restoreSchema(value as Record<string, unknown>)
        : value;
  }
  const kinds: Record<string, string> = {
    object: 'Object',
    array: Array.isArray(input.items) ? 'Tuple' : 'Array',
    string: 'String',
    number: 'Number',
    integer: 'Integer',
    boolean: 'Boolean',
    null: 'Null',
  };
  const kind = typeof input.type === 'string' ? kinds[input.type] : undefined;
  if ('const' in input) result[Kind] = 'Literal';
  else if (Array.isArray(input.anyOf)) result[Kind] = 'Union';
  else if (Array.isArray(input.allOf)) result[Kind] = 'Intersect';
  else if (kind) result[Kind] = kind;
  else if ('not' in input) result[Kind] = 'Never';
  return result;
}
