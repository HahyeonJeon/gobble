import { useState } from 'react';
import {
  defaultDependencyNavigation,
  dependencyPairKey,
  type DependencyObservation,
  type DependencyTarget,
  type SharedReference,
  type SurfaceData,
} from '@gobble/contracts';
import { DependencyGraph } from './DependencyGraph';
import { DependencyDetails } from './DependencyDetails';
import { dependencyMarks, DependencyMarkLabel } from './dependency-marks';

/** A temporary presenter owns only its camera. No workspace selection, draft or navigation writes. */
export function DependencyReferenceContent({
  data,
  observation,
  reference,
  target,
}: {
  data: Extract<SurfaceData, { kind: 'run' }>;
  observation: DependencyObservation;
  reference: SharedReference;
  target: DependencyTarget['selection'];
}) {
  const [camera, setCamera] = useState(defaultDependencyNavigation().camera);
  return (
    <div className="dependency-reference-content">
      <p className="dependency-scope">
        {observation.groups.length} observed groups · {observation.edges.length} observed
        dependencies · Read-only reference
      </p>
      {observation.display === 'graph' && (
        <DependencyGraph
          observation={observation}
          marks={[reference]}
          selection={target}
          camera={camera}
          ready
          readOnly
          reveal={1}
          onCamera={setCamera}
          onSelect={() => {}}
        />
      )}
      <details className="dependency-edge-list" open={observation.display !== 'graph'}>
        <summary>Reference target list</summary>
        <ul className="dependency-reference-list">
          {observation.groups.map((group) => {
            const marks = dependencyMarks([reference], {
              kind: 'run-group',
              coordinateSpace: 'observed-authored-task-group',
              taskId: group.taskId,
            });
            return (
              <li key={group.taskId} data-agent-mark={marks.length > 0}>
                {group.taskId} <DependencyMarkLabel marks={marks} />
              </li>
            );
          })}
          {observation.edges.map((edge) => {
            const marks = dependencyMarks([reference], {
              kind: 'run-dependency',
              coordinateSpace: 'observed-authored-task-pair',
              ...edge,
            });
            return (
              <li key={dependencyPairKey(edge)} data-agent-mark={marks.length > 0}>
                {edge.fromTaskId} → {edge.toTaskId} <DependencyMarkLabel marks={marks} />
              </li>
            );
          })}
        </ul>
      </details>
      <DependencyDetails
        observation={observation}
        value={data.value}
        selection={target}
        ready
        visible
        readOnly
        onSelect={() => {}}
        onOpenLogs={() => {}}
        onDiscuss={() => {}}
        onShow={() => {}}
        onTasks={() => {}}
      />
    </div>
  );
}
