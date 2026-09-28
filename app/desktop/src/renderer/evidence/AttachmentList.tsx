import type { DraftAttachment, EvidenceManifest, PreviewEvidence } from '@gobble/contracts';
import { ModalDialog } from '../ui/ModalDialog';
import { Icon } from '../workspace/Icon';
import { evidenceLabel } from '../shared-context/marks';
import { EvidencePreview } from './EvidencePreview';
import '../styles/evidence.css';

export function AttachmentList({
  items,
  manifests,
  expanded,
  onExpand,
  request,
  onRemove,
  disabled = false,
  label,
}: {
  items: DraftAttachment[];
  manifests: EvidenceManifest[];
  expanded: string | null;
  onExpand: (id: string | null) => void;
  request: (attachmentId: string) => PreviewEvidence | null;
  onRemove?: (id: string) => void;
  disabled?: boolean;
  label?: string;
}) {
  const manifest = manifests.find((item) => item.attachmentId === expanded);
  const previewRequest = expanded ? request(expanded) : null;
  return (
    <div
      className="attachment-list"
      aria-label={label ?? (onRemove ? 'Message attachments' : 'Sent attachments')}
    >
      <div className="attachment-chips">
        {items.map((item) => (
          <div className="attachment-chip" key={item.attachmentId}>
            <button
              type="button"
              className="attachment-toggle"
              aria-expanded={expanded === item.attachmentId}
              aria-haspopup="dialog"
              title={item.label + ' · ' + evidenceLabel(item.evidence)}
              disabled={!request(item.attachmentId)}
              onClick={() => onExpand(expanded === item.attachmentId ? null : item.attachmentId)}
            >
              <Icon
                name={
                  item.evidence.selection?.kind === 'image' ||
                  (item.evidence.selection?.kind === 'notebook' &&
                    item.evidence.selection.selector.kind === 'image')
                    ? 'image'
                    : 'file'
                }
              />
              <span>
                <strong>{item.label}</strong>
                <small>
                  {evidenceLabel(item.evidence)}
                  {item.presentation ? ' · Plot context' : ''}
                </small>
              </span>
              <span className="attachment-preview-hint" aria-hidden="true">
                Preview
              </span>
            </button>
            {onRemove && (
              <button
                className="icon-button small"
                aria-label={'Remove attachment ' + item.label}
                disabled={disabled}
                onClick={() => onRemove(item.attachmentId)}
              >
                <Icon name="close" />
              </button>
            )}
          </div>
        ))}
      </div>
      {manifest && previewRequest && (
        <ModalDialog
          title={'Attachment: ' + manifest.label}
          className="evidence-dialog"
          onClose={() => onExpand(null)}
        >
          <EvidencePreview
            key={JSON.stringify(previewRequest)}
            manifest={manifest}
            request={previewRequest}
          />
        </ModalDialog>
      )}
    </div>
  );
}
