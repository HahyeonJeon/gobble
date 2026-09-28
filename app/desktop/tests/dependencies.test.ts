import { describe, expect, it } from 'vitest';
import { readFile } from 'node:fs/promises';
import { createHash } from 'node:crypto';
import {
  parse,
  DependencyTargetSchema,
  DependencyReadSchema,
  EvidenceRefSchema,
  projectDependencies,
  readDependencyCapture,
  resolveDependencyTarget,
  dependencyOrder,
  DEPENDENCY_LIMITS,
  type DependencyTarget,
  type DependencyObservation,
} from '@gobble/contracts';
import { dependencySnapshot } from './fixtures/dependency-snapshot';
import { presentDependencies } from '../src/main/service/dependency-presentation';
import {
  captureDependency,
  dependencyRevision,
  readDependencyCaptureBytes,
} from '../src/main/evidence/dependency-capture';
import { presentRun } from '../src/main/service/run-presentation';
import { dataRevision, observedDataRevision } from '../src/main/workspace/selection';
const observed = () => projectDependencies(presentDependencies(dependencySnapshot()));
const target = (
  observation: DependencyObservation,
  selection: DependencyTarget['selection'] = {
    kind: 'run-group',
    coordinateSpace: 'observed-authored-task-group',
    taskId: 'align',
  },
): DependencyTarget => ({
  schemaVersion: 4,
  projectId: observation.source.projectId,
  resource: { kind: 'run', runRef: observation.source.runRef },
  dataRevision: dependencyRevision(observation),
  selection,
});
const edge: DependencyTarget['selection'] = {
  kind: 'run-dependency',
  coordinateSpace: 'observed-authored-task-pair',
  fromTaskId: 'prepare',
  toTaskId: 'align',
};

