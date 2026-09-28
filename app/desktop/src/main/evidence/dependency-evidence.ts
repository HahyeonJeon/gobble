import {
  EVIDENCE_TEXT_BYTES,
  validateManifest,
  type DraftAttachment,
  type SurfaceData,
} from '@gobble/contracts';
import { captureDependency } from './dependency-capture';
import { contentHash, evidencePreview, type EvidenceAsset } from './materialize';
import { AppProblem } from '../problem';

/** Adapts dependency capture to the existing publication, quota, preview and delivery pipeline. */
export function materializeDependencyEvidence(
  attachment: DraftAttachment,
  data: SurfaceData,
): EvidenceAsset {
  if (
    attachment.evidence.schemaVersion !== 4 ||
    data.kind !== 'run' ||
    !data.dependencies ||
    attachment.capture ||
    attachment.presentation
  )
    throw new AppProblem('invalid_request', 'A current dependency observation is required.');
  const capturedAt = Date.now();
  const result = captureDependency(data.dependencies, attachment.evidence, capturedAt);
  const bytes = Buffer.from(JSON.stringify({ kind: 'run', dependency: result.capture }));
  if (bytes.length > EVIDENCE_TEXT_BYTES)
    throw new AppProblem(
      'unsupported',
      'This dependency capture exceeds the message evidence limit.',
    );
  const capture: NonNullable<DraftAttachment['capture']> = {
    kind: 'observed',
    capturedAt,
    asset: { hash: contentHash(bytes), byteLength: bytes.length, mediaType: 'application/json' },
    representation: {
      kind: 'run',
      truncated:
        result.capture.groups.some((g) => g.truncated || g.membership === 'partial-preview') ||
        result.capture.diagnostics.length > 0,
    },
  };
  const { kind: _kind, ...saved } = capture;
  const asset: EvidenceAsset = { manifest: { ...attachment, capture, ...saved }, bytes };
  validateManifest(asset.manifest);
  evidencePreview(asset);
  return asset;
}
