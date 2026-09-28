import { useRef, useState } from 'react';
import { evidenceLabel } from '../shared-context/marks';
import type { Command } from '../workspace/useWorkspace';
import '../styles/selection.css';
import type { EvidenceRef, RenderAcknowledgment } from '@gobble/contracts';
import { Icon } from '../workspace/Icon';

/** Actions belong to the loaded source. A selection is not yet a message attachment. */
export function SelectionToolbar({
  evidence,
  surfaceId,
  title,
  ready,
  command,
  acknowledgment,
}: {
  evidence: EvidenceRef;
  surfaceId: string;
  title: string;
  ready: boolean;
  command: Command;
  acknowledgment: RenderAcknowledgment;
}) {
  const [pending, setPending] = useState(false);
  const working = useRef(false);
  const [feedback, setFeedback] = useState('');
  async function apply(kind: 'attach' | 'share' | 'clear') {
    if (!ready || working.current) return;
    working.current = true;
    setPending(true);
    setFeedback('');
    try {
      const ok = await command(
        kind === 'clear'
          ? { kind: 'select', surfaceId, evidence: null, acknowledgment }
          : kind === 'share'
            ? { kind: 'share', surfaceId, evidence, note: '' }
            : { kind: 'attach', surfaceId, evidence, acknowledgment },
      );
      if (ok)
        setFeedback(
          kind === 'attach'
            ? 'Added to message'
            : kind === 'share'
              ? 'Shared in Project chat'
              : 'Selection cleared',
        );
    } finally {
      working.current = false;
      setPending(false);
    }
  }
  return (
    <div className="selection-toolbar" role="region" aria-label={'Selection in ' + title}>
      <span
        className="selection-label"
        aria-description="Local selection, not included in messages"
        title={title + ' · ' + evidenceLabel(evidence)}
      >
        {title} · {evidenceLabel(evidence)}
      </span>
      <div className="selection-actions">
        <button
          disabled={!ready || pending}
          aria-label={'Add to message from ' + title}
          onClick={() => void apply('attach')}
        >
          Add to message
        </button>
        {
          <button
            disabled={!ready || pending}
            aria-label={'Share mark from ' + title}
            title="Share a pointer in Project chat without sending source content"
            onClick={() => void apply('share')}
          >
            Share mark
          </button>
        }
        <button
          className="icon-button"
          disabled={!ready || pending}
          aria-label={'Remove selection from ' + title}
          title="Clear selection"
          onClick={() => void apply('clear')}
        >
          <Icon name="close" />
        </button>
      </div>
      {feedback && (
        <small className="selection-feedback" role="status">
          {feedback}
        </small>
      )}
    </div>
  );
}
