import { evidenceTextBytes } from './evidence';
import { validateNotebookRepresentation } from './notebook-evidence';
import { validatePdfImage } from './pdf';
import type { DraftAttachment, EvidenceManifest } from './evidence';
import { validateReferencePresentation } from './reference-presentation';
import type { WorkspaceDocument } from './workspace-document';
import { ContractValidationError, validateEvidence } from './validation';

/** Expand durable capture metadata for either local preview or delivery preparation. */
export function draftCaptureManifest(attachment: DraftAttachment): EvidenceManifest | undefined {
  if (!attachment.capture) return undefined;
  const { kind: _kind, ...saved } = attachment.capture;
  const manifest = { ...attachment, ...saved };
  validateManifest(manifest);
  return manifest;
}

/** Source freshness is checked by main at use time; closed Surfaces remain valid historical targets. */
export function validateEvidenceDocument(doc: WorkspaceDocument): void {
  const fail = (message: string): never => {
    throw new ContractValidationError(message);
  };
  const ids = new Set<string>();
  const check = (item: DraftAttachment) => {
    if (item.evidence.projectId !== doc.workspace.projectId)
      fail('Attachment belongs to another Project.');
    if (ids.has(item.attachmentId)) fail('Attachment identities must be unique.');
    ids.add(item.attachmentId);
    validateEvidence(item.evidence, doc.workspace.projectId);
    if (item.evidence.schemaVersion === 8 && item.capture?.kind !== 'report')
      fail('Report evidence requires the saved report.');
    if (item.evidence.schemaVersion === 7 && item.capture?.kind !== 'pipeline')
      fail('Pipeline evidence requires frozen capture bytes.');
    if (item.capture?.kind === 'pipeline' && item.evidence.schemaVersion !== 7)
      fail('Pipeline capture requires a pipeline target.');
    if (item.evidence.schemaVersion === 6 && item.capture?.kind !== 'notebook')
      fail('Notebook evidence requires frozen capture bytes.');
    if (item.capture?.kind === 'notebook') {
      if (item.evidence.schemaVersion !== 6) fail('Notebook capture requires a Notebook target.');
      else validateNotebookRepresentation(item.evidence, item.capture.representation);
    }
    if (item.evidence.schemaVersion === 5 && item.capture?.kind !== 'pdf')
      fail('PDF evidence requires frozen raster bytes.');
    if (item.capture?.kind === 'pdf') {
      if (item.evidence.schemaVersion !== 5) fail('PDF capture requires a PDF target.');
      else validatePdfImage(item.evidence, item.capture.representation);
    }
    if (item.evidence.schemaVersion === 4 && !item.capture)
      fail('Dependency evidence requires frozen capture bytes.');
    validateReferencePresentation(item.evidence, item.presentation);
    if (
      item.capture?.kind === 'observed' &&
      (item.evidence.resource.kind === 'file' ||
        item.capture.representation.kind !== item.evidence.resource.kind ||
        item.presentation)
    )
      fail('A captured observation requires its exact Run or log subject.');
  };
  for (const item of doc.chat.attachments ?? []) check(item);
  for (const submission of doc.collaboration?.submissions ?? []) {
    const items = submission.evidence ?? [];
    if (!!items.length !== !!submission.preparedEvidenceId)
      fail('Sent evidence needs its preparation identity.');
    if (items.filter((item) => item.representation.kind === 'image').length > 2)
      fail('A message can include at most two images.');
    for (const item of items) {
      check(item);
      validateManifest(item);
    }
    if (items.reduce((sum, item) => sum + evidenceTextBytes(item), 0) > 64 * 1024)
      fail('Message evidence exceeds its text limit.');
  }
}
export function validateManifest(item: EvidenceManifest): void {
  const fail = (): never => {
    throw new ContractValidationError(
      'Evidence representation does not match its asset or selection.',
    );
  };
  validateEvidence(item.evidence, item.evidence.projectId);
  validateReferencePresentation(item.evidence, item.presentation);
  if (item.presentation && item.representation.kind !== 'table') fail();
  const view = item.representation;
  const selection = item.evidence.selection;
  const resourceKind = item.evidence.resource.kind;
  if (item.evidence.schemaVersion === 4 && (!item.capture || view.kind !== 'run')) fail();
  if (
    item.capture &&
    (item.capture.capturedAt !== item.capturedAt ||
      JSON.stringify(item.capture.asset) !== JSON.stringify(item.asset) ||
      JSON.stringify(item.capture.representation) !== JSON.stringify(item.representation))
  )
    fail();
  if (
    (resourceKind === 'run' && view.kind !== 'run') ||
    (resourceKind === 'log' && view.kind !== 'log') ||
    (resourceKind === 'file' && (view.kind === 'run' || view.kind === 'log'))
  )
    fail();
  if (item.evidence.schemaVersion === 8) {
    const saved = item.evidence.resource.saved;
    if (
      item.capture?.kind !== 'report' ||
      view.kind !== 'report' ||
      item.asset.mediaType !== 'application/json' ||
      item.asset.hash !== saved.asset.hash ||
      item.asset.byteLength !== saved.asset.byteLength ||
      item.capturedAt !== saved.capturedAt
    )
      fail();
    return;
  }
  if (item.capture?.kind === 'report' || view.kind === 'report') fail();
  if (item.evidence.schemaVersion === 7) {
    if (
      item.capture?.kind !== 'pipeline' ||
      view.kind !== 'pipeline' ||
      item.asset.mediaType !== 'application/json' ||
      item.asset.byteLength > 65536
    )
      fail();
    return;
  }
  if (item.capture?.kind === 'pipeline' || view.kind === 'pipeline') fail();
  if (item.evidence.schemaVersion === 6) {
    if (
      item.capture?.kind !== 'notebook' ||
      !('notebook' in view) ||
      (view.kind === 'image'
        ? item.asset.mediaType !== 'image/png' || item.asset.byteLength > 786432
        : item.asset.mediaType !== 'application/json' || item.asset.byteLength > 65536)
    )
      fail();
    if ('notebook' in view) validateNotebookRepresentation(item.evidence, view);
    return;
  }
  if (item.capture?.kind === 'notebook' || 'notebook' in view) fail();
  if (item.evidence.schemaVersion === 5) {
    if (
      item.capture?.kind !== 'pdf' ||
      view.kind !== 'image' ||
      !('pdf' in view) ||
      item.asset.mediaType !== 'image/png' ||
      item.asset.byteLength > 786432
    )
      fail();
    if (view.kind === 'image' && 'pdf' in view) validatePdfImage(item.evidence, view);
    return;
  }
  if (item.capture?.kind === 'pdf' || (view.kind === 'image' && 'pdf' in view)) fail();
  if (view.kind === 'image') {
    if (
      item.asset.mediaType !== 'image/png' ||
      item.asset.byteLength > 786432 ||
      view.crop.x + view.crop.width > view.originalWidth ||
      view.crop.y + view.crop.height > view.originalHeight ||
      (selection &&
        (selection.kind !== 'image' ||
          selection.originalWidth !== view.originalWidth ||
          selection.originalHeight !== view.originalHeight))
    )
      fail();
    if (
      selection?.kind === 'image' &&
      (view.crop.x !== Math.floor(selection.x * view.originalWidth) ||
        view.crop.y !== Math.floor(selection.y * view.originalHeight) ||
        view.crop.x + view.crop.width !==
          Math.min(
            view.originalWidth,
            Math.ceil((selection.x + selection.width) * view.originalWidth),
          ) ||
        view.crop.y + view.crop.height !==
          Math.min(
            view.originalHeight,
            Math.ceil((selection.y + selection.height) * view.originalHeight),
          ))
    )
      fail();
  } else if (
    item.asset.mediaType !== 'application/json' ||
    item.asset.byteLength > 65536 ||
    (selection &&
      (view.kind === 'log'
        ? selection.kind !== 'text' && selection.kind !== 'log-text'
        : view.kind === 'run'
          ? !['run-task', 'run-group', 'run-dependency'].includes(selection.kind)
          : selection.kind !== view.kind))
  )
    fail();
}
