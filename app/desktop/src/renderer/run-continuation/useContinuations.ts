import { useMemo, useSyncExternalStore } from 'react';
import type { Continuation } from '@gobble/contracts';

type Snapshot = { values: Continuation[]; issue: string; loaded: boolean };
/** Shared read-only catalog. Mount, refresh and restore never confirm execution. */
class Continuations {
  private snapshot: Snapshot = { values: [], issue: '', loaded: false };
  private listeners = new Set<() => void>();
  private generation = 0;
  private timer: ReturnType<typeof setTimeout> | undefined;
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
  private async poll(generation: number) {
    try {
      const result = await window.gobble.continuations.list({ projectId: this.projectId });
      if (generation !== this.generation) return;
      this.snapshot = result.ok
        ? { values: result.value, issue: '', loaded: true }
        : { ...this.snapshot, issue: result.error.message };
    } catch {
      if (generation !== this.generation) return;
      this.snapshot = {
        ...this.snapshot,
        issue: 'Continuation reviews are unavailable. Reconnecting…',
      };
    }
    this.listeners.forEach((f) => f());
    if (generation === this.generation)
      this.timer = setTimeout(() => void this.poll(generation), 2000);
  }
}
const projects = new Map<string, Continuations>();
export function useContinuations(projectId: string) {
  const store = useMemo(() => {
    let store = projects.get(projectId);
    if (!store) {
      store = new Continuations(projectId);
      projects.set(projectId, store);
    }
    return store;
  }, [projectId]);
  return useSyncExternalStore(store.subscribe, store.getSnapshot);
}
export function latestContinuation(values: Continuation[], runRef: string) {
  return values
    .filter((v) => v.runRef === runRef)
    .sort(
      (a, b) => b.createdAt.localeCompare(a.createdAt) || b.requestId.localeCompare(a.requestId),
    )[0];
}
