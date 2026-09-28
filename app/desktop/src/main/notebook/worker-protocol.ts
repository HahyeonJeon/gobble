import { Type } from '@sinclair/typebox';
import { Value } from '@sinclair/typebox/value';
import { LIMITS, NotebookProblem, PROFILE, type Snapshot } from './model';
const c = { additionalProperties: false },
  str = Type.String({ maxLength: LIMITS.bytes }),
  num = Type.Integer({ minimum: 0, maximum: Number.MAX_SAFE_INTEGER });
const text = Type.Object({ kind: Type.Literal('text'), text: str, raw: str, digest: str }, c);
const image = Type.Object(
  {
    kind: Type.Literal('image'),
    mime: Type.Union([Type.Literal('image/png'), Type.Literal('image/jpeg')]),
    base64: str,
    width: num,
    height: num,
    digest: str,
  },
  c,
);
const output = Type.Object(
  {
    index: num,
    type: str,
    mime: str,
    alternatives: Type.Array(str, { maxItems: LIMITS.nodes }),
    notice: str,
    part: Type.Union([
      text,
      image,
      Type.Object({ kind: Type.Literal('unavailable'), reason: str }, c),
    ]),
  },
  c,
);
const cell = Type.Object(
  {
    index: num,
    address: Type.Union([
      Type.Object({ kind: Type.Literal('id'), id: str }, c),
      Type.Object({ kind: Type.Literal('ordinal'), index: num }, c),
    ]),
    type: Type.Union(['code', 'markdown', 'raw', 'unknown'].map((t) => Type.Literal(t))),
    source: text,
    outputs: Type.Array(output, { maxItems: LIMITS.outputsPerCell }),
    notice: str,
  },
  c,
);
const snapshot = Type.Object(
  {
    profile: Type.Literal(PROFILE),
    revision: str,
    bytes: num,
    minor: num,
    language: str,
    cells: Type.Array(cell, { maxItems: LIMITS.cells }),
    textBytes: num,
  },
  c,
);
const reply = Type.Union([
  Type.Object({ ok: Type.Literal(true), value: snapshot }, c),
  Type.Object(
    {
      ok: Type.Literal(false),
      code: Type.Union(
        ['invalid', 'unsupported', 'limit', 'stale', 'cancelled', 'timeout', 'busy'].map((t) =>
          Type.Literal(t),
        ),
      ),
      message: str,
    },
    c,
  ),
]);
export function parseWorkerReply(
  value: unknown,
): { ok: true; value: Snapshot } | { ok: false; code: NotebookProblem['code']; message: string } {
  if (!Value.Check(reply, value))
    throw new NotebookProblem('invalid', 'Notebook worker returned an invalid result.');
  // The schema above checks every data field from the dedicated, bundled worker.
  return value as ReturnType<typeof parseWorkerReply>;
}
