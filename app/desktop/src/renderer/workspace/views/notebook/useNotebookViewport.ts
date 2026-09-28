import { useEffect, type RefObject } from 'react';
import {
  parse,
  NotebookSelectionSchema,
  type NotebookSelection,
  type NotebookViewport,
  type RenderAcknowledgment,
} from '@gobble/contracts';
type Box = { left: number; top: number; right: number; bottom: number };
function clipping(element: HTMLElement): Box | undefined {
  if (!element.getClientRects().length) return;
  let box: Box = { left: 0, top: 0, right: window.innerWidth, bottom: window.innerHeight };
  for (let parent: HTMLElement | null = element; parent; parent = parent.parentElement) {
    const style = getComputedStyle(parent);
    if (style.visibility === 'hidden' || style.display === 'none') return;
    const r = parent.getBoundingClientRect();
    if (/auto|scroll|hidden|clip/.test(style.overflowX))
      box = {
        ...box,
        left: Math.max(box.left, r.left + parent.clientLeft),
        right: Math.min(box.right, r.left + parent.clientLeft + parent.clientWidth),
      };
    if (/auto|scroll|hidden|clip/.test(style.overflowY))
      box = {
        ...box,
        top: Math.max(box.top, r.top + parent.clientTop),
        bottom: Math.min(box.bottom, r.top + parent.clientTop + parent.clientHeight),
      };
  }
  return box.right > box.left && box.bottom > box.top ? box : undefined;
}
/** Only fully painted units grant authority. DOM offsets describe public projection text, never raw bytes. */
export function measureNotebookViewport(root: HTMLElement): NotebookViewport {
  const parts: NotebookSelection[] = [],
    budget = { units: 4096 };
  for (const element of root.querySelectorAll<HTMLElement>('[data-notebook-selection]')) {
    if (parts.length >= 64) break;
    const clip = clipping(element);
    if (!clip) continue;
    let selection: NotebookSelection;
    try {
      selection = parse(
        NotebookSelectionSchema,
        JSON.parse(element.dataset.notebookSelection ?? ''),
      );
    } catch {
      continue;
    }
    if (selection.selector.kind === 'image') {
      if (element.dataset.imagePainted !== 'true') continue;
      const r = element.getBoundingClientRect(),
        original = selection.selector.rect;
      const x = Math.max(0, Math.ceil(((clip.left - r.left) / r.width) * original.width)),
        y = Math.max(0, Math.ceil(((clip.top - r.top) / r.height) * original.height));
      const right = Math.min(
          original.width,
          Math.floor(((clip.right - r.left) / r.width) * original.width),
        ),
        bottom = Math.min(
          original.height,
          Math.floor(((clip.bottom - r.top) / r.height) * original.height),
        );
      if (right > x && bottom > y)
        parts.push({
          ...selection,
          selector: { ...selection.selector, rect: { x, y, width: right - x, height: bottom - y } },
        });
      continue;
    }
    if (!budget.units) continue;
    const text = element.textContent ?? '',
      base = selection.selector.start;
    const walker = document.createTreeWalker(element, NodeFilter.SHOW_TEXT),
      nodes: Array<{ node: Node; offset: number }> = [];
    let node: Node | null,
      offset = 0;
    while ((node = walker.nextNode())) {
      nodes.push({ node, offset });
      offset += node.textContent?.length ?? 0;
    }
    const range = document.createRange();
    const position = (at: number) => {
      for (let i = nodes.length - 1; i >= 0; i--) if (nodes[i]!.offset <= at) return nodes[i];
      return undefined;
    };
    let start: number | undefined;
    const flush = (end: number) => {
      if (start !== undefined && parts.length < 64) {
        parts.push({
          ...selection,
          selector: {
            kind: 'text',
            coordinateSpace: 'notebook-display-utf16',
            start: base + start,
            end: base + end,
          },
        });
        start = undefined;
      }
    };
    for (let at = 0; at < text.length;) {
      // Escaped controls are indivisible. Treat literal lookalikes conservatively as one unit too.
      const escape = /^\\u[0-9a-fA-F]{4}/.exec(text.slice(at)),
        length = escape?.[0].length ?? (text.codePointAt(at)! > 65535 ? 2 : 1),
        end = at + length;
      const a = position(at),
        b = position(end);
      if (!a || !b) break;
      range.setStart(a.node, at - a.offset);
      range.setEnd(b.node, end - b.offset);
      const rects = [...range.getClientRects()];
      const visible =
        rects.length > 0 &&
        rects.every(
          (r) =>
            r.height > 0 &&
            r.left >= clip.left - 0.1 &&
            r.top >= clip.top - 0.1 &&
            r.right <= clip.right + 0.1 &&
            r.bottom <= clip.bottom + 0.1,
        );
      if (visible && length <= budget.units && parts.length < 64) {
        start ??= at;
        budget.units -= length;
      } else flush(at);
      at = end;
      if (!budget.units || parts.length >= 64) {
        flush(at);
        break;
      }
    }
    flush(text.length);
  }
  return { parts };
}
export function useNotebookViewport(
  root: RefObject<HTMLDivElement | null>,
  ack: RenderAcknowledgment,
  ready: boolean,
) {
  useEffect(() => {
    const element = root.current;
    if (!element || !ready) return;
    let frame = 0,
      closed = false,
      last = '';
    const report = (viewport: NotebookViewport) => {
      const key = JSON.stringify(viewport);
      if (key === last) return;
      last = key;
      void window.gobble.workspace.notebookViewport({ acknowledgment: ack, viewport }).catch(() => {
        /* Replaced leases cannot accept late reports. */
      });
    };
    const changed = () => {
      if (closed) return;
      report({ parts: [] });
      cancelAnimationFrame(frame);
      frame = requestAnimationFrame(() => {
        if (!closed) report(measureNotebookViewport(element));
      });
    };
    const resize = new ResizeObserver(changed),
      mutation = new MutationObserver(changed);
    resize.observe(element);
    mutation.observe(element, {
      subtree: true,
      childList: true,
      characterData: true,
      attributes: true,
      attributeFilter: ['hidden', 'style', 'data-image-painted', 'data-notebook-selection', 'open'],
    });
    element.addEventListener('scroll', changed, true);
    element.addEventListener('load', changed, true);
    window.addEventListener('resize', changed);
    changed();
    return () => {
      closed = true;
      cancelAnimationFrame(frame);
      resize.disconnect();
      mutation.disconnect();
      element.removeEventListener('scroll', changed, true);
      element.removeEventListener('load', changed, true);
      window.removeEventListener('resize', changed);
      report({ parts: [] });
    };
  }, [root, ack.requestId, ready]);
}
