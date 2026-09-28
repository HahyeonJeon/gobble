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
  DraftAttachmentV6Schema as DraftAttachmentSchema,
  CollaborationHistoryV6Schema as CollaborationHistorySchema,
  SharedReferenceV6Schema as SharedReferenceSchema,
  WorkspaceV6Schema as WorkspaceSchema,
} from './document-content-v6';
/** Frozen v6: exact plot context and original v2 references. */
export const WorkspaceDocumentV6Schema = Type.Composite(
  [
    Type.Omit(WorkspaceDocumentV1Schema, [
      'schemaVersion',
      'discussion',
      'collaboration',
      'workspace',
      'selections',
    ]),
    Type.Object({
      schemaVersion: Type.Literal(6),
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

export type WorkspaceDocumentV6 = Static<typeof WorkspaceDocumentV6Schema>;
