import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { createRoot } from 'react-dom/client';
import type { Capture, PartAddress, Target, ViewSnapshot } from './model';
import { visibleParts } from './visibility';
import { ImageContent, MarkdownPreview, TextContent } from './reader-parts';
import './bridge';
import './style.css';
function App() {
  const [name, setName] = useState('python'),
    [loadedName, setLoadedName] = useState(''),
    [snapshot, setSnapshot] = useState<ViewSnapshot | null>(null),
    [status, setStatus] = useState('Choose Open notebook.'),
    [loading, setLoading] = useState(false);
  const [selection, setSelection] = useState<Target | null>(null),
    [capture, setCapture] = useState<Capture | null>(null),
    [reference, setReference] = useState(false),
    [draft, setDraft] = useState('Should we revisit this threshold?');
  const [activeImage, setActiveImage] = useState('');
  const [folded, setFolded] = useState<string[]>([]),
    [sourceMode, setSourceMode] = useState<string[]>([]),
    [page, setPage] = useState(0);
  const reader = useRef<HTMLDivElement>(null),
    composer = useRef<HTMLTextAreaElement>(null),
    origin = useRef({ scroll: 0, focus: null as HTMLElement | null }),
    frame = useRef(0),
    generation = useRef(Date.now());
  const addresses = useMemo(() => {
    const m = new Map<string, PartAddress>();
    for (const c of snapshot?.cells ?? []) {
      m.set('source-' + c.index, { cell: c.address, part: { kind: 'source' } });
      for (const o of c.outputs)
        if (o.part.kind !== 'unavailable')
          m.set(c.index + '-' + o.index, {
            cell: c.address,
            part: { kind: 'output', index: o.index, mime: o.mime, digest: o.part.digest },
          });
    }
    return m;
  }, [snapshot]);
  const changed = useCallback(() => {
    window.notebook.invalidate();
    cancelAnimationFrame(frame.current);
    frame.current = requestAnimationFrame(() => {
      if (!reader.current || !snapshot || reference || loading) return;
      const parts = visibleParts(reader.current, addresses);
      window.notebook
        .visible({ revision: snapshot.revision, generation: ++generation.current, parts })
        .then((result) => {
          if (!result.ok) setStatus('Observation unavailable: ' + result.message);
        })
        .catch((e) => setStatus(String(e)));
    });
  }, [snapshot, reference, loading, addresses]);
  useEffect(() => {
    changed();
    const ro = new ResizeObserver(changed);
    const el = reader.current;
    if (el) {
      ro.observe(el);
      el.addEventListener('scroll', changed, true);
    }
    return () => {
      cancelAnimationFrame(frame.current);
      ro.disconnect();
      el?.removeEventListener('scroll', changed, true);
    };
  }, [changed, folded, sourceMode, page]);
  async function open() {
    setLoading(true);
    setStatus('Reading notebook…');
    const result = await window.notebook.open(name);
    setLoading(false);
    if (result.ok) {
      setActiveImage('');
      setSnapshot(result.value);
      setLoadedName(name);
      setSelection(null);
      setReference(false);
      setFolded([]);
      setSourceMode([]);
      setPage(0);
      setStatus('Ready · Saved outputs · Execution not verified');
    } else setStatus(result.message);
  }
  async function attach() {
    if (!selection) return;
    const result = await window.notebook.capture(selection);
    if (result.ok) {
      setCapture(result.value);
      setStatus('Captured exact saved evidence.');
      composer.current?.focus();
    } else setStatus(result.message);
  }
  function show() {
    if (!capture) return;
    origin.current = {
      scroll: reader.current?.scrollTop ?? 0,
      focus: document.activeElement instanceof HTMLElement ? document.activeElement : null,
    };
    setReference(true);
  }
  function back() {
    setReference(false);
    requestAnimationFrame(() => {
      if (reader.current) reader.current.scrollTop = origin.current.scroll;
      origin.current.focus?.focus({ preventScroll: true });
    });
  }
  const pick = (target: Target) => {
    if (reference) return;
    setSelection(target);
    setStatus(
      target.selector.kind === 'text'
        ? `Selected UTF-16 ${target.selector.start}–${target.selector.end}`
        : `Selected image region ${target.selector.rect.width} × ${target.selector.rect.height}`,
    );
  };
  const toggle = (key: string, values: string[], setter: (next: string[]) => void) =>
    setter(values.includes(key) ? values.filter((v) => v !== key) : [...values, key]);
  return (
    <>
      <header className="top">
        <strong>Gobble · Notebook qualification</strong>
        <span>R4a1 · Synthetic files · No Agent or kernel</span>
      </header>
      <div className="controls">
        <label>
          Notebook fixture
          <select
            aria-label="Notebook fixture"
            value={name}
            disabled={loading}
            onChange={(e) => setName(e.target.value)}
          >
            {[
              'python',
              'r',
              'legacy',
              'reordered',
              'changed',
              'unsafe',
              'invalid',
              'many',
              'large',
              'jpeg',
              'long-text',
            ].map((n) => (
              <option key={n}>{n}</option>
            ))}
          </select>
        </label>
        <button
          disabled={loading}
          onClick={() =>
            open().catch((e) => {
              setLoading(false);
              setStatus(String(e));
            })
          }
        >
          Open notebook
        </button>
        <span role="status">{status}</span>
      </div>
      <main>
        <section className="pane">
          <header>
            <strong>{loadedName ? loadedName + '.ipynb' : 'Notebook reader'}</strong>
            <span>
              {snapshot
                ? `nbformat 4.${snapshot.minor} · ${snapshot.language} · ${snapshot.cells.length} cells`
                : ''}
            </span>
          </header>
          {reference && capture && (
            <div className="reference">
              <strong>
                {snapshot?.revision === capture.target.revision
                  ? 'Captured reference'
                  : 'Earlier version · Captured reference'}
              </strong>
              <button onClick={back}>Return to my view</button>
            </div>
          )}
          <div
            ref={reader}
            className="reader"
            hidden={reference}
            onScroll={changed}
            aria-label="Notebook cells"
          >
            {snapshot?.cells.slice(page * 20, page * 20 + 20).map((c) => {
              const key = 'source-' + c.index,
                address = addresses.get(key);
              if (!address) return null;
              return (
                <article className="cell" key={snapshot.revision + key}>
                  <div className="cell-header">
                    <strong>
                      Cell {c.index + 1} · {c.type}
                    </strong>
                    <span>
                      {c.address.kind === 'id' ? c.address.id : 'Revision-scoped ordinal'}
                    </span>
                    {c.type === 'markdown' && (
                      <button onClick={() => toggle(key, sourceMode, setSourceMode)}>
                        {sourceMode.includes(key) ? 'Preview' : 'Source'}
                      </button>
                    )}
                  </div>
                  {c.notice && <small className="notice">{c.notice}</small>}
                  {c.type === 'markdown' && !sourceMode.includes(key) ? (
                    <MarkdownPreview text={c.source.text} />
                  ) : (
                    <TextContent
                      text={c.source.text}
                      address={address}
                      revision={snapshot.revision}
                      partKey={key}
                      select={pick}
                    />
                  )}
                  {c.outputs.length > 0 && (
                    <div className="output-header">
                      <strong>Saved outputs</strong>
                      <button onClick={() => toggle(key, folded, setFolded)}>
                        {folded.includes(key) ? 'Expand outputs' : 'Collapse outputs'}
                      </button>
                    </div>
                  )}
                  {!folded.includes(key) &&
                    c.outputs.map((o) => {
                      const partKey = c.index + '-' + o.index,
                        address = addresses.get(partKey);
                      return (
                        <section className="output" key={o.index}>
                          <small>
                            Output {o.index + 1} · {o.mime || o.type}
                          </small>
                          {o.notice && <p className="notice">{o.notice}</p>}
                          {o.part.kind === 'unavailable' ? (
                            <p className="unavailable">Preview unavailable · {o.part.reason}</p>
                          ) : !address ? null : o.part.kind === 'text' ? (
                            <TextContent
                              text={o.part.text}
                              address={address}
                              revision={snapshot.revision}
                              partKey={partKey}
                              select={pick}
                            />
                          ) : (
                            <ImageContent
                              active={activeImage === partKey}
                              activate={() => setActiveImage(partKey)}
                              address={address}
                              revision={snapshot.revision}
                              partKey={partKey}
                              width={o.part.width}
                              height={o.part.height}
                              select={pick}
                              onChange={changed}
                            />
                          )}
                        </section>
                      );
                    })}
                </article>
              );
            })}
            {snapshot && snapshot.cells.length > 20 && (
              <div className="paging">
                <button disabled={page === 0} onClick={() => setPage((p) => p - 1)}>
                  Previous cells
                </button>
                <span>
                  Cells {page * 20 + 1}–{Math.min((page + 1) * 20, snapshot.cells.length)}
                </span>
                <button
                  disabled={(page + 1) * 20 >= snapshot.cells.length}
                  onClick={() => setPage((p) => p + 1)}
                >
                  Next cells
                </button>
              </div>
            )}
          </div>
          {reference && capture && (
            <div className="captured-content">
              <h2>{capture.label}</h2>
              {capture.representation === 'text/plain' ? (
                <pre>{capture.text}</pre>
              ) : (
                <img
                  src={'data:image/png;base64,' + capture.base64}
                  alt="Captured Notebook region"
                />
              )}
              <p>
                Immutable captured evidence. The original reading position and selection are
                retained.
              </p>
            </div>
          )}
          <footer>
            <span>
              {selection
                ? selection.part.kind === 'source'
                  ? 'Cell source selected'
                  : 'Saved output selected'
                : 'Select exact source, output text or an image region'}
            </span>
            <button
              disabled={!selection || reference || loading}
              onClick={() => attach().catch((e) => setStatus(String(e)))}
            >
              Discuss selection
            </button>
          </footer>
        </section>
        <aside className="chat">
          <header>
            <strong>Discussion draft</strong>
            <span>Qualification only</span>
          </header>
          <div className="conversation">
            <p>Select a part of the Notebook, then attach its exact saved content.</p>
            <p>Changing the source never updates a captured attachment.</p>
            {capture && (
              <div className="evidence">
                <strong>{capture.label}</strong>
                <small>{capture.target.revision.slice(0, 23)}…</small>
                {capture.representation === 'text/plain' ? (
                  <pre>{capture.text}</pre>
                ) : (
                  <img src={'data:image/png;base64,' + capture.base64} alt="Captured attachment" />
                )}
                <button onClick={show}>Show captured reference</button>
              </div>
            )}
          </div>
          <div className="composer">
            <label htmlFor="draft">Your message</label>
            <textarea
              ref={composer}
              id="draft"
              value={draft}
              onChange={(e) => setDraft(e.target.value)}
            />
            <small>Draft stays local · Sending is not part of qualification</small>
          </div>
        </aside>
      </main>
    </>
  );
}
const root = document.getElementById('root');
if (!root) throw new Error('Missing qualification root.');
createRoot(root).render(<App />);
