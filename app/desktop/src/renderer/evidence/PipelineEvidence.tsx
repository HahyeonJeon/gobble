import {
  pipelineSubjectLabel,
  type PipelineSubject,
  type PipelineCapture,
} from '@gobble/contracts';

export function PipelineSubjectContent({ subject }: { subject: PipelineSubject }) {
  const port = subject.kind === 'input' || subject.kind === 'port' ? subject.port : undefined;
  return (
    <div className="pipeline-saved-subject">
      <strong>{pipelineSubjectLabel(subject)}</strong>
      {subject.kind === 'step' && (
        <>
          <p>
            {subject.step.image || 'Local tool'} · {subject.step.cpu || 'Unspecified'} CPU
            {subject.step.memory ? ' · ' + subject.step.memory + ' memory' : ''}
          </p>
          <dl>
            <div>
              <dt>Inputs</dt>
              <dd>{subject.step.inputs.map((p) => p.name).join(', ') || 'None declared'}</dd>
            </div>
            <div>
              <dt>Declared outputs</dt>
              <dd>{subject.step.outputs.map((p) => p.name).join(', ') || 'None declared'}</dd>
            </div>
          </dl>
          {(subject.step.settings ?? []).map((s) => (
            <p key={s.key}>
              {s.label}: {s.value === null ? 'Tool default' : s.value + ' ' + s.unit}
            </p>
          ))}
        </>
      )}
      {port && (
        <>
          <p>
            {port.kind} · {port.path || 'No path declared'}
          </p>
          {port.members.map((m) => (
            <p key={m.name}>
              {m.name} · {m.path}
            </p>
          ))}
        </>
      )}
      {subject.kind === 'setting' && (
        <p>
          {subject.setting.value === null
            ? 'Tool default'
            : subject.setting.value + ' ' + subject.setting.unit}
        </p>
      )}
      {subject.kind === 'connection' && (
        <p>
          Output <b>{subject.connection.fromPort}</b> connects to input{' '}
          <b>{subject.connection.toPort}</b>.
        </p>
      )}
    </div>
  );
}
/** Frozen semantic facts, displayed without resolving or opening current source. */
export function PipelineEvidence({ capture }: { capture: PipelineCapture }) {
  return (
    <div>
      <small>
        {capture.pipelineName} · Checked {new Date(capture.checkedAt).toLocaleString('en')}
      </small>
      <PipelineSubjectContent subject={capture.subject} />
      <p className="muted">Saved pipeline version · Independent of later checks</p>
    </div>
  );
}
