import type { DraftAttachment, WorkspaceDocument } from '@gobble/contracts';
import { AppProblem } from '../problem';
import type { DraftConsumption } from './ports';

export function invalidateAttachments(doc: WorkspaceDocument): void {
  doc.chat.attachmentRevision = (doc.chat.attachmentRevision ?? 0) + 1;
}
export function attachEvidence(doc: WorkspaceDocument, attachment: DraftAttachment): void {
  const items = doc.chat.attachments ?? [];
  if (items.length >= 16)
    throw new AppProblem(
      'unsupported',
      'A message can include up to 16 attachments. Remove an attachment first.',
    );
  doc.chat.attachments = [...items, attachment];
  invalidateAttachments(doc);
}
export function detachEvidence(doc: WorkspaceDocument, attachmentId: string): void {
  if (!doc.chat.attachments?.some((item) => item.attachmentId === attachmentId))
    throw new AppProblem('not_found', 'This draft attachment is no longer available.');
  doc.chat.attachments = doc.chat.attachments.filter((item) => item.attachmentId !== attachmentId);
  invalidateAttachments(doc);
}
export function consumeDraft(doc: WorkspaceDocument, consumed: string | DraftConsumption): void {
  if (typeof consumed === 'string') {
    if (doc.chat.attachments?.length)
      throw new AppProblem('invalid_request', 'Prepare the draft attachments before sending.');
    if (doc.chat.draft === consumed) doc.chat.draft = '';
    return;
  }
  consumed.assertCurrent();
  if (
    doc.chat.draft !== consumed.text ||
    doc.chat.recipientAgentId !== consumed.agentId ||
    (doc.chat.attachmentRevision ?? 0) !== consumed.attachmentRevision ||
    JSON.stringify((doc.chat.attachments ?? []).map((item) => item.attachmentId)) !==
      JSON.stringify(consumed.attachmentIds)
  )
    throw new AppProblem(
      'stale_revision',
      'The draft or recipient changed. Review the current attachments before sending.',
    );
  doc.chat.draft = '';
  doc.chat.attachments = [];
  invalidateAttachments(doc);
}
