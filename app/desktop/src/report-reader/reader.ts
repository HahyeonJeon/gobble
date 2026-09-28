import type { FastqcContent, ReportBlock } from '@gobble/contracts';
import { FASTQC_STYLE, FASTQC_ICONS } from './profile';

function unsupported(): never {
  throw new Error('This report is not a complete supported FastQC 0.12.1 report.');
}
function attrs(element: Element, expected: Record<string, string> = {}): void {
  if (
    element.namespaceURI ||
    element.attributes.length !== Object.keys(expected).length ||
    Object.entries(expected).some(([key, value]) => element.getAttribute(key) !== value)
  )
    unsupported();
}
function elements(element: Element): Element[] {
  if (
    Array.from(element.childNodes).some(
      (n) => n.nodeType !== 1 && (n.nodeType !== 3 || n.textContent?.trim()),
    )
  )
    unsupported();
  return Array.from(element.children);
}
function tags(element: Element, names: string[]): Element[] {
  const children = elements(element);
  if (children.length !== names.length || children.some((child, i) => child.tagName !== names[i]))
    unsupported();
  return children;
}
function plain(element: Element): string {
  if (Array.from(element.childNodes).some((n) => n.nodeType !== 3)) unsupported();
  return element.textContent ?? '';
}
function image(element: Element, decorative: boolean): Extract<ReportBlock, { kind: 'image' }> {
  const alt = element.getAttribute('alt') ?? unsupported();
  const src = element.getAttribute('src') ?? unsupported();
  attrs(element, { ...(decorative ? {} : { class: 'indented' }), src, alt });
  if (
    element.childNodes.length ||
    !/^data:image\/png;base64,(?:[A-Za-z0-9+/]{4})*(?:[A-Za-z0-9+/]{2}==|[A-Za-z0-9+/]{3}=)?$/.test(
      src,
    )
  )
    unsupported();
  const base64 = src.slice(22);
  const raw = atob(base64);
  if (
    raw.length < 45 ||
    raw.length > 1024 * 1024 ||
    raw.slice(0, 8) !== '\x89PNG\r\n\x1a\n' ||
    raw.slice(12, 16) !== 'IHDR'
  )
    unsupported();
  const bytes = Uint8Array.from(raw, (c) => c.charCodeAt(0));
  const view = new DataView(bytes.buffer);
  const width = view.getUint32(16),
    height = view.getUint32(20);
  if (
    !width ||
    !height ||
    width > 1536 ||
    height > 1536 ||
    (decorative && (width !== 32 || height !== 32))
  )
    unsupported();
  if (decorative) {
    if (FASTQC_ICONS[alt] !== base64) unsupported();
    return { kind: 'image', id: 'image-0', alt, width, height, base64 };
  }
  let offset = 8,
    idat = false,
    ended = false;
  while (offset + 12 <= bytes.length) {
    const length = view.getUint32(offset),
      type = raw.slice(offset + 4, offset + 8);
    if (
      length > bytes.length - offset - 12 ||
      !['IHDR', 'IDAT', 'IEND', 'PLTE', 'tRNS', 'gAMA', 'sRGB', 'cHRM', 'pHYs', 'sBIT'].includes(
        type,
      )
    )
      unsupported();
    if ((offset === 8) !== (type === 'IHDR') || (type === 'IHDR' && length !== 13)) unsupported();
    // Check the source PNG chunks, without decoding or changing its compressed pixels.
    let crc = 0xffffffff;
    for (let i = offset + 4; i < offset + 8 + length; i++) {
      crc ^= bytes[i]!;
      for (let bit = 0; bit < 8; bit++) crc = (crc >>> 1) ^ (crc & 1 ? 0xedb88320 : 0);
    }
    if ((crc ^ 0xffffffff) >>> 0 !== view.getUint32(offset + 8 + length)) unsupported();
    if (type === 'IDAT') idat = true;
    offset += length + 12;
    if (type === 'IEND') {
      if (length || offset !== bytes.length) unsupported();
      ended = true;
      break;
    }
  }
  // Qualified 0.12.1 data URLs can omit the final one or two IEND CRC bytes.
  // Keep those original bytes. No image data/chunk is repaired or regenerated.
  const tail = raw.slice(offset);
  if (
    !ended &&
    [10, 11].includes(tail.length) &&
    '\x00\x00\x00\x00IEND\xae\x42\x60\x82'.startsWith(tail)
  )
    ended = true;
  if (!idat || !ended) unsupported();
  return { kind: 'image', id: 'image-0', alt, width, height, base64 };
}
function status(element: Element): string {
  const label = image(element, true).alt;
  if (!['[OK]', '[PASS]', '[WARN]', '[WARNING]', '[FAIL]'].includes(label)) unsupported();
  return label;
}

