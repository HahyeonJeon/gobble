import { useState } from 'react';
import type { RunFollowUp } from '@gobble/contracts';
import { AttachmentList } from '../evidence/AttachmentList';
import '../styles/run-feedback.css';
/** Opens the sent capture, never a live task with a similar label. */
export function FollowUpOrigin({ value }: { value: RunFollowUp }) {
  const [expanded, setExpanded] = useState<string | null>(null);
  return (
    <details className="follow-up-origin">
      <summary>Following {value.runName} · View original evidence</summary>
      <p>
        Saved observations from the earlier analysis. This link does not mean its results are
        reused.
      </p>
      <AttachmentList
        items={value.evidence}
        manifests={value.evidence}
        expanded={expanded}
        onExpand={setExpanded}
        label="Original analysis evidence"
        request={(attachmentId) => ({
          kind: 'sent',
          projectId: value.projectId,
          requestId: value.submissionId,
          attachmentId,
        })}
      />
    </details>
  );
}
