import { EvidenceService } from '../src/main/evidence/service';
import { EvidenceStorage } from '../src/main/evidence/storage';
import type { EvidenceWorkspace, EvidenceDraft } from '../src/main/evidence/ports';
import type { WorkspaceResources } from '../src/main/workspace/service';
import { beforeAll, afterAll, afterEach, it, expect } from 'vitest';
import { build } from 'esbuild';
import { mkdtemp, rm, writeFile } from 'node:fs/promises';
import { join, resolve } from 'node:path';
import { tmpdir } from 'node:os';
import { NotebookHost } from '../src/main/notebook/host';
import { NotebookViews } from '../src/main/workspace/notebook-views';
import { activeWorkers, readNotebook } from '../src/main/notebook/worker-client';
import { fixture } from '../../qualification/notebook/fixtures';
import { digest } from '../src/main/notebook/text';
import type { FileContent, NotebookTarget, Surface } from '@gobble/contracts';
import { materializeNotebook } from '../src/main/evidence/notebook-evidence';
import { evidencePreview } from '../src/main/evidence/materialize';
let scratch: string, worker: string;
const owners: NotebookViews[] = [];
beforeAll(async () => {
  scratch = await mkdtemp(join(tmpdir(), 'gobble-notebook-unit-'));
  worker = join(scratch, 'worker.cjs');
  await build({
    entryPoints: [resolve('desktop/src/main/notebook/worker.ts')],
    outfile: worker,
    bundle: true,
    platform: 'node',
    format: 'cjs',
    target: 'node24',
  });
});
afterEach(async () => {
  owners.splice(0).forEach((owner) => owner.clear());
  await expect.poll(activeWorkers).toBe(0);
});
afterAll(async () => {
  await rm(scratch, { recursive: true, force: true });
});
const surface: Surface = {
  surfaceId: 'srf_notebook',
  projectId: 'prj_notebook',
  resource: { kind: 'file', resourceId: 'res_notebook' },
  view: 'notebook',
  pinned: false,
  openedBy: { kind: 'user' },
};
function file(name: Parameters<typeof fixture>[0] = 'python'): FileContent {
  const bytes = fixture(name);
  return {
    projectId: 'prj_notebook',
    resourceId: 'res_notebook',
    name: 'study.ipynb',
    size: bytes.length,
    revision: digest(bytes),
    content: { kind: 'notebook', base64: bytes.toString('base64') },
  };
}
function owner() {
  const views = new NotebookViews(() => new NotebookHost(worker));
  owners.push(views);
  views.retain('prj_notebook', ['srf_notebook']);
  return views;
}
function target(revision: string): NotebookTarget {
  return {
    schemaVersion: 6,
    projectId: 'prj_notebook',
    resource: { kind: 'file', resourceId: 'res_notebook' },
    origin: { surfaceId: 'srf_notebook' },
    dataRevision: revision,
    selection: {
      kind: 'notebook',
      profile: 'notebook-passive-1',
      cell: { kind: 'id', id: 'load' },
      part: { kind: 'source' },
      selector: { kind: 'text', coordinateSpace: 'notebook-display-utf16', start: 0, end: 24 },
    },
  };
}
it('retains the same source until explicit refresh and refuses replaced or mismatched bytes', async () => {
  const views = owner(),
    original = file();
  let reads = 0;
  const read = async () => {
    reads++;
    return original;
  };
  const loaded = await views.read(surface, read, false);
  await views.read(surface, read, false);
  expect(reads).toBe(1);
  const t = target(original.revision);
  views.validate(surface.surfaceId, t, loaded);
  await views.read(surface, async () => file('changed'), true);
  expect(() => views.capture(surface.surfaceId, t, loaded)).toThrow(/changed/);
  await expect(
    views.read(surface, async () => ({ ...original, revision: 'sha256:' + 'a'.repeat(64) }), true),
  ).rejects.toThrow(/revision/);
  await expect(views.read(surface, read, false)).resolves.toMatchObject({ kind: 'file' });
});
it('freezes text and original CRLF quote in the common evidence representation', async () => {
  const views = owner(),
    original = file(),
    data = await views.read(surface, async () => original, false);
  const t = target(original.revision);
  t.selection.selector = {
    kind: 'text',
    coordinateSpace: 'notebook-display-utf16',
    start: 0,
    end: 25,
  };
  const asset = await materializeNotebook(
    views,
    { attachmentId: 'att_n', evidence: t, label: 'study.ipynb', createdAt: 1 },
    data,
  );
  const preview = evidencePreview(asset);
  if (!('notebook' in preview)) throw new Error('Missing Notebook text evidence');
  expect(preview.notebook.rawQuote).toContain('\r\n');
  expect(preview.text).not.toContain('\r');
  views.clear();
  expect(evidencePreview(asset)).toEqual(preview);
  const corrupt = JSON.parse(asset.bytes.toString());
  corrupt.notebook.rawQuote = 'changed';
  expect(() => evidencePreview({ ...asset, bytes: Buffer.from(JSON.stringify(corrupt)) })).toThrow(
    /quote/,
  );
});
it('release and supersession prevent late source publication', async () => {
  const views = owner();
  let release!: (f: FileContent) => void;
  const pending = views.read(
    surface,
    () =>
      new Promise((resolve) => {
        release = resolve;
      }),
    false,
  );
  const rejected = expect(pending).rejects.toThrow(/replaced/);
  views.retain('prj_notebook', []);
  release(file());
  await rejected;
  views.retain('prj_notebook', ['srf_notebook']);
  await expect(views.read(surface, async () => file(), false)).resolves.toMatchObject({
    kind: 'file',
  });
});
it('worker deadline and abort terminate owned work and restore admission', async () => {
  const stuck = join(scratch, 'stuck.cjs');
  await writeFile(stuck, 'setInterval(() => {}, 1000)');
  await expect(
    readNotebook(fixture('python'), new AbortController().signal, stuck, 30),
  ).rejects.toThrow(/deadline/);
  const control = new AbortController();
  const pending = readNotebook(fixture('python'), control.signal, stuck);
  const result = expect(pending).rejects.toThrow(/cancelled/);
  control.abort();
  await result;
  expect(activeWorkers()).toBe(0);
  await expect(
    readNotebook(fixture('python'), new AbortController().signal, worker),
  ).resolves.toMatchObject({ profile: 'notebook-passive-1' });
});

