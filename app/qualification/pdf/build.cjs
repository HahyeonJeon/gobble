const path = require('node:path');
const fs = require('node:fs/promises');
const { tmpdir } = require('node:os');
const { execFileSync } = require('node:child_process');
const { createHash } = require('node:crypto');
const { build } = require('esbuild');
const appRoot = path.resolve(__dirname, '../..');
const evidence = path.resolve(appRoot, '../docs/desktop-workspace/stages/r3c1-review');
async function prepare() {
  const scratch = await fs.mkdtemp(path.join(tmpdir(), 'gobble-r3c1-'));
  try {
    const candidate = process.env.R3C1_CANDIDATE || path.join(scratch, 'candidate');
    if (!process.env.R3C1_CANDIDATE) {
      await fs.mkdir(candidate);
      await fs.writeFile(
        path.join(candidate, 'package.json'),
        JSON.stringify({ private: true, dependencies: { 'pdfjs-dist': '6.3.289' } }),
      );
      execFileSync(
        'npm',
        ['install', '--ignore-scripts', '--no-audit', '--no-fund', '--omit=optional'],
        { cwd: candidate, stdio: 'pipe' },
      );
    }
    const pdfjs = path.join(candidate, 'node_modules/pdfjs-dist');
    const info = JSON.parse(await fs.readFile(path.join(pdfjs, 'package.json'), 'utf8'));
    if (info.version !== '6.3.289') throw new Error('The qualification requires PDF.js 6.3.289.');
    await fs.mkdir(evidence, { recursive: true });
    await fs.copyFile(
      path.join(candidate, 'package-lock.json'),
      path.join(evidence, 'candidate-package-lock.json'),
    );
    const assets = path.join(scratch, 'assets');
    await fs.mkdir(path.join(assets, 'pdf/vendor'), { recursive: true });
    await fs.mkdir(path.join(assets, 'ui/assets'), { recursive: true });
    await fs.cp(
      path.join(pdfjs, 'standard_fonts'),
      path.join(assets, 'pdf/vendor/standard_fonts'),
      {
        recursive: true,
      },
    );
    await fs.cp(path.join(pdfjs, 'cmaps'), path.join(assets, 'pdf/vendor/cmaps'), {
      recursive: true,
    });
    await fs.copyFile(
      path.join(pdfjs, 'build/pdf.worker.mjs'),
      path.join(assets, 'pdf/vendor/pdf.worker.mjs'),
    );
    await fs.copyFile(
      path.join(__dirname, 'fixtures/encrypted.pdf'),
      path.join(assets, 'encrypted.pdf'),
    );
    for (const [entry, outfile, platform, format] of [
      ['main.ts', 'main.cjs', 'node', 'cjs'],
      ['preload.ts', 'preload.cjs', 'node', 'cjs'],
      ['ui-preload.ts', 'ui-preload.cjs', 'node', 'cjs'],
      ['decoder.ts', 'assets/pdf/decoder.js', 'browser', 'esm'],
      ['renderer.tsx', 'assets/ui/assets/renderer.js', 'browser', 'esm'],
      ['worker.ts', 'assets/pdf/worker.js', 'browser', 'esm'],
      ['geometry.ts', 'geometry.cjs', 'node', 'cjs'],
      ['protocol.ts', 'protocol.cjs', 'node', 'cjs'],
      ['fixtures.ts', 'fixtures.cjs', 'node', 'cjs'],
      ['raster.ts', 'raster.cjs', 'node', 'cjs'],
    ]) {
      await build({
        entryPoints: [path.join(__dirname, entry)],
        outfile: path.join(scratch, outfile),
        bundle: true,
        format,
        platform,
        external: platform === 'node' ? ['electron'] : [],
        jsx: 'automatic',
        target: platform === 'browser' ? 'chrome152' : 'node24',
        alias: { 'pdfjs-dist': path.join(pdfjs, 'build/pdf.mjs') },
        logLevel: 'silent',
      });
    }
    await fs.writeFile(
      path.join(assets, 'pdf/index.html'),
      '<!doctype html><html lang="en"><head><meta charset="utf-8"><title>PDF decode host</title></head><body><script type="module" src="/decoder.js"></script></body></html>',
    );
    await fs.writeFile(
      path.join(assets, 'ui/index.html'),
      '<!doctype html><html lang="en"><head><meta charset="utf-8"><title>Gobble · PDF qualification</title><meta name="viewport" content="width=device-width,initial-scale=1"><link rel="stylesheet" href="/assets/renderer.css"></head><body><div id="root"></div><script type="module" src="/assets/renderer.js"></script></body></html>',
    );
    const config = {
      extends: path.join(appRoot, 'tsconfig.base.json'),
      compilerOptions: {
        lib: ['ES2022', 'DOM'],
        types: ['node'],
        typeRoots: [path.join(appRoot, 'node_modules/@types')],
        jsx: 'react-jsx',
        paths: { 'pdfjs-dist': [path.join(pdfjs, 'types/src/pdf.d.ts')] },
      },
      include: [path.join(__dirname, '*.ts'), path.join(__dirname, '*.tsx')],
    };
    await fs.writeFile(path.join(scratch, 'tsconfig.json'), JSON.stringify(config));
    try {
      execFileSync(
        path.join(appRoot, 'node_modules/.bin/tsc'),
        ['-p', path.join(scratch, 'tsconfig.json')],
        { cwd: appRoot, stdio: 'pipe' },
      );
    } catch (error) {
      await fs.writeFile(
        path.join(evidence, 'construction-failure.log'),
        error.stdout || String(error),
      );
      throw error;
    }
    const manifest = {
      version: info.version,
      license: info.license,
      optionalNativeCanvasInstalled: await fs
        .stat(path.join(candidate, 'node_modules/@napi-rs/canvas'))
        .then(
          () => true,
          () => false,
        ),
      assets: [],
    };
    async function inventory(directory, prefix = '') {
      for (const entry of await fs.readdir(directory, { withFileTypes: true })) {
        const rel = prefix + entry.name;
        if (entry.isDirectory()) await inventory(path.join(directory, entry.name), rel + '/');
        else {
          const bytes = await fs.readFile(path.join(directory, entry.name));
          manifest.assets.push({
            path: rel,
            bytes: bytes.length,
            sha256: createHash('sha256').update(bytes).digest('hex'),
          });
        }
      }
    }
    await inventory(path.join(assets, 'pdf'));
    await fs.mkdir(path.join(evidence, 'licenses'), { recursive: true });
    for (const name of [
      'LICENSE',
      'standard_fonts/LICENSE_FOXIT',
      'standard_fonts/LICENSE_LIBERATION',
      'cmaps/LICENSE',
    ])
      await fs.copyFile(
        path.join(pdfjs, name),
        path.join(evidence, 'licenses', name.replaceAll('/', '-')),
      );
    await fs.writeFile(
      path.join(evidence, 'assets.json'),
      JSON.stringify(manifest, null, 2) + '\n',
    );
    const env = Object.fromEntries(
      Object.entries(process.env).filter(
        ([key]) => !['ELECTRON_RUN_AS_NODE', 'ELECTRON_RENDERER_URL'].includes(key),
      ),
    );
    Object.assign(env, {
      R3C1_ASSETS: assets,
      R3C1_PRELOAD: path.join(scratch, 'preload.cjs'),
      R3C1_UI_PRELOAD: path.join(scratch, 'ui-preload.cjs'),
      R3C1_PROFILE: path.join(scratch, 'profile'),
    });
    return { scratch, assets, env, main: path.join(scratch, 'main.cjs'), evidence };
  } catch (error) {
    await fs.rm(scratch, { recursive: true, force: true });
    throw error;
  }
}
module.exports = { prepare, appRoot, evidence };
if (require.main === module)
  prepare()
    .then(({ scratch, assets, main }) => {
      console.log(JSON.stringify({ scratch, assets, main }));
    })
    .catch((error) => {
      console.error(error.stdout?.toString() || error.message);
      process.exitCode = 1;
    });
