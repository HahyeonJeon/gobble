import { constants } from 'node:fs';
import { link, lstat, mkdir, open, readdir, unlink } from 'node:fs/promises';
import { dirname, join } from 'node:path';
import { randomUUID } from 'node:crypto';
import { EvidenceHashSchema, ProjectIdSchema, parse } from '@gobble/contracts';
import { AppProblem } from '../problem';
import { contentHash } from './materialize';

type BlobManifest = { asset: { hash: string; byteLength: number } };
type StoredBlob<M extends BlobManifest = BlobManifest> = { manifest: M; bytes: Buffer };

export const EVIDENCE_QUOTA = 64 * 1024 * 1024;
const missing = (error: unknown) =>
  error instanceof Error && 'code' in error && error.code === 'ENOENT';

/** Immutable Project-local blobs. Only the Workspace writer owns references to these bytes. */
export class EvidenceStorage {
  private queues = new Map<string, Promise<unknown>>();
  constructor(
    private readonly directory: string,
    private readonly quota = EVIDENCE_QUOTA,
  ) {}
  private async projectDirectory(projectId: string): Promise<string> {
    parse(ProjectIdSchema, projectId);
    await mkdir(this.directory, { recursive: true, mode: 0o700 });
    if (!(await lstat(this.directory)).isDirectory())
      throw new Error('Invalid evidence directory.');
    const path = join(this.directory, projectId);
    await mkdir(path, { mode: 0o700 }).catch((error) => {
      if (!error || error.code !== 'EEXIST') throw error;
    });
    if (!(await lstat(path)).isDirectory()) throw new Error('Invalid Project evidence directory.');
    for (const parent of [dirname(this.directory), this.directory]) {
      const handle = await open(parent, 'r');
      try {
        await handle.sync();
      } finally {
        await handle.close();
      }
    }
    return path;
  }
  private path(directory: string, hash: string): string {
    parse(EvidenceHashSchema, hash);
    return join(directory, hash.slice(7) + '.blob');
  }
  private async readFile(path: string, manifest: BlobManifest): Promise<Buffer> {
    const file = await open(path, constants.O_RDONLY | constants.O_NOFOLLOW | constants.O_NONBLOCK);
    try {
      const stat = await file.stat();
      if (!stat.isFile() || stat.size !== manifest.asset.byteLength || stat.size > 1024 * 1024)
        throw new Error('Invalid evidence size.');
      const bytes = Buffer.alloc(stat.size + 1);
      let offset = 0;
      while (offset < bytes.length) {
        const { bytesRead } = await file.read(bytes, offset, bytes.length - offset, null);
        if (!bytesRead) break;
        offset += bytesRead;
      }
      const result = bytes.subarray(0, offset);
      if (offset !== stat.size || contentHash(result) !== manifest.asset.hash)
        throw new Error('Invalid evidence hash.');
      return result;
    } finally {
      await file.close();
    }
  }
  async read<M extends BlobManifest>(projectId: string, manifest: M): Promise<StoredBlob<M>> {
    try {
      const directory = await this.projectDirectory(projectId);
      return {
        manifest,
        bytes: await this.readFile(this.path(directory, manifest.asset.hash), manifest),
      };
    } catch {
      throw new AppProblem(
        'internal',
        'Evidence unavailable: its saved content is missing or corrupt. The original record has been preserved.',
      );
    }
  }
  private async check(directory: string, assets: StoredBlob[]): Promise<StoredBlob[]> {
    const entries = await readdir(directory, { withFileTypes: true });
    if (entries.length > 4096)
      throw new AppProblem('unsupported', 'This Project has reached its evidence storage limit.');
    let size = 0;
    for (const entry of entries) {
      if (!entry.isFile())
        throw new AppProblem(
          'internal',
          'Evidence storage contains an unsupported entry. Its files have been preserved.',
        );
      size += (await lstat(join(directory, entry.name))).size;
    }
    const pending: StoredBlob[] = [];
    const hashes = new Set<string>();
    for (const asset of assets) {
      if (
        asset.bytes.length > 1024 * 1024 ||
        asset.bytes.length !== asset.manifest.asset.byteLength ||
        contentHash(asset.bytes) !== asset.manifest.asset.hash
      )
        throw new Error('Invalid prepared bytes.');
      if (hashes.has(asset.manifest.asset.hash)) continue;
      hashes.add(asset.manifest.asset.hash);
      try {
        await this.readFile(this.path(directory, asset.manifest.asset.hash), asset.manifest);
      } catch (error) {
        if (!missing(error))
          throw new AppProblem(
            'internal',
            'Evidence unavailable: an existing asset is corrupt. Its bytes have been preserved.',
          );
        pending.push(asset);
        size += asset.bytes.length;
      }
    }
    if (size > this.quota)
      throw new AppProblem(
        'unsupported',
        'This Project has reached its 64 MiB evidence quota. The content was not saved.',
      );
    return pending;
  }
  async checkCapacity(projectId: string, assets: StoredBlob[]): Promise<void> {
    await this.check(await this.projectDirectory(projectId), assets);
  }
  put(projectId: string, assets: StoredBlob[]): Promise<void> {
    return this.putAssets(projectId, assets, false);
  }
  putCaptured(projectId: string, asset: StoredBlob): Promise<void> {
    return this.putAssets(projectId, [asset], true);
  }
  private putAssets(projectId: string, assets: StoredBlob[], captured: boolean): Promise<void> {
    const operation = (this.queues.get(projectId) ?? Promise.resolve())
      .then(async () => {
        const directory = await this.projectDirectory(projectId);
        const pending = await this.check(directory, assets);
        for (const asset of pending) {
          if (captured) {
            // Track only new capture-owned blobs. Pre-existing historical blobs are never candidates.
            const marker = join(directory, '.capture-' + asset.manifest.asset.hash.slice(7));
            const file = await open(marker, 'wx', 0o600).catch(async (error) => {
              if (error?.code !== 'EEXIST') throw error;
              const stat = await lstat(marker);
              if (!stat.isFile() || stat.size !== 0) throw new Error('Invalid capture marker.');
              return undefined;
            });
            if (file) {
              try {
                await file.sync();
              } finally {
                await file.close();
              }
            }
          }
          const temp = join(directory, '.pending-' + randomUUID());
          const file = await open(temp, 'wx', 0o600);
          try {
            try {
              await file.writeFile(asset.bytes);
              await file.sync();
            } finally {
              await file.close();
            }
            // link is an exclusive publication: an existing immutable hash is never overwritten.
            await link(temp, this.path(directory, asset.manifest.asset.hash));
          } finally {
            await unlink(temp).catch((error) => {
              if (!missing(error)) throw error;
            });
          }
        }
        const handle = await open(directory, 'r');
        try {
          await handle.sync();
        } finally {
          await handle.close();
        }
      })
      .catch((error) => {
        if (error instanceof AppProblem) throw error;
        throw new AppProblem(
          'internal',
          'Evidence could not be saved. Existing records have been preserved.',
        );
      });
    const settled = operation.catch(() => {});
    this.queues.set(projectId, settled);
    void settled.then(() => {
      if (this.queues.get(projectId) === settled) this.queues.delete(projectId);
    });
    return operation;
  }
  /** Caller holds the Workspace queue and supplies all validated primary/backup references. */
  reclaimCaptured(projectId: string, retained: Set<string>): Promise<void> {
    const keep = new Set(retained);
    const operation = (this.queues.get(projectId) ?? Promise.resolve()).then(async () => {
      const directory = await this.projectDirectory(projectId);
      const entries = await readdir(directory, { withFileTypes: true });
      if (entries.length > 8192) throw new Error('Evidence directory exceeds cleanup bounds.');
      for (const entry of entries) {
        if (!/^\.capture-[a-f0-9]{64}$/.test(entry.name)) continue;
        const hash = 'sha256:' + entry.name.slice('.capture-'.length);
        if (keep.has(hash)) continue;
        const marker = join(directory, entry.name);
        const markerStat = await lstat(marker);
        if (!markerStat.isFile() || markerStat.size !== 0)
          throw new Error('Invalid capture marker.');
        const blob = this.path(directory, hash);
        const stat = await lstat(blob).catch((error) => {
          if (missing(error)) return undefined;
          throw error;
        });
        if (stat && !stat.isFile()) throw new Error('Invalid capture blob.');
        if (stat) await unlink(blob);
        await unlink(marker);
      }
      const handle = await open(directory, 'r');
      try {
        await handle.sync();
      } finally {
        await handle.close();
      }
    });
    const settled = operation.catch(() => {});
    this.queues.set(projectId, settled);
    void settled.then(() => {
      if (this.queues.get(projectId) === settled) this.queues.delete(projectId);
    });
    return operation;
  }
}
