import {
  APP_INFO_CHANNEL,
  AppInfoRequestSchema,
  type AppInfoResult,
  parse,
} from '@gobble/contracts';
import { app, ipcMain, type WebContents } from 'electron';
import { isTrustedSender } from './security/origin';

export function registerAppInfo(
  getWorkspace: () => WebContents | undefined,
  expectedURL: string,
): void {
  ipcMain.handle(APP_INFO_CHANNEL, (event, input: unknown): AppInfoResult => {
    // Resolve frame and role synchronously, before any future asynchronous work.
    const trusted = isTrustedSender(
      {
        isWorkspaceWindow: event.sender === getWorkspace(),
        isMainFrame: event.senderFrame !== null && event.senderFrame === event.sender.mainFrame,
        url: event.senderFrame?.url ?? '',
      },
      expectedURL,
    );
    if (!trusted)
      return {
        schemaVersion: 1,
        ok: false,
        error: { code: 'forbidden', message: 'This view cannot access the app.', retry: 'never' },
      };
    try {
      parse(AppInfoRequestSchema, input);
    } catch {
      return {
        schemaVersion: 1,
        ok: false,
        error: {
          code: 'invalid_request',
          message: 'Unsupported app information request.',
          retry: 'never',
        },
      };
    }
    return {
      schemaVersion: 1,
      ok: true,
      value: {
        name: 'Gobble',
        appVersion: app.getVersion(),
        electronVersion: process.versions.electron,
        stage: 'workspace',
      },
    };
  });
}
