import {
  CONTINUATION_CHANNELS,
  ContinuationProjectInputSchema,
  ContinuationSupportInputSchema,
  ContinuationInputSchema,
  ContinuationCheckInputSchema,
  ContinuationConfirmInputSchema,
  ContinuationStopInputSchema,
} from '@gobble/contracts';
import {
  LAUNCH_CHANNELS,
  LaunchProjectInputSchema,
  LaunchReviewInputSchema,
  LaunchCheckInputSchema,
  LaunchStartInputSchema,
  LaunchStopInputSchema,
} from '@gobble/contracts';
import {
  PREPARATION_CHANNELS,
  PreparationEnginesInputSchema,
  PreparationInputSchema,
  PreparePipelineInputSchema,
} from '@gobble/contracts';
import {
  CREATION_CHANNELS,
  AdoptCreationInputSchema,
  CreationAdoptionInputSchema,
  CreatePipelineDraftInputSchema,
  UpdatePipelineDraftInputSchema,
  DiscardPipelineDraftInputSchema,
  PipelineDraftInputSchema,
  CreationSelectSchema,
  CreationCandidateInputSchema,
  CreationConnectSchema,
} from '@gobble/contracts';
import type { PipelineReviewHost } from '../pipeline-review/host';
import {
  PipelineAdoptInputSchema,
  PipelineAdoptionInputSchema,
  PipelineReviewSelectSchema,
} from '@gobble/contracts';
import { dialog, ipcMain, type BrowserWindow } from 'electron';
import {
  ImportPipelineInputSchema,
  PipelineInspectionInputSchema,
  CheckPipelineInputSchema,
  CancelPipelineCheckInputSchema,
  parse,
  SERVICE_CHANNELS,
  ChooseProjectInputSchema,
  ProjectInputSchema,
  DirectoryInputSchema,
  FileInputSchema,
  AttachRunInputSchema,
  RegisterPipelineInputSchema,
  SnapshotInputSchema,
  LogsInputSchema,
  AppInfoRequestSchema,
  type ContractError,
} from '@gobble/contracts';
import { AppProblem } from '../problem';
import { isTrustedSender } from '../security/origin';
import type { ProjectService } from './project-service';

