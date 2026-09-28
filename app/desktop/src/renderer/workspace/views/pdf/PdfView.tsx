import { usePageImage } from './usePageImage';
import { usePdfViewport } from './usePdfViewport';
import { PdfMarks } from './PdfMarks';
import type { SharedReference } from '@gobble/contracts';
import { useEffect, useRef, useState, type KeyboardEvent } from 'react';
import {
  defaultPdfNavigation,
  validatePdfSelection,
  type PdfContent,
  type PdfNavigation,
  type PdfSelection,
  type EvidenceRef,
  type RenderAcknowledgment,
  type Surface,
} from '@gobble/contracts';
import type { Rect } from '@gobble/contracts/pdf-decoder';
import { pixelRegion, regionFromPixels } from '@gobble/contracts/pdf-geometry';
import type { Command } from '../../useWorkspace';
import { SelectionToolbar } from '../../../selections/SelectionToolbar';
import '../../../styles/pdf.css';

export function PdfView({
  content,
  surface,
  title,
  evidence,
  acknowledgment,
  ready,
  onReady,
  onFailure,
  onSelect,
  onDiscuss,
  onNavigate,
  onRefresh,
  command,
  marks = [],
}: {
  marks?: SharedReference[];
  content: PdfContent;
  surface: Extract<Surface, { view: 'pdf' }>;
  title: string;
  evidence: EvidenceRef | null;
  acknowledgment: RenderAcknowledgment;
  ready: boolean;
  onReady: () => void;
  onFailure: (message: string) => void;
  onSelect: (selection: PdfSelection | null) => Promise<boolean>;
  onDiscuss: (selection: PdfSelection) => Promise<void>;
  onNavigate: (navigation: PdfNavigation) => Promise<boolean>;
  onRefresh: () => void;
  command: Command;
}) {
  const { page, pageCount } = content,
    { model, raster } = page;
  const navigation = surface.pdf ?? defaultPdfNavigation();
  const image = usePageImage(raster.png),
    area = useRef<HTMLDivElement>(null);
  const scroll = useRef<HTMLDivElement>(null),
    img = useRef<HTMLImageElement>(null);
  usePdfViewport(scroll, area, content, acknowledgment, ready);
  useEffect(() => {
    if (img.current?.complete && img.current.naturalWidth) onReady();
  }, [acknowledgment.requestId, image, onReady]);
  const [editing, setEditing] = useState(false),
    [draft, setDraft] = useState<Rect>();
  const [pageNumber, setPageNumber] = useState(String(model.pageIndex + 1));
  const drag = useRef<{ x: number; y: number }>(undefined);
  let selected: PdfSelection | undefined;
  if (evidence?.schemaVersion === 5 && evidence.dataRevision === model.revision) {
    try {
      validatePdfSelection(evidence.selection, content);
      selected = evidence.selection;
    } catch {}
  }
  const region = editing ? draft : selected?.region;
  const pixels = region
    ? pixelRegion(region, page.viewport, raster.width, raster.height)
    : undefined;
  const selection = (region: Rect, scope: 'region' | 'page'): PdfSelection => ({
    kind: 'pdf',
    coordinateSpace: 'pdf-user-space',
    scope,
    region,
    profile: model.profile,
    pageIndex: model.pageIndex,
    modelHash: page.modelHash,
  });
  function startRegion() {
    const [x, y, right, top] = model.viewBox,
      w = right - x,
      h = top - y;
    setDraft(selected?.region ?? [x + w / 4, y + h / 4, right - w / 4, top - h / 4]);
    setEditing(true);
    area.current?.focus();
  }
  function keyRegion(event: KeyboardEvent) {
    if (!editing || !draft || !ready) return;
    if (event.key === 'Escape') {
      setEditing(false);
      setDraft(undefined);
      return;
    }
    if (event.key === 'Enter') {
      event.preventDefault();
      void commit();
      return;
    }
    if (!['ArrowLeft', 'ArrowRight', 'ArrowUp', 'ArrowDown'].includes(event.key)) return;
    event.preventDefault();
    const [x, y, r, t] = draft,
      [left, bottom, right, top] = model.viewBox;
    // Keyboard directions are screen-relative, even after page rotation.
    const box = pixelRegion(draft, page.viewport, raster.width, raster.height);
    const dx = event.key === 'ArrowLeft' ? -8 : event.key === 'ArrowRight' ? 8 : 0;
    const dy = event.key === 'ArrowUp' ? -8 : event.key === 'ArrowDown' ? 8 : 0;
    const rect: Rect = event.shiftKey
      ? [box.x, box.y, box.x + Math.max(8, box.width + dx), box.y + Math.max(8, box.height + dy)]
      : [box.x + dx, box.y + dy, box.x + box.width + dx, box.y + box.height + dy];
    const next = regionFromPixels(rect, page.viewport);
    if (event.shiftKey)
      setDraft([
        Math.max(left, next[0]),
        Math.max(bottom, next[1]),
        Math.min(right, next[2]),
        Math.min(top, next[3]),
      ]);
    else {
      const sx = Math.max(left - x, Math.min(right - r, next[0] - x)),
        sy = Math.max(bottom - y, Math.min(top - t, next[1] - y));
      setDraft([x + sx, y + sy, r + sx, t + sy]);
    }
  }
  async function commit() {
    if (!ready || !draft) return;
    if (await onSelect(selection(draft, 'region'))) setEditing(false);
  }
  async function goPage() {
    const n = Number(pageNumber);
    if (Number.isInteger(n) && n >= 1 && n <= pageCount)
      await onNavigate({ ...navigation, pageIndex: n - 1 });
    else setPageNumber(String(model.pageIndex + 1));
  }
  function point(event: React.PointerEvent<HTMLDivElement>) {
    const box = event.currentTarget.getBoundingClientRect();
    return {
      x: Math.max(
        0,
        Math.min(raster.width, ((event.clientX - box.left) / box.width) * raster.width),
      ),
      y: Math.max(
        0,
        Math.min(raster.height, ((event.clientY - box.top) / box.height) * raster.height),
      ),
    };
  }
  function dragRegion(event: React.PointerEvent<HTMLDivElement>) {
    if (!drag.current) return;
    const end = point(event),
      start = drag.current;
    if (Math.abs(end.x - start.x) < 2 || Math.abs(end.y - start.y) < 2) return;
    const rect = regionFromPixels(
      [
        Math.min(start.x, end.x),
        Math.min(start.y, end.y),
        Math.max(start.x, end.x),
        Math.max(start.y, end.y),
      ],
      page.viewport,
    );
    const box = model.viewBox;
    setDraft([
      Math.max(box[0], rect[0]),
      Math.max(box[1], rect[1]),
      Math.min(box[2], rect[2]),
      Math.min(box[3], rect[3]),
    ]);
  }
  return (
    <section className="pdf-view" aria-label={title + ' PDF reader'} data-ready={ready}>
      <div className="pdf-toolbar" aria-label="PDF navigation">
        <button
          disabled={!ready || model.pageIndex === 0}
          onClick={() => void onNavigate({ ...navigation, pageIndex: model.pageIndex - 1 })}
          aria-label="Previous PDF page"
        >
          Previous
        </button>
        <form
          onSubmit={(event) => {
            event.preventDefault();
            void goPage();
          }}
        >
          <label>
            Page{' '}
            <input
              aria-label="PDF page number"
              type="number"
              min={1}
              max={pageCount}
              value={pageNumber}
              disabled={!ready}
              onChange={(event) => setPageNumber(event.target.value)}
            />
          </label>
          <span>of {pageCount}</span>
          <button disabled={!ready}>Go</button>
        </form>
        <button
          disabled={!ready || model.pageIndex + 1 >= pageCount}
          onClick={() => void onNavigate({ ...navigation, pageIndex: model.pageIndex + 1 })}
          aria-label="Next PDF page"
        >
          Next
        </button>
        <label className="sr-only" htmlFor={'pdf-zoom-' + surface.surfaceId}>
          PDF zoom
        </label>
        <select
          id={'pdf-zoom-' + surface.surfaceId}
          aria-label="PDF zoom"
          disabled={!ready}
          value={navigation.fitWidth !== false ? 'fit' : navigation.scale}
          onChange={(event) =>
            void onNavigate({
              ...navigation,
              fitWidth: event.target.value === 'fit',
              scale:
                event.target.value === 'fit'
                  ? 1
                  : (Number(event.target.value) as PdfNavigation['scale']),
            })
          }
        >
          <option value="fit">Fit width</option>
          {[0.75, 1, 1.25, 1.5].map((n) => (
            <option key={n} value={n}>
              {n * 100}%
            </option>
          ))}
        </select>
        <button
          disabled={!ready}
          onClick={() =>
            void onNavigate({
              ...navigation,
              rotation: ((navigation.rotation + 90) % 360) as PdfNavigation['rotation'],
            })
          }
        >
          Rotate
        </button>
        <button disabled={!ready} onClick={onRefresh}>
          Refresh PDF
        </button>
      </div>
      <details className="pdf-support">
        <summary>Read-only PDF · Page and region capture</summary>
        <p>
          Text selection, form and annotation appearance, interactive content and password-protected
          files are not supported. Select a page or region to share its exact displayed pixels.
          Refresh reads a newer source version.
        </p>
      </details>
      {selected && !editing && evidence && (
        <SelectionToolbar
          evidence={evidence}
          surfaceId={surface.surfaceId}
          title={title}
          ready={ready}
          command={command}
          acknowledgment={acknowledgment}
        />
      )}
      <div ref={scroll} className="pdf-scroll">
        <div
          ref={area}
          className={'pdf-page' + (editing ? ' selecting' : '')}
          tabIndex={0}
          role="group"
          aria-label={'PDF page ' + (model.pageIndex + 1) + ' selection area'}
          aria-describedby={'pdf-help-' + surface.surfaceId}
          onKeyDown={keyRegion}
          style={{ width: raster.width, maxWidth: navigation.fitWidth !== false ? '100%' : 'none' }}
          onPointerDown={(event) => {
            if (!ready || !editing || event.button !== 0) return;
            drag.current = point(event);
            event.currentTarget.setPointerCapture(event.pointerId);
            event.currentTarget.focus();
            event.preventDefault();
          }}
          onPointerMove={dragRegion}
          onPointerUp={(event) => {
            dragRegion(event);
            drag.current = undefined;
            if (event.currentTarget.hasPointerCapture(event.pointerId))
              event.currentTarget.releasePointerCapture(event.pointerId);
          }}
          onPointerCancel={() => {
            drag.current = undefined;
          }}
        >
          {image && (
            <img
              ref={img}
              src={image}
              alt={'Rendered PDF page ' + (model.pageIndex + 1) + ' of ' + title}
              draggable={false}
              onLoad={onReady}
              onError={() =>
                onFailure('The PDF page image could not be displayed. Retry the view.')
              }
            />
          )}
          <PdfMarks content={content} marks={marks} />
          {pixels && (
            <div
              className="pdf-region"
              aria-hidden="true"
              style={{
                left: (pixels.x / raster.width) * 100 + '%',
                top: (pixels.y / raster.height) * 100 + '%',
                width: (pixels.width / raster.width) * 100 + '%',
                height: (pixels.height / raster.height) * 100 + '%',
              }}
            />
          )}
        </div>
      </div>
      <div className="pdf-selection-controls">
        <span id={'pdf-help-' + surface.surfaceId}>
          {editing
            ? 'Drag a region, or use arrows to move · Shift+arrows resize · Enter applies · Esc cancels'
            : 'Select part of this page to discuss it.'}
        </span>
        {editing ? (
          <>
            <button disabled={!ready || !draft} onClick={() => void commit()}>
              Use region
            </button>
            <button
              onClick={() => {
                setEditing(false);
                setDraft(undefined);
              }}
            >
              Cancel selection
            </button>
          </>
        ) : (
          <>
            <button disabled={!ready} onClick={startRegion}>
              Select region
            </button>
            <button
              disabled={!ready}
              onClick={() => void onDiscuss(selection(model.viewBox, 'page'))}
            >
              Add page to message
            </button>
          </>
        )}
      </div>
    </section>
  );
}
