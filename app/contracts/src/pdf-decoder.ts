import { Type, type Static } from '@sinclair/typebox';
import { Value } from '@sinclair/typebox/value';

export const LIMITS = {
  fileBytes: 8 * 1024 * 1024,
  pages: 200,
  rasterPixels: 4_000_000,
  sourceImagePixels: 20_000_000,
  pagePngBase64: 6 * 1024 * 1024,
  capturePngBase64: 1024 * 1024,
  textItems: 1000,
  textUtf16: 32_768,
  jobMs: 5000,
} as const;
export const PROFILE = 'pdfjs-6.3.289-region-v1:no-xfa,no-annotations,path-fonts,no-wasm';
export const HOST_URL = 'pdf-host://decoder/index.html';
export const REPLY_CHANNEL = 'gobble:pdf:reply';
export const JOB_CHANNEL = 'gobble:pdf:job';
const closed = { additionalProperties: false } as const;
const coordinate = Type.Number({ minimum: -1e7, maximum: 1e7 });
const hash = Type.String({ pattern: '^sha256:[a-f0-9]{64}$' });
export const RectSchema = Type.Tuple([coordinate, coordinate, coordinate, coordinate]);
export const MatrixSchema = Type.Tuple([
  coordinate,
  coordinate,
  coordinate,
  coordinate,
  coordinate,
  coordinate,
]);
const pageIndex = Type.Integer({ minimum: 0, maximum: LIMITS.pages - 1 });
const rotation = Type.Union([
  Type.Literal(0),
  Type.Literal(90),
  Type.Literal(180),
  Type.Literal(270),
]);
const rasterShape = {
  width: Type.Integer({ minimum: 1, maximum: LIMITS.rasterPixels }),
  height: Type.Integer({ minimum: 1, maximum: LIMITS.rasterPixels }),
  png: Type.String({ maxLength: LIMITS.pagePngBase64, pattern: '^[A-Za-z0-9+/]*={0,2}$' }),
};
const ItemSchema = Type.Object(
  {
    id: Type.Integer({ minimum: 0, maximum: LIMITS.textItems - 1 }),
    text: Type.String({ maxLength: LIMITS.textUtf16 }),
    transform: MatrixSchema,
    width: coordinate,
    height: coordinate,
    direction: Type.String({ maxLength: 8 }),
  },
  closed,
);
export const ModelSchema = Type.Object(
  {
    revision: hash,
    profile: Type.Literal(PROFILE),
    pageIndex,
    viewBox: RectSchema,
    userUnit: Type.Number({ minimum: 0.001, maximum: 75000 }),
    intrinsicRotation: rotation,
    text: Type.Object(
      {
        capability: Type.Literal('diagnostic-only'),
        items: Type.Array(ItemSchema, { maxItems: LIMITS.textItems }),
        truncated: Type.Boolean(),
      },
      closed,
    ),
  },
  closed,
);
export const PageSchema = Type.Object(
  {
    kind: Type.Literal('page'),
    model: ModelSchema,
    modelHash: hash,
    renditionId: Type.String({ minLength: 1, maxLength: 100 }),
    viewport: MatrixSchema,
    raster: Type.Object(rasterShape, closed),
    timings: Type.Object(
      { modelMs: Type.Number({ minimum: 0 }), renderMs: Type.Number({ minimum: 0 }) },
      closed,
    ),
  },
  closed,
);
export const OpenedSchema = Type.Object(
  {
    kind: Type.Literal('opened'),
    revision: hash,
    profile: Type.Literal(PROFILE),
    pageCount: Type.Integer({ minimum: 1, maximum: LIMITS.pages }),
  },
  closed,
);
export const CaptureSchema = Type.Object(
  {
    kind: Type.Literal('capture'),
    revision: hash,
    modelHash: hash,
    renditionId: Type.String({ minLength: 1, maxLength: 100 }),
    pageIndex,
    region: RectSchema,
    pixelRect: Type.Object(
      {
        x: Type.Integer({ minimum: 0 }),
        y: Type.Integer({ minimum: 0 }),
        width: Type.Integer({ minimum: 1 }),
        height: Type.Integer({ minimum: 1 }),
      },
      closed,
    ),
    raster: Type.Object(
      {
        ...rasterShape,
        png: Type.String({ maxLength: LIMITS.capturePngBase64, pattern: '^[A-Za-z0-9+/]*={0,2}$' }),
      },
      closed,
    ),
  },
  closed,
);
export const ResultSchema = Type.Union([OpenedSchema, PageSchema, CaptureSchema]);
export const ReplySchema = Type.Union([
  Type.Object(
    { jobId: Type.String({ maxLength: 100 }), ok: Type.Literal(true), value: ResultSchema },
    closed,
  ),
  Type.Object(
    {
      jobId: Type.String({ maxLength: 100 }),
      ok: Type.Literal(false),
      message: Type.String({ maxLength: 500 }),
    },
    closed,
  ),
]);
export type Rect = Static<typeof RectSchema>;
export type Matrix = Static<typeof MatrixSchema>;
export type PageModel = Static<typeof ModelSchema>;
export type PageResult = Static<typeof PageSchema>;
export type CaptureResult = Static<typeof CaptureSchema>;
export type Result = Static<typeof ResultSchema>;
export type Reply = Static<typeof ReplySchema>;
export type Job =
  | { jobId: string; kind: 'open'; bytes: Uint8Array; revision: string }
  | {
      jobId: string;
      kind: 'page';
      pageIndex: number;
      scale: number;
      rotation: 0 | 90 | 180 | 270;
      dpr: number;
    }
  | { jobId: string; kind: 'capture'; renditionId: string; modelHash: string; region: Rect };

export function parseReply(input: unknown): Reply {
  if (!Value.Check(ReplySchema, input)) throw new Error('Invalid PDF host reply.');
  if (input.ok && input.value.kind !== 'opened') {
    const { width, height } = input.value.raster;
    if (width * height > LIMITS.rasterPixels) throw new Error('PDF raster pixel limit exceeded.');
  }
  return input;
}
export function parseRegion(input: unknown): Rect {
  if (!Value.Check(RectSchema, input)) throw new Error('Invalid PDF region.');
  return input;
}
