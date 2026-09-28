import { Type, type Static } from '@sinclair/typebox';
import {
  AgentIdSchema,
  closed,
  CounterSchema,
  RequestIdSchema,
  DecisionIdSchema,
} from './identity';
import { WorkspaceV8Schema as WorkspaceSchema } from './document-content-v8';
import { DraftAttachmentV8Schema as DraftAttachmentSchema } from './evidence-v8';
import { CollaborationHistoryV8Schema as CollaborationHistorySchema } from './document-content-v8';
import { WorkspaceDocumentV1Schema } from './workspace-document-v1';
import {
  SharedReferenceV3Schema as SharedReferenceSchema,
  SharedReferenceIdSchema,
} from './shared-reference-v3';
import { LocalSelectionV3Schema as LocalSelectionSchema } from './reference-v3';
import { ViewLinkSchema } from './tabular-view';
import { SurfaceV7Schema } from './surface-v7';
/** Frozen v7, before Run/log navigation state. */
export const WorkspaceDocumentV7Schema = Type.Composite(
  [
    Type.Omit(WorkspaceDocumentV1Schema, [
      'schemaVersion',
      'discussion',
      'collaboration',
      'workspace',
      'selections',
    ]),
    Type.Object({
      schemaVersion: Type.Literal(7),
      viewLinks: Type.Array(ViewLinkSchema, { maxItems: 64 }),
      selections: Type.Array(LocalSelectionSchema, { maxItems: 64 }),
      workspace: Type.Object(
        { ...WorkspaceSchema.properties, surfaces: Type.Array(SurfaceV7Schema, { maxItems: 64 }) },
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

export type WorkspaceDocumentV7 = Static<typeof WorkspaceDocumentV7Schema>;
