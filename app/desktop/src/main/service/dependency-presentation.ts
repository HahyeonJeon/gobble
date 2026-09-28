import {
  parse,
  RunSnapshotSchema,
  DependencyReadSchema,
  DEPENDENCY_LIMITS,
  dependencyPairKey,
  type DependencyRead,
  type DependencyTopology,
  type TaskDependency,
} from '@gobble/contracts';
import { AppProblem } from '../problem';
import { presentRun } from './run-presentation';

const record = (value: unknown): value is Record<string, unknown> =>
  value !== null && typeof value === 'object' && !Array.isArray(value);
const id = (value: unknown): value is string =>
  typeof value === 'string' && value.length > 0 && value.length <= 256;
function topology(value: unknown): DependencyTopology {
  if (value === undefined) return { kind: 'unavailable', reason: 'missing' };
  if (!Array.isArray(value)) return { kind: 'unavailable', reason: 'invalid' };
  const edges: TaskDependency[] = [],
    seen = new Set<string>();
  let invalidEntries = 0,
    duplicateEntries = 0;
  const inspectedEntries = Math.min(value.length, DEPENDENCY_LIMITS.inspectedEdges);
  for (const item of value.slice(0, inspectedEntries)) {
    if (!record(item) || !id(item.from) || !id(item.to)) {
      invalidEntries++;
      continue;
    }
    const edge = { fromTaskId: item.from, toTaskId: item.to },
      key = dependencyPairKey(edge);
    if (seen.has(key)) {
      duplicateEntries++;
      continue;
    }
    seen.add(key);
    edges.push(edge);
  }
  return {
    kind: 'reported',
    availableEntries: value.length,
    inspectedEntries,
    invalidEntries,
    duplicateEntries,
    truncated: inspectedEntries < value.length,
    edges,
  };
}
/** Same coherent monitor response, no second Plan read, no writes and no changes to legacy hashing. */
export function presentDependencies(input: unknown): DependencyRead {
  const value = parse(RunSnapshotSchema, input);
  if (value.engineRevision !== value.snapshot.snapshot)
    throw new AppProblem(
      'incompatible_runtime',
      'The monitor envelope and snapshot revisions disagree.',
    );
  const raw: unknown = value.snapshot;
  const observedTopology = topology(record(raw) ? raw.edges : undefined);
  // The old presentation keeps its exact algorithm. This independent reader owns availability.
  const snapshot = { ...value.snapshot, edges: [] };
  const { dependencies: _dependencies, ...run } = presentRun({ ...value, snapshot });
  const result = parse(DependencyReadSchema, { schemaVersion: 1, run, topology: observedTopology });
  if (Buffer.byteLength(JSON.stringify(result)) > 1024 * 1024)
    throw new AppProblem(
      'unsupported',
      'This dependency observation exceeds the view memory limit.',
    );
  return result;
}
