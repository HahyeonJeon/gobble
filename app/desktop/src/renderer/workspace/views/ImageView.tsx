import { useEffect, useRef, useState, type PointerEvent } from 'react';
import { markStyle, type MarkProps } from '../../shared-context/marks';
import { referenceAuthor, type FileContent, type Selection } from '@gobble/contracts';
type Image = Extract<FileContent['content'], { kind: 'image' }>;
type Region = { x: number; y: number; width: number; height: number };
const full: Region = { x: 0, y: 0, width: 1, height: 1 };
export function ImageView({
  content,
  revision,
  title,
  selection,
  onReady,
  onSelect,
  canSelect,
  marks = [],
}: MarkProps & {
  content: Image;
  revision: string;
  title: string;
  selection: Selection | undefined;
  onReady: () => void;
  onSelect: (selection: Selection | null) => Promise<boolean>;
  canSelect: boolean;
}) {
  const [url, setUrl] = useState<string | null>(null);
  const [failed, setFailed] = useState(false);
  const [ready, setReady] = useState(false);
  const [region, setRegion] = useState<Region>(selection?.kind === 'image' ? selection : full);
  const viewport = useRef<HTMLDivElement>(null);
  const [available, setAvailable] = useState({ width: content.width, height: content.height });
  const [dragging, setDragging] = useState(false);
  const [editing, setEditing] = useState(false);
  const start = useRef<{ x: number; y: number } | null>(null);
  useEffect(() => {
    const bytes = Uint8Array.from(atob(content.base64), (character) => character.charCodeAt(0));
    const source = URL.createObjectURL(new Blob([bytes], { type: content.mediaType }));
    setUrl(source);
    return () => URL.revokeObjectURL(source);
  }, [content]);
  useEffect(() => {
    const element = viewport.current;
    if (!element) return;
    const observer = new ResizeObserver(([entry]) => {
      if (entry && entry.contentRect.width > 0 && entry.contentRect.height > 0)
        setAvailable({ width: entry.contentRect.width, height: entry.contentRect.height });
    });
    observer.observe(element);
    return () => observer.disconnect();
  }, []);
  const scale = Math.min(1, available.width / content.width, available.height / content.height);
  function select(value: Region) {
    return onSelect({
      kind: 'image',
      coordinateSpace: 'normalized-original-image',
      ...value,
      originalWidth: content.width,
      originalHeight: content.height,
      contentHash: revision,
    });
  }
  function point(event: PointerEvent<HTMLImageElement>) {
    const bounds = event.currentTarget.getBoundingClientRect();
    return {
      x: Math.max(0, Math.min(1, (event.clientX - bounds.left) / bounds.width)),
      y: Math.max(0, Math.min(1, (event.clientY - bounds.top) / bounds.height)),
    };
  }
  const valid =
    region.width > 0 &&
    region.height > 0 &&
    region.x + region.width <= 1 &&
    region.y + region.height <= 1;
  return (
    <div className="data-view">
      <div className="image-scroll" ref={viewport}>
        {failed ? (
          <p className="inline-error" role="alert">
            This image could not be decoded. Its original file is unchanged.
          </p>
        ) : (
          url && (
            <div
              className="image-frame"
              style={{ width: content.width * scale, height: content.height * scale }}
            >
              <img
                src={url}
                alt={title}
                draggable={false}
                onLoad={() => {
                  setReady(true);
                  onReady();
                }}
                onError={() => setFailed(true)}
                onPointerDown={(event) => {
                  if (!ready || !canSelect) return;
                  event.preventDefault();
                  start.current = point(event);
                  event.currentTarget.setPointerCapture(event.pointerId);
                  setDragging(true);
                }}
                onPointerMove={(event) => {
                  const first = start.current;
                  if (!first) return;
                  const next = point(event);
                  setRegion({
                    x: Math.min(first.x, next.x),
                    y: Math.min(first.y, next.y),
                    width: Math.abs(first.x - next.x),
                    height: Math.abs(first.y - next.y),
                  });
                }}
                onPointerUp={(event) => {
                  if (!start.current) return;
                  const first = start.current,
                    last = point(event);
                  start.current = null;
                  setDragging(false);
                  event.currentTarget.releasePointerCapture(event.pointerId);
                  const selected = {
                    x: Math.min(first.x, last.x),
                    y: Math.min(first.y, last.y),
                    width: Math.abs(first.x - last.x),
                    height: Math.abs(first.y - last.y),
                  };
                  if (selected.width > 0.002 && selected.height > 0.002) {
                    setRegion(selected);
                    void select(selected);
                  }
                }}
                onPointerCancel={() => {
                  start.current = null;
                  setDragging(false);
                  setRegion(selection?.kind === 'image' ? selection : full);
                }}
              />
              {marks.map((reference) => {
                const selected = reference.evidence.selection;
                if (selected?.kind !== 'image') return null;
                return (
                  <div
                    key={reference.referenceId}
                    data-reference-id={reference.referenceId}
                    className="shared-image-mark"
                    style={{
                      ...markStyle(reference),
                      left: selected.x * 100 + '%',
                      top: selected.y * 100 + '%',
                      width: selected.width * 100 + '%',
                      height: selected.height * 100 + '%',
                    }}
                  >
                    <span>{referenceAuthor(reference)}</span>
                  </div>
                );
              })}
              {(selection?.kind === 'image' || editing || dragging) && (
                <div
                  className="image-selection"
                  aria-hidden="true"
                  style={{
                    left: region.x * 100 + '%',
                    top: region.y * 100 + '%',
                    width: region.width * 100 + '%',
                    height: region.height * 100 + '%',
                  }}
                />
              )}
            </div>
          )
        )}
      </div>
      {editing && (
        <fieldset className="region-editor">
          <legend>Image region (%)</legend>
          {(['x', 'y', 'width', 'height'] as const).map((key) => (
            <label key={key}>
              {key === 'x' ? 'Left' : key === 'y' ? 'Top' : key === 'width' ? 'Width' : 'Height'}
              <input
                type="number"
                min={0}
                max={100}
                step={1}
                value={Math.round(region[key] * 100)}
                onChange={(event) => {
                  const value = event.currentTarget.valueAsNumber;
                  if (Number.isFinite(value))
                    setRegion((previous) => ({
                      ...previous,
                      [key]: Math.max(0, Math.min(100, value)) / 100,
                    }));
                }}
              />
            </label>
          ))}
          <button disabled={!ready || !canSelect || !valid} onClick={() => void select(region)}>
            Use region
          </button>
        </fieldset>
      )}
      <footer className="view-footer">
        <span>
          Image preview · {content.width} × {content.height}
        </span>
        <button
          disabled={!ready || !canSelect || failed}
          onClick={() => setEditing((value) => !value)}
        >
          {editing ? 'Hide region controls' : 'Select region'}
        </button>
      </footer>
    </div>
  );
}
