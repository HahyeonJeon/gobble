import type { ToolInputSchema } from '@gobble/contracts';
import type { JsonValue } from './generated';

/** Codex 0.153.4 accepts homogeneous arrays, not Draft 7 tuple-form `items`.
 * Preserve the exact fixed length. Never widen heterogeneous or variadic tuples.
 * This is a provider adapter; published contracts and Main validation stay unchanged.
 */
export function codexToolSchema(schema: ToolInputSchema): JsonValue {
  function convert(value: JsonValue): JsonValue {
    if (Array.isArray(value)) return value.map(convert);
    if (!value || typeof value !== 'object') return value;
    const result = Object.fromEntries(
      Object.entries(value).map(([key, entry]) => [key, convert(entry!)]),
    );
    if (Array.isArray(result.items)) {
      const items = result.items;
      if (
        !items.length ||
        result.minItems !== items.length ||
        result.maxItems !== items.length ||
        result.additionalItems !== false ||
        items.some((item) => JSON.stringify(item) !== JSON.stringify(items[0]))
      )
        throw new Error('Codex tool schemas require fixed homogeneous tuples.');
      result.items = items[0]!;
      delete result.additionalItems;
    }
    return result;
  }
  return convert(JSON.parse(JSON.stringify(schema)) as JsonValue);
}
