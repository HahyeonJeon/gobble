import {
  referenceAuthor,
  type EvidenceRef,
  type SharedReference,
  type WorkspaceDocument,
} from '@gobble/contracts';
import type { CSSProperties } from 'react';
export type MarkProps = {
  marks?: SharedReference[];
  reveal?: WorkspaceDocument['referenceReveal'];
};
export function evidenceLabel(evidence: EvidenceRef): string {
  if (evidence.schemaVersion === 8) return 'Whole saved report';
  const selected = evidence.selection;
  if (!selected) return 'Whole view';
  switch (selected.kind) {
    case 'pipeline':
      return 'Pipeline ' + selected.subject.kind;
    case 'notebook': {
      const cell =
        selected.cell.kind === 'id' ? selected.cell.id : 'Cell ' + (selected.cell.index + 1);
      return (
        cell +
        ' · ' +
        (selected.part.kind === 'source' ? 'Source' : 'Output ' + (selected.part.index + 1)) +
        (selected.selector.kind === 'image' ? ' · Image region' : ' · Text range')
      );
    }
    case 'pdf':
      return (
        'PDF page ' + (selected.pageIndex + 1) + (selected.scope === 'region' ? ' · Region' : '')
      );
    case 'table':
      return (
        selected.rowKeys.length +
        ' ' +
        (selected.rowKeys.length === 1 ? 'row' : 'rows') +
        ' · ' +
        selected.columns.length +
        ' columns'
      );
    case 'image':
      return 'Image region';
    case 'run-group':
      return 'Group · ' + selected.taskId;
    case 'run-dependency':
      return selected.fromTaskId + ' → ' + selected.toTaskId;
    case 'run-task':
      return (
        'Task · ' +
        selected.instanceId +
        ' · ' +
        (selected.attempt ? 'Attempt ' + selected.attempt : 'Not started')
      );
    case 'log-text':
      return selected.stream + ' · Preview lines ' + selected.start.line + '–' + selected.end.line;
    case 'text':
      return (
        'Text · L' +
        selected.start.line +
        ':' +
        selected.start.column +
        '–L' +
        selected.end.line +
        ':' +
        selected.end.column
      );
  }
}
export const markStyle = (reference: SharedReference): CSSProperties =>
  ({ '--mark-color': reference.author.kind === 'user' ? '#137461' : '#905c16' }) as CSSProperties;
export function TextHighlights({ text, marks }: { text: string; marks: SharedReference[] }) {
  const lines = text.split('\n');
  const offset = (p: { line: number; column: number }) =>
    lines.slice(0, p.line - 1).reduce((sum, line) => sum + line.length + 1, 0) + p.column;
  const ranges = marks.flatMap((reference) => {
    const selection = reference.evidence.selection;
    return selection?.kind === 'text' || selection?.kind === 'log-text'
      ? [{ reference, start: offset(selection.start), end: offset(selection.end) }]
      : [];
  });
  const boundaries = [
    ...new Set([0, text.length, ...ranges.flatMap((range) => [range.start, range.end])]),
  ].sort((a, b) => a - b);
  return boundaries.slice(0, -1).map((start, index) => {
    const end = boundaries[index + 1]!;
    const matches = ranges.filter((range) => range.start <= start && range.end >= end);
    return matches.length ? (
      <mark
        key={start}
        data-reference-ids={matches.map((item) => item.reference.referenceId).join(' ')}
        style={markStyle(matches[0]!.reference)}
        title={matches.map((item) => referenceAuthor(item.reference)).join(', ')}
      >
        {text.slice(start, end)}
      </mark>
    ) : (
      text.slice(start, end)
    );
  });
}
