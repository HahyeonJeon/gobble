import { Type, type Static } from '@sinclair/typebox';
import {
  AgentIdSchema,
  closed,
  CounterSchema,
  RequestIdSchema,
  DecisionIdSchema,
} from './identity';
import { WorkspaceDocumentV1Schema } from './workspace-document-v1';
import { SharedReferenceIdSchema } from './shared-context';
import { LocalSelectionV2Schema as LocalSelectionSchema } from './reference-v2';
import { ViewLinkSchema } from './tabular-view';
import {
  DraftAttachmentV5Schema as DraftAttachmentSchema,
  CollaborationHistoryV5Schema as CollaborationHistorySchema,
  SharedReferenceV5Schema as SharedReferenceSchema,
  WorkspaceV5Schema as WorkspaceSchema,
} from './document-content-v5';
/** v5 adds Surface-owned table settings and linked selection-origin metadata. */
export const WorkspaceDocumentV5Schema = Type.Composite(
  [
    Type.Omit(WorkspaceDocumentV1Schema, [
      'schemaVersion',
      'discussion',
      'collaboration',
      'workspace',
      'selections',
    ]),
    Type.Object({
      schemaVersion: Type.Literal(5),
      viewLinks: Type.Array(ViewLinkSchema, { maxItems: 64 }),
      selections: Type.Array(LocalSelectionSchema, { maxItems: 64 }),
      workspace: WorkspaceSchema,
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

export type WorkspaceDocumentV5 = Static<typeof WorkspaceDocumentV5Schema>;
