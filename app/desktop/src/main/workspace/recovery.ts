import { dialog, type BrowserWindow } from 'electron';

/** A lost renderer invalidates all reading leases before an explicit native reload. */
export function installRendererRecovery(
  window: BrowserWindow,
  target: string,
  disconnect: () => void,
  isQuitting: () => boolean,
): void {
  let recovering = false;
  window.webContents.on('render-process-gone', (_event, details) => {
    if (window.isDestroyed() || isQuitting() || details.reason === 'clean-exit') return;
    disconnect();
    if (recovering) return;
    recovering = true;
    dialog
      .showMessageBox(window, {
        type: 'error',
        title: 'Workspace display stopped',
        message: 'The workspace display stopped unexpectedly.',
        detail: 'Reload to restore your saved Project, draft and captured attachments.',
        buttons: ['Reload workspace', 'Keep window open'],
        defaultId: 0,
        cancelId: 1,
      })
      .then(async ({ response }) => {
        if (response === 0 && !window.isDestroyed() && !isQuitting()) await window.loadURL(target);
      })
      .catch(() => {
        if (!window.isDestroyed() && !isQuitting())
          dialog.showErrorBox(
            'Workspace could not reload',
            'Close this window and reopen Gobble to restore your saved workspace.',
          );
      })
      .finally(() => {
        recovering = false;
      });
  });
}
