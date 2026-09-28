import { TextHighlights } from '../../shared-context/marks';
import type { SharedReference } from '@gobble/contracts';
import { useEffect, useLayoutEffect, useRef, useState } from 'react';
import type { LogPresentation, Selection } from '@gobble/contracts';
import { moveTextRange } from '../../shared-context/text-range';
import { ObservationNotice } from './ObservationNotice';

type Stream = 'stdout' | 'stderr';
type LogSelection = Extract<Selection, { kind: 'log-text' }>;
const position = (text: string, offset: number) => {
  const lines = text.slice(0, offset).split('\n');
  return { line: lines.length, column: lines.at(-1)?.length ?? 0 };
};
const offset = (text: string, point: LogSelection['start']) =>
  text
    .split('\n')
    .slice(0, point.line - 1)
    .reduce((total, line) => total + line.length + 1, 0) + point.column;

/** A stream-local gesture produces an addressed range; scroll and pointer state stay in React. */
export function LogView({
  marks = [],
  value,
  stream,
  revision,
  selection,
  ready,
  onReady,
  onSelect,
  onDiscuss,
  onStream,
  onRefresh,
  refreshProblem,
}: {
  marks?: SharedReference[];
  value: LogPresentation;
  stream: Stream;
  revision: string;
  selection: Selection | undefined;
  ready: boolean;
  onReady: () => void;
  onSelect: (selection: Selection | null) => Promise<boolean>;
  onDiscuss: (selection: Selection) => Promise<void>;
  onStream: (stream: Stream) => Promise<boolean>;
  onRefresh: () => void;
  refreshProblem: string | undefined;
}) {
  const mirror = useRef<HTMLPreElement>(null);
  const textarea = useRef<HTMLTextAreaElement>(null);
  const positions = useRef<Partial<Record<Stream, { top: number; left: number }>>>({});
  const currentSelection = useRef(selection);
  currentSelection.current = selection;
  const [range, setRange] = useState<LogSelection | null>(null);
  const observed = value.streams[stream];
  const text = observed.text;
  useEffect(onReady, [onReady]);
  // Restore only when entering a stream or replacing its source, never during a native drag.
  useLayoutEffect(() => {
    const input = textarea.current;
    if (!input) return;
    const saved = currentSelection.current;
    const matching = saved?.kind === 'log-text' && saved.stream === stream ? saved : null;
    input.setSelectionRange(
      matching ? offset(text, matching.start) : 0,
      matching ? offset(text, matching.end) : 0,
    );
    setRange(matching);
    input.scrollTop = positions.current[stream]?.top ?? 0;
    input.scrollLeft = positions.current[stream]?.left ?? 0;
  }, [stream, revision, text]);
  function readRange(): LogSelection | null {
    const input = textarea.current;
    return input && input.selectionEnd > input.selectionStart
      ? {
          kind: 'log-text',
          coordinateSpace: 'decoded-preview-utf16-line-column',
          stream,
          start: position(text, input.selectionStart),
          end: position(text, input.selectionEnd),
        }
      : null;
  }
  function persistRange() {
    const selected = readRange();
    setRange(selected);
    const saved = currentSelection.current;
    if (ready && (selected || (saved?.kind === 'log-text' && saved.stream === stream)))
      void onSelect(selected);
  }
  return (
    <div className="data-view log-view">
      <div className="observation-toolbar">
        <div className="log-streams" role="group" aria-label="Log stream">
          {(['stderr', 'stdout'] as const).map((item) => (
            <button
              key={item}
              aria-pressed={item === stream}
              disabled={!ready}
              onClick={() => {
                if (item !== stream) void onStream(item);
              }}
            >
              {item}
              <span>
                {value.streams[item].availability === 'no-text-returned' ? ' · empty' : ''}
              </span>
            </button>
          ))}
        </div>
        <span className="log-attempt" title={value.instance}>
          {value.instance} · Attempt {value.attempt}
        </span>
        <button disabled={!ready} onClick={onRefresh}>
          Refresh
        </button>
      </div>
      <ObservationNotice observedAt={value.observedAt} problem={refreshProblem} ready={ready} />
      {value.error && <p className="log-source-error">{value.error}</p>}
      <div className="log-text-container">
        {!text && (
          <p className="observation-empty">
            No text returned for {stream}. Completeness is unknown.
          </p>
        )}
        <div className="text-layer log-layer">
          <pre className="text-highlights" ref={mirror} aria-hidden="true">
            <TextHighlights
              text={text}
              marks={marks.filter(
                (mark) =>
                  mark.evidence.selection?.kind === 'log-text' &&
                  mark.evidence.selection.stream === stream,
              )}
            />
            {'\n'}
          </pre>
          <textarea
            ref={textarea}
            className="text-preview log-text"
            aria-label={stream + ' log text'}
            value={text}
            readOnly
            spellCheck={false}
            onSelect={() => setRange(readRange())}
            onMouseUp={persistRange}
            onKeyUp={persistRange}
            onScroll={(event) => {
              if (mirror.current) {
                mirror.current.scrollTop = event.currentTarget.scrollTop;
                mirror.current.scrollLeft = event.currentTarget.scrollLeft;
              }
              positions.current[stream] = {
                top: event.currentTarget.scrollTop,
                left: event.currentTarget.scrollLeft,
              };
            }}
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
              const caret = next.direction === 'backward' ? next.start : next.end;
              const style = getComputedStyle(input);
              const lineHeight = parseFloat(style.lineHeight);
              const top =
                (position(text, caret).line - 1) * lineHeight + parseFloat(style.paddingTop);
              if (top < input.scrollTop) input.scrollTop = top;
              else if (top + lineHeight > input.scrollTop + input.clientHeight)
                input.scrollTop = top + lineHeight - input.clientHeight;
              setRange(readRange());
            }}
          />
        </div>
      </div>
      <div className="observation-actions">
        <div>
          {range ? (
            <>
              <strong>
                {stream} · Preview lines {range.start.line}–{range.end.line}
              </strong>
              <small>Selected text and attempt metadata will be attached</small>
            </>
          ) : (
            <span>Select text to discuss this {stream} preview.</span>
          )}
        </div>
        <button
          disabled={!ready || !range}
          onClick={() => {
            const selected = readRange();
            if (selected) void onDiscuss(selected);
          }}
        >
          Discuss selection
        </button>
      </div>
      <details className="observation-details">
        <summary>Up to {value.tailLimitBytes} bytes per stream · Observation details</summary>
        <dl>
          <dt>Source bytes reported</dt>
          <dd>{observed.sourceBytesReported ?? 'Unknown'}</dd>
          <dt>Decoded preview bytes</dt>
          <dd>{observed.decodedUtf8Bytes}</dd>
          <dt>Earlier bytes omitted</dt>
          <dd>{observed.earlierBytesOmitted ? 'Yes' : 'Not established'}</dd>
          <dt>Completeness</dt>
          <dd>Unknown · Preview line numbers are not source file offsets</dd>
          <dt>Engine revision</dt>
          <dd>{value.engineRevision}</dd>
        </dl>
      </details>
    </div>
  );
}
