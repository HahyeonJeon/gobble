import { Menu, type BrowserWindow, type MenuItemConstructorOptions } from 'electron';
import { WORKSPACE_CHANNELS, type WorkspaceShortcut } from '@gobble/contracts';

export function installWorkspaceMenu(getWindow: () => BrowserWindow | undefined): void {
  const send = (action: WorkspaceShortcut) => {
    const window = getWindow();
    if (window && !window.isDestroyed())
      window.webContents.send(WORKSPACE_CHANNELS.shortcut, action);
  };
  const template: MenuItemConstructorOptions[] = [
    ...(process.platform === 'darwin' ? [{ role: 'appMenu' as const }] : []),
    {
      label: 'File',
      submenu: [
        { label: 'Open Folder…', accelerator: 'CmdOrCtrl+O', click: () => send('open-folder') },
        { label: 'Close View', accelerator: 'CmdOrCtrl+W', click: () => send('close-view') },
        { label: 'Close Window', role: 'close', accelerator: 'CmdOrCtrl+Shift+W' },
        ...(process.platform === 'darwin' ? [] : [{ role: 'quit' as const }]),
      ],
    },
    { role: 'editMenu' },
    {
      label: 'View',
      submenu: [
        { label: 'Split Workspace', accelerator: 'CmdOrCtrl+\\', click: () => send('split') },
        { label: 'Pin or Unpin View', accelerator: 'CmdOrCtrl+Shift+P', click: () => send('pin') },
        { type: 'separator' },
        { role: 'resetZoom' },
        { role: 'zoomIn' },
        { role: 'zoomOut' },
        { type: 'separator' },
        { role: 'togglefullscreen' },
      ],
    },
    { role: 'windowMenu' },
  ];
  Menu.setApplicationMenu(Menu.buildFromTemplate(template));
}
