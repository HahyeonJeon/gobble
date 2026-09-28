import { useEffect, useState } from 'react';
import { creationFacts, type CreationContext } from '@gobble/contracts';
/** A label is resolved from the attached immutable identity, never the current selection. */
export function CreationReferenceLabel({
  projectId,
  context,
}: {
  projectId: string;
  context: CreationContext;
}) {
  const identity = JSON.stringify(context);
  const [resolved, setResolved] = useState<{ identity: string; label: string } | null>(null);
  useEffect(() => {
    if (context.kind !== 'candidate') return;
    let active = true;
    void window.gobble.creation.state({ projectId, draftId: context.draftId }).then((result) => {
      if (!active || !result.ok) return;
      const candidate = result.value.candidates.find((c) => c.candidateId === context.candidateId);
      if (!candidate) return;
      try {
        const facts = creationFacts(candidate, context);
        const value =
          'object' in facts && facts.object && 'value' in facts.object ? facts.object : undefined;
        const detail = value
          ? value.value === null
            ? 'Tool default'
            : `${value.value} ${'unit' in value ? value.unit : ''}`
          : '';
        setResolved({ identity, label: `${facts.label}${detail ? ' · ' + detail : ''}` });
      } catch {
        /* Historical identities never fall back to another candidate. */
      }
    });
    return () => {
      active = false;
    };
  }, [projectId, identity]);
  return (
    <span>
      {context.kind === 'draft'
        ? 'New pipeline · Selected data'
        : `Proposed addition · ${resolved?.identity === identity ? resolved.label : (context.target?.kind ?? 'Whole flow')}`}
    </span>
  );
}
