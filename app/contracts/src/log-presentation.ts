import { Type, type Static } from '@sinclair/typebox';
import { closed, ProjectIdSchema, RunRefSchema, RevisionSchema, TimestampSchema } from './identity';

export const LogStreamSchema = Type.Object(
  {
    text: Type.String({ maxLength: 4096 }),
    sourceBytesReported: Type.Union([
      Type.Integer({ minimum: 0, maximum: Number.MAX_SAFE_INTEGER }),
      Type.Null(),
    ]),
    decodedUtf8Bytes: Type.Integer({ minimum: 0, maximum: 12288 }),
    earlierBytesOmitted: Type.Boolean(),
    completeness: Type.Literal('unknown'),
    availability: Type.Union([Type.Literal('text-returned'), Type.Literal('no-text-returned')]),
  },
  closed,
);

/** Normalized observation, not a file cursor or an atomic control/log snapshot. */
export const LogPresentationSchema = Type.Object(
  {
    schemaVersion: Type.Literal(1),
    projectId: ProjectIdSchema,
    runRef: RunRefSchema,
    engineRevision: RevisionSchema,
    observedAt: TimestampSchema,
    instance: Type.String({ minLength: 1, maxLength: 256 }),
    attempt: Type.Integer({ minimum: 1, maximum: Number.MAX_SAFE_INTEGER }),
    tailLimitBytes: Type.Literal(4096),
    error: Type.Union([Type.String({ maxLength: 4096 }), Type.Null()]),
    streams: Type.Object({ stdout: LogStreamSchema, stderr: LogStreamSchema }, closed),
  },
  closed,
);
export type LogPresentation = Static<typeof LogPresentationSchema>;
