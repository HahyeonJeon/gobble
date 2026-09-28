import { useEffect, useRef } from 'react';
import type { PdfContent, RenderAcknowledgment, SharedReference } from '@gobble/contracts';
import { pixelRegion } from '@gobble/contracts/pdf-geometry';
import { usePageImage } from './usePageImage';
import { usePdfViewport } from './usePdfViewport';
import { PdfMarks } from './PdfMarks';
export function PdfReferenceContent({
  content,
  reference,
  acknowledgment,
  ready,
  onReady,
  onFailure,
}: {
  content: PdfContent;
  reference: SharedReference;
  acknowledgment: RenderAcknowledgment;
  ready: boolean;
  onReady: () => void;
  onFailure: (message: string) => void;
}) {
  const scroll = useRef<HTMLDivElement>(null),
    area = useRef<HTMLDivElement>(null),
    image = usePageImage(content.page.raster.png);
  usePdfViewport(scroll, area, content, acknowledgment, ready);
  const positioned = useRef(false);
  useEffect(() => {
    if (
      positioned.current ||
      !ready ||
      !scroll.current ||
      !area.current ||
      reference.evidence.schemaVersion !== 5
    )
      return;
    const box = pixelRegion(
      reference.evidence.selection.region,
      content.page.viewport,
      content.page.raster.width,
      content.page.raster.height,
    );
    positioned.current = true;
    scroll.current.scrollTop = Math.max(
      0,
      (box.y / content.page.raster.height) * area.current.clientHeight - 30,
    );
  }, [ready, reference, content]);
  return (
    <div
      ref={scroll}
      className="pdf-scroll pdf-reference-scroll"
      tabIndex={0}
      aria-label="PDF reference page"
    >
      <div
        ref={area}
        className="pdf-page"
        style={{ width: content.page.raster.width, maxWidth: '100%' }}
      >
        {image && (
          <img
            src={image}
            alt={'Referenced PDF page ' + (content.page.model.pageIndex + 1)}
            onLoad={onReady}
            onError={() =>
              onFailure('The PDF reference image could not be displayed. Return and try again.')
            }
          />
        )}
        <PdfMarks content={content} marks={[reference]} />
      </div>
    </div>
  );
}
