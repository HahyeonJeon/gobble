import { textWindow } from './text-window';
import { describe, expect, it } from 'vitest';
import { parseNotebook } from './parser';
import { code, encode, notebook, fixture, png } from './fixtures';
import { LIMITS, PROFILE, type Target } from './model';
import { resolveTarget, validateReceipt, assertObserved } from './targets';
import { projectText, rawRange } from './text';
import { imageSource } from './image-source';
const target = (revision: string, start = 0, end = 4): Target => ({
  revision,
  profile: PROFILE,
  cell: { kind: 'id', id: 'filter' },
  part: { kind: 'source' },
  selector: { kind: 'text', start, end },
});
describe('Notebook format and passive projection', () => {
  for (let minor = 0; minor <= 5; minor++)
    it('reads format 4.' + minor + ' without changing source bytes', () => {
      const bytes = encode(notebook([code()], minor)),
        before = Buffer.from(bytes);
      expect(parseNotebook(bytes).minor).toBe(minor);
      expect(bytes.equals(before)).toBe(true);
    });
  it('preserves UTF-16 and CRLF source, normalizes only the display projection', () => {
    const s = parseNotebook(fixture('python'));
    expect(s.cells[1]?.source.raw).toContain('🧬 and CRLF\r\n');
    expect(s.cells[1]?.source.text).not.toContain('\r');
    expect(projectText('a\r\nb').text).toBe('a\nb');
    expect(rawRange(projectText('a\r\nb'), 0, 3)).toBe('a\r\nb');
  });
  it('blocks required, duplicate and invalid IDs without repairing them', () => {
    for (const cells of [
      [{ ...code(), id: undefined }],
      [code('same'), code('same')],
      [code('bad id')],
    ])
      expect(() => parseNotebook(encode(notebook(cells)))).toThrow(/IDs/);
  });
  it('uses explicit ordinal identity only for missing legacy IDs', () => {
    const s = parseNotebook(fixture('legacy'));
    expect(s.cells[0]?.address).toEqual({ kind: 'ordinal', index: 0 });
  });
  it('keeps saved R source without inventing a kernel', () => {
    expect(parseNotebook(fixture('r')).language).toBe('R');
  });
  it('does not treat a reordered ID as the same revision', () => {
    const a = parseNotebook(fixture('python')),
      b = parseNotebook(fixture('reordered'));
    expect(b.cells[0]?.address).toEqual({ kind: 'id', id: 'filter' });
    expect(() => resolveTarget(b, target(a.revision))).toThrow(/version/);
  });
  it('rejects unknown versions, malformed JSON and non-UTF8 bytes', () => {
    for (const bytes of [
      encode({ ...notebook(), nbformat: 5 }),
      encode(notebook([], 6)),
      Buffer.from('{'),
      Buffer.from([255, 255]),
    ])
      expect(() => parseNotebook(bytes)).toThrow();
  });
  it('rejects malformed source, execution fields and outputs', () => {
    for (const c of [
      { ...code(), source: 7 },
      { ...code(), execution_count: -1 },
      { ...code(), outputs: {} },
    ])
      expect(() => parseNotebook(encode(notebook([c])))).toThrow();
  });
  it('offers text fallback and preserves unavailable output slots', () => {
    const s = parseNotebook(fixture('unsafe')),
      outs = s.cells[2]?.outputs;
    expect(outs?.[0]?.mime).toBe('text/plain');
    expect(outs?.[0]?.notice).toMatch(/Static/);
    expect(outs?.[2]?.part.kind).toBe('unavailable');
    expect(outs?.[3]?.part.kind).toBe('unavailable');
  });
  it('escapes terminal controls and binds capture to the displayed projection', () => {
    const p = projectText('\x1b[31merror\x1b[0m');
    expect(p.text).toContain('\\u001b');
    expect(rawRange(p, 0, 6)).toBe('\x1b');
    expect(() => rawRange(p, 1, 3)).toThrow(/splits/);
  });
  it('rejects unpaired or split surrogate positions and empty ranges', () => {
    expect(() => projectText('\ud800')).toThrow(/surrogate/);
    const p = projectText('a🧬b');
    expect(() => rawRange(p, 1, 2)).toThrow(/splits/);
    expect(() => rawRange(p, 1, 1)).toThrow();
    expect(rawRange(p, 1, 3)).toBe('🧬');
  });
  it('selects repeated output slots by index and exact representation digest', () => {
    const c = code();
    c.outputs = [0, 1].map(() => ({ output_type: 'stream', name: 'stdout', text: 'same' }));
    const s = parseNotebook(encode(notebook([c]))),
      part = s.cells[0]?.outputs[1]?.part;
    if (part?.kind !== 'text') throw Error('fixture');
    const t = {
      ...target(s.revision),
      part: { kind: 'output', index: 1, mime: 'text/plain', digest: part.digest },
    };
    expect(resolveTarget(s, t).rawQuote).toBe('same');
    expect(() => resolveTarget(s, { ...t, part: { ...t.part, mime: 'text/html' } })).toThrow();
  });
});
describe('Bounds and exact visibility', () => {
  it('accepts 1,000 cells and rejects one more', () => {
    expect(parseNotebook(fixture('many')).cells).toHaveLength(1000);
    expect(() =>
      parseNotebook(encode(notebook(Array.from({ length: 1001 }, (_, i) => code('id' + i))))),
    ).toThrow(/1,000/);
  });
  it('checks total and per-cell output count', () => {
    const c = code();
    c.outputs = Array.from({ length: 101 }, () => ({
      output_type: 'stream',
      name: 'stdout',
      text: 'x',
    }));
    expect(() => parseNotebook(encode(notebook([c])))).toThrow(/100/);
    c.outputs.length = 100;
    expect(parseNotebook(encode(notebook([c]))).cells[0]?.outputs).toHaveLength(100);
    expect(() =>
      parseNotebook(
        encode(notebook(Array.from({ length: 11 }, (_, i) => ({ ...c, id: 'c' + i })))),
      ),
    ).toThrow(/1,000 outputs/);
  });
  it('checks below/at/above text and source limits', () => {
    for (const n of [LIMITS.text - 1, LIMITS.text])
      expect(parseNotebook(encode(notebook([code('filter', 'a'.repeat(n))]))).textBytes).toBe(n);
    expect(() =>
      parseNotebook(encode(notebook([code('filter', 'a'.repeat(LIMITS.text + 1))]))),
    ).toThrow(/1 MiB/);
    const base = encode(notebook()).toString();
    for (const n of [LIMITS.bytes - 1, LIMITS.bytes])
      expect(parseNotebook(Buffer.from(base + ' '.repeat(n - Buffer.byteLength(base)))).bytes).toBe(
        n,
      );
    expect(() => parseNotebook(Buffer.alloc(LIMITS.bytes + 1))).toThrow(/8 MiB/);
  });
  it('checks depth and structural-item limits before parsing', () => {
    expect(() => parseNotebook(Buffer.from('['.repeat(33) + '0' + ']'.repeat(33)))).toThrow(
      /depth/,
    );
    expect(() => parseNotebook(encode({ many: Array.from({ length: 100001 }, () => 0) }))).toThrow(
      /item/,
    );
  });
  it('accepts the 4 MP PNG boundary and rejects oversized/header/bad base64 input', () => {
    expect(imageSource('image/png', png(2000, 2000).toString('base64')).width).toBe(2000);
    expect(() => imageSource('image/png', png(2001, 2000).toString('base64'))).toThrow(/4 million/);
    expect(() => imageSource('image/png', 'not base64!')).toThrow();
    expect(() => imageSource('image/png', Buffer.alloc(40).toString('base64'))).toThrow(/header/);
  });
  it('blocks target schema injection, foreign revisions, bad ranges and unavailable parts', () => {
    const s = parseNotebook(fixture('python')),
      t = target(s.revision);
    for (const v of [
      { ...t, path: '/etc/passwd' },
      { ...t, selector: { kind: 'text', start: NaN, end: 4 } },
      { ...t, cell: { kind: 'ordinal', index: 2 } },
      { ...t, revision: 'sha256:' + '0'.repeat(64) },
    ])
      expect(() => resolveTarget(s, v)).toThrow();
  });
  it('receipt scopes reject offscreen, changed and unreturned target ranges', () => {
    const s = parseNotebook(fixture('python')),
      t = target(s.revision);
    const r = validateReceipt(s, {
      revision: s.revision,
      generation: 1,
      parts: [{ cell: t.cell, part: t.part, selectors: [t.selector] }],
    });
    expect(() => assertObserved(r, t)).not.toThrow();
    expect(() => assertObserved(r, target(s.revision, 0, 5))).toThrow(/visible/);
    expect(() => validateReceipt(s, { ...r, revision: 'sha256:' + '0'.repeat(64) })).toThrow(
      /changed/,
    );
    expect(() => assertObserved({ ...r, parts: [] }, t)).toThrow(/visible/);
  });
});

it('accepts equivalent cell addresses regardless of JSON property order', () => {
  const s = parseNotebook(fixture('python'));
  const t = { ...target(s.revision), cell: { id: 'filter', kind: 'id' } };
  expect(resolveTarget(s, t).rawQuote).toBe('keep');
});
it('rejects noncanonical base64 and explicitly refuses EXIF JPEG orientation', () => {
  expect(() => imageSource('image/png', 'AA=A')).toThrow();
  const jpeg = Buffer.from([255, 216, 255, 225, 0, 4, 0, 0, 255, 217]);
  expect(() => imageSource('image/jpeg', jpeg.toString('base64'))).toThrow(/EXIF/);
});

it('bounds text rendering by code units and lines without splitting surrogate pairs', () => {
  const s = 'a'.repeat(2047) + '🧬end';
  const first = textWindow(s, 0);
  expect(first.end).toBe(2047);
  expect(textWindow(s, first.end).text).toBe('🧬end');
  expect(textWindow('x\n'.repeat(10000), 0).text.split('\n')).toHaveLength(201);
});
