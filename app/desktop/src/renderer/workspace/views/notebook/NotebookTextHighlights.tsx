import { Fragment } from 'react';
import { referenceAuthor, type NotebookSelection, type SharedReference } from '@gobble/contracts';
import { markStyle } from '../../../shared-context/marks';
/** Segment one bounded text window while preserving its exact DOM text offsets. */
export function NotebookTextHighlights({
  text,
  start,
  end,
  selected,
  marks,
}: {
  text: string;
  start: number;
  end: number;
  selected: Extract<NotebookSelection['selector'], { kind: 'text' }> | undefined;
  marks: SharedReference[];
}) {
  const ranges = marks.flatMap((mark) =>
    mark.evidence.schemaVersion === 6 && mark.evidence.selection.selector.kind === 'text'
      ? [{ mark, range: mark.evidence.selection.selector }]
      : [],
  );
  const points = new Set([start, end]);
  for (const range of [...ranges.map((item) => item.range), ...(selected ? [selected] : [])])
    for (const at of [range.start, range.end]) if (at > start && at < end) points.add(at);
  const sorted = [...points].sort((a, b) => a - b);
  return sorted.slice(0, -1).map((a, i) => {
    const b = sorted[i + 1]!,
      authored = ranges
        .filter((item) => item.range.start <= a && item.range.end >= b)
        .map((item) => item.mark),
      local = selected && selected.start <= a && selected.end >= b;
    const value = text.slice(a, b);
    return authored.length ? (
      <mark
        key={a}
        className={'notebook-authored-text' + (local ? ' local-selected' : '')}
        data-reference-ids={authored.map((m) => m.referenceId).join(' ')}
        style={markStyle(authored[0]!)}
        title={authored.map((m) => referenceAuthor(m) + ': ' + m.note).join('\n')}
      >
        {value}
      </mark>
    ) : local ? (
      <mark key={a}>{value}</mark>
    ) : (
      <Fragment key={a}>{value}</Fragment>
    );
  });
}
