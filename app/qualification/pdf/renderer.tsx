import { useEffect, useRef, useState } from 'react';
import { createRoot } from 'react-dom/client';
import type { PageResult, CaptureResult, Rect } from './protocol';
import { regionFromPixels } from './geometry';
import './style.css';

declare global {
  interface Window {
    pdfPreview: { request(command: unknown): Promise<unknown> };
  }
}
function usePng(base64: string | undefined) {
  const [resource, setResource] = useState<{ source: string; url: string }>();
  useEffect(() => {
    if (!base64) {
      setResource(undefined);
      return;
    }
    const bytes = Uint8Array.from(atob(base64), (c) => c.charCodeAt(0));
    const next = URL.createObjectURL(new Blob([bytes], { type: 'image/png' }));
    setResource({ source: base64, url: next });
    return () => URL.revokeObjectURL(next);
  }, [base64]);
  return resource?.source === base64 ? resource?.url : undefined;
}
function App() {
  const [name, setName] = useState('report');
  const [activeName, setActiveName] = useState('');
  const [page, setPage] = useState<PageResult>();
  const [capture, setCapture] = useState<CaptureResult>();
  const [pageCount, setPageCount] = useState(0);
  const [index, setIndex] = useState(0),
    [scale, setScale] = useState(1),
    [rotation, setRotation] = useState(0),
    [dpr, setDpr] = useState(1);
  const [busy, setBusy] = useState(false),
    [message, setMessage] = useState('Open a fixture to inspect the real PDF renderer.');
  const [region, setRegion] = useState<Rect>();
  const epoch = useRef(0);
  const [painted, setPainted] = useState<string>();
  const canCapture = !busy && !!page && painted === page.renditionId;
  const pageUrl = usePng(page?.raster.png),
    captureUrl = usePng(capture?.raster.png);
  async function perform(work: (ticket: number) => Promise<void>) {
    const ticket = ++epoch.current;
    setBusy(true);
    setMessage('Rendering PDF…');
    try {
      await work(ticket);
      if (ticket === epoch.current)
        setMessage('Ready · Page and region selection qualified; text targeting is deferred.');
    } catch (error) {
      if (ticket === epoch.current) {
        setMessage(error instanceof Error ? error.message : String(error));
        setPage(undefined);
        setPageCount(0);
      }
    } finally {
      if (ticket === epoch.current) setBusy(false);
    }
  }
  async function renderPage(
    pageIndex: number,
    zoom: number,
    rotate: number,
    pixelScale: number,
    ticket: number,
  ) {
    const result = (await window.pdfPreview.request({
      kind: 'page',
      pageIndex,
      scale: zoom,
      rotation: rotate,
      dpr: pixelScale,
    })) as PageResult;
    if (ticket !== epoch.current) return;
    setPainted(undefined);
    setPage(result);
    setRegion(undefined);
    setCapture(undefined);
  }
  async function open() {
    await perform(async (ticket) => {
      setPage(undefined);
      setCapture(undefined);
      setRegion(undefined);
      const opened = (await window.pdfPreview.request({ kind: 'open', fixture: name })) as {
        pageCount: number;
      };
      if (ticket !== epoch.current) return;
      setActiveName(name);
      setPageCount(opened.pageCount);
      setIndex(0);
      await renderPage(0, scale, rotation, dpr, ticket);
    });
  }
  async function captureRegion(rect: Rect) {
    if (!page || !canCapture) return;
    setRegion(rect);
    await perform(async (ticket) => {
      const result = (await window.pdfPreview.request({
        kind: 'capture',
        renditionId: page.renditionId,
        modelHash: page.modelHash,
        region: rect,
      })) as CaptureResult;
      if (ticket === epoch.current) setCapture(result);
    });
  }
  const drag = useRef<{ x: number; y: number }>(undefined);
  return (
    <main>
      <header>
        <div>
          <span className="eyebrow">R3c1 · NATIVE PDF QUALIFICATION</span>
          <h1>One page. One source of evidence.</h1>
        </div>
        <span className="badge">Isolated fixtures · No Project data</span>
      </header>
      <section className="toolbar" aria-label="PDF controls">
        <label>
          Fixture
          <select
            value={name}
            disabled={busy}
            onChange={(e) => setName(e.target.value)}
            aria-label="PDF fixture"
          >
            {[
              'report',
              'rotated',
              'user-unit',
              'scanned',
              'unicode',
              'dense',
              'active',
              'many-pages',
              'too-many-pages',
              'noise',
              'oversized-image',
              'corrupt-image',
              'malformed',
              'encrypted',
              'limit-minus',
              'limit-exact',
              'limit-plus',
            ].map((item) => (
              <option key={item}>{item}</option>
            ))}
          </select>
        </label>
        <button onClick={() => void open()} disabled={busy}>
          Open PDF
        </button>
        <label>
          Page
          <input
            aria-label="Page number"
            type="number"
            min={1}
            max={pageCount || 1}
            value={index + 1}
            disabled={busy || !pageCount}
            onChange={(e) => {
              const next = Number(e.target.value) - 1;
              setIndex(next);
              void perform((ticket) => renderPage(next, scale, rotation, dpr, ticket));
            }}
          />
          <span>/ {pageCount || '—'}</span>
        </label>
        <label>
          Scale
          <select
            aria-label="Render scale"
            value={scale}
            disabled={busy || !page}
            onChange={(e) => {
              const next = Number(e.target.value);
              setScale(next);
              void perform((ticket) => renderPage(index, next, rotation, dpr, ticket));
            }}
          >
            {[0.75, 1, 1.25].map((n) => (
              <option key={n} value={n}>
                {n * 100}%
              </option>
            ))}
          </select>
        </label>
        <label>
          Rotate
          <select
            aria-label="Page rotation"
            value={rotation}
            disabled={busy || !page}
            onChange={(e) => {
              const next = Number(e.target.value);
              setRotation(next);
              void perform((ticket) => renderPage(index, scale, next, dpr, ticket));
            }}
          >
            {[0, 90, 180, 270].map((n) => (
              <option key={n} value={n}>
                {n}°
              </option>
            ))}
          </select>
        </label>
        <label>
          Pixels
          <select
            aria-label="Pixel scale"
            value={dpr}
            disabled={busy || !page}
            onChange={(e) => {
              const next = Number(e.target.value);
              setDpr(next);
              void perform((ticket) => renderPage(index, scale, rotation, next, ticket));
            }}
          >
            <option value={1}>1×</option>
            <option value={2}>2×</option>
          </select>
        </label>
        <button
          onClick={() => {
            epoch.current++;
            setBusy(false);
            void window.pdfPreview
              .request({ kind: 'cancel' })
              .catch(() =>
                setMessage('Unable to stop the decoder. Close the qualification window.'),
              );
            setPage(undefined);
            setPageCount(0);
            setCapture(undefined);
            setRegion(undefined);
            setMessage('Decoder stopped. Open a fixture to start again.');
          }}
        >
          Stop decoder
        </button>
      </section>
      <div className="status" role="status" data-busy={busy}>
        {message}
      </div>
      <div className="panels">
        <section className="reader" aria-label="Rendered PDF">
          <div className="pane-title">
            <strong>{page ? activeName + '.pdf' : 'No source'}</strong>
            <span>{page ? 'Page ' + (page.model.pageIndex + 1) : 'No page'}</span>
          </div>
          <div className="canvas-area">
            {page && pageUrl ? (
              <img
                src={pageUrl}
                alt="PDF page rendered by the isolated decoder"
                draggable={false}
                onLoad={() => setPainted(page.renditionId)}
                onPointerDown={(e) => {
                  if (!canCapture) return;
                  const box = e.currentTarget.getBoundingClientRect();
                  drag.current = {
                    x: ((e.clientX - box.left) / box.width) * page.raster.width,
                    y: ((e.clientY - box.top) / box.height) * page.raster.height,
                  };
                  e.currentTarget.setPointerCapture(e.pointerId);
                  e.preventDefault();
                }}
                onPointerCancel={() => {
                  drag.current = undefined;
                }}
                onPointerUp={(e) => {
                  if (!drag.current || !canCapture) return;
                  const box = e.currentTarget.getBoundingClientRect();
                  const x = Math.max(
                      0,
                      Math.min(
                        page.raster.width,
                        ((e.clientX - box.left) / box.width) * page.raster.width,
                      ),
                    ),
                    y = Math.max(
                      0,
                      Math.min(
                        page.raster.height,
                        ((e.clientY - box.top) / box.height) * page.raster.height,
                      ),
                    );
                  const rect: Rect = [
                    Math.min(drag.current.x, x),
                    Math.min(drag.current.y, y),
                    Math.max(drag.current.x, x),
                    Math.max(drag.current.y, y),
                  ];
                  drag.current = undefined;
                  e.currentTarget.releasePointerCapture(e.pointerId);
                  if (rect[2] - rect[0] > 2 && rect[3] - rect[1] > 2)
                    void captureRegion(regionFromPixels(rect, page.viewport));
                }}
              />
            ) : (
              <div className="empty">Choose a synthetic PDF and open it.</div>
            )}
          </div>
          <footer>
            <span>Drag on the page to capture a region.</span>
            <button disabled={!canCapture} onClick={() => void captureRegion([100, 120, 220, 180])}>
              Capture known region
            </button>
            <button
              disabled={!canCapture}
              onClick={() => page && void captureRegion(page.model.viewBox)}
            >
              Capture page
            </button>
          </footer>
        </section>
        <aside>
          <section className="evidence">
            <div className="pane-title">
              <strong>Captured evidence</strong>
              <span>Same rendition</span>
            </div>
            {captureUrl ? (
              <img src={captureUrl} alt="Captured PDF region" />
            ) : (
              <p>Select a region on the page. The crop comes from the same retained raster.</p>
            )}
            {capture && (
              <p className="capture-label">
                {capture.raster.width} × {capture.raster.height} px · Page {capture.pageIndex + 1}
              </p>
            )}
          </section>
          <section className="facts">
            <h2>Source and boundaries</h2>
            {page ? (
              <dl>
                <dt>Source revision</dt>
                <dd>{page.model.revision.slice(0, 27)}…</dd>
                <dt>Page box</dt>
                <dd>{page.model.viewBox.join(', ')}</dd>
                <dt>Page unit / intrinsic rotation</dt>
                <dd>
                  {page.model.userUnit} / {page.model.intrinsicRotation}°
                </dd>
                <dt>Raster</dt>
                <dd>
                  {page.raster.width} × {page.raster.height} px
                </dd>
                <dt>Text capability</dt>
                <dd>Diagnostic extraction only · {page.model.text.items.length} items</dd>
                <dt>Selected source region</dt>
                <dd>{region?.map((n) => n.toFixed(2)).join(', ') || 'None'}</dd>
              </dl>
            ) : (
              <p>No active decoder result.</p>
            )}
            <p className="note">
              This screen qualifies the PDF adapter. Project navigation, Chat delivery and Agent
              tools are later checkpoints.
            </p>
          </section>
        </aside>
      </div>
    </main>
  );
}
createRoot(document.getElementById('root')!).render(<App />);
