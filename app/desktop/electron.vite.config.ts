import { buildReportReader } from './report-build';
import { buildNotebookWorker } from './notebook-build';
import { buildPdfAssets } from './pdf-build';
import { defineConfig } from 'electron-vite';
import react from '@vitejs/plugin-react';
import { DEVELOPMENT_URL } from './src/main/security/origin';

export default defineConfig({
  main: {
    plugins: [
      { name: 'report-reader', buildStart: buildReportReader },
      { name: 'offline-pdf-assets', buildStart: buildPdfAssets },
      { name: 'notebook-worker', buildStart: buildNotebookWorker },
    ],
    build: { externalizeDeps: false },
  },
  preload: {
    build: {
      externalizeDeps: false,
      rollupOptions: { output: { format: 'cjs', entryFileNames: 'index.cjs' } },
    },
  },
  renderer: {
    plugins: [react()],
    // Lazy dependency names may contain dots; emitted names follow the shell's existing allowlist.
    build: { rollupOptions: { output: { chunkFileNames: 'assets/chunk-[hash].js' } } },
    server: {
      host: '127.0.0.1',
      port: 5173,
      strictPort: true,
      headers: {
        'Content-Security-Policy': [
          "default-src 'none'",
          "script-src 'self' 'unsafe-inline'",
          "style-src 'self' 'unsafe-inline'",
          `connect-src 'self' ${DEVELOPMENT_URL.replace('http:', 'ws:')}`,
          "img-src 'self' blob:",
          "font-src 'self'",
          "frame-src 'none'",
          "frame-ancestors 'none'",
          "object-src 'none'",
          "base-uri 'none'",
          "form-action 'none'",
        ].join('; '),
      },
    },
  },
});
