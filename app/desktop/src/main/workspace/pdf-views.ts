import { createHash } from 'node:crypto';
import {
  defaultPdfNavigation,
  validatePdfSelection,
  type FileContent,
  type Surface,
  type SurfaceData,
  type PdfTarget,
} from '@gobble/contracts';
import type { PageResult } from '@gobble/contracts/pdf-decoder';
import { PdfHost } from '../pdf/host';
import { AppProblem } from '../problem';

type Entry = {
  host: PdfHost;
  source?: Omit<FileContent, 'content'>;
  pageCount: number;
  page?: PageResult;
  busy: boolean;
};
/** Transient source/decoder ownership. Only the visible PDF Surfaces retain a host. */
export class PdfViews {
  private entries = new Map<string, Entry>();
  private allowed = new Set<string>();
  private referenceHost: PdfHost | undefined;
  constructor(private readonly createHost: () => PdfHost) {}
  private key(projectId: string, surfaceId: string) {
    return projectId + '/' + surfaceId;
  }
  retain(projectId: string, surfaceIds: string[]): void {
    this.allowed = new Set(surfaceIds.map((id) => this.key(projectId, id)));
    for (const [key, entry] of this.entries)
      if (!this.allowed.has(key)) {
        entry.host.stop();
        this.entries.delete(key);
      }
  }
  clear(): void {
    this.referenceHost?.stop();
    this.referenceHost = undefined;
    this.allowed.clear();
    for (const entry of this.entries.values()) entry.host.stop();
    this.entries.clear();
  }
  async read(
    surface: Surface,
    read: () => Promise<FileContent>,
    refresh: boolean,
  ): Promise<SurfaceData> {
    if (surface.view !== 'pdf')
      throw new AppProblem('invalid_request', 'A PDF Surface is required.');
    const key = this.key(surface.projectId, surface.surfaceId);
    if (!this.allowed.has(key))
      throw new AppProblem('stale_revision', 'Reveal the PDF before reading it.');
    let entry = this.entries.get(key);
    if (entry && (refresh || entry.busy || entry.host.diagnostics().closed)) {
      entry.host.stop();
      this.entries.delete(key);
      entry = undefined;
    }
    if (!entry) {
      if (this.entries.size >= 2)
        throw new AppProblem('unsupported', 'At most two PDF readers can be active.');
      entry = { host: this.createHost(), pageCount: 0, busy: true };
      this.entries.set(key, entry);
    } else entry.busy = true;
    const current = entry;
    const assert = () => {
      if (this.entries.get(key) !== current || !this.allowed.has(key))
        throw new AppProblem('stale_revision', 'This PDF request was replaced.');
    };
    try {
      if (!current.source) {
        const file = await read();
        assert();
        if (file.content.kind !== 'pdf')
          throw new AppProblem(
            'stale_revision',
            'The file format changed. Close and reopen this view.',
          );
        const bytes = Buffer.from(file.content.base64, 'base64');
        if (
          bytes.length !== file.size ||
          'sha256:' + createHash('sha256').update(bytes).digest('hex') !== file.revision
        )
          throw new AppProblem('invalid_request', 'PDF bytes do not match the source revision.');
        await current.host.start();
        assert();
        const opened = await current.host.open(bytes);
        assert();
        if (opened.kind !== 'opened')
          throw new AppProblem('internal', 'PDF opening did not return document information.');
        const { content: _content, ...source } = file;
        current.source = source;
        current.pageCount = opened.pageCount;
      }
      const navigation = surface.pdf ?? defaultPdfNavigation();
      if (navigation.pageIndex >= current.pageCount)
        throw new AppProblem(
          'unsupported',
          'This page is no longer in the PDF. Close this view and reopen the file to start at page 1.',
        );
      const page = await current.host.page(
        navigation.pageIndex,
        navigation.scale,
        navigation.rotation,
        1,
      );
      assert();
      current.page = page;
      return {
        kind: 'file',
        value: { ...current.source, content: { kind: 'pdf', pageCount: current.pageCount, page } },
      };
    } catch (error) {
      if (this.entries.get(key) === current) {
        current.host.stop();
        this.entries.delete(key);
      }
      if (error instanceof AppProblem) throw error;
      throw new AppProblem(
        'unsupported',
        error instanceof Error ? error.message : 'The PDF could not be rendered.',
      );
    } finally {
      current.busy = false;
    }
  }
  /** One bounded exact-page read for User Show; never mutates an interactive reader. */
  async reference(target: PdfTarget, read: () => Promise<FileContent>): Promise<SurfaceData> {
    if (this.referenceHost)
      throw new AppProblem('unsupported', 'Another PDF reference is loading.');
    const host = this.createHost();
    this.referenceHost = host;
    const assert = () => {
      if (this.referenceHost !== host)
        throw new AppProblem('stale_revision', 'The PDF reference was cancelled.');
    };
    try {
      const file = await read();
      assert();
      if (file.content.kind !== 'pdf' || file.revision !== target.dataRevision)
        throw new AppProblem(
          'stale_revision',
          'This PDF source changed. Open matching captured evidence when available.',
        );
      const bytes = Buffer.from(file.content.base64, 'base64');
      if (
        bytes.length !== file.size ||
        'sha256:' + createHash('sha256').update(bytes).digest('hex') !== file.revision
      )
        throw new AppProblem('invalid_request', 'PDF bytes do not match the source revision.');
      await host.start();
      assert();
      const opened = await host.open(bytes);
      assert();
      if (opened.kind !== 'opened')
        throw new AppProblem('unsupported', 'PDF page information is unavailable.');
      const page = await host.page(target.selection.pageIndex, 1, 0, 1);
      assert();
      const content = { kind: 'pdf' as const, pageCount: opened.pageCount, page };
      validatePdfSelection(target.selection, content);
      return { kind: 'file', value: { ...file, content } };
    } catch (error) {
      if (error instanceof AppProblem) throw error;
      throw new AppProblem(
        'unsupported',
        error instanceof Error ? error.message : 'The PDF reference could not be read.',
      );
    } finally {
      host.stop();
      if (this.referenceHost === host) this.referenceHost = undefined;
    }
  }
  async capture(surfaceId: string, target: PdfTarget, data: SurfaceData) {
    if (data.kind !== 'file' || data.value.content.kind !== 'pdf')
      throw new AppProblem('invalid_request', 'A rendered PDF is required.');
    validatePdfSelection(target.selection, data.value.content);
    const key = this.key(target.projectId, surfaceId),
      entry = this.entries.get(key),
      page = data.value.content.page;
    if (
      !entry ||
      entry.busy ||
      entry.source?.revision !== target.dataRevision ||
      entry.page?.renditionId !== page.renditionId
    )
      throw new AppProblem('stale_revision', 'This PDF page has changed. Select again.');
    entry.busy = true;
    try {
      const capture = await entry.host.capture(
        page.renditionId,
        page.modelHash,
        target.selection.region,
      );
      if (this.entries.get(key) !== entry || !this.allowed.has(key))
        throw new AppProblem('stale_revision', 'The PDF was closed during capture.');
      return capture;
    } catch (error) {
      if (error instanceof AppProblem) throw error;
      throw new AppProblem(
        'unsupported',
        error instanceof Error ? error.message : 'The PDF region could not be captured.',
      );
    } finally {
      entry.busy = false;
    }
  }
}