it('previews an addressed captured draft offline after the reader is released, with an active Project check', async () => {
  const views = owner(),
    original = file(),
    data = await views.read(surface, async () => original, false);
  const attachment = {
    attachmentId: 'att_offline',
    evidence: target(original.revision),
    label: 'study.ipynb',
    createdAt: 1,
  };
  const asset = await materializeNotebook(views, attachment, data);
  const storage = new EvidenceStorage(join(scratch, 'offline-evidence'));
  await storage.putCaptured(surface.projectId, asset);
  views.clear();
  const draft: EvidenceDraft = {
    rendererSessionId: 'rnd_notebook',
    attachmentRevision: 1,
    agent: null,
    attachments: [
      {
        ...attachment,
        capture: {
          kind: 'notebook',
          capturedAt: asset.manifest.capture!.capturedAt,
          asset: asset.manifest.asset,
          representation: { kind: 'text', truncated: false, notebook: true },
        },
      },
    ],
  };
  const read = async (id: string) => {
    if (id !== surface.projectId) throw new Error('Inactive Project');
    return structuredClone(draft);
  };
  const workspace: EvidenceWorkspace = {
    readEvidenceDraft: read,
    assertEvidenceSession: () => {},
    readSentEvidence: async () => {
      throw new Error('No message');
    },
  };
  const unavailable = async () => {
    throw new Error('Live source must not be read');
  };
  const resources: WorkspaceResources = {
    projects: async () => [],
    read: unavailable,
    describe: unavailable,
  };
  const service = new EvidenceService(
    workspace,
    resources,
    new EvidenceStorage(join(scratch, 'offline-evidence')),
    unavailable,
    {
      require: () => {
        throw new Error('Account must not be required');
      },
      supportsImages: () => false,
    },
  );
  const request = {
    kind: 'draft' as const,
    projectId: surface.projectId,
    attachmentId: attachment.attachmentId,
  };
  await expect(service.preview(request)).resolves.toEqual(evidencePreview(asset));
  await expect(service.preview({ ...request, projectId: 'prj_other' })).rejects.toThrow(
    'Inactive Project',
  );
  await expect(service.preview({ ...request, attachmentId: 'att_missing' })).rejects.toThrow(
    'no captured attachment',
  );
  let reads = 0;
  workspace.readEvidenceDraft = async (id) => {
    const value = await read(id);
    if (++reads === 2) value.rendererSessionId = 'rnd_replaced';
    return value;
  };
  await expect(service.preview(request)).rejects.toThrow('changed');
  reads = 0;
  workspace.readEvidenceDraft = async (id) => {
    const value = await read(id);
    if (++reads === 2) value.attachments = [];
    return value;
  };
  await expect(service.preview(request)).rejects.toThrow('changed');
});

it('opens an exact temporary reference without replacing the retained User reader', async () => {
  const views = owner(),
    original = file(),
    data = await views.read(surface, async () => original, false),
    t = target(original.revision);
  const ref = await views.reference(t, async () => original);
  expect(ref).toMatchObject({
    kind: 'file',
    value: {
      content: {
        kind: 'notebook',
        reference: { target: t, content: { kind: 'text', text: '# Unicode 🧬 and CRLF\nim' } },
      },
    },
  });
  expect(() => views.validate(surface.surfaceId, t, data)).not.toThrow();
  await expect(views.reference(t, async () => file('changed'))).rejects.toThrow(
    /exact Notebook source/,
  );
  await expect(
    views.reference(t, async () => ({ ...original, resourceId: 'res_foreign' })),
  ).rejects.toThrow(/exact Notebook source/);
});
it('cancels reference reading on Project disposal and leaves no worker', async () => {
  const views = owner(),
    original = file();
  let release!: (value: FileContent) => void;
  const pending = views.reference(
    target(original.revision),
    () =>
      new Promise((resolve) => {
        release = resolve;
      }),
  );
  views.clear();
  release(original);
  await expect(pending).rejects.toThrow(/replaced/);
});
