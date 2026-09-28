import { contextBridge, ipcRenderer } from 'electron';
contextBridge.exposeInMainWorld('pdfPreview', {
  request: (command: unknown): Promise<unknown> =>
    ipcRenderer.invoke('pdf-qualification:preview', command),
});
