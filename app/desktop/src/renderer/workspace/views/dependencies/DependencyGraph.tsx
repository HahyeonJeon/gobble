import { dependencyMarks, DependencyMarkLabel } from './dependency-marks';
import { useEffect, useId, useLayoutEffect, useMemo, useRef } from 'react';
import {
  dependencyPairKey,
  type DependencyCamera,
  type DependencyObservation,
  type Selection,
  type SharedReference,
} from '@gobble/contracts';
import { layoutDependencies, NODE_HEIGHT, NODE_WIDTH } from './dependency-layout';

type Props = {
  observation: DependencyObservation;
  marks?: SharedReference[];
  readOnly?: boolean;
  selection: Selection | undefined;
  camera: DependencyCamera;
  ready: boolean;
  reveal: number;
  onCamera: (camera: DependencyCamera) => void;
  onSelect: (selection: Selection) => void;
};
export function DependencyGraph({
  observation,
  marks = [],
  readOnly = false,
  selection,
  camera,
  ready,
  reveal,
  onCamera,
  onSelect,
}: Props) {
  const viewport = useRef<HTMLDivElement>(null);
  const pending = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);
  const lastReveal = useRef(0);
  const marker = useId().replaceAll(':', '');
  const layout = useMemo(() => layoutDependencies(observation), [observation]);
  useLayoutEffect(() => {
    const element = viewport.current;
    if (element) {
      element.scrollLeft = camera.x * camera.zoom;
      element.scrollTop = camera.y * camera.zoom;
    }
  }, [camera.x, camera.y, camera.zoom]);
  useEffect(() => {
    if (!ready) clearTimeout(pending.current);
  }, [ready]);
  useEffect(() => () => clearTimeout(pending.current), []);
  useEffect(() => {
    if (reveal === lastReveal.current) return;
    lastReveal.current = reveal;
    if (!reveal || !viewport.current || !selection) return;
    const id =
      selection.kind === 'run-group'
        ? selection.taskId
        : selection.kind === 'run-dependency'
          ? selection.fromTaskId
          : selection.kind === 'run-task'
            ? observation.groups.find((g) =>
                g.members.some((m) => m.instanceId === selection.instanceId),
              )?.taskId
            : undefined;
    const position = id ? layout.positions.get(id) : undefined;
    if (position) {
      viewport.current.scrollLeft = Math.max(0, position.x * camera.zoom - 24);
      viewport.current.scrollTop = Math.max(0, position.y * camera.zoom - 24);
    }
  }, [reveal, selection, observation, layout, camera.zoom]); // Only the explicit token moves the camera.
  const zoom = (value: number) => {
    clearTimeout(pending.current);
    onCamera({ ...camera, zoom: Math.min(2, Math.max(0.25, value)) });
  };
  const chosenGroup =
    selection?.kind === 'run-task'
      ? observation.groups.find((g) => g.members.some((m) => m.instanceId === selection.instanceId))
          ?.taskId
      : selection?.kind === 'run-group'
        ? selection.taskId
        : undefined;
  return (
    <div className="dependency-graph">
      <div className="dependency-camera" role="group" aria-label="Graph navigation">
        <span>Scroll to navigate</span>
        <button
          aria-label="Zoom out dependencies"
          disabled={!ready || camera.zoom <= 0.25}
          onClick={() => zoom(camera.zoom - 0.25)}
        >
          −
        </button>
        <output aria-label="Dependency zoom">{Math.round(camera.zoom * 100)}%</output>
        <button
          aria-label="Zoom in dependencies"
          disabled={!ready || camera.zoom >= 2}
          onClick={() => zoom(camera.zoom + 0.25)}
        >
          +
        </button>
        <button
          disabled={!ready}
          onClick={() => {
            const element = viewport.current;
            if (element) {
              clearTimeout(pending.current);
              onCamera({
                zoom: Math.max(
                  0.25,
                  Math.min(
                    1,
                    element.clientWidth / layout.width,
                    element.clientHeight / layout.height,
                  ),
                ),
                x: 0,
                y: 0,
              });
            }
          }}
        >
          Fit graph
        </button>
      </div>
      <div
        ref={viewport}
        className="dependency-viewport"
        tabIndex={0}
        aria-label="Dependency graph. Scroll or use arrow keys to navigate."
        onScroll={(event) => {
          if (!ready) return;
          const element = event.currentTarget;
          const next = {
            zoom: camera.zoom,
            x: element.scrollLeft / camera.zoom,
            y: element.scrollTop / camera.zoom,
          };
          clearTimeout(pending.current);
          if (Math.abs(next.x - camera.x) + Math.abs(next.y - camera.y) > 1)
            pending.current = setTimeout(() => onCamera(next), 180);
        }}
      >
        <div style={{ width: layout.width * camera.zoom, height: layout.height * camera.zoom }}>
          <div
            className="dependency-canvas"
            style={{
              width: layout.width,
              height: layout.height,
              transform: `scale(${camera.zoom})`,
            }}
          >
            <svg width={layout.width} height={layout.height} aria-hidden="true">
              <defs>
                <marker
                  id={marker}
                  viewBox="0 0 10 10"
                  refX="9"
                  refY="5"
                  markerWidth="10"
                  markerHeight="10"
                  markerUnits="userSpaceOnUse"
                  orient="auto"
                >
                  <path d="M 0 0 L 10 5 L 0 10 z" fill="currentColor" />
                </marker>
              </defs>
              {observation.edges.map((edge) => {
                const a = layout.positions.get(edge.fromTaskId),
                  b = layout.positions.get(edge.toTaskId);
                if (!a || !b) return null;
                const d = `M${a.x + NODE_WIDTH},${a.y + NODE_HEIGHT / 2} C${a.x + NODE_WIDTH + 32},${a.y + NODE_HEIGHT / 2} ${b.x - 32},${b.y + NODE_HEIGHT / 2} ${b.x},${b.y + NODE_HEIGHT / 2}`;
                const selected =
                  selection?.kind === 'run-dependency' &&
                  selection.fromTaskId === edge.fromTaskId &&
                  selection.toTaskId === edge.toTaskId;
                const marked = dependencyMarks(marks, {
                  kind: 'run-dependency',
                  coordinateSpace: 'observed-authored-task-pair',
                  ...edge,
                });
                return (
                  <g
                    key={dependencyPairKey(edge)}
                    data-selected={selected && !readOnly}
                    data-agent-mark={marked.length > 0}
                  >
                    <path className="dependency-line" d={d} markerEnd={`url(#${marker})`} />
                    <path
                      className="dependency-hit"
                      d={d}
                      onClick={() =>
                        ready &&
                        !readOnly &&
                        onSelect({
                          kind: 'run-dependency',
                          coordinateSpace: 'observed-authored-task-pair',
                          ...edge,
                        })
                      }
                    />
                  </g>
                );
              })}
            </svg>
            {observation.groups.map((group) => {
              const position = layout.positions.get(group.taskId);
              const marked = dependencyMarks(marks, {
                kind: 'run-group',
                coordinateSpace: 'observed-authored-task-group',
                taskId: group.taskId,
              });
              return (
                position && (
                  <button
                    key={group.taskId}
                    className="dependency-node"
                    data-group-id={group.taskId}
                    aria-label={`Task group ${group.taskId}`}
                    aria-pressed={!readOnly && chosenGroup === group.taskId}
                    data-agent-mark={marked.length > 0}
                    aria-disabled={readOnly}
                    disabled={!ready}
                    style={{
                      left: position.x,
                      top: position.y,
                      width: NODE_WIDTH,
                      height: NODE_HEIGHT,
                    }}
                    onFocus={(event) =>
                      event.currentTarget.scrollIntoView({ block: 'nearest', inline: 'nearest' })
                    }
                    onClick={() =>
                      !readOnly &&
                      onSelect({
                        kind: 'run-group',
                        coordinateSpace: 'observed-authored-task-group',
                        taskId: group.taskId,
                      })
                    }
                  >
                    <strong title={group.taskId}>{group.taskId}</strong>
                    <small>
                      {group.members.length} observed{' '}
                      {group.members.length === 1 ? 'instance' : 'instances'}
                    </small>
                    <DependencyMarkLabel marks={marked} />
                  </button>
                )
              );
            })}
          </div>
        </div>
      </div>
    </div>
  );
}
