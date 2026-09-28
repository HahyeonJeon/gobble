import { readReport } from './report';
import { reportTarget } from '@gobble/contracts';
import type { PipelineReviewHost } from '../pipeline-review/host';
import { notebookObservation } from './notebook-observation';
import type { NotebookCapture } from '../evidence/notebook-evidence';
import { pdfObservation } from './pdf-observation';
import type { PdfRasterCapture } from '../evidence/pdf-evidence';
import { runObservation } from './run-observation';
import { ObservedReads } from './observed';
import { presentLog } from '../service/log-presentation';
import type { QuestionTools } from '../questions/ports';
import { setTimeout as delay } from 'node:timers/promises';
import { createHash, randomUUID } from 'node:crypto';
import {
  pipelineSubject,
  pipelineSubjectLabel,
  parse,
  observedTargetVisible,
  resourceKey,
  validateReferencePresentation,
  SHARED_TOOLSET,
  type ReferencePresentation,
  type RenderAcknowledgment,
  type EvidenceRef,
} from '@gobble/contracts';
import { AppProblem } from '../problem';
import type { ProjectService } from '../service/project-service';
import type { WorkspaceController } from '../workspace/controller';
import type { WorkspaceResources } from '../workspace/service';
import { checkSelection } from '../workspace/selection';
import { toolSchemas } from './catalog';
import { arrangeView, openViews, releaseView, type PreparedView } from './layout';
import { textObservation, type ImageRenderer } from './observation';
import { observeTabular } from './tabular';
import { publishReference, requireEvidence } from './references';
import {
  textResult,
  type SharedToolExecutor,
  type ToolCall,
  type ToolContext,
  type ToolResult,
} from './ports';

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
type Receipt = { digest: string; result: Promise<ToolResult> };
type TurnCache = {
  bytes: number;
  dismissals: number;
  receipts: Map<string, Receipt>;
  observed: ObservedReads;
};
export type ResourceCatalog = Pick<
  ProjectService,
  'listFiles' | 'listRuns' | 'readRun' | 'readLogs'
>;

