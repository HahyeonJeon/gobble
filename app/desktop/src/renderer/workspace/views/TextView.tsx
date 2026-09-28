import { useEffect, useRef, useState } from 'react';
import { moveTextRange } from '../../shared-context/text-range';
import { TextHighlights, type MarkProps } from '../../shared-context/marks';
import type { Selection } from '@gobble/contracts';

function position(text: string, offset: number) {
  const lines = text.slice(0, offset).split('\n');
  return { line: lines.length, column: lines.at(-1)?.length ?? 0 };
}
export function TextView({
  text,
  label,
  footer,
  selection,
  onReady,
  onSelect,
  canSelect,
  marks = [],
  reveal,
}: MarkProps & {
  text: string;
  label: string;
  footer: string;
  selection: Selection | undefined;
  onReady: () => void;
  onSelect: (selection: Selection | null) => Promise<boolean>;
  canSelect: boolean;
}) {
  const element = useRef<HTMLTextAreaElement>(null);
  const mirror = useRef<HTMLPreElement>(null);
  const [hasSelection, setHasSelection] = useState(false);
  useEffect(onReady, [onReady]);
  useEffect(() => {
    if (selection?.kind !== 'text' || !element.current) return;
    const lines = text.split('\n');
    const offset = (point: { line: number; column: number }) =>
      lines.slice(0, point.line - 1).reduce((total, line) => total + line.length + 1, 0) +
      point.column;
    // Restore the range without taking keyboard focus from another pane or the composer.
    element.current.setSelectionRange(offset(selection.start), offset(selection.end));
  }, [text, selection]);
  useEffect(() => {
    if (!reveal || !element.current || !mirror.current) return;
    const marker = [...mirror.current.querySelectorAll<HTMLElement>('[data-reference-ids]')].find(
      (item) => item.dataset.referenceIds?.split(' ').includes(reveal.referenceId),
    );
    if (marker) {
      element.current.scrollTop = Math.max(0, marker.offsetTop - 24);
      element.current.scrollLeft = Math.max(0, marker.offsetLeft - 24);
      mirror.current.scrollTop = element.current.scrollTop;
      mirror.current.scrollLeft = element.current.scrollLeft;
    }
  }, [reveal?.requestId, text]);
  return (
    <div className="data-view">
      <div className="text-layer">
        <pre className="text-highlights" ref={mirror} aria-hidden="true">
          <TextHighlights text={text} marks={marks} />
          {'\n'}
        </pre>
        <textarea
          className="text-preview"
          ref={element}
          value={text}
          readOnly
          spellCheck={false}
          aria-label={label}
          onKeyDown={(event) => {
            if (
              event.altKey ||
              ((event.ctrlKey || event.metaKey) && !['Home', 'End'].includes(event.key))
            )
              return;
            const input = event.currentTarget;
            const next = moveTextRange(
              text,
              input.selectionStart,
              input.selectionEnd,
              input.selectionDirection,
              event.key,
              event.shiftKey,
              event.ctrlKey || event.metaKey,
            );
            if (!next) return;
            event.preventDefault();
            input.setSelectionRange(next.start, next.end, next.direction);
            setHasSelection(next.end > next.start);
          }}
          onScroll={(event) => {
            if (mirror.current) {
              mirror.current.scrollTop = event.currentTarget.scrollTop;
              mirror.current.scrollLeft = event.currentTarget.scrollLeft;
            }
          }}
          onSelect={(event) =>
            setHasSelection(event.currentTarget.selectionEnd > event.currentTarget.selectionStart)
          }
        />
      </div>
      <footer className="view-footer">
        <span>{footer}</span>
        <button
          disabled={!hasSelection || !canSelect}
          onClick={() => {
            const textarea = element.current;
            if (!textarea || textarea.selectionEnd <= textarea.selectionStart) return;
            void onSelect({
              kind: 'text',
              coordinateSpace: 'utf16-line-column',
              start: position(text, textarea.selectionStart),
              end: position(text, textarea.selectionEnd),
            });
          }}
        >
          Use text selection
        </button>
      </footer>
    </div>
  );
}
