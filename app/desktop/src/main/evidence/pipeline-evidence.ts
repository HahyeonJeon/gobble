import {
  pipelineSubject,
  pipelineSubjectLabel,
  validateManifest,
  type DraftAttachment,
  type SurfaceData,
  type PipelineCapture,
} from '@gobble/contracts';
import { checkSelection } from '../workspace/selection';
import { AppProblem } from '../problem';
import { contentHash, evidencePreview, type EvidenceAsset } from './materialize';

/** Snapshot the acknowledged artifact's selected facts, never renderer text or re-read source. */
export function materializePipelineEvidence(
  attachment: DraftAttachment,
  data: SurfaceData,
): EvidenceAsset {
  if (
    attachment.evidence.schemaVersion !== 7 ||
    data.kind !== 'pipeline' ||
    !data.value.artifact ||
    attachment.capture ||
    attachment.presentation
  )
    throw new AppProblem('invalid_request', 'A checked pipeline selection is required.');
  checkSelection(attachment.evidence, data);
  const capturedAt = Date.now();
  const subject = pipelineSubject(data.value.artifact.flow, attachment.evidence.selection.subject);
  const pipeline: PipelineCapture = {
    schemaVersion: 1,
    capturedAt,
    target: structuredClone(attachment.evidence),
    pipelineName: data.value.artifact.flow.name,
    checkedAt: data.value.artifact.checkedAt,
    subject: structuredClone(subject),
  };
  const bytes = Buffer.from(JSON.stringify({ kind: 'pipeline', pipeline }));
  if (bytes.length > 65536)
    throw new AppProblem(
      'unsupported',
      'This step exceeds the attachment limit. Choose a port or setting.',
    );
  const capture = {
    kind: 'pipeline' as const,
    capturedAt,
    asset: {
      hash: contentHash(bytes),
      byteLength: bytes.length,
      mediaType: 'application/json' as const,
    },
    representation: { kind: 'pipeline' as const, truncated: false as const },
  };
  const { kind: _kind, ...saved } = capture;
  const asset: EvidenceAsset = {
    manifest: {
      ...attachment,
      label: pipelineSubjectLabel(subject).slice(0, 512),
      capture,
      ...saved,
    },
    bytes,
  };
  validateManifest(asset.manifest);
  evidencePreview(asset);
  return asset;
}
