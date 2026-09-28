import type { FlowNavigation } from './flow-navigation';
import { useLayoutEffect, useEffect, useId, useMemo, useRef, useState } from 'react';
import type { PipelineFlow as Flow } from '@gobble/contracts';
import { flowLayout, stepLabel, type FlowTarget } from './flow-layout';
import { connectionLabel, isFlowParent } from './flow-selection';
import { flowRoutes } from './flow-routing';
import { PipelineNode } from './PipelineNode';

export function PipelineFlow({
  flow,
  selected,
  onSelect,
  list,
  marks = [],
  navigation,
  changes,
  creation = false,
  statuses,
  plan,
  selectableSteps,
}: {
  creation?: boolean;
  statuses?: Record<string, string>;
  plan?: Record<string, { action: string; label: string }> | undefined;
  selectableSteps?: string[];
  flow: Flow;
  selected: FlowTarget | null;
  onSelect: (value: FlowTarget) => void;
  list: boolean;
  navigation?: FlowNavigation;
  changes?: Array<{ stepId: string; kind: string; number: number }>;
  marks?: Array<{ target: FlowTarget; label: string; author: 'agent' | 'user' }>;
}) {
  const layout = useMemo(() => flowLayout(flow), [flow]);
  const routes = useMemo(() => flowRoutes(flow, layout), [flow, layout]);
  const viewport = useRef<HTMLDivElement>(null);
  const [fit, setFit] = useState(1);
  const [zoom, setZoomState] = useState<number | null>(navigation?.zoom ?? null);
  const setZoom = (value: number | null) => {
    if (navigation) navigation.zoom = value;
    setZoomState(value);
  };
  useEffect(() => {
    const element = viewport.current;
    if (!element) return;
    const update = () =>
      setFit(
        Math.min(
          1,
          Math.max(
            0.25,
            Math.min(element.clientWidth / routes.width, element.clientHeight / routes.height),
          ),
        ),
      );
    const observer = new ResizeObserver(update);
    observer.observe(element);
    update();
    return () => observer.disconnect();
  }, [list, routes.width, routes.height]);
  const scale = zoom ?? fit;
  useLayoutEffect(() => {
    if (viewport.current && navigation) {
      viewport.current.scrollLeft = navigation.left;
      viewport.current.scrollTop = navigation.top;
    }
  }, [list, scale, navigation]);
  const marker = useId().replaceAll(':', '');
  const active = (kind: FlowTarget['kind'], id: string) =>
    !!selected && isFlowParent(selected, kind, id);
  const marked = (kind: FlowTarget['kind'], id: string) =>
    marks.filter((m) => isFlowParent(m.target, kind, id));
  // Comparison mode exposes only targets that open a checked change.
  const addedStepFor = (edgeId: string) => {
    const edge = flow.connections.find((connection) => connection.id === edgeId);
    return changes?.find(
      (change) => change.kind === 'added-step' && change.stepId === edge?.toTask,
    );
  };
  const selectEdge = (edgeId: string) => {
    const added = addedStepFor(edgeId);
    onSelect(
      added && !creation ? { kind: 'step', id: added.stepId } : { kind: 'connection', id: edgeId },
    );
  };
  if (list)
    return (
      <div className="pipeline-step-list" aria-label="Connected steps">
        {flow.inputs.length > 0 && (
          <section>
            <h3>Pipeline inputs</h3>
            {flow.inputs.map((input) => (
              <button
                disabled={!!changes && !creation}
                key={input.name}
                aria-pressed={active('input', input.name)}
                onClick={() => onSelect({ kind: 'input', id: input.name })}
              >
                {input.name}
                {marked('input', input.name).map((m) => (
                  <small key={m.label}> · {m.label} mark</small>
                ))}
                <span>{input.kind}</span>
              </button>
            ))}
          </section>
        )}
        {flow.steps.map((step) => (
          <section key={step.id}>
            <button
              disabled={!!changes && !changes.some((change) => change.stepId === step.id)}
              aria-pressed={active('step', step.id)}
              onClick={() => onSelect({ kind: 'step', id: step.id })}
            >
              <strong>{stepLabel(step)}</strong>
              {changes
                ?.filter((c) => c.stepId === step.id)
                .map((c) => (
                  <small key={c.number}>
                    {c.number} · {c.kind === 'added-step' ? 'Added' : 'Changed'}
                  </small>
                ))}
              {marked('step', step.id).map((m) => (
                <small key={m.label}>{m.label} mark</small>
              ))}
              <span>{step.display.samples?.join(', ') || step.name}</span>
            </button>
            {flow.connections
              .filter((edge) => edge.toTask === step.id)
              .map((edge) => (
                <button
                  className="pipeline-connection-item"
                  disabled={!!changes && !creation && !addedStepFor(edge.id)}
                  key={edge.id}
                  aria-pressed={active('connection', edge.id)}
                  onClick={() => selectEdge(edge.id)}
                >
                  {connectionLabel(flow, edge.id)}
                  {marked('connection', edge.id).map((m) => (
                    <small key={m.label}>{m.label} mark</small>
                  ))}
                </button>
              ))}
          </section>
        ))}
      </div>
    );
  return (
    <div className="pipeline-graph">
      <div className="pipeline-zoom" role="group" aria-label="Flow zoom">
        <button onClick={() => setZoom(null)} aria-label="Fit pipeline flow">
          Fit
        </button>
        <button
          onClick={() => setZoom(Math.max(0.25, scale - 0.15))}
          disabled={scale <= 0.25}
          aria-label="Zoom out pipeline flow"
        >
          −
        </button>
        <output aria-label="Flow zoom level">{Math.round(scale * 100)}%</output>
        <button
          onClick={() => setZoom(Math.min(2, scale + 0.15))}
          disabled={scale >= 2}
          aria-label="Zoom in pipeline flow"
        >
          +
        </button>
      </div>
      <div
        ref={viewport}
        onScroll={(event) => {
          if (navigation) {
            navigation.left = event.currentTarget.scrollLeft;
            navigation.top = event.currentTarget.scrollTop;
          }
        }}
        className="pipeline-flow-scroll"
        tabIndex={0}
        aria-label="Pipeline flow. Scroll to explore steps and connections."
      >
        <div style={{ width: routes.width * scale, height: routes.height * scale }}>
          <div
            className="pipeline-flow-canvas"
            style={{
              width: routes.width,
              height: routes.height,
              transform: `scale(${scale})`,
              transformOrigin: 'top left',
            }}
          >
            <svg
              className="pipeline-connections-layer"
              width={routes.width}
              height={routes.height}
              aria-label="Pipeline connections"
            >
              <defs>
                <marker
                  id={marker}
                  viewBox="0 0 10 10"
                  refX="9"
                  refY="5"
                  markerWidth="6"
                  markerHeight="6"
                  orient="auto"
                >
                  <path d="M 0 0 L 10 5 L 0 10 z" fill="currentColor" />
                </marker>
              </defs>
              {routes.edges.map((edge) => {
                const d = edge.path;
                const select = () => selectEdge(edge.id);
                const interactive =
                  !selectableSteps && (creation || !changes || !!addedStepFor(edge.id));
                return (
                  <g
                    key={edge.id}
                    data-added-edge={!!addedStepFor(edge.id)}
                    data-agent-mark={marked('connection', edge.id).some(
                      (m) => m.author === 'agent',
                    )}
                    className={active('connection', edge.id) ? 'is-selected' : ''}
                  >
                    <path d={d} className="pipeline-edge" markerEnd={`url(#${marker})`} />
                    <circle className="pipeline-port" cx={edge.start.x} cy={edge.start.y} r="3.5" />
                    <circle className="pipeline-port" cx={edge.end.x} cy={edge.end.y} r="3.5" />
                    {interactive && (
                      <path
                        d={d}
                        className="pipeline-edge-target"
                        tabIndex={0}
                        role="button"
                        aria-label={connectionLabel(flow, edge.id)}
                        aria-pressed={active('connection', edge.id)}
                        onClick={select}
                        onKeyDown={(event) => {
                          if (event.key === 'Enter' || event.key === ' ') {
                            event.preventDefault();
                            select();
                          }
                        }}
                      />
                    )}
                  </g>
                );
              })}
            </svg>
            {layout.nodes.map((node) => (
              <PipelineNode
                key={node.key}
                node={node}
                {...(changes && !(creation && node.target.kind === 'input')
                  ? {
                      changes: changes.filter(
                        (c) => node.target.kind !== 'input' && c.stepId === node.target.id,
                      ),
                    }
                  : {})}
                {...(statuses && node.target.kind === 'step'
                  ? { status: statuses[node.target.id] }
                  : {})}
                plan={node.target.kind === 'step' ? plan?.[node.target.id] : undefined}
                disabled={
                  !!selectableSteps &&
                  (node.target.kind !== 'step' || !selectableSteps.includes(node.target.id))
                }
                scale={scale}
                selected={active(node.target.kind, node.target.id)}
                onSelect={onSelect}
                marks={marked(node.target.kind, node.target.id)}
              />
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}
