import { useId } from 'react';
import {
  logUnavailableReason,
  type DependencyObservation,
  type RunPresentation,
  type Selection,
  type LogTarget,
} from '@gobble/contracts';

type Props = {
  observation: DependencyObservation;
  value: RunPresentation;
  selection: Selection | undefined;
  ready: boolean;
  readOnly?: boolean;
  visible: boolean;
  onSelect: (selection: Selection) => void;
  onOpenLogs: (target: LogTarget) => void;
  onDiscuss: () => void;
  onShow: () => void;
  onTasks: () => void;
};
export function DependencyDetails({
  observation,
  value,
  selection,
  ready,
  readOnly = false,
  visible,
  onSelect,
  onOpenLogs,
  onDiscuss,
  onShow,
  onTasks,
}: Props) {
  const radio = useId();
  const chosen =
    selection?.kind === 'run-task'
      ? value.tasks.find(
          (t) => t.instanceId === selection.instanceId && t.attempt === selection.attempt,
        )
      : undefined;
  const ids =
    selection?.kind === 'run-group'
      ? [selection.taskId]
      : selection?.kind === 'run-dependency'
        ? [selection.fromTaskId, selection.toTaskId]
        : chosen?.taskId
          ? [chosen.taskId]
          : [];
  const groups = [...new Set(ids)].flatMap((id) =>
    observation.groups.filter((g) => g.taskId === id),
  );
  const label =
    selection?.kind === 'run-group'
      ? `Group · ${selection.taskId}`
      : selection?.kind === 'run-dependency'
        ? `${selection.fromTaskId} → ${selection.toTaskId}`
        : chosen
          ? `${chosen.instanceId} · Attempt ${chosen.attempt}`
          : 'Select a group or dependency';
  const unavailable = chosen ? logUnavailableReason(chosen) : null;
  return (
    <section className="dependency-detail" aria-label="Dependency selection">
      <div className="dependency-selection-actions">
        <div>
          <strong>{label}</strong>
          <small>
            {!selection
              ? 'Choose a node or a directed pair from the list.'
              : !groups.length
                ? 'This target is outside the returned dependency groups.'
                : !visible
                  ? 'Selection is outside this search.'
                  : selection.kind === 'run-dependency'
                    ? 'Authored task dependency · Not a sample-level or causal relationship'
                    : chosen
                      ? unavailable || chosen.status || 'State unavailable'
                      : 'Observed member counts · No aggregate execution state'}
          </small>
        </div>
        {!readOnly && selection && !visible && (
          <button disabled={!ready} onClick={groups.length ? onShow : onTasks}>
            {groups.length ? 'Show selected target' : 'Show in Tasks'}
          </button>
        )}
        {!readOnly && chosen && (
          <button
            disabled={!ready || !visible || !!unavailable}
            onClick={() =>
              onOpenLogs({
                runRef: value.runRef,
                instanceId: chosen.instanceId,
                attempt: chosen.attempt,
              })
            }
          >
            Open logs
          </button>
        )}
        {!readOnly && (
          <button disabled={!ready || !selection || !visible} onClick={onDiscuss}>
            {selection?.kind === 'run-task'
              ? 'Discuss task'
              : selection?.kind === 'run-dependency'
                ? 'Discuss dependency'
                : 'Discuss group'}
          </button>
        )}
      </div>
      {groups.length > 0 && (
        <div className="dependency-members">
          {groups.map((group) => (
            <details key={group.taskId} open={selection?.kind !== 'run-dependency'}>
              <summary>
                <strong>{group.taskId}</strong> · {group.members.length} observed ·{' '}
                {group.membership === 'partial-preview' ? 'Partial preview' : 'Complete preview'}
              </summary>
              <p className="dependency-counts">
                {group.counts.states
                  .map((s) => `${s.count} ${s.status || 'state unavailable'}`)
                  .join(' · ') || 'No instances in this preview'}
                <br />
                {group.counts.templates} templates · {group.counts.unstarted} unstarted ·{' '}
                {group.counts.attempted} attempted
                {group.counts.unknownTemplate
                  ? ` · ${group.counts.unknownTemplate} template flags unavailable`
                  : ''}
              </p>
              {group.members.length > 0 && (
                <table className="task-table">
                  <thead>
                    <tr>
                      <th scope="col">Observed instance</th>
                      <th scope="col">State</th>
                      <th scope="col">Attempt</th>
                    </tr>
                  </thead>
                  <tbody>
                    {group.members.map((member) => (
                      <tr
                        key={member.instanceId}
                        data-selected={chosen?.instanceId === member.instanceId}
                      >
                        <td>
                          <label className="task-choice">
                            <input
                              type="radio"
                              name={radio}
                              checked={
                                chosen?.instanceId === member.instanceId &&
                                chosen.attempt === member.attempt
                              }
                              disabled={!ready || readOnly}
                              aria-label={`Instance ${member.instanceId} · Attempt ${member.attempt}`}
                              onChange={() =>
                                onSelect({
                                  kind: 'run-task',
                                  coordinateSpace: 'observed-instance-attempt',
                                  instanceId: member.instanceId,
                                  attempt: member.attempt,
                                })
                              }
                            />
                            <span>{member.instanceId}</span>
                          </label>
                        </td>
                        <td>{member.status || 'Unavailable'}</td>
                        <td>{member.template ? 'Template' : member.attempt || '—'}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              )}
            </details>
          ))}
        </div>
      )}
    </section>
  );
}
