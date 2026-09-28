import { dependencyOrder, type DependencyObservation } from '@gobble/contracts';
export const NODE_WIDTH = 180;
export const NODE_HEIGHT = 70;
export type DependencyLayout = {
  positions: Map<string, { x: number; y: number }>;
  width: number;
  height: number;
};
/** Bounded read-only geometry. No identifiers or execution facts are derived from coordinates. */
export function layoutDependencies(observation: DependencyObservation): DependencyLayout {
  const order = dependencyOrder(
    observation.groups.map((g) => g.taskId),
    observation.edges,
  );
  if (!order) return { positions: new Map(), width: 1, height: 1 };
  const ranks = new Map(order.map((id) => [id, 0]));
  const next = new Map(order.map((id) => [id, [] as string[]]));
  for (const edge of observation.edges) next.get(edge.fromTaskId)!.push(edge.toTaskId);
  for (const id of order)
    for (const to of next.get(id)!) ranks.set(to, Math.max(ranks.get(to)!, ranks.get(id)! + 1));
  const rows = new Map<number, number>();
  const positions = new Map(
    order.map((id) => {
      const rank = ranks.get(id)!,
        row = rows.get(rank) ?? 0;
      rows.set(rank, row + 1);
      return [id, { x: 24 + rank * 240, y: 24 + row * 100 }] as const;
    }),
  );
  return {
    positions,
    width: Math.max(0, ...[...positions.values()].map((p) => p.x)) + NODE_WIDTH + 24,
    height: Math.max(0, ...[...positions.values()].map((p) => p.y)) + NODE_HEIGHT + 24,
  };
}
