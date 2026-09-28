import { RunContinuationService } from './run-continuation';
import { RunLaunchService } from './run-launch';
import { RunPreparationService } from './run-preparation';
import { PipelineCreationService } from './pipeline-creation';
import { PipelineDraftService } from './pipeline-drafts';
import { PipelineProposalService } from './pipeline-proposals';
import {
  PipelineInspectionResultSchema,
  validatePipelineFlow,
  type PipelineInspectionInput,
  type CheckPipelineInput,
  type CancelPipelineCheckInput,
  parse,
  ProjectsResultSchema,
  ProjectResultSchema,
  DirectoryResultSchema,
  FileResultSchema,
  RunsResultSchema,
  PipelinesResultSchema,
  PipelineResultSchema,
  type RegisterPipelineInput,
  RunResultSchema,
  SnapshotResultSchema,
  LogsResultSchema,
  RunReportResultSchema,
  type DirectoryInput,
  type FileInput,
  type ProjectInput,
  type AttachRunInput,
  type SnapshotInput,
  type LogsInput,
  type ContractError,
} from '@gobble/contracts';
import { AppProblem } from '../problem';
import type { ProjectServiceClient } from './client';

type Result<T> = { ok: true; value: T } | { ok: false; error: ContractError };
function unwrap<T>(result: Result<T>): T {
  if (!result.ok) throw new AppProblem(result.error.code, result.error.message);
  return result.value;
}
function association(actual: string, expected: string): void {
  if (actual !== expected)
    throw new AppProblem('internal', 'The service returned data for a different resource.');
}

/** Named native-service API. Owns routes, response validation and association; not process lifetime. */
export class ProjectService {
  readonly proposals: PipelineProposalService;
  readonly drafts: PipelineDraftService;
  readonly creation: PipelineCreationService;
  readonly launches: RunLaunchService;
  readonly continuations: RunContinuationService;
  readonly preparations: RunPreparationService;
  constructor(private readonly transport: Pick<ProjectServiceClient, 'request'>) {
    this.proposals = new PipelineProposalService(transport);
    this.drafts = new PipelineDraftService(transport);
    this.creation = new PipelineCreationService(transport);
    this.launches = new RunLaunchService(transport);
    this.continuations = new RunContinuationService(transport);
    this.preparations = new RunPreparationService(transport);
  }

