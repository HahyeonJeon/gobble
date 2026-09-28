import type { Matrix, Rect } from './protocol';

export function assertRegion(rect: Rect, box: Rect): void {
  if (
    !rect.every(Number.isFinite) ||
    !box.every(Number.isFinite) ||
    rect[0] >= rect[2] ||
    rect[1] >= rect[3] ||
    rect[0] < box[0] ||
    rect[1] < box[1] ||
    rect[2] > box[2] ||
    rect[3] > box[3]
  )
    throw new Error('Region must be nonempty and contained in the source page.');
}
export function transformPoint(m: Matrix, x: number, y: number): [number, number] {
  return [m[0] * x + m[2] * y + m[4], m[1] * x + m[3] * y + m[5]];
}
export function invert(m: Matrix): Matrix {
  const determinant = m[0] * m[3] - m[1] * m[2];
  if (!m.every(Number.isFinite) || !Number.isFinite(determinant) || Math.abs(determinant) < 1e-12)
    throw new Error('Invalid page transformation.');
  return [
    m[3] / determinant,
    -m[1] / determinant,
    -m[2] / determinant,
    m[0] / determinant,
    (m[2] * m[5] - m[3] * m[4]) / determinant,
    (m[1] * m[4] - m[0] * m[5]) / determinant,
  ];
}
export function pixelRegion(rect: Rect, matrix: Matrix, width: number, height: number) {
  const corners = [
    [rect[0], rect[1]],
    [rect[0], rect[3]],
    [rect[2], rect[1]],
    [rect[2], rect[3]],
  ] as const;
  const points = corners.map(([x, y]) => transformPoint(matrix, x, y));
  const left = Math.max(0, Math.floor(Math.min(...points.map((p) => p[0]))));
  const top = Math.max(0, Math.floor(Math.min(...points.map((p) => p[1]))));
  const right = Math.min(width, Math.ceil(Math.max(...points.map((p) => p[0]))));
  const bottom = Math.min(height, Math.ceil(Math.max(...points.map((p) => p[1]))));
  if (right <= left || bottom <= top) throw new Error('Region has no rendered pixels.');
  return { x: left, y: top, width: right - left, height: bottom - top };
}
export function regionFromPixels(rect: Rect, matrix: Matrix): Rect {
  const inv = invert(matrix);
  const points = [
    [rect[0], rect[1]],
    [rect[0], rect[3]],
    [rect[2], rect[1]],
    [rect[2], rect[3]],
  ].map(([x, y]) => transformPoint(inv, x!, y!));
  return [
    Math.min(...points.map((p) => p[0])),
    Math.min(...points.map((p) => p[1])),
    Math.max(...points.map((p) => p[0])),
    Math.max(...points.map((p) => p[1])),
  ];
}
