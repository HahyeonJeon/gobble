import {
  reportReading,
  validateManifest,
  type DraftAttachment,
  type SurfaceData,
} from '@gobble/contracts';
import type { EvidenceAsset } from './materialize';
import { checkSelection } from '../workspace/selection';
import { createHash } from 'node:crypto';
import {
  parse,
  RunReportSourceSchema,
  SavedReportSchema,
  SavedReportRecordSchema,
  SurfaceDataSchema,
  REPORT_BYTES,
  REPORT_TEXT_BYTES,
  type RunReportSource,
  type SavedReport,
  type SavedReportRecord,
  type FastqcContent,
} from '@gobble/contracts';
import { AppProblem } from '../problem';
import type { EvidenceStorage } from './storage';

const digest = (bytes: Buffer) => 'sha256:' + createHash('sha256').update(bytes).digest('hex');
/** Retains a verified rendition; runtime/source resolution belongs to ProjectService. */
export class ReportEvidence {
  constructor(
    private readonly storage: EvidenceStorage,
    private readonly decode: (source: Buffer) => Promise<FastqcContent>,
  ) {}
  async capture(input: RunReportSource): Promise<SavedReportRecord> {
    const source = parse(RunReportSourceSchema, input);
    const bytes = Buffer.from(source.base64, 'base64');
    if (
      bytes.toString('base64') !== source.base64 ||
      bytes.length !== source.evidence.size ||
      digest(bytes) !== source.evidence.sha256
    )
      throw new AppProblem(
        'stale_revision',
        'The quality report does not match its recorded output.',
      );
    const content = await this.decode(bytes);
    const { path: _path, ...producer } = source.evidence;
    const report = parse(SavedReportSchema, {
      schemaVersion: 1,
      projectId: source.projectId,
      runRef: source.runRef,
      producer,
      capturedAt: Date.now(),
      content,
    });
    const payload = Buffer.from(JSON.stringify(report));
    const saved = parse(SavedReportRecordSchema, {
      projectId: report.projectId,
      runRef: report.runRef,
      producer,
      capturedAt: report.capturedAt,
      title: content.title,
      profile: content.profile,
      asset: { hash: digest(payload), byteLength: payload.length },
    });
    validateSavedReport(report, saved);
    await this.storage.putCaptured(source.projectId, { manifest: saved, bytes: payload });
    return saved;
  }
  async read(
    projectId: string,
    input: SavedReportRecord,
  ): Promise<Extract<import('@gobble/contracts').SurfaceData, { kind: 'report' }>> {
    const saved = parse(SavedReportRecordSchema, input);
    if (saved.projectId !== projectId)
      throw new AppProblem('forbidden', 'This report belongs to another Project.');
    const { bytes } = await this.storage.read(projectId, saved);
    const report = parse(SavedReportSchema, JSON.parse(bytes.toString('utf8')));
    validateSavedReport(report, saved);
    return { kind: 'report', value: report, saved };
  }
}

function validateSavedReport(report: SavedReport, saved: SavedReportRecord): void {
  const data = { kind: 'report' as const, value: report, saved };
  parse(SurfaceDataSchema, data);
  if (
    report.projectId !== saved.projectId ||
    report.runRef !== saved.runRef ||
    Object.entries(saved.producer).some(
      ([key, value]) => report.producer[key as keyof typeof report.producer] !== value,
    ) ||
    report.capturedAt !== saved.capturedAt ||
    report.content.profile !== saved.profile ||
    report.content.title !== saved.title
  )
    throw new AppProblem('internal', 'The saved report identity is inconsistent.');
  const charts = report.content.modules
    .flatMap((module) => module.blocks)
    .filter((block) => block.kind === 'image');
  if (new Set(charts.map((chart) => chart.id)).size !== charts.length)
    throw new AppProblem('unsupported', 'Report chart identities must be unique.');
  for (const chart of charts) {
    const png = Buffer.from(chart.base64, 'base64');
    if (
      png.length < 33 ||
      png.length > REPORT_BYTES ||
      png.toString('base64') !== chart.base64 ||
      png.subarray(0, 8).toString('hex') !== '89504e470d0a1a0a' ||
      png.toString('ascii', 12, 16) !== 'IHDR' ||
      png.readUInt32BE(8) !== 13 ||
      png.readUInt32BE(16) !== chart.width ||
      png.readUInt32BE(20) !== chart.height
    )
      throw new AppProblem('unsupported', 'A report chart does not match its bounded dimensions.');
  }
  if (
    Buffer.byteLength(JSON.stringify(data)) > REPORT_BYTES ||
    Buffer.byteLength(
      JSON.stringify(report.content, (key, value) => (key === 'base64' ? undefined : value)),
    ) > REPORT_TEXT_BYTES
  )
    throw new AppProblem('unsupported', 'This complete report exceeds the saved report limits.');
}

/** Reuses the saved blob verbatim; attachment identity does not create another report copy. */
export function reportAsset(attachment: DraftAttachment, data: SurfaceData): EvidenceAsset {
  if (
    data.kind !== 'report' ||
    attachment.evidence.schemaVersion !== 8 ||
    attachment.capture ||
    attachment.presentation
  )
    throw new AppProblem('invalid_request', 'A whole saved report is required.');
  checkSelection(attachment.evidence, data);
  validateSavedReport(data.value, data.saved);
  const representation = reportRepresentation(data.value);
  const bytes = Buffer.from(JSON.stringify(data.value));
  if (digest(bytes) !== data.saved.asset.hash || bytes.length !== data.saved.asset.byteLength)
    throw new AppProblem('internal', 'The report does not match its saved content.');
  const capture = {
    kind: 'report' as const,
    capturedAt: data.saved.capturedAt,
    asset: { ...data.saved.asset, mediaType: 'application/json' as const },
    representation,
  };
  const manifest = {
    ...attachment,
    capture,
    capturedAt: capture.capturedAt,
    asset: capture.asset,
    representation,
  };
  validateManifest(manifest);
  return { manifest, bytes };
}
export function reportRepresentation(report: SavedReport) {
  return {
    kind: 'report' as const,
    truncated: false as const,
    textByteLength: Buffer.byteLength(JSON.stringify(reportReading(report))),
    imageCount: report.content.modules.flatMap((m) => m.blocks).filter((b) => b.kind === 'image')
      .length,
  };
}
export function reportPreview(asset: EvidenceAsset) {
  validateManifest(asset.manifest);
  const target = asset.manifest.evidence;
  if (
    target.schemaVersion !== 8 ||
    digest(asset.bytes) !== asset.manifest.asset.hash ||
    asset.bytes.length !== asset.manifest.asset.byteLength
  )
    throw new AppProblem('internal', 'The attached report identity is inconsistent.');
  const report = parse(
    SavedReportSchema,
    JSON.parse(new TextDecoder('utf-8', { fatal: true }).decode(asset.bytes)),
  );
  validateSavedReport(report, target.resource.saved);
  const rep = reportRepresentation(report);
  if (
    asset.manifest.representation.kind !== 'report' ||
    rep.textByteLength !== asset.manifest.representation.textByteLength ||
    rep.imageCount !== asset.manifest.representation.imageCount
  )
    throw new AppProblem('internal', 'The attached report inventory is inconsistent.');
  return { kind: 'report' as const, report };
}
