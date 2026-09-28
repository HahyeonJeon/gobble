import { contextBridge, ipcRenderer } from 'electron';
import { JOB_CHANNEL, REPLY_CHANNEL, type Job, type Reply } from '@gobble/contracts/pdf-decoder';

// This host never receives the general Workspace, filesystem or account bridge.
contextBridge.exposeInMainWorld('pdfJob', {
  listen(callback: (job: Job) => void) {
    const listener = (_event: Electron.IpcRendererEvent, job: Job) => callback(job);
    ipcRenderer.on(JOB_CHANNEL, listener);
    window.addEventListener('pagehide', () => ipcRenderer.removeListener(JOB_CHANNEL, listener), {
      once: true,
    });
  },
  reply(value: Reply) {
    ipcRenderer.send(REPLY_CHANNEL, value);
  },
});
