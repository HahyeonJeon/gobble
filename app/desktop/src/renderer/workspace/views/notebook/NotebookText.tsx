import { NotebookTextHighlights } from './NotebookTextHighlights';
import { useRef, useState } from 'react';
import type { NotebookSelection, SharedReference } from '@gobble/contracts';
import { textWindow } from './text-window';

/** DOM text is a bounded window; selection offsets always address the complete saved part. */
export function NotebookText({
  text,
  address,
  marks,
  label,
  ready,
  selected,
  onSelect,
}: {
  text: string;
  address: NotebookSelection;
  marks: SharedReference[];
  label: string;
  ready: boolean;
  selected: Extract<NotebookSelection['selector'], { kind: 'text' }> | undefined;
  onSelect: (selector: NotebookSelection['selector']) => Promise<boolean>;
}) {
  const ref = useRef<HTMLPreElement>(null);
  const [start, setStart] = useState(0);
  const history = useRef<number[]>([]);
  const segment = textWindow(text, start);
  const [from, setFrom] = useState('0'),
    [to, setTo] = useState(String(Math.min(text.length, 2048)));
  const [problem, setProblem] = useState('');
  async function select(a: number, b: number) {
    if (!ready) return;
    if (
      !Number.isSafeInteger(a) ||
      !Number.isSafeInteger(b) ||
      a < 0 ||
      b <= a ||
      b > text.length
    ) {
      setProblem('Choose a nonempty range inside this text.');
      return;
    }
    setProblem('');
    try {
      await onSelect({ kind: 'text', coordinateSpace: 'notebook-display-utf16', start: a, end: b });
    } catch {
      setProblem('Selection could not be saved. Try again.');
    }
  }
  function readSelection() {
    const element = ref.current,
      selection = window.getSelection();
    if (!ready || !element || !selection?.rangeCount || selection.isCollapsed) return;
    const range = selection.getRangeAt(0);
    if (!element.contains(range.startContainer) || !element.contains(range.endContainer)) return;
    const before = range.cloneRange();
    before.selectNodeContents(element);
    before.setEnd(range.startContainer, range.startOffset);
    const through = range.cloneRange();
    through.selectNodeContents(element);
    through.setEnd(range.endContainer, range.endOffset);
    void select(start + before.toString().length, start + through.toString().length).catch(() =>
      setProblem('Selection could not be saved. Try again.'),
    );
  }
  return (
    <div className="notebook-text">
      <pre
        ref={ref}
        data-notebook-selection={
          segment.text
            ? JSON.stringify({
                ...address,
                selector: {
                  kind: 'text',
                  coordinateSpace: 'notebook-display-utf16',
                  start,
                  end: segment.end,
                },
              })
            : undefined
        }
        aria-label={label}
        tabIndex={0}
        onPointerUp={readSelection}
        onKeyUp={readSelection}
      >
        <NotebookTextHighlights
          text={text}
          start={start}
          end={segment.end}
          selected={selected}
          marks={marks}
        />
      </pre>
      <div className="notebook-text-actions">
        <button disabled={!ready || !segment.text} onClick={() => void select(start, segment.end)}>
          Select displayed text
        </button>
        {(start > 0 || segment.end < text.length) && (
          <>
            <button
              disabled={!history.current.length}
              onClick={() => setStart(history.current.pop() ?? 0)}
            >
              Previous text
            </button>
            <span>
              Showing {start + 1}–{segment.end} of {text.length} characters
            </span>
            <button
              disabled={segment.end >= text.length}
              onClick={() => {
                history.current.push(start);
                setStart(segment.end);
              }}
            >
              Next text
            </button>
          </>
        )}
        <details className="notebook-range">
          <summary>Select a text range</summary>
          <form
            onSubmit={(event) => {
              event.preventDefault();
              void select(Number(from), Number(to));
            }}
          >
            <label>
              Start offset
              <input
                aria-label={label + ' start offset'}
                type="number"
                min={0}
                max={text.length}
                value={from}
                onChange={(e) => setFrom(e.target.value)}
              />
            </label>
            <label>
              End offset
              <input
                aria-label={label + ' end offset'}
                type="number"
                min={1}
                max={text.length}
                value={to}
                onChange={(e) => setTo(e.target.value)}
              />
            </label>
            <button disabled={!ready}>Use range</button>
            <small>UTF-16 offsets · End excluded</small>
          </form>
        </details>
      </div>
      {problem && <p role="alert">{problem}</p>}
    </div>
  );
}
