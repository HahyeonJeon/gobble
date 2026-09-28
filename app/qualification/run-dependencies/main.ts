import { app, BrowserWindow, protocol, session } from 'electron';
import { readShellAsset } from '../../desktop/src/main/security/content';

protocol.registerSchemesAsPrivileged([
  {
    scheme: 'app',
    privileges: { standard: true, secure: true, supportFetchAPI: true, corsEnabled: true },
  },
]);
async function main() {
  app.setPath('userData', process.env.R3B_QUALIFICATION_PROFILE!);
  await app.whenReady();
  protocol.handle('app', (request) =>
    readShellAsset(process.env.R3B_QUALIFICATION_ASSETS!, request.url),
  );
  session.defaultSession.setPermissionCheckHandler(() => false);
  session.defaultSession.setPermissionRequestHandler((_webContents, _permission, done) =>
    done(false),
  );
  const window = new BrowserWindow({
    width: 1280,
    height: 840,
    webPreferences: {
      nodeIntegration: false,
      contextIsolation: true,
      sandbox: true,
      webSecurity: true,
      allowRunningInsecureContent: false,
    },
  });
  window.webContents.setWindowOpenHandler(() => ({ action: 'deny' }));
  window.webContents.on('will-navigate', (event, url) => {
    if (url !== 'app://gobble/index.html') event.preventDefault();
  });
  await window.loadURL('app://gobble/index.html');
  app.on('window-all-closed', () => app.quit());
}
void main().catch((error) => {
  process.stderr.write(String(error));
  app.exit(1);
});
