import {
  CONTINUATION_CHANNELS,
  ContinuationResultSchema,
  ContinuationsResultSchema,
  ContinuationSupportResultSchema,
} from '@gobble/contracts';
import { LAUNCH_CHANNELS, LaunchResultSchema, LaunchReviewsResultSchema } from '@gobble/contracts';
import {
  PREPARATION_CHANNELS,
  PreparationResultSchema,
  PreparationEnginesResultSchema,
  PreparationsResultSchema,
} from '@gobble/contracts';
import {
  CREATION_CHANNELS,
  CreationAdoptionResultSchema,
  CreationDraftsResultSchema,
  CreationDraftResultSchema,
  CreationStateResultSchema,
  CreationCandidateResultSchema,
  CreationEnginesResultSchema,
} from '@gobble/contracts';
import {
  PipelineReviewStateResultSchema,
  PipelineReviewSelectedResultSchema,
  PipelineAdoptionResultSchema,
} from '@gobble/contracts';
import { contextBridge, ipcRenderer } from 'electron';
import {
  APP_INFO_CHANNEL,
  AppInfoResultSchema,
  type DesktopBridge,
  parse,
  SERVICE_CHANNELS,
  ProjectsResultSchema,
  ChooseProjectResultSchema,
  DirectoryResultSchema,
  FileResultSchema,
  RunsResultSchema,
  PipelinesResultSchema,
  PipelineResultSchema,
  PipelineInspectionResultSchema,
  ImportPipelineResultSchema,
  RunResultSchema,
  SnapshotResultSchema,
  LogsResultSchema,
  WORKSPACE_CHANNELS,
  WorkspaceResultSchema,
  WorkspaceBootstrapResultSchema,
  SurfaceLoadResultSchema,
  WorkspaceAcknowledgedResultSchema,
  parseWorkspaceDocument,
  WorkspaceShortcutSchema,
  COLLABORATION_CHANNELS,
  CollaborationStatusSchema,
  CollaborationStatusResultSchema,
  CollaborationAcceptedResultSchema,
  EVIDENCE_CHANNELS,
  PreparedEvidenceResultSchema,
  EvidencePreviewResultSchema,
  NotebookImageResultSchema,
} from '@gobble/contracts';

async function invoke<T>(
  channel: string,
  input: unknown,
  validate: (data: unknown) => T,
): Promise<T> {
  try {
    return validate(await ipcRenderer.invoke(channel, input));
  } catch {
    return validate({
      schemaVersion: 1,
      ok: false,
      error: {
        code: 'internal',
        message: 'The app connection is unavailable.',
        retry: 'after_reconnect',
      },
    });
  }
}

