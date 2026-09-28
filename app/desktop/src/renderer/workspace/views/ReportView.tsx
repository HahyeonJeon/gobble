import { markStyle } from '../../shared-context/marks';
import { useEffect, useId, useRef, useState } from 'react';
import {
  referenceAuthor,
  type SharedReference,
  type SavedReport,
  type LogTarget,
} from '@gobble/contracts';
import '../../styles/report.css';

export function ReportView({
  report,
  moduleId,
  ready,
  onNavigate,
  onReady,
  onFailure,
  onOpenLogs,
  onAttach,
  onShare,
  marks = [],
}: {
  moduleId: string | undefined;
  ready: boolean;
  onNavigate: (moduleId: string) => Promise<boolean>;
  report: SavedReport;
  onReady: () => void;
  onFailure: (message: string) => void;
  onOpenLogs?: (target: LogTarget) => void;
  onAttach?: () => void;
  onShare?: () => void;
  marks?: SharedReference[];
}) {
  const prefix = useId();
  const root = useRef<HTMLDivElement>(null);
  const [images, setImages] = useState<{ report: SavedReport; urls: Map<string, string> } | null>(
    null,
  );
  const section = moduleId ?? report.content.modules[0]!.id;
  useEffect(() => {
    let active = true;
    const urls = new Map<string, string>();
    const loading = report.content.modules
      .flatMap((m) => m.blocks)
      .filter((b) => b.kind === 'image')
      .map(async (chart) => {
        const bytes = Uint8Array.from(atob(chart.base64), (c) => c.charCodeAt(0));
        const url = URL.createObjectURL(new Blob([bytes], { type: 'image/png' }));
        urls.set(chart.id, url);
        const image = new Image();
        image.src = url;
        await image.decode();
        if (image.naturalWidth !== chart.width || image.naturalHeight !== chart.height)
          throw new Error('Invalid report image.');
      });
    void Promise.all(loading).then(
      () => {
        if (active) setImages({ report, urls });
      },
      () => {
        if (active)
          onFailure(
            'An original report chart could not be displayed. The saved report has been preserved.',
          );
      },
    );
    return () => {
      active = false;
      for (const url of urls.values()) URL.revokeObjectURL(url);
    };
  }, [report, onFailure]);
  useEffect(() => {
    if (images?.report === report) onReady();
  }, [images, report, onReady]);
  const content = report.content;
  useEffect(() => {
    if (!moduleId || images?.report !== report) return;
    const element = root.current?.querySelector<HTMLElement>('[data-module="' + moduleId + '"]');
    element?.scrollIntoView({ block: 'start' });
    element?.focus({ preventScroll: true });
  }, [images, report, moduleId]);
  return (
    <div className="report-view" ref={root}>
      <header className="report-toolbar">
        <span className="report-saved">Saved report</span>
        {onAttach && (
          <button disabled={!ready} onClick={onAttach}>
            Attach report
          </button>
        )}
        {onShare && (
          <button disabled={!ready} onClick={onShare}>
            Point to report
          </button>
        )}
        <label>
          Section{' '}
          <select
            aria-label="Report section"
            disabled={!ready}
            value={section}
            onChange={(e) => void onNavigate(e.target.value)}
          >
            {content.modules.map((m) => (
              <option key={m.id} value={m.id}>
                {m.title}
              </option>
            ))}
          </select>
        </label>
        {onOpenLogs && (
          <button
            onClick={() =>
              onOpenLogs({
                runRef: report.runRef,
                instanceId: report.producer.instance,
                attempt: report.producer.attempt,
              })
            }
          >
            Producer logs · attempt {report.producer.attempt}
          </button>
        )}
      </header>
      {marks.length > 0 && (
        <aside className="report-marks" aria-label="Report pointers">
          {marks.map((mark) => (
            <div
              key={mark.referenceId}
              data-reference-id={mark.referenceId}
              style={markStyle(mark)}
            >
              <strong>{referenceAuthor(mark)} · Whole report</strong>
              {mark.note && <span> — {mark.note}</span>}
            </div>
          ))}
        </aside>
      )}
      <div className="report-reading">
        <header className="report-heading">
          <span className="report-wordmark">{content.logoLabel}</span>
          <h2>{content.headerTitle}</h2>
          <p>{content.headerFilename}</p>
        </header>
        <details className="report-summary">
          <summary>
            {content.summaryHeading} · {content.modules.length} sections
          </summary>
          <nav aria-label="Report contents">
            {content.summary.map((s) => (
              <button
                key={s.moduleId}
                disabled={!ready}
                onClick={() => void onNavigate(s.moduleId)}
              >
                <Status label={s.status} />
                <span>{s.title}</span>
              </button>
            ))}
          </nav>
        </details>
        {content.modules.map((module) => (
          <section
            className="report-module"
            key={module.id}
            data-module={module.id}
            tabIndex={-1}
            aria-labelledby={prefix + module.id}
          >
            <h3 id={prefix + module.id}>
              <Status label={module.status} />
              {module.title}
            </h3>
            {module.blocks.map((block, i) =>
              block.kind === 'text' ? (
                <p className="report-text" key={i}>
                  {block.text}
                </p>
              ) : block.kind === 'table' ? (
                <div className="report-table" key={i}>
                  <table>
                    <thead>
                      <tr>
                        {block.headers.map((v, c) => (
                          <th key={c} scope="col">
                            {v}
                          </th>
                        ))}
                      </tr>
                    </thead>
                    <tbody>
                      {block.rows.map((row, r) => (
                        <tr key={r}>
                          {row.map((v, c) => (
                            <td key={c}>{v}</td>
                          ))}
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              ) : (
                <figure key={block.id}>
                  {images?.report === report ? (
                    <img
                      src={images.urls.get(block.id)}
                      width={block.width}
                      height={block.height}
                      alt={block.alt}
                    />
                  ) : (
                    <p role="status">Loading original chart…</p>
                  )}
                  <figcaption>{block.alt}</figcaption>
                </figure>
              ),
            )}
          </section>
        ))}
        <footer>{content.footer}</footer>
        <details className="report-identity">
          <summary>Saved report details</summary>
          <dl>
            <dt>Captured</dt>
            <dd>{new Date(report.capturedAt).toLocaleString('en-US')}</dd>
            <dt>Producer</dt>
            <dd>
              {report.producer.instance} · attempt {report.producer.attempt}
            </dd>
            <dt>Source SHA-256</dt>
            <dd>{report.producer.sha256}</dd>
          </dl>
        </details>
      </div>
    </div>
  );
}
function Status({ label }: { label: string }) {
  const tone =
    label === '[FAIL]' ? 'fail' : ['[WARN]', '[WARNING]'].includes(label) ? 'warn' : 'pass';
  return <span className={'report-status ' + tone}>{label}</span>;
}
