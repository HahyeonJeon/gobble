import { BrowserWindow, session } from 'electron';
import { readFile } from 'node:fs/promises';
import { randomUUID } from 'node:crypto';
import { FastqcContentSchema, parse, type FastqcContent, REPORT_BYTES } from '@gobble/contracts';
import { AppProblem } from './problem';

/** Each bounded job owns a disposable sandbox with no preload, network or App bridge. */
export class FastqcReader {
  private jobs = new Set<() => void>();
  private closed = false;
  constructor(private readonly scriptPath: string) {}
  async read(source: Buffer): Promise<FastqcContent> {
    if (this.closed || !source.length || source.length > REPORT_BYTES)
      throw new AppProblem(
        'unsupported',
        'Quality report reading is unavailable or exceeds its limit.',
      );
    let text: string;
    try {
      text = new TextDecoder('utf-8', { fatal: true }).decode(source);
    } catch {
      throw new AppProblem('unsupported', 'The report source is not valid UTF-8.');
    }
    const script = await readFile(this.scriptPath, 'utf8');
    if (this.closed) throw new AppProblem('unsupported', 'Report reader closed.');
    const isolated = session.fromPartition('report-' + randomUUID(), { cache: false });
    isolated.setPermissionCheckHandler(() => false);
    isolated.setPermissionRequestHandler((_wc, _permission, callback) => callback(false));
    isolated.on('will-download', (event) => event.preventDefault());
    const bootstrap =
      'data:text/html,' +
      encodeURIComponent(
        `<meta http-equiv="Content-Security-Policy" content="default-src 'none'; base-uri 'none'; form-action 'none'">`,
      );
    isolated.webRequest.onBeforeRequest((details, callback) =>
      callback({ cancel: details.resourceType !== 'mainFrame' || details.url !== bootstrap }),
    );
    const window = new BrowserWindow({
      show: false,
      webPreferences: {
        session: isolated,
        sandbox: true,
        contextIsolation: true,
        nodeIntegration: false,
        webSecurity: true,
        devTools: false,
      },
    });
    window.webContents.setWindowOpenHandler(() => ({ action: 'deny' }));
    window.webContents.on('will-navigate', (event) => event.preventDefault());
    window.webContents.on('will-attach-webview', (event) => event.preventDefault());
    let timer: ReturnType<typeof setTimeout> | undefined;
    let cancel!: () => void;
    const deadline = new Promise<never>((_resolve, reject) => {
      cancel = () =>
        reject(
          new AppProblem(
            'unsupported',
            'The quality report could not be read completely within its limits.',
          ),
        );
      timer = setTimeout(cancel, 10_000);
    });
    this.jobs.add(cancel);
    window.webContents.on('render-process-gone', cancel);
    try {
      return await Promise.race([
        deadline,
        (async () => {
          await window.loadURL(bootstrap);
          isolated.webRequest.onBeforeRequest((_details, callback) => callback({ cancel: true }));
          await window.webContents.executeJavaScript(script);
          const result: unknown = await window.webContents.executeJavaScript(
            'FastqcDecoder.read(' + JSON.stringify(text) + ')',
          );
          return parse(FastqcContentSchema, result);
        })(),
      ]);
    } catch {
      throw new AppProblem(
        'unsupported',
        'This quality report is incomplete, unsupported, or exceeds the reading limits. No partial report was saved.',
      );
    } finally {
      clearTimeout(timer);
      this.jobs.delete(cancel);
      if (!window.isDestroyed()) window.destroy();
      await isolated.clearStorageData();
    }
  }
  close(): void {
    this.closed = true;
    for (const cancel of this.jobs) cancel();
  }
}
