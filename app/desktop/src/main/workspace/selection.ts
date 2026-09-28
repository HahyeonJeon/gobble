import { dependencyRevision } from '../evidence/dependency-capture';
import {
  type EvidenceRef,
  type SurfaceData,
  resolveReferenceTarget,
  logPreviewText,
} from '@gobble/contracts';
import { createHash } from 'node:crypto';
import { AppProblem } from '../problem';
import { presentLog } from '../service/log-presentation';

/** Versioned semantic content, excluding observation time. No native paths or live log cursor. */
export function observedDataRevision(data: SurfaceData): string {
  if (data.kind === 'report') return data.saved.asset.hash;
  if (data.kind === 'pipeline')
    return 'sha256:' + createHash('sha256').update(JSON.stringify(data.value)).digest('hex');
  if (data.kind === 'file')
    throw new AppProblem('unsupported', 'A Run or log observation is required.');
  const value =
    data.kind === 'log' && !('schemaVersion' in data.value) ? presentLog(data.value) : data.value;
  const { observedAt: _observedAt, ...facts } = value;
  return (
    'sha256:' +
    createHash('sha256')
      .update(JSON.stringify({ schemaVersion: 1, kind: data.kind, facts }))
      .digest('hex')
  );
}

export const dataRevision = (data: SurfaceData): string =>
  data.kind === 'report'
    ? data.saved.asset.hash
    : data.kind === 'pipeline'
      ? observedDataRevision(data)
      : data.kind === 'file'
        ? data.value.revision
        : data.kind === 'log'
          ? 'sha256:' + createHash('sha256').update(logPreviewText(data.value)).digest('hex')
          : data.value.engineRevision;

export function checkSelection(evidence: EvidenceRef, data: SurfaceData): void {
  const resolution = resolveReferenceTarget(evidence, {
    projectId: evidence.projectId,
    resource: evidence.resource,
    dataRevision:
      evidence.schemaVersion === 4
        ? data.kind === 'run' && data.dependencies
          ? dependencyRevision(data.dependencies)
          : 'unavailable'
        : evidence.schemaVersion === 3
          ? observedDataRevision(data)
          : dataRevision(data),
    data,
  });
  if (resolution.kind !== 'exact')
    throw new AppProblem(
      'stale_revision',
      'This selection no longer matches the displayed data. Refresh the view and select again.',
    );
}
