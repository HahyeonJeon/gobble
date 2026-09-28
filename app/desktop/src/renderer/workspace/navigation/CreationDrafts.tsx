import { useEffect, useRef, useState } from 'react';
import type { CreationDraft } from '@gobble/contracts';
import { requestId } from '../useWorkspace';
import type { OpenResource } from './types';
export function CreationDrafts({
  projectId,
  revision,
  onOpen,
  onChanged,
}: {
  projectId: string;
  revision: number;
  onOpen: OpenResource;
  onChanged: () => void;
}) {
  const [drafts, setDrafts] = useState<CreationDraft[]>([]),
    [busy, setBusy] = useState(false),
    [issue, setIssue] = useState(''),
    [refresh, setRefresh] = useState(0);
  const active = useRef(true);
  const signature = useRef<string | null>(null);
  useEffect(() => {
    active.current = true;
    return () => {
      active.current = false;
    };
  }, []);
  useEffect(() => {
    let live = true;
    let timer: ReturnType<typeof setTimeout>;
    async function refreshDrafts() {
      const result = await window.gobble.creation.list({ projectId });
      if (!live) return;
      if (result.ok) {
        const next = JSON.stringify(result.value.drafts);
        if (signature.current !== null && signature.current !== next) onChanged();
        signature.current = next;
        setDrafts(result.value.drafts);
        setIssue('');
      } else setIssue(result.error.message);
      timer = setTimeout(() => void refreshDrafts(), 3000);
    }
    void refreshDrafts();
    return () => {
      live = false;
      clearTimeout(timer);
    };
  }, [projectId, revision, refresh, onChanged]);
  async function create() {
    if (busy) return;
    setBusy(true);
    setIssue('');
    try {
      const r = await window.gobble.creation.create({
        projectId,
        requestId: requestId(),
        brief: '',
      });
      if (!active.current) return;
      if (!r.ok) setIssue(r.error.message);
      else {
        setRefresh((n) => n + 1);
        onOpen({ kind: 'creation-draft', draftId: r.value.draftId }, 'active');
      }
    } finally {
      if (active.current) setBusy(false);
    }
  }
  return (
    <>
      <button
        aria-label="New pipeline"
        className="text-button pipeline-import"
        disabled={busy}
        onClick={() => void create()}
      >
        ＋ {busy ? 'Creating…' : 'New pipeline'}
      </button>
      {drafts.length > 0 && (
        <>
          <small className="sidebar-note muted">Drafts</small>
          <ul className="file-list">
            {drafts.map((d, i) => (
              <li key={d.draftId}>
                <button
                  title={d.input?.relativePath || 'Select data to begin'}
                  onClick={() => onOpen({ kind: 'creation-draft', draftId: d.draftId }, 'active')}
                >
                  <span className="truncate">
                    ◇ {d.input?.relativePath.split('/').at(-1) || `New pipeline ${i + 1}`}
                  </span>
                </button>
              </li>
            ))}
          </ul>
        </>
      )}
      {issue && (
        <p className="sidebar-note inline-error" role="alert">
          {issue}
        </p>
      )}
    </>
  );
}
