import { ProjectService } from '../src/main/service/project-service';
import { afterEach, describe, expect, it } from 'vitest';
import { mkdtemp, mkdir, readFile, rm, writeFile } from 'node:fs/promises';
import { join, resolve } from 'node:path';
import { tmpdir } from 'node:os';
import { spawn } from 'node:child_process';
import { randomBytes } from 'node:crypto';
import { once } from 'node:events';
import { ProjectServiceClient } from '../src/main/service/client';
import {
  parse,
  ProjectResultSchema,
  ProjectsResultSchema,
  DirectoryResultSchema,
  FileResultSchema,
  RunsResultSchema,
  RunResultSchema,
  ServiceHandshakeSchema,
  CapabilitiesResultSchema,
} from '@gobble/contracts';

const clients: ProjectServiceClient[] = [];
const directories: string[] = [];
const executable = resolve('desktop/out/service/gobble-service');

async function temporary() {
  const path = await mkdtemp(join(tmpdir(), 'gobble-service-test-'));
  directories.push(path);
  return path;
}
function client(profile: string) {
  const value = new ProjectServiceClient(executable, profile);
  clients.push(value);
  return value;
}
afterEach(async () => {
  await Promise.all(clients.splice(0).map((client) => client.stop()));
  await Promise.all(
    directories.splice(0).map((path) => rm(path, { recursive: true, force: true })),
  );
});

