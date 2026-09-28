import { build } from 'esbuild';
import { mkdir, cp, writeFile } from 'node:fs/promises';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { createRequire } from 'node:module';
const root = dirname(fileURLToPath(import.meta.url));
const require = createRequire(import.meta.url);
/** Fixed offline assets get their own protocol/CSP; the product shell policy is unchanged. */
export async function buildPdfAssets(): Promise<void> {
  const output = resolve(root, 'out/pdf');
  const vendor = dirname(require.resolve('pdfjs-dist/package.json'));
  await mkdir(output, { recursive: true });
  for (const name of ['standard_fonts', 'cmaps'])
    await cp(resolve(vendor, name), resolve(output, 'vendor', name), { recursive: true });
  await cp(resolve(vendor, 'build/pdf.worker.mjs'), resolve(output, 'vendor/pdf.worker.mjs'));
  await cp(resolve(vendor, 'LICENSE'), resolve(output, 'LICENSE-PDFJS'));
  for (const entry of ['decoder', 'worker', 'preload'])
    await build({
      entryPoints: [resolve(root, 'src/pdf', entry + '.ts')],
      outfile: resolve(output, entry + (entry === 'preload' ? '.cjs' : '.js')),
      bundle: true,
      platform: entry === 'preload' ? 'node' : 'browser',
      format: entry === 'preload' ? 'cjs' : 'esm',
      external: entry === 'preload' ? ['electron'] : [],
      target: entry === 'preload' ? 'node24' : 'chrome152',
      logLevel: 'silent',
    });
  await writeFile(
    resolve(output, 'index.html'),
    '<!doctype html><html lang="en"><head><meta charset="utf-8"><title>PDF decode host</title></head><body><script type="module" src="/decoder.js"></script></body></html>',
  );
}
