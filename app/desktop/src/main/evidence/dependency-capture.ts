import { createHash } from 'node:crypto';
import {
  parse,
  DependencyCaptureSchema,
  DependencyObservationSchema,
  DependencyTargetSchema,
  readDependencyCapture,
  resolveDependencyTarget,
  DEPENDENCY_LIMITS,
  type DependencyCapture,
  type DependencyObservation,
} from '@gobble/contracts';
import { AppProblem } from '../problem';

/** Dedicated versioned hash; old Run/task/log hash algorithms and captured bytes remain unchanged. */
export function dependencyRevision(input: DependencyObservation): string {
  const observation = parse(DependencyObservationSchema, input);
  const { observedAt: _time, ...source } = observation.source;
  return (
    'sha256:' +
    createHash('sha256')
      .update(
        JSON.stringify({
          schemaVersion: 1,
          kind: 'run-dependencies',
          facts: { ...observation, source },
        }),
      )
      .digest('hex')
  );
}

/** Bounded data-only capture, published through the existing evidence store. */
export function captureDependency(
  observation: DependencyObservation,
  target: unknown,
  capturedAt: number,
): {
  capture: DependencyCapture;
  bytes: Buffer;
  hash: string;
} {
  const ref = parse(DependencyTargetSchema, target);
  if (
    resolveDependencyTarget(ref, { observation, dataRevision: dependencyRevision(observation) })
      .kind !== 'exact'
  )
    throw new AppProblem(
      'stale_revision',
      'The dependency target does not match this observation.',
    );
  const selection = ref.selection;
  const taskIds =
    selection.kind === 'run-group'
      ? [selection.taskId]
      : [selection.fromTaskId, selection.toTaskId];
  const groups = observation.groups
    .filter((group) => taskIds.includes(group.taskId))
    .map((group) => ({
      taskId: group.taskId,
      membership: group.membership,
      counts: structuredClone(group.counts),
      observedMembers: group.members.length,
      members: structuredClone(group.members.slice(0, DEPENDENCY_LIMITS.captureMembers)),
      truncated: group.members.length > DEPENDENCY_LIMITS.captureMembers,
    }));
  if (groups.length !== new Set(taskIds).size)
    throw new AppProblem('stale_revision', 'An endpoint is outside the returned group scope.');
  const capture = parse(DependencyCaptureSchema, {
    schemaVersion: 1,
    capturedAt,
    target: structuredClone(ref),
    source: structuredClone(observation.source),
    scope: structuredClone(observation.scope),
    diagnostics: [...observation.diagnostics],
    groups,
    edge:
      selection.kind === 'run-dependency'
        ? { fromTaskId: selection.fromTaskId, toTaskId: selection.toTaskId }
        : null,
  });
  const encode = () => Buffer.from(JSON.stringify(capture), 'utf8');
  let bytes = encode();
  // Preserve IDs, counts and provenance. Remove whole member records; never truncate identifiers/text.
  while (
    bytes.length > DEPENDENCY_LIMITS.captureBytes &&
    groups.some((group) => group.members.length)
  ) {
    const largest = groups.reduce((a, b) => (a.members.length >= b.members.length ? a : b));
    largest.members.pop();
    largest.truncated = true;
    bytes = encode();
  }
  if (bytes.length > DEPENDENCY_LIMITS.captureBytes)
    throw new AppProblem('unsupported', 'Dependency metadata exceeds the capture size limit.');
  readDependencyCapture(capture);
  return { capture, bytes, hash: 'sha256:' + createHash('sha256').update(bytes).digest('hex') };
}

/** Stored bytes need both their expected content hash and the versioned semantic reader. */
export function readDependencyCaptureBytes(bytes: Buffer, expectedHash: string): DependencyCapture {
  if (
    bytes.length > DEPENDENCY_LIMITS.captureBytes ||
    'sha256:' + createHash('sha256').update(bytes).digest('hex') !== expectedHash
  )
    throw new AppProblem('invalid_request', 'Dependency capture integrity check failed.');
  return readDependencyCapture(JSON.parse(bytes.toString('utf8')));
}
