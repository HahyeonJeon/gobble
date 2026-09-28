import { nativeImage } from 'electron';
import { readNotebook } from './worker-client';
import {
  LIMITS,
  NotebookProblem,
  type Snapshot,
  type Capture,
  type ViewReceipt,
  type ViewSnapshot,
  type ImagePart,
} from './model';
import { assertObserved, resolveTarget, validateReceipt } from './targets';
import { digest } from './text';
function decoded(part: ImagePart) {
  const image = nativeImage.createFromBuffer(Buffer.from(part.base64, 'base64'));
  const size = image.getSize();
  if (image.isEmpty() || size.width !== part.width || size.height !== part.height)
    throw new NotebookProblem(
      'unsupported',
      'Image decoding failed or orientation does not match the saved coordinates.',
    );
  return image;
}
/** Owns one immutable document and one visible receipt; captures are returned, not retained here. */
export class NotebookHost {
  private snapshot: Snapshot | undefined;
  private request: AbortController | undefined;
  private receipt: ViewReceipt | undefined;
  private generation = 0;
  private closed = false;
  async load(bytes: Uint8Array): Promise<ViewSnapshot> {
    if (this.closed) throw new NotebookProblem('cancelled', 'Reader is closed.');
    this.request?.abort();
    const request = new AbortController();
    this.request = request;
    this.receipt = undefined;
    try {
      const snapshot = await readNotebook(bytes, request.signal);
      if (this.closed || this.request !== request)
        throw new NotebookProblem('cancelled', 'Notebook load was superseded.');
      this.snapshot = snapshot;
      return this.view();
    } finally {
      if (this.request === request) this.request = undefined;
    }
  }
  private current(): Snapshot {
    if (!this.snapshot) throw new NotebookProblem('stale', 'Open a Notebook first.');
    return this.snapshot;
  }
  view(): ViewSnapshot {
    const s = this.current();
    const view: ViewSnapshot = {
      ...s,
      cells: s.cells.map((c) => ({
        ...c,
        source: { kind: 'text', text: c.source.text, digest: c.source.digest },
        outputs: c.outputs.map((o) => ({
          ...o,
          part:
            o.part.kind === 'text'
              ? { kind: 'text', text: o.part.text, digest: o.part.digest }
              : o.part.kind === 'image'
                ? {
                    kind: 'image',
                    mime: o.part.mime,
                    width: o.part.width,
                    height: o.part.height,
                    digest: o.part.digest,
                  }
                : o.part,
        })),
      })),
    };
    return structuredClone(view);
  }
  visible(input: unknown): number {
    if (this.request) throw new NotebookProblem('stale', 'Notebook is changing.');
    const receipt = validateReceipt(this.current(), input);
    if (receipt.generation <= this.generation)
      throw new NotebookProblem('stale', 'Visible report is no longer current.');
    this.generation = receipt.generation;
    this.receipt = structuredClone(receipt);
    return this.generation;
  }
  observe(): ViewReceipt {
    if (!this.receipt || this.request)
      throw new NotebookProblem('stale', 'The visible Notebook is not ready.');
    return structuredClone(this.receipt);
  }
  invalidate(): void {
    this.receipt = undefined;
  }
  image(input: unknown): { base64: string; width: number; height: number } {
    const { part } = resolveTarget(this.current(), input);
    if (part.kind !== 'image') throw new NotebookProblem('invalid', 'An image part is required.');
    const png = decoded(part).toPNG();
    if (png.length > LIMITS.bytes)
      throw new NotebookProblem('limit', 'Decoded display image exceeds 8 MiB.');
    return { base64: png.toString('base64'), width: part.width, height: part.height };
  }
  capture(input: unknown, observedGeneration?: number): Capture {
    const { target, part, rawQuote } = resolveTarget(this.current(), input);
    if (this.request) throw new NotebookProblem('stale', 'Notebook is changing.');
    if (observedGeneration !== undefined) {
      const receipt = this.observe();
      if (receipt.generation !== observedGeneration)
        throw new NotebookProblem('stale', 'The visible observation expired.');
      assertObserved(receipt, target);
    }
    const c = target.cell,
      label = (c.kind === 'id' ? c.id : 'Cell ' + (c.index + 1)) + ' · ' + target.part.kind;
    if (part.kind === 'text' && target.selector.kind === 'text') {
      const text = part.text.slice(target.selector.start, target.selector.end);
      if (
        Buffer.byteLength(text) > LIMITS.captureText ||
        Buffer.byteLength(rawQuote) > LIMITS.captureText
      )
        throw new NotebookProblem('limit', 'Text capture exceeds 64 KiB.');
      return { target, label, representation: 'text/plain', text, rawQuote, digest: digest(text) };
    }
    if (part.kind !== 'image' || target.selector.kind !== 'image')
      throw new NotebookProblem('invalid', 'Invalid capture representation.');
    const rect = target.selector.rect;
    if (rect.width > LIMITS.captureEdge || rect.height > LIMITS.captureEdge)
      throw new NotebookProblem(
        'limit',
        'Image capture exceeds 1536 pixels; select a smaller region.',
      );
    const png = decoded(part).crop(rect).toPNG();
    if (png.length > LIMITS.captureBytes)
      throw new NotebookProblem('limit', 'Image capture exceeds 768 KiB; select a smaller region.');
    return {
      target,
      label,
      representation: 'image/png',
      base64: png.toString('base64'),
      width: rect.width,
      height: rect.height,
      digest: digest(png),
    };
  }
  close(): void {
    this.closed = true;
    this.request?.abort();
    this.snapshot = undefined;
    this.receipt = undefined;
  }
}