export function registerProjectService(
  service: ProjectService,
  getWindow: () => BrowserWindow | undefined,
  expectedURL: string,
  reviewHost?: () => PipelineReviewHost,
): void {
  function register<Input>(
    channel: string,
    validate: (input: unknown) => Input,
    handle: (input: Input, window: BrowserWindow) => Promise<unknown>,
  ) {
    ipcMain.handle(channel, (event, input: unknown) => {
      const window = getWindow();
      if (
        !window ||
        window.isDestroyed() ||
        !isTrustedSender(
          {
            isWorkspaceWindow: event.sender === window.webContents,
            isMainFrame: event.senderFrame !== null && event.senderFrame === event.sender.mainFrame,
            url: event.senderFrame?.url ?? '',
          },
          expectedURL,
        )
      )
        return failure('forbidden', 'This view cannot access Projects.');
      let value: Input;
      try {
        value = validate(input);
      } catch {
        return failure('invalid_request', 'The Project request is invalid.');
      }
      return handle(value, window)
        .then((value) => ({ schemaVersion: 1, ok: true, value }))
        .catch((error: unknown) =>
          error instanceof AppProblem
            ? failure(error.code, error.message)
            : failure(
                'runtime_unavailable',
                'The Project service is unavailable. Try again after reconnecting.',
              ),
        );
    });
  }
  register(
    CONTINUATION_CHANNELS.support,
    (input) => parse(ContinuationSupportInputSchema, input),
    (input) => service.continuations.support(input),
  );
  register(
    CONTINUATION_CHANNELS.list,
    (input) => parse(ContinuationProjectInputSchema, input),
    (input) => service.continuations.list(input),
  );
  register(
    CONTINUATION_CHANNELS.read,
    (input) => parse(ContinuationInputSchema, input),
    (input) => service.continuations.read(input),
  );
  register(
    CONTINUATION_CHANNELS.check,
    (input) => parse(ContinuationCheckInputSchema, input),
    (input) => service.continuations.check(input),
  );
  register(
    CONTINUATION_CHANNELS.confirm,
    (input) => parse(ContinuationConfirmInputSchema, input),
    (input) => service.continuations.confirm(input),
  );
  register(
    CONTINUATION_CHANNELS.refresh,
    (input) => parse(ContinuationInputSchema, input),
    (input) => service.continuations.refresh(input),
  );
  register(
    CONTINUATION_CHANNELS.stop,
    (input) => parse(ContinuationStopInputSchema, input),
    (input) => service.continuations.stop(input),
  );
  register(
    LAUNCH_CHANNELS.list,
    (input) => parse(LaunchProjectInputSchema, input),
    (input) => service.launches.list(input),
  );
  register(
    LAUNCH_CHANNELS.read,
    (input) => parse(LaunchReviewInputSchema, input),
    (input) => service.launches.read(input),
  );
  register(
    LAUNCH_CHANNELS.check,
    (input) => parse(LaunchCheckInputSchema, input),
    (input) => service.launches.check(input),
  );
  register(
    LAUNCH_CHANNELS.cancel,
    (input) => parse(LaunchReviewInputSchema, input),
    (input) => service.launches.cancel(input),
  );
  register(
    LAUNCH_CHANNELS.start,
    (input) => parse(LaunchStartInputSchema, input),
    (input) => service.launches.start(input),
  );
  register(
    LAUNCH_CHANNELS.refresh,
    (input) => parse(LaunchReviewInputSchema, input),
    (input) => service.launches.refresh(input),
  );
  register(
    LAUNCH_CHANNELS.stop,
    (input) => parse(LaunchStopInputSchema, input),
    (input) => service.launches.stop(input),
  );
  register(
    PREPARATION_CHANNELS.engines,
    (input) => parse(PreparationEnginesInputSchema, input),
    () => service.preparations.engines(),
  );
  register(
    PREPARATION_CHANNELS.list,
    (input) => parse(PipelineInspectionInputSchema, input),
    (input) => service.preparations.list(input),
  );
  register(
    PREPARATION_CHANNELS.read,
    (input) => parse(PreparationInputSchema, input),
    (input) => service.preparations.read(input),
  );
  register(
    PREPARATION_CHANNELS.prepare,
    (input) => parse(PreparePipelineInputSchema, input),
    (input) => service.preparations.prepare(input),
  );
  register(
    PREPARATION_CHANNELS.cancel,
    (input) => parse(PreparationInputSchema, input),
    (input) => service.preparations.cancel(input),
  );
  register(
    CREATION_CHANNELS.adopt,
    (input) => parse(AdoptCreationInputSchema, input),
    (input) => service.creation.adopt(input),
  );
  register(
    CREATION_CHANNELS.outcome,
    (input) => parse(CreationAdoptionInputSchema, input),
    (input) => service.creation.outcome(input),
  );
  register(
    CREATION_CHANNELS.list,
    (input) => parse(ProjectInputSchema, input),
    (input) => service.drafts.list(input),
  );
  register(
    CREATION_CHANNELS.create,
    (input) => parse(CreatePipelineDraftInputSchema, input),
    (input) => service.drafts.create(input),
  );
  register(
    CREATION_CHANNELS.update,
    (input) => parse(UpdatePipelineDraftInputSchema, input),
    (input) => service.drafts.update(input),
  );
  register(
    CREATION_CHANNELS.discard,
    (input) => parse(DiscardPipelineDraftInputSchema, input),
    (input) => service.drafts.discard(input),
  );
  register(
    CREATION_CHANNELS.state,
    (input) => parse(PipelineDraftInputSchema, input),
    (input) => reviewHost!().creation.state(input),
  );
  register(
    CREATION_CHANNELS.select,
    (input) => parse(CreationSelectSchema, input),
    (input) => reviewHost!().creation.select(input),
  );
  register(
    CREATION_CHANNELS.cancel,
    (input) => parse(CreationCandidateInputSchema, input),
    (input) => service.creation.cancel(input),
  );
  register(
    CREATION_CHANNELS.engines,
    (input) => parse(AppInfoRequestSchema, input),
    () => service.creation.engines(),
  );
  register(
    CREATION_CHANNELS.connect,
    (input) => parse(CreationConnectSchema, input),
    (input) => service.creation.connect(input),
  );
  register(
    SERVICE_CHANNELS.pipelineReviews,
    (input) => parse(PipelineInspectionInputSchema, input),
    (input) => reviewHost!().list(input),
  );
  register(
    SERVICE_CHANNELS.pipelineReviewSelect,
    (input) => parse(PipelineReviewSelectSchema, input),
    (input) => reviewHost!().select(input),
  );
  register(
    SERVICE_CHANNELS.pipelineAdopt,
    (input) => parse(PipelineAdoptInputSchema, input),
    (input) => service.proposals.adopt(input),
  );
  register(
    SERVICE_CHANNELS.pipelineAdoption,
    (input) => parse(PipelineAdoptionInputSchema, input),
    (input) => service.proposals.outcome(input),
  );
  register(
    SERVICE_CHANNELS.projectsList,
    (input) => parse(AppInfoRequestSchema, input),
    () => service.listProjects(),
  );
  register(
    SERVICE_CHANNELS.projectsChoose,
    (input) => parse(ChooseProjectInputSchema, input),
    async (input, window) => {
      const selection = await dialog.showOpenDialog(window, {
        title: 'Choose a Project folder',
        buttonLabel: 'Open Project',
        properties: ['openDirectory'],
      });
      const root = selection.filePaths[0];
      if (selection.canceled || !root || window.isDestroyed()) return { kind: 'cancelled' };
      const project = await service.registerProject({ requestId: input.requestId, root, name: '' });
      return { kind: 'selected', project };
    },
  );
  register(
    SERVICE_CHANNELS.filesList,
    (input) => parse(DirectoryInputSchema, input),
    (input) => service.listFiles(input),
  );
  register(
    SERVICE_CHANNELS.filesRead,
    (input) => parse(FileInputSchema, input),
    async (input) => {
      const file = await service.readFile(input);
      if (file.content.kind === 'notebook')
        throw new AppProblem('unsupported', 'Open this Notebook in a Project pane.');
      if (file.content.kind === 'pdf')
        throw new AppProblem('unsupported', 'Open this PDF in a Project pane.');
      return file;
    },
  );
  register(
    SERVICE_CHANNELS.pipelineImport,
    (input) => parse(ImportPipelineInputSchema, input),
    async (input, window) => {
      const selection = await dialog.showOpenDialog(window, {
        title: 'Choose an analysis folder in this Project',
        buttonLabel: 'Import pipeline',
        properties: ['openDirectory'],
      });
      const path = selection.filePaths[0];
      if (selection.canceled || !path || window.isDestroyed()) return { kind: 'cancelled' };
      return { kind: 'selected', pipeline: await service.importPipeline({ ...input, path }) };
    },
  );
  register(
    SERVICE_CHANNELS.pipelineInspection,
    (input) => parse(PipelineInspectionInputSchema, input),
    (input) => service.pipelineInspection(input),
  );
  register(
    SERVICE_CHANNELS.pipelineCheck,
    (input) => parse(CheckPipelineInputSchema, input),
    (input) => service.checkPipeline(input),
  );
  register(
    SERVICE_CHANNELS.pipelineCancelCheck,
    (input) => parse(CancelPipelineCheckInputSchema, input),
    (input) => service.cancelPipelineCheck(input),
  );
  register(
    SERVICE_CHANNELS.pipelinesList,
    (input) => parse(ProjectInputSchema, input),
    (input) => service.listPipelines(input),
  );
  register(
    SERVICE_CHANNELS.pipelinesRegister,
    (input) => parse(RegisterPipelineInputSchema, input),
    (input) => service.registerPipeline(input),
  );
  register(
    SERVICE_CHANNELS.runsList,
    (input) => parse(ProjectInputSchema, input),
    (input) => service.listRuns(input),
  );
  register(
    SERVICE_CHANNELS.runsAttach,
    (input) => parse(AttachRunInputSchema, input),
    (input) => service.attachRun(input),
  );
  register(
    SERVICE_CHANNELS.runsSnapshot,
    (input) => parse(SnapshotInputSchema, input),
    (input) => service.readRun(input),
  );
  register(
    SERVICE_CHANNELS.runsLogs,
    (input) => parse(LogsInputSchema, input),
    (input) => service.readLogs(input),
  );
}

function failure(code: ContractError['code'], message: string) {
  return {
    schemaVersion: 1,
    ok: false,
    error: {
      code,
      message,
      retry:
        code === 'runtime_unavailable'
          ? 'after_reconnect'
          : code === 'stale_revision'
            ? 'after_refresh'
            : 'never',
    },
  };
}
