import { describe, expect, it } from 'vitest';
import {
  numericCell,
  projectScatter,
  rowsInBox,
  validateViewport,
  parse,
  ReferencePresentationSchema,
  type ScatterSpec,
  type TableContent,
} from '../src';

const spec: ScatterSpec = {
  xColumnId: 'x',
  yColumnId: 'y',
  labelColumnId: 'sample',
  xScale: 'linear',
  yScale: 'linear',
};
const table = (): TableContent => ({
  kind: 'table',
  columns: [
    { id: 'sample', name: 'Sample' },
    { id: 'x', name: 'PC1' },
    { id: 'y', name: 'PC2' },
    { id: 'group', name: 'Group' },
  ],
  rows: [
    { key: 'row_1', cells: ['duplicate', '0', '1', 'A'] },
    { key: 'row_2', cells: ['duplicate', '2', '3', 'B'] },
    { key: 'row_3', cells: ['missing', '', '4', 'A'] },
    { key: 'row_4', cells: ['invalid', 'NaN', '4', 'B'] },
  ],
  truncated: false,
});

describe('revision-scoped tabular projection', () => {
  it('distinguishes zero and decimal/scientific notation from missing, locale or non-finite values', () => {
    for (const [raw, expected] of [
      ['0', 0],
      [' -0 ', -0],
      ['.5', 0.5],
      ['1.', 1],
      ['-2.4e2', -240],
    ] as const)
      expect(numericCell(raw)).toBe(expected);
    for (const raw of [
      '',
      ' ',
      'NaN',
      'Infinity',
      '1e309',
      '1e-999',
      '1,234',
      '0x10',
      '2 mg',
      '−2',
      '1_000',
    ])
      expect(numericCell(raw), raw).toBeNull();
  });
  it('preserves raw cells and exact distinct row keys despite duplicate labels, reorder and filtering', () => {
    const input = table(),
      original = structuredClone(input);
    const projected = projectScatter(input, spec);
    expect(projected.points.map((p) => p.rowKey)).toEqual(['row_1', 'row_2']);
    expect(projected.excludedRows).toBe(2);
    expect(input).toEqual(original);
    input.rows.reverse();
    expect(
      projectScatter(input, spec, { kind: 'equals', columnId: 'group', value: 'A' }).points,
    ).toEqual([projected.points[0]]);
    expect(rowsInBox(projectScatter(input, spec).points, { x: [-1, 1], y: [0, 2] })).toEqual([
      'row_1',
    ]);
  });
  it('returns honest empty/truncated projection and refuses unknown columns or ambiguous keys', () => {
    expect(projectScatter({ ...table(), rows: [], truncated: true }, spec)).toMatchObject({
      points: [],
      filteredRows: 0,
      excludedRows: 0,
      truncated: true,
    });
    expect(() => projectScatter(table(), { ...spec, xColumnId: 'unknown' })).toThrow('missing');
    const duplicate = table();
    duplicate.rows[1]!.key = 'row_1';
    expect(() => projectScatter(duplicate, spec)).toThrow('unique');
    const columns = table();
    columns.columns.push({ id: 'x', name: 'Other' });
    expect(() => projectScatter(columns, spec)).toThrow('ambiguous');
  });
  it('requires typed, finite, ordered presentation context without accepting arbitrary metadata', () => {
    const value = {
      schemaVersion: 1,
      kind: 'scatter',
      spec,
      filter: { kind: 'all' },
      viewport: { x: [0, 1], y: [-1, 1] },
      revision: { spec: 1, view: 2, filter: 0 },
    };
    expect(parse(ReferencePresentationSchema, value)).toEqual(value);
    expect(() =>
      parse(ReferencePresentationSchema, { ...value, metadata: { url: 'file:///private' } }),
    ).toThrow();
    expect(() =>
      parse(ReferencePresentationSchema, { ...value, spec: { ...spec, xScale: 'log' } }),
    ).toThrow();
    for (const range of [
      [1, 1],
      [2, 1],
      [-Infinity, 1],
      [0, NaN],
    ])
      expect(() => validateViewport({ x: range as [number, number], y: [0, 1] })).toThrow();
  });
});