describe('native service and TypeScript contract integration', () => {
  it('stores an independent creation draft through the actual service and restores its tombstone', async () => {
    const profile = await temporary();
    const root = await temporary();
    await writeFile(join(root, 'reads.fastq.gz'), 'metadata-only fixture');
    const first = client(profile);
    const result = parse(
      ProjectResultSchema,
      await first.request('/v1/projects', 'POST', {
        requestId: 'req_project',
        root,
        name: 'Read study',
      }),
    );
    if (!result.ok) throw new Error(result.error.message);
    const projectId = result.value.projectId;
    const service = new ProjectService(first);
    const files = await service.listFiles({ projectId });
    const resourceId = files.entries.find((file) => file.name === 'reads.fastq.gz')!.resourceId;
    const draft = await service.drafts.create({
      projectId,
      requestId: 'req_draft',
      brief: 'Trim and check reads',
    });
    const updated = await service.drafts.update({
      projectId,
      draftId: draft.draftId,
      requestId: 'req_input',
      expectedGeneration: 1,
      brief: draft.brief,
      resourceId,
      readLayout: 'single-end',
    });
    expect(updated).toMatchObject({
      generation: 2,
      input: { resourceId, relativePath: 'reads.fastq.gz', readLayout: 'single-end' },
    });
    expect(JSON.stringify(updated)).not.toContain(root);
    expect((await service.listPipelines({ projectId })).pipelines).toEqual([]);
    await first.stop();
    const second = client(profile);
    const restored = new ProjectService(second);
    expect(await restored.drafts.read({ projectId, draftId: draft.draftId })).toEqual(updated);
    await restored.drafts.discard({
      projectId,
      draftId: draft.draftId,
      requestId: 'req_discard',
      expectedGeneration: 2,
    });
    expect((await restored.drafts.list({ projectId })).drafts).toEqual([]);
    await expect(
      restored.drafts.update({
        projectId,
        draftId: draft.draftId,
        requestId: 'req_late',
        expectedGeneration: 2,
        brief: 'late',
        resourceId: '',
        readLayout: '',
      }),
    ).rejects.toThrow('changed or was discarded');
    const retry = await restored.drafts.create({
      projectId,
      requestId: 'req_draft',
      brief: 'Trim and check reads',
    });
    expect(retry).toMatchObject({ draftId: draft.draftId, state: 'discarded', generation: 3 });
    expect((await restored.listRuns({ projectId })).runs).toEqual([]);
  });

  it('registers a folder, reads real CSV/text and restores resource references', async () => {
    const profile = await temporary();
    const root = await temporary();
    await writeFile(join(root, 'samples.csv'), 'sample,condition\nS03,control\n');
    await writeFile(join(root, 'notes.txt'), 'Review S03');
    const first = client(profile);
    const created = parse(
      ProjectResultSchema,
      await first.request('/v1/projects', 'POST', { requestId: 'req_create', root, name: 'Atlas' }),
    );
    expect(created.ok).toBe(true);
    if (!created.ok) throw new Error(created.error.message);
    expect(JSON.stringify(created)).not.toContain(root);
    const project = created.value;
    const listing = parse(
      DirectoryResultSchema,
      await first.request(`/v1/projects/${project.projectId}/files`),
    );
    if (!listing.ok) throw new Error(listing.error.message);
    const csv = listing.value.entries.find((entry) => entry.name === 'samples.csv');
    if (!csv) throw new Error('CSV missing');
    const preview = parse(
      FileResultSchema,
      await first.request(`/v1/projects/${project.projectId}/files/${csv.resourceId}`),
    );
    expect(preview).toMatchObject({
      ok: true,
      value: { content: { kind: 'table', rows: [{ key: 'row_1', cells: ['S03', 'control'] }] } },
    });
    await first.stop();
    const restored = client(profile);
    expect(parse(ProjectsResultSchema, await restored.request('/v1/projects'))).toMatchObject({
      ok: true,
      value: [project],
    });
    expect(
      parse(
        FileResultSchema,
        await restored.request(`/v1/projects/${project.projectId}/files/${csv.resourceId}`),
      ),
    ).toEqual(preview);
    expect(
      parse(
        ProjectResultSchema,
        await restored.request('/v1/projects', 'POST', {
          requestId: 'req_create',
          root,
          name: 'Atlas',
        }),
      ),
    ).toEqual(created);
    const state = await readFile(join(profile, 'service/catalog.json'), 'utf8');
    expect(state).not.toContain('token');
  });

  it('discovers unresolved runs and reports a missing runtime without fabricating state', async () => {
    const root = await temporary();
    await mkdir(join(root, 'runs/existing/.gobble'), { recursive: true });
    const service = client(await temporary());
    const project = parse(
      ProjectResultSchema,
      await service.request('/v1/projects', 'POST', {
        requestId: 'req_project',
        root,
        name: 'Atlas',
      }),
    );
    if (!project.ok) throw new Error(project.error.message);
    const runs = parse(
      RunsResultSchema,
      await service.request(`/v1/projects/${project.value.projectId}/runs`),
    );
    if (!runs.ok) throw new Error(runs.error.message);
    expect(runs.value.runs).toEqual([]);
    expect(runs.value.candidates).toHaveLength(1);
    const candidate = runs.value.candidates[0];
    if (!candidate) throw new Error('Candidate missing');
    const attached = parse(
      RunResultSchema,
      await service.request(`/v1/projects/${project.value.projectId}/runs/attach`, 'POST', {
        requestId: 'req_attach',
        workspaceResourceId: candidate.workspaceResourceId,
      }),
    );
    expect(attached).toMatchObject({ ok: false, error: { code: 'incompatible_runtime' } });
  });

  it('handles process startup failure and stops without a hanging child', async () => {
    const missing = new ProjectServiceClient(
      join(await temporary(), 'missing-binary'),
      await temporary(),
    );
    clients.push(missing);
    await expect(missing.request('/v1/projects')).rejects.toThrow('could not start');
    await missing.stop();
  });

  it('uses a private bootstrap token and exits when the parent pipe closes', async () => {
    const profile = await temporary();
    const token = randomBytes(32).toString('hex');
    const child = spawn(executable, [], { stdio: ['pipe', 'pipe', 'pipe'] });
    const closed = once(child, 'close');
    try {
      const ready = new Promise<string>((resolve, reject) => {
        let buffer = '';
        child.on('error', reject);
        child.stdout.on('data', (data: Buffer) => {
          buffer += data.toString();
          if (buffer.includes('\n')) resolve(buffer.split('\n')[0] ?? '');
        });
        child.stderr.on('data', () => {});
      });
      child.stdin.write(`${JSON.stringify({ schemaVersion: 1, token, profilePath: profile })}\n`);
      const line = await ready;
      expect(line).not.toContain(token);
      expect(child.spawnargs.join(' ')).not.toContain(token);
      const handshake = parse(ServiceHandshakeSchema, JSON.parse(line));
      const url = `http://127.0.0.1:${handshake.port}/v1/capabilities`;
      const refused = await fetch(url);
      expect(refused.status).toBe(403);
      const browser = await fetch(url, {
        headers: { Authorization: `Bearer ${token}`, Origin: 'https://example.invalid' },
      });
      expect(browser.status).toBe(403);
      const response = await fetch(url, { headers: { Authorization: `Bearer ${token}` } });
      expect(parse(CapabilitiesResultSchema, await response.json())).toMatchObject({
        ok: true,
        value: {
          protocolVersion: 1,
          mutations: [
            'register_project',
            'register_pipeline',
            'import_pipeline',
            'check_pipeline',
            'prepare_pipeline',
            'cancel_preparation',
            'check_launch',
            'cancel_launch_check',
            'start_launch',
            'refresh_launch',
            'stop_launch',
            'check_continuation',
            'confirm_continuation',
            'refresh_continuation',
            'stop_continuation',
            'cancel_pipeline_check',
            'propose_pipeline',
            'adopt_pipeline',
            'create_pipeline_draft',
            'update_pipeline_draft',
            'discard_pipeline_draft',
            'bind_creation_runtime',
            'submit_creation_candidate',
            'cancel_creation_check',
            'attach_run',
          ],
        },
      });
      child.stdin.end();
      expect((await closed)[0]).toBe(0);
      await expect(fetch(url)).rejects.toThrow();
    } finally {
      child.kill('SIGKILL');
      await closed;
    }
  });
});
