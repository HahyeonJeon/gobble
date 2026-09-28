import { SavedReportSchema, ReportRepresentationSchema } from './run-report';
import { PipelineCaptureSchema } from './pipeline-reference';
import { NotebookRepresentationSchema, NotebookTextEvidenceSchema } from './notebook-evidence';
import { PdfImageRepresentationSchema } from './pdf';
import { DependencyCaptureSchema } from './run-dependencies';
import { Type, type Static } from '@sinclair/typebox';
import { closed, TimestampSchema } from './identity';
import { EvidenceRefSchema } from './context';
import { ReferencePresentationSchema } from './tabular-view';
import { ObservedEvidenceV2Schema } from './observed-evidence-v2';
import { ObservedEvidenceSchema } from './observed-evidence';

export const ATTACHMENT_LIMIT = 16;
export const EVIDENCE_TEXT_BYTES = 64 * 1024;
export const EVIDENCE_IMAGE_BYTES = 1024 * 1024;
export const AttachmentIdSchema = Type.String({ pattern: '^att_[A-Za-z0-9_-]{1,128}$' });
export const PreparedEvidenceIdSchema = Type.String({ pattern: '^prep_[A-Za-z0-9_-]{1,128}$' });
export const EvidenceHashSchema = Type.String({ pattern: '^sha256:[a-f0-9]{64}$' });
export const DraftAttachmentSchema = Type.Object(
  {
    attachmentId: AttachmentIdSchema,
    evidence: EvidenceRefSchema,
    presentation: Type.Optional(ReferencePresentationSchema),
    label: Type.String({ minLength: 1, maxLength: 512 }),
    createdAt: TimestampSchema,
    capture: Type.Optional(
      Type.Union([
        Type.Object(
          {
            kind: Type.Literal('report'),
            capturedAt: TimestampSchema,
            asset: Type.Object(
              {
                hash: EvidenceHashSchema,
                byteLength: Type.Integer({ minimum: 1, maximum: 1024 * 1024 }),
                mediaType: Type.Literal('application/json'),
              },
              closed,
            ),
            representation: ReportRepresentationSchema,
          },
          closed,
        ),
        Type.Object(
          {
            kind: Type.Literal('pipeline'),
            capturedAt: TimestampSchema,
            asset: Type.Object(
              {
                hash: EvidenceHashSchema,
                byteLength: Type.Integer({ minimum: 1, maximum: EVIDENCE_TEXT_BYTES }),
                mediaType: Type.Literal('application/json'),
              },
              closed,
            ),
            representation: Type.Object(
              { kind: Type.Literal('pipeline'), truncated: Type.Literal(false) },
              closed,
            ),
          },
          closed,
        ),
        Type.Object(
          {
            kind: Type.Literal('notebook'),
            capturedAt: TimestampSchema,
            asset: Type.Object(
              {
                hash: EvidenceHashSchema,
                byteLength: Type.Integer({ minimum: 1, maximum: 786432 }),
                mediaType: Type.Union([
                  Type.Literal('application/json'),
                  Type.Literal('image/png'),
                ]),
              },
              closed,
            ),
            representation: NotebookRepresentationSchema,
          },
          closed,
        ),
        Type.Object(
          {
            kind: Type.Literal('pdf'),
            capturedAt: TimestampSchema,
            asset: Type.Object(
              {
                hash: EvidenceHashSchema,
                byteLength: Type.Integer({ minimum: 1, maximum: 786432 }),
                mediaType: Type.Literal('image/png'),
              },
              closed,
            ),
            representation: PdfImageRepresentationSchema,
          },
          closed,
        ),
        Type.Object(
          {
            kind: Type.Literal('observed'),
            capturedAt: TimestampSchema,
            asset: Type.Object(
              {
                hash: EvidenceHashSchema,
                byteLength: Type.Integer({ minimum: 1, maximum: EVIDENCE_TEXT_BYTES }),
                mediaType: Type.Literal('application/json'),
              },
              closed,
            ),
            representation: Type.Object(
              {
                kind: Type.Union([Type.Literal('run'), Type.Literal('log')]),
                truncated: Type.Boolean(),
              },
              closed,
            ),
          },
          closed,
        ),
      ]),
    ),
  },
  closed,
);
const pixel = Type.Integer({ minimum: 0, maximum: 20_000_000 });
const size = Type.Integer({ minimum: 1, maximum: 20_000_000 });
export const EvidenceRepresentationV10Schema = Type.Union([
  Type.Object(
    {
      kind: Type.Union([
        Type.Literal('text'),
        Type.Literal('table'),
        Type.Literal('run'),
        Type.Literal('log'),
      ]),
      truncated: Type.Boolean(),
    },
    closed,
  ),
  Type.Object(
    {
      kind: Type.Literal('image'),
      width: Type.Integer({ minimum: 1, maximum: 1536 }),
      height: Type.Integer({ minimum: 1, maximum: 1536 }),
      originalWidth: size,
      originalHeight: size,
      crop: Type.Object({ x: pixel, y: pixel, width: size, height: size }, closed),
    },
    closed,
  ),
]);
export const EvidenceRepresentationSchema = Type.Union([
  ReportRepresentationSchema,
  Type.Object({ kind: Type.Literal('pipeline'), truncated: Type.Literal(false) }, closed),
  ...NotebookRepresentationSchema.anyOf,
  PdfImageRepresentationSchema,
  ...EvidenceRepresentationV10Schema.anyOf,
]);
export const EvidenceManifestSchema = Type.Composite(
  [
    DraftAttachmentSchema,
    Type.Object({
      capturedAt: TimestampSchema,
      asset: Type.Object(
        {
          hash: EvidenceHashSchema,
          byteLength: Type.Integer({ minimum: 1, maximum: EVIDENCE_IMAGE_BYTES }),
          mediaType: Type.Union([Type.Literal('application/json'), Type.Literal('image/png')]),
        },
        closed,
      ),
      representation: EvidenceRepresentationSchema,
    }),
  ],
  closed,
);
export const EvidenceTextContentSchema = Type.Union([
  Type.Object({ kind: Type.Literal('pipeline'), pipeline: PipelineCaptureSchema }, closed),
  NotebookTextEvidenceSchema,
  Type.Object({ kind: Type.Literal('run'), dependency: DependencyCaptureSchema }, closed),
  Type.Object(
    {
      kind: Type.Union([Type.Literal('text'), Type.Literal('run'), Type.Literal('log')]),
      text: Type.String({ maxLength: EVIDENCE_TEXT_BYTES }),
      observation: Type.Optional(Type.Union([ObservedEvidenceSchema, ObservedEvidenceV2Schema])),
    },
    closed,
  ),
  Type.Object(
    {
      kind: Type.Literal('table'),
      presentation: Type.Optional(ReferencePresentationSchema),
      scope: Type.Optional(
        Type.Object(
          {
            requestedRows: Type.Integer({ minimum: 0, maximum: 500 }),
            returnedRows: Type.Integer({ minimum: 0, maximum: 500 }),
            previewRows: Type.Integer({ minimum: 0, maximum: 500 }),
            sourceTruncated: Type.Boolean(),
          },
          closed,
        ),
      ),
      columns: Type.Array(
        Type.Object(
          {
            id: Type.String({ maxLength: 256 }),
            name: Type.String({ maxLength: EVIDENCE_TEXT_BYTES }),
          },
          closed,
        ),
        { maxItems: 100 },
      ),
      rows: Type.Array(
        Type.Object(
          {
            key: Type.String({ maxLength: 256 }),
            cells: Type.Array(Type.String({ maxLength: EVIDENCE_TEXT_BYTES }), { maxItems: 100 }),
          },
          closed,
        ),
        { maxItems: 500 },
      ),
    },
    closed,
  ),
]);
export const EvidencePreviewSchema = Type.Union([
  Type.Object({ kind: Type.Literal('report'), report: SavedReportSchema }, closed),
  EvidenceTextContentSchema,
  Type.Object(
    {
      kind: Type.Literal('image'),
      base64: Type.String({
        minLength: 1,
        maxLength: EVIDENCE_IMAGE_BYTES,
        pattern: '^[A-Za-z0-9+/]+={0,2}$',
      }),
    },
    closed,
  ),
]);
export type DraftAttachment = Static<typeof DraftAttachmentSchema>;
export type EvidenceManifest = Static<typeof EvidenceManifestSchema>;
export type EvidenceTextContent = Static<typeof EvidenceTextContentSchema>;
export type EvidencePreview = Static<typeof EvidencePreviewSchema>;

/** Text delivered to the recipient, distinct from retained report bytes. */
export function evidenceTextBytes(item: EvidenceManifest): number {
  return item.representation.kind === 'report'
    ? item.representation.textByteLength
    : item.asset.mediaType === 'application/json'
      ? item.asset.byteLength
      : 0;
}
