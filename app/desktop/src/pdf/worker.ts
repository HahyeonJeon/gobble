// Capture PDF.js worker warnings before importing the unmodified vendor worker.
// These messages only reduce capability; they cannot grant a read or publish evidence.
const port = self as unknown as { postMessage(value: unknown): void };
const originalWarn = console.warn.bind(console);
console.warn = (...args: unknown[]) => {
  const message = args.map(String).join(' ');
  if (message.startsWith('Warning:'))
    port.postMessage({ type: 'gobble-pdf-warning', message: message.slice(0, 300) });
  else originalWarn(...args);
};
const workerModule = 'pdf-host://decoder/vendor/pdf.worker.mjs';
await import(workerModule);
port.postMessage({ type: 'gobble-pdf-ready' });
export {};
