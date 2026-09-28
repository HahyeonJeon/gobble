import { SurfaceV3Schema } from './surface-v3';
import { Type, type Static } from '@sinclair/typebox';
import {
  AgentIdSchema,
  closed,
  RequestIdSchema,
  SurfaceIdSchema,
  TimestampSchema,
} from './identity';
import { LegacyDecisionSchema } from './decision';
import { EvidenceRefV1Schema } from './reference-v1';
import { PaneIdSchema, WorkspaceSchema } from './workspace';

import { SubmissionSchema } from './collaboration';

// Freeze the legacy message shape: v2 evidence must never enter the v1 reader.
const legacySubmission = Type.Composite(
  [
    Type.Pick(SubmissionSchema, [
      'requestId',
      'agentId',
      'model',
      'effort',
      'createdAt',
      'state',
      'threadId',
      'turnId',
      'response',
      'problem',
    ]),
    Type.Object({ text: Type.String({ minLength: 1, maxLength: 16000 }) }),
  ],
  closed,
);

export const StoredLegacyDecisionSchema = Type.Composite(
  [
    Type.Omit(LegacyDecisionSchema, ['evidence']),
    Type.Object({ evidence: Type.Array(EvidenceRefV1Schema, { minItems: 1, maxItems: 16 }) }),
  ],
  closed,
);

export const WorkspaceDocumentV1Schema = Type.Object(
  {
    schemaVersion: Type.Literal(1),
    workspace: Type.Composite(
      [
        Type.Omit(WorkspaceSchema, ['decisions', 'surfaces']),
        Type.Object({
          surfaces: Type.Array(SurfaceV3Schema, { maxItems: 64 }),
          decisions: Type.Array(StoredLegacyDecisionSchema, { maxItems: 1000 }),
        }),
      ],
      closed,
    ),
    collaboration: Type.Optional(
      Type.Object({ submissions: Type.Array(legacySubmission, { maxItems: 40 }) }, closed),
    ),
    titles: Type.Array(
      Type.Object(
        {
          surfaceId: SurfaceIdSchema,
          title: Type.String({ minLength: 1, maxLength: 512 }),
        },
        closed,
      ),
      { maxItems: 64 },
    ),
    activePane: PaneIdSchema,
    maximizedPane: Type.Union([PaneIdSchema, Type.Null()]),
    discussion: Type.Object(
      {
        collapsed: Type.Boolean(),
        height: Type.Integer({ minimum: 160, maximum: 440 }),
        draft: Type.String({ maxLength: 16000 }),
        recipientAgentId: Type.Union([AgentIdSchema, Type.Null()]),
      },
      closed,
    ),
    selections: Type.Array(EvidenceRefV1Schema, { maxItems: 64 }),
    activity: Type.Array(
      Type.Object(
        {
          id: RequestIdSchema,
          text: Type.String({ minLength: 1, maxLength: 700 }),
          at: TimestampSchema,
        },
        closed,
      ),
      { maxItems: 100 },
    ),
    receipts: Type.Array(
      Type.Object(
        {
          requestId: RequestIdSchema,
          digest: Type.String({ pattern: '^[a-f0-9]{64}$' }),
        },
        closed,
      ),
      { maxItems: 256 },
    ),
  },
  closed,
);

export type WorkspaceDocumentV1 = Static<typeof WorkspaceDocumentV1Schema>;
