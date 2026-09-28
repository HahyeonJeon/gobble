import { resourceKey, type SurfaceData, type ViewLink } from '@gobble/contracts';
import { AppProblem } from '../problem';

/** One bounded immutable load per visible link revision; no additional persisted data owner. */
export class TabularSnapshots {
  private readonly entries = new Map<string, Promise<SurfaceData>>();
  private key(projectId: string, link: ViewLink): string {
    return JSON.stringify([projectId, link.linkId, resourceKey(link.resource), link.dataRevision]);
  }
  retain(projectId: string, visibleLinks: ViewLink[]): void {
    const keep = new Set(visibleLinks.map((link) => this.key(projectId, link)));
    for (const key of this.entries.keys()) if (!keep.has(key)) this.entries.delete(key);
  }
  clear(): void {
    this.entries.clear();
  }
  /** Seed the committed revision with the exact data already validated by the command. */
  seed(projectId: string, link: ViewLink, data: SurfaceData): void {
    if (
      data.kind !== 'file' ||
      data.value.projectId !== projectId ||
      data.value.resourceId !== link.resource.resourceId ||
      data.value.revision !== link.dataRevision ||
      data.value.content.kind !== 'table'
    )
      throw new AppProblem(
        'stale_revision',
        'The linked snapshot does not match the committed source.',
      );
    this.entries.set(this.key(projectId, link), Promise.resolve(structuredClone(data)));
  }
  async read(
    projectId: string,
    link: ViewLink,
    load: () => Promise<SurfaceData>,
  ): Promise<SurfaceData> {
    const key = this.key(projectId, link);
    let pending = this.entries.get(key);
    if (!pending) {
      pending = load().then((data) => {
        if (
          data.kind !== 'file' ||
          data.value.projectId !== projectId ||
          data.value.resourceId !== link.resource.resourceId ||
          data.value.revision !== link.dataRevision ||
          data.value.content.kind !== 'table'
        )
          throw new AppProblem(
            'stale_revision',
            'The linked source changed. Reopen its current revision.',
          );
        return structuredClone(data);
      });
      this.entries.set(key, pending);
    }
    try {
      return structuredClone(await pending);
    } catch (error) {
      if (this.entries.get(key) === pending) this.entries.delete(key);
      throw error;
    }
  }
}
