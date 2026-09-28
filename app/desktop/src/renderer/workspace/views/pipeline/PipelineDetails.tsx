import { pipelineSubject, pipelineSubjectLabel, type PipelineFlow } from '@gobble/contracts';
import { connectionLabel, flowSelector } from './flow-selection';
import { type FlowTarget } from './flow-layout';
import { PipelinePorts as Ports } from './PipelinePorts';

export function PipelineDetails({
  flow,
  selected,
  onClose,
  onSelect,
  onDiscuss,
  canDiscuss,
  feedback,
  marks = [],
}: {
  flow: PipelineFlow;
  selected: FlowTarget;
  onClose: () => void;
  onSelect: (value: FlowTarget) => void;
  onDiscuss: () => void;
  canDiscuss: boolean;
  feedback: string;
  marks?: Array<{ target: FlowTarget; label: string; author: 'agent' | 'user' }>;
}) {
  const step = ['step', 'port', 'setting'].includes(selected.kind)
    ? flow.steps.find((value) => value.id === selected.id)
    : undefined;
  const input =
    selected.kind === 'input' ? flow.inputs.find((value) => value.name === selected.id) : undefined;
  const edge =
    selected.kind === 'connection'
      ? flow.connections.find((value) => value.id === selected.id)
      : undefined;
  const subject = pipelineSubject(flow, flowSelector(flow, selected));
  const title =
    subject.kind === 'connection'
      ? 'Connection'
      : subject.kind === 'setting'
        ? subject.setting.label
        : subject.kind === 'port' || subject.kind === 'input'
          ? subject.port.name
          : pipelineSubjectLabel(subject);
  const connections = flow.connections.filter((value) =>
    step
      ? value.fromTask === step.id || value.toTask === step.id
      : input
        ? value.fromTask === '' && value.fromPort === input.name
        : value.id === edge?.id,
  );
  return (
    <section className="pipeline-details" aria-label={`Details for ${title}`}>
      <header>
        <div>
          <span className="muted">
            {selected.kind === 'setting'
              ? 'Setting'
              : selected.kind === 'port'
                ? selected.direction === 'input'
                  ? 'Input port'
                  : 'Output port'
                : step
                  ? 'Step'
                  : input
                    ? 'Pipeline input'
                    : 'Data connection'}
          </span>
          <h3>{title}</h3>
        </div>
        <button onClick={onClose} aria-label="Close step details">
          Close
        </button>
      </header>
      <div className="pipeline-discuss-action">
        <span>
          {selected.kind === 'setting' && subject.kind === 'setting'
            ? subject.setting.value === null
              ? 'Tool default'
              : subject.setting.value + ' ' + subject.setting.unit
            : selected.kind === 'port' && subject.kind === 'port'
              ? subject.direction + ' · ' + subject.port.name
              : 'Selected ' + selected.kind}
        </span>
        <button className="primary" disabled={!canDiscuss} onClick={onDiscuss}>
          Add to message
        </button>
        {feedback && <small role="status">{feedback}</small>}
      </div>
      {step && (
        <>
          <div className="pipeline-detail-columns">
            <Ports
              label="Inputs"
              ports={step.inputs}
              active={
                selected.kind === 'port' && selected.direction === 'input'
                  ? selected.name
                  : undefined
              }
              marks={marks.flatMap((m) =>
                m.author === 'agent' &&
                m.target.kind === 'port' &&
                m.target.id === step.id &&
                m.target.direction === 'input'
                  ? [{ name: m.target.name, label: m.label }]
                  : [],
              )}
              onSelect={(name) => onSelect({ kind: 'port', id: step.id, name, direction: 'input' })}
            />
            <Ports
              label="Declared outputs"
              ports={step.outputs}
              active={
                selected.kind === 'port' && selected.direction === 'output'
                  ? selected.name
                  : undefined
              }
              marks={marks.flatMap((m) =>
                m.author === 'agent' &&
                m.target.kind === 'port' &&
                m.target.id === step.id &&
                m.target.direction === 'output'
                  ? [{ name: m.target.name, label: m.label }]
                  : [],
              )}
              onSelect={(name) =>
                onSelect({ kind: 'port', id: step.id, name, direction: 'output' })
              }
            />
          </div>
          <section className="pipeline-settings">
            <h4>Analysis settings</h4>
            {step.settings?.length ? (
              step.settings.map((setting) => (
                <button
                  key={setting.key}
                  data-agent-mark={marks.some(
                    (m) =>
                      m.author === 'agent' &&
                      m.target.kind === 'setting' &&
                      m.target.id === step.id &&
                      m.target.name === setting.key,
                  )}
                  aria-pressed={selected.kind === 'setting' && selected.name === setting.key}
                  onClick={() => onSelect({ kind: 'setting', id: step.id, name: setting.key })}
                >
                  <span>{setting.label}</span>
                  <strong>
                    {setting.value === null ? 'Tool default' : setting.value + ' ' + setting.unit}
                  </strong>
                </button>
              ))
            ) : (
              <p className="muted">This module does not publish supported analysis settings.</p>
            )}
          </section>
          <dl className="pipeline-facts">
            <div>
              <dt>Tool</dt>
              <dd>{step.image || 'Local tool'}</dd>
            </div>
            <div>
              <dt>Requested compute</dt>
              <dd>
                {step.cpu ? `${step.cpu} CPU` : 'CPU not specified'}
                {step.memory ? ` · ${step.memory} memory` : ''}
              </dd>
            </div>
            {step.display.samples?.length ? (
              <div>
                <dt>Samples</dt>
                <dd>{step.display.samples.join(', ')}</dd>
              </div>
            ) : null}
          </dl>
          {(step.control.branch ||
            step.control.merge ||
            step.control.scatter ||
            step.control.gather ||
            step.control.when ||
            step.control.skipIfFalse ||
            step.control.skipIfMissingPort) && (
            <details className="pipeline-rules">
              <summary>Flow rules</summary>
              <dl>
                {Object.entries(step.control)
                  .filter(([, value]) => (Array.isArray(value) ? value.length : value))
                  .map(([name, value]) => (
                    <div key={name}>
                      <dt>
                        {
                          (
                            {
                              branch: 'Branch',
                              merge: 'Merge',
                              scatter: 'Parallel group',
                              gather: 'Gather group',
                              when: 'Condition group',
                              scatterFromKind: 'Parallel input kind',
                              scatterFromTask: 'Parallel source step',
                              scatterFromPort: 'Parallel source port',
                              scatterFromPath: 'Parallel source data',
                              scatterMembers: 'Parallel members',
                              scatterMemberPaths: 'Parallel member data',
                              skipIfMissingTask: 'Required source step',
                              skipIfMissingPort: 'Required source port',
                              skipIfMissingPath: 'Required data',
                              skipIfFalse: 'Required condition',
                            } as Record<string, string>
                          )[name]
                        }
                      </dt>
                      <dd>{Array.isArray(value) ? value.join(', ') : value}</dd>
                    </div>
                  ))}
              </dl>
            </details>
          )}
          <p className="pipeline-detail-note muted">
            This view shows the step’s inputs, expected outputs and requested computing resources.
          </p>
        </>
      )}
      {input && <Ports label="Declared input" ports={[input]} />}
      {connections.length > 0 && (
        <section>
          <h4>{edge ? 'From output to input' : 'Connections'}</h4>
          <ul className="pipeline-connections">
            {connections.map((connection) => (
              <li key={connection.id}>
                <button onClick={() => onSelect({ kind: 'connection', id: connection.id })}>
                  {connectionLabel(flow, connection.id)}
                </button>
              </li>
            ))}
          </ul>
        </section>
      )}
      {edge && (
        <>
          <div className="pipeline-endpoints">
            {edge.fromTask && (
              <button onClick={() => onSelect({ kind: 'step', id: edge.fromTask })}>
                Show source step
              </button>
            )}
            <button onClick={() => onSelect({ kind: 'step', id: edge.toTask })}>
              Show destination step
            </button>
          </div>
          {edge.wait.length > 0 && (
            <details>
              <summary>Required data before this step</summary>
              {edge.wait.map((path, index) => (
                <p key={index} className="pipeline-data-path">
                  {path}
                </p>
              ))}
            </details>
          )}
        </>
      )}
    </section>
  );
}
