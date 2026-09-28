import type { ReportEvidence } from '../evidence/report';
import type { SavedReportRecord, LogsInput } from '@gobble/contracts';
import { continuationContext } from '../service/continuation-context';
import type { RunPresentationV2 } from '@gobble/contracts';
import type { NotebookViews } from './notebook-views';
import type { NotebookTarget, NotebookImage } from '@gobble/contracts';
import type { PdfViews } from './pdf-views';
import { projectDependencies } from '@gobble/contracts';
import { presentDependencies } from '../service/dependency-presentation';
import {
  logTarget,
  type ProjectInfo,
  type ResourceRef,
  type SurfaceData,
  type Surface,
} from '@gobble/contracts';
import type { ProjectService } from '../service/project-service';
import { presentRun } from '../service/run-presentation';
import { presentLog } from '../service/log-presentation';
import { AppProblem } from '../problem';

export interface WorkspaceResources {
  projects(): Promise<ProjectInfo[]>;
  captureReport?(input: LogsInput): Promise<SavedReportRecord>;
  notebookImage?(surfaceId: string, target: NotebookTarget, data: SurfaceData): NotebookImage;
  validateNotebook?(surfaceId: string, target: NotebookTarget, data: SurfaceData): void;
  checkPdfSource?(target: import('@gobble/contracts').PdfTarget): Promise<void>;
  checkNotebookSource?(target: NotebookTarget): Promise<void>;
  readNotebookReference?(target: NotebookTarget): Promise<SurfaceData>;
  readPdfReference?(target: import('@gobble/contracts').PdfTarget): Promise<SurfaceData>;
  readSurface?(surface: Surface, refresh: boolean): Promise<SurfaceData>;
  retainViews?(projectId: string, surfaceIds: string[]): void;
  clearViews?(): void;
  read(projectId: string, resource: ResourceRef): Promise<SurfaceData>;
  describe(
    projectId: string,
    resource: ResourceRef,
  ): Promise<{ title: string; view: Exclude<Surface['view'], 'scatter'> }>;
}