  async listProjects() {
    return unwrap(parse(ProjectsResultSchema, await this.transport.request('/v1/projects')));
  }
  async registerProject(input: { requestId: string; root: string; name: string }) {
    return unwrap(
      parse(ProjectResultSchema, await this.transport.request('/v1/projects', 'POST', input)),
    );
  }
  async listFiles(input: DirectoryInput) {
    const query = new URLSearchParams(input.directoryId ? { directoryId: input.directoryId } : {});
    const value = unwrap(
      parse(
        DirectoryResultSchema,
        await this.transport.request(`/v1/projects/${input.projectId}/files?${query}`),
      ),
    );
    association(value.projectId, input.projectId);
    if (input.directoryId) association(value.directoryId, input.directoryId);
    return value;
  }
  async readFile(input: FileInput) {
    const query = new URLSearchParams(
      input.expectedRevision ? { expectedRevision: input.expectedRevision } : {},
    );
    const value = unwrap(
      parse(
        FileResultSchema,
        await this.transport.request(
          `/v1/projects/${input.projectId}/files/${input.resourceId}?${query}`,
        ),
      ),
    );
    association(value.projectId, input.projectId);
    association(value.resourceId, input.resourceId);
    if (input.expectedRevision) association(value.revision, input.expectedRevision);
    return value;
  }
  private async pipelineInspectionRequest(
    input: PipelineInspectionInput,
    operation: 'inspection' | 'check' | 'cancel',
    payload?: unknown,
  ) {
    const value = unwrap(
      parse(
        PipelineInspectionResultSchema,
        await this.transport.request(
          `/v1/projects/${input.projectId}/pipelines/${input.pipelineId}/${operation}`,
          operation === 'inspection' ? 'GET' : 'POST',
          payload,
        ),
      ),
    );
    association(value.projectId, input.projectId);
    association(value.pipelineId, input.pipelineId);
    if (value.artifact) validatePipelineFlow(value.artifact.flow);
    if (value.state === 'ready' && !value.artifact)
      throw new AppProblem('internal', 'The checked pipeline has no flow.');
    return value;
  }
  pipelineInspection(input: PipelineInspectionInput) {
    return this.pipelineInspectionRequest(input, 'inspection');
  }
  async checkPipeline(input: CheckPipelineInput) {
    const value = await this.pipelineInspectionRequest(input, 'check', {
      requestId: input.requestId,
    });
    association(value.jobId ?? '', input.requestId);
    return value;
  }
  async cancelPipelineCheck(input: CancelPipelineCheckInput) {
    const value = await this.pipelineInspectionRequest(input, 'cancel', { jobId: input.jobId });
    association(value.jobId ?? '', input.jobId);
    return value;
  }
  async importPipeline(input: { projectId: string; requestId: string; path: string }) {
    const { projectId, ...selection } = input;
    const value = unwrap(
      parse(
        PipelineResultSchema,
        await this.transport.request(
          `/v1/projects/${projectId}/pipelines/import`,
          'POST',
          selection,
        ),
      ),
    );
    association(value.projectId, projectId);
    return value;
  }
  async listPipelines(input: ProjectInput) {
    const value = unwrap(
      parse(
        PipelinesResultSchema,
        await this.transport.request(`/v1/projects/${input.projectId}/pipelines`),
      ),
    );
    association(value.projectId, input.projectId);
    const ids = new Set<string>();
    const packages = new Set<string>();
    for (const pipeline of value.pipelines) {
      association(pipeline.projectId, input.projectId);
      if (
        ids.has(pipeline.pipelineId) ||
        (pipeline.origin.kind === 'imported' && packages.has(pipeline.origin.packageResourceId))
      )
        throw new AppProblem('internal', 'The service returned duplicate Pipeline registrations.');
      ids.add(pipeline.pipelineId);
      if (pipeline.origin.kind === 'imported') packages.add(pipeline.origin.packageResourceId);
    }
    return value;
  }
  async registerPipeline(input: RegisterPipelineInput) {
    const { projectId, ...registration } = input;
    const value = unwrap(
      parse(
        PipelineResultSchema,
        await this.transport.request(`/v1/projects/${projectId}/pipelines`, 'POST', registration),
      ),
    );
    association(value.projectId, projectId);
    if (value.origin.kind !== 'imported')
      throw new AppProblem('internal', 'The service returned a different resource origin.');
    association(value.origin.packageResourceId, input.packageResourceId);
    return value;
  }
  async listRuns(input: ProjectInput) {
    const value = unwrap(
      parse(RunsResultSchema, await this.transport.request(`/v1/projects/${input.projectId}/runs`)),
    );
    association(value.projectId, input.projectId);
    for (const run of value.runs) association(run.projectId, input.projectId);
    return value;
  }
  async attachRun(input: AttachRunInput) {
    const value = unwrap(
      parse(
        RunResultSchema,
        await this.transport.request(`/v1/projects/${input.projectId}/runs/attach`, 'POST', {
          requestId: input.requestId,
          workspaceResourceId: input.workspaceResourceId,
        }),
      ),
    );
    association(value.projectId, input.projectId);
    association(value.workspaceResourceId, input.workspaceResourceId);
    return value;
  }
  async readRun(input: SnapshotInput) {
    const value = unwrap(
      parse(
        SnapshotResultSchema,
        await this.transport.request(
          `/v1/projects/${input.projectId}/runs/${input.runRef}/snapshot`,
        ),
      ),
    );
    association(value.projectId, input.projectId);
    association(value.runRef, input.runRef);
    association(value.snapshot.snapshot, value.engineRevision);
    return value;
  }
  async readReport(input: LogsInput) {
    const query = new URLSearchParams({ instance: input.instance, attempt: String(input.attempt) });
    const value = unwrap(
      parse(
        RunReportResultSchema,
        await this.transport.request(
          `/v1/projects/${input.projectId}/runs/${input.runRef}/report?${query}`,
        ),
      ),
    );
    association(value.projectId, input.projectId);
    association(value.runRef, input.runRef);
    association(value.evidence.instance, input.instance);
    if (value.evidence.attempt !== input.attempt)
      throw new AppProblem('stale_revision', 'The service returned another report attempt.');
    return value;
  }
  async readLogs(input: LogsInput) {
    const query = new URLSearchParams({ instance: input.instance, attempt: String(input.attempt) });
    const value = unwrap(
      parse(
        LogsResultSchema,
        await this.transport.request(
          `/v1/projects/${input.projectId}/runs/${input.runRef}/logs?${query}`,
        ),
      ),
    );
    association(value.projectId, input.projectId);
    association(value.runRef, input.runRef);
    association(value.instance, input.instance);
    if (value.attempt !== input.attempt)
      throw new AppProblem('stale_revision', 'The service returned another log attempt.');
    return value;
  }
}
