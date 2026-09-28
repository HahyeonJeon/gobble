import { app, BrowserWindow, ipcMain, protocol, session } from 'electron';
import { readFile } from 'node:fs/promises';
import { join } from 'node:path';
import { Type } from '@sinclair/typebox';
import { Value } from '@sinclair/typebox/value';
import { PdfHost } from './host';
import { FIXTURE_NAMES, fixture, type FixtureName } from './fixtures';
import { parseRegion, type PageResult } from './protocol';
import { readShellAsset } from '../../desktop/src/main/security/content';

protocol.registerSchemesAsPrivileged([
  {
    scheme: 'app',
    privileges: { standard: true, secure: true, supportFetchAPI: true, corsEnabled: true },
  },
  {
    scheme: 'pdf-host',
    privileges: { standard: true, secure: true, supportFetchAPI: true, corsEnabled: true },
  },
]);
const closed = { additionalProperties: false };
const CommandSchema = Type.Union([
  Type.Object(
    {
      kind: Type.Literal('open'),
      fixture: Type.Union(FIXTURE_NAMES.map((name) => Type.Literal(name))),
    },
    closed,
  ),
  Type.Object(
    {
      kind: Type.Literal('page'),
      pageIndex: Type.Integer({ minimum: 0, maximum: 199 }),
      scale: Type.Number({ minimum: 0.1, maximum: 4 }),
      rotation: Type.Union([
        Type.Literal(0),
        Type.Literal(90),
        Type.Literal(180),
        Type.Literal(270),
      ]),
      dpr: Type.Union([Type.Literal(1), Type.Literal(2)]),
    },
    closed,
  ),
  Type.Object(
    {
      kind: Type.Literal('capture'),
      renditionId: Type.String({ maxLength: 100 }),
      modelHash: Type.String({ maxLength: 100 }),
      region: Type.Array(Type.Number(), { minItems: 4, maxItems: 4 }),
    },
    closed,
  ),
  Type.Object({ kind: Type.Literal('cancel') }, closed),
]);
let host: PdfHost | undefined, activePage: PageResult | undefined;
const retired: ReturnType<PdfHost['diagnostics']>[] = [];
const assets = process.env.R3C1_ASSETS!;
const preload = process.env.R3C1_PRELOAD!;
const previewUrl = 'app://gobble/index.html';
async function replace(name: FixtureName, signal?: AbortSignal) {
  stop();
  const current = new PdfHost(join(assets, 'pdf'), preload);
  host = current;
  try {
    const bytes =
      name === 'encrypted' ? await readFile(join(assets, 'encrypted.pdf')) : fixture(name);
    await current.start();
    const opened = await current.open(bytes, signal);
    if (host !== current) throw new Error('PDF job superseded.');
    return opened;
  } catch (error) {
    if (host === current) stop();
    throw error;
  }
}
function stop() {
  activePage = undefined;
  const current = host;
  host = undefined;
  if (current) {
    current.stop();
    retired.push(current.diagnostics());
  }
}
async function page(
  pageIndex: number,
  scale = 1,
  rotation: 0 | 90 | 180 | 270 = 0,
  dpr = 1,
  signal?: AbortSignal,
) {
  const current = host;
  if (!current) throw new Error('Open a PDF first.');
  activePage = undefined;
  const result = await current.page(pageIndex, scale, rotation, dpr, signal);
  if (host !== current) throw new Error('PDF job superseded.');
  activePage = result;
  return result;
}
async function main() {
  app.setPath('userData', process.env.R3C1_PROFILE!);
  await app.whenReady();
  protocol.handle('app', (request) => readShellAsset(join(assets, 'ui'), request.url));
  session.defaultSession.setPermissionCheckHandler(() => false);
  session.defaultSession.setPermissionRequestHandler((_wc, _p, done) => done(false));
  const window = new BrowserWindow({
    title: 'Gobble · PDF qualification',
    width: 1280,
    height: 840,
    show: process.argv.includes('--show'),
    webPreferences: {
      preload: process.env.R3C1_UI_PRELOAD!,
      sandbox: true,
      contextIsolation: true,
      nodeIntegration: false,
      webSecurity: true,
      allowRunningInsecureContent: false,
    },
  });
  window.webContents.setWindowOpenHandler(() => ({ action: 'deny' }));
  window.webContents.on('will-navigate', (event) => event.preventDefault());
  window.webContents.on('will-attach-webview', (event) => event.preventDefault());
  ipcMain.handle('pdf-qualification:preview', async (event, input: unknown) => {
    if (
      event.sender !== window.webContents ||
      event.senderFrame !== window.webContents.mainFrame ||
      event.senderFrame.url !== previewUrl
    )
      throw new Error('Unknown PDF preview sender.');
    if (!Value.Check(CommandSchema, input)) throw new Error('Invalid PDF preview command.');
    if (input.kind === 'cancel') {
      stop();
      return null;
    }
    if (input.kind === 'open') return replace(input.fixture);
    if (input.kind === 'page') return page(input.pageIndex, input.scale, input.rotation, input.dpr);
    if (!activePage || !host) throw new Error('No current page.');
    return host.capture(input.renditionId, input.modelHash, parseRegion(input.region));
  });
  // Only the native test harness can access this Main-world observer.
  Object.assign(globalThis, {
    pdfQualification: {
      replace,
      page,
      stop,
      capture: async (renditionId: string, modelHash: string, region: unknown) => {
        if (!host) throw new Error('No PDF host.');
        return host.capture(renditionId, modelHash, parseRegion(region));
      },
      current: () => activePage,
      host: () => host,
      status: () => ({ current: host?.diagnostics(), retired }),
      preview: () => window,
    },
  });
  await window.loadURL(previewUrl);
  window.on('closed', () => {
    stop();
    app.quit();
  });
  app.on('before-quit', stop);
}
void main().catch((error) => {
  process.stderr.write(String(error) + '\n');
  app.exit(1);
});
