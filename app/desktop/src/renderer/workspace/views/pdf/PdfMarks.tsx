import {
  referenceAuthor,
  validatePdfSelection,
  type PdfContent,
  type SharedReference,
} from '@gobble/contracts';
import { pixelRegion } from '@gobble/contracts/pdf-geometry';
/** Authored pointers never change the User's local selection or capture pixels. */
export function PdfMarks({ content, marks }: { content: PdfContent; marks: SharedReference[] }) {
  return marks.slice(-4).map((mark) => {
    const evidence = mark.evidence;
    if (
      mark.retracted ||
      evidence.schemaVersion !== 5 ||
      evidence.dataRevision !== content.page.model.revision
    )
      return null;
    try {
      validatePdfSelection(evidence.selection, content);
    } catch {
      return null;
    }
    const raster = content.page.raster,
      box = pixelRegion(
        evidence.selection.region,
        content.page.viewport,
        raster.width,
        raster.height,
      );
    return (
      <div
        key={mark.referenceId}
        className={'pdf-shared-mark ' + mark.author.kind}
        data-reference-ids={mark.referenceId}
        data-label-placement={box.y >= 24 ? 'above' : 'below'}
        role="note"
        aria-label={referenceAuthor(mark) + (mark.note ? ': ' + mark.note : '')}
        style={{
          left: (box.x / raster.width) * 100 + '%',
          top: (box.y / raster.height) * 100 + '%',
          width: (box.width / raster.width) * 100 + '%',
          height: (box.height / raster.height) * 100 + '%',
        }}
      >
        <span>{referenceAuthor(mark)}</span>
      </div>
    );
  });
}
