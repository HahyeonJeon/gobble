import { useMemo, type CSSProperties, type MouseEvent } from 'react';
import {
  ReactFlow,
  Background,
  Controls,
  MarkerType,
  Position,
  type Node,
  type Edge,
  type NodeChange,
  type EdgeChange,
} from '@xyflow/react';
import dagre from '@dagrejs/dagre';
import '@xyflow/react/dist/style.css';
import {
  dependencyOrder,
  dependencyPairKey,
  type DependencyObservation,
  type DependencyTarget,
} from '../../contracts/src/index';

type Selection = DependencyTarget['selection'];
type Props = {
  observation: DependencyObservation;
  selected: Selection | null;
  select: (selection: Selection) => void;
};
const groupSelection = (id: string): Selection => ({
  kind: 'run-group',
  coordinateSpace: 'observed-authored-task-group',
  taskId: id,
});
const edgeSelection = (from: string, to: string): Selection => ({
  kind: 'run-dependency',
  coordinateSpace: 'observed-authored-task-pair',
  fromTaskId: from,
  toTaskId: to,
});

function simpleLayout(observation: DependencyObservation) {
  const order = dependencyOrder(
    observation.groups.map((g) => g.taskId),
    observation.edges,
  )!;
  const ranks = new Map(order.map((id) => [id, 0]));
  for (const id of order)
    for (const edge of observation.edges)
      if (edge.fromTaskId === id)
        ranks.set(edge.toTaskId, Math.max(ranks.get(edge.toTaskId)!, ranks.get(id)! + 1));
  const row = new Map<number, number>();
  return new Map(
    order.map((id) => {
      const rank = ranks.get(id)!,
        index = row.get(rank) ?? 0;
      row.set(rank, index + 1);
      return [id, { x: 25 + rank * 220, y: 25 + index * 90 }];
    }),
  );
}
export function SimpleDependencies({ observation, selected, select }: Props) {
  const positions = useMemo(() => simpleLayout(observation), [observation]);
  const width = Math.max(...[...positions.values()].map((p) => p.x)) + 190,
    height = Math.max(...[...positions.values()].map((p) => p.y)) + 80;
  return (
    <div
      className="simple-scroll"
      tabIndex={0}
      aria-label="Dependency overview. Scroll to navigate."
    >
      <div className="simple-canvas" style={{ width, height }}>
        <svg width={width} height={height} aria-label="Dependency connections">
          <defs>
            <marker
              id="arrow"
              viewBox="0 0 10 10"
              refX="9"
              refY="5"
              markerWidth="7"
              markerHeight="7"
              orient="auto-start-reverse"
            >
              <path d="M 0 0 L 10 5 L 0 10 z" fill="#6b837b" />
            </marker>
          </defs>
          {observation.edges.map((edge) => {
            const a = positions.get(edge.fromTaskId)!,
              b = positions.get(edge.toTaskId)!,
              key = dependencyPairKey(edge);
            const hit =
              selected?.kind === 'run-dependency' &&
              selected.fromTaskId === edge.fromTaskId &&
              selected.toTaskId === edge.toTaskId;
            return (
              <path
                key={key}
                data-edge-id={key}
                className={hit ? 'selected-edge' : 'dependency-edge'}
                d={`M${a.x + 160},${a.y + 30} C${a.x + 190},${a.y + 30} ${b.x - 30},${b.y + 30} ${b.x},${b.y + 30}`}
                markerEnd="url(#arrow)"
                onClick={() => select(edgeSelection(edge.fromTaskId, edge.toTaskId))}
              />
            );
          })}
        </svg>
        {observation.groups.map((group) => {
          const position = positions.get(group.taskId)!;
          return (
            <button
              key={group.taskId}
              data-task-id={group.taskId}
              className="task-node"
              aria-pressed={selected?.kind === 'run-group' && selected.taskId === group.taskId}
              style={{ left: position.x, top: position.y } as CSSProperties}
              onFocus={(event) =>
                event.currentTarget.scrollIntoView({ block: 'nearest', inline: 'nearest' })
              }
              onClick={() => select(groupSelection(group.taskId))}
            >
              <strong>{group.taskId}</strong>
              <small>{group.members.length} observed instance</small>
            </button>
          );
        })}
      </div>
    </div>
  );
}
export function FlowDependencies({ observation, selected, select }: Props) {
  const geometry = useMemo(() => {
    const graph = new dagre.graphlib.Graph()
      .setGraph({ rankdir: 'LR', ranksep: 60, nodesep: 26 })
      .setDefaultEdgeLabel(() => ({}));
    for (const group of observation.groups) graph.setNode(group.taskId, { width: 160, height: 60 });
    for (const edge of observation.edges) graph.setEdge(edge.fromTaskId, edge.toTaskId);
    dagre.layout(graph);
    return new Map(
      observation.groups.map((group) => {
        const p = graph.node(group.taskId);
        return [group.taskId, { x: p.x - 80, y: p.y - 30 }];
      }),
    );
  }, [observation]);
  const nodes: Node[] = observation.groups.map((group) => ({
    id: group.taskId,
    type: 'default',
    data: {
      label: (
        <span>
          <strong>{group.taskId}</strong>
          <small>{group.members.length} observed instance</small>
        </span>
      ),
    },
    position: geometry.get(group.taskId)!,
    sourcePosition: Position.Right,
    targetPosition: Position.Left,
    draggable: false,
    connectable: false,
    selected: selected?.kind === 'run-group' && selected.taskId === group.taskId,
    style: { width: 160, height: 60 },
    ariaLabel: 'Task group ' + group.taskId,
  }));
  const edges: Edge[] = observation.edges.map((edge) => ({
    id: dependencyPairKey(edge),
    source: edge.fromTaskId,
    target: edge.toTaskId,
    selected:
      selected?.kind === 'run-dependency' &&
      selected.fromTaskId === edge.fromTaskId &&
      selected.toTaskId === edge.toTaskId,
    markerEnd: { type: MarkerType.ArrowClosed },
    ariaLabel: edge.fromTaskId + ' to ' + edge.toTaskId,
    interactionWidth: 20,
  }));
  return (
    <ReactFlow
      nodes={nodes}
      edges={edges}
      nodesDraggable={false}
      nodesConnectable={false}
      edgesReconnectable={false}
      onNodesChange={(changes: NodeChange[]) => {
        const chosen = changes.find((change) => change.type === 'select');
        if (chosen?.selected) select(groupSelection(chosen.id));
      }}
      onEdgesChange={(changes: EdgeChange[]) => {
        const chosen = changes.find((change) => change.type === 'select');
        const edge = observation.edges.find((edge) => dependencyPairKey(edge) === chosen?.id);
        if (chosen?.selected && edge) select(edgeSelection(edge.fromTaskId, edge.toTaskId));
      }}
      deleteKeyCode={null}
      selectionOnDrag={false}
      nodesFocusable
      edgesFocusable
      minZoom={0.4}
      maxZoom={2}
      onNodeClick={(_event: MouseEvent, node: Node) => select(groupSelection(node.id))}
      onEdgeClick={(_event: MouseEvent, edge: Edge) =>
        select(edgeSelection(edge.source, edge.target))
      }
      ariaLabelConfig={{
        'node.a11yDescription.default':
          'Press Enter to select this Task group. This graph is read-only.',
        'node.a11yDescription.keyboardDisabled': 'This Task group is read-only.',
        'edge.a11yDescription.default':
          'Press Enter to select this dependency. This graph is read-only.',
      }}
    >
      <Background />
      <Controls showInteractive={false} />
    </ReactFlow>
  );
}
