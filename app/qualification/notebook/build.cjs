const fs = require('node:fs/promises');
const path = require('node:path');
const os = require('node:os');
const { execFileSync } = require('node:child_process');
const { createHash } = require('node:crypto');
const { build } = require('esbuild');
const appRoot = path.resolve(__dirname, '../..');
const evidence = path.resolve(appRoot, '../docs/desktop-workspace/stages/r4a1-review');
async function prepare() {
  await fs.mkdir(evidence, { recursive: true });
  const scratch = await fs.mkdtemp(path.join(os.tmpdir(), 'gobble-r4a1-'));
  const cleanup = () => fs.rm(scratch, { recursive: true, force: true });
  try {
    execFileSync(
      path.join(appRoot, 'node_modules/.bin/tsc'),
      ['-p', path.join(__dirname, 'tsconfig.json')],
      { cwd: appRoot, stdio: 'pipe' },
    );
    for (const [entry, platform, format] of [
      ['main.ts', 'node', 'cjs'],
      ['preload.ts', 'node', 'cjs'],
      ['worker.ts', 'node', 'cjs'],
      ['worker-client.ts', 'node', 'cjs'],
      ['parser.ts', 'node', 'cjs'],
      ['targets.ts', 'node', 'cjs'],
      ['fixtures.ts', 'node', 'cjs'],
      ['renderer.tsx', 'browser', 'esm'],
    ]) {
      await build({
        entryPoints: [path.join(__dirname, entry)],
        outfile: path.join(
          scratch,
          entry.replace(/\.tsx?$/, platform === 'browser' ? '.js' : '.cjs'),
        ),
        bundle: true,
        platform,
        format,
        external: platform === 'node' ? ['electron'] : [],
        target: platform === 'browser' ? 'chrome152' : 'node24',
        jsx: 'automatic',
        logLevel: 'silent',
      });
    }
    await fs.writeFile(
      path.join(scratch, 'index.html'),
      '<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Gobble · Notebook qualification</title><link rel="stylesheet" href="/renderer.css"></head><body><div id="root"></div><script type="module" src="/renderer.js"></script></body></html>',
    );
    const env = Object.fromEntries(
      Object.entries(process.env).filter(
        ([key]) => !['ELECTRON_RUN_AS_NODE', 'ELECTRON_RENDERER_URL'].includes(key),
      ),
    );
    env.R4A1_PROFILE = path.join(scratch, 'profile');
    const files = [];
    for (const name of await fs.readdir(scratch)) {
      const bytes = await fs.readFile(path.join(scratch, name));
      files.push({
        name,
        bytes: bytes.length,
        sha256: createHash('sha256').update(bytes).digest('hex'),
      });
    }
    await fs.writeFile(
      path.join(evidence, 'tested-build.json'),
      JSON.stringify({ files }, null, 2) + '\n',
    );
    return { scratch, main: path.join(scratch, 'main.cjs'), env, cleanup };
  } catch (error) {
    await fs.writeFile(
      path.join(evidence, 'construction-' + Date.now() + '.log'),
      error.stdout?.toString() || String(error),
    );
    await cleanup();
    throw error;
  }
}
module.exports = { prepare, evidence, appRoot };
if (require.main === module)
  prepare()
    .then(async (b) => {
      console.log('Strict types and all isolated entries built.');
      await b.cleanup();
    })
    .catch((e) => {
      console.error(e);
      process.exitCode = 1;
    });
