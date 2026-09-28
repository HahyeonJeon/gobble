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
import { ViewLinkV4Schema } from './view-link-v4';
import { SurfaceV4Schema } from './surface-v4';
/** v4 adds explicit linked tabular membership without merging existing local selections. */
export const WorkspaceDocumentV4Schema = Type.Composite(
  [
    Type.Omit(WorkspaceDocumentV1Schema, [
      'schemaVersion',
      'discussion',
      'collaboration',
      'workspace',
      'selections',
    ]),
    Type.Object({
      schemaVersion: Type.Literal(4),
      viewLinks: Type.Array(ViewLinkV4Schema, { maxItems: 64 }),
      selections: Type.Array(LocalSelectionSchema, { maxItems: 64 }),
      workspace: Type.Object(
        { ...WorkspaceSchema.properties, surfaces: Type.Array(SurfaceV4Schema, { maxItems: 64 }) },
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

export type WorkspaceDocumentV4 = Static<typeof WorkspaceDocumentV4Schema>;
