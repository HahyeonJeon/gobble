// Qualification-only host. No production preload, IPC or connection state is reused.
const { app, BrowserWindow, WebContentsView, protocol, session } = require('electron');
const fs = require('node:fs/promises');
const path = require('node:path');
const { X509Certificate } = require('node:crypto');
const cfg = require(process.env.R4B_CONNECTION);
app.setName('Gobble Jupyter qualification');
app.setPath('userData', path.join(cfg.scratch, 'electron-profile'));
protocol.registerSchemesAsPrivileged([
  { scheme: 'r4b', privileges: { standard: true, secure: true, supportFetchAPI: true } },
]);
const preferences = (partition) => ({
  partition,
  sandbox: true,
  contextIsolation: true,
  nodeIntegration: false,
  webSecurity: true,
  allowRunningInsecureContent: false,
  webviewTag: false,
});
function guard(contents, origin) {
  contents.setWindowOpenHandler(() => ({ action: 'deny' }));
  contents.on('will-attach-webview', (e) => e.preventDefault());
  for (const event of ['will-navigate', 'will-frame-navigate', 'will-redirect'])
    contents.on(event, (e, url) => {
      const target = typeof url === 'string' ? url : e.url;
      if (!target || new URL(target).origin !== origin) e.preventDefault();
    });
}
function denyPermissions(s) {
  s.setPermissionRequestHandler((_w, _p, reply) => reply(false));
  s.setPermissionCheckHandler(() => false);
  s.setDevicePermissionHandler(() => false);
}
app
  .whenReady()
  .then(async () => {
    const local = session.fromPartition('r4b-local');
    denyPermissions(local);
    local.webRequest.onBeforeRequest((d, cb) =>
      cb({ cancel: !d.url.startsWith('r4b://review/') && !d.url.startsWith('data:') }),
    );
    local.protocol.handle('r4b', async (request) => {
      const url = new URL(request.url);
      const allowed = ['index.html', 'connected.html', 'component.js', 'component.css'];
      const file = url.pathname.slice(1);
      if (url.hostname !== 'review' || !allowed.includes(file))
        return new Response('Missing', { status: 404 });
      return new Response(await fs.readFile(path.join(__dirname, file)), {
        headers: {
          'Content-Type': file.endsWith('.html')
            ? 'text/html'
            : file.endsWith('.css')
              ? 'text/css'
              : 'text/javascript',
          'Content-Security-Policy':
            "default-src 'none'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src data:; font-src data:; connect-src 'none'; object-src 'none'; frame-src 'none'; base-uri 'none'; form-action 'none'",
        },
      });
    });
    const a = new BrowserWindow({
      width: 1280,
      height: 850,
      show: true,
      webPreferences: preferences('r4b-local'),
    });
    guard(a.webContents, 'r4b://review');
    await a.loadURL('r4b://review/index.html');
    let b, view;
    const connected = session.fromPartition('r4b-connected');
    denyPermissions(connected);
    const expected = new X509Certificate(await fs.readFile(cfg.cert)).fingerprint256;
    connected.setCertificateVerifyProc((request, callback) => {
      let valid = false;
      try {
        valid =
          request.hostname === '127.0.0.1' &&
          new X509Certificate(request.certificate.data).fingerprint256 === expected;
      } catch {}
      callback(valid ? 0 : -2);
    });
    const allowed = (url) =>
      url.startsWith(cfg.origin + '/') ||
      url.startsWith(cfg.origin.replace('https:', 'wss:') + '/');
    connected.webRequest.onBeforeRequest((d, cb) =>
      cb({
        cancel:
          !allowed(d.url) &&
          !d.url.startsWith('data:') &&
          !d.url.startsWith('blob:' + cfg.origin + '/'),
      }),
    );
    connected.webRequest.onBeforeSendHeaders((d, cb) => {
      if (allowed(d.url)) d.requestHeaders.Authorization = 'token ' + cfg.token;
      cb({ requestHeaders: d.requestHeaders });
    });
    global.r4b = {
      async openConnected() {
        b = new BrowserWindow({
          width: 1280,
          height: 850,
          show: true,
          webPreferences: preferences('r4b-local'),
        });
        guard(b.webContents, 'r4b://review');
        await b.loadURL('r4b://review/connected.html');
        view = new WebContentsView({ webPreferences: preferences('r4b-connected') });
        guard(view.webContents, cfg.origin);
        b.contentView.addChildView(view);
        const resize = () => {
          const [width, height] = b.getContentSize();
          view.setBounds({ x: 22, y: 138, width: width - 365, height: height - 160 });
        };
        b.on('resize', resize);
        resize();
        b.on('closed', () => {
          if (!view.webContents.isDestroyed()) view.webContents.close();
        });
        await view.webContents.loadURL(cfg.origin + '/lab/tree/study.ipynb');
        return view.webContents.id;
      },
      // BrowserWindow capture omits child WebContentsViews; capture the actual surface.
      async screenshotConnected(file) {
        await fs.writeFile(file, (await view.webContents.capturePage()).toPNG());
      },
      async externalEdit() {
        const file = path.join(cfg.project, 'study.ipynb');
        const notebook = JSON.parse(await fs.readFile(file, 'utf8'));
        notebook.cells[0].source = 'external_change = 2';
        await fs.writeFile(file, JSON.stringify(notebook));
      },
      async closeConnected() {
        const contents = view.webContents;
        const destroyed = new Promise((resolve) => contents.once('destroyed', resolve));
        b.destroy();
        await destroyed;
        return contents.isDestroyed();
      },
    };
  })
  .catch((error) => {
    console.error(error);
    app.exit(1);
  });
app.on('window-all-closed', () => app.quit());