/** Strict known-dialect decoding. No source node is ever mounted or executed. */
export async function read(source: string): Promise<FastqcContent> {
  if (
    new TextEncoder().encode(source).byteLength > 1024 * 1024 ||
    !source.startsWith('<!DOCTYPE html>') ||
    /<\?|<!/i.test(source.slice(15))
  )
    unsupported();
  const doc = new DOMParser().parseFromString(source, 'application/xml');
  if (
    doc.querySelector('parsererror') ||
    !doc.doctype ||
    doc.doctype.name !== 'html' ||
    doc.doctype.publicId ||
    doc.doctype.systemId
  )
    unsupported();
  let count = 0;
  const bounded = (node: Node, depth: number) => {
    if (++count > 12000 || depth > 16) unsupported();
    for (const child of Array.from(node.childNodes)) bounded(child, depth + 1);
  };
  bounded(doc, 0);
  const html = doc.documentElement;
  if (html.tagName !== 'html') unsupported();
  attrs(html);
  const [head, body] = tags(html, ['head', 'body']) as [Element, Element];
  attrs(head);
  attrs(body);
  const [title, style] = tags(head, ['title', 'style']) as [Element, Element];
  attrs(title);
  attrs(style, { type: 'text/css' });
  if (plain(style) !== FASTQC_STYLE) unsupported();
  const sections = tags(body, ['div', 'div', 'div', 'div']);
  const [header, summary, main, footer] = sections as [Element, Element, Element, Element];
  ['header', 'summary', 'main', 'footer'].forEach((name, i) =>
    attrs(sections[i]!, { class: name }),
  );
  const [heading, filename] = tags(header, ['div', 'div']) as [Element, Element];
  attrs(heading, { id: 'header_title' });
  attrs(filename, { id: 'header_filename' });
  if (
    heading.firstChild?.nodeName !== 'img' ||
    heading.childNodes.length !== 2 ||
    heading.lastChild?.nodeType !== 3
  )
    unsupported();
  const logoLabel = image(heading.firstChild as Element, true).alt;
  if (logoLabel !== 'FastQC') unsupported();
  if (
    filename.childNodes.length !== 3 ||
    filename.firstChild?.nodeType !== 3 ||
    filename.lastChild?.nodeType !== 3 ||
    filename.childNodes[1]?.nodeName !== 'br'
  )
    unsupported();
  attrs(filename.childNodes[1] as Element);
  const [summaryHeading, list] = tags(summary, ['h2', 'ul']) as [Element, Element];
  attrs(summaryHeading);
  attrs(list);
  const summaryItems = elements(list).map((li) => {
    if (li.tagName !== 'li') unsupported();
    attrs(li);
    const [icon, link] = tags(li, ['img', 'a']) as [Element, Element];
    const href = link.getAttribute('href') ?? unsupported();
    if (!/^#M[0-9]+$/.test(href)) unsupported();
    attrs(link, { href });
    return { moduleId: href.slice(1), title: plain(link), status: status(icon) };
  });
  let imageCount = 0;
  const modules = elements(main).map((module) => {
    if (module.tagName !== 'div') unsupported();
    attrs(module, { class: 'module' });
    const [h2, ...children] = elements(module);
    if (!h2 || h2.tagName !== 'h2' || !children.length || children.length > 32) unsupported();
    const id = h2.getAttribute('id') ?? unsupported();
    if (!/^M[0-9]+$/.test(id)) unsupported();
    attrs(h2, { id });
    if (
      h2.childNodes.length !== 2 ||
      h2.firstChild?.nodeName !== 'img' ||
      h2.lastChild?.nodeType !== 3
    )
      unsupported();
    const blocks: ReportBlock[] = children.map((child) => {
      attrs(child);
      if (child.tagName === 'p') {
        if (child.children.length === 0) return { kind: 'text', text: plain(child) };
        const [img] = tags(child, ['img']);
        if (++imageCount > 16) unsupported();
        return { ...image(img!, false), id: 'image-' + imageCount };
      }
      if (child.tagName !== 'table') return unsupported();
      const [thead, tbody] = tags(child, ['thead', 'tbody']) as [Element, Element];
      attrs(thead);
      attrs(tbody);
      const rows = (parent: Element, cellName: string) =>
        elements(parent).map((tr) => {
          if (tr.tagName !== 'tr') unsupported();
          attrs(tr);
          const cells = elements(tr);
          if (!cells.length || cells.length > 32) unsupported();
          return cells.map((cell) => {
            if (cell.tagName !== cellName) unsupported();
            attrs(cell);
            return plain(cell);
          });
        });
      const headers = rows(thead, 'th');
      const values = rows(tbody, 'td');
      if (
        headers.length !== 1 ||
        values.length > 2000 ||
        values.some((r) => r.length !== headers[0]!.length)
      )
        unsupported();
      return { kind: 'table', headers: headers[0]!, rows: values };
    });
    return {
      id,
      title: h2.lastChild!.textContent!,
      status: status(h2.firstChild as Element),
      blocks,
    };
  });
  if (
    !modules.length ||
    modules.length > 16 ||
    new Set(modules.map((m) => m.id)).size !== modules.length ||
    summaryItems.length !== modules.length ||
    summaryItems.some((s, i) => s.moduleId !== modules[i]!.id || s.title !== modules[i]!.title)
  )
    unsupported();
  if (
    footer.childNodes.length !== 3 ||
    footer.firstChild?.nodeType !== 3 ||
    footer.lastChild?.nodeType !== 3 ||
    footer.childNodes[1]?.nodeName !== 'a'
  )
    unsupported();
  const link = footer.childNodes[1] as Element;
  attrs(link, { href: 'http://www.bioinformatics.babraham.ac.uk/projects/fastqc/' });
  if (plain(link) !== 'FastQC' || footer.textContent !== 'Produced by FastQC  (version 0.12.1)')
    unsupported();
  const content: FastqcContent = {
    profile: 'fastqc-0.12.1-v1',
    title: plain(title),
    logoLabel,
    headerTitle: heading.lastChild!.textContent!,
    headerFilename: filename.firstChild!.textContent + '\n' + filename.lastChild!.textContent,
    summaryHeading: plain(summaryHeading),
    summary: summaryItems,
    modules,
    footer: footer.textContent,
  };
  const textContent = JSON.stringify(content, (key, value) =>
    key === 'base64' ? undefined : value,
  );
  if (new TextEncoder().encode(textContent).byteLength > 64 * 1024) unsupported();
  for (const chart of content.modules.flatMap((m) => m.blocks).filter((b) => b.kind === 'image')) {
    const bytes = Uint8Array.from(atob(chart.base64), (c) => c.charCodeAt(0));
    const bitmap = await createImageBitmap(new Blob([bytes], { type: 'image/png' }));
    try {
      if (bitmap.width !== chart.width || bitmap.height !== chart.height) unsupported();
    } finally {
      bitmap.close();
    }
  }
  return content;
}
