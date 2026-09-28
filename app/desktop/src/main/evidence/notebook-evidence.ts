import {
  notebookPart,
  type DraftAttachment,
  type SurfaceData,
  type NotebookRepresentation,
} from '@gobble/contracts';
import type { NotebookViews } from '../workspace/notebook-views';
import { AppProblem } from '../problem';
import { capturedManifest } from './capture';
import { contentHash, evidencePreview, type EvidenceAsset } from './materialize';

export type NotebookCapture = (
  attachment: DraftAttachment,
  data: SurfaceData,
) => Promise<EvidenceAsset>;

/** Materialize the Main-owned saved part into the existing immutable asset store. */
export async function materializeNotebook(
  views: NotebookViews,
  attachment: DraftAttachment,
  data: SurfaceData,
): Promise<EvidenceAsset> {
  const target = attachment.evidence;
  if (
    target.schemaVersion !== 6 ||
    !target.origin ||
    data.kind !== 'file' ||
    data.value.content.kind !== 'notebook'
  )
    throw new AppProblem('invalid_request', 'An exact Notebook selection is required.');
  const capture = views.capture(target.origin.surfaceId, target, data),
    capturedAt = Date.now();
  let bytes: Buffer, representation: NotebookRepresentation;
  if (capture.representation === 'text/plain') {
    bytes = Buffer.from(
      JSON.stringify({
        kind: 'text',
        text: capture.text,
        notebook: { target, capturedAt, rawQuote: capture.rawQuote },
      }),
    );
    if (bytes.length > 65536)
      throw new AppProblem(
        'unsupported',
        'Notebook text and its source quote exceed 64 KiB. Select a smaller range.',
      );
    representation = { kind: 'text', truncated: false, notebook: true };
  } else {
    const part = notebookPart(data.value.content.document, target.selection);
    if (part.kind !== 'image' || target.selection.selector.kind !== 'image')
      throw new AppProblem('invalid_request', 'A saved image region is required.');
    bytes = Buffer.from(capture.base64, 'base64');
    representation = {
      kind: 'image',
      notebook: true,
      width: capture.width,
      height: capture.height,
      originalWidth: part.width,
      originalHeight: part.height,
      crop: target.selection.selector.rect,
    };
  }
  const asset = {
    bytes,
    manifest: capturedManifest({
      ...attachment,
      capture: {
        kind: 'notebook',
        capturedAt,
        asset: {
          hash: contentHash(bytes),
          byteLength: bytes.length,
          mediaType: representation.kind === 'image' ? 'image/png' : 'application/json',
        },
        representation,
      },
    }),
  };
  evidencePreview(asset);
  return asset;
}
