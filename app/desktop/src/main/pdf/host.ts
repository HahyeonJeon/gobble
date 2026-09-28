import { BrowserWindow, ipcMain, session, type IpcMainEvent } from 'electron';
import { readFile } from 'node:fs/promises';
import { join } from 'node:path';
import { createHash, randomUUID } from 'node:crypto';
import {
  HOST_URL,
  JOB_CHANNEL,
  REPLY_CHANNEL,
  LIMITS,
  parseReply,
  type Job,
  type Result,
  type PageResult,
  type CaptureResult,
  type Rect,
} from '@gobble/contracts/pdf-decoder';
import { assertRegion, pixelRegion } from '@gobble/contracts/pdf-geometry';
import { assertPngHeader } from './raster';

const hashBytes = (bytes: Uint8Array) =>
  'sha256:' + createHash('sha256').update(bytes).digest('hex');

export const PDF_CSP = [
  "default-src 'none'",
  "script-src 'self'",
  "worker-src 'self'",
  "style-src 'none'",
  "img-src 'none'",
  "font-src 'none'",
  "connect-src 'self'",
  "object-src 'none'",
  "frame-src 'none'",
  "base-uri 'none'",
  "form-action 'none'",
].join('; ');
type Pending = {
  job: Job;
  resolve: (value: Result) => void;
  reject: (error: Error) => void;
  timer: ReturnType<typeof setTimeout>;
  cleanup: () => void;
};
export class PdfHost {
  private window: BrowserWindow | undefined;
  private pending: Pending | undefined;
  private revision = '';
  private current: PageResult | undefined;
  private readonly partition = 'gobble-pdf-' + randomUUID();
  private readonly requests: string[] = [];
  private readonly denied: string[] = [];
  private disposed = false;
  private protocolReady = false;
  private readonly replyListener = (event: IpcMainEvent, input: unknown) =>
    this.receive(event, input);
  constructor(
    private readonly assets: string,
    private readonly preload: string,
  ) {}
  async start(): Promise<void> {
    if (this.window || this.disposed) throw new Error('PDF host is already started or closed.');
    const scope = session.fromPartition(this.partition, { cache: false });
    scope.setPermissionCheckHandler(() => false);
    scope.on('will-download', (event) => event.preventDefault());
    scope.setPermissionRequestHandler((_wc, _p, done) => done(false));
    scope.webRequest.onBeforeRequest((details, done) => {
      const valid = details.url.startsWith('pdf-host://decoder/');
      if (!valid) this.denied.push(new URL(details.url).protocol);
      done({ cancel: !valid });
    });
    await scope.protocol.handle('pdf-host', async (request) => {
      let pathname: string;
      try {
        const url = new URL(request.url);
        if (url.host !== 'decoder' || url.search || url.hash || url.username || url.password)
          return new Response(null, { status: 404 });
        pathname = url.pathname;
      } catch {
        return new Response(null, { status: 400 });
      }
      const allowed =
        pathname === '/index.html' ||
        pathname === '/decoder.js' ||
        pathname === '/worker.js' ||
        pathname === '/vendor/pdf.worker.mjs' ||
        /^\/vendor\/(standard_fonts|cmaps)\/[A-Za-z0-9_.-]+\.(pfb|ttf|bcmap)$/.test(pathname);
      if (!allowed) return new Response(null, { status: 404 });
      this.requests.push(pathname);
      try {
        const bytes = await readFile(join(this.assets, pathname.slice(1)));
        const mime = pathname.endsWith('.html')
          ? 'text/html'
          : pathname.endsWith('.js') || pathname.endsWith('.mjs')
            ? 'text/javascript'
            : 'application/octet-stream';
        return new Response(new Uint8Array(bytes), {
          headers: {
            'Content-Type': mime,
            'Content-Security-Policy': PDF_CSP,
            'X-Content-Type-Options': 'nosniff',
          },
        });
      } catch {
        return new Response(null, { status: 404 });
      }
    });
    this.protocolReady = true;
    if (this.disposed) {
      scope.protocol.unhandle('pdf-host');
      this.protocolReady = false;
      throw new Error('PDF host closed during startup.');
    }
    const window = new BrowserWindow({
      show: false,
      width: 800,
      height: 800,
      webPreferences: {
        preload: this.preload,
        session: scope,
        sandbox: true,
        contextIsolation: true,
        nodeIntegration: false,
        webSecurity: true,
        backgroundThrottling: false,
        allowRunningInsecureContent: false,
      },
    });
    this.window = window;
    ipcMain.on(REPLY_CHANNEL, this.replyListener);
    window.webContents.setWindowOpenHandler(() => ({ action: 'deny' }));
    window.webContents.on('will-navigate', (event) => event.preventDefault());
    window.webContents.on('will-frame-navigate', (event) => event.preventDefault());
    window.webContents.on('will-attach-webview', (event) => event.preventDefault());
    window.webContents.on('render-process-gone', () => this.stop(new Error('PDF decoder exited.')));
    await window.loadURL(HOST_URL);
  }
  async open(bytes: Uint8Array, signal?: AbortSignal): Promise<Result> {
    if (bytes.byteLength > LIMITS.fileBytes) throw new Error('PDF exceeds the 8 MiB reader limit.');
    if (this.revision) throw new Error('Create a new host for a new source.');
    this.revision = hashBytes(bytes);
    return this.request(
      { kind: 'open', bytes: bytes.slice(), revision: this.revision, jobId: randomUUID() },
      signal,
    );
  }
  async page(
    pageIndex: number,
    scale = 1,
    rotation: 0 | 90 | 180 | 270 = 0,
    dpr = 1,
    signal?: AbortSignal,
  ): Promise<PageResult> {
    this.current = undefined;
    const result = await this.request(
      { kind: 'page', pageIndex, scale, rotation, dpr, jobId: randomUUID() },
      signal,
    );
    if (result.kind !== 'page') throw new Error('Expected a PDF page.');
    this.current = result;
    return result;
  }
  async capture(renditionId: string, modelHash: string, region: Rect): Promise<CaptureResult> {
    const page = this.current;
    if (!page || page.renditionId !== renditionId || page.modelHash !== modelHash)
      throw new Error('The PDF rendition is no longer current.');
    assertRegion(region, page.model.viewBox);
    const result = await this.request({
      kind: 'capture',
      renditionId,
      modelHash,
      region,
      jobId: randomUUID(),
    });
    if (result.kind !== 'capture') throw new Error('Expected PDF evidence.');
    if (result.pageIndex !== page.model.pageIndex) throw new Error('PDF capture page mismatch.');
    const pixels = pixelRegion(region, page.viewport, page.raster.width, page.raster.height);
    if (
      JSON.stringify(pixels) !== JSON.stringify(result.pixelRect) ||
      result.raster.width !== pixels.width ||
      result.raster.height !== pixels.height
    )
      throw new Error('PDF crop geometry mismatch.');
    return result;
  }
  private request(job: Job, signal?: AbortSignal): Promise<Result> {
    if (signal?.aborted) return Promise.reject(new Error('PDF job cancelled.'));
    if (!this.window || this.window.isDestroyed())
      return Promise.reject(new Error('PDF host is closed.'));
    if (this.pending) return Promise.reject(new Error('PDF host already has a job.'));
    return new Promise((resolve, reject) => {
      const abort = () => this.stop(new Error('PDF job cancelled.'));
      const timer = setTimeout(() => this.stop(new Error('PDF job timed out.')), LIMITS.jobMs);
      this.pending = {
        job,
        resolve,
        reject,
        timer,
        cleanup: () => signal?.removeEventListener('abort', abort),
      };
      signal?.addEventListener('abort', abort, { once: true });
      this.window!.webContents.send(JOB_CHANNEL, job);
    });
  }
  private receive(event: IpcMainEvent, input: unknown) {
    if (
      event.sender !== this.window?.webContents ||
      event.senderFrame !== this.window.webContents.mainFrame ||
      event.senderFrame.url !== HOST_URL
    )
      return;
    const pending = this.pending;
    if (!pending) return;
    let result: Result;
    try {
      const reply = parseReply(input);
      if (reply.jobId !== pending.job.jobId) throw new Error('PDF job ID mismatch.');
      if (!reply.ok) throw new Error(reply.message);
      result = reply.value;
      if (result.kind !== 'opened')
        assertPngHeader(result.raster.png, result.raster.width, result.raster.height);
      const revision = result.kind === 'page' ? result.model.revision : result.revision;
      if (revision !== this.revision) throw new Error('PDF reply source mismatch.');
      if (result.kind === 'page') {
        if (
          pending.job.kind !== 'page' ||
          pending.job.pageIndex !== result.model.pageIndex ||
          pending.job.jobId !== result.renditionId
        )
          throw new Error('PDF reply page mismatch.');
        if (hashBytes(Buffer.from(JSON.stringify(result.model))) !== result.modelHash)
          throw new Error('PDF model hash mismatch.');
        assertRegion(result.model.viewBox, result.model.viewBox);
      } else if (result.kind === 'capture') {
        if (
          pending.job.kind !== 'capture' ||
          result.renditionId !== pending.job.renditionId ||
          result.modelHash !== pending.job.modelHash ||
          JSON.stringify(result.region) !== JSON.stringify(pending.job.region)
        )
          throw new Error('PDF reply target mismatch.');
      } else if (pending.job.kind !== 'open') throw new Error('PDF reply type mismatch.');
    } catch (error) {
      this.finish();
      pending.reject(error instanceof Error ? error : new Error('Invalid PDF reply.'));
      return;
    }
    this.finish();
    pending.resolve(result);
  }
  private finish() {
    if (this.pending) {
      clearTimeout(this.pending.timer);
      this.pending.cleanup();
      this.pending = undefined;
    }
  }
  stop(reason = new Error('PDF host closed.')) {
    if (this.disposed) return;
    this.disposed = true;
    const pending = this.pending;
    this.finish();
    this.current = undefined;
    ipcMain.removeListener(REPLY_CHANNEL, this.replyListener);
    const window = this.window;
    this.window = undefined;
    if (window && !window.isDestroyed()) window.destroy();
    if (this.protocolReady) {
      session.fromPartition(this.partition).protocol.unhandle('pdf-host');
      this.protocolReady = false;
    }
    pending?.reject(reason);
  }
  diagnostics() {
    return {
      requests: [...this.requests],
      denied: [...this.denied],
      pending: !!this.pending,
      closed: !this.window,
      webContentsId: this.window?.webContents.id,
      pid: this.window?.webContents.getOSProcessId(),
    };
  }
}