describe('dependency source and projection', () => {
  it('distinguishes absent, empty and invalid topology without changing Task facts', () => {
    const raw = dependencySnapshot();
    raw.snapshot.edges = undefined;
    expect(projectDependencies(presentDependencies(raw))).toMatchObject({
      topology: 'unavailable',
      display: 'unavailable',
      diagnostics: ['missing-topology'],
    });
    raw.snapshot.edges = [];
    const empty = projectDependencies(presentDependencies(raw));
    expect(empty).toMatchObject({
      topology: 'reported',
      display: 'graph',
      edges: [],
      diagnostics: [],
    });
    raw.snapshot.edges = null;
    expect(projectDependencies(presentDependencies(raw)).diagnostics).toEqual(['invalid-topology']);
    expect(projectDependencies(presentDependencies(raw)).groups).toEqual(empty.groups);
  });
  it('separates authored groups, exact attempts and templates without inventing group state', () => {
    const value = observed(),
      group = value.groups.find((g) => g.taskId === 'align')!;
    expect(group.members.map((m) => [m.instanceId, m.attempt])).toEqual([
      ['align', 0],
      ['align:S01', 1],
      ['align:S03', 2],
    ]);
    expect(group.counts).toMatchObject({
      templates: 1,
      unstarted: 0,
      attempted: 2,
      states: [
        { status: 'pending', count: 1 },
        { status: 'completed', count: 1 },
        { status: 'failed', count: 1 },
      ],
    });
    expect('status' in group).toBe(false);
    expect(value.edges).toHaveLength(2);
    expect(value.edges.every((e) => !e.fromTaskId.includes(':'))).toBe(true);
  });
  it('retains endpoint groups with no observed members and preserves unmapped identities', () => {
    const raw = dependencySnapshot();
    raw.snapshot.tasks[0]!.task_id = '';
    raw.snapshot.edges = [{ from: 'external', to: 'align' }];
    const value = projectDependencies(presentDependencies(raw));
    expect(value.unmappedInstances).toEqual(['prepare']);
    expect(value.scope.membershipComplete).toBe(false);
    expect(value.groups.find((g) => g.taskId === 'external')).toMatchObject({
      members: [],
      membership: 'partial-preview',
    });
  });
  it('keeps unknown state/flag facts and never infers a template from its name', () => {
    const raw = dependencySnapshot();
    raw.snapshot.tasks[0]!.status = 'future-state';
    const task: Record<string, unknown> = raw.snapshot.tasks[0]!;
    delete task.template;
    delete task.expanded;
    const group = projectDependencies(presentDependencies(raw)).groups.find(
      (g) => g.taskId === 'prepare',
    )!;
    expect(group.members[0]).toMatchObject({
      status: 'future-state',
      template: null,
      expanded: null,
    });
    expect(group.counts.unknownTemplate).toBe(1);
  });
  it('reports malformed and duplicate edges while retaining only valid directed pairs', () => {
    const raw = dependencySnapshot();
    raw.snapshot.edges = [
      { from: 'prepare', to: 'align' },
      { from: 'prepare', to: 'align' },
      { from: '', to: 'align' },
      false,
    ];
    const read = presentDependencies(raw),
      value = projectDependencies(read);
    expect(read.topology).toMatchObject({
      kind: 'reported',
      invalidEntries: 2,
      duplicateEntries: 1,
    });
    expect(value).toMatchObject({
      display: 'list',
      edges: [{ fromTaskId: 'prepare', toTaskId: 'align' }],
    });
    expect(value.diagnostics).toContain('invalid-topology');
  });
  it('detects self cycles and disconnected cycles without recursion or fabricated scheduling', () => {
    expect(
      dependencyOrder(
        ['a', 'b', 'c'],
        [
          { fromTaskId: 'b', toTaskId: 'c' },
          { fromTaskId: 'c', toTaskId: 'b' },
        ],
      ),
    ).toBeNull();
    const raw = dependencySnapshot();
    raw.snapshot.edges = [{ from: 'align', to: 'align' }];
    expect(projectDependencies(presentDependencies(raw))).toMatchObject({
      display: 'list',
      diagnostics: ['cyclic-topology'],
    });
    expect(dependencyOrder(['a', 'b', 'c'], [{ fromTaskId: 'a', toTaskId: 'c' }])).toEqual([
      'a',
      'b',
      'c',
    ]);
  });
  it('bounds edge inspection, groups and membership while reporting all omitted scopes', () => {
    const raw = dependencySnapshot(),
      member = raw.snapshot.tasks[0]!;
    raw.snapshot.tasks = Array.from({ length: 1200 }, (_, i) => ({
      ...member,
      identity: 'sample-' + i,
      task_id: 'task-' + i,
    }));
    raw.snapshot.edges = Array.from({ length: 1300 }, (_, i) => ({
      from: 'task-' + i,
      to: 'task-' + (i + 1),
    }));
    const read = presentDependencies(raw),
      value = projectDependencies(read);
    expect(value.groups).toHaveLength(DEPENDENCY_LIMITS.groups);
    expect(value.edges.length).toBeLessThanOrEqual(DEPENDENCY_LIMITS.edges);
    expect(value.scope).toMatchObject({
      availableTasks: 1200,
      returnedTasks: 1000,
      membershipComplete: false,
      inspectedEdgeEntries: 1000,
      availableEdgeEntries: 1300,
    });
    expect(value.diagnostics).toEqual(
      expect.arrayContaining([
        'partial-membership',
        'partial-topology',
        'group-limit',
        'edge-limit',
      ]),
    );
    expect(value.display).toBe('list');
  });
  it('rejects duplicate instance identity, inconsistent envelope and forged count contracts', () => {
    const raw = dependencySnapshot();
    raw.snapshot.tasks.push({ ...raw.snapshot.tasks[0]! });
    expect(() => presentDependencies(raw)).toThrow('identities');
    const mismatch = dependencySnapshot();
    mismatch.engineRevision = 'another';
    expect(() => presentDependencies(mismatch)).toThrow('disagree');
    const read = presentDependencies(dependencySnapshot());
    read.run.preview!.returnedTasks = 0;
    expect(() => projectDependencies(read)).toThrow('counts');
    const altered = presentDependencies(dependencySnapshot());
    if (altered.topology.kind === 'reported') altered.topology.inspectedEntries = 0;
    expect(() => projectDependencies(altered)).toThrow('counts');
    expect(() => parse(DependencyReadSchema, { ...read, unexpected: true })).toThrow();
  });
});

