import { parse } from './validation';
import { ContractValidationError } from './validation-error';
import {
  DependencyReadSchema,
  DependencyObservationSchema,
  DEPENDENCY_LIMITS,
  type DependencyGroup,
  type DependencyObservation,
  type TaskDependency,
} from './run-dependencies';

export const dependencyPairKey = (edge: TaskDependency): string =>
  JSON.stringify([edge.fromTaskId, edge.toTaskId]);
const validId = (id: string | null): id is string =>
  id !== null && id.length > 0 && id.length <= 256;

/** O(V+E), iterative: safe for long/cyclic input; reports topology, never an execution state. */
export function dependencyOrder(
  ids: readonly string[],
  edges: readonly TaskDependency[],
): string[] | null {
  const degree = new Map(ids.map((id) => [id, 0])),
    next = new Map(ids.map((id) => [id, [] as string[]]));
  for (const edge of edges) {
    if (!degree.has(edge.fromTaskId) || !degree.has(edge.toTaskId)) return null;
    degree.set(edge.toTaskId, degree.get(edge.toTaskId)! + 1);
    next.get(edge.fromTaskId)!.push(edge.toTaskId);
  }
  const queue = ids.filter((id) => degree.get(id) === 0),
    ordered: string[] = [];
  for (let index = 0; index < queue.length; index++) {
    const id = queue[index]!;
    ordered.push(id);
    for (const to of next.get(id)!) {
      const remaining = degree.get(to)! - 1;
      degree.set(to, remaining);
      if (remaining === 0) queue.push(to);
    }
  }
  return ordered.length === ids.length ? ordered : null;
}

/** Bounded presentation only. Missing preview members never become absent Run instances. */
export function projectDependencies(input: unknown): DependencyObservation {
  const read = parse(DependencyReadSchema, input),
    { run, topology } = read;
  if (
    run.tasks.length > 1000 ||
    (run.preview &&
      (run.preview.returnedTasks !== run.tasks.length ||
        run.preview.availableTasks < run.tasks.length ||
        run.preview.truncated !== run.preview.availableTasks > run.tasks.length))
  )
    throw new ContractValidationError('Task preview counts are inconsistent.');
  let complete = !!run.preview && !run.preview.truncated;
  const diagnostics: DependencyObservation['diagnostics'] = [];
  const groups = new Map<string, DependencyGroup>(),
    identities = new Set<string>(),
    unmappedInstances: string[] = [];
  const group = (taskId: string) => {
    let value = groups.get(taskId);
    if (!value) {
      value = {
        taskId,
        members: [],
        membership: complete ? 'complete-preview' : 'partial-preview',
        counts: { templates: 0, unstarted: 0, attempted: 0, unknownTemplate: 0, states: [] },
      };
      groups.set(taskId, value);
    }
    return value;
  };
  for (const task of run.tasks) {
    if (
      !validId(task.instanceId) ||
      identities.has(task.instanceId) ||
      !Number.isSafeInteger(task.attempt) ||
      task.attempt < 0
    )
      throw new ContractValidationError('Invalid or duplicate instance identity.');
    identities.add(task.instanceId);
    if (!validId(task.taskId)) {
      unmappedInstances.push(task.instanceId);
      continue;
    }
    const value = group(task.taskId),
      template = task.template ?? null;
    value.members.push({
      instanceId: task.instanceId,
      attempt: task.attempt,
      status: task.status,
      template,
      expanded: task.expanded ?? null,
    });
    if (template === true) value.counts.templates++;
    else if (task.attempt === 0) value.counts.unstarted++;
    else value.counts.attempted++;
    if (template === null) value.counts.unknownTemplate++;
    const state = value.counts.states.find((s) => s.status === task.status);
    if (state) state.count++;
    else value.counts.states.push({ status: task.status, count: 1 });
  }
  let edges: TaskDependency[] = [];
  if (topology.kind === 'unavailable')
    diagnostics.push(topology.reason === 'missing' ? 'missing-topology' : 'invalid-topology');
  else {
    if (
      topology.inspectedEntries > topology.availableEntries ||
      topology.inspectedEntries > DEPENDENCY_LIMITS.inspectedEdges ||
      topology.truncated !== topology.inspectedEntries < topology.availableEntries ||
      topology.edges.length + topology.invalidEntries + topology.duplicateEntries !==
        topology.inspectedEntries
    )
      throw new ContractValidationError('Dependency entry counts are inconsistent.');
    const seen = new Set<string>();
    for (const edge of topology.edges) {
      const key = dependencyPairKey(edge);
      if (seen.has(key)) throw new ContractValidationError('Duplicate projected dependency.');
      seen.add(key);
      group(edge.fromTaskId);
      group(edge.toTaskId);
    }
    edges = topology.edges;
    if (topology.invalidEntries) diagnostics.push('invalid-topology');
    if (topology.duplicateEntries) diagnostics.push('duplicate-edges');
    if (topology.truncated) diagnostics.push('partial-topology');
    if (dependencyOrder([...groups.keys()], edges) === null) diagnostics.push('cyclic-topology');
  }
  if (unmappedInstances.length) complete = false;
  for (const value of groups.values())
    value.membership = complete ? 'complete-preview' : 'partial-preview';
  if (!complete) diagnostics.push('partial-membership');
  if (unmappedInstances.length) diagnostics.push('unmapped-members');
  if (groups.size > DEPENDENCY_LIMITS.groups) diagnostics.push('group-limit');
  if (edges.length > DEPENDENCY_LIMITS.edges) diagnostics.push('edge-limit');
  const returned = [...groups.values()]
    .sort((a, b) => (a.taskId < b.taskId ? -1 : a.taskId > b.taskId ? 1 : 0))
    .slice(0, DEPENDENCY_LIMITS.groups);
  const included = new Set(returned.map((g) => g.taskId));
  const returnedEdges = edges
    .filter((e) => included.has(e.fromTaskId) && included.has(e.toTaskId))
    .slice(0, DEPENDENCY_LIMITS.edges);
  const fallback = diagnostics.some((d) =>
    [
      'invalid-topology',
      'partial-topology',
      'cyclic-topology',
      'group-limit',
      'edge-limit',
    ].includes(d),
  );
  return parse(DependencyObservationSchema, {
    schemaVersion: 1,
    source: {
      projectId: run.projectId,
      runRef: run.runRef,
      engineRevision: run.engineRevision,
      observedAt: run.observedAt,
      imageId: run.imageId,
      runId: run.runId,
    },
    topology: topology.kind,
    display: topology.kind === 'unavailable' ? 'unavailable' : fallback ? 'list' : 'graph',
    diagnostics,
    scope: {
      availableTasks: run.preview?.availableTasks ?? null,
      returnedTasks: run.tasks.length,
      membershipComplete: complete,
      observedGroups: groups.size,
      observedEdges: edges.length,
      availableEdgeEntries: topology.kind === 'reported' ? topology.availableEntries : null,
      inspectedEdgeEntries: topology.kind === 'reported' ? topology.inspectedEntries : 0,
      invalidEdgeEntries: topology.kind === 'reported' ? topology.invalidEntries : 0,
      duplicateEdgeEntries: topology.kind === 'reported' ? topology.duplicateEntries : 0,
    },
    groups: returned,
    edges: returnedEdges,
    unmappedInstances,
  });
}
