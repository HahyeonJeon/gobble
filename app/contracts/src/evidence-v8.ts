// Frozen Workspace v7/v8 attachment shapes.
import { Type } from '@sinclair/typebox';
import { closed, TimestampSchema } from './identity';
import { EvidenceRefV3Schema as EvidenceRefSchema } from './reference-v3';
import { ReferencePresentationSchema } from './tabular-view';
import {
  AttachmentIdSchema,
  EvidenceHashSchema,
  EvidenceRepresentationV10Schema as EvidenceRepresentationSchema,
  EVIDENCE_TEXT_BYTES,
  EVIDENCE_IMAGE_BYTES,
} from './evidence';
export const DraftAttachmentV8Schema = Type.Object(
  {
    attachmentId: AttachmentIdSchema,
    evidence: EvidenceRefSchema,
    presentation: Type.Optional(ReferencePresentationSchema),
    label: Type.String({ minLength: 1, maxLength: 512 }),
    createdAt: TimestampSchema,
    capture: Type.Optional(
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
    ),
  },
  closed,
);
export const EvidenceManifestV8Schema = Type.Composite(
  [
    DraftAttachmentV8Schema,
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
