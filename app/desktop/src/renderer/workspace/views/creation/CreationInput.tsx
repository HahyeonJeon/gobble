import { useEffect, useState } from 'react';
import type { DesktopBridge } from '@gobble/contracts';
type Directory = Awaited<ReturnType<DesktopBridge['files']['list']>>;
/** Browses Project-owned resource IDs. Metadata only; no upload or file-content read. */
export function CreationInput({
  projectId,
  busy,
  onSelect,
}: {
  projectId: string;
  busy: boolean;
  onSelect: (resourceId: string) => void;
}) {
  const [path, setPath] = useState<string[]>([]),
    [result, setResult] = useState<Directory | null>(null),
    [selected, setSelected] = useState(''),
    [issue, setIssue] = useState('');
  const directory = path.at(-1);
  useEffect(() => {
    let active = true;
    void window.gobble.projects.list().then((r) => {
      if (!active) return;
      if (!r.ok) {
        setIssue(r.error.message);
        return;
      }
      const p = r.value.find((p) => p.projectId === projectId);
      if (p) setPath([p.rootResourceId]);
    });
    return () => {
      active = false;
    };
  }, [projectId]);
  useEffect(() => {
    if (!directory) return;
    let active = true;
    setSelected('');
    setResult(null);
    void window.gobble.files.list({ projectId, directoryId: directory }).then((r) => {
      if (active) setResult(r);
    });
    return () => {
      active = false;
    };
  }, [projectId, directory]);
  return (
    <section className="creation-input" aria-label="Choose pipeline data">
      <h3>Choose your data</h3>
      <p>One single-end FASTQ file. Then describe your analysis in Chat.</p>
      <div className="toolbar">
        <button disabled={path.length < 2 || busy} onClick={() => setPath((p) => p.slice(0, -1))}>
          ↑ Parent folder
        </button>
        <small>Project files · .fastq / .fq / .gz</small>
      </div>
      {issue && <p role="alert">{issue}</p>}
      {!result ? (
        <p>Loading files…</p>
      ) : !result.ok ? (
        <p role="alert">{result.error.message}</p>
      ) : (
        <ul className="creation-file-list">
          {result.value.entries
            .filter(
              (e) =>
                e.kind === 'directory' ||
                (e.kind === 'file' && /\.(fastq|fq)(\.gz)?$/i.test(e.name)),
            )
            .map((e) => (
              <li key={e.resourceId}>
                <button
                  disabled={busy}
                  aria-pressed={selected === e.resourceId}
                  onClick={() =>
                    e.kind === 'directory'
                      ? setPath((p) => [...p, e.resourceId])
                      : setSelected(e.resourceId)
                  }
                >
                  {e.kind === 'directory' ? '▸ ' : '◇ '}
                  {e.name}
                </button>
              </li>
            ))}
          {result.value.entries.length === 0 && <li>This folder is empty.</li>}
          {result.value.truncated && <li>Only the first 500 entries are shown.</li>}
        </ul>
      )}
      <button
        className="primary-button"
        disabled={!selected || busy}
        onClick={() => onSelect(selected)}
      >
        Use selected file
      </button>
      <small>Single-end reads · Trimming + quality report</small>
    </section>
  );
}
