import { PresentationState } from './presentation-state';
import { findCurrentDependency } from './current-dependency';
import {
  prepareObservedReference,
  type ObservedReferenceSession,
} from './observed-reference-views';
import { createHash, randomUUID } from 'node:crypto';
import {
  parseWorkspaceDocument,
  observedTargetVisible,
  isTabularAction,
  type SurfaceData,
  viewLinkFor,
  isQuestion,
  type Question,
  resourceKey,
  type RenderAcknowledgment,
  type SurfaceLoad,
  type WorkspaceBootstrap,
  type WorkspaceCommand,
  type WorkspaceDocument,
  type WindowState,
  type PaneId,
  type WorkspacePresentationInput,
} from '@gobble/contracts';
import {
  selectReply,
  cancelReply,
  dismissQuestion,
  validateReplyCommit,
  recordReply,
  requireQuestion,
} from '../questions/model';
import type { ReplyCommit, QuestionWorkspace } from '../questions/ports';
import { emptyWorkspace, reusableSurface, transition } from './model';
import { AppProblem } from '../problem';
import { publishReference, requireEvidence, sharedReference } from '../shared-context/references';
import { checkSelection } from './selection';
import {
  attachEvidence,
  consumeDraft,
  detachEvidence,
  invalidateAttachments,
} from '../evidence/draft';
import type { DraftConsumption, EvidenceDraft, EvidenceWorkspace } from '../evidence/ports';
import type { EvidenceCapture } from '../evidence/capture';
import { retainedEvidenceHashes } from '../evidence/references';
import { prepareReferenceView, type ReferenceSession } from './reference-views';
import { applyTabularAction } from './tabular-commands';
import type { SurfaceRequest } from './render-session';
import type { WorkspaceResources } from './service';
import type { WorkspaceStorage } from './storage';
import {
  collaborationOf,
  type CollaborationStore,
  type ProjectCollaboration,
} from '../collaboration/store';

const defaultWindow = (): WindowState => ({
  schemaVersion: 1,
  activeProjectId: null,
  bounds: null,
});
const canonical = (value: unknown): string => {
  if (Array.isArray(value)) return '[' + value.map(canonical).join(',') + ']';
  if (value !== null && typeof value === 'object')
    return (
      '{' +
      Object.entries(value)
        .sort(([a], [b]) => a.localeCompare(b))
        .map(([key, item]) => JSON.stringify(key) + ':' + canonical(item))
        .join(',') +
      '}'
    );
  return JSON.stringify(value);
};

