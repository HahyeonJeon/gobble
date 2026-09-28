import type { CSSProperties } from 'react';
import { nodeWidth, nodeHeight, type FlowLayout, type FlowTarget } from './flow-layout';

/** Decorative role glyphs, not guessed tool branding or execution status. */
function RoleGlyph({ input }: { input: boolean }) {
  return (
    <svg
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.7"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      {input ? (
        <path d="M6 3h8l4 4v14H6V3Z M14 3v5h4 M9 12h6 M9 16h4" />
      ) : (
        <>
          <rect x="6" y="6" width="12" height="12" rx="3" />
          <path d="M9 2v4 M15 2v4 M9 18v4 M15 18v4 M2 9h4 M2 15h4 M18 9h4 M18 15h4 M10 10h4v4h-4z" />
        </>
      )}
    </svg>
  );
}

export function PipelineNode({
  node,
  selected,
  scale,
  onSelect,
  marks = [],
  changes,
  status,
  plan,
  disabled,
}: {
  status?: string | undefined;
  plan?: { action: string; label: string } | undefined;
  disabled?: boolean;
  node: FlowLayout['nodes'][number];
  selected: boolean;
  marks?: Array<{ label: string; author: 'agent' | 'user' }>;
  changes?: Array<{ kind: string; number: number }>;
  scale: number;
  onSelect: (target: FlowTarget) => void;
}) {
  const input = node.target.kind === 'input';
  return (
    <button
      disabled={disabled || changes?.length === 0}
      data-run-status={status}
      data-continuation={plan?.action}
      data-change={changes ? (changes[0]?.kind ?? 'unchanged') : undefined}
      data-agent-mark={marks.some((m) => m.author === 'agent')}
      className={`pipeline-node ${input ? 'pipeline-input-node' : ''}`}
      style={
        {
          left: node.x,
          top: node.y,
          width: nodeWidth,
          height: nodeHeight,
          '--flow-title-size': `${Math.max(13, 10 / scale)}px`,
          '--flow-detail-size': `${Math.max(10, 8 / scale)}px`,
        } as CSSProperties
      }
      title={`${node.label} · ${node.detail}${selected ? ' · Selected' : ''}`}
      aria-label={`Inspect ${node.label}${node.detail !== node.label ? ` · ${node.detail}` : ''}${plan ? ` · ${plan.label}` : ''}`}
      aria-pressed={selected}
      onClick={() => onSelect(node.target)}
    >
      <span className="pipeline-node-heading">
        <span className="pipeline-node-icon">
          <RoleGlyph input={input} />
        </span>
        <strong>{node.label}</strong>
      </span>
      {!!changes?.length && (
        <span className="review-node-number">
          {changes.map((c) => c.number).join(', ')} {changes[0]?.kind === 'added-step' ? '+' : '◆'}
        </span>
      )}
      {marks.length > 0 && (
        <span className="pipeline-mark-badge">
          {[...new Set(marks.map((m) => m.label))].join(', ')} mark
        </span>
      )}
      <span className="pipeline-node-footer">
        <span className="pipeline-node-detail">{node.detail}</span>
        {(plan || status) && <span className="pipeline-run-state">{plan?.label ?? status}</span>}
        {selected && (
          <span className="pipeline-selected-badge" aria-hidden="true">
            ●
          </span>
        )}
      </span>
    </button>
  );
}
