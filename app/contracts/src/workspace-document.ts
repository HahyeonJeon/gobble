import { SavedReportRecordSchema } from './run-report';
import { CreationContextSchema, CreationMarkSchema } from './creation-review';
import {
  PipelineReviewContextSchema,
  PipelineReviewMarkSchema,
  validatePipelineReviewContext,
} from './pipeline-proposal';
import { Type, type Static } from '@sinclair/typebox';
import { AgentIdSchema, closed, CounterSchema, ProjectIdSchema, RequestIdSchema } from './identity';
import { WorkspaceSchema } from './workspace';
import { DecisionIdSchema } from './identity';
import { validateQuestions } from './question-validation';
import { DraftAttachmentSchema } from './evidence';
import { validateEvidenceDocument } from './evidence-validation';
import { CollaborationHistorySchema } from './collaboration';
import { WorkspaceDocumentV1Schema } from './workspace-document-v1';
import { SharedReferenceSchema, SharedReferenceIdSchema } from './shared-context';
import { LocalSelectionSchema } from './reference-target';
import { ViewLinkSchema } from './tabular-view';
import { validateViewLinks } from './view-links';
import { sameResource } from './resource';
import { validateReferencePresentation } from './reference-presentation';
import {
  ContractValidationError,
  parse,
  parseWorkspace,
  validateSelection,
  validateEvidence,
} from './validation';