export class WorkspaceController
  implements CollaborationStore, EvidenceWorkspace, QuestionWorkspace
{
  private readonly listeners = new Set<(document: WorkspaceDocument) => void>();
  private queue: Promise<unknown> = Promise.resolve();
  private documents = new Map<string, WorkspaceDocument>();
  private pendingEvidenceCleanup = new Set<string>();
  private windowState = defaultWindow();
  private windowWritable = true;
  private readonly presentation: PresentationState;
  private activeProjectId: string | null = null;
  private closing = false;
  private visiblePanes: PaneId[] = [];
  private notice: string | null = null;
  private modalTokens = new Set<string>();
  private uiEpoch = 0;
  private evidenceEpoch = 0;
  private closedResources = new Map<string, number>();
  private closeSequence = 0;
  unsaved = false;

  constructor(
    private readonly storage: WorkspaceStorage,
    private readonly resources: WorkspaceResources,
    private readonly windowAvailable: () => boolean = () => false,
    private readonly observedEvidence?: EvidenceCapture,
  ) {
    this.presentation = new PresentationState(resources);
  }

  private enqueue<T>(work: () => Promise<T>): Promise<T> {
    if (this.closing)
      return Promise.reject(
        new AppProblem('runtime_unavailable', 'The workspace is shutting down.'),
      );
    const result = this.queue.then(work);
    this.queue = result.catch(() => {});
    return result;
  }
  async initialize(): Promise<WindowState> {
    try {
      this.windowState = (await this.storage.window.read()) ?? defaultWindow();
    } catch {
      this.windowWritable = false;
      this.notice = 'Window state could not be restored. Its saved file has been preserved.';
    }
    return structuredClone(this.windowState);
  }
  disconnect(): void {
    this.presentation.disconnect();
    this.evidenceEpoch++;
    this.modalTokens.clear();
    this.uiEpoch++;
    this.visiblePanes = [];
  }
  interaction(input: { token: string; blocked: boolean }): Promise<{ accepted: true }> {
    if (input.blocked) {
      if (this.modalTokens.size >= 16 && !this.modalTokens.has(input.token))
        return Promise.reject(new AppProblem('unsupported', 'Too many dialogs are open.'));
      this.modalTokens.add(input.token);
    } else this.modalTokens.delete(input.token);
    this.uiEpoch++;
    return Promise.resolve({ accepted: true });
  }
  /** UI availability changes invalidate in-flight observations even when focus later returns. */
  invalidatePresentation(): void {
    this.uiEpoch++;
  }
  requireSharedUI(projectId: string): void {
    this.requireActive(projectId);
    if (!this.windowAvailable() || this.modalTokens.size)
      throw new AppProblem(
        'runtime_unavailable',
        'The Project window is unavailable. Reveal it and close dialogs before sharing views.',
      );
  }
  sharedPresentation(projectId: string, surfaceId: string): 'ready' | 'loading' | 'hidden' {
    try {
      this.requireSharedUI(projectId);
    } catch {
      return 'hidden';
    }
    return this.presentation.rendering.presentation(surfaceId);
  }
  sharedReferenceView(projectId: string, surfaceId: string) {
    if (this.activeProjectId !== projectId) return undefined;
    const value = this.presentation.referenceViews.forSurface(surfaceId);
    return value ? structuredClone(value) : undefined;
  }
  get dismissalSequence(): number {
    return this.closeSequence;
  }
  dismissedSince(
    projectId: string,
    resource: import('@gobble/contracts').ResourceRef,
    sequence: number,
  ): boolean {
    return (this.closedResources.get(projectId + '/' + resourceKey(resource)) ?? 0) > sequence;
  }
  readShared(projectId: string): Promise<WorkspaceDocument> {
    return this.enqueue(async () => structuredClone(await this.registeredDocument(projectId)));
  }
  changeShared<T>(
    projectId: string,
    guard: () => void,
    change: (doc: WorkspaceDocument) => T,
    tabularSource?: { surfaceId: string; data: SurfaceData },
  ): Promise<T> {
    return this.enqueue(async () => {
      const doc = structuredClone(await this.registeredDocument(projectId));
      this.requireSharedUI(projectId);
      guard();
      const value = change(doc);
      doc.workspace.revision++;
      // No I/O occurs between authority validation and initiating the atomic write.
      const saved = await this.commit(doc);
      if (tabularSource) {
        const link = viewLinkFor(saved, tabularSource.surfaceId);
        if (link) this.presentation.tabularSnapshots.seed(projectId, link, tabularSource.data);
      }
      this.pruneLeases(saved);
      return value;
    });
  }
  captureShared(projectId: string, surfaceId: string, guard: () => void) {
    return this.enqueue(async () => {
      const doc = await this.registeredDocument(projectId);
      this.requireSharedUI(projectId);
      guard();
      const surface = doc.workspace.surfaces.find((item) => item.surfaceId === surfaceId);
      if (!surface) throw new AppProblem('not_found', 'This view is no longer open.');
      const snapshot = this.presentation.rendering.snapshot(surfaceId);
      const { acknowledgment } = snapshot;
      const isPdf = snapshot.data.kind === 'file' && snapshot.data.value.content.kind === 'pdf';
      const isNotebook =
        snapshot.data.kind === 'file' && snapshot.data.value.content.kind === 'notebook';
      if (isNotebook && snapshot.observedReferenceView)
        throw new AppProblem('unsupported', 'Return to the Notebook reader before observing it.');
      const notebookGeneration = snapshot.notebookViewport?.generation;
      if (isNotebook)
        this.presentation.rendering.assertNotebookViewport(acknowledgment, notebookGeneration);
      const viewportGeneration = snapshot.pdfViewport?.generation;
      if (isPdf) this.presentation.rendering.assertPdfViewport(acknowledgment, viewportGeneration);
      return {
        ...snapshot,
        link: snapshot.link
          ? {
              ...snapshot.link,
              rowKeys: viewLinkFor(doc, surfaceId)?.rowKeys ?? snapshot.link.rowKeys,
            }
          : undefined,
        assert: this.observationGuard(
          projectId,
          () => {
            guard();
            if (isNotebook)
              this.presentation.rendering.assertNotebookViewport(
                acknowledgment,
                notebookGeneration,
              );
            if (isPdf)
              this.presentation.rendering.assertPdfViewport(acknowledgment, viewportGeneration);
          },
          snapshot.acknowledgment,
          this.uiEpoch,
        ),
      };
    });
  }
  /** Captures only identity, not the potentially large loaded image. */
  private observationGuard(
    projectId: string,
    guard: () => void,
    acknowledgment: RenderAcknowledgment,
    epoch: number,
  ): () => void {
    return () => {
      this.requireSharedUI(projectId);
      guard();
      if (epoch !== this.uiEpoch)
        throw new AppProblem('stale_revision', 'The presentation changed during observation.');
      this.presentation.rendering.assertAcknowledgment(acknowledgment);
    };
  }
  notebookImage(input: import('@gobble/contracts').NotebookImageInput) {
    return this.enqueue(async () => {
      const { acknowledgment: ack, target } = input;
      this.requireActive(ack.projectId);
      if (target.projectId !== ack.projectId || target.origin?.surfaceId !== ack.surfaceId)
        throw new AppProblem('forbidden', 'This image belongs to another view.');
      this.presentation.rendering.assertAcknowledgment(ack);
      if (!this.resources.notebookImage)
        throw new AppProblem('unsupported', 'Notebook images are unavailable.');
      return this.resources.notebookImage(
        ack.surfaceId,
        target,
        this.presentation.rendering.selectableData(ack.surfaceId),
      );
    });
  }
  notebookViewport(
    input: import('@gobble/contracts').NotebookViewportInput,
  ): Promise<{ accepted: true }> {
    return this.enqueue(async () => {
      const ack = input.acknowledgment;
      this.requireActive(ack.projectId);
      this.presentation.rendering.assertAcknowledgment(ack);
      const data = this.presentation.rendering.selectableData(ack.surfaceId);
      if (!this.resources.validateNotebook)
        throw new AppProblem('unsupported', 'Notebook validation is unavailable.');
      if (data.kind !== 'file' || data.value.content.kind !== 'notebook')
        throw new AppProblem('invalid_request', 'A Notebook reader is required.');
      for (const selection of input.viewport.parts)
        this.resources.validateNotebook(
          ack.surfaceId,
          {
            schemaVersion: 6,
            projectId: ack.projectId,
            resource: { kind: 'file', resourceId: data.value.resourceId },
            origin: { surfaceId: ack.surfaceId },
            dataRevision: ack.dataRevision,
            selection,
          },
          data,
        );
      return this.presentation.rendering.setNotebookViewport(input);
    });
  }
  pdfViewport(input: import('@gobble/contracts').PdfViewportInput): Promise<{ accepted: true }> {
    return this.enqueue(async () => {
      this.requireActive(input.acknowledgment.projectId);
      return this.presentation.rendering.setPdfViewport(input);
    });
  }
  present(input: WorkspacePresentationInput): Promise<{ accepted: true }> {
    return this.enqueue(async () => {
      this.requireActive(input.projectId);
      if (input.rendererSessionId !== this.presentation.rendering.id)
        throw new AppProblem('stale_revision', 'This window session is no longer current.');
      const doc = await this.document(input.projectId);
      if (
        input.visiblePanes.some(
          (pane) =>
            (pane === 'secondary' && doc.workspace.layout.kind !== 'split') ||
            (doc.maximizedPane !== null && doc.maximizedPane !== pane),
        )
      )
        throw new AppProblem('stale_revision', 'The workspace layout changed.');
      this.visiblePanes = [...input.visiblePanes];
      this.pruneLeases(doc);
      return { accepted: true };
    });
  }
  private pruneLeases(doc: WorkspaceDocument): void {
    this.presentation.reconcile(doc, this.visiblePanes);
  }
  async flush(): Promise<void> {
    await this.queue;
  }
  async stop(): Promise<void> {
    this.closing = true;
    this.disconnect();
    await this.queue;
  }
  saveBounds(bounds: NonNullable<WindowState['bounds']>): Promise<void> {
    return this.enqueue(async () => {
      this.windowState = { ...this.windowState, bounds };
      await this.saveWindow();
    });
  }
  private async saveWindow(): Promise<void> {
    if (!this.windowWritable) return;
    try {
      await this.storage.window.write(this.windowState);
    } catch {
      this.windowWritable = false;
      this.notice = 'Window position could not be saved. Project layouts are stored separately.';
      this.unsaved = true;
    }
  }
  private async document(projectId: string): Promise<WorkspaceDocument> {
    let doc = this.documents.get(projectId);
    if (!doc) {
      doc = (await this.storage.project(projectId).read()) ?? emptyWorkspace(projectId);
      this.documents.set(projectId, doc);
      this.pendingEvidenceCleanup.add(projectId);
    }
    return doc;
  }
  private async commit(doc: WorkspaceDocument): Promise<WorkspaceDocument> {
    parseWorkspaceDocument(doc);
    const projectId = doc.workspace.projectId;
    const previous = this.documents.get(projectId);
    const before = retainedEvidenceHashes(previous ? [previous] : []);
    const after = retainedEvidenceHashes([doc]);
    const referencesChanged =
      before.size !== after.size || [...before].some((hash) => !after.has(hash));
    try {
      await this.storage.project(doc.workspace.projectId).write(doc);
    } catch (error) {
      this.unsaved = true;
      throw error;
    }
    this.documents.set(doc.workspace.projectId, doc);
    const cleanupPending = this.pendingEvidenceCleanup.delete(projectId);
    // One later successful write also advances the recovery backup. Ordinary keystrokes
    // must not repeatedly scan every recovery document and sync the asset directory.
    if (referencesChanged) this.pendingEvidenceCleanup.add(projectId);
    if (referencesChanged || cleanupPending) await this.reclaimObserved(projectId);
    if (doc.workspace.projectId === this.activeProjectId) {
      for (const listener of this.listeners) {
        try {
          listener(structuredClone(doc));
        } catch {
          /* A closed window does not invalidate a save. */
        }
      }
    }
    return structuredClone(doc);
  }
  private async reclaimObserved(projectId: string): Promise<void> {
    if (!this.observedEvidence) return;
    try {
      const roots = await this.storage.retainedDocuments(projectId);
      await this.observedEvidence.reclaim(projectId, retainedEvidenceHashes(roots));
    } catch {
      // A missing, corrupt or externally changed recovery root forbids deletion.
      // Keep the immutable bytes within quota; cleanup failure never undoes a saved draft.
    }
  }
  onDocument(listener: (document: WorkspaceDocument) => void): () => void {
    this.listeners.add(listener);
    return () => this.listeners.delete(listener);
  }
  private async registeredDocument(projectId: string): Promise<WorkspaceDocument> {
    if (!(await this.resources.projects()).some((project) => project.projectId === projectId))
      throw new AppProblem('not_found', 'This Project is no longer registered.');
    return this.document(projectId);
  }
  readQuestions(projectId: string) {
    return this.enqueue(async () => {
      const doc = await this.registeredDocument(projectId);
      return structuredClone({
        questions: doc.workspace.decisions.filter(isQuestion),
        submissions: (doc.collaboration?.submissions ?? []).map(
          ({ requestId, agentId, threadId, state, evidence, replyToQuestionId }) => ({
            requestId,
            agentId,
            threadId,
            state,
            ...(evidence ? { evidence } : {}),
            ...(replyToQuestionId ? { replyToQuestionId } : {}),
          }),
        ),
      });
    });
  }
  changeQuestions(
    projectId: string,
    guard: () => void,
    change: (questions: Question[]) => Question[],
  ): Promise<void> {
    return this.enqueue(async () => {
      const doc = structuredClone(await this.registeredDocument(projectId));
      guard();
      doc.workspace.decisions = [
        ...doc.workspace.decisions.filter((item) => !isQuestion(item)),
        ...change(doc.workspace.decisions.filter(isQuestion)),
      ];
      doc.workspace.revision++;
      await this.commit(doc);
    });
  }
  readQuestionEvidence(projectId: string, questionId: string, attachmentId: string) {
    return this.enqueue(async () => {
      this.requireActive(projectId);
      const doc = await this.registeredDocument(projectId);
      const question = requireQuestion(doc.workspace.decisions.filter(isQuestion), questionId);
      const manifest = question.evidence.find((item) => item.attachmentId === attachmentId);
      if (!manifest)
        throw new AppProblem(
          'not_found',
          'Evidence unavailable: this question has no such attachment.',
        );
      return structuredClone(manifest);
    });
  }
  /** Trusted User intent may arrive as a dialog closes. It changes only the
   * private composer context; Agent view operations retain the shared UI gate. */
  selectPipelineReview(
    projectId: string,
    context: import('@gobble/contracts').PipelineReviewContext | null,
  ): Promise<void> {
    return this.enqueue(async () => {
      this.requireActive(projectId);
      const doc = structuredClone(await this.registeredDocument(projectId));
      if (context) {
        doc.chat.pipelineReview = context;
        delete doc.chat.pipelineCreation;
      } else delete doc.chat.pipelineReview;
      doc.chat.collapsed = false;
      doc.workspace.revision++;
      await this.commit(doc);
    });
  }
  selectPipelineCreation(
    projectId: string,
    context: import('@gobble/contracts').CreationContext | null,
  ): Promise<void> {
    return this.enqueue(async () => {
      this.requireActive(projectId);
      const doc = structuredClone(await this.registeredDocument(projectId));
      if (context) {
        doc.chat.pipelineCreation = context;
        delete doc.chat.pipelineReview;
      } else delete doc.chat.pipelineCreation;
      doc.chat.collapsed = false;
      doc.workspace.revision++;
      await this.commit(doc);
    });
  }
  readCollaboration(projectId: string): Promise<ProjectCollaboration> {
    return this.enqueue(async () => collaborationOf(await this.registeredDocument(projectId)));
  }
  readEvidenceDraft(projectId: string): Promise<EvidenceDraft> {
    return this.enqueue(async () => {
      this.requireActive(projectId);
      const doc = await this.registeredDocument(projectId);
      return structuredClone({
        rendererSessionId: this.presentation.rendering.id + ':' + this.evidenceEpoch,
        attachmentRevision: doc.chat.attachmentRevision ?? 0,
        attachments: doc.chat.attachments ?? [],
        agent:
          doc.workspace.agents.find((item) => item.agentId === doc.chat.recipientAgentId) ?? null,
      });
    });
  }
  assertEvidenceSession(projectId: string, rendererSessionId: string): void {
    this.requireActive(projectId);
    if (rendererSessionId !== this.presentation.rendering.id + ':' + this.evidenceEpoch)
      throw new AppProblem(
        'stale_revision',
        'The workspace session changed. Prepare the attachments again.',
      );
  }
  /** Addressed reads survive Pane closure, but not a Project switch or renderer disconnect. */
  sentEvidenceGuard(projectId: string): () => void {
    this.requireActive(projectId);
    const epoch = this.evidenceEpoch;
    return () => {
      this.requireActive(projectId);
      if (epoch !== this.evidenceEpoch)
        throw new AppProblem(
          'stale_revision',
          'This report read belongs to an earlier Project session.',
        );
    };
  }
  readSentEvidence(projectId: string, requestId: string, attachmentId: string) {
    return this.enqueue(async () => {
      this.requireActive(projectId);
      const doc = await this.registeredDocument(projectId);
      const manifest = doc.collaboration?.submissions
        .find((item) => item.requestId === requestId)
        ?.evidence?.find((item) => item.attachmentId === attachmentId);
      if (!manifest)
        throw new AppProblem(
          'not_found',
          'Evidence unavailable: this message has no such attachment.',
        );
      return structuredClone(manifest);
    });
  }
  changeCollaboration(
    projectId: string,
    change: (current: ProjectCollaboration) => ProjectCollaboration,
    consumedDraft?: string | DraftConsumption,
    reply?: ReplyCommit,
  ): Promise<void> {
    return this.enqueue(async () => {
      const doc = structuredClone(await this.registeredDocument(projectId));
      const previousConfiguration = JSON.stringify(
        doc.workspace.agents.find((item) => item.agentId === doc.chat.recipientAgentId)
          ?.configuration,
      );
      const next = change(collaborationOf(doc));
      doc.workspace.agents = next.agents;
      doc.collaboration = next.history;
      if (
        previousConfiguration !==
        JSON.stringify(
          next.agents.find((item) => item.agentId === doc.chat.recipientAgentId)?.configuration,
        )
      )
        invalidateAttachments(doc);
      if (consumedDraft !== undefined) {
        validateReplyCommit(doc, reply);
        consumeDraft(doc, consumedDraft);
        const submitted = next.history.submissions.at(-1);
        if (
          submitted?.pipelineReview &&
          JSON.stringify(submitted.pipelineReview) === JSON.stringify(doc.chat.pipelineReview)
        )
          delete doc.chat.pipelineReview;
        if (
          submitted?.pipelineCreation &&
          JSON.stringify(submitted.pipelineCreation) === JSON.stringify(doc.chat.pipelineCreation)
        )
          delete doc.chat.pipelineCreation;
        if (reply) recordReply(doc, reply);
      }
      doc.workspace.revision += 1;
      await this.commit(doc);
    });
  }
  private async bootstrap(): Promise<WorkspaceBootstrap> {
    const projects = await this.resources.projects();
    let document: WorkspaceDocument | null = null;
    if (
      this.activeProjectId &&
      projects.some((project) => project.projectId === this.activeProjectId)
    ) {
      try {
        document = await this.document(this.activeProjectId);
      } catch (error) {
        this.notice =
          error instanceof AppProblem ? error.message : 'This workspace could not be restored.';
        this.activeProjectId = null;
      }
    } else this.activeProjectId = null;
    return structuredClone({
      rendererSessionId: this.presentation.rendering.id,
      projects,
      document,
      notice: this.notice,
    });
  }
  connect(): Promise<WorkspaceBootstrap> {
    return this.enqueue(async () => {
      this.disconnect();
      this.activeProjectId = this.windowState.activeProjectId;
      return this.bootstrap();
    });
  }
  read(projectId: string): Promise<WorkspaceDocument> {
    return this.enqueue(async () => {
      this.requireActive(projectId);
      return structuredClone(await this.document(projectId));
    });
  }
  openProject(projectId: string | null): Promise<WorkspaceBootstrap> {
    return this.enqueue(async () => {
      const projects = await this.resources.projects();
      if (projectId && !projects.some((project) => project.projectId === projectId))
        throw new AppProblem('not_found', 'This Project is no longer registered.');
      if (projectId) await this.commit(await this.document(projectId));
      this.activeProjectId = projectId;
      this.evidenceEpoch++;
      this.visiblePanes = [];
      this.presentation.clear();
      this.windowState = { ...this.windowState, activeProjectId: projectId };
      await this.saveWindow();
      return this.bootstrap();
    });
  }
  private requireActive(projectId: string): void {
    if (projectId !== this.activeProjectId)
      throw new AppProblem('forbidden', 'Open this Project before changing its workspace.');
  }
  command(input: WorkspaceCommand): Promise<WorkspaceDocument> {
    return this.enqueue(async () => {
      this.requireActive(input.projectId);
      const current = await this.document(input.projectId);
      const action = input.action;
      if (
        ['openScatter', 'openLinkedTable', 'scatterSettings', 'scatterViewport'].includes(
          action.kind,
        ) ||
        (action.kind === 'duplicateView' &&
          current.workspace.surfaces.some(
            (surface) => surface.surfaceId === action.surfaceId && surface.view === 'scatter',
          ))
      )
        throw new AppProblem(
          'unsupported',
          'CSV chart creation has been removed. Open the source table.',
        );

      const hash = createHash('sha256').update(canonical(input.action)).digest('hex');
      const receipt = current.receipts.find((item) => item.requestId === input.requestId);
      if (receipt) {
        if (receipt.digest !== hash)
          throw new AppProblem(
            'request_conflict',
            'This request ID belongs to another workspace change.',
          );
        return structuredClone(current);
      }
      if (current.workspace.revision !== input.expectedRevision)
        throw new AppProblem(
          'stale_revision',
          'The workspace changed. Refresh it before trying this action again.',
        );
      if (
        'surfaceId' in action &&
        action.kind === 'select' &&
        current.workspace.surfaces.some(
          (s) => s.surfaceId === action.surfaceId && s.view === 'report',
        )
      )
        throw new AppProblem(
          'unsupported',
          'Reports support whole-report attachments and pointers, not region selections.',
        );
      if (action.kind === 'open' && action.resource.kind === 'report') {
        const saved = action.resource.saved;
        if (!(current.savedReports ?? []).some((r) => canonical(r) === canonical(saved)))
          throw new AppProblem('not_found', 'This saved report is not retained in this Project.');
      }
      let opened;
      if (input.action.kind === 'open') {
        if (!reusableSurface(current, input.action)) {
          const metadata = await this.resources.describe(input.projectId, input.action.resource);
          opened = { ...metadata, surfaceId: 'srf_' + randomUUID() };
        }
      }
      if (input.action.kind === 'duplicateView') opened = { surfaceId: 'srf_' + randomUUID() };
      if (action.kind === 'reportNavigate') {
        const ack = action.acknowledgment;
        if (ack.projectId !== input.projectId || ack.surfaceId !== action.surfaceId)
          throw new AppProblem('forbidden', 'This section belongs to another View.');
        this.presentation.rendering.assertAcknowledgment(ack);
        const data = this.presentation.rendering.selectableData(action.surfaceId);
        if (
          data.kind !== 'report' ||
          !data.value.content.modules.some((m) => m.id === action.moduleId)
        )
          throw new AppProblem('invalid_request', 'This section is outside the saved report.');
      }
      if (input.action.kind === 'notebookNavigate') {
        const { acknowledgment: ack, surfaceId, navigation } = input.action;
        if (ack.projectId !== input.projectId || ack.surfaceId !== surfaceId)
          throw new AppProblem('forbidden', 'This Notebook belongs to another view.');
        this.presentation.rendering.assertAcknowledgment(ack);
        const data = this.presentation.rendering.selectableData(surfaceId);
        if (
          data.kind !== 'file' ||
          data.value.content.kind !== 'notebook' ||
          navigation.page >= Math.max(1, Math.ceil(data.value.content.document.cells.length / 20))
        )
          throw new AppProblem('invalid_request', 'Cell page is outside this Notebook.');
      }
      if (input.action.kind === 'pdfNavigate') {
        const { acknowledgment: ack, surfaceId, navigation } = input.action;
        if (ack.projectId !== input.projectId || ack.surfaceId !== surfaceId)
          throw new AppProblem('forbidden', 'This page belongs to another view.');
        this.presentation.rendering.assertAcknowledgment(ack);
        const data = this.presentation.rendering.selectableData(surfaceId);
        if (
          data.kind !== 'file' ||
          data.value.content.kind !== 'pdf' ||
          navigation.pageIndex >= data.value.content.pageCount
        )
          throw new AppProblem('invalid_request', 'Page is outside this PDF.');
      }
      if (input.action.kind === 'select') {
        const selectedSurfaceId = input.action.surfaceId;
        const linked = viewLinkFor(current, selectedSurfaceId);
        const surface = current.workspace.surfaces.find(
          (item) => item.surfaceId === selectedSurfaceId,
        );
        const ack = input.action.acknowledgment;
        if (
          (linked ||
            surface?.view === 'scatter' ||
            (input.action.evidence?.schemaVersion ?? 0) >= 3) &&
          !ack
        )
          throw new AppProblem('stale_revision', 'Select from the current rendered view.');
        if (ack) {
          if (ack.projectId !== input.projectId || ack.surfaceId !== input.action.surfaceId)
            throw new AppProblem('forbidden', 'The render acknowledgment belongs to another view.');
          this.presentation.rendering.assertAcknowledgment(ack);
        }
      }
      if (input.action.kind === 'select' && input.action.evidence) {
        const evidence = input.action.evidence;
        const surfaceId = input.action.surfaceId;
        const surface = current.workspace.surfaces.find((item) => item.surfaceId === surfaceId);
        if (
          evidence.projectId !== input.projectId ||
          !surface ||
          resourceKey(surface.resource) !== resourceKey(evidence.resource)
        )
          throw new AppProblem('forbidden', 'This selection belongs to another view.');
        checkSelection(
          evidence,
          this.presentation.rendering.selectableData(input.action.surfaceId),
        );
        const snapshot = this.presentation.rendering.snapshot(surfaceId);
        if (evidence.schemaVersion === 6) {
          if (!this.resources.validateNotebook)
            throw new AppProblem('unsupported', 'Notebook selection is unavailable.');
          if (evidence.origin?.surfaceId !== surfaceId)
            throw new AppProblem('forbidden', 'Select from this Notebook view.');
          this.resources.validateNotebook(surfaceId, evidence, snapshot.data);
        }
        if (!observedTargetVisible(snapshot.surface, snapshot.data, evidence))
          throw new AppProblem('stale_revision', 'Reveal this task or stream before selecting it.');
      }
      this.presentation.referenceViews.assertBaseAction(input.action);
      this.presentation.observedReferences.assertBaseAction(input.action);
      let next: WorkspaceDocument;
      let referenceSession: ReferenceSession | undefined;
      let observedSession: ObservedReferenceSession | undefined;
      let clearReference = false;
      let reloadSurfaceId: string | undefined;
      let referenceData: SurfaceData | undefined;
      let tabularData: SurfaceData | undefined;
      if (isTabularAction(action)) {
        if (action.kind === 'refreshLinked') {
          const surface = current.workspace.surfaces.find(
            (item) => item.surfaceId === action.surfaceId,
          );
          const link = viewLinkFor(current, action.surfaceId);
          if (!surface || !link || link.dataRevision !== action.dataRevision)
            throw new AppProblem(
              'stale_revision',
              'The linked source has changed. Reopen its current view.',
            );
          tabularData = await this.resources.read(input.projectId, surface.resource);
        } else {
          if (
            action.acknowledgment.projectId !== input.projectId ||
            action.acknowledgment.surfaceId !== action.surfaceId
          )
            throw new AppProblem('forbidden', 'This observation belongs to another view.');
          this.presentation.rendering.assertAcknowledgment(action.acknowledgment);
          tabularData = this.presentation.rendering.selectableData(action.surfaceId);
        }
        next = structuredClone(current);
        applyTabularAction(next, action, tabularData, input.requestId);
      } else if (
        action.kind === 'reply' ||
        action.kind === 'cancelReply' ||
        action.kind === 'dismissQuestion'
      ) {
        next = structuredClone(current);
        if (action.kind === 'reply') selectReply(next, action.questionId);
        else if (action.kind === 'cancelReply') cancelReply(next);
        else dismissQuestion(next, action.questionId);
      } else if (action.kind === 'attach') {
        requireEvidence(current, action.evidence, action.surfaceId);
        const snapshot = this.presentation.rendering.snapshot(action.surfaceId);
        checkSelection(action.evidence, snapshot.data);
        if (!observedTargetVisible(snapshot.surface, snapshot.data, action.evidence))
          throw new AppProblem(
            'stale_revision',
            'Reveal this task or stream before discussing it.',
          );
        if (
          snapshot.data.kind !== 'file' ||
          action.evidence.schemaVersion === 5 ||
          action.evidence.schemaVersion === 6
        ) {
          if (
            !action.acknowledgment ||
            action.acknowledgment.projectId !== input.projectId ||
            action.acknowledgment.surfaceId !== action.surfaceId
          )
            throw new AppProblem('stale_revision', 'Attach from the current rendered observation.');
          this.presentation.rendering.assertAcknowledgment(action.acknowledgment);
        }
        next = structuredClone(current);
        const attachment = {
          attachmentId: 'att_' + input.requestId.slice(4),
          evidence: action.evidence,
          label:
            current.titles.find((item) => item.surfaceId === action.surfaceId)?.title ?? 'View',
          createdAt: Date.now(),
        };
        if (
          snapshot.data.kind !== 'file' ||
          action.evidence.schemaVersion === 5 ||
          action.evidence.schemaVersion === 6
        ) {
          if (!this.observedEvidence)
            throw new AppProblem('unsupported', 'Observed evidence storage is unavailable.');
          await this.reclaimObserved(input.projectId);
          this.pendingEvidenceCleanup.add(input.projectId);
          const captured = await this.observedEvidence.capture(attachment, snapshot.data);
          this.requireActive(input.projectId);
          this.presentation.rendering.assertAcknowledgment(snapshot.acknowledgment);
          attachEvidence(next, captured);
        } else attachEvidence(next, attachment);
      } else if (action.kind === 'detach') {
        next = structuredClone(current);
        detachEvidence(next, action.attachmentId);
      } else if (action.kind === 'share') {
        if (action.evidence.schemaVersion === 6) {
          if (!this.resources.validateNotebook)
            throw new AppProblem('unsupported', 'Notebook validation is unavailable.');
          this.resources.validateNotebook(
            action.surfaceId,
            action.evidence,
            this.presentation.rendering.selectableData(action.surfaceId),
          );
        }
        if (action.evidence.schemaVersion === 4)
          throw new AppProblem(
            'unsupported',
            'Use Discuss for dependency evidence. Shared dependency marks are not available yet.',
          );
        requireEvidence(current, action.evidence, action.surfaceId);
        checkSelection(
          action.evidence,
          this.presentation.rendering.selectableData(action.surfaceId),
        );
        next = structuredClone(current);
        publishReference(
          next,
          {
            referenceId: 'ref_' + input.requestId.slice(4),
            evidence: action.evidence,
            author: { kind: 'user' },
            createdAt: Date.now(),
            note: action.note,
            retracted: false,
          },
          action.surfaceId,
        );
      } else if (action.kind === 'retract') {
        next = structuredClone(current);
        const reference = sharedReference(next, action.referenceId);
        reference.retracted = true;
        if (next.referenceReveal?.referenceId === action.referenceId) {
          if (this.presentation.observedReferences.has(next.referenceReveal.requestId))
            this.presentation.observedReferences.returnTo(next, next.referenceReveal.requestId);
          delete next.referenceReveal;
          clearReference = true;
        }
      } else if (action.kind === 'returnReferenceView') {
        next = structuredClone(current);
        if (this.presentation.observedReferences.has(action.referenceRequestId))
          this.presentation.observedReferences.returnTo(next, action.referenceRequestId);
        else this.presentation.referenceViews.requireReturn(action.referenceRequestId);
        delete next.referenceReveal;
        clearReference = true;
      } else if (action.kind === 'currentDependency') {
        const found = await findCurrentDependency(
          current,
          sharedReference(current, action.referenceId),
          this.resources,
        );
        next = found.document;
        reloadSurfaceId = found.surfaceId;
        clearReference = true;
      } else if (action.kind === 'currentTask') {
        const reference = sharedReference(current, action.referenceId);
        const { resource, selection } = reference.evidence;
        const instanceId =
          selection?.kind === 'run-task'
            ? selection.instanceId
            : resource.kind === 'log'
              ? resource.taskId
              : undefined;
        if (
          reference.retracted ||
          reference.evidence.schemaVersion !== 3 ||
          (resource.kind !== 'run' && resource.kind !== 'log') ||
          !instanceId
        )
          throw new AppProblem('unsupported', 'This reference has no current Task to find.');
        const runResource = { kind: 'run' as const, runRef: resource.runRef };
        const existing = current.workspace.surfaces.find(
          (surface) => resourceKey(surface.resource) === resourceKey(runResource),
        );
        const metadata = existing
          ? undefined
          : {
              ...(await this.resources.describe(input.projectId, runResource)),
              surfaceId: 'srf_' + randomUUID(),
            };
        next = transition(
          current,
          { kind: 'open', resource: runResource, pane: current.activePane, duplicate: false },
          metadata,
        );
        const surface = next.workspace.surfaces.find(
          (surface) => resourceKey(surface.resource) === resourceKey(runResource),
        )!;
        if (surface.view !== 'run')
          throw new AppProblem('unsupported', 'This source has no Run view.');
        surface.runView = {
          ...surface.runView,
          mode: 'tasks',
          query: instanceId.slice(0, 200),
          status: null,
          viewRevision: (surface.runView?.viewRevision ?? 0) + 1,
        };
        next.selections = next.selections.filter((item) => item.surfaceId !== surface.surfaceId);
        reloadSurfaceId = surface.surfaceId;
        delete next.referenceReveal;
        clearReference = true;
      } else if (action.kind === 'reveal') {
        const reference = sharedReference(current, action.referenceId);
        if (reference.retracted) throw new AppProblem('not_found', 'This reference was retracted.');
        if (reference.evidence.schemaVersion === 6) {
          if (!this.resources.readNotebookReference)
            throw new AppProblem('unsupported', 'Notebook reference reading is unavailable.');
          let data: SurfaceData;
          try {
            data = await this.resources.readNotebookReference(reference.evidence);
          } catch {
            throw new AppProblem(
              'stale_revision',
              'This Notebook version is unavailable. No highlight was applied. Open matching captured evidence when available.',
            );
          }
          next = structuredClone(current);
          observedSession = prepareObservedReference(next, reference, data, input.requestId);
        } else if (reference.evidence.schemaVersion === 5) {
          if (!this.resources.readPdfReference)
            throw new AppProblem('unsupported', 'PDF reference reading is unavailable.');
          let data: SurfaceData;
          try {
            data = await this.resources.readPdfReference(reference.evidence);
          } catch {
            throw new AppProblem(
              'stale_revision',
              'This PDF version is unavailable. No highlight was applied. Open matching captured evidence when available.',
            );
          }
          next = structuredClone(current);
          observedSession = prepareObservedReference(next, reference, data, input.requestId);
        } else if (reference.evidence.schemaVersion >= 3 && reference.evidence.selection) {
          const existing = current.workspace.surfaces.find(
            (item) => resourceKey(item.resource) === resourceKey(reference.evidence.resource),
          );
          let data: SurfaceData | undefined;
          if (existing) {
            try {
              data = this.presentation.rendering.snapshot(existing.surfaceId).data;
            } catch {
              /* Hidden/closed sources require a current read. */
            }
          }
          if (!data) data = await this.resources.read(input.projectId, reference.evidence.resource);
          next = structuredClone(current);
          observedSession = prepareObservedReference(next, reference, data, input.requestId);
        } else if (reference.presentation) {
          referenceData = await this.resources.read(input.projectId, reference.evidence.resource);
          next = structuredClone(current);
          referenceSession = prepareReferenceView(next, reference, referenceData, input.requestId);
        } else {
          clearReference = true;
          const existing = current.workspace.surfaces.find(
            (item) =>
              item.view !== 'scatter' &&
              resourceKey(item.resource) === resourceKey(reference.evidence.resource),
          );
          const metadata = existing
            ? undefined
            : {
                ...(await this.resources.describe(input.projectId, reference.evidence.resource)),
                surfaceId: 'srf_' + randomUUID(),
              };
          next = transition(
            current,
            {
              kind: 'open',
              resource: reference.evidence.resource,
              pane: current.activePane,
              duplicate: false,
            },
            metadata,
          );
        }
        next.referenceReveal = { referenceId: action.referenceId, requestId: input.requestId };
        if (
          reference.evidence.resource.kind === 'log' &&
          reference.evidence.selection?.kind === 'text'
        ) {
          const log = next.workspace.surfaces.find(
            (surface) => resourceKey(surface.resource) === resourceKey(reference.evidence.resource),
          );
          if (log?.view === 'log')
            log.logView = { stream: 'legacy', viewRevision: (log.logView?.viewRevision ?? 0) + 1 };
        }
      } else if (action.kind === 'openReport') {
        let saved = current.savedReports?.find(
          (r) =>
            r.runRef === action.runRef &&
            r.producer.instance === action.instance &&
            r.producer.attempt === action.attempt,
        );
        const base = structuredClone(current);
        if (!saved) {
          if (!this.resources.captureReport)
            throw new AppProblem('unsupported', 'Quality report capture is unavailable.');
          if ((base.savedReports?.length ?? 0) >= 128)
            throw new AppProblem('unsupported', 'This Project has reached its saved report limit.');
          this.pendingEvidenceCleanup.add(input.projectId);
          saved = await this.resources.captureReport({
            projectId: input.projectId,
            runRef: action.runRef,
            instance: action.instance,
            attempt: action.attempt,
          });
          if (
            saved.projectId !== input.projectId ||
            saved.runRef !== action.runRef ||
            saved.producer.instance !== action.instance ||
            saved.producer.attempt !== action.attempt
          )
            throw new AppProblem('internal', 'Report capture returned another producer.');
          base.savedReports = [...(base.savedReports ?? []), saved];
        }
        const resource = { kind: 'report' as const, saved };
        const metadata = await this.resources.describe(input.projectId, resource);
        next = transition(
          base,
          { kind: 'open', resource, pane: 'secondary', duplicate: false },
          { ...metadata, surfaceId: 'srf_' + randomUUID() },
        );
        next.maximizedPane = null;
      } else next = transition(current, action, opened);
      next.workspace.revision += 1;
      next.receipts = [...next.receipts, { requestId: input.requestId, digest: hash }].slice(-256);
      const activity = this.activityText(next, input, current);
      if (activity)
        next.activity = [
          ...next.activity,
          { id: input.requestId, text: activity, at: Date.now() },
        ].slice(-100);
      const saved = await this.commit(next);
      if (observedSession) {
        this.presentation.referenceViews.clear();
        this.presentation.observedReferences.set(observedSession);
      } else if (referenceSession) {
        this.presentation.observedReferences.clear();
        this.presentation.referenceViews.set(referenceSession);
        const link = viewLinkFor(saved, referenceSession.surfaceIds[0]!);
        if (link && referenceData)
          this.presentation.tabularSnapshots.seed(input.projectId, link, referenceData);
      } else if (clearReference) {
        this.presentation.referenceViews.clear();
        this.presentation.observedReferences.clear();
      }
      if (tabularData && 'surfaceId' in action) {
        const link = viewLinkFor(saved, action.surfaceId);
        if (link) this.presentation.tabularSnapshots.seed(input.projectId, link, tabularData);
      }
      if (action.kind === 'close') {
        const closed = current.workspace.surfaces.find(
          (item) => item.surfaceId === action.surfaceId,
        )!;
        this.closedResources.set(
          input.projectId + '/' + resourceKey(closed.resource),
          ++this.closeSequence,
        );
      }
      if (reloadSurfaceId) this.presentation.rendering.forget(reloadSurfaceId);
      this.pruneLeases(saved);
      return saved;
    });
  }
  private activityText(
    doc: WorkspaceDocument,
    input: WorkspaceCommand,
    previous: WorkspaceDocument,
  ): string | null {
    const action = input.action;
    const title =
      'surfaceId' in action
        ? (doc.titles.find((item) => item.surfaceId === action.surfaceId)?.title ?? 'View')
        : 'View';
    switch (action.kind) {
      case 'refreshLinked':
        return viewLinkFor(doc, action.surfaceId)?.dataRevision !==
          viewLinkFor(previous, action.surfaceId)?.dataRevision
          ? 'Updated the linked source and cleared the previous selection and view settings.'
          : 'Refreshed the linked views.';
      case 'openReport':
        return 'Opened a saved quality report.';
      case 'open':
        return action.duplicate ? 'Duplicated a view into another pane.' : 'Opened a Project view.';
      case 'close':
        return 'Closed a view. Its source is unchanged.';
      case 'pin':
        return (action.pinned ? 'Pinned ' : 'Unpinned ') + title + '.';
      case 'move':
        return 'Moved ' + title + ' to the ' + action.pane + ' pane.';
      case 'arrange':
        return action.layout === 'split' ? 'Opened a second pane.' : 'Combined the panes.';
      default:
        return null;
    }
  }
  updateDraft(projectId: string, draft: string): Promise<WorkspaceDocument> {
    return this.enqueue(async () => {
      this.requireActive(projectId);
      const doc = structuredClone(await this.document(projectId));
      if (doc.chat.draft === draft) return doc;
      doc.chat.draft = draft;
      doc.workspace.revision += 1;
      return this.commit(doc);
    });
  }
  async loadSurface(input: SurfaceRequest): Promise<SurfaceLoad> {
    const ticket = await this.enqueue(async () => {
      this.requireActive(input.projectId);
      const doc = await this.document(input.projectId);
      const surface = doc.workspace.surfaces.find((item) => item.surfaceId === input.surfaceId);
      if (!surface) throw new AppProblem('not_found', 'This view is no longer open.');
      this.pruneLeases(doc);
      if (input.refresh && this.presentation.observedReferences.forSurface(input.surfaceId))
        throw new AppProblem(
          'invalid_request',
          'Return to your view before refreshing the source.',
        );
      return this.presentation.rendering.begin(
        input,
        surface,
        viewLinkFor(doc, surface.surfaceId),
        this.presentation.referenceViews.forSurface(surface.surfaceId),
        this.presentation.observedReferences.forSurface(surface.surfaceId),
      );
    });
    try {
      const read = () =>
        this.resources.readSurface
          ? this.resources.readSurface(ticket.surface, !!input.refresh)
          : this.resources.read(input.projectId, ticket.surface.resource);
      const referenceData = this.presentation.observedReferences.dataFor(input.surfaceId);
      const retained = referenceData ?? this.presentation.rendering.retainedData(ticket);
      let data: SurfaceData;
      let refreshProblem = input.refresh
        ? undefined
        : this.presentation.rendering.retainedProblem(ticket);
      try {
        data =
          retained && !input.refresh
            ? retained
            : ticket.link
              ? await this.presentation.tabularSnapshots.read(input.projectId, ticket.link, read)
              : await read();
      } catch (error) {
        if (!retained) throw error;
        data = retained;
        refreshProblem =
          error instanceof AppProblem
            ? error.message.slice(0, 1000)
            : 'The runtime observation could not be refreshed.';
      }
      return await this.enqueue(async () => {
        this.requireActive(input.projectId);
        let load: SurfaceLoad;
        try {
          load = this.presentation.rendering.complete(ticket, data, refreshProblem);
        } catch (error) {
          // Admission limits are refresh failures too. A stale ticket must still fail closed.
          if (!retained || !(error instanceof AppProblem) || error.code !== 'unsupported')
            throw error;
          data = retained;
          refreshProblem = error.message.slice(0, 1000);
          load = this.presentation.rendering.complete(ticket, data, refreshProblem);
        }
        if (
          (data.kind !== 'file' ||
            data.value.content.kind === 'pdf' ||
            data.value.content.kind === 'notebook') &&
          !referenceData
        ) {
          const doc = await this.document(input.projectId);
          const selected = doc.selections.find((item) => item.surfaceId === input.surfaceId);
          if (selected) {
            let exact = true;
            try {
              checkSelection(selected.evidence, data);
            } catch {
              exact = false;
            }
            if (!exact) {
              const next = structuredClone(doc);
              next.selections = next.selections.filter(
                (item) => item.surfaceId !== input.surfaceId,
              );
              next.workspace.revision++;
              await this.commit(next);
            }
          }
        }
        return { ...load, ...(refreshProblem ? { refreshProblem } : {}) };
      });
    } finally {
      this.presentation.rendering.release(ticket);
    }
  }
  acknowledge(input: RenderAcknowledgment): Promise<{ accepted: true }> {
    return this.enqueue(async () => {
      this.requireActive(input.projectId);
      return this.presentation.rendering.acknowledge(input);
    });
  }
  invalidate(input: RenderAcknowledgment): Promise<{ accepted: true }> {
    return this.enqueue(async () => {
      this.requireActive(input.projectId);
      return this.presentation.rendering.invalidate(input);
    });
  }
}
