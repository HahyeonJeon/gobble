import type { PipelineFlow, PipelineStep } from '@gobble/contracts';

export type FlowTarget =
  | { kind: 'step' | 'input' | 'connection'; id: string }
  | { kind: 'port'; id: string; name: string; direction: 'input' | 'output' }
  | { kind: 'setting'; id: string; name: string };
export const stepLabel = (step: PipelineStep) => step.display.stage || step.name;
export const inputKey = (name: string) => JSON.stringify(['input', name]);
export const stepKey = (id: string) => JSON.stringify(['step', id]);

export const nodeWidth = 216;
export const nodeHeight = 104;
export const columnGap = 72;
export const rowGap = 56;

/** Geometry only. All semantic relationships and endpoints come from checked facts. */
export function flowLayout(flow: PipelineFlow) {
  const nodes = [
    ...flow.inputs.map((input) => ({
      key: inputKey(input.name),
      target: { kind: 'input' as const, id: input.name },
      label: input.name,
      detail: input.kind,
    })),
    ...flow.steps.map((step) => ({
      key: stepKey(step.id),
      target: { kind: 'step' as const, id: step.id },
      label: stepLabel(step),
      detail:
        step.display.samples?.join(', ') ||
        `${step.inputs.length} input${step.inputs.length === 1 ? '' : 's'} · ${step.outputs.length} output${step.outputs.length === 1 ? '' : 's'}`,
    })),
  ];
  const incoming = new Map(nodes.map((node) => [node.key, 0]));
  const next = new Map<string, string[]>();
  const ranks = new Map(nodes.map((node) => [node.key, 0]));
  for (const edge of flow.connections) {
    const from = edge.fromTask ? stepKey(edge.fromTask) : inputKey(edge.fromPort);
    const to = stepKey(edge.toTask);
    next.set(from, [...(next.get(from) ?? []), to]);
    incoming.set(to, incoming.get(to)! + 1);
  }
  const queue = nodes.filter((node) => incoming.get(node.key) === 0).map((node) => node.key);
  for (let index = 0; index < queue.length; index++)
    for (const to of next.get(queue[index]!) ?? []) {
      ranks.set(to, Math.max(ranks.get(to)!, ranks.get(queue[index]!)! + 1));
      incoming.set(to, incoming.get(to)! - 1);
      if (incoming.get(to) === 0) queue.push(to);
    }
  const rows = new Map<number, number>();
  const placed = nodes.map((node) => {
    const rank = ranks.get(node.key)!;
    const row = rows.get(rank) ?? 0;
    rows.set(rank, row + 1);
    return {
      ...node,
      rank,
      row,
      x: 28 + rank * (nodeWidth + columnGap),
      y: 32 + row * (nodeHeight + rowGap),
    };
  });
  return {
    nodes: placed,
    width: Math.max(420, ...placed.map((node) => node.x + nodeWidth + 28)),
    height: Math.max(280, ...placed.map((node) => node.y + nodeHeight + rowGap)),
  };
}

export type FlowLayout = ReturnType<typeof flowLayout>;
