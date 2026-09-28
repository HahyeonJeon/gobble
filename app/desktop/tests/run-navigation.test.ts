import { readFile } from 'node:fs/promises';
import { describe, expect, it } from 'vitest';
import {
  visibleRunTasks,
  observedTargetVisible,
  logUnavailableReason,
  type RunPresentation,
  type EvidenceRef,
  type Surface,
  SharedToolRequestV4Schema,
} from '@gobble/contracts';
const value: RunPresentation = {
  projectId: 'prj_nav',
  runRef: 'run_nav',
  engineRevision: 'revision',
  observedAt: 1000,
  imageId: 'image',
  runId: 'engine',
  pipelineName: null,
  status: 'failed',
  dependencies: [],
  tasks: [
    {
      instanceId: 'align[S01]',
      taskId: 'align',
      name: 'Align',
      status: 'failed',
      attempt: 2,
      reason: null,
    },
    {
      instanceId: 'align[S02]',
      taskId: 'align',
      name: 'Align',
      status: 'custom-state',
      attempt: 1,
      reason: null,
    },
    {
      instanceId: 'align',
      taskId: 'align',
      name: 'Align',
      status: null,
      attempt: 0,
      template: true,
      reason: null,
    },
  ],
};
const surface: Surface = {
  projectId: value.projectId,
  surfaceId: 'srf_run',
  resource: { kind: 'run', runRef: value.runRef },
  view: 'run',
  pinned: false,
  openedBy: { kind: 'user' },
  runView: { query: 'S01', status: null, viewRevision: 1 },
};
const ref: EvidenceRef = {
  schemaVersion: 3,
  projectId: value.projectId,
  resource: surface.resource,
  dataRevision: 'revision',
  selection: {
    kind: 'run-task',
    coordinateSpace: 'observed-instance-attempt',
    instanceId: 'align[S02]',
    attempt: 1,
  },
};
describe('returned task preview navigation', () => {
  it('keeps duplicate labels, custom states, templates and exact identities distinct', () => {
    expect(visibleRunTasks(value, { query: 'ALIGN', status: null })).toHaveLength(3);
    expect(visibleRunTasks(value, { query: 's02', status: 'custom-state' })).toEqual([
      value.tasks[1],
    ]);
    expect(visibleRunTasks(value, { query: 's02', status: 'failed' })).toEqual([]);
    expect(visibleRunTasks(value, { query: '', status: '' })).toEqual([value.tasks[2]]);
    expect(logUnavailableReason(value.tasks[2]!)).toContain('Template');
    expect(logUnavailableReason({ ...value.tasks[0]!, attempt: 0 })).toContain('No attempt');
    expect(logUnavailableReason(value.tasks[0]!)).toBeNull();
  });
  it('disallows attachment of a task outside the saved filter', () => {
    expect(observedTargetVisible(surface, { kind: 'run', value }, ref)).toBe(false);
    expect(
      observedTargetVisible(
        { ...surface, runView: { query: '', status: null, viewRevision: 2 } },
        { kind: 'run', value },
        ref,
      ),
    ).toBe(true);
  });
});

it('keeps the registered toolset v4 input shapes byte-equivalent to the published contract', async () => {
  const published = JSON.parse(
    await readFile(new URL('../../contracts/schema/v7.json', import.meta.url), 'utf8'),
  );
  expect(JSON.stringify(SharedToolRequestV4Schema)).toBe(
    JSON.stringify(published.definitions.SharedToolRequestSchema),
  );
});
