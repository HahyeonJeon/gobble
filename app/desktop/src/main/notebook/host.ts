import { nativeImage } from 'electron';
import { readNotebook } from './worker-client';
import {
  LIMITS,
  NotebookProblem,
  type Snapshot,
  type Capture,
  type ViewSnapshot,
  type ImagePart,
} from './model';
import { resolveTarget } from './targets';
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
/** Owns one decoded document and cancellable read. RenderSession owns readiness; EvidenceStorage owns captures. */
export class NotebookHost {
  private snapshot: Snapshot | undefined;
  private request: AbortController | undefined;
  private closed = false;
  constructor(private readonly workerPath: string) {}
  validate(input: unknown): void {
    if (this.request) throw new NotebookProblem('stale', 'Notebook is changing.');
    resolveTarget(this.current(), input);
  }
  async load(bytes: Uint8Array): Promise<ViewSnapshot> {
    if (this.closed) throw new NotebookProblem('cancelled', 'Reader is closed.');
    this.request?.abort();
    const request = new AbortController();
    this.request = request;
    try {
      const snapshot = await readNotebook(bytes, request.signal, this.workerPath);
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
  image(input: unknown): { base64: string; width: number; height: number } {
    const { part } = resolveTarget(this.current(), input);
    if (part.kind !== 'image') throw new NotebookProblem('invalid', 'An image part is required.');
    const png = decoded(part).toPNG();
    if (png.length > LIMITS.bytes)
      throw new NotebookProblem('limit', 'Decoded display image exceeds 8 MiB.');
    return { base64: png.toString('base64'), width: part.width, height: part.height };
  }
  capture(input: unknown): Capture {
    const { target, part, rawQuote } = resolveTarget(this.current(), input);
    if (this.request) throw new NotebookProblem('stale', 'Notebook is changing.');
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
  }
}
