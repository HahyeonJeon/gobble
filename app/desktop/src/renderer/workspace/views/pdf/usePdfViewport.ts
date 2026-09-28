import { useEffect, type RefObject } from 'react';
import type { PdfContent, RenderAcknowledgment } from '@gobble/contracts';
import { regionFromPixels } from '@gobble/contracts/pdf-geometry';
import type { Rect } from '@gobble/contracts/pdf-decoder';

/** Report fully visible raster pixels; scrolling has no durable Workspace write. */
export function usePdfViewport(
  scroll: RefObject<HTMLElement | null>,
  page: RefObject<HTMLElement | null>,
  content: PdfContent,
  ack: RenderAcknowledgment,
  ready: boolean,
) {
  useEffect(() => {
    const root = scroll.current,
      area = page.current;
    if (!root || !area || !ready) return;
    let previous: string | undefined;
    const report = () => {
      const a = area.getBoundingClientRect(),
        b = root.getBoundingClientRect(),
        raster = content.page.raster;
      const left = Math.max(a.left, b.left + root.clientLeft, 0),
        top = Math.max(a.top, b.top + root.clientTop, 0),
        right = Math.min(a.right, b.left + root.clientLeft + root.clientWidth, innerWidth),
        bottom = Math.min(a.bottom, b.top + root.clientTop + root.clientHeight, innerHeight);
      let region: Rect | null = null;
      if (a.width > 0 && a.height > 0 && right > left && bottom > top) {
        const pixels: Rect = [
          Math.ceil(((left - a.left) / a.width) * raster.width),
          Math.ceil(((top - a.top) / a.height) * raster.height),
          Math.floor(((right - a.left) / a.width) * raster.width),
          Math.floor(((bottom - a.top) / a.height) * raster.height),
        ];
        if (pixels[2] > pixels[0] && pixels[3] > pixels[1]) {
          const raw = regionFromPixels(pixels, content.page.viewport),
            box = content.page.model.viewBox;
          region = [
            Math.max(box[0], raw[0]),
            Math.max(box[1], raw[1]),
            Math.min(box[2], raw[2]),
            Math.min(box[3], raw[3]),
          ];
        }
      }
      const key = JSON.stringify(region);
      if (key === previous) return;
      previous = key;
      void window.gobble.workspace.pdfViewport({ acknowledgment: ack, region }).catch(() => {});
    };
    const resize = new ResizeObserver(report);
    resize.observe(root);
    resize.observe(area);
    root.addEventListener('scroll', report, { passive: true });
    window.addEventListener('resize', report);
    report();
    return () => {
      resize.disconnect();
      root.removeEventListener('scroll', report);
      window.removeEventListener('resize', report);
    };
  }, [scroll, page, content, ack, ready]);
}