/** v21 adds whole-report addressed evidence and references. */
export const WorkspaceDocumentSchema = Type.Composite(
  [
    Type.Omit(WorkspaceDocumentV1Schema, [
      'schemaVersion',
      'discussion',
      'collaboration',
      'workspace',
      'selections',
    ]),
    Type.Object({
      schemaVersion: Type.Literal(21),
      savedReports: Type.Optional(Type.Array(SavedReportRecordSchema, { maxItems: 128 })),
      creationMarks: Type.Optional(Type.Array(CreationMarkSchema, { maxItems: 64 })),
      pipelineReviewMarks: Type.Optional(Type.Array(PipelineReviewMarkSchema, { maxItems: 64 })),
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
          pipelineReview: Type.Optional(PipelineReviewContextSchema),
          pipelineCreation: Type.Optional(CreationContextSchema),
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

export const WindowStateSchema = Type.Object(
  {
    schemaVersion: Type.Literal(1),
    activeProjectId: Type.Union([ProjectIdSchema, Type.Null()]),
    bounds: Type.Union([
      Type.Null(),
      Type.Object(
        {
          x: Type.Integer({ minimum: -100000, maximum: 100000 }),
          y: Type.Integer({ minimum: -100000, maximum: 100000 }),
          width: Type.Integer({ minimum: 640, maximum: 20000 }),
          height: Type.Integer({ minimum: 480, maximum: 20000 }),
        },
        closed,
      ),
    ]),
  },
  closed,
);
export type WindowState = Static<typeof WindowStateSchema>;
export type WorkspaceDocument = Static<typeof WorkspaceDocumentSchema>;

export function parseWorkspaceDocument(input: unknown): WorkspaceDocument {
  const doc = parse(WorkspaceDocumentSchema, input);
  const workspace = parseWorkspace(doc.workspace);
  const fail = (message: string): never => {
    throw new ContractValidationError(message);
  };
  if (
    workspace.layout.kind === 'single' &&
    (doc.activePane !== 'primary' || doc.maximizedPane !== null)
  )
    fail('A single pane cannot address another pane.');
  const reports = doc.savedReports ?? [];
  if (
    new Set(reports.map((r) => r.asset.hash)).size !== reports.length ||
    reports.some((r) => r.projectId !== workspace.projectId)
  )
    fail('Saved reports must be unique and belong to this Project.');
  for (const surface of workspace.surfaces) {
    if (surface.resource.kind === 'report') {
      const saved = surface.resource.saved;
      if (
        !reports.some(
          (r) => r.asset.hash === saved.asset.hash && JSON.stringify(r) === JSON.stringify(saved),
        )
      )
        fail('Report View requires its retained Project record.');
    }
  }
  const surfaces = new Map(workspace.surfaces.map((surface) => [surface.surfaceId, surface]));
  if (
    doc.titles.length !== surfaces.size ||
    new Set(doc.titles.map((item) => item.surfaceId)).size !== surfaces.size ||
    doc.titles.some((item) => !surfaces.has(item.surfaceId))
  )
    fail('Every surface needs one title.');
  if (
    doc.chat.recipientAgentId !== null &&
    !workspace.agents.some((agent) => agent.agentId === doc.chat.recipientAgentId)
  )
    fail('Chat recipient is not attached to this Project.');
  if (new Set(doc.selections.map((item) => item.surfaceId)).size !== doc.selections.length)
    fail('Selections must be unique per surface.');
  for (const { surfaceId, evidence } of doc.selections) {
    const surface = surfaces.get(surfaceId);
    if (
      evidence.projectId !== workspace.projectId ||
      !surface ||
      !sameResource(evidence.resource, surface.resource)
    )
      fail('Selection belongs to another surface.');
    if (evidence.selection) {
      validateSelection(evidence.selection);
      if (
        surface?.view !== evidence.selection.kind &&
        !(surface?.view === 'scatter' && evidence.selection.kind === 'table') &&
        !(surface?.view === 'log' && ['text', 'log-text'].includes(evidence.selection.kind)) &&
        !(
          surface?.view === 'run' &&
          ['run-task', 'run-group', 'run-dependency'].includes(evidence.selection.kind)
        )
      )
        fail('Selection does not match the displayed content.');
    }
  }
  if (new Set(doc.receipts.map((item) => item.requestId)).size !== doc.receipts.length)
    fail('Request receipts must be unique.');
  validateViewLinks(doc);
  const submissions = doc.collaboration?.submissions ?? [];
  if (doc.chat.pipelineReview && doc.chat.pipelineCreation)
    fail('Choose one Pipeline discussion context.');
  if (doc.chat.pipelineReview) validatePipelineReviewContext(doc.chat.pipelineReview);
  for (const item of submissions) {
    if (item.pipelineReview && item.pipelineCreation) fail('Conflicting Pipeline contexts.');
    if (item.allowPipelineCreation && item.pipelineCreation?.kind !== 'draft')
      fail('Creation authoring requires draft intent.');
    if (item.pipelineReview) validatePipelineReviewContext(item.pipelineReview);
    if (item.allowPipelineProposal && (!item.pipelineReview || item.pipelineReview.proposalId))
      fail('Source proposal authority requires a current Pipeline context.');
  }
  for (const mark of doc.creationMarks ?? []) {
    if (
      mark.context.kind !== 'candidate' ||
      !mark.context.target ||
      !workspace.agents.some((a) => a.agentId === mark.agentId) ||
      !submissions.some(
        (s) =>
          s.requestId === mark.submissionId &&
          s.agentId === mark.agentId &&
          s.pipelineCreation?.draftId === mark.context.draftId,
      )
    )
      fail('A creation reference needs an exact addition and originating submission.');
  }
  for (const mark of doc.pipelineReviewMarks ?? []) {
    validatePipelineReviewContext(mark.context);
    if (
      (mark.context.preparationId
        ? !mark.context.preparationSection
        : !mark.context.proposalId || !mark.context.changeId) ||
      !workspace.agents.some((a) => a.agentId === mark.agentId) ||
      !submissions.some((s) => s.requestId === mark.submissionId && s.agentId === mark.agentId)
    )
      fail(
        'A Pipeline reference requires an originating Agent submission and an exact change or preparation section.',
      );
  }
  if (new Set(submissions.map((item) => item.requestId)).size !== submissions.length)
    fail('Submission IDs must be unique.');
  const pending = new Set<string>();
  for (const item of submissions) {
    if (!item.text.trim() && !item.evidence?.length) fail('A message needs text or evidence.');
    if (!workspace.agents.some((agent) => agent.agentId === item.agentId))
      fail('Submission recipient is not attached to this Project.');
    if (['submitting', 'running', 'uncertain'].includes(item.state)) {
      if (pending.has(item.agentId)) fail('An agent may have only one pending submission.');
      pending.add(item.agentId);
    }
  }
  const references = doc.sharedReferences ?? [];
  if (new Set(references.map((item) => item.referenceId)).size !== references.length)
    fail('Shared references must have unique IDs.');
  if (
    doc.referenceReveal &&
    !references.some((item) => item.referenceId === doc.referenceReveal?.referenceId)
  )
    fail('Revealed reference is unavailable.');
  for (const reference of references) {
    validateEvidence(reference.evidence, workspace.projectId);
    validateReferencePresentation(reference.evidence, reference.presentation);
    if (reference.author.kind === 'agent') {
      const author = reference.author;
      if (!workspace.agents.some((agent) => agent.agentId === author.agentId))
        fail('Shared author is not attached.');
      if (
        !submissions.some(
          (item) =>
            item.requestId === reference.originSubmissionId && item.agentId === author.agentId,
        )
      )
        fail('Shared reference has no originating submission.');
    } else if (reference.originSubmissionId)
      fail('A user reference cannot impersonate an Agent submission.');
  }
  validateEvidenceDocument(doc);
  validateQuestions(doc);
  return doc;
}
