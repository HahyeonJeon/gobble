import { readFile } from 'node:fs/promises';
import { describe, expect, it } from 'vitest';
import { logResource, logTarget, readStoredWorkspace, sameResource } from '@gobble/contracts';
import { paneOf } from '../src/main/workspace/model';
import { ProjectService } from '../src/main/service/project-service';
import { presentRun } from '../src/main/service/run-presentation';

function snapshot() {
  return {
    projectId: 'prj_atlas',
    runRef: 'run_first',
    engineRevision: 'revision-7',
    observedAt: 1000,
    runtimeBinding: {
      endpoint: 'unix:///fixture',
      daemonId: 'fixture',
      imageId: 'sha256:' + 'a'.repeat(64),
      platform: 'linux/amd64' as const,
      projectPath: '/gobble/project' as const,
      workspacePath: '/gobble/project/run',
    },
    availability: 'available' as const,
    snapshot: {
      schema_version: 2 as const,
      snapshot: 'revision-7',
      pipeline: 'RNA analysis',
      run: { id: 'engine-first', status: 'blocked' },
      tasks: [
        {
          identity: 'align:S03',
          task_id: 'align',
          name: 'Align S03',
          attempt: 2,
          status: 'failed',
          reason: 'Tool exit',
        },
        {
          identity: 'align:S04',
          task_id: 'align',
          name: 'Align S04',
          attempt: 1,
          status: 'running',
        },
      ],
      edges: [{ from: 'prepare', to: 'align' }],
      logs: [],
    },
  };
}

describe('domain identity and persisted v1 compatibility', () => {
  it('migrates v1 presentation while retaining log identity, selection and draft', async () => {
    const raw = JSON.parse(
      await readFile(new URL('./fixtures/workspace-v1.json', import.meta.url), 'utf8'),
    ) as unknown;
    const doc = readStoredWorkspace(raw);
    expect(doc.schemaVersion).toBe(21);
    expect(doc.paneOrientation).toBe('vertical');
    const resource = doc.workspace.surfaces[0]?.resource;
    if (!resource || resource.kind !== 'log') throw new Error('Missing saved log');
    expect(logTarget(resource)).toEqual({
      runRef: 'run_first',
      instanceId: 'align:S03',
      attempt: 2,
    });
    expect(sameResource(resource, logResource(logTarget(resource)))).toBe(true);
    expect(doc.chat.draft).toBe('Keep this draft');
    expect(paneOf(doc, 'srf_legacy')).toBe('primary');
    expect(() => paneOf(doc, 'srf_missing')).toThrow('not placed');
  });
});

describe('native Run facts to typed display model', () => {
  it('preserves instance IDs, authored dependencies and observed Pipeline name as separate concepts', () => {
    const value = presentRun(snapshot());
    expect(value.pipelineName).toBe('RNA analysis');
    expect(value.status).toBe('blocked');
    expect(value.tasks.map(({ instanceId, taskId }) => ({ instanceId, taskId }))).toEqual([
      { instanceId: 'align:S03', taskId: 'align' },
      { instanceId: 'align:S04', taskId: 'align' },
    ]);
    expect(value.dependencies).toEqual([{ fromTaskId: 'prepare', toTaskId: 'align' }]);
    expect(
      logResource({ runRef: value.runRef, instanceId: value.tasks[0]!.instanceId, attempt: 2 })
        .taskId,
    ).toBe('align:S03');
  });
  it('keeps missing metadata unavailable and rejects malformed supplied metadata', () => {
    const value = snapshot();
    Reflect.deleteProperty(value.snapshot, 'pipeline');
    Reflect.deleteProperty(value.snapshot.tasks[0]!, 'task_id');
    expect(presentRun(value).pipelineName).toBeNull();
    expect(presentRun(value).tasks[0]?.taskId).toBeNull();
    Object.assign(value.snapshot, { pipeline: 42 });
    expect(() => presentRun(value)).toThrow('invalid display metadata');
  });
});

describe('named service gateway', () => {
  it('rejects an otherwise valid response addressed to another Project or Run', async () => {
    const value = snapshot();
    const service = new ProjectService({
      request: async () => ({ schemaVersion: 1, ok: true, value }),
    });
    await expect(
      service.readRun({ projectId: 'prj_other', runRef: value.runRef }),
    ).rejects.toMatchObject({ code: 'internal' });
    await expect(
      service.readRun({ projectId: value.projectId, runRef: 'run_other' }),
    ).rejects.toMatchObject({ code: 'internal' });
    await expect(
      service.readRun({ projectId: value.projectId, runRef: value.runRef }),
    ).resolves.toEqual(value);
  });
  it('preserves service error classification instead of treating it as successful metadata', async () => {
    const service = new ProjectService({
      request: async () => ({
        schemaVersion: 1,
        ok: false,
        error: { code: 'incompatible_runtime', message: 'Wrong runtime', retry: 'never' },
      }),
    });
    await expect(service.listProjects()).rejects.toMatchObject({
      code: 'incompatible_runtime',
      message: 'Wrong runtime',
    });
  });
  it('addresses logs by encoded instance and exact attempt, rejecting a different returned attempt', async () => {
    let path = '';
    const value = {
      projectId: 'prj_atlas',
      runRef: 'run_first',
      engineRevision: 'revision-7',
      observedAt: 1000,
      instance: 'align:S03/1',
      attempt: 2,
      tailLimitBytes: 4096,
      logs: [],
    };
    const service = new ProjectService({
      request: async (url) => {
        path = url;
        return { schemaVersion: 1, ok: true, value };
      },
    });
    const input = {
      projectId: value.projectId,
      runRef: value.runRef,
      instance: value.instance,
      attempt: 2,
    };
    await expect(service.readLogs(input)).resolves.toEqual(value);
    expect(new URL(path, 'http://local').searchParams.get('instance')).toBe('align:S03/1');
    value.attempt = 3;
    await expect(service.readLogs(input)).rejects.toMatchObject({ code: 'stale_revision' });
  });
});
