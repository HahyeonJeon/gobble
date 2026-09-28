import { constants } from 'node:fs';
import { mkdir, open, rename, unlink, readdir } from 'node:fs/promises';
import { dirname, join } from 'node:path';
import { randomUUID } from 'node:crypto';
import {
  parse,
  readStoredWorkspace,
  ProjectIdSchema,
  WindowStateSchema,
  type WorkspaceDocument,
  type WindowState,
} from '@gobble/contracts';
import { AppProblem } from '../problem';

const limit = 4 * 1024 * 1024;
const missing = (error: unknown) =>
  error instanceof Error && 'code' in error && error.code === 'ENOENT';

async function readBounded(path: string): Promise<string | undefined> {
  let file;
  try {
    file = await open(path, constants.O_RDONLY | constants.O_NOFOLLOW | constants.O_NONBLOCK);
    const stat = await file.stat();
    if (!stat.isFile() || stat.size > limit) throw new Error('Unsupported state file.');
    const buffer = Buffer.alloc(Math.min(stat.size + 1, limit + 1));
    let length = 0;
    while (length < buffer.length) {
      const { bytesRead } = await file.read(buffer, length, buffer.length - length, null);
      if (!bytesRead) break;
      length += bytesRead;
    }
    if (length > limit) throw new Error('State file exceeds its limit.');
    const after = await file.stat();
    if (after.size !== stat.size || after.mtimeMs !== stat.mtimeMs || length !== stat.size)
      throw new Error('State changed while being read.');
    return new TextDecoder('utf-8', { fatal: true }).decode(buffer.subarray(0, length));
  } catch (error) {
    if (missing(error)) return undefined;
    throw error;
  } finally {
    await file?.close();
  }
}

async function atomicWrite(path: string, value: string): Promise<void> {
  const temp = join(dirname(path), '.workspace-' + randomUUID());
  const file = await open(temp, 'wx', 0o600);
  try {
    try {
      await file.writeFile(value, 'utf8');
      await file.sync();
    } finally {
      await file.close();
    }
    await rename(temp, path);
    const directory = await open(dirname(path), 'r');
    try {
      await directory.sync();
    } finally {
      await directory.close();
    }
  } finally {
    await unlink(temp).catch((error) => {
      if (!missing(error)) throw error;
    });
  }
}

// One instance per state file, called only through the controller's serialized queue.
export class AtomicStateFile<T> {
  private loaded = false;
  private previous: string | undefined;
  private failed = false;
  private migrationSuffix: string | undefined;

  constructor(
    private readonly path: string,
    private readonly validate: (value: unknown) => T,
    private readonly preserveMigration?: (value: unknown) => string | undefined,
  ) {}

  async read(): Promise<T | undefined> {
    if (this.failed)
      throw new AppProblem(
        'internal',
        'Workspace storage is unavailable. Restart the app to retry.',
      );
    try {
      const raw = await readBounded(this.path);
      if (raw === undefined && (await readBounded(this.path + '.backup')) !== undefined)
        throw new Error('Primary state is missing.');
      const input: unknown = raw === undefined ? undefined : JSON.parse(raw);
      const value = raw === undefined ? undefined : this.validate(input);
      this.migrationSuffix = raw === undefined ? undefined : this.preserveMigration?.(input);
      this.previous = raw;
      this.loaded = true;
      return value;
    } catch {
      this.failed = true;
      throw new AppProblem(
        'internal',
        'Saved workspace is unavailable. Its files have been preserved for recovery.',
      );
    }
  }