/** Application use cases. Only WorkspaceController commits Project state. */
export class SharedContextHost implements SharedToolExecutor {
  private readonly turns = new Map<string, TurnCache>();
  constructor(
    private readonly workspace: WorkspaceController,
    private readonly resources: WorkspaceResources,
    private readonly catalog: ResourceCatalog,
    private readonly image: ImageRenderer,
    private readonly supportsImages: (model: string) => boolean,
    private readonly questions?: QuestionTools,
    private readonly pdfCapture?: PdfRasterCapture,
    private readonly notebookCapture?: NotebookCapture,
    private readonly pipelineReview?: PipelineReviewHost,
  ) {}
  begin(projectId: string, agentId: string, submissionId: string): void {
    this.questions?.begin(projectId, agentId, submissionId);
    this.turns.set(projectId + '/' + agentId + '/' + submissionId, {
      dismissals: this.workspace.dismissalSequence,
      bytes: 0,
      observed: new ObservedReads(),
      receipts: new Map(),
    });
  }
  release(projectId: string, agentId: string, submissionId: string): void {
    this.questions?.release(projectId, agentId, submissionId);
    this.pipelineReview?.release(projectId, agentId, submissionId);
    this.turns.delete(projectId + '/' + agentId + '/' + submissionId);
  }
  /** Resolve a portable pointer against a ready instance; origin never constrains identity. */
  private async pointSource(
    projectId: string,
    evidence: EvidenceRef,
    guard: () => void,
    observation?: RenderAcknowledgment,
  ) {
    if (observation) {
      if (observation.projectId !== projectId || evidence.projectId !== projectId)
        throw new AppProblem('forbidden', 'This observation belongs to another Project.');
      const view = await this.workspace.captureShared(projectId, observation.surfaceId, guard);
      if (canonical(view.acknowledgment) !== canonical(observation))
        throw new AppProblem(
          'stale_revision',
          'The observed presentation changed. Observe again before pointing.',
        );
      if (
        evidence.schemaVersion !== 4 &&
        view.surface.view === 'run' &&
        view.surface.runView?.mode === 'dependencies'
      )
        throw new AppProblem('unsupported', 'Switch this Run to Tasks before using task pointers.');
      checkSelection(evidence, view.data);
      if (!observedTargetVisible(view.surface, view.data, evidence))
        throw new AppProblem(
          'stale_revision',
          'This target is no longer displayed. Observe the current view again.',
        );
      return view;
    }
    if (
      evidence.schemaVersion === 4 ||
      evidence.schemaVersion === 5 ||
      evidence.schemaVersion === 6 ||
      evidence.schemaVersion === 7 ||
      evidence.schemaVersion === 8
    )
      throw new AppProblem('invalid_request', 'Observed pointers require an observation receipt.');
    const doc = await this.workspace.readShared(projectId);
    guard();
    if (evidence.projectId !== projectId)
      throw new AppProblem('forbidden', 'This reference belongs to another Project.');
    const candidates = doc.workspace.surfaces.filter(
      (surface) =>
        surface.view !== 'scatter' &&
        !(surface.view === 'run' && surface.runView?.mode === 'dependencies') &&
        resourceKey(surface.resource) === resourceKey(evidence.resource) &&
        this.workspace.sharedPresentation(projectId, surface.surfaceId) === 'ready',
    );
    candidates.sort(
      (a, b) =>
        Number(b.surfaceId === evidence.origin?.surfaceId) -
        Number(a.surfaceId === evidence.origin?.surfaceId),
    );
    if (!candidates.length)
      throw new AppProblem(
        'not_found',
        'Open this resource in a visible view before sharing a mark. Plot references require the observation receipt returned by workspace_observe.',
      );
    for (const surface of candidates) {
      const view = await this.workspace.captureShared(projectId, surface.surfaceId, guard);
      try {
        checkSelection(evidence, view.data);
      } catch (error) {
        if (error instanceof AppProblem && error.code === 'stale_revision') continue;
        throw error;
      }
      return view;
    }
    throw new AppProblem(
      'stale_revision',
      'This reference does not match any ready view. Observe the current version before sharing a mark.',
    );
  }
  async execute(context: ToolContext, call: ToolCall): Promise<ToolResult> {
    const guard = () => {
      context.signal.throwIfAborted();
      context.assert();
    };
    try {
      guard();
      const key =
        context.agent.projectId + '/' + context.agent.agentId + '/' + context.submission.requestId;
      // At most two active conversations; cache is released on every terminal/revocation path.
      const turn = this.turns.get(key);
      if (!turn)
        throw new AppProblem('forbidden', 'This submission has no active shared-tool session.');
      const digest = createHash('sha256')
        .update(canonical({ tool: call.tool, arguments: call.arguments }))
        .digest('hex');
      const invocation = call.threadId + '/' + call.turnId + '/' + call.callId;
      const previous = turn.receipts.get(invocation);
      if (previous) {
        if (previous.digest !== digest)
          throw new AppProblem(
            'request_conflict',
            'This tool call ID was used with different arguments.',
          );
        const result = await previous.result;
        guard();
        result.assertCurrent?.();
        return result;
      }
      if (turn.receipts.size >= 32)
        throw new AppProblem(
          'unsupported',
          'This message reached its limit of 32 shared tool calls.',
        );
      const result = this.dispatch(context, call, guard, turn.dismissals, turn.observed)
        .then((value) => {
          guard();
          value.assertCurrent?.();
          const bytes = Buffer.byteLength(JSON.stringify(value));
          if (turn.bytes + bytes > 8 * 1024 * 1024 - 65536)
            throw new AppProblem(
              'unsupported',
              'This message reached its shared observation cache limit. Start a new message for more content.',
            );
          turn.bytes += bytes;
          return value;
        })
        .catch((error) => this.failure(error));
      turn.receipts.set(invocation, { digest, result });
      return await result;
    } catch (error) {
      return this.failure(error);
    }
  }
  private failure(error: unknown): ToolResult {
    return {
      success: false,
      content: [
        {
          type: 'text',
          text: JSON.stringify({
            error: {
              code: error instanceof AppProblem ? error.code : 'invalid_request',
              message:
                error instanceof AppProblem
                  ? error.message
                  : 'The shared request is invalid, expired or unavailable.',
            },
          }),
        },
      ],
    };
  }
  private async observeReady(
    projectId: string,
    surfaceId: string,
    guard: () => void,
    signal: AbortSignal,
  ) {
    for (let attempt = 0; ; attempt++) {
      guard();
      try {
        return await this.workspace.captureShared(projectId, surfaceId, guard);
      } catch (error) {
        if (!(error instanceof AppProblem) || error.code !== 'stale_revision' || attempt >= 24)
          throw error;
        await delay(100, undefined, { signal });
      }
    }
  }
  private async questionObservation(
    context: ToolContext,
    evidence: EvidenceRef,
    data: import('@gobble/contracts').SurfaceData,
    image: ImageRenderer,
    assert: () => void,
    presentation?: ReferencePresentation,
  ) {
    if (!this.questions) return {};
    try {
      return {
        evidenceId: await this.questions.observe(
          context,
          evidence,
          data,
          image,
          assert,
          presentation,
        ),
      };
    } catch (error) {
      assert();
      if (!(error instanceof AppProblem) || error.code !== 'unsupported') throw error;
      return { questionEvidenceUnavailable: error.message };
    }
  }
  private async dispatch(
    context: ToolContext,
    call: ToolCall,
    guard: () => void,
    dismissals: number,
    observedReads: ObservedReads,
  ): Promise<ToolResult> {
    const projectId = context.agent.projectId;
    if (Buffer.byteLength(JSON.stringify(call.arguments)) > 64 * 1024)
      throw new AppProblem('unsupported', 'The tool arguments exceed the size limit.');
    if (call.tool.startsWith('gobble_pipeline_') || call.tool.startsWith('gobble_creation_')) {
      if (!this.pipelineReview)
        throw new AppProblem('unsupported', 'Pipeline review is unavailable.');
      return textResult(await this.pipelineReview.execute(context, call, guard));
    }
    switch (call.tool) {
      case 'read_report':
        return readReport(
          context,
          parse(toolSchemas.read_report, call.arguments),
          this.workspace,
          this.resources,
          this.supportsImages,
          guard,
        );
      case 'workspace_question': {
        if (!this.questions) throw new AppProblem('unsupported', 'Question tools are unavailable.');
        const assert = () => {
          guard();
          this.workspace.requireSharedUI(projectId);
        };
        assert();
        return textResult(await this.questions.create(context, call, assert));
      }
      case 'workspace_list': {
        parse(toolSchemas.workspace_list, call.arguments);
        const doc = await this.workspace.readShared(projectId);
        guard();
        let available = true;
        try {
          this.workspace.requireSharedUI(projectId);
        } catch {
          available = false;
        }
        return textResult({
          projectId,
          toolsetVersion: SHARED_TOOLSET,
          tabularObservation: 'semantic-data-and-presentation; no plot pixels',
          available,
          viewLinks: doc.viewLinks.map(
            ({ rowKeys: _privateSelection, selectionSourceId: _privateOrigin, ...link }) => link,
          ),
          layout: doc.workspace.layout,
          maximizedPane: doc.maximizedPane,
          surfaces: doc.workspace.surfaces.map((surface) => ({
            ...surface,
            referenceView: this.workspace.sharedReferenceView(projectId, surface.surfaceId) ?? null,
            presentation: this.workspace.sharedPresentation(projectId, surface.surfaceId),
            title: doc.titles.find((item) => item.surfaceId === surface.surfaceId)?.title,
          })),
          references: doc.sharedReferences ?? [],
        });
      }
      case 'workspace_resources': {
        const input = parse(toolSchemas.workspace_resources, call.arguments);
        const value = await this.catalog.listFiles({ projectId, ...input });
        guard();
        return textResult(value);
      }
      case 'workspace_open': {
        const input = parse(toolSchemas.workspace_open, call.arguments);
        this.workspace.requireSharedUI(projectId);
        if (new Set(input.resources.map(resourceKey)).size !== input.resources.length)
          throw new AppProblem('invalid_request', 'Open each resource only once.');
        const prepared: PreparedView[] = [];
        for (const resource of input.resources) {
          if (resource.kind === 'report') {
            const doc = await this.workspace.readShared(projectId);
            guard();
            if (
              !(doc.savedReports ?? []).some(
                (saved) => canonical(saved) === canonical(resource.saved),
              )
            )
              throw new AppProblem(
                'not_found',
                'This saved report is not retained in this Project.',
              );
          }
          const metadata = await this.resources.describe(projectId, resource);
          guard();
          prepared.push({ ...metadata, resource, surfaceId: 'srf_' + randomUUID() });
        }
        const opened = await this.workspace.changeShared(projectId, guard, (doc) =>
          openViews(doc, prepared, context.agent, input.preferredPane ?? 'primary', (resource) =>
            this.workspace.dismissedSince(projectId, resource, dismissals),
          ),
        );
        return textResult({
          views: opened.map((view) => ({
            ...view,
            presentation: this.workspace.sharedPresentation(projectId, view.surfaceId),
            visibility:
              view.visibility === 'background'
                ? 'background'
                : this.workspace.sharedPresentation(projectId, view.surfaceId) === 'ready'
                  ? 'revealed'
                  : 'opening',
          })),
          observed: false,
        });
      }
      case 'workspace_arrange': {
        const input = parse(toolSchemas.workspace_arrange, call.arguments);
        return textResult(
          await this.workspace.changeShared(projectId, guard, (doc) =>
            arrangeView(doc, input.surfaceId, input.pane, context.agent.agentId),
          ),
        );
      }
      case 'workspace_release': {
        const input = parse(toolSchemas.workspace_release, call.arguments);
        return textResult(
          await this.workspace.changeShared(projectId, guard, (doc) => {
            releaseView(doc, input.surfaceId, context.agent.agentId);
            return { released: true };
          }),
        );
      }
      case 'workspace_observe': {
        const input = parse(toolSchemas.workspace_observe, call.arguments);
        const view = await this.observeReady(projectId, input.surfaceId, guard, context.signal);
        if (view.surface.view === 'report') {
          if (
            view.data.kind !== 'report' ||
            input.selection ||
            input.stream ||
            input.representation ||
            input.scope === 'source-preview'
          )
            throw new AppProblem(
              'unsupported',
              'Reports support whole-report foreground observation. Attach the report to this message for complete reading.',
            );
          const evidence = reportTarget(view.data.saved, view.surface.surfaceId);
          const sectionId = view.surface.reportModuleId;
          const module =
            view.data.value.content.modules.find((m) => m.id === sectionId) ??
            view.data.value.content.modules[0]!;
          const receipt = observedReads.recordReport(view, evidence, view.assert);
          view.assert();
          return {
            ...textResult({
              receipt,
              content: {
                kind: 'report',
                title: view.data.saved.title,
                selectedSection: { id: module.id, title: module.title, status: module.status },
                imagesReturned: [],
                coverage:
                  'Whole-report identity and selected section heading only. This is not pixel observation. Attach this report to your message and use read_report for full reading.',
              },
            }),
            assertCurrent: view.assert,
          };
        }
        if (view.surface.view === 'notebook') {
          if (input.stream || input.representation)
            throw new AppProblem(
              'invalid_request',
              'Notebook observations have no task representation or log stream.',
            );
          const result = notebookObservation(view, input.selection, input.scope ?? 'view');
          if (!this.resources.validateNotebook)
            throw new AppProblem('unsupported', 'Notebook validation is unavailable.');
          for (const target of result.targets)
            this.resources.validateNotebook(view.surface.surfaceId, target, view.data);
          let imageUrl: string | undefined;
          if (result.image) {
            if (!this.notebookCapture || !this.supportsImages(context.submission.model))
              throw new AppProblem(
                'unsupported',
                'Notebook image observations require an image-capable model.',
              );
            const asset = await this.notebookCapture(
              {
                attachmentId: 'att_' + randomUUID(),
                evidence: result.image,
                label: 'Notebook image observation',
                createdAt: Date.now(),
              },
              view.data,
            );
            view.assert();
            imageUrl = 'data:image/png;base64,' + asset.bytes.toString('base64');
          }
          const explicit = input.selection ? result.targets[0] : undefined;
          const questionEvidence = explicit
            ? await this.questionObservation(context, explicit, view.data, this.image, view.assert)
            : {
                questionEvidenceUnavailable:
                  'Observe one exact part selector to obtain question evidence.',
              };
          view.assert();
          const receipt = observedReads.recordNotebook(
            view,
            result.targets,
            result.scope,
            view.assert,
          );
          return {
            success: true,
            assertCurrent: view.assert,
            content: [
              {
                type: 'text',
                text: JSON.stringify({
                  receipt: { ...receipt, ...questionEvidence },
                  content: {
                    kind: 'notebook',
                    scope: result.scope,
                    textParts: result.textParts,
                    imagesAvailable: result.imagesAvailable,
                    image: result.image ?? null,
                  },
                }),
              },
              ...(imageUrl ? [{ type: 'image' as const, url: imageUrl }] : []),
            ],
          };
        }
        if (view.surface.view === 'pdf') {
          if (input.stream || input.representation)
            throw new AppProblem(
              'invalid_request',
              'PDF pages have no task representation or log stream.',
            );
          if (!this.pdfCapture || !this.supportsImages(context.submission.model))
            throw new AppProblem('unsupported', 'PDF observations require an image-capable model.');
          const scope = input.scope ?? 'view',
            pdf = pdfObservation(view, input.selection, scope);
          const asset = await this.pdfCapture(
            {
              attachmentId: 'att_' + randomUUID(),
              evidence: pdf.evidence,
              label: 'PDF observation',
              createdAt: Date.now(),
            },
            view.data,
          );
          view.assert();
          const questionEvidence = await this.questionObservation(
            context,
            pdf.evidence,
            view.data,
            this.image,
            view.assert,
          );
          view.assert();
          const receipt = observedReads.recordPdf(view, pdf.evidence, scope, view.assert);
          return {
            assertCurrent: view.assert,
            success: true,
            content: [
              {
                type: 'text',
                text: JSON.stringify({
                  receipt: { ...receipt, ...questionEvidence },
                  content: { ...pdf.content, image: asset.manifest.representation },
                }),
              },
              { type: 'image', url: 'data:image/png;base64,' + asset.bytes.toString('base64') },
            ],
          };
        }
        const dependency =
          input.representation === 'dependencies' ||
          input.selection?.kind === 'run-group' ||
          input.selection?.kind === 'run-dependency' ||
          (!input.representation &&
            !input.selection &&
            view.surface.view === 'run' &&
            view.surface.runView?.mode === 'dependencies');
        if (input.representation && view.data.kind !== 'run')
          throw new AppProblem('invalid_request', 'Representation requires a Run view.');
        if (dependency && (input.stream || input.representation === 'tasks'))
          throw new AppProblem(
            'invalid_request',
            'Dependency selectors cannot address tasks or log streams.',
          );
        if (view.data.kind === 'pipeline' && input.stream)
          throw new AppProblem('invalid_request', 'Pipeline views have no log stream.');
        const observed =
          view.data.kind === 'pipeline'
            ? observedReads.observePipeline(view, input.selection, input.scope ?? 'view')
            : dependency
              ? observedReads.observeDependencies(view, input.selection, input.scope ?? 'view')
              : observedReads.observe(view, input.selection, input.scope ?? 'view', input.stream);
        if (observed) {
          view.assert();
          const questionEvidence = observed.receipt.evidence?.selection
            ? await this.questionObservation(
                context,
                observed.receipt.evidence,
                view.data,
                this.image,
                view.assert,
              )
            : {};
          return {
            ...textResult({ ...observed, receipt: { ...observed.receipt, ...questionEvidence } }),
            assertCurrent: view.assert,
          };
        }
        if (
          input.stream ||
          input.selection?.kind === 'run-group' ||
          input.selection?.kind === 'run-dependency' ||
          input.selection?.kind === 'run-task' ||
          input.selection?.kind === 'log-text' ||
          input.selection?.kind === 'pdf' ||
          input.selection?.kind === 'notebook' ||
          input.selection?.kind === 'pipeline'
        )
          throw new AppProblem(
            'invalid_request',
            'Observed task and stream selectors require a Run or log view.',
          );
        if (
          view.surface.resource.kind === 'pipeline' ||
          view.surface.resource.kind === 'creation-draft'
        )
          throw new AppProblem(
            'unsupported',
            'Pipeline discussion references are not available yet.',
          );
        const tabular = observeTabular(view, input.selection, input.scope ?? 'view');
        const selection =
          tabular?.selection ??
          (tabular?.content.kind === 'table' && tabular.content.rows.length
            ? {
                kind: 'table' as const,
                coordinateSpace: 'revision-row-column-keys' as const,
                rowKeys: tabular.content.rows.map((row) => row.key),
                columns: tabular.content.columns.map((column) => column.id),
              }
            : input.selection);
        const evidence: EvidenceRef = {
          projectId,
          schemaVersion: 2,
          origin: { surfaceId: input.surfaceId },
          resource: view.surface.resource,
          dataRevision: view.acknowledgment.dataRevision,
          ...(selection ? { selection } : {}),
        };
        checkSelection(evidence, view.data);
        const receipt = {
          evidence,
          observedAt: Date.now(),
          generation: view.acknowledgment.generation,
          source: 'loaded-preview',
          observation: view.acknowledgment,
          ...(tabular
            ? {
                scope: tabular.scope,
                presentation: tabular.presentation ?? null,
                plotPixelsIncluded: false,
              }
            : {}),
        };
        if (view.data.kind === 'file' && view.data.value.content.kind === 'image') {
          if (!this.supportsImages(context.submission.model))
            throw new AppProblem(
              'unsupported',
              'This model does not advertise image input. Choose an image-capable model.',
            );
          const image = await this.image(
            view.data.value.content,
            input.selection?.kind === 'image' ? input.selection : undefined,
          );
          view.assert();
          const questionEvidence = await this.questionObservation(
            context,
            evidence,
            view.data,
            async () => image,
            view.assert,
          );
          return {
            assertCurrent: view.assert,
            success: true,
            content: [
              {
                type: 'text',
                text: JSON.stringify({
                  receipt: { ...receipt, ...questionEvidence },
                  image: {
                    width: image.width,
                    height: image.height,
                    crop: image.crop,
                    originalWidth: view.data.value.content.width,
                    originalHeight: view.data.value.content.height,
                    contentHash: view.data.value.revision,
                  },
                }),
              },
              { type: 'image', url: image.url },
            ],
          };
        }
        const content = tabular?.content ?? textObservation(view.data, input.selection);
        view.assert();
        const questionEvidence =
          tabular?.content.kind === 'table' && !tabular.content.rows.length
            ? {
                questionEvidenceUnavailable:
                  'No rows were returned. Observe exact source rows before attaching question evidence.',
              }
            : await this.questionObservation(
                context,
                evidence,
                view.data,
                this.image,
                view.assert,
                evidence.selection?.kind === 'table' ? tabular?.presentation : undefined,
              );
        return {
          ...textResult({ receipt: { ...receipt, ...questionEvidence }, content }),
          assertCurrent: view.assert,
        };
      }
      case 'workspace_point': {
        const input = parse(toolSchemas.workspace_point, call.arguments);
        if (input.evidence.schemaVersion >= 3)
          observedReads.assertPoint(input.observedReadId, input.evidence, input.observation);
        const pointGuard = () => {
          guard();
          if (
            input.evidence.schemaVersion === 5 ||
            input.evidence.schemaVersion === 6 ||
            input.evidence.schemaVersion === 8
          )
            observedReads.assertPoint(input.observedReadId, input.evidence, input.observation);
        };
        const view = await this.pointSource(
          projectId,
          input.evidence,
          pointGuard,
          input.observation,
        );
        const presentation =
          view.data.kind === 'file' && view.data.value.content.kind === 'table'
            ? view.referenceView?.presentation
            : undefined;
        validateReferencePresentation(input.evidence, presentation);
        const referenceId = 'ref_' + randomUUID();
        return textResult(
          await this.workspace.changeShared(
            projectId,
            () => {
              guard();
              view.assert();
            },
            (doc) => {
              requireEvidence(doc, input.evidence, view.surface.surfaceId);
              publishReference(
                doc,
                {
                  referenceId,
                  ...(input.evidence.schemaVersion === 7 &&
                  view.data.kind === 'pipeline' &&
                  view.data.value.artifact
                    ? {
                        label: pipelineSubjectLabel(
                          pipelineSubject(
                            view.data.value.artifact.flow,
                            input.evidence.selection.subject,
                          ),
                        ).slice(0, 512),
                      }
                    : {}),
                  evidence: input.evidence,
                  ...(presentation ? { presentation: structuredClone(presentation) } : {}),
                  note: input.note,
                  author: {
                    kind: 'agent',
                    agentId: context.agent.agentId,
                    name: context.agent.name,
                  },
                  createdAt: Date.now(),
                  retracted: false,
                  originSubmissionId: context.submission.requestId,
                },
                view.surface.surfaceId,
              );
              return { referenceId, published: true };
            },
          ),
        );
      }
      case 'gobble_runs': {
        parse(toolSchemas.gobble_runs, call.arguments);
        const value = await this.catalog.listRuns({ projectId });
        guard();
        return textResult(value);
      }
      case 'gobble_run': {
        const input = parse(toolSchemas.gobble_run, call.arguments);
        const value = await this.resources.read(projectId, { kind: 'run', runRef: input.runRef });
        guard();
        if (value.kind !== 'run')
          throw new AppProblem('unsupported', 'A Run observation is required.');
        return textResult({
          source: 'runtime-read',
          displayed: false,
          content: runObservation(value.value),
        });
      }
      case 'gobble_logs': {
        const input = parse(toolSchemas.gobble_logs, call.arguments);
        const value = await this.catalog.readLogs({ projectId, ...input });
        guard();
        return textResult({ source: 'runtime-read', displayed: false, content: presentLog(value) });
      }
      default:
        throw new AppProblem('forbidden', 'This tool is not registered for shared views.');
    }
  }
}
