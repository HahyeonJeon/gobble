import type { PipelineFlow } from '@gobble/contracts';
import {
  inputKey,
  stepKey,
  nodeWidth,
  nodeHeight,
  columnGap,
  type FlowLayout,
} from './flow-layout';

type Point = { x: number; y: number };

/** Round only the corners of an orthogonal polyline; never interpolate a diagonal. */
function roundedPath(points: Point[]): string {
  let path = `M ${points[0]!.x} ${points[0]!.y}`;
  for (let index = 1; index < points.length - 1; index++) {
    const before = points[index - 1]!;
    const corner = points[index]!;
    const after = points[index + 1]!;
    const incoming = Math.abs(corner.x - before.x) + Math.abs(corner.y - before.y);
    const outgoing = Math.abs(after.x - corner.x) + Math.abs(after.y - corner.y);
    const radius = Math.min(12, incoming / 2, outgoing / 2);
    if (!radius) continue;
    const start = {
      x: corner.x + ((before.x - corner.x) / incoming) * radius,
      y: corner.y + ((before.y - corner.y) / incoming) * radius,
    };
    const end = {
      x: corner.x + ((after.x - corner.x) / outgoing) * radius,
      y: corner.y + ((after.y - corner.y) / outgoing) * radius,
    };
    path += ` L ${start.x} ${start.y} Q ${corner.x} ${corner.y} ${end.x} ${end.y}`;
  }
  const end = points.at(-1)!;
  return `${path} L ${end.x} ${end.y}`;
}

/** Routes only existing artifact edges. Column gutters and row corridors keep
 * long connections out of cards; each edge retains its own identity/hit target. */
export function flowRoutes(flow: PipelineFlow, layout: FlowLayout) {
  const nodes = new Map(layout.nodes.map((node) => [node.key, node]));
  const endpoints = flow.connections.map((edge) => ({
    edge,
    from: nodes.get(edge.fromTask ? stepKey(edge.fromTask) : inputKey(edge.fromPort))!,
    to: nodes.get(stepKey(edge.toTask))!,
  }));
  const outgoing = new Map<string, string[]>([
    ...flow.inputs.map((port): [string, string[]] => [inputKey(port.name), [port.name]]),
    ...flow.steps.map((step): [string, string[]] => [
      stepKey(step.id),
      step.outputs.map((port) => port.name),
    ]),
  ]);
  const incoming = new Map(
    flow.steps.map((step) => [stepKey(step.id), step.inputs.map((port) => port.name)]),
  );
  const leavingColumn = new Map<number, string[]>();
  const enteringColumn = new Map<number, string[]>();
  for (const { edge, from, to } of endpoints) {
    leavingColumn.set(from.rank, [...(leavingColumn.get(from.rank) ?? []), edge.id]);
    enteringColumn.set(to.rank, [...(enteringColumn.get(to.rank) ?? []), edge.id]);
  }
  // First declared port follows the center spine; additional named ports
  // alternate below/above it. Edges at one port share one real anchor.
  const offset = (ports: string[], port: string) => {
    const index = ports.indexOf(port);
    return Math.ceil(index / 2) * (index % 2 === 1 ? 1 : -1) * Math.min(16, 64 / ports.length);
  };
  const laneOffset = (ids: string[], id: string) =>
    16 + ((ids.indexOf(id) + 1) / (ids.length + 1)) * (columnGap / 2 - 20);
  const corridors = new Map<number, number>();
  let extraLane = 0;
  let height = layout.height;
  const edges = endpoints.map(({ edge, from, to }) => {
    const start = {
      x: from.x + nodeWidth,
      y: from.y + nodeHeight / 2 + offset(outgoing.get(from.key)!, edge.fromPort),
    };
    const end = {
      x: to.x,
      y: to.y + nodeHeight / 2 + offset(incoming.get(to.key)!, edge.toPort),
    };
    const leaveX = start.x + laneOffset(leavingColumn.get(from.rank)!, edge.id);
    const enterX = end.x - laneOffset(enteringColumn.get(to.rank)!, edge.id);
    let points: Point[];
    if (to.rank === from.rank + 1) {
      points = [start, { x: leaveX, y: start.y }, { x: leaveX, y: end.y }, end];
    } else {
      const row = Math.max(from.row, to.row);
      const lane = corridors.get(row) ?? 0;
      corridors.set(row, lane + 1);
      // Four lanes fit in a row gap; additional long edges get their own
      // corridor below the diagram instead of running through a card.
      const y =
        lane < 4
          ? Math.max(from.y, to.y) + nodeHeight + 12 + lane * 10
          : layout.height + 12 + extraLane++ * 12;
      height = Math.max(height, y + 24);
      points = [
        start,
        { x: leaveX, y: start.y },
        { x: leaveX, y },
        { x: enterX, y },
        { x: enterX, y: end.y },
        end,
      ];
    }
    points = points.filter(
      (point, index) =>
        index === 0 || point.x !== points[index - 1]!.x || point.y !== points[index - 1]!.y,
    );
    return { id: edge.id, start, end, points, path: roundedPath(points) };
  });
  return { edges, width: layout.width, height };
}
