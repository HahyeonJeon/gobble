import { reportAsset } from './report';
import { materializePipelineEvidence } from './pipeline-evidence';
import { materializeDependencyEvidence } from './dependency-evidence';
import {
  EVIDENCE_TEXT_BYTES,
  EvidenceTextContentSchema,
  parse,
  draftCaptureManifest,
  observedEvidenceText,
  textRange,
  logPreviewText,
  type DraftAttachment,
  type EvidenceManifest,
  type ObservedEvidence,
  type ObservedEvidenceV2,
  type SurfaceData,
} from '@gobble/contracts';
import { checkSelection } from '../workspace/selection';
import { presentLog } from '../service/log-presentation';
import { AppProblem } from '../problem';
import { contentHash, evidencePreview, type EvidenceAsset } from './materialize';
import type { EvidenceStorage } from './storage';

export function capturedManifest(attachment: DraftAttachment): EvidenceManifest {
  const manifest = draftCaptureManifest(attachment);
  if (!manifest) throw new AppProblem('invalid_request', 'A captured observation is required.');
  return manifest;
}

/** Materialize exactly the retained Main observation, never text supplied by the renderer. */
export function materializeObserved(attachment: DraftAttachment, data: SurfaceData): EvidenceAsset {
  if (attachment.evidence.schemaVersion === 8) return reportAsset(attachment, data);
  if (attachment.evidence.schemaVersion === 7) return materializePipelineEvidence(attachment, data);
  if (attachment.evidence.schemaVersion === 4)
    return materializeDependencyEvidence(attachment, data);
  if (
    attachment.evidence.schemaVersion === 5 ||
    attachment.evidence.schemaVersion === 6 ||
    data.kind === 'report' ||
    data.kind === 'file' ||
    data.kind === 'pipeline' ||
    attachment.capture ||
    attachment.presentation
  )
    throw new AppProblem('unsupported', 'Only Run and log observations use this capture path.');
  checkSelection(attachment.evidence, data);
  let source: ObservedEvidence['source'] | ObservedEvidenceV2['source'];
  if (data.kind === 'run') {
    const value = structuredClone(data.value);
    const observedTaskCount = value.tasks.length;
    const selection = attachment.evidence.selection;
    if (selection?.kind === 'run-task') {
      value.tasks = value.tasks.filter(
        (task) => task.instanceId === selection.instanceId && task.attempt === selection.attempt,
      );
      const availableTasks = value.preview?.availableTasks ?? observedTaskCount;
      value.preview = {
        availableTasks,
        returnedTasks: value.tasks.length,
        truncated: value.tasks.length < availableTasks,
      };
      value.dependencies = [];
    }
    source = { kind: 'run', value, observedTaskCount };
  } else {
    const value =
      'schemaVersion' in data.value ? structuredClone(data.value) : presentLog(data.value);
    const selection = attachment.evidence.selection;
    const selectedStream = selection?.kind === 'log-text' ? selection.stream : null;
    const text = selectedStream ? value.streams[selectedStream].text : logPreviewText(value);
    const selectedText =
      selection?.kind === 'text' || selection?.kind === 'log-text'
        ? textRange(text, selection)
        : text;
    const { text: _stdout, ...stdout } = value.streams.stdout;
    const { text: _stderr, ...stderr } = value.streams.stderr;
    source = {
      kind: 'log',
      value: { ...value, streams: { stdout, stderr } },
      selectedStream,
      selectedText,
    };
  }
  const capturedAt = Date.now();
  const observation = {
    schemaVersion: source.kind === 'run' && 'continuation' in source.value ? 2 : 1,
    capturedAt,
    target: structuredClone(attachment.evidence),
    source,
  };
  const content = parse(EvidenceTextContentSchema, {
    kind: source.kind,
    text: observedEvidenceText(observation),
    observation,
  });
  const bytes = Buffer.from(JSON.stringify(content));
  if (bytes.length > EVIDENCE_TEXT_BYTES)
    throw new AppProblem(
      'unsupported',
      'This observation exceeds 64 KiB. Select one task or a smaller excerpt.',
    );
  const truncated =
    source.kind === 'log'
      ? Object.values(source.value.streams).some((stream) => stream.earlierBytesOmitted)
      : (source.value.preview?.truncated ?? false);
  const capture: NonNullable<DraftAttachment['capture']> = {
    kind: 'observed',
    capturedAt,
    asset: { hash: contentHash(bytes), byteLength: bytes.length, mediaType: 'application/json' },
    representation: { kind: source.kind, truncated },
  };
  const asset = { manifest: capturedManifest({ ...attachment, capture }), bytes };
  evidencePreview(asset);
  return asset;
}

/** I/O port called only inside the Workspace writer; storage owns immutable bytes. */
export class EvidenceCapture {
  constructor(
    private readonly storage: EvidenceStorage,
    private readonly pdfCapture?: (
      attachment: DraftAttachment,
      data: SurfaceData,
    ) => Promise<EvidenceAsset>,
    private readonly notebookCapture?: (
      attachment: DraftAttachment,
      data: SurfaceData,
    ) => Promise<EvidenceAsset>,
  ) {}
  async capture(attachment: DraftAttachment, data: SurfaceData): Promise<DraftAttachment> {
    if (attachment.evidence.schemaVersion === 5 && !this.pdfCapture)
      throw new AppProblem('unsupported', 'PDF capture is unavailable.');
    if (attachment.evidence.schemaVersion === 6 && !this.notebookCapture)
      throw new AppProblem('unsupported', 'Notebook capture is unavailable.');
    const asset =
      attachment.evidence.schemaVersion === 6
        ? await this.notebookCapture!(attachment, data)
        : attachment.evidence.schemaVersion === 5
          ? await this.pdfCapture!(attachment, data)
          : materializeObserved(attachment, data);
    await this.storage.putCaptured(attachment.evidence.projectId, asset);
    const capture = asset.manifest.capture;
    if (!capture) throw new AppProblem('internal', 'The observation capture is missing.');
    return { ...attachment, label: asset.manifest.label, capture };
  }
  reclaim(projectId: string, retained: Set<string>): Promise<void> {
    return this.storage.reclaimCaptured(projectId, retained);
  }
}
