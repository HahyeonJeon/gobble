import { build } from 'esbuild';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
/** Node parser worker is a separate emitted entry, never part of the renderer bundle. */
export async function buildNotebookWorker(): Promise<void> {
  const root = dirname(fileURLToPath(import.meta.url));
  await build({
    entryPoints: [resolve(root, 'src/main/notebook/worker.ts')],
    outfile: resolve(root, 'out/notebook/worker.cjs'),
    bundle: true,
    platform: 'node',
    format: 'cjs',
    target: 'node24',
    logLevel: 'silent',
  });
}
