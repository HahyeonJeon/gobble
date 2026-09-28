import { SurfaceV3Schema } from './surface-v3';
import { Type, type Static } from '@sinclair/typebox';
import { AgentIdSchema, closed, CounterSchema, RequestIdSchema } from './identity';
import { WorkspaceV5Schema as WorkspaceSchema } from './document-content-v5';
import { DecisionIdSchema } from './identity';
import { DraftAttachmentV5Schema as DraftAttachmentSchema } from './document-content-v5';
import { CollaborationHistoryV5Schema as CollaborationHistorySchema } from './document-content-v5';
import { WorkspaceDocumentV1Schema } from './workspace-document-v1';
import { SharedReferenceIdSchema } from './shared-context';
import { SharedReferenceV5Schema as SharedReferenceSchema } from './document-content-v5';
import { LocalSelectionV2Schema as LocalSelectionSchema } from './reference-v2';

/** v3 separates portable targets from Surface-owned local selections. */
export const WorkspaceDocumentV3Schema = Type.Composite(
  [
    Type.Omit(WorkspaceDocumentV1Schema, [
      'schemaVersion',
      'discussion',
      'collaboration',
      'workspace',
      'selections',
    ]),
    Type.Object({
      schemaVersion: Type.Literal(3),
      selections: Type.Array(LocalSelectionSchema, { maxItems: 64 }),
      workspace: Type.Composite(
        [
          Type.Omit(WorkspaceSchema, ['surfaces']),
          Type.Object({ surfaces: Type.Array(SurfaceV3Schema, { maxItems: 64 }) }),
        ],
        closed,
      ),
      collaboration: Type.Optional(CollaborationHistorySchema),
      referenceReveal: Type.Optional(
        Type.Object({ referenceId: SharedReferenceIdSchema, requestId: RequestIdSchema }, closed),
      ),
      sharedReferences: Type.Optional(Type.Array(SharedReferenceSchema, { maxItems: 128 })),
      paneOrientation: Type.Literal('vertical'),
      chat: Type.Object(
        {
          collapsed: Type.Boolean(),
          width: Type.Integer({ minimum: 360, maximum: 560 }),
          draft: Type.String({ maxLength: 16000 }),
          replyToQuestionId: Type.Optional(DecisionIdSchema),
          attachments: Type.Optional(Type.Array(DraftAttachmentSchema, { maxItems: 16 })),
          attachmentRevision: Type.Optional(CounterSchema),
          recipientAgentId: Type.Union([AgentIdSchema, Type.Null()]),
        },
        closed,
      ),
    }),
  ],
  closed,
);

export type WorkspaceDocumentV3 = Static<typeof WorkspaceDocumentV3Schema>;
