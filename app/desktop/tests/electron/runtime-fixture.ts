import { chmod, copyFile, mkdir, mkdtemp, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join, resolve } from 'node:path';

/** Isolated synthetic runtime process; no engine execution or Docker daemon. */
export async function runtimeFixture(baseDirectory?: string) {
  const directory = baseDirectory ?? (await mkdtemp(join(tmpdir(), 'gobble-runtime-ui-')));
  await mkdir(directory, { recursive: true });
  const root = join(directory, 'Runtime fixture');
  const bin = join(directory, 'bin');
  const profile = join(directory, 'profile');
  const monitorPath = join(directory, 'monitor.json');
  const executable = join(directory, 'codex');
  await copyFile(resolve('desktop/tests/fixtures/codex.cjs'), executable);
  await chmod(executable, 0o700);
  await mkdir(join(root, 'runs', 'existing', '.gobble'), { recursive: true });
  await mkdir(bin);
  const image = 'sha256:' + 'a'.repeat(64);
  await writeFile(
    join(root, '.gobble-runtime.json'),
    JSON.stringify({ format: 1, image, daemon: 'fixture-daemon' }),
  );
  const monitor = {
    schema_version: 2,
    snapshot: 'fixture-revision-4',
    pipeline: 'Fixture',
    run: { id: 'fixture-run', status: 'completed' },
    tasks: [
      {
        identity: 'prepare',
        task_id: 'prepare',
        name: 'Prepare samples',
        status: 'completed',
        attempt: 1,
      },
      {
        identity: 'align:S03',
        task_id: 'align',
        name: 'Align reads',
        status: 'failed',
        attempt: 2,
      },
      {
        identity: 'align:S04',
        task_id: 'align',
        name: 'Align reads',
        status: 'custom-state',
        attempt: 0,
      },
      {
        identity: 'align-template',
        task_id: 'align',
        name: 'Align template',
        status: 'pending',
        attempt: 0,
        template: true,
      },
    ],
    edges: [{ from: 'prepare', to: 'align' }],
    logs: [
      {
        identity: 'align:S03',
        stdout_tail: 'Read sample S03\nAlignment complete',
        stderr_tail: 'ERROR 😀 exact\nShared line\nstderr only',
        stdout_size: 8192,
        stderr_size: 80,
      },
    ],
  };
  await writeFile(monitorPath, JSON.stringify(monitor));
  // Test-local process boundary only; no daemon, image, engine or analysis runs here.
  const script =
    '#!' +
    process.execPath +
    '\n' +
    'const args = process.argv.slice(2);\n' +
    'if (args[0] === "context") console.log("unix:///fixture/docker.sock");\n' +
    'else if (args[0] === "info") console.log("fixture-daemon linux");\n' +
    'else if (args[0] === "image") console.log(' +
    JSON.stringify(image + ' linux/amd64') +
    ');\n' +
    'else if (args[0] === "run") console.log(JSON.stringify(args.includes("identity") ? ' +
    JSON.stringify({
      schema_version: 2,
      view: 'identity',
      match: true,
      required: { identity_mode: 'local-pin', gobble_module: 'fixture' },
    }) +
    ' : ' +
    'JSON.parse(require("node:fs").readFileSync(' +
    JSON.stringify(monitorPath) +
    ', "utf8"))' +
    '));\n' +
    'else if (args[0] !== "rm") process.exitCode = 1;\n';
  await writeFile(join(bin, 'docker'), script);
  await chmod(join(bin, 'docker'), 0o700);
  const environment = {
    PATH: bin + ':' + (process.env.PATH ?? ''),
    DOCKER_HOST: 'unix:///fixture/docker.sock',
    DOCKER_CONTEXT: '',
    GOBBLE_CODEX_EXECUTABLE: executable,
  };
  return { directory, root, profile, monitorPath, monitor, environment };
}
