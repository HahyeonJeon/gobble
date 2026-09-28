// Storage-only v2 reader. Reuse unchanged fields, explicitly freeze every reference branch.
import { SurfaceV3Schema } from './surface-v3';
import { Type, type Static } from '@sinclair/typebox';
import {
  closed,
  AgentIdSchema,
  CounterSchema,
  DecisionIdSchema,
  RequestIdSchema,
} from './identity';
import { EvidenceRefV1Schema } from './reference-v1';
import { WorkspaceDocumentV1Schema, StoredLegacyDecisionSchema } from './workspace-document-v1';
import { WorkspaceSchema } from './workspace';
import {
  DraftAttachmentV5Schema as DraftAttachmentSchema,
  EvidenceManifestV5Schema as EvidenceManifestSchema,
} from './document-content-v5';
import { SubmissionV5Schema as SubmissionSchema } from './document-content-v5';
import { QuestionV5Schema as QuestionSchema } from './document-content-v5';
import { SharedReferenceIdSchema } from './shared-context';
import { SharedReferenceV5Schema as SharedReferenceSchema } from './document-content-v5';

const draft = Type.Composite(
  [Type.Omit(DraftAttachmentSchema, ['evidence']), Type.Object({ evidence: EvidenceRefV1Schema })],
  closed,
);
const manifest = Type.Composite(
  [Type.Omit(EvidenceManifestSchema, ['evidence']), Type.Object({ evidence: EvidenceRefV1Schema })],
  closed,
);
const submission = Type.Composite(
  [
    Type.Omit(SubmissionSchema, ['evidence']),
    Type.Object({ evidence: Type.Optional(Type.Array(manifest, { minItems: 1, maxItems: 16 })) }),
  ],
  closed,
);
const question = Type.Composite(
  [
    Type.Omit(QuestionSchema, ['evidence']),
    Type.Object({ evidence: Type.Array(manifest, { minItems: 1, maxItems: 16 }) }),
  ],
  closed,
);
const reference = Type.Composite(
  [Type.Omit(SharedReferenceSchema, ['evidence']), Type.Object({ evidence: EvidenceRefV1Schema })],
  closed,
);
export const WorkspaceDocumentV2Schema = Type.Composite(
  [
    Type.Omit(WorkspaceDocumentV1Schema, [
      'schemaVersion',
      'discussion',
      'collaboration',
      'workspace',
    ]),
    Type.Object({
      schemaVersion: Type.Literal(2),
      workspace: Type.Composite(
        [
          Type.Omit(WorkspaceSchema, ['decisions', 'surfaces']),
          Type.Object({
            surfaces: Type.Array(SurfaceV3Schema, { maxItems: 64 }),
            decisions: Type.Array(Type.Union([StoredLegacyDecisionSchema, question]), {
              maxItems: 1000,
            }),
          }),
        ],
        closed,
      ),
      collaboration: Type.Optional(
        Type.Object({ submissions: Type.Array(submission, { maxItems: 40 }) }, closed),
      ),
      referenceReveal: Type.Optional(
        Type.Object({ referenceId: SharedReferenceIdSchema, requestId: RequestIdSchema }, closed),
      ),
      sharedReferences: Type.Optional(Type.Array(reference, { maxItems: 128 })),
      paneOrientation: Type.Literal('vertical'),
      chat: Type.Object(
        {
          collapsed: Type.Boolean(),
          width: Type.Integer({ minimum: 360, maximum: 560 }),
          draft: Type.String({ maxLength: 16000 }),
          replyToQuestionId: Type.Optional(DecisionIdSchema),
          attachments: Type.Optional(Type.Array(draft, { maxItems: 16 })),
          attachmentRevision: Type.Optional(CounterSchema),
          recipientAgentId: Type.Union([AgentIdSchema, Type.Null()]),
        },
        closed,
      ),
    }),
  ],
  closed,
);
export type WorkspaceDocumentV2 = Static<typeof WorkspaceDocumentV2Schema>;
