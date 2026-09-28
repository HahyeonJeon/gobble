import { LIMITS, type PartAddress, type Selector, type VisiblePart, type Rect } from './model';
type Box = { left: number; right: number; top: number; bottom: number };
function intersection(a: Box, b: Box): Box {
  return {
    left: Math.max(a.left, b.left),
    right: Math.min(a.right, b.right),
    top: Math.max(a.top, b.top),
    bottom: Math.min(a.bottom, b.bottom),
  };
}
function included(rect: Box, clip: Box) {
  return (
    rect.right > rect.left &&
    rect.bottom > rect.top &&
    rect.left >= clip.left - 0.1 &&
    rect.right <= clip.right + 0.1 &&
    rect.top >= clip.top - 0.1 &&
    rect.bottom <= clip.bottom + 0.1
  );
}
/** DOM geometry only. Host validates the corresponding content and generation independently. */
export function visibleParts(
  container: HTMLElement,
  addresses: Map<string, PartAddress>,
): VisiblePart[] {
  const result: VisiblePart[] = [];
  let units = 0,
    scanned = 0;
  const view = intersection(container.getBoundingClientRect(), {
    left: 0,
    right: innerWidth,
    top: 0,
    bottom: innerHeight,
  });
  for (const el of container.querySelectorAll<HTMLElement>('[data-part]')) {
    if (!el.checkVisibility()) continue;
    const key = el.dataset.part,
      addr = key ? addresses.get(key) : undefined;
    if (!addr) continue;
    let clip = intersection(view, el.getBoundingClientRect());
    for (let parent = el.parentElement; parent; parent = parent.parentElement) {
      const style = getComputedStyle(parent);
      if (style.overflowX !== 'visible' || style.overflowY !== 'visible') {
        const b = parent.getBoundingClientRect();
        clip = intersection(clip, {
          left: b.left + parent.clientLeft,
          top: b.top + parent.clientTop,
          right: b.left + parent.clientLeft + parent.clientWidth,
          bottom: b.top + parent.clientTop + parent.clientHeight,
        });
      }
      if (parent === container) break;
    }
    if (clip.right <= clip.left || clip.bottom <= clip.top) continue;
    const selectors: Selector[] = [];
    if (el instanceof HTMLImageElement) {
      if (!el.complete || !el.naturalWidth || !el.naturalHeight) continue;
      const box = el.getBoundingClientRect(),
        x = Math.ceil(((clip.left - box.left) * el.naturalWidth) / box.width),
        y = Math.ceil(((clip.top - box.top) * el.naturalHeight) / box.height);
      const rect: Rect = {
        x,
        y,
        width: Math.floor(((clip.right - box.left) * el.naturalWidth) / box.width) - x,
        height: Math.floor(((clip.bottom - box.top) * el.naturalHeight) / box.height) - y,
      };
      if (rect.width > 0 && rect.height > 0) {
        selectors.push({ kind: 'image', rect });
        units++;
      }
    } else {
      for (const line of el.querySelectorAll<HTMLElement>('[data-start]')) {
        const lineBox = line.getBoundingClientRect();
        if (lineBox.bottom < clip.top || lineBox.top > clip.bottom) continue;
        const node = line.firstChild;
        if (!node || node.nodeType !== Node.TEXT_NODE) continue;
        const text = node.textContent ?? '',
          offset = Number(line.dataset.start);
        let start: number | undefined,
          end = 0;
        const flush = () => {
          if (start !== undefined) {
            selectors.push({ kind: 'text', start, end });
            start = undefined;
          }
        };
        for (let i = 0; i < text.length && scanned < 4096;) {
          scanned++;
          const step = (text.codePointAt(i) ?? 0) > 0xffff ? 2 : 1;
          const range = document.createRange();
          range.setStart(node, i);
          range.setEnd(node, i + step);
          if (
            included(range.getBoundingClientRect(), clip) &&
            units + step <= LIMITS.visibleUnits
          ) {
            start ??= offset + i;
            end = offset + i + step;
            units += step;
          } else flush();
          i += step;
        }
        flush();
        if (units >= LIMITS.visibleUnits || scanned >= 4096) break;
      }
    }
    if (selectors.length) result.push({ ...addr, selectors });
    if (units >= LIMITS.visibleUnits || scanned >= 4096 || result.length >= LIMITS.visibleParts)
      break;
  }
  return result;
}
export function textSelection(element: HTMLElement): { start: number; end: number } | null {
  const selected = getSelection();
  if (!selected?.rangeCount || selected.isCollapsed) return null;
  const range = selected.getRangeAt(0);
  if (!element.contains(range.startContainer) || !element.contains(range.endContainer)) return null;
  const before = range.cloneRange();
  before.selectNodeContents(element);
  before.setEnd(range.startContainer, range.startOffset);
  const through = range.cloneRange();
  through.selectNodeContents(element);
  through.setEnd(range.endContainer, range.endOffset);
  const start = before.toString().length,
    end = through.toString().length;
  return end > start ? { start, end } : null;
}
