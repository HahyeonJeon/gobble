import { ipcMain, type BrowserWindow } from 'electron';
import {
  AppInfoRequestSchema,
  parse,
  RenderAcknowledgmentSchema,
  WORKSPACE_CHANNELS,
  WorkspaceCommandSchema,
  WorkspaceDraftInputSchema,
  WorkspaceLoadInputSchema,
  WorkspaceInteractionInputSchema,
  WorkspacePresentationInputSchema,
  PdfViewportInputSchema,
  NotebookViewportInputSchema,
  NotebookImageInputSchema,
  WorkspaceProjectInputSchema,
  WorkspaceReadInputSchema,
  type ContractError,
} from '@gobble/contracts';
import { isTrustedSender } from '../security/origin';
import type { WorkspaceController } from './controller';
import { AppProblem } from '../problem';

export function registerWorkspace(
  controller: WorkspaceController,
  getWindow: () => BrowserWindow | undefined,
  expectedURL: string,
): void {
  function register<Input>(
    channel: string,
    validate: (input: unknown) => Input,
    handle: (input: Input) => Promise<unknown>,
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
        return failure('forbidden', 'This view cannot access the workspace.');
      let value: Input;
      try {
        value = validate(input);
      } catch {
        return failure('invalid_request', 'The workspace request is invalid.');
      }
      return handle(value)
        .then((result) => ({ schemaVersion: 1, ok: true, value: result }))
        .catch((error: unknown) =>
          error instanceof AppProblem
            ? failure(error.code, error.message)
            : failure('internal', 'The workspace request could not be completed. Try again.'),
        );
    });
  }
  register(
    WORKSPACE_CHANNELS.read,
    (input) => parse(WorkspaceReadInputSchema, input),
    (input) => controller.read(input.projectId),
  );
  register(
    WORKSPACE_CHANNELS.connect,
    (input) => parse(AppInfoRequestSchema, input),
    () => controller.connect(),
  );
  register(
    WORKSPACE_CHANNELS.openProject,
    (input) => parse(WorkspaceProjectInputSchema, input),
    (input) => controller.openProject(input.projectId),
  );
  register(
    WORKSPACE_CHANNELS.command,
    (input) => parse(WorkspaceCommandSchema, input),
    (input) => controller.command(input),
  );
  register(
    WORKSPACE_CHANNELS.updateDraft,
    (input) => parse(WorkspaceDraftInputSchema, input),
    (input) => controller.updateDraft(input.projectId, input.draft),
  );
  register(
    WORKSPACE_CHANNELS.notebookImage,
    (input) => parse(NotebookImageInputSchema, input),
    (input) => controller.notebookImage(input),
  );
  register(
    WORKSPACE_CHANNELS.notebookViewport,
    (input) => parse(NotebookViewportInputSchema, input),
    (input) => controller.notebookViewport(input),
  );
  register(
    WORKSPACE_CHANNELS.pdfViewport,
    (input) => parse(PdfViewportInputSchema, input),
    (input) => controller.pdfViewport(input),
  );
  register(
    WORKSPACE_CHANNELS.present,
    (input) => parse(WorkspacePresentationInputSchema, input),
    (input) => controller.present(input),
  );
  register(
    WORKSPACE_CHANNELS.interaction,
    (input) => parse(WorkspaceInteractionInputSchema, input),
    (input) => controller.interaction(input),
  );
  register(
    WORKSPACE_CHANNELS.loadSurface,
    (input) => parse(WorkspaceLoadInputSchema, input),
    (input) => controller.loadSurface(input),
  );
  register(
    WORKSPACE_CHANNELS.acknowledge,
    (input) => parse(RenderAcknowledgmentSchema, input),
    (input) => controller.acknowledge(input),
  );
  register(
    WORKSPACE_CHANNELS.invalidate,
    (input) => parse(RenderAcknowledgmentSchema, input),
    (input) => controller.invalidate(input),
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
        code === 'stale_revision'
          ? 'after_refresh'
          : code === 'runtime_unavailable'
            ? 'after_reconnect'
            : 'never',
    },
  };
}
