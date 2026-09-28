// Test-only Docker/engine protocol peer. No daemon, container, or bioinformatics tool is executed.
const fs = require('node:fs');
const path = require('node:path');
const crypto = require('node:crypto');
const base = process.env.GOBBLE_CONTINUATION_UI_FIXTURE;
const fixture = JSON.parse(fs.readFileSync(path.join(base, 'fixture.json')));
const args = process.argv.slice(2),
  joined = args.join(' ');
const read = (name) => JSON.parse(fs.readFileSync(path.join(base, name)));
const save = (name, value) => fs.writeFileSync(path.join(base, name), JSON.stringify(value));
const exists = (name) => fs.existsSync(path.join(base, name));
const hash = (v) => 'sha256:' + crypto.createHash('sha256').update(JSON.stringify(v)).digest('hex');
const out = (v) => console.log(typeof v === 'string' ? v : JSON.stringify(v));
fs.appendFileSync(path.join(base, 'commands.jsonl'), JSON.stringify(args) + '\n');
const original = fixture.original;
const values = (key) => args.flatMap((a, i) => (a === key ? [args[i + 1]] : []));
const binding = original.runtime;
const receipt = exists('receipt.json') ? read('receipt.json') : undefined;
const origin = {
  schemaVersion: 2,
  intentDigest: fixture.review.originDigest,
  requestId: original.intent.requestId,
  preparedDigest: original.intent.preparedDigest,
  workspaceId: original.intent.workspaceId,
  lease: 'a'.repeat(32),
};
if (args[0] === 'context') out(binding.endpoint);
else if (args[0] === 'info') out(binding.daemonId + ' linux');
else if (args[0] === 'image')
  out(
    joined.includes('continuation.scope')
      ? exists('old-engine')
        ? '<no value>'
        : 'single-end-trim-fastqc-v1'
      : binding.imageId + ' linux/amd64',
  );
else if (args[0] === 'create') {
  const mounts = values('--mount').map((m) => {
    const parts = Object.fromEntries(m.split(',').map((s) => s.split('=')));
    return { Source: parts.src, Destination: parts.dst, RW: !('readonly' in parts) };
  });
  save('controller.json', {
    Image: binding.imageId,
    Config: {
      Image: binding.imageId,
      User: '0:0',
      Env: values('--env'),
      WorkingDir: binding.workspacePath,
      Entrypoint: ['/usr/local/bin/gobble'],
      Cmd: ['prepared-continue', '--workspace', binding.workspacePath],
      Labels: { 'io.gobble.launch': values('--label')[0].slice('io.gobble.launch='.length) },
    },
    HostConfig: {
      NetworkMode: 'none',
      ReadonlyRootfs: true,
      Privileged: false,
      CapAdd: ['CAP_SETUID', 'CAP_SETGID'],
      CapDrop: ['ALL'],
      SecurityOpt: ['no-new-privileges'],
    },
    State: { Status: 'created' },
    Mounts: mounts,
  });
  if (exists('lose-ack')) process.exit(1);
  out('fixture-controller');
} else if (args[0] === 'inspect') {
  if (!exists('controller.json') || exists('lose-ack')) process.exit(1);
  out(read('controller.json'));
} else if (args[0] === 'start') {
  const controller = read('controller.json');
  const bundle = controller.Mounts.find((m) => m.Destination === '/gobble/launch').Source;
  const intent = JSON.parse(fs.readFileSync(path.join(bundle, 'continuation.json')));
  const r = { intent, intentDigest: hash(intent), lease: 'b'.repeat(32), snapshot: 'c'.repeat(32) };
  save('receipt.json', r);
  controller.State.Status = 'running';
  save('controller.json', controller);
  out('fixture-controller');
} else if (joined.includes('prepared-capabilities'))
  out({ schemaVersion: 1, launchSchemaVersion: 2, continuationVersion: 1 });
else if (joined.includes('prepared-continuation-review')) {
  if (exists('blocked')) {
    console.error('Checked result changed. Check the saved Run again.');
    process.exit(1);
  }
  out(fixture.review);
} else if (joined.includes('prepared-continuation-receipt'))
  out({ schemaVersion: 1, found: !!receipt, ...(receipt ? { receipt } : {}) });
else if (joined.includes('prepared-status'))
  out({
    schemaVersion: 2,
    admission: origin,
    status: receipt && !exists('stopped') ? 'running' : 'stopped',
    snapshot: receipt ? 'd'.repeat(32) : fixture.review.snapshot,
    ownerActive: !!receipt && !exists('stopped'),
    ownerLive: !!receipt && !exists('stopped'),
    epoch: {
      head: receipt ? hash(receipt) : origin.intentDigest,
      lease: receipt?.lease ?? origin.lease,
    },
  });
else if (joined.includes('prepared-stop')) {
  save('stopped', true);
  out({ status: 'settled', lease: receipt?.lease ?? origin.lease });
} else if (args[0] === 'run') {
  if (args.includes('identity'))
    out({ schema_version: 2, view: 'identity', match: true, required: fixture.identity });
  else
    out({
      schema_version: 2,
      snapshot: receipt ? 'd'.repeat(32) : fixture.review.snapshot,
      pipeline: 'Read quality analysis',
      run: { id: 'fixture-run', status: receipt && !exists('stopped') ? 'running' : 'stopped' },
      tasks: [
        { identity: 'trim', task_id: 'trim', name: 'Trim reads', status: 'succeeded', attempt: 1 },
        {
          identity: 'qc',
          task_id: 'qc',
          name: 'Quality check',
          status: receipt ? 'running' : 'canceled',
          attempt: receipt ? 2 : 1,
        },
      ],
      edges: [{ from: 'trim', to: 'qc' }],
      logs: [],
    });
} else if (args[0] !== 'rm') process.exit(1);
