import { createHash } from 'node:crypto';
import {
  NotebookDocumentSchema,
  parse,
  notebookPart,
  type NotebookTarget,
  type FileContent,
  type Surface,
  type SurfaceData,
} from '@gobble/contracts';
import { NotebookHost } from '../notebook/host';
import type { Target } from '../notebook/model';
import { AppProblem } from '../problem';

export function privateNotebookTarget(target: NotebookTarget): Target {
  const { cell, part, profile, selector } = target.selection;
  return {
    revision: target.dataRevision,
    profile,
    cell,
    part,
    selector:
      selector.kind === 'text'
        ? { kind: 'text', start: selector.start, end: selector.end }
        : { kind: 'image', rect: selector.rect },
  };
}
type Entry = { host: NotebookHost; source?: Omit<FileContent, 'content'>; busy: boolean };
/** Retains only visible Notebook sources. The Workspace writer owns Project and render authority. */
export class NotebookViews {
  private entries = new Map<string, Entry>();
  private referenceHost: NotebookHost | undefined;
  private allowed = new Set<string>();
  constructor(private readonly createHost: () => NotebookHost) {}
  private key(projectId: string, surfaceId: string) {
    return projectId + '/' + surfaceId;
  }
  retain(projectId: string, surfaceIds: string[]): void {
    this.allowed = new Set(surfaceIds.map((id) => this.key(projectId, id)));
    for (const [key, entry] of this.entries)
      if (!this.allowed.has(key)) {
        entry.host.close();
        this.entries.delete(key);
      }
  }
  clear(): void {
    this.referenceHost?.close();
    this.referenceHost = undefined;
    this.allowed.clear();
    for (const e of this.entries.values()) e.host.close();
    this.entries.clear();
  }
  async read(
    surface: Surface,
    read: () => Promise<FileContent>,
    refresh: boolean,
  ): Promise<SurfaceData> {
    const key = this.key(surface.projectId, surface.surfaceId);
    if (surface.view !== 'notebook' || !this.allowed.has(key))
      throw new AppProblem('stale_revision', 'Reveal the Notebook before reading it.');
    let entry = this.entries.get(key);
    if (entry && (refresh || entry.busy)) {
      entry.host.close();
      this.entries.delete(key);
      entry = undefined;
    }
    if (!entry) {
      if (this.entries.size >= 2)
        throw new AppProblem('unsupported', 'At most two Notebook readers can be active.');
      entry = { host: this.createHost(), busy: true };
      this.entries.set(key, entry);
    } else entry.busy = true;
    const current = entry;
    const assert = () => {
      if (this.entries.get(key) !== current || !this.allowed.has(key))
        throw new AppProblem('stale_revision', 'This Notebook read was replaced.');
    };
    try {
      if (!current.source) {
        const file = await read();
        assert();
        if (file.content.kind !== 'notebook')
          throw new AppProblem(
            'stale_revision',
            'The file format changed. Close and reopen this view.',
          );
        const bytes = Buffer.from(file.content.base64, 'base64');
        if (
          bytes.toString('base64') !== file.content.base64 ||
          bytes.length !== file.size ||
          'sha256:' + createHash('sha256').update(bytes).digest('hex') !== file.revision
        )
          throw new AppProblem(
            'invalid_request',
            'Notebook bytes do not match the source revision.',
          );
        await current.host.load(bytes);
        assert();
        const { content: _content, ...source } = file;
        current.source = source;
      }
      const document = parse(NotebookDocumentSchema, current.host.view());
      if (document.revision !== current.source.revision)
        throw new AppProblem('invalid_request', 'Notebook projection has a different revision.');
      return {
        kind: 'file',
        value: { ...current.source, content: { kind: 'notebook', document } },
      };
    } catch (error) {
      if (this.entries.get(key) === current) {
        current.host.close();
        this.entries.delete(key);
      }
      throw notebookProblem(error);
    } finally {
      current.busy = false;
    }
  }
  async reference(target: NotebookTarget, read: () => Promise<FileContent>): Promise<SurfaceData> {
    this.referenceHost?.close();
    const host = this.createHost();
    this.referenceHost = host;
    const assert = () => {
      if (this.referenceHost !== host)
        throw new AppProblem('stale_revision', 'Notebook reference reading was replaced.');
    };
    try {
      const file = await read();
      assert();
      if (
        file.projectId !== target.projectId ||
        file.resourceId !== target.resource.resourceId ||
        file.revision !== target.dataRevision ||
        file.content.kind !== 'notebook'
      )
        throw new AppProblem('stale_revision', 'The exact Notebook source is unavailable.');
      const bytes = Buffer.from(file.content.base64, 'base64');
      if (
        bytes.toString('base64') !== file.content.base64 ||
        bytes.length !== file.size ||
        'sha256:' + createHash('sha256').update(bytes).digest('hex') !== target.dataRevision
      )
        throw new AppProblem('invalid_request', 'Notebook bytes do not match the source revision.');
      await host.load(bytes);
      assert();
      const document = parse(NotebookDocumentSchema, host.view()),
        capture = host.capture(privateNotebookTarget(target));
      const content =
        capture.representation === 'text/plain'
          ? { kind: 'text' as const, text: capture.text }
          : {
              kind: 'image' as const,
              base64: capture.base64,
              width: capture.width,
              height: capture.height,
            };
      return {
        kind: 'file',
        value: { ...file, content: { kind: 'notebook', document, reference: { target, content } } },
      };
    } catch (error) {
      throw notebookProblem(error);
    } finally {
      host.close();
      if (this.referenceHost === host) this.referenceHost = undefined;
    }
  }
  private current(surfaceId: string, target: NotebookTarget, data: SurfaceData): Entry {
    const key = this.key(target.projectId, surfaceId),
      entry = this.entries.get(key);
    if (
      !entry ||
      entry.busy ||
      !this.allowed.has(key) ||
      entry.source?.revision !== target.dataRevision ||
      entry.source.resourceId !== target.resource.resourceId ||
      data.kind !== 'file' ||
      data.value.content.kind !== 'notebook' ||
      data.value.projectId !== target.projectId ||
      data.value.resourceId !== target.resource.resourceId ||
      data.value.revision !== target.dataRevision
    )
      throw new AppProblem('stale_revision', 'The Notebook changed. Select from its current view.');
    notebookPart(data.value.content.document, target.selection);
    return entry;
  }
  validate(surfaceId: string, target: NotebookTarget, data: SurfaceData): void {
    try {
      this.current(surfaceId, target, data).host.validate(privateNotebookTarget(target));
    } catch (error) {
      throw notebookProblem(error);
    }
  }
  image(surfaceId: string, target: NotebookTarget, data: SurfaceData) {
    try {
      return this.current(surfaceId, target, data).host.image(privateNotebookTarget(target));
    } catch (error) {
      throw notebookProblem(error);
    }
  }
  capture(surfaceId: string, target: NotebookTarget, data: SurfaceData) {
    try {
      return this.current(surfaceId, target, data).host.capture(privateNotebookTarget(target));
    } catch (error) {
      throw notebookProblem(error);
    }
  }
}
function notebookProblem(error: unknown): AppProblem {
  return error instanceof AppProblem
    ? error
    : new AppProblem(
        'unsupported',
        error instanceof Error ? error.message : 'Notebook content is unavailable.',
      );
}
