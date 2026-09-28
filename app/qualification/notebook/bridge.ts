import type { Capture, HostResult, Target, ViewReceipt, ViewSnapshot } from './model';
export interface NotebookBridge {
  open(name: string): Promise<HostResult<ViewSnapshot>>;
  visible(receipt: ViewReceipt): Promise<HostResult<number>>;
  invalidate(): void;
  image(target: Target): Promise<HostResult<{ base64: string; width: number; height: number }>>;
  capture(target: Target): Promise<HostResult<Capture>>;
}
declare global {
  interface Window {
    notebook: NotebookBridge;
  }
}
