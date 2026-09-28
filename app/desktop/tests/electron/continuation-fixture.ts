import { execFile } from 'node:child_process';
import { promisify } from 'node:util';
import { mkdir, writeFile, readFile, chmod, copyFile } from 'node:fs/promises';
import { join, resolve } from 'node:path';

/** Native serialized fixture + isolated process peer. Never modifies a user's profile. */
export async function continuationFixture(base: string) {
  await mkdir(base, { recursive: true });
  const port = (name: string, path: string) => ({ name, path, kind: 'file', members: [] });
  const control = {
    branch: '',
    merge: '',
    scatter: '',
    gather: '',
    when: '',
    scatterFromKind: '',
    scatterFromTask: '',
    scatterFromPort: '',
    scatterFromPath: '',
    scatterMembers: [],
    scatterMemberPaths: [],
    skipIfMissingTask: '',
    skipIfMissingPort: '',
    skipIfMissingPath: '',
    skipIfFalse: '',
  };
  const step = (
    id: string,
    name: string,
    inputs: ReturnType<typeof port>[],
    outputs: ReturnType<typeof port>[],
  ) => ({
    id,
    name,
    module: name,
    display: { stage: name },
    image: 'fixture',
    backend: 'docker',
    cpu: 1,
    memory: '',
    inputs,
    outputs,
    control,
    settings: [],
  });
  await writeFile(
    join(base, 'flow.json'),
    JSON.stringify({
      schemaVersion: 2,
      name: 'Read quality analysis',
      inputs: [port('reads', 'inputs/reads.fastq.gz')],
      steps: [
        step(
          'trim',
          'Trim reads',
          [port('read1', 'inputs/reads.fastq.gz')],
          [port('trimmed_read1', 'work/trimmed.fq.gz')],
        ),
        step(
          'qc',
          'Quality check',
          [port('reads', 'work/trimmed.fq.gz')],
          [port('report', 'results/quality.html')],
        ),
      ],
      connections: [
        {
          id: 'reads-trim',
          fromTask: '',
          fromPort: 'reads',
          toTask: 'trim',
          toPort: 'read1',
          wait: [],
        },
        {
          id: 'trim-qc',
          fromTask: 'trim',
          fromPort: 'trimmed_read1',
          toTask: 'qc',
          toPort: 'reads',
          wait: [],
        },
      ],
    }),
  );
  await promisify(execFile)(
    'go',
    ['test', './internal/appservice', '-run', '^TestExportContinuationUIFixture$', '-count=1'],
    {
      cwd: resolve('..'),
      env: { ...process.env, GOBBLE_CONTINUATION_UI_FIXTURE: base },
      timeout: 90000,
    },
  ).catch((e) => {
    throw new Error(String(e.stdout) + String(e.stderr));
  });
  const fixture = JSON.parse(await readFile(join(base, 'fixture.json'), 'utf8'));
  const bin = join(base, 'bin');
  await mkdir(bin);
  const source = await readFile(resolve('desktop/tests/fixtures/continuation-docker.cjs'), 'utf8');
  await writeFile(
    join(bin, 'docker'),
    '#!' +
      process.execPath +
      '\n' +
      source.replace('process.env.GOBBLE_CONTINUATION_UI_FIXTURE', JSON.stringify(base)),
  );
  await chmod(join(bin, 'docker'), 0o700);
  const codex = join(base, 'codex');
  await copyFile(resolve('desktop/tests/fixtures/codex.cjs'), codex);
  await chmod(codex, 0o700);
  return {
    ...fixture,
    base,
    environment: {
      PATH: bin + ':' + process.env.PATH,
      GOBBLE_CONTINUATION_UI_FIXTURE: base,
      GOBBLE_CODEX_EXECUTABLE: codex,
      DOCKER_HOST: fixture.original.runtime.endpoint,
      DOCKER_CONTEXT: '',
    },
  };
}
