import {
  defaultDependencyNavigation,
  containsObservedTarget,
  matchingDependencies,
  matchingDependencyGroups,
  type DependencyObservation,
  type DependencyTarget,
  type Surface,
} from '@gobble/contracts';
import { dependencyRevision } from '../evidence/dependency-capture';
import { AppProblem } from '../problem';

/** Returns whole addresses and bounded context. Only returned entries authorize a later pointer. */
export function dependencyObservation(
  value: DependencyObservation,
  surface: Surface,
  scope: 'view' | 'source-preview',
  selection?: DependencyTarget['selection'],
) {
  const navigation =
    surface.view === 'run'
      ? (surface.runView?.dependencies ?? defaultDependencyNavigation())
      : defaultDependencyNavigation();
  const filtered =
    scope === 'view' && (navigation.representation === 'list' || value.display !== 'graph');
  const visibleGroups = filtered ? matchingDependencyGroups(value, navigation.query) : value.groups;
  let groups = visibleGroups;
  let edges = filtered ? matchingDependencies(value, navigation.query) : value.edges;
  if (selection?.kind === 'run-group') {
    if (!groups.some((group) => group.taskId === selection.taskId))
      throw new AppProblem('invalid_request', 'This group is outside the observed scope.');
    groups = groups.filter((group) => group.taskId === selection.taskId);
    edges = [];
  } else if (selection?.kind === 'run-dependency') {
    edges = edges.filter(
      (edge) => edge.fromTaskId === selection.fromTaskId && edge.toTaskId === selection.toTaskId,
    );
    if (!edges.length)
      throw new AppProblem(
        'invalid_request',
        'This directed dependency is outside the observed scope.',
      );
    // An edge includes both endpoint contexts, even when only one endpoint matched list search.
    groups = value.groups.filter(
      (group) => group.taskId === selection.fromTaskId || group.taskId === selection.toTaskId,
    );
  }
  if (
    selection?.kind === 'run-dependency' &&
    groups.length !== new Set([selection.fromTaskId, selection.toTaskId]).size
  )
    throw new AppProblem('invalid_request', 'Dependency endpoint context is unavailable.');
  const revision = dependencyRevision(value);
  const target = (selected: DependencyTarget['selection']): DependencyTarget => ({
    schemaVersion: 4,
    projectId: value.source.projectId,
    resource: { kind: 'run', runRef: value.source.runRef },
    dataRevision: revision,
    selection: selected,
  });
  const content = {
    kind: 'run-dependencies' as const,
    source: value.source,
    dataRevision: revision,
    topology: value.topology,
    diagnostics: value.diagnostics,
    scope,
    observationScope: value.scope,
    requestedGroups: groups.length,
    requestedEdges: edges.length,
    groups: [] as Array<{
      taskId: string;
      membership: DependencyObservation['groups'][number]['membership'];
      counts: DependencyObservation['groups'][number]['counts'];
      observedMembers: number;
      members: DependencyObservation['groups'][number]['members'];
      truncated: boolean;
      target: DependencyTarget;
    }>,
    edges: [] as Array<{ fromTaskId: string; toTaskId: string; target: DependencyTarget }>,
    returnedGroups: 0,
    returnedEdges: 0,
    truncated: false,
  };
  const fits = () => Buffer.byteLength(JSON.stringify(content)) <= 48 * 1024;
  for (const group of groups) {
    const entry = {
      taskId: group.taskId,
      membership: group.membership,
      counts: group.counts,
      observedMembers: group.members.length,
      members: group.members.slice(0, 20),
      truncated: group.members.length > 20,
      target: target({
        kind: 'run-group',
        coordinateSpace: 'observed-authored-task-group',
        taskId: group.taskId,
      }),
    };
    content.groups.push(entry);
    while (!fits() && entry.members.length) {
      entry.members.pop();
      entry.truncated = true;
    }
    if (!fits()) {
      content.groups.pop();
      break;
    }
  }
  for (const edge of edges) {
    // Endpoints must be in this observation, not inferred from a name or a previous read.
    if (
      ![edge.fromTaskId, edge.toTaskId].every((id) =>
        value.groups.some((group) => group.taskId === id),
      )
    )
      continue;
    content.edges.push({
      ...edge,
      target: target({
        kind: 'run-dependency',
        coordinateSpace: 'observed-authored-task-pair',
        ...edge,
      }),
    });
    if (!fits()) {
      content.edges.pop();
      break;
    }
  }
  content.returnedGroups = content.groups.length;
  content.returnedEdges = content.edges.length;
  content.truncated =
    content.groups.length < groups.length ||
    content.edges.length < edges.length ||
    content.groups.some((group) => group.truncated);
  const targets = [
    ...content.groups
      .filter((group) => visibleGroups.some((visible) => visible.taskId === group.taskId))
      .map((group) => group.target),
    ...content.edges.map((edge) => edge.target),
  ];
  const evidence = selection ? target(selection) : undefined;
  if (selection?.kind === 'run-dependency' && content.groups.length !== groups.length)
    throw new AppProblem(
      'unsupported',
      'Dependency endpoint context exceeds the observation size limit.',
    );
  if (evidence && !targets.some((item) => containsObservedTarget(item, evidence)))
    throw new AppProblem('unsupported', 'This target exceeds the observation size limit.');
  return { content, targets, evidence };
}
