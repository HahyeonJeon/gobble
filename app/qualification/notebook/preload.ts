import { contextBridge, ipcRenderer } from 'electron';
import type { NotebookBridge } from './bridge';
// Main is the sole reply producer. Every request is validated in its named handler.
const bridge: NotebookBridge = {
  open: (name) => ipcRenderer.invoke('notebook:open', name),
  visible: (receipt) => ipcRenderer.invoke('notebook:visible', receipt),
  invalidate: () => ipcRenderer.send('notebook:invalidate'),
  image: (target) => ipcRenderer.invoke('notebook:image', target),
  capture: (target) => ipcRenderer.invoke('notebook:capture', target),
};
contextBridge.exposeInMainWorld('notebook', bridge);