describe('exact dependency references and bounded capture', () => {
  it('resolves only returned IDs/direction and refuses labels, old observations and other Runs', () => {
    const value = observed(),
      ref = target(value, edge),
      source = { observation: value, dataRevision: dependencyRevision(value) };
    expect(resolveDependencyTarget(ref, source)).toEqual({ kind: 'exact' });
    expect(
      resolveDependencyTarget(
        { ...ref, selection: { ...edge, fromTaskId: 'align', toTaskId: 'prepare' } },
        source,
      ).kind,
    ).toBe('unavailable');
    expect(
      resolveDependencyTarget(
        target(value, {
          kind: 'run-group',
          coordinateSpace: 'observed-authored-task-group',
          taskId: 'Same label',
        }),
        source,
      ).kind,
    ).toBe('unavailable');
    expect(resolveDependencyTarget({ ...ref, dataRevision: 'old' }, source).kind).toBe(
      'historical',
    );
    expect(
      resolveDependencyTarget({ ...ref, resource: { kind: 'run', runRef: 'run_other' } }, source),
    ).toMatchObject({ reason: 'resource-mismatch' });
    expect(resolveDependencyTarget({ ...ref, projectId: 'prj_other' }, source)).toMatchObject({
      reason: 'resource-mismatch',
    });
    expect(resolveDependencyTarget(ref)).toMatchObject({ reason: 'source-unavailable' });
    expect(() =>
      parse(DependencyTargetSchema, { ...ref, selection: { ...edge, index: 0 } }),
    ).toThrow();
    expect(parse(EvidenceRefSchema, ref)).toEqual(ref); // R3b2 activates local targets; Agent inputs remain frozen.
  });
  it('freezes only the selected group or edge context, with no logs/private paths/canvas state', () => {
    const value = observed(),
      ref = target(value, edge),
      before = structuredClone(value);
    const result = captureDependency(value, ref, 2000);
    expect(result.capture.groups.map((g) => g.taskId)).toEqual(['align', 'prepare']);
    expect(result.capture.edge).toEqual({ fromTaskId: 'prepare', toTaskId: 'align' });
    expect(result.bytes.toString()).not.toMatch(
      /NEVER INCLUDE|private|runtime.sock|viewport|coordinates/,
    );
    expect(result.hash).toBe('sha256:' + createHash('sha256').update(result.bytes).digest('hex'));
    value.groups[0]!.members[0]!.attempt = 99;
    expect(JSON.parse(result.bytes.toString())).toEqual(result.capture);
    expect(result.capture.groups[0]!.members[0]!.attempt).toBe(
      before.groups[0]!.members[0]!.attempt,
    );
    expect(() => captureDependency(value, ref, 2000)).toThrow('does not match');
  });
  it('retains exact counts while bounding group member content and capture byte size', () => {
    const raw = dependencySnapshot(),
      member = raw.snapshot.tasks[2]!;
    raw.snapshot.tasks = Array.from({ length: 1000 }, (_, i) => ({
      ...member,
      identity: 'sample-' + i,
    }));
    const value = projectDependencies(presentDependencies(raw)),
      result = captureDependency(value, target(value), 2000);
    expect(result.capture.groups[0]).toMatchObject({ observedMembers: 1000, truncated: true });
    expect(result.capture.groups[0]!.members).toHaveLength(100);
    expect(result.bytes.length).toBeLessThanOrEqual(65536);
    expect(result.capture.groups[0]!.counts.attempted).toBe(1000);
  });

  it('reads immutable captures and rejects tampered bytes, extra endpoints and inconsistent membership', () => {
    const value = observed(),
      result = captureDependency(value, target(value), 2000);
    expect(readDependencyCaptureBytes(result.bytes, result.hash)).toEqual(result.capture);
    expect(() =>
      readDependencyCaptureBytes(
        Buffer.from(result.bytes.toString().replace('align', 'other')),
        result.hash,
      ),
    ).toThrow('integrity');
    const count = structuredClone(result.capture);
    count.groups[0]!.observedMembers++;
    expect(() => readDependencyCapture(count)).toThrow('metadata');
    const categories = structuredClone(result.capture);
    categories.groups[0]!.counts.templates++;
    categories.groups[0]!.counts.attempted--;
    expect(() => readDependencyCapture(categories)).toThrow('metadata');
    const scope = structuredClone(result.capture);
    scope.groups[0]!.taskId = 'other';
    expect(() => readDependencyCapture(scope)).toThrow('metadata');
    const wrong = structuredClone(result.capture);
    wrong.source.projectId = 'prj_other';
    expect(() => readDependencyCapture(wrong)).toThrow('metadata');
  });
  it('preserves complete identities when byte limits trim members and refuses oversized fixed metadata', () => {
    const raw = dependencySnapshot(),
      member = raw.snapshot.tasks[2]!;
    raw.snapshot.tasks = Array.from({ length: 1000 }, (_, i) => ({
      ...member,
      identity: String(i) + 'x'.repeat(240),
      status: 's'.repeat(400),
    }));
    const value = projectDependencies(presentDependencies(raw)),
      result = captureDependency(value, target(value), 2000);
    expect(result.bytes.length).toBeLessThanOrEqual(65536);
    expect(result.capture.groups[0]!.members.length).toBeLessThan(100);
    expect(result.capture.groups[0]!.members.every((m) => m.instanceId.length >= 241)).toBe(true);
    expect(readDependencyCaptureBytes(result.bytes, result.hash)).toEqual(result.capture);
    raw.snapshot.tasks.forEach((task, i) => (task.status = 's'.repeat(80) + i));
    const excessive = projectDependencies(presentDependencies(raw));
    expect(() => captureDependency(excessive, target(excessive), 2000)).toThrow('metadata exceeds');
  });
  it('ignores observation time for identity but changes revision when topology or attempts change', () => {
    const value = observed(),
      same = structuredClone(value);
    same.source.observedAt++;
    expect(dependencyRevision(same)).toBe(dependencyRevision(value));
    same.edges.pop();
    expect(dependencyRevision(same)).not.toBe(dependencyRevision(value));
    const changed = observed();
    changed.groups[0]!.members[0]!.attempt++;
    expect(dependencyRevision(changed)).not.toBe(dependencyRevision(value));
  });
  it('preserves legacy Run schema and both old revision algorithms after dependency projection', async () => {
    const raw = dependencySnapshot(),
      before = presentRun(raw),
      source = { kind: 'run' as const, value: before };
    const v2 = dataRevision(source),
      v3 = observedDataRevision(source);
    presentDependencies(raw);
    projectDependencies(presentDependencies(raw));
    expect(presentRun(raw)).toEqual(before);
    expect(dataRevision(source)).toBe(v2);
    expect(observedDataRevision(source)).toBe(v3);
    const published = JSON.parse(
      await readFile(new URL('../../contracts/schema/v9.json', import.meta.url), 'utf8'),
    );
    const { RunPresentationSchema } = await import('@gobble/contracts');
    expect(JSON.parse(JSON.stringify(RunPresentationSchema))).toEqual(
      published.definitions.RunPresentationSchema,
    );
  });
});