const bridge: DesktopBridge = Object.freeze({
  continuations: Object.freeze<DesktopBridge['continuations']>({
    support: (input) =>
      invoke(CONTINUATION_CHANNELS.support, input, (data) =>
        parse(ContinuationSupportResultSchema, data),
      ),
    list: (input) =>
      invoke(CONTINUATION_CHANNELS.list, input, (data) => parse(ContinuationsResultSchema, data)),
    read: (input) =>
      invoke(CONTINUATION_CHANNELS.read, input, (data) => parse(ContinuationResultSchema, data)),
    check: (input) =>
      invoke(CONTINUATION_CHANNELS.check, input, (data) => parse(ContinuationResultSchema, data)),
    confirm: (input) =>
      invoke(CONTINUATION_CHANNELS.confirm, input, (data) => parse(ContinuationResultSchema, data)),
    refresh: (input) =>
      invoke(CONTINUATION_CHANNELS.refresh, input, (data) => parse(ContinuationResultSchema, data)),
    stop: (input) =>
      invoke(CONTINUATION_CHANNELS.stop, input, (data) => parse(ContinuationResultSchema, data)),
  }),
  launches: Object.freeze<DesktopBridge['launches']>({
    list: (input) =>
      invoke(LAUNCH_CHANNELS.list, input, (data) => parse(LaunchReviewsResultSchema, data)),
    read: (input) => invoke(LAUNCH_CHANNELS.read, input, (data) => parse(LaunchResultSchema, data)),
    check: (input) =>
      invoke(LAUNCH_CHANNELS.check, input, (data) => parse(LaunchResultSchema, data)),
    cancel: (input) =>
      invoke(LAUNCH_CHANNELS.cancel, input, (data) => parse(LaunchResultSchema, data)),
    start: (input) =>
      invoke(LAUNCH_CHANNELS.start, input, (data) => parse(LaunchResultSchema, data)),
    refresh: (input) =>
      invoke(LAUNCH_CHANNELS.refresh, input, (data) => parse(LaunchResultSchema, data)),
    stop: (input) => invoke(LAUNCH_CHANNELS.stop, input, (data) => parse(LaunchResultSchema, data)),
  }),
  preparations: Object.freeze<DesktopBridge['preparations']>({
    engines: () =>
      invoke(PREPARATION_CHANNELS.engines, {}, (data) =>
        parse(PreparationEnginesResultSchema, data),
      ),
    list: (input) =>
      invoke(PREPARATION_CHANNELS.list, input, (data) => parse(PreparationsResultSchema, data)),
    read: (input) =>
      invoke(PREPARATION_CHANNELS.read, input, (data) => parse(PreparationResultSchema, data)),
    prepare: (input) =>
      invoke(PREPARATION_CHANNELS.prepare, input, (data) => parse(PreparationResultSchema, data)),
    cancel: (input) =>
      invoke(PREPARATION_CHANNELS.cancel, input, (data) => parse(PreparationResultSchema, data)),
  }),
  creation: Object.freeze<DesktopBridge['creation']>({
    adopt: (input) =>
      invoke(CREATION_CHANNELS.adopt, input, (data) => parse(CreationAdoptionResultSchema, data)),
    outcome: (input) =>
      invoke(CREATION_CHANNELS.outcome, input, (data) => parse(CreationAdoptionResultSchema, data)),
    list: (input) =>
      invoke(CREATION_CHANNELS.list, input, (data) => parse(CreationDraftsResultSchema, data)),
    create: (input) =>
      invoke(CREATION_CHANNELS.create, input, (data) => parse(CreationDraftResultSchema, data)),
    update: (input) =>
      invoke(CREATION_CHANNELS.update, input, (data) => parse(CreationDraftResultSchema, data)),
    discard: (input) =>
      invoke(CREATION_CHANNELS.discard, input, (data) => parse(CreationDraftResultSchema, data)),
    state: (input) =>
      invoke(CREATION_CHANNELS.state, input, (data) => parse(CreationStateResultSchema, data)),
    select: (input) =>
      invoke(CREATION_CHANNELS.select, input, (data) =>
        parse(PipelineReviewSelectedResultSchema, data),
      ),
    cancel: (input) =>
      invoke(CREATION_CHANNELS.cancel, input, (data) => parse(CreationCandidateResultSchema, data)),
    engines: () =>
      invoke(CREATION_CHANNELS.engines, { schemaVersion: 1 }, (data) =>
        parse(CreationEnginesResultSchema, data),
      ),
    connect: (input) =>
      invoke(CREATION_CHANNELS.connect, input, (data) => parse(CreationEnginesResultSchema, data)),
  }),

  pipelineReviews: Object.freeze<DesktopBridge['pipelineReviews']>({
    list: (input) =>
      invoke(SERVICE_CHANNELS.pipelineReviews, input, (data) =>
        parse(PipelineReviewStateResultSchema, data),
      ),
    select: (input) =>
      invoke(SERVICE_CHANNELS.pipelineReviewSelect, input, (data) =>
        parse(PipelineReviewSelectedResultSchema, data),
      ),
    adopt: (input) =>
      invoke(SERVICE_CHANNELS.pipelineAdopt, input, (data) =>
        parse(PipelineAdoptionResultSchema, data),
      ),
    outcome: (input) =>
      invoke(SERVICE_CHANNELS.pipelineAdoption, input, (data) =>
        parse(PipelineAdoptionResultSchema, data),
      ),
  }),
  evidence: Object.freeze<DesktopBridge['evidence']>({
    prepare: (input) =>
      invoke(EVIDENCE_CHANNELS.prepare, input, (value) =>
        parse(PreparedEvidenceResultSchema, value),
      ),
    preview: (input) =>
      invoke(EVIDENCE_CHANNELS.preview, input, (value) =>
        parse(EvidencePreviewResultSchema, value),
      ),
  }),
  collaboration: Object.freeze<DesktopBridge['collaboration']>({
    status: () =>
      invoke(COLLABORATION_CHANNELS.status, { schemaVersion: 1 }, (value) =>
        parse(CollaborationStatusResultSchema, value),
      ),
    account: (input) =>
      invoke(COLLABORATION_CHANNELS.account, input, (value) =>
        parse(CollaborationStatusResultSchema, value),
      ),
    configure: (input) =>
      invoke(COLLABORATION_CHANNELS.configure, input, (value) =>
        parse(CollaborationAcceptedResultSchema, value),
      ),
    send: (input) =>
      invoke(COLLABORATION_CHANNELS.send, input, (value) =>
        parse(CollaborationAcceptedResultSchema, value),
      ),
    agent: (input) =>
      invoke(COLLABORATION_CHANNELS.agent, input, (value) =>
        parse(CollaborationAcceptedResultSchema, value),
      ),
    onChanged: (listener) => {
      const receive = (_event: unknown, raw: unknown) => {
        try {
          listener(parse(CollaborationStatusSchema, raw));
        } catch {
          /* Ignore malformed host events. */
        }
      };
      ipcRenderer.on(COLLABORATION_CHANNELS.changed, receive);
      return () => ipcRenderer.removeListener(COLLABORATION_CHANNELS.changed, receive);
    },
  }),
  workspace: Object.freeze<DesktopBridge['workspace']>({
    onChanged: (listener) => {
      const receive = (_event: unknown, raw: unknown) => {
        try {
          listener(parseWorkspaceDocument(raw));
        } catch {
          /* Ignore malformed host events. */
        }
      };
      ipcRenderer.on(COLLABORATION_CHANNELS.document, receive);
      return () => ipcRenderer.removeListener(COLLABORATION_CHANNELS.document, receive);
    },
    onShortcut: (listener) => {
      const receive = (_event: unknown, input: unknown) => {
        try {
          listener(parse(WorkspaceShortcutSchema, input));
        } catch {
          /* Ignore an invalid host event. */
        }
      };
      ipcRenderer.on(WORKSPACE_CHANNELS.shortcut, receive);
      return () => ipcRenderer.removeListener(WORKSPACE_CHANNELS.shortcut, receive);
    },
    connect: () => invoke(WORKSPACE_CHANNELS.connect, { schemaVersion: 1 }, validateBootstrap),
    read: (input) => invoke(WORKSPACE_CHANNELS.read, input, validateWorkspace),
    openProject: (input) => invoke(WORKSPACE_CHANNELS.openProject, input, validateBootstrap),
    command: (input) => invoke(WORKSPACE_CHANNELS.command, input, validateWorkspace),
    updateDraft: (input) => invoke(WORKSPACE_CHANNELS.updateDraft, input, validateWorkspace),
    notebookImage: (input) =>
      invoke(WORKSPACE_CHANNELS.notebookImage, input, (data) =>
        parse(NotebookImageResultSchema, data),
      ),
    notebookViewport: (input) =>
      invoke(WORKSPACE_CHANNELS.notebookViewport, input, (data) =>
        parse(WorkspaceAcknowledgedResultSchema, data),
      ),
    pdfViewport: (input) =>
      invoke(WORKSPACE_CHANNELS.pdfViewport, input, (data) =>
        parse(WorkspaceAcknowledgedResultSchema, data),
      ),
    present: (input) =>
      invoke(WORKSPACE_CHANNELS.present, input, (data) =>
        parse(WorkspaceAcknowledgedResultSchema, data),
      ),
    interaction: (input) =>
      invoke(WORKSPACE_CHANNELS.interaction, input, (data) =>
        parse(WorkspaceAcknowledgedResultSchema, data),
      ),
    loadSurface: (input) =>
      invoke(WORKSPACE_CHANNELS.loadSurface, input, (data) => parse(SurfaceLoadResultSchema, data)),
    acknowledge: (input) =>
      invoke(WORKSPACE_CHANNELS.acknowledge, input, (data) =>
        parse(WorkspaceAcknowledgedResultSchema, data),
      ),
    invalidate: (input) =>
      invoke(WORKSPACE_CHANNELS.invalidate, input, (data) =>
        parse(WorkspaceAcknowledgedResultSchema, data),
      ),
  }),
  projects: Object.freeze<DesktopBridge['projects']>({
    list: () =>
      invoke(SERVICE_CHANNELS.projectsList, { schemaVersion: 1 }, (data) =>
        parse(ProjectsResultSchema, data),
      ),
    chooseFolder: (input) =>
      invoke(SERVICE_CHANNELS.projectsChoose, input, (data) =>
        parse(ChooseProjectResultSchema, data),
      ),
  }),
  files: Object.freeze<DesktopBridge['files']>({
    list: (input) =>
      invoke(SERVICE_CHANNELS.filesList, input, (data) => parse(DirectoryResultSchema, data)),
    read: (input) =>
      invoke(SERVICE_CHANNELS.filesRead, input, (data) => parse(FileResultSchema, data)),
  }),
  pipelines: Object.freeze<DesktopBridge['pipelines']>({
    importFolder: (input) =>
      invoke(SERVICE_CHANNELS.pipelineImport, input, (data) =>
        parse(ImportPipelineResultSchema, data),
      ),
    inspection: (input) =>
      invoke(SERVICE_CHANNELS.pipelineInspection, input, (data) =>
        parse(PipelineInspectionResultSchema, data),
      ),
    check: (input) =>
      invoke(SERVICE_CHANNELS.pipelineCheck, input, (data) =>
        parse(PipelineInspectionResultSchema, data),
      ),
    cancelCheck: (input) =>
      invoke(SERVICE_CHANNELS.pipelineCancelCheck, input, (data) =>
        parse(PipelineInspectionResultSchema, data),
      ),
    list: (input) =>
      invoke(SERVICE_CHANNELS.pipelinesList, input, (data) => parse(PipelinesResultSchema, data)),
    register: (input) =>
      invoke(SERVICE_CHANNELS.pipelinesRegister, input, (data) =>
        parse(PipelineResultSchema, data),
      ),
  }),
  runs: Object.freeze<DesktopBridge['runs']>({
    list: (input) =>
      invoke(SERVICE_CHANNELS.runsList, input, (data) => parse(RunsResultSchema, data)),
    attach: (input) =>
      invoke(SERVICE_CHANNELS.runsAttach, input, (data) => parse(RunResultSchema, data)),
    snapshot: (input) =>
      invoke(SERVICE_CHANNELS.runsSnapshot, input, (data) => parse(SnapshotResultSchema, data)),
    logs: (input) =>
      invoke(SERVICE_CHANNELS.runsLogs, input, (data) => parse(LogsResultSchema, data)),
  }),
  async getAppInfo() {
    try {
      const result: unknown = await ipcRenderer.invoke(APP_INFO_CHANNEL, { schemaVersion: 1 });
      return parse(AppInfoResultSchema, result);
    } catch {
      return {
        schemaVersion: 1,
        ok: false,
        error: {
          code: 'internal',
          message: 'The app connection is unavailable.',
          retry: 'after_reconnect',
        },
      };
    }
  },
});

contextBridge.exposeInMainWorld('gobble', bridge);

function validateWorkspace(data: unknown) {
  const result = parse(WorkspaceResultSchema, data);
  if (result.ok) parseWorkspaceDocument(result.value);
  return result;
}
function validateBootstrap(data: unknown) {
  const result = parse(WorkspaceBootstrapResultSchema, data);
  if (result.ok && result.value.document) parseWorkspaceDocument(result.value.document);
  return result;
}
