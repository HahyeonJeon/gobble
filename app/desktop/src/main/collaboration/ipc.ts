import { ipcMain, type BrowserWindow } from 'electron';
import {
  AccountActionSchema,
  AgentActionSchema,
  AppInfoRequestSchema,
  COLLABORATION_CHANNELS,
  ConfigureAgentSchema,
  SendMessageSchema,
  EVIDENCE_CHANNELS,
  PrepareEvidenceSchema,
  PreviewEvidenceSchema,
  parse,
  type ContractError,
} from '@gobble/contracts';
import { isTrustedSender } from '../security/origin';
import { AppProblem } from '../problem';
import type { WorkspaceController } from '../workspace/controller';
import type { CollaborationHost } from './host';

export function registerCollaboration(
  host: CollaborationHost,
  workspace: WorkspaceController,
  getWindow: () => BrowserWindow | undefined,
  expectedURL: string,
): () => void {
  const send = (channel: string, value: unknown) => {
    const window = getWindow();
    if (window && !window.isDestroyed() && window.webContents.getURL() === expectedURL)
      window.webContents.send(channel, value);
  };
  const unsubscribe = [
    host.onChanged((value) => send(COLLABORATION_CHANNELS.changed, value)),
    workspace.onDocument((value) => send(COLLABORATION_CHANNELS.document, value)),
  ];
  function register<T>(
    channel: string,
    validate: (value: unknown) => T,
    handle: (value: T) => unknown | Promise<unknown>,
  ) {
    ipcMain.handle(channel, (event, raw: unknown) => {
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
        return failure('forbidden', 'This view cannot access agents.');
      let input: T;
      try {
        input = validate(raw);
      } catch {
        return failure('invalid_request', 'The agent request is invalid.');
      }
      return Promise.resolve()
        .then(() => handle(input))
        .then((value) => ({ schemaVersion: 1, ok: true, value }))
        .catch((error: unknown) =>
          error instanceof AppProblem
            ? failure(error.code, error.message)
            : failure(
                'internal',
                'The agent request could not be completed. Check status before trying again.',
              ),
        );
    });
  }
  register(
    COLLABORATION_CHANNELS.status,
    (value) => parse(AppInfoRequestSchema, value),
    () => host.status(),
  );
  register(
    COLLABORATION_CHANNELS.account,
    (value) => parse(AccountActionSchema, value),
    (value) => host.accountAction(value),
  );
  register(
    COLLABORATION_CHANNELS.configure,
    (value) => parse(ConfigureAgentSchema, value),
    async (value) => {
      await host.coordinator.configure(value);
      return { accepted: true };
    },
  );
  register(
    COLLABORATION_CHANNELS.send,
    (value) => parse(SendMessageSchema, value),
    async (value) => {
      await host.coordinator.send(value);
      return { accepted: true };
    },
  );
  register(
    COLLABORATION_CHANNELS.agent,
    (value) => parse(AgentActionSchema, value),
    async (value) => {
      await host.coordinator.agentAction(value);
      return { accepted: true };
    },
  );
  register(
    EVIDENCE_CHANNELS.prepare,
    (value) => parse(PrepareEvidenceSchema, value),
    (value) => {
      if (!host.evidence)
        throw new AppProblem('unsupported', 'Attachment preparation is unavailable.');
      return host.evidence.prepare(value);
    },
  );
  register(
    EVIDENCE_CHANNELS.preview,
    (value) => parse(PreviewEvidenceSchema, value),
    (value) => {
      if (!host.evidence) throw new AppProblem('unsupported', 'Evidence preview is unavailable.');
      return host.evidence.preview(value);
    },
  );
  return () => {
    for (const stop of unsubscribe) stop();
    for (const channel of [
      COLLABORATION_CHANNELS.status,
      COLLABORATION_CHANNELS.account,
      COLLABORATION_CHANNELS.configure,
      COLLABORATION_CHANNELS.send,
      COLLABORATION_CHANNELS.agent,
      EVIDENCE_CHANNELS.prepare,
      EVIDENCE_CHANNELS.preview,
    ])
      ipcMain.removeHandler(channel);
  };
}
function failure(code: ContractError['code'], message: string) {
  return {
    schemaVersion: 1,
    ok: false,
    error: { code, message, retry: code === 'runtime_unavailable' ? 'after_reconnect' : 'never' },
  };
}
