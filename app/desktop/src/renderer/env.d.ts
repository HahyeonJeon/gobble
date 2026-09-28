import type { DesktopBridge } from '@gobble/contracts';

declare global {
  interface Window {
    readonly gobble: DesktopBridge;
  }
}
