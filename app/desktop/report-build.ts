import { build } from 'esbuild';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
export async function buildReportReader(): Promise<void> {
  const root = dirname(fileURLToPath(import.meta.url));
  await build({
    entryPoints: [resolve(root, 'src/report-reader/reader.ts')],
    outfile: resolve(root, 'out/report-reader/reader.js'),
    bundle: true,
    platform: 'browser',
    format: 'iife',
    globalName: 'FastqcDecoder',
    target: 'chrome144',
    logLevel: 'silent',
  });
}
