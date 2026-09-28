import { Type } from '@sinclair/typebox';
import { Value } from '@sinclair/typebox/value';
import {
  NotebookProblem,
  PROFILE,
  type Target,
  type PartAddress,
  type Snapshot,
  type TextPart,
  type ImagePart,
  type Rect,
} from './model';
import { rawRange } from './text';
const closed = { additionalProperties: false };
const integer = Type.Integer({ minimum: 0, maximum: Number.MAX_SAFE_INTEGER });
const hash = Type.String({ pattern: '^sha256:[0-9a-f]{64}$' });
const cell = Type.Union([
  Type.Object(
    { kind: Type.Literal('id'), id: Type.String({ pattern: '^[a-zA-Z0-9_-]{1,64}$' }) },
    closed,
  ),
  Type.Object({ kind: Type.Literal('ordinal'), index: integer }, closed),
]);
const part = Type.Union([
  Type.Object({ kind: Type.Literal('source') }, closed),
  Type.Object(
    {
      kind: Type.Literal('output'),
      index: integer,
      mime: Type.String({ maxLength: 100 }),
      digest: hash,
    },
    closed,
  ),
]);
const rectangle = Type.Object(
  {
    x: integer,
    y: integer,
    width: Type.Integer({ minimum: 1 }),
    height: Type.Integer({ minimum: 1 }),
  },
  closed,
);
const selector = Type.Union([
  Type.Object({ kind: Type.Literal('text'), start: integer, end: integer }, closed),
  Type.Object({ kind: Type.Literal('image'), rect: rectangle }, closed),
]);
const targetSchema = Type.Object(
  { revision: hash, profile: Type.Literal(PROFILE), cell, part, selector },
  closed,
);
export function parseTarget(value: unknown): Target {
  if (!Value.Check(targetSchema, value))
    throw new NotebookProblem('invalid', 'Invalid exact Notebook target.');
  return value;
}

export function resolvePart(snapshot: Snapshot, address: PartAddress): TextPart | ImagePart {
  const id = address.cell;
  const c =
    id.kind === 'id'
      ? snapshot.cells.find((c) => c.address.kind === 'id' && c.address.id === id.id)
      : snapshot.cells[id.index];
  if (
    !c ||
    (id.kind === 'ordinal'
      ? c.address.kind !== 'ordinal' || c.address.index !== id.index
      : c.address.kind !== 'id' || c.address.id !== id.id)
  )
    throw new NotebookProblem('invalid', 'Cell address is unavailable.');
  if (address.part.kind === 'source') return c.source;
  const o = c.outputs[address.part.index];
  if (
    !o ||
    o.part.kind === 'unavailable' ||
    o.mime !== address.part.mime ||
    o.part.digest !== address.part.digest
  )
    throw new NotebookProblem('stale', 'Saved output representation changed or is unavailable.');
  return o.part;
}
export function contains(a: Rect, b: Rect): boolean {
  return (
    b.x >= a.x && b.y >= a.y && b.x + b.width <= a.x + a.width && b.y + b.height <= a.y + a.height
  );
}
export function resolveTarget(
  snapshot: Snapshot,
  input: unknown,
): { target: Target; part: TextPart | ImagePart; rawQuote: string } {
  const target = parseTarget(input);
  if (target.revision !== snapshot.revision || target.profile !== snapshot.profile)
    throw new NotebookProblem('stale', 'The Notebook source version changed.');
  const part = resolvePart(snapshot, target),
    s = target.selector;
  if (s.kind === 'text' && part.kind === 'text')
    return { target, part, rawQuote: rawRange(part, s.start, s.end) };
  if (
    s.kind === 'image' &&
    part.kind === 'image' &&
    contains({ x: 0, y: 0, width: part.width, height: part.height }, s.rect)
  )
    return { target, part, rawQuote: '' };
  throw new NotebookProblem('invalid', 'Selection does not match the saved representation bounds.');
}
