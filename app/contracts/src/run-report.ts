import { Type, type Static } from '@sinclair/typebox';
import {
  closed,
  ProjectIdSchema,
  RunRefSchema,
  SurfaceIdSchema,
  TimestampSchema,
} from './identity';
import { serviceResult } from './result';

export const REPORT_BYTES = 1024 * 1024;
export const REPORT_TEXT_BYTES = 64 * 1024;
const hash = Type.String({ pattern: '^sha256:[a-f0-9]{64}$' });
const text = Type.String({ maxLength: REPORT_TEXT_BYTES });
export const ReportProducerSchema = Type.Object(
  {
    schemaVersion: Type.Literal(1),
    runId: Type.String({ minLength: 1, maxLength: 256 }),
    snapshot: Type.String({ pattern: '^[a-f0-9]{32,128}$' }),
    originDigest: hash,
    instance: Type.String({ minLength: 1, maxLength: 256 }),
    attempt: Type.Integer({ minimum: 1, maximum: Number.MAX_SAFE_INTEGER }),
    port: Type.Literal('html'),
    recipe: Type.Literal('fastqc-v1'),
    sha256: hash,
    size: Type.Integer({ minimum: 1, maximum: REPORT_BYTES }),
  },
  closed,
);
export const RunReportSourceSchema = Type.Object(
  {
    projectId: ProjectIdSchema,
    runRef: RunRefSchema,
    observedAt: TimestampSchema,
    evidence: Type.Composite(
      [ReportProducerSchema, Type.Object({ path: Type.String({ minLength: 1, maxLength: 4096 }) })],
      closed,
    ),
    base64: Type.String({ minLength: 1, maxLength: Math.ceil(REPORT_BYTES / 3) * 4 }),
  },
  closed,
);
export const RunReportResultSchema = serviceResult(RunReportSourceSchema);
export const ReportBlockSchema = Type.Union([
  Type.Object({ kind: Type.Literal('text'), text }, closed),
  Type.Object(
    {
      kind: Type.Literal('table'),
      headers: Type.Array(text, { minItems: 1, maxItems: 32 }),
      rows: Type.Array(Type.Array(text, { minItems: 1, maxItems: 32 }), { maxItems: 2000 }),
    },
    closed,
  ),
  Type.Object(
    {
      kind: Type.Literal('image'),
      id: Type.String({ pattern: '^image-[0-9]+$' }),
      alt: text,
      width: Type.Integer({ minimum: 1, maximum: 1536 }),
      height: Type.Integer({ minimum: 1, maximum: 1536 }),
      base64: Type.String({ minLength: 1, maxLength: Math.ceil(REPORT_BYTES / 3) * 4 }),
    },
    closed,
  ),
]);
export const FastqcContentSchema = Type.Object(
  {
    profile: Type.Literal('fastqc-0.12.1-v1'),
    title: text,
    logoLabel: text,
    headerTitle: text,
    headerFilename: text,
    summaryHeading: text,
    summary: Type.Array(
      Type.Object(
        { moduleId: Type.String({ pattern: '^M[0-9]+$' }), title: text, status: text },
        closed,
      ),
      { minItems: 1, maxItems: 16 },
    ),
    modules: Type.Array(
      Type.Object(
        {
          id: Type.String({ pattern: '^M[0-9]+$' }),
          title: text,
          status: text,
          blocks: Type.Array(ReportBlockSchema, { minItems: 1, maxItems: 32 }),
        },
        closed,
      ),
      { minItems: 1, maxItems: 16 },
    ),
    footer: text,
  },
  closed,
);
export const SavedReportSchema = Type.Object(
  {
    schemaVersion: Type.Literal(1),
    projectId: ProjectIdSchema,
    runRef: RunRefSchema,
    producer: ReportProducerSchema,
    capturedAt: TimestampSchema,
    content: FastqcContentSchema,
  },
  closed,
);
export const SavedReportRecordSchema = Type.Object(
  {
    projectId: ProjectIdSchema,
    runRef: RunRefSchema,
    producer: ReportProducerSchema,
    capturedAt: TimestampSchema,
    title: Type.String({ minLength: 1, maxLength: 512 }),
    profile: Type.Literal('fastqc-0.12.1-v1'),
    asset: Type.Object(
      { hash, byteLength: Type.Integer({ minimum: 1, maximum: REPORT_BYTES }) },
      closed,
    ),
  },
  closed,
);
export type RunReportSource = Static<typeof RunReportSourceSchema>;
export type SavedReport = Static<typeof SavedReportSchema>;
export type SavedReportRecord = Static<typeof SavedReportRecordSchema>;
export type FastqcContent = Static<typeof FastqcContentSchema>;
export type ReportBlock = Static<typeof ReportBlockSchema>;

/** Whole saved report only; image IDs are reading addresses, never selection coordinates. */
export const ReportTargetSchema = Type.Object(
  {
    schemaVersion: Type.Literal(8),
    projectId: ProjectIdSchema,
    resource: Type.Object({ kind: Type.Literal('report'), saved: SavedReportRecordSchema }, closed),
    dataRevision: hash,
    origin: Type.Optional(Type.Object({ surfaceId: SurfaceIdSchema }, closed)),
    selection: Type.Optional(Type.Never()),
  },
  closed,
);
export type ReportTarget = Static<typeof ReportTargetSchema>;
export const ReportRepresentationSchema = Type.Object(
  {
    kind: Type.Literal('report'),
    truncated: Type.Literal(false),
    textByteLength: Type.Integer({ minimum: 1, maximum: REPORT_TEXT_BYTES }),
    imageCount: Type.Integer({ minimum: 0, maximum: 16 }),
  },
  closed,
);
/** Complete reading content with PNG inventory; original pixels are fetched separately. */
export function reportReading(report: SavedReport) {
  return {
    ...report,
    content: {
      ...report.content,
      modules: report.content.modules.map((module) => ({
        ...module,
        blocks: module.blocks.map((block) => {
          if (block.kind !== 'image') return block;
          const { base64: _pixels, ...inventory } = block;
          return inventory;
        }),
      })),
    },
  };
}
export function reportTarget(saved: SavedReportRecord, surfaceId?: string): ReportTarget {
  return {
    schemaVersion: 8,
    projectId: saved.projectId,
    resource: { kind: 'report', saved },
    dataRevision: saved.asset.hash,
    ...(surfaceId ? { origin: { surfaceId } } : {}),
  };
}
