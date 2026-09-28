import { dependencyMarks, DependencyMarkLabel } from './dependency-marks';
import { useEffect, useState } from 'react';
import {
  defaultDependencyNavigation,
  matchingDependencies,
  matchingDependencyGroups,
  dependencyPairKey,
  type DependencyNavigation,
  type DependencyBrowse,
  type DependencyCamera,
  type DependencyObservation,
  type RunPresentation,
  type Selection,
  type LogTarget,
  type SharedReference,
} from '@gobble/contracts';
import { DependencyGraph } from './DependencyGraph';
import { DependencyDetails } from './DependencyDetails';

type Props = {
  value: RunPresentation;
  marks?: SharedReference[];
  observation: DependencyObservation | undefined;
  problem: string | undefined;
  navigation: DependencyNavigation | undefined;
  selection: Selection | undefined;
  ready: boolean;
  onCamera: (camera: DependencyCamera) => Promise<boolean>;
  onNavigate: (navigation: DependencyBrowse) => Promise<boolean>;
  onSelect: (selection: Selection | null) => Promise<boolean>;
  onDiscuss: (selection: Selection) => Promise<void>;
  onOpenLogs: (target: LogTarget) => void;
  onTasks: () => void;
  onRefresh: () => void;
};
const diagnostics: Record<DependencyObservation['diagnostics'][number], string> = {
  'missing-topology': 'Dependencies were not returned.',
  'invalid-topology': 'Some dependency metadata is invalid.',
  'duplicate-edges': 'Repeated dependencies were combined.',
  'partial-topology': 'Only part of the dependency list was inspected.',
  'cyclic-topology': 'A cycle was reported. Showing the list.',
  'group-limit': 'Group limit reached. Showing a bounded list.',
  'edge-limit': 'Dependency limit reached. Showing a bounded list.',
  'partial-membership': 'Instance membership is a partial preview.',
  'unmapped-members': 'Some instances have no authored task ID.',
};
export function DependencyView({
  value,
  marks = [],
  observation,
  problem,
  navigation: stored,
  selection,
  ready,
  onNavigate,
  onCamera,
  onSelect,
  onDiscuss,
  onOpenLogs,
  onTasks,
  onRefresh,
}: Props) {
  const navigation = stored ?? defaultDependencyNavigation();
  const [query, setQuery] = useState(navigation.query);
  const [reveal, setReveal] = useState(0);
  useEffect(() => setQuery(navigation.query), [navigation.query]);
  useEffect(() => {
    if (!ready || query === navigation.query) return;
    const timer = setTimeout(() => {
      void onNavigate({ representation: navigation.representation, query }).then((accepted) => {
        if (!accepted) setQuery(navigation.query);
      });
    }, 200);
    return () => clearTimeout(timer);
  }, [query, navigation.query, navigation.representation, onNavigate, ready]);
  const canAct = ready && query === navigation.query;
  const graph =
    observation?.display === 'graph' &&
    navigation.representation === 'graph' &&
    observation.groups.length > 0;
  const groups = observation ? matchingDependencyGroups(observation, navigation.query) : [];
  const edges = observation ? matchingDependencies(observation, navigation.query) : [];
  const visible =
    !!observation &&
    !!selection &&
    (selection.kind === 'run-group'
      ? (graph ? observation.groups : groups).some((g) => g.taskId === selection.taskId)
      : selection.kind === 'run-dependency'
        ? (graph ? observation.edges : edges).some(
            (e) => e.fromTaskId === selection.fromTaskId && e.toTaskId === selection.toTaskId,
          )
        : selection.kind === 'run-task' &&
          observation.groups.some((g) =>
            g.members.some(
              (m) => m.instanceId === selection.instanceId && m.attempt === selection.attempt,
            ),
          ));
  const choose = (target: Selection) => {
    void onSelect(target);
  };
  return (
    <div className="dependency-view">
      <div className="observation-toolbar">
        <input
          type="search"
          aria-label="Find task group"
          placeholder="Find task group"
          maxLength={200}
          value={query}
          onChange={(e) => setQuery(e.target.value)}
        />
        <select
          aria-label="Dependency representation"
          value={navigation.representation}
          disabled={!ready}
          onChange={(e) =>
            void onNavigate({
              query: navigation.query,
              representation: e.target.value === 'list' ? 'list' : 'graph',
            })
          }
        >
          <option value="graph">Graph</option>
          <option value="list">List</option>
        </select>
        <button
          disabled={!canAct || !selection || !observation}
          onClick={() => {
            void onNavigate({ representation: navigation.representation, query: '' }).then(
              (accepted) => {
                if (accepted) setReveal((n) => n + 1);
              },
            );
          }}
        >
          Show selected target
        </button>
        <button disabled={!ready} onClick={onRefresh}>
          Refresh
        </button>
      </div>
      {!observation ? (
        <p className="observation-empty" role="status">
          {problem || 'Dependencies are unavailable for this observation.'}{' '}
          <button onClick={onTasks}>Show Tasks</button>
        </p>
      ) : (
        <>
          <div className="dependency-scope">
            <span>
              {observation.groups.length} of {observation.scope.observedGroups} observed groups ·{' '}
              {observation.edges.length} of {observation.scope.observedEdges} observed dependencies
              · {observation.scope.returnedTasks} returned instances
            </span>
            {observation.diagnostics.length > 0 && (
              <details>
                <summary>Preview limitations</summary>
                <ul>
                  {observation.diagnostics.map((d) => (
                    <li key={d}>{diagnostics[d]}</li>
                  ))}
                </ul>
              </details>
            )}
          </div>
          {graph ? (
            <>
              {query && (
                <div className="dependency-find-results" aria-label="Matching groups">
                  {groups.length ? (
                    groups.map((g) => (
                      <button
                        key={g.taskId}
                        data-agent-mark={
                          dependencyMarks(marks, {
                            kind: 'run-group',
                            coordinateSpace: 'observed-authored-task-group',
                            taskId: g.taskId,
                          }).length > 0
                        }
                        disabled={!canAct}
                        onClick={() => {
                          void onSelect({
                            kind: 'run-group',
                            coordinateSpace: 'observed-authored-task-group',
                            taskId: g.taskId,
                          }).then((accepted) => {
                            if (accepted) setReveal((n) => n + 1);
                          });
                        }}
                      >
                        {g.taskId}
                      </button>
                    ))
                  ) : (
                    <span>No groups match in this preview.</span>
                  )}
                </div>
              )}
              <DependencyGraph
                observation={observation}
                marks={marks}
                selection={selection}
                camera={navigation.camera}
                ready={canAct}
                reveal={reveal}
                onCamera={(camera) => {
                  void onCamera(camera);
                }}
                onSelect={choose}
              />
              <details className="dependency-edge-list">
                <summary>Dependency list · {edges.length}</summary>
                <div>
                  {edges.length ? (
                    edges.map((e) => (
                      <button
                        key={dependencyPairKey(e)}
                        data-agent-mark={
                          dependencyMarks(marks, {
                            kind: 'run-dependency',
                            coordinateSpace: 'observed-authored-task-pair',
                            ...e,
                          }).length > 0
                        }
                        disabled={!canAct}
                        aria-pressed={
                          selection?.kind === 'run-dependency' &&
                          selection.fromTaskId === e.fromTaskId &&
                          selection.toTaskId === e.toTaskId
                        }
                        onClick={() =>
                          choose({
                            kind: 'run-dependency',
                            coordinateSpace: 'observed-authored-task-pair',
                            ...e,
                          })
                        }
                      >
                        {e.fromTaskId} → {e.toTaskId}
                        <DependencyMarkLabel
                          marks={dependencyMarks(marks, {
                            kind: 'run-dependency',
                            coordinateSpace: 'observed-authored-task-pair',
                            ...e,
                          })}
                        />
                      </button>
                    ))
                  ) : (
                    <p>No dependencies reported in this scope.</p>
                  )}
                </div>
              </details>
            </>
          ) : (
            <div className="dependency-list" aria-label="Dependency overview list">
              {observation.display !== 'graph' && (
                <p role="status">
                  {observation.display === 'unavailable'
                    ? 'Dependencies unavailable. Known task groups are listed below.'
                    : 'Showing a bounded list because this topology cannot be drawn completely.'}
                </p>
              )}
              <div>
                <strong>Task groups</strong>
                {groups.map((g) => (
                  <button
                    key={g.taskId}
                    data-agent-mark={
                      dependencyMarks(marks, {
                        kind: 'run-group',
                        coordinateSpace: 'observed-authored-task-group',
                        taskId: g.taskId,
                      }).length > 0
                    }
                    disabled={!canAct}
                    aria-pressed={selection?.kind === 'run-group' && selection.taskId === g.taskId}
                    onClick={() =>
                      choose({
                        kind: 'run-group',
                        coordinateSpace: 'observed-authored-task-group',
                        taskId: g.taskId,
                      })
                    }
                  >
                    {g.taskId}
                    <small>{g.members.length} observed</small>
                    <DependencyMarkLabel
                      marks={dependencyMarks(marks, {
                        kind: 'run-group',
                        coordinateSpace: 'observed-authored-task-group',
                        taskId: g.taskId,
                      })}
                    />
                  </button>
                ))}
                {!groups.length && <p>No groups match in this preview.</p>}
              </div>
              <div>
                <strong>Directed dependencies</strong>
                {edges.map((e) => (
                  <button
                    key={dependencyPairKey(e)}
                    data-agent-mark={
                      dependencyMarks(marks, {
                        kind: 'run-dependency',
                        coordinateSpace: 'observed-authored-task-pair',
                        ...e,
                      }).length > 0
                    }
                    disabled={!canAct}
                    aria-pressed={
                      selection?.kind === 'run-dependency' &&
                      selection.fromTaskId === e.fromTaskId &&
                      selection.toTaskId === e.toTaskId
                    }
                    onClick={() =>
                      choose({
                        kind: 'run-dependency',
                        coordinateSpace: 'observed-authored-task-pair',
                        ...e,
                      })
                    }
                  >
                    {e.fromTaskId} → {e.toTaskId}
                    <DependencyMarkLabel
                      marks={dependencyMarks(marks, {
                        kind: 'run-dependency',
                        coordinateSpace: 'observed-authored-task-pair',
                        ...e,
                      })}
                    />
                  </button>
                ))}
                {!edges.length && (
                  <p>
                    {observation.topology === 'reported'
                      ? 'No dependencies reported in this scope.'
                      : 'Dependency topology was not available.'}
                  </p>
                )}
              </div>
            </div>
          )}
          <DependencyDetails
            observation={observation}
            value={value}
            selection={selection}
            ready={canAct}
            visible={visible}
            onSelect={choose}
            onOpenLogs={onOpenLogs}
            onDiscuss={() => selection && void onDiscuss(selection)}
            onShow={() => void onNavigate({ representation: navigation.representation, query: '' })}
            onTasks={onTasks}
          />
        </>
      )}
    </div>
  );
}
