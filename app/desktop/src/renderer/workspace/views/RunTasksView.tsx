import { referenceAuthor, type SharedReference } from '@gobble/contracts';
import { markStyle } from '../../shared-context/marks';
import { useEffect, useId, useRef, useState } from 'react';
import {
  visibleRunTasks,
  logUnavailableReason,
  type RunFilter,
  type RunPresentationV2,
  type LogTarget,
  type Selection,
} from '@gobble/contracts';

export function RunTasksView({
  marks = [],
  value,
  filter,
  selection,
  ready,
  onSelect,
  onDiscuss,
  onOpenLogs,
  onFilter,
  onRefresh,
}: {
  marks?: SharedReference[];
  value: RunPresentationV2;
  filter: RunFilter | undefined;
  selection: Selection | undefined;
  ready: boolean;
  onSelect: (selection: Selection | null) => Promise<boolean>;
  onDiscuss: (selection: Selection) => Promise<void>;
  onOpenLogs: (target: LogTarget) => void;
  onFilter: (filter: RunFilter) => Promise<boolean>;
  onRefresh: () => void;
}) {
  const radioGroup = useId();
  const [query, setQuery] = useState(filter?.query ?? '');
  const status = filter?.status ?? null;
  const previousQuery = useRef(filter?.query ?? '');
  useEffect(() => {
    const previous = previousQuery.current;
    previousQuery.current = filter?.query ?? '';
    setQuery((current) => (current === previous ? (filter?.query ?? '') : current));
  }, [filter?.query]);
  useEffect(() => {
    if (query === (filter?.query ?? '')) return;
    const timer = setTimeout(() => {
      void onFilter({ query, status }).then((accepted) => {
        if (!accepted) setQuery((current) => (current === query ? (filter?.query ?? '') : current));
      });
    }, 200);
    return () => clearTimeout(timer);
  }, [query, filter?.query, status, onFilter]);
  // Content follows the acknowledged navigation; the input may hold an uncommitted edit.
  const tasks = visibleRunTasks(value, { query: filter?.query ?? '', status });
  const selected =
    selection?.kind === 'run-task'
      ? value.tasks.find(
          (task) => task.instanceId === selection.instanceId && task.attempt === selection.attempt,
        )
      : undefined;
  const visible = selected && tasks.includes(selected);
  const canAct = ready && query === (filter?.query ?? '');
  const unavailable = selected ? logUnavailableReason(selected) : null;
  const statuses = [...new Set(value.tasks.map((task) => task.status ?? ''))];
  if (status !== null && !statuses.includes(status)) statuses.push(status);
  return (
    <div className="data-view run-view">
      <div className="observation-toolbar">
        <strong>Tasks</strong>
        <input
          type="search"
          aria-label="Search tasks"
          placeholder="Search tasks"
          value={query}
          maxLength={200}
          onChange={(event) => setQuery(event.target.value)}
        />
        <select
          aria-label="Task state"
          value={status === null ? 'all' : 'state:' + status}
          disabled={!ready}
          onChange={(event) =>
            void onFilter({
              query,
              status: event.target.value === 'all' ? null : event.target.value.slice(6),
            })
          }
        >
          <option value="all">All states</option>
          {statuses.map((state) => (
            <option key={state} value={'state:' + state}>
              {state || 'Unavailable'}
            </option>
          ))}
        </select>
        <button disabled={!ready} onClick={onRefresh}>
          Refresh
        </button>
      </div>
      <div className="task-scroll">
        <table className="task-table">
          <thead>
            <tr>
              <th scope="col">Task instance</th>
              <th scope="col">State</th>
              <th scope="col">Attempt</th>
            </tr>
          </thead>
          <tbody>
            {tasks.map((task) => (
              <tr key={task.instanceId} data-selected={selected === task}>
                <td>
                  <label className="task-choice">
                    <input
                      type="radio"
                      name={radioGroup}
                      checked={selected === task}
                      disabled={!canAct}
                      aria-label={`${task.name || task.instanceId} · ${task.instanceId} · Attempt ${task.attempt}`}
                      onChange={() =>
                        void onSelect({
                          kind: 'run-task',
                          coordinateSpace: 'observed-instance-attempt',
                          instanceId: task.instanceId,
                          attempt: task.attempt,
                        })
                      }
                    />
                    <span>
                      <strong>{task.name || task.instanceId}</strong>
                      <small>{task.instanceId}</small>
                      {marks
                        .filter(
                          (mark) =>
                            mark.evidence.selection?.kind === 'run-task' &&
                            mark.evidence.selection.instanceId === task.instanceId &&
                            mark.evidence.selection.attempt === task.attempt,
                        )
                        .map((mark) => (
                          <small
                            className="agent-task-mark"
                            style={markStyle(mark)}
                            key={mark.referenceId}
                          >
                            {referenceAuthor(mark)} · Reference
                          </small>
                        ))}
                    </span>
                  </label>
                </td>
                <td>
                  <span className="task-state">{task.status || 'Unavailable'}</span>
                </td>
                <td>{task.template ? 'Template' : task.attempt || '—'}</td>
              </tr>
            ))}
          </tbody>
        </table>
        {!tasks.length && (
          <p className="observation-empty">
            {value.tasks.length
              ? 'No tasks match in this preview.'
              : 'No tasks returned in this observation.'}
          </p>
        )}
      </div>
      <div className="observation-actions">
        <div>
          {selected ? (
            <>
              <strong>
                {selected.instanceId} · Attempt {selected.attempt}
              </strong>
              <small>
                {!visible
                  ? 'Selection is outside this filter'
                  : unavailable || selected.reason || 'Selected task instance'}
              </small>
            </>
          ) : (
            <span>Select a task to discuss it or open its logs.</span>
          )}
        </div>
        {selected && !visible && (
          <button
            disabled={!canAct}
            onClick={() => {
              setQuery(selected.instanceId.slice(0, 200));
              void onFilter({ query: selected.instanceId.slice(0, 200), status: null });
            }}
          >
            Show selected task
          </button>
        )}
        <button
          disabled={!canAct || !visible || !!unavailable}
          onClick={() =>
            selected &&
            onOpenLogs({
              runRef: value.runRef,
              instanceId: selected.instanceId,
              attempt: selected.attempt,
            })
          }
        >
          Open logs
        </button>
        <button
          disabled={!canAct || !visible}
          onClick={() => selection && void onDiscuss(selection)}
        >
          Discuss task
        </button>
      </div>
      <details className="observation-details">
        <summary>
          {tasks.length} shown · {value.tasks.length} loaded
          {value.preview?.truncated ? ` of ${value.preview.availableTasks}` : ''} · Run details
        </summary>
        <dl>
          <dt>Run</dt>
          <dd>{value.runId}</dd>
          <dt>Pipeline</dt>
          <dd>{value.pipelineName || 'Unavailable'}</dd>
          <dt>State</dt>
          <dd>{value.status || 'Unavailable'}</dd>
          <dt>Engine revision</dt>
          <dd>{value.engineRevision}</dd>
          <dt>Runtime image</dt>
          <dd>{value.imageId}</dd>
        </dl>
        {value.dependencies.length > 0 && (
          <div aria-label="Authored task dependencies">
            <strong>Authored task dependencies</strong>
            {value.dependencies.map((edge, index) => (
              <p key={index}>
                {edge.fromTaskId} → {edge.toTaskId}
              </p>
            ))}
          </div>
        )}
      </details>
    </div>
  );
}
