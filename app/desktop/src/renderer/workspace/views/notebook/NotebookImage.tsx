import { referenceAuthor } from '@gobble/contracts';
import { markStyle } from '../../../shared-context/marks';
import { useEffect, useRef, useState } from 'react';
import type {
  SharedReference,
  NotebookTarget,
  NotebookSelection,
  RenderAcknowledgment,
} from '@gobble/contracts';
type Rect = Extract<NotebookSelection['selector'], { kind: 'image' }>['rect'];

/** One on-demand Blob per reader; image intent is never used as evidence bytes. */
export function NotebookImage({
  target,
  marks,
  width,
  height,
  acknowledgment,
  active,
  onActivate,
  ready,
  onSelect,
}: {
  target: NotebookTarget;
  marks: SharedReference[];
  width: number;
  height: number;
  acknowledgment: RenderAcknowledgment;
  active: boolean;
  onActivate: (active: boolean) => void;
  ready: boolean;
  onSelect: (selector: NotebookSelection['selector']) => Promise<boolean>;
}) {
  const [url, setUrl] = useState<string>();
  const [problem, setProblem] = useState(''),
    [retry, setRetry] = useState(0);
  const [painted, setPainted] = useState(false),
    [scale, setScale] = useState(1);
  const [editing, setEditing] = useState(false),
    [rect, setRect] = useState<Rect>();
  const drag = useRef<{ x: number; y: number } | undefined>(undefined);
  const area = useRef<HTMLDivElement>(null);
  const targetKey = JSON.stringify(target);
  useEffect(() => {
    setUrl(undefined);
    setPainted(false);
    setProblem('');
    setEditing(false);
    setRect(undefined);
    if (!active) return;
    let current = true,
      objectURL: string | undefined;
    window.gobble.workspace
      .notebookImage({ target, acknowledgment })
      .then((result) => {
        if (!current) return;
        if (!result.ok) {
          setProblem(result.error.message);
          return;
        }
        const bytes = Uint8Array.from(atob(result.value.base64), (c) => c.charCodeAt(0));
        objectURL = URL.createObjectURL(new Blob([bytes], { type: 'image/png' }));
        setUrl(objectURL);
      })
      .catch(() => {
        if (current) setProblem('This saved image could not be loaded. Try again.');
      });
    return () => {
      current = false;
      if (objectURL) URL.revokeObjectURL(objectURL);
    };
  }, [active, targetKey, retry]);
  const point = (x: number, y: number) => {
    const box = area.current?.getBoundingClientRect();
    return box
      ? {
          x: Math.max(0, Math.min(width, Math.round(((x - box.left) / box.width) * width))),
          y: Math.max(0, Math.min(height, Math.round(((y - box.top) / box.height) * height))),
        }
      : { x: 0, y: 0 };
  };
  function update(x: number, y: number) {
    const a = drag.current,
      b = point(x, y);
    if (a)
      setRect({
        x: Math.min(a.x, b.x),
        y: Math.min(a.y, b.y),
        width: Math.abs(a.x - b.x),
        height: Math.abs(a.y - b.y),
      });
  }
  async function commit() {
    if (!ready || !painted || !rect?.width || !rect.height) return;
    try {
      if (await onSelect({ kind: 'image', coordinateSpace: 'natural-image-pixels', rect }))
        setEditing(false);
    } catch {
      setProblem('Selection could not be saved. Try again.');
    }
  }
  function startRegion() {
    setRect({ x: 0, y: 0, width: Math.min(120, width), height: Math.min(80, height) });
    setEditing(true);
    area.current?.focus();
  }
  return (
    <div className="notebook-image">
      <div className="notebook-image-actions">
        <button disabled={!ready} onClick={() => onActivate(!active)}>
          {active ? 'Hide image' : 'View saved image'}
        </button>
        <span>
          {width} × {height} pixels
        </span>
        {active && painted && (
          <>
            <label>
              Zoom{' '}
              <select
                aria-label="Notebook image zoom"
                value={scale}
                onChange={(e) => setScale(Number(e.target.value))}
              >
                <option value={0.5}>50%</option>
                <option value={1}>100%</option>
                <option value={1.5}>150%</option>
              </select>
            </label>
            {!editing && (
              <button disabled={!ready} onClick={startRegion}>
                Select image region
              </button>
            )}
          </>
        )}
      </div>
      {active && !url && !problem && <p role="status">Loading saved image…</p>}
      {problem && (
        <div role="alert">
          {problem}
          <button onClick={() => setRetry((n) => n + 1)}>Retry image</button>
        </div>
      )}
      {active && url && (
        <>
          <div className="notebook-image-scroll">
            <div
              ref={area}
              data-notebook-selection={JSON.stringify(target.selection)}
              data-image-painted={painted}
              tabIndex={0}
              role="group"
              aria-label="Notebook image selection area"
              className={'notebook-image-canvas' + (editing ? ' selecting' : '')}
              style={{ width: width * scale, height: height * scale }}
              onKeyDown={(event) => {
                if (!editing || !rect || !ready) return;
                if (event.key === 'Escape') {
                  setEditing(false);
                  setRect(undefined);
                  event.preventDefault();
                  return;
                }
                if (event.key === 'Enter') {
                  event.preventDefault();
                  void commit();
                  return;
                }
                const [dx, dy] =
                  event.key === 'ArrowLeft'
                    ? [-1, 0]
                    : event.key === 'ArrowRight'
                      ? [1, 0]
                      : event.key === 'ArrowUp'
                        ? [0, -1]
                        : event.key === 'ArrowDown'
                          ? [0, 1]
                          : [0, 0];
                if (!dx && !dy) return;
                event.preventDefault();
                setRect(
                  event.shiftKey
                    ? {
                        ...rect,
                        width: Math.max(1, Math.min(width - rect.x, rect.width + dx!)),
                        height: Math.max(1, Math.min(height - rect.y, rect.height + dy!)),
                      }
                    : {
                        ...rect,
                        x: Math.max(0, Math.min(width - rect.width, rect.x + dx!)),
                        y: Math.max(0, Math.min(height - rect.height, rect.y + dy!)),
                      },
                );
              }}
              onPointerDown={(e) => {
                if (!editing || !ready || e.button !== 0) return;
                e.preventDefault();
                drag.current = point(e.clientX, e.clientY);
                e.currentTarget.setPointerCapture(e.pointerId);
                e.currentTarget.focus();
              }}
              onPointerMove={(e) => update(e.clientX, e.clientY)}
              onPointerUp={(e) => {
                update(e.clientX, e.clientY);
                drag.current = undefined;
                if (e.currentTarget.hasPointerCapture(e.pointerId))
                  e.currentTarget.releasePointerCapture(e.pointerId);
              }}
              onPointerCancel={() => {
                drag.current = undefined;
              }}
            >
              <img
                src={url}
                alt="Saved Notebook output"
                draggable={false}
                onLoad={() => setPainted(true)}
                onError={() => {
                  setPainted(false);
                  setProblem('This saved image could not be displayed.');
                }}
              />
              {marks.map((mark) => {
                const selector =
                  mark.evidence.selection?.kind === 'notebook'
                    ? mark.evidence.selection.selector
                    : undefined;
                if (selector?.kind !== 'image') return null;
                const r = selector.rect;
                return (
                  <div
                    key={mark.referenceId}
                    className="notebook-authored-image"
                    data-reference-ids={mark.referenceId}
                    title={referenceAuthor(mark) + ': ' + mark.note}
                    style={{
                      ...markStyle(mark),
                      left: (r.x / width) * 100 + '%',
                      top: (r.y / height) * 100 + '%',
                      width: (r.width / width) * 100 + '%',
                      height: (r.height / height) * 100 + '%',
                    }}
                  >
                    <span>{referenceAuthor(mark)}</span>
                  </div>
                );
              })}
              {rect && (
                <div
                  className="notebook-image-region"
                  aria-hidden="true"
                  style={{
                    left: (rect.x / width) * 100 + '%',
                    top: (rect.y / height) * 100 + '%',
                    width: (rect.width / width) * 100 + '%',
                    height: (rect.height / height) * 100 + '%',
                  }}
                />
              )}
            </div>
          </div>
          {editing && (
            <div className="notebook-image-actions">
              <small>Drag a region · Arrows move · Shift+arrows resize</small>
              <button
                disabled={!ready || !rect?.width || !rect.height}
                onClick={() => void commit()}
              >
                Use image region
              </button>
              <button
                onClick={() => {
                  setEditing(false);
                  setRect(undefined);
                }}
              >
                Cancel selection
              </button>
            </div>
          )}
        </>
      )}
    </div>
  );
}