  async write(value: T): Promise<void> {
    if (this.failed)
      throw new AppProblem(
        'internal',
        'Workspace storage is unavailable. Restart the app to retry.',
      );
    if (!this.loaded) await this.read();
    const raw = JSON.stringify(this.validate(value)) + '\n';
    if (Buffer.byteLength(raw) > limit)
      throw new AppProblem('unsupported', 'This workspace has reached its storage limit.');
    if (raw === this.previous) return;
    try {
      if ((await readBounded(this.path)) !== this.previous)
        throw new Error('State was changed outside this app.');
      await mkdir(dirname(this.path), { recursive: true, mode: 0o700 });
      if (this.previous !== undefined && this.migrationSuffix) {
        const archive = this.path + this.migrationSuffix;
        const existing = await readBounded(archive);
        if (existing !== undefined && existing !== this.previous)
          throw new Error('A different migration backup already exists.');
        if (existing === undefined) {
          // Exclusive creation preserves external archives and makes a failed partial backup explicit.
          const backup = await open(archive, 'wx', 0o600);
          try {
            await backup.writeFile(this.previous, 'utf8');
            await backup.sync();
          } finally {
            await backup.close();
          }
        }
      }
      if (this.previous !== undefined) await atomicWrite(this.path + '.backup', this.previous);
      await atomicWrite(this.path, raw);
      this.previous = raw;
      this.migrationSuffix = undefined;
    } catch {
      this.failed = true;
      throw new AppProblem(
        'internal',
        'Changes could not be saved. Keep this app open and preserve the workspace files.',
      );
    }
  }
  async assertUnchanged(): Promise<void> {
    if (!this.loaded || this.failed || (await readBounded(this.path)) !== this.previous)
      throw new Error('Workspace storage changed; preserve captured assets.');
  }
}

export class WorkspaceStorage {
  private readonly projects = new Map<string, AtomicStateFile<WorkspaceDocument>>();
  readonly window: AtomicStateFile<WindowState>;
  constructor(private readonly directory: string) {
    this.window = new AtomicStateFile(join(directory, 'window.json'), (value) =>
      parse(WindowStateSchema, value),
    );
  }
  project(projectId: string): AtomicStateFile<WorkspaceDocument> {
    parse(ProjectIdSchema, projectId);
    let file = this.projects.get(projectId);
    if (!file) {
      file = new AtomicStateFile(
        join(this.directory, 'projects', projectId + '.json'),
        (value) => {
          const doc = readStoredWorkspace(value);
          if (doc.workspace.projectId !== projectId) throw new Error('Foreign workspace.');
          return doc;
        },
        (value) =>
          value !== null &&
          typeof value === 'object' &&
          'schemaVersion' in value &&
          (value.schemaVersion === 1 ||
            value.schemaVersion === 2 ||
            value.schemaVersion === 3 ||
            value.schemaVersion === 4 ||
            value.schemaVersion === 5 ||
            value.schemaVersion === 6 ||
            value.schemaVersion === 7 ||
            value.schemaVersion === 8 ||
            value.schemaVersion === 9 ||
            value.schemaVersion === 10 ||
            value.schemaVersion === 11 ||
            value.schemaVersion === 12 ||
            value.schemaVersion === 13 ||
            value.schemaVersion === 14 ||
            value.schemaVersion === 15 ||
            value.schemaVersion === 16 ||
            value.schemaVersion === 17 ||
            value.schemaVersion === 18 ||
            value.schemaVersion === 19 ||
            value.schemaVersion === 20)
            ? `.v${value.schemaVersion}.backup`
            : undefined,
      );
      this.projects.set(projectId, file);
    }
    return file;
  }
  /** Read-only recovery roots. Unknown/corrupt/missing primary state forbids reclamation. */
  async retainedDocuments(projectId: string): Promise<WorkspaceDocument[]> {
    const state = this.project(projectId);
    await state.assertUnchanged();
    const directory = join(this.directory, 'projects');
    const prefix = projectId + '.json';
    const names = (await readdir(directory))
      .filter(
        (name) => name === prefix || (name.startsWith(prefix + '.') && name.endsWith('.backup')),
      )
      .sort();
    if (!names.includes(prefix) || names.length > 32) throw new Error('Incomplete recovery roots.');
    const documents: WorkspaceDocument[] = [];
    for (const name of names) {
      const raw = await readBounded(join(directory, name));
      if (raw === undefined) throw new Error('A recovery root disappeared.');
      const doc = readStoredWorkspace(JSON.parse(raw));
      if (doc.workspace.projectId !== projectId) throw new Error('Foreign recovery root.');
      documents.push(doc);
    }
    await state.assertUnchanged();
    return documents;
  }
}
