import { execFileSync } from 'node:child_process';
import { mkdirSync } from 'node:fs';
import { fileURLToPath } from 'node:url';

const directory = fileURLToPath(new URL('../desktop/out/service/', import.meta.url));
mkdirSync(directory, { recursive: true });
execFileSync('go', ['build', '-o', `${directory}/gobble-service`, './cmd/gobble-service'], {
  cwd: fileURLToPath(new URL('../../', import.meta.url)),
  stdio: 'inherit',
});
