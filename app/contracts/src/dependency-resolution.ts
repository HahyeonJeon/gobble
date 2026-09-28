import { ContractValidationError } from './validation-error';
import { parse } from './validation';
import {
  DependencyCaptureSchema,
  type DependencyCapture,
  DependencyTargetSchema,
  DependencyObservationSchema,
  type DependencyTarget,
  type DependencyObservation,
} from './run-dependencies';
import type { ReferenceResolution } from './reference-resolution';

/** No display index, label lookup, reversed pair, wildcard membership or cross-revision remapping. */
export function resolveDependencyTarget(
  target: unknown,
  source?: { observation: DependencyObservation; dataRevision: string },
): ReferenceResolution {
  if (!source) return { kind: 'unavailable', reason: 'source-unavailable' };
  let ref: DependencyTarget, observation: DependencyObservation;
  try {
    ref = parse(DependencyTargetSchema, target);
    observation = parse(DependencyObservationSchema, source.observation);
  } catch {
    return { kind: 'unavailable', reason: 'selection-unavailable' };
  }
  if (
    ref.projectId !== observation.source.projectId ||
    ref.resource.runRef !== observation.source.runRef
  )
    return { kind: 'unavailable', reason: 'resource-mismatch' };
  if (ref.dataRevision !== source.dataRevision)
    return { kind: 'historical', currentRevision: source.dataRevision };
  const selection = ref.selection;
  const present =
    selection.kind === 'run-group'
      ? observation.groups.some((g) => g.taskId === selection.taskId)
      : observation.topology === 'reported' &&
        observation.edges.some(
          (e) => e.fromTaskId === selection.fromTaskId && e.toTaskId === selection.toTaskId,
        );
  return present ? { kind: 'exact' } : { kind: 'unavailable', reason: 'selection-unavailable' };
}

/** Closed archive reader, including relationships that JSON Schema cannot express. */
export function readDependencyCapture(input: unknown): DependencyCapture {
  const capture = parse(DependencyCaptureSchema, input),
    { target, source } = capture;
  const invalid = () => {
    throw new ContractValidationError('Dependency capture metadata does not match its target.');
  };
  if (target.projectId !== source.projectId || target.resource.runRef !== source.runRef) invalid();
  const selected = target.selection;
  const expected = new Set(
    selected.kind === 'run-group' ? [selected.taskId] : [selected.fromTaskId, selected.toTaskId],
  );
  if (
    selected.kind === 'run-group'
      ? capture.edge !== null
      : capture.edge?.fromTaskId !== selected.fromTaskId ||
        capture.edge?.toTaskId !== selected.toTaskId
  )
    invalid();
  const groups = new Set<string>(),
    instances = new Set<string>();
  for (const group of capture.groups) {
    if (
      !expected.has(group.taskId) ||
      groups.has(group.taskId) ||
      group.observedMembers < group.members.length ||
      group.truncated !== group.observedMembers > group.members.length
    )
      invalid();
    groups.add(group.taskId);
    const counts = group.counts;
    if (
      counts.templates + counts.unstarted + counts.attempted !== group.observedMembers ||
      counts.unknownTemplate > group.observedMembers ||
      counts.states.reduce((sum, state) => sum + state.count, 0) !== group.observedMembers ||
      new Set(counts.states.map((state) => state.status)).size !== counts.states.length
    )
      invalid();
    const omitted = group.observedMembers - group.members.length;
    const categories = [
      [counts.templates, group.members.filter((m) => m.template === true).length],
      [
        counts.unstarted,
        group.members.filter((m) => m.template !== true && m.attempt === 0).length,
      ],
      [counts.attempted, group.members.filter((m) => m.template !== true && m.attempt > 0).length],
      [counts.unknownTemplate, group.members.filter((m) => m.template === null).length],
    ] as const;
    if (categories.some(([total, returned]) => total < returned || total - returned > omitted))
      invalid();
    for (const member of group.members) {
      if (instances.has(member.instanceId)) invalid();
      instances.add(member.instanceId);
    }
    for (const state of counts.states)
      if (group.members.filter((member) => member.status === state.status).length > state.count)
        invalid();
    if (
      group.members.some((member) => !counts.states.some((state) => state.status === member.status))
    )
      invalid();
  }
  if (groups.size !== expected.size || capture.scope.observedGroups < groups.size) invalid();
  return capture;
}
