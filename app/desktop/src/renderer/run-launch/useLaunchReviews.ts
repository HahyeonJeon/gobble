import { useEffect, useMemo, useSyncExternalStore } from 'react';
import type { LaunchReview } from '@gobble/contracts';

type Snapshot = { values: LaunchReview[]; issue: string };
/** One shared observation per Project. This cache owns no execution decisions or
 * durable state. Chat, the preparation panel and Run Flow see the same read. */
class LaunchReviews {
  private snapshot: Snapshot = { values: [], issue: '' };
  private listeners = new Set<() => void>();
  private generation = 0;
  private timer: ReturnType<typeof setTimeout> | undefined;
  private reconcilers = 0;
  constructor(private readonly projectId: string) {}
  getSnapshot = () => this.snapshot;
  subscribe = (listener: () => void) => {
    this.listeners.add(listener);
    if (this.listeners.size === 1) void this.poll(++this.generation);
    return () => {
      this.listeners.delete(listener);
      if (!this.listeners.size) {
        this.generation++;
        clearTimeout(this.timer);
      }
    };
  };
  reconcile() {
    this.reconcilers++;
    return () => {
      this.reconcilers--;
    };
  }
  private publish(next: Snapshot) {
    this.snapshot = next;
    this.listeners.forEach((f) => f());
  }
  private async poll(generation: number) {
    try {
      const result = await window.gobble.launches.list({ projectId: this.projectId });
      if (generation !== this.generation) return;
      if (result.ok) {
        this.publish({ values: result.value, issue: '' });
        if (this.reconcilers > 0)
          for (const v of result.value) {
            const op = v.operation;
            if (generation !== this.generation) return;
            if (!this.reconcilers) break;
            if (
              op &&
              op.state !== 'rejected' &&
              op.state !== 'recovery-required' &&
              (op.state !== 'admitted' ||
                !['succeeded', 'failed', 'stopped', 'interrupted'].includes(op.runStatus) ||
                (!!op.stopRequestId && op.stopState === 'requested'))
            ) {
              await window.gobble.launches.refresh({
                projectId: this.projectId,
                reviewId: v.requestId,
              });
            }
          }
      } else this.publish({ ...this.snapshot, issue: result.error.message });
    } catch {
      if (generation === this.generation)
        this.publish({ ...this.snapshot, issue: 'Run reviews are unavailable. Reconnecting…' });
    }
    if (generation === this.generation)
      this.timer = setTimeout(() => void this.poll(generation), 3000);
  }
}
const projects = new Map<string, LaunchReviews>();
export function useLaunchReviews(projectId: string, reconcile = false) {
  const store = useMemo(() => {
    let value = projects.get(projectId);
    if (!value) {
      value = new LaunchReviews(projectId);
      projects.set(projectId, value);
    }
    return value;
  }, [projectId]);
  useEffect(() => (reconcile ? store.reconcile() : undefined), [store, reconcile]);
  return useSyncExternalStore(store.subscribe, store.getSnapshot);
}
