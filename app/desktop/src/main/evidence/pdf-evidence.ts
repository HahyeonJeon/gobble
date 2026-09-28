import { nativeImage } from 'electron';
import { pixelRegion } from '@gobble/contracts/pdf-geometry';
import type { CaptureResult, PageResult } from '@gobble/contracts/pdf-decoder';
import { assertPngHeader } from '../pdf/raster';
import { checkSelection } from '../workspace/selection';
import {
  type DraftAttachment,
  type SurfaceData,
  type PdfImageRepresentation,
} from '@gobble/contracts';
import type { PdfViews } from '../workspace/pdf-views';
import { AppProblem } from '../problem';
import { contentHash, type EvidenceAsset } from './materialize';
import { capturedManifest } from './capture';

/** Freeze the decoder's same-rendition crop into the existing image delivery representation. */
export async function materializePdf(
  views: PdfViews,
  attachment: DraftAttachment,
  data: SurfaceData,
): Promise<EvidenceAsset> {
  const target = attachment.evidence;
  if (
    target.schemaVersion !== 5 ||
    !target.origin ||
    data.kind !== 'file' ||
    data.value.content.kind !== 'pdf'
  )
    throw new AppProblem('invalid_request', 'A PDF page selection is required.');
  const page = data.value.content.page;
  const crop = await views.capture(target.origin.surfaceId, target, data);
  return pdfAsset(attachment, page, crop);
}

export type PdfRasterCapture = (
  attachment: DraftAttachment,
  data: SurfaceData,
) => Promise<EvidenceAsset>;
/** Lossless crop of the already validated Main raster; never a desktop screenshot or resample. */
export const capturePdfRaster: PdfRasterCapture = async (attachment, data) => {
  const target = attachment.evidence;
  if (target.schemaVersion !== 5 || data.kind !== 'file' || data.value.content.kind !== 'pdf')
    throw new AppProblem('invalid_request', 'A rendered PDF target is required.');
  checkSelection(target, data);
  const page = data.value.content.page;
  assertPngHeader(page.raster.png, page.raster.width, page.raster.height);
  const crop = pixelRegion(
    target.selection.region,
    page.viewport,
    page.raster.width,
    page.raster.height,
  );
  if (crop.width > 1536 || crop.height > 1536)
    throw new AppProblem(
      'unsupported',
      'This PDF region exceeds the image limit. Observe a smaller region.',
    );
  const image = nativeImage.createFromBuffer(Buffer.from(page.raster.png, 'base64'));
  const size = image.getSize();
  if (size.width !== page.raster.width || size.height !== page.raster.height)
    throw new AppProblem('invalid_request', 'The PDF raster dimensions changed.');
  const png = image.crop(crop).toPNG();
  return pdfAsset(attachment, page, {
    pixelRect: crop,
    raster: { width: crop.width, height: crop.height, png: png.toString('base64') },
  });
};
function pdfAsset(
  attachment: DraftAttachment,
  page: PageResult,
  crop: Pick<CaptureResult, 'pixelRect' | 'raster'>,
): EvidenceAsset {
  if (crop.raster.width > 1536 || crop.raster.height > 1536)
    throw new AppProblem(
      'unsupported',
      'This capture is too large. Select a smaller region or reduce the page zoom.',
    );
  const bytes = Buffer.from(crop.raster.png, 'base64');
  if (bytes.length > 786432)
    throw new AppProblem(
      'unsupported',
      'This PDF capture exceeds the image delivery limit. Select a smaller region.',
    );
  const { profile, pageIndex, viewBox, userUnit, intrinsicRotation } = page.model;
  const representation: PdfImageRepresentation = {
    kind: 'image',
    width: crop.raster.width,
    height: crop.raster.height,
    originalWidth: page.raster.width,
    originalHeight: page.raster.height,
    crop: crop.pixelRect,
    pdf: {
      profile,
      pageIndex,
      viewBox,
      userUnit,
      intrinsicRotation,
      modelHash: page.modelHash,
      viewport: page.viewport,
    },
  };
  return {
    bytes,
    manifest: capturedManifest({
      ...attachment,
      capture: {
        kind: 'pdf',
        capturedAt: Date.now(),
        asset: { hash: contentHash(bytes), byteLength: bytes.length, mediaType: 'image/png' },
        representation,
      },
    }),
  };
}
