import { useEffect, useRef, useState } from 'react';
import { PROFILE, type PartAddress, type Target, type Rect } from './model';
import { textSelection } from './visibility';
import './bridge';
import { textWindow } from './text-window';
export function TextContent({
  text,
  address,
  revision,
  partKey,
  select,
}: {
  text: string;
  address: PartAddress;
  revision: string;
  partKey: string;
  select: (target: Target) => void;
}) {
  const ref = useRef<HTMLPreElement>(null);
  const [windowStart, setWindowStart] = useState(0);
  const history = useRef<number[]>([]);
  const segment = textWindow(text, windowStart);
  useEffect(() => {
    window.notebook.invalidate();
    ref.current?.dispatchEvent(new Event('scroll'));
  }, [windowStart]);
  let offset = windowStart;
  const lines = segment.text.match(/[^\n]*\n|[^\n]+$/g) ?? [];
  const [start, setStart] = useState(0),
    [end, setEnd] = useState(
      Math.min(text.length, text.indexOf('\n') < 0 ? text.length : text.indexOf('\n')),
    );
  const useRange = (a: number, b: number) =>
    select({
      ...address,
      revision,
      profile: PROFILE,
      selector: { kind: 'text', start: a, end: b },
    });
  return (
    <div className="text-content">
      <pre ref={ref} data-part={partKey} tabIndex={0} aria-label="Selectable saved text">
        {lines.map((line, i) => {
          const at = offset;
          offset += line.length;
          return (
            <span key={i} data-start={at}>
              {line}
            </span>
          );
        })}
      </pre>
      {(windowStart > 0 || segment.end < text.length) && (
        <div className="paging">
          <button
            disabled={!history.current.length}
            onClick={() => setWindowStart(history.current.pop() ?? 0)}
          >
            Previous text
          </button>
          <span>
            UTF-16 {windowStart}–{segment.end} of {text.length}
          </span>
          <button
            disabled={segment.end === text.length}
            onClick={() => {
              history.current.push(windowStart);
              setWindowStart(segment.end);
            }}
          >
            Next text
          </button>
        </div>
      )}
      <div className="part-actions">
        <button
          onClick={() => {
            if (ref.current) {
              const s = textSelection(ref.current);
              if (s) useRange(s.start + windowStart, s.end + windowStart);
            }
          }}
        >
          Use text selection
        </button>
        <details>
          <summary>Keyboard range</summary>
          <div className="range-fields">
            <label>
              Start UTF-16
              <input
                type="number"
                min={0}
                max={text.length}
                value={start}
                onChange={(e) => setStart(Number(e.target.value))}
              />
            </label>
            <label>
              End UTF-16
              <input
                type="number"
                min={0}
                max={text.length}
                value={end}
                onChange={(e) => setEnd(Number(e.target.value))}
              />
            </label>
            <button onClick={() => useRange(start, end)}>Use range</button>
          </div>
        </details>
      </div>
    </div>
  );
}
export function ImageContent({
  address,
  revision,
  partKey,
  width,
  height,
  onChange,
  select,
  active,
  activate,
}: {
  active: boolean;
  activate: () => void;
  address: PartAddress;
  revision: string;
  partKey: string;
  width: number;
  height: number;
  onChange: () => void;
  select: (target: Target) => void;
}) {
  const [url, setUrl] = useState(''),
    [problem, setProblem] = useState(''),
    [rect, setRect] = useState<Rect>({
      x: 0,
      y: 0,
      width: Math.min(80, width),
      height: Math.min(40, height),
    }),
    [scale, setScale] = useState(100);
  const attempt = useRef(0);
  useEffect(() => {
    if (!active) {
      attempt.current++;
      setUrl('');
    }
  }, [active]);
  const image = useRef<HTMLImageElement>(null),
    drag = useRef<{ x: number; y: number } | null>(null);
  const target = (r: Rect): Target => ({
    ...address,
    revision,
    profile: PROFILE,
    selector: { kind: 'image', rect: r },
  });
  async function open() {
    activate();
    const current = ++attempt.current;
    const result = await window.notebook.image(target({ x: 0, y: 0, width, height }));
    if (attempt.current !== current) return;
    if (result.ok) {
      setUrl('data:image/png;base64,' + result.value.base64);
      onChange();
    } else setProblem(result.message);
  }
  const point = (x: number, y: number) => {
    const b = image.current?.getBoundingClientRect();
    if (!b) return { x: 0, y: 0 };
    return {
      x: Math.max(0, Math.min(width, Math.round(((x - b.left) * width) / b.width))),
      y: Math.max(0, Math.min(height, Math.round(((y - b.top) * height) / b.height))),
    };
  };
  return (
    <div className="image-content">
      <div className="part-actions">
        {url ? (
          <button
            onClick={() => {
              setUrl('');
              onChange();
            }}
          >
            Hide image
          </button>
        ) : (
          <button onClick={() => open().catch((e) => setProblem(String(e)))}>
            View saved image
          </button>
        )}
        <span>
          {width} × {height} pixels
        </span>
        {url && (
          <label>
            Scale
            <select
              aria-label="Image scale"
              value={scale}
              onChange={(e) => {
                setScale(Number(e.target.value));
                onChange();
              }}
            >
              <option value={50}>50%</option>
              <option value={100}>100%</option>
              <option value={150}>150%</option>
            </select>
          </label>
        )}
      </div>
      {problem && <p role="alert">{problem}</p>}
      {url && active && (
        <>
          <div className="image-scroll" onScroll={onChange}>
            <img
              ref={image}
              src={url}
              alt="Saved Notebook output"
              data-part={partKey}
              draggable={false}
              style={{ width: (width * scale) / 100, height: (height * scale) / 100 }}
              onLoad={onChange}
              onPointerDown={(e) => {
                e.preventDefault();
                drag.current = point(e.clientX, e.clientY);
                e.currentTarget.setPointerCapture(e.pointerId);
              }}
              onPointerUp={(e) => {
                const a = drag.current,
                  b = point(e.clientX, e.clientY);
                drag.current = null;
                if (a) {
                  const r = {
                    x: Math.min(a.x, b.x),
                    y: Math.min(a.y, b.y),
                    width: Math.abs(a.x - b.x),
                    height: Math.abs(a.y - b.y),
                  };
                  setRect(r);
                  if (r.width && r.height) select(target(r));
                }
              }}
              onPointerCancel={() => {
                drag.current = null;
              }}
            />
          </div>
          <details>
            <summary>Select region by coordinates</summary>
            <div className="range-fields">
              {(['x', 'y', 'width', 'height'] as const).map((key) => (
                <label key={key}>
                  {key}
                  <input
                    aria-label={'Region ' + key}
                    type="number"
                    value={rect[key]}
                    onChange={(e) => setRect({ ...rect, [key]: Number(e.target.value) })}
                  />
                </label>
              ))}
              <button onClick={() => select(target(rect))}>Use image region</button>
            </div>
          </details>
        </>
      )}
    </div>
  );
}
export function MarkdownPreview({ text }: { text: string }) {
  // Minimal passive blocks: no HTML parsing, links, images, math or inline interpreter.
  return (
    <div className="markdown">
      {text
        .split('\n')
        .map((line, i) =>
          line.startsWith('# ') ? (
            <h2 key={i}>{line.slice(2)}</h2>
          ) : (
            <p key={i}>{line || '\u00a0'}</p>
          ),
        )}
      <small>
        Basic saved Markdown (bounded preview) · Use Source for exact text and continuation
      </small>
    </div>
  );
}
