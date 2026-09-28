import {
  app,
  BrowserWindow,
  ipcMain,
  protocol,
  session,
  nativeImage,
  type IpcMainInvokeEvent,
  type IpcMainEvent,
} from 'electron';
import { readFile } from 'node:fs/promises';
import { join } from 'node:path';
import { NotebookHost } from './host';
import { fixture, FIXTURES, png, type FixtureName } from './fixtures';
import { NotebookProblem, type HostResult } from './model';
import { activeWorkers } from './worker-client';
app.setName('Gobble Notebook qualification');
if (!process.env.R4A1_PROFILE) throw new Error('An isolated qualification profile is required.');
app.setPath('userData', process.env.R4A1_PROFILE);
protocol.registerSchemesAsPrivileged([
  { scheme: 'notebook', privileges: { standard: true, secure: true, supportFetchAPI: true } },
]);
const readers = new Map<number, { window: BrowserWindow; host: NotebookHost }>();
const blocked: string[] = [];
const csp =
  "default-src 'none'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src data:; connect-src 'none'; object-src 'none'; frame-src 'none'; base-uri 'none'; form-action 'none'";
function owner(event: IpcMainInvokeEvent | IpcMainEvent): NotebookHost {
  const reader = readers.get(event.sender.id);
  if (
    !reader ||
    !event.senderFrame ||
    event.senderFrame !== event.sender.mainFrame ||
    event.senderFrame.url !== 'notebook://review/index.html'
  )
    throw new NotebookProblem('invalid', 'Untrusted Notebook request sender.');
  return reader.host;
}
async function reply<T>(work: () => Promise<T> | T): Promise<HostResult<T>> {
  try {
    return { ok: true, value: await work() };
  } catch (e) {
    return {
      ok: false,
      code: e instanceof NotebookProblem ? e.code : 'invalid',
      message: e instanceof Error ? e.message : 'Notebook operation failed.',
    };
  }
}
const jpeg = () => nativeImage.createFromBuffer(png()).toJPEG(90).toString('base64');
function source(name: unknown): Buffer {
  if (typeof name !== 'string' || !FIXTURES.some((n) => n === name))
    throw new NotebookProblem('invalid', 'Unknown qualification fixture.');
  return fixture(name as FixtureName, name === 'jpeg' ? jpeg() : undefined); // Membership checked above.
}
async function createReader(): Promise<BrowserWindow> {
  if (readers.size >= 2) throw new NotebookProblem('busy', 'At most two readers are allowed.');
  const window = new BrowserWindow({
    title: 'Gobble · Notebook qualification',
    width: 1380,
    height: 920,
    minWidth: 900,
    minHeight: 650,
    show: true,
    webPreferences: {
      preload: join(__dirname, 'preload.cjs'),
      partition: 'r4a1',
      sandbox: true,
      contextIsolation: true,
      nodeIntegration: false,
      webSecurity: true,
      allowRunningInsecureContent: false,
      webviewTag: false,
    },
  });
  const host = new NotebookHost();
  readers.set(window.webContents.id, { window, host });
  window.webContents.setWindowOpenHandler(() => ({ action: 'deny' }));
  window.webContents.on('will-navigate', (e) => e.preventDefault());
  window.webContents.on('will-frame-navigate', (e) => e.preventDefault());
  window.webContents.on('will-redirect', (e) => e.preventDefault());
  window.webContents.on('will-attach-webview', (e) => e.preventDefault());
  const id = window.webContents.id;
  window.on('closed', () => {
    host.close();
    readers.delete(id);
  });
  await window.loadURL('notebook://review/index.html');
  return window;
}
app
  .whenReady()
  .then(async () => {
    const s = session.fromPartition('r4a1');
    s.setPermissionRequestHandler((_w, _p, callback) => callback(false));
    s.setPermissionCheckHandler(() => false);
    s.webRequest.onBeforeRequest((details, callback) => {
      const allowed =
        details.url.startsWith('notebook://review/') ||
        details.url.startsWith('data:image/png;base64,');
      if (!allowed) blocked.push(details.url.slice(0, 200));
      callback({ cancel: !allowed });
    });
    s.protocol.handle('notebook', async (request) => {
      const url = new URL(request.url);
      const assets: Record<string, string> = {
        '/index.html': 'index.html',
        '/renderer.js': 'renderer.js',
        '/renderer.css': 'renderer.css',
      };
      const asset = assets[url.pathname];
      if (url.hostname !== 'review' || !asset || url.search)
        return new Response('Not found', { status: 404 });
      return new Response(new Uint8Array(await readFile(join(__dirname, asset))), {
        headers: {
          'Content-Type': asset.endsWith('.html')
            ? 'text/html'
            : asset.endsWith('.css')
              ? 'text/css'
              : 'text/javascript',
          'Content-Security-Policy': csp,
        },
      });
    });
    ipcMain.handle('notebook:open', (event, name: unknown) =>
      reply(() => owner(event).load(source(name))),
    );
    ipcMain.handle('notebook:visible', (event, value: unknown) =>
      reply(() => owner(event).visible(value)),
    );
    ipcMain.handle('notebook:image', (event, value: unknown) =>
      reply(() => owner(event).image(value)),
    );
    ipcMain.handle('notebook:capture', (event, value: unknown) =>
      reply(() => owner(event).capture(value)),
    );
    ipcMain.on('notebook:invalidate', (event) => {
      try {
        owner(event).invalidate();
      } catch {
        /* An invalid sender receives no authority. */
      }
    });
    await createReader();
    // Main-only test observer; deliberately unavailable through either renderer or preload.
    Object.assign(globalThis, {
      notebookQualification: { readers, source, createReader, blocked, activeWorkers, owner },
    });
  })
  .catch((error) => {
    console.error(error);
    app.exit(1);
  });
app.on('window-all-closed', () => app.quit());
