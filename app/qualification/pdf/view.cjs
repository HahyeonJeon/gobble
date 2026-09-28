const { spawn } = require('node:child_process');
const fs = require('node:fs/promises');
const { prepare } = require('./build.cjs');

// The demo has its own temporary profile and never starts Gobble services or Codex.
async function show() {
  const build = await prepare();
  try {
    await new Promise((resolve, reject) => {
      const child = spawn(require('electron'), [build.main, '--show'], {
        env: build.env,
        stdio: 'inherit',
      });
      child.once('error', reject);
      child.once('exit', (code, signal) => {
        if (code === 0) resolve();
        else reject(new Error(`PDF qualification closed with ${signal || code}.`));
      });
    });
  } finally {
    await fs.rm(build.scratch, { recursive: true, force: true });
  }
}

void show().catch((error) => {
  console.error(error.message);
  process.exitCode = 1;
});
