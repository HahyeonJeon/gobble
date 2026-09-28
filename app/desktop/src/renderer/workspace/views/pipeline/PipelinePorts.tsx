import type { PipelinePort } from '@gobble/contracts';

export function PipelinePorts({
  label,
  ports,
  onSelect,
  active,
  marks = [],
}: {
  label: string;
  ports: PipelinePort[];
  onSelect?: (name: string) => void;
  active?: string | undefined;
  marks?: Array<{ name: string; label: string }>;
}) {
  return (
    <section>
      <h4>{label}</h4>
      {ports.length === 0 ? (
        <p className="muted">None declared</p>
      ) : (
        <ul className="pipeline-ports">
          {ports.map((port) => (
            <li key={port.name}>
              {onSelect ? (
                <button
                  aria-pressed={active === port.name}
                  data-agent-mark={marks.some((m) => m.name === port.name)}
                  onClick={() => onSelect(port.name)}
                  aria-label={'Select ' + label.toLowerCase() + ' port ' + port.name}
                >
                  {port.name}
                  {marks
                    .filter((m) => m.name === port.name)
                    .map((m) => (
                      <small key={m.label}> · {m.label} mark</small>
                    ))}
                </button>
              ) : (
                <strong>{port.name}</strong>
              )}
              <span>
                {port.kind}
                {port.members.length ? ` · ${port.members.length} members` : ''}
              </span>
              {port.path && <span className="pipeline-data-path">{port.path}</span>}
              {port.members.length > 0 && (
                <details>
                  <summary>Show members</summary>
                  {port.members.map((member) => (
                    <p key={member.name}>
                      {member.name} · {member.path}
                    </p>
                  ))}
                </details>
              )}
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}
