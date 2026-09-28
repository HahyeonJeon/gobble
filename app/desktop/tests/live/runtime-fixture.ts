import { execFile } from 'node:child_process';
import { promisify } from 'node:util';
import { appendFile, copyFile, mkdir, readFile, realpath, writeFile } from 'node:fs/promises';
import { join, resolve } from 'node:path';

const execute = promisify(execFile);

/** Owns only its fresh Project and detached controller; never resets the Docker daemon. */
export async function createRuntimeFixture(directory: string, imageReference: string) {
  await mkdir(directory, { recursive: true });
  const base = await realpath(directory);
  const projectName = 'verified-integration';
  const project = join(base, projectName);
  const audit = join(base, 'commands.jsonl');
  let controller: string | null = null;
  const env = Object.fromEntries(
    Object.entries(process.env).filter(([key]) => !key.startsWith('GOBBLE_')),
  );
  async function docker(cwd: string, args: string[]) {
    const startedAt = new Date().toISOString();
    try {
      const result = await execute('docker', args, {
        cwd,
        env,
        timeout: 180_000,
        maxBuffer: 8 * 1024 * 1024,
      });
      await appendFile(
        audit,
        JSON.stringify({ startedAt, cwd, args, exitCode: 0, ...result }) + '\n',
      );
      return result.stdout.trim();
    } catch (error) {
      const detail = error as {
        code?: unknown;
        stdout?: string;
        stderr?: string;
        message?: string;
      };
      await appendFile(
        audit,
        JSON.stringify({ startedAt, cwd, args, error: detail.message, ...detail }) + '\n',
      );
      throw error;
    }
  }
  const image = await docker(base, ['image', 'inspect', '--format', '{{.Id}}', imageReference]);
  const platform = await docker(base, [
    'image',
    'inspect',
    '--format',
    '{{.Os}}/{{.Architecture}}',
    image,
  ]);
  if (!/^sha256:[a-f0-9]{64}$/.test(image) || platform !== 'linux/amd64')
    throw new Error(
      'Live verification requires an existing exact Linux/amd64 Gobble runtime image.',
    );
  const compose = (await readFile(resolve('../distribution/runtime/compose.yaml'), 'utf8')).replace(
    'ghcr.io/hahyeonjeon/gobble:develop',
    image,
  );
  await writeFile(join(base, 'compose.yaml'), compose);
  const composeName = 'gobble-desktop-live-' + process.pid;
  const composeArgs = ['compose', '--project-name', composeName, 'run'];
  const cliAt = (cwd: string, ...args: string[]) =>
    docker(cwd, [...composeArgs, '--rm', '-T', 'gobble', ...args]);
  await cliAt(base, 'init', projectName);
  const cli = (...args: string[]) => cliAt(project, ...args);
  return {
    base,
    project,
    image,
    cli,
    async startGated() {
      const pipeline = join(project, 'pipeline.go');
      const source = await readFile(pipeline, 'utf8');
      const changed = source
        .replace(
          'Name: "count-sequences",',
          'Name: "count-sequences",\n        Image: ' + JSON.stringify(image) + ',',
        )
        .replace(
          "awk '/^>/",
          "echo stage6-started; i=0; until test -f inputs/continue; do i=$((i+1)); test $i -lt 180 || exit 88; sleep 1; done; echo stage6-finished; awk '/^>/",
        );
      if (changed === source) throw new Error('The starter pipeline format has changed.');
      await writeFile(pipeline, changed);
      await mkdir(join(project, 'runs', 'live', 'inputs'), { recursive: true });
      await copyFile(
        join(project, 'runs', 'hello', 'inputs', 'sequences.fasta'),
        join(project, 'runs', 'live', 'inputs', 'sequences.fasta'),
      );
      controller = await docker(project, [
        ...composeArgs,
        '-d',
        'gobble',
        'run',
        '.',
        '--workspace',
        'runs/live',
        '--cap',
        '1',
      ]);
      if (!/^[a-f0-9]{64}$/.test(controller))
        throw new Error('Docker did not return one detached controller ID.');
      return controller;
    },
    async controllerRunning() {
      if (!controller) return false;
      return (
        (await docker(project, ['inspect', '--format', '{{.State.Running}}', controller])) ===
        'true'
      );
    },
    async release() {
      // Inputs are staged copies. Release the gate inside this fixture's exact task
      // container; changing the workspace input cannot reach an already running task.
      const controls = join(project, 'runs/live/.gobble');
      const pointer = JSON.parse(await readFile(join(controls, 'current.json'), 'utf8'));
      if (!/^[a-f0-9]{32}$/.test(pointer.current)) throw new Error('Invalid fixture checkpoint');
      const state = JSON.parse(
        await readFile(join(controls, 'checkpoints', pointer.current, 'tasks.json'), 'utf8'),
      );
      const task = state.tasks.find(
        (item: { id: string; attempt: number }) =>
          item.id === 'count-sequences' && item.attempt === 1,
      );
      if (!task || !/^[a-f0-9]{64}$/.test(task.runtime_id))
        throw new Error('Missing owned task container');
      const token = await docker(project, [
        'inspect',
        '--format',
        '{{index .Config.Labels "io.gobble.submission"}}',
        task.runtime_id,
      ]);
      if (token !== task.submission.token) throw new Error('Task ownership differs');
      await docker(project, ['exec', task.runtime_id, 'touch', '/work/inputs/continue']);
    },
    async controllerExit() {
      if (!controller) throw new Error('No owned controller');
      return docker(project, ['wait', controller]);
    },
    async close() {
      if (!controller) return;
      try {
        if (await this.controllerRunning()) await cli('stop', '--workspace', 'runs/live');
        await docker(project, ['rm', controller]);
      } finally {
        controller = null;
      }
    },
  };
}
