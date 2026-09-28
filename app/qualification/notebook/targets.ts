import { Type } from '@sinclair/typebox';
import { Value } from '@sinclair/typebox/value';
import {
  LIMITS,
  NotebookProblem,
  PROFILE,
  type Target,
  type PartAddress,
  type Snapshot,
  type TextPart,
  type ImagePart,
  type Rect,
  type ViewReceipt,
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
const receiptSchema = Type.Object(
  {
    revision: hash,
    generation: integer,
    parts: Type.Array(
      Type.Object(
        { cell, part, selectors: Type.Array(selector, { maxItems: LIMITS.visibleUnits }) },
        closed,
      ),
      { maxItems: LIMITS.visibleParts },
    ),
  },
  closed,
);
export function parseTarget(value: unknown): Target {
  if (!Value.Check(targetSchema, value))
    throw new NotebookProblem('invalid', 'Invalid exact Notebook target.');
  return value;
}
export function parseReceipt(value: unknown): ViewReceipt {
  if (!Value.Check(receiptSchema, value))
    throw new NotebookProblem('invalid', 'Invalid visible receipt.');
  return value;
}
export function addressKey(value: PartAddress): string {
  const c = value.cell,
    p = value.part;
  return JSON.stringify([
    c.kind,
    c.kind === 'id' ? c.id : c.index,
    p.kind,
    ...(p.kind === 'output' ? [p.index, p.mime, p.digest] : []),
  ]);
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
export function validateReceipt(snapshot: Snapshot, input: unknown): ViewReceipt {
  const receipt = parseReceipt(input);
  let units = 0;
  if (receipt.revision !== snapshot.revision)
    throw new NotebookProblem('stale', 'Visible source changed.');
  for (const p of receipt.parts)
    for (const s of p.selectors) {
      resolveTarget(snapshot, {
        cell: p.cell,
        part: p.part,
        selector: s,
        profile: PROFILE,
        revision: snapshot.revision,
      });
      units += s.kind === 'text' ? s.end - s.start : 1;
    }
  if (units > LIMITS.visibleUnits)
    throw new NotebookProblem('limit', 'Visible scope exceeds its bounded profile.');
  return receipt;
}
export function assertObserved(receipt: ViewReceipt, target: Target): void {
  if (receipt.revision !== target.revision)
    throw new NotebookProblem('stale', 'Observation is from another source.');
  const p = receipt.parts.find((p) => addressKey(p) === addressKey(target)),
    s = target.selector;
  if (
    !p?.selectors.some((v) =>
      v.kind === 'text' && s.kind === 'text'
        ? s.start >= v.start && s.end <= v.end
        : v.kind === 'image' && s.kind === 'image' && contains(v.rect, s.rect),
    )
  )
    throw new NotebookProblem('stale', 'Target was not fully visible in this observation.');
}
