import { useEffect, useRef, useState } from 'react';
import {
  filterTable,
  linkedSelectionColumns,
  type RenderAcknowledgment,
  type Surface,
  type TableContent,
  type ViewLink,
} from '@gobble/contracts';
import type { Command } from '../workspace/useWorkspace';
import { Icon } from '../workspace/Icon';
import '../styles/selection.css';

export function LinkedSelectionToolbar({
  link,
  surfaces,
  content,
  acknowledgment,
  ready,
  command,
}: {
  link: ViewLink;
  surfaces: Surface[];
  content: TableContent;
  acknowledgment: RenderAcknowledgment;
  ready: boolean;
  command: Command;
}) {
  const [pending, setPending] = useState(false),
    [feedback, setFeedback] = useState<{ target: string; text: string } | null>(null);
  const working = useRef(false),
    mounted = useRef(true);
  useEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
    };
  }, []);
  const visible = new Set(filterTable(content, link.filter).map((row) => row.key));
  const hidden = link.rowKeys.filter((id) => !visible.has(id)).length;
  const columns = linkedSelectionColumns(surfaces, link, content);
  const targetKey = JSON.stringify([link.rowKeys, columns]);
  async function apply(discuss: boolean) {
    if (!ready || working.current) return;
    working.current = true;
    setPending(true);
    setFeedback(null);
    try {
      const ok = await command(
        discuss
          ? { kind: 'discussSelection', surfaceId: acknowledgment.surfaceId, acknowledgment }
          : { kind: 'select', surfaceId: acknowledgment.surfaceId, acknowledgment, evidence: null },
      );
      if (mounted.current && ok)
        setFeedback({
          target: targetKey,
          text: discuss ? 'Added to message' : 'Selection cleared',
        });
    } finally {
      working.current = false;
      if (mounted.current) setPending(false);
    }
  }
  return (
    <div className="selection-toolbar linked-selection" role="region" aria-label="Linked selection">
      <span
        className="selection-label"
        title={
          'Included columns: ' +
          columns.map((id) => content.columns.find((c) => c.id === id)?.name ?? id).join(', ')
        }
      >
        {link.rowKeys.length} selected · {columns.length} columns
        {hidden ? ` · ${hidden} outside filter` : ''}
      </span>
      <div className="selection-actions">
        <button
          disabled={!ready || pending || !link.rowKeys.length}
          onClick={() => void apply(true)}
        >
          Discuss selection
        </button>
        <button
          className="icon-button"
          disabled={!ready || pending || !link.rowKeys.length}
          aria-label="Clear linked selection"
          title="Clear selection"
          onClick={() => void apply(false)}
        >
          <Icon name="close" />
        </button>
      </div>
      {feedback?.target === targetKey && (
        <small className="selection-feedback" role="status">
          {feedback.text}
        </small>
      )}
    </div>
  );
}
