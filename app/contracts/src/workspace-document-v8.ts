import { Type } from '@sinclair/typebox';
import { AgentIdSchema, closed, CounterSchema, RequestIdSchema } from './identity';
import { WorkspaceV8Schema as WorkspaceSchema } from './document-content-v8';
import { DecisionIdSchema } from './identity';
import { DraftAttachmentV8Schema as DraftAttachmentSchema } from './evidence-v8';
import { CollaborationHistoryV8Schema as CollaborationHistorySchema } from './document-content-v8';
import { WorkspaceDocumentV1Schema } from './workspace-document-v1';
import {
  SharedReferenceV3Schema as SharedReferenceSchema,
  SharedReferenceIdSchema,
} from './shared-reference-v3';
import { LocalSelectionV3Schema as LocalSelectionSchema } from './reference-v3';
import { ViewLinkSchema } from './tabular-view';

/** v8 adds saved Run/log navigation state; evidence bytes are unchanged. */
export const WorkspaceDocumentV8Schema = Type.Composite(
  [
    Type.Omit(WorkspaceDocumentV1Schema, [
      'schemaVersion',
      'discussion',
      'collaboration',
      'workspace',
      'selections',
    ]),
    Type.Object({
      schemaVersion: Type.Literal(8),
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