/** Resolves registered resources for the workspace; never owns Project registrations or engine state. */
export class ServiceResources implements WorkspaceResources {
  constructor(
    private readonly service: ProjectService,
    private readonly pdf?: PdfViews,
    private readonly notebook?: NotebookViews,
    private readonly reports?: ReportEvidence,
  ) {}
  async captureReport(input: LogsInput): Promise<SavedReportRecord> {
    if (!this.reports)
      throw new AppProblem('unsupported', 'Quality report capture is unavailable.');
    return this.reports.capture(await this.service.readReport(input));
  }
  /** A captured page already proves its geometry; currentness needs only exact source bytes. */
  async checkPdfSource(target: import('@gobble/contracts').PdfTarget): Promise<void> {
    try {
      const file = await this.service.readFile({
        projectId: target.projectId,
        resourceId: target.resource.resourceId,
        expectedRevision: target.dataRevision,
      });
      if (file.content.kind !== 'pdf' || file.revision !== target.dataRevision)
        throw new AppProblem('stale_revision', 'The PDF source changed.');
    } catch (error) {
      if (error instanceof AppProblem && ['not_found', 'stale_revision'].includes(error.code))
        throw new AppProblem('stale_revision', 'The PDF source changed or is unavailable.');
      throw error;
    }
  }
  async checkNotebookSource(target: NotebookTarget): Promise<void> {
    try {
      const file = await this.service.readFile({
        projectId: target.projectId,
        resourceId: target.resource.resourceId,
        expectedRevision: target.dataRevision,
      });
      if (file.content.kind !== 'notebook' || file.revision !== target.dataRevision)
        throw new AppProblem('stale_revision', 'The Notebook source changed.');
    } catch (error) {
      if (error instanceof AppProblem && ['not_found', 'stale_revision'].includes(error.code))
        throw new AppProblem('stale_revision', 'The Notebook source changed or is unavailable.');
      throw error;
    }
  }
  async readNotebookReference(target: NotebookTarget): Promise<SurfaceData> {
    if (!this.notebook)
      throw new AppProblem('unsupported', 'Notebook reference reading is unavailable.');
    return this.notebook.reference(target, () =>
      this.service.readFile({
        projectId: target.projectId,
        resourceId: target.resource.resourceId,
        expectedRevision: target.dataRevision,
      }),
    );
  }
  async readPdfReference(target: import('@gobble/contracts').PdfTarget): Promise<SurfaceData> {
    if (!this.pdf) throw new AppProblem('unsupported', 'PDF reference reading is unavailable.');
    return this.pdf.reference(target, () =>
      this.service.readFile({
        projectId: target.projectId,
        resourceId: target.resource.resourceId,
        expectedRevision: target.dataRevision,
      }),
    );
  }
  retainViews(projectId: string, surfaceIds: string[]): void {
    this.pdf?.retain(projectId, surfaceIds);
    this.notebook?.retain(projectId, surfaceIds);
  }
  clearViews(): void {
    this.pdf?.clear();
    this.notebook?.clear();
  }
  async readSurface(surface: Surface, refresh: boolean): Promise<SurfaceData> {
    if (surface.view === 'notebook') {
      const resource = surface.resource;
      if (!this.notebook || resource.kind !== 'file')
        throw new AppProblem('unsupported', 'Notebook reading is unavailable.');
      return this.notebook.read(
        surface,
        () =>
          this.service.readFile({ projectId: surface.projectId, resourceId: resource.resourceId }),
        refresh,
      );
    }
    if (surface.view !== 'pdf') return this.read(surface.projectId, surface.resource);
    const resource = surface.resource;
    if (!this.pdf || resource.kind !== 'file')
      throw new AppProblem('unsupported', 'PDF reading is unavailable.');
    return this.pdf.read(
      surface,
      () =>
        this.service.readFile({ projectId: surface.projectId, resourceId: resource.resourceId }),
      refresh,
    );
  }
  validateNotebook(surfaceId: string, target: NotebookTarget, data: SurfaceData): void {
    if (!this.notebook) throw new AppProblem('unsupported', 'Notebook reading is unavailable.');
    this.notebook.validate(surfaceId, target, data);
  }
  notebookImage(surfaceId: string, target: NotebookTarget, data: SurfaceData): NotebookImage {
    if (!this.notebook) throw new AppProblem('unsupported', 'Notebook reading is unavailable.');
    return this.notebook.image(surfaceId, target, data);
  }
  projects() {
    return this.service.listProjects();
  }
  async read(projectId: string, resource: ResourceRef): Promise<SurfaceData> {
    switch (resource.kind) {
      case 'report':
        if (!this.reports)
          throw new AppProblem('unsupported', 'Saved report reading is unavailable here.');
        return this.reports.read(projectId, resource.saved);
      case 'creation-draft':
        throw new AppProblem('unsupported', 'Use the exact creation review tools for this draft.');
      case 'pipeline':
        return {
          kind: 'pipeline',
          value: await this.service.pipelineInspection({
            projectId,
            pipelineId: resource.pipelineId,
          }),
        };
      case 'file': {
        const value = await this.service.readFile({ projectId, resourceId: resource.resourceId });
        if (value.content.kind === 'notebook')
          throw new AppProblem(
            'unsupported',
            'Read this Notebook through workspace_observe in its ready Project view.',
          );
        if (value.content.kind === 'pdf')
          throw new AppProblem(
            'unsupported',
            'Read this PDF through workspace_observe in its ready Project view.',
          );
        return { kind: 'file', value: { ...value, content: value.content } };
      }
      case 'run': {
        const raw = await this.service.readRun({ projectId, runRef: resource.runRef });
        const value: RunPresentationV2 = presentRun(raw);
        // Capability absence or a failed catalog read must not hide the ordinary Run.
        try {
          const context = continuationContext(
            value,
            raw.snapshot.snapshot,
            await this.service.continuations.list({ projectId }),
          );
          if (context) value.continuation = context;
        } catch {
          /* No continuation overlay without exact native evidence. */
        }
        try {
          return {
            kind: 'run',
            value,
            dependencies: projectDependencies(presentDependencies(raw)),
          };
        } catch {
          return {
            kind: 'run',
            value,
            dependencyProblem:
              'Dependencies are unavailable for this observation. Tasks remain available.',
          };
        }
      }
      case 'log': {
        const target = logTarget(resource);
        return {
          kind: 'log',
          value: presentLog(
            await this.service.readLogs({
              projectId,
              runRef: target.runRef,
              instance: target.instanceId,
              attempt: target.attempt,
            }),
          ),
        };
      }
    }
  }
  async describe(projectId: string, resource: ResourceRef) {
    if (resource.kind === 'report') {
      await this.read(projectId, resource);
      return { title: resource.saved.title, view: 'report' as const };
    }
    if (resource.kind === 'creation-draft') {
      const d = await this.service.drafts.read({ projectId, draftId: resource.draftId });
      return {
        title: d.input
          ? 'New pipeline · ' + d.input.relativePath.split('/').at(-1)
          : 'New pipeline',
        view: 'creation-draft' as const,
      };
    }
    if (resource.kind === 'file') {
      const value = await this.service.readFile({ projectId, resourceId: resource.resourceId });
      return { title: value.name || 'Untitled file', view: value.content.kind };
    }
    if (resource.kind === 'pipeline') {
      const { pipelines } = await this.service.listPipelines({ projectId });
      const pipeline = pipelines.find((value) => value.pipelineId === resource.pipelineId);
      if (!pipeline) throw new AppProblem('not_found', 'Pipeline not found in this Project.');
      return { title: pipeline.name, view: 'pipeline' as const };
    }
    const { runs } = await this.service.listRuns({ projectId });
    const run = runs.find((item) => item.runRef === resource.runRef);
    if (!run) throw new AppProblem('not_found', 'Run not found in this Project.');
    if (resource.kind === 'log') {
      await this.read(projectId, resource);
      const target = logTarget(resource);
      return { title: target.instanceId + ' · attempt ' + target.attempt, view: 'log' as const };
    }
    return { title: run.name || 'Run', view: 'run' as const };
  }
}
