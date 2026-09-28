import type { DependencyCapture } from '@gobble/contracts';
/** Readable frozen facts; this presenter never resolves targets against a live Run. */
export function DependencyEvidence({ capture }: { capture: DependencyCapture }) {
  const selected = capture.target.selection;
  return (
    <div className="dependency-evidence">
      <strong>
        {selected.kind === 'run-group'
          ? `Task group · ${selected.taskId}`
          : `${selected.fromTaskId} → ${selected.toTaskId}`}
      </strong>
      <p>
        {selected.kind === 'run-dependency'
          ? 'Reported authored-task dependency. It does not establish sample routing or a cause of failure.'
          : 'Observed instances belonging to this authored task group.'}
      </p>
      <small>
        Observed {new Date(capture.source.observedAt).toLocaleString('en')} ·{' '}
        {capture.scope.returnedTasks} returned instances ·{' '}
        {capture.scope.membershipComplete
          ? 'Complete instance preview'
          : 'Partial instance preview'}
      </small>
      {capture.groups.map((group) => (
        <details key={group.taskId} open>
          <summary>
            {group.taskId} · {group.observedMembers} observed instances
          </summary>
          <p>
            {group.counts.states
              .map((s) => `${s.count} ${s.status || 'state unavailable'}`)
              .join(' · ') || 'No instances returned'}
          </p>
          <small>
            {group.counts.templates} templates · {group.counts.unstarted} unstarted ·{' '}
            {group.counts.attempted} attempted · {group.members.length} member records captured
            {group.truncated ? ' · Capture truncated' : ''}
          </small>
          {group.members.length > 0 && (
            <table>
              <thead>
                <tr>
                  <th scope="col">Instance</th>
                  <th scope="col">State</th>
                  <th scope="col">Attempt</th>
                </tr>
              </thead>
              <tbody>
                {group.members.map((m) => (
                  <tr key={m.instanceId}>
                    <th scope="row">{m.instanceId}</th>
                    <td>{m.status || 'Unavailable'}</td>
                    <td>{m.template ? 'Template' : m.attempt || '—'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </details>
      ))}
      <small>Frozen source facts · No logs or private runtime paths included</small>
    </div>
  );
}
