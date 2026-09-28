import { useEffect, useState } from 'react';
import type { DesktopBridge } from '@gobble/contracts';
type Result = Awaited<ReturnType<DesktopBridge['creation']['engines']>>;
/** Host-selected installed capabilities. Never accepts a runtime path or command. */
export function CreationEngine() {
  const [result, setResult] = useState<Result | null>(null),
    [busy, setBusy] = useState(false),
    [selected, setSelected] = useState('');
  useEffect(() => {
    let active = true;
    void window.gobble.creation.engines().then((v) => {
      if (active) setResult(v);
    });
    return () => {
      active = false;
    };
  }, []);
  async function load(connect = false) {
    setBusy(true);
    try {
      setResult(
        await (connect
          ? window.gobble.creation.connect({
              engineId: selected || (result?.ok ? result.value.engines[0]?.engineId : '') || '',
            })
          : window.gobble.creation.engines()),
      );
    } finally {
      setBusy(false);
    }
  }
  if (result?.ok && result.value.ready)
    return <small className="creation-engine-ready">● Analysis engine connected</small>;
  return (
    <div className="creation-engine" role="status">
      <span>
        {!result
          ? 'Looking for an analysis engine…'
          : result.ok
            ? result.value.issue
            : result.error.message}
      </span>
      {result?.ok && result.value.engines.length > 0 && (
        <>
          <select
            aria-label="Analysis engine"
            disabled={busy}
            value={selected || result.value.engines[0]?.engineId}
            onChange={(e) => setSelected(e.target.value)}
          >
            {result.value.engines.map((e) => (
              <option key={e.engineId} value={e.engineId}>
                {e.label}
              </option>
            ))}
          </select>
          <button disabled={busy} onClick={() => void load(true)}>
            {busy ? 'Connecting…' : 'Connect engine'}
          </button>
        </>
      )}
      <button disabled={busy} onClick={() => void load()}>
        Refresh engine
      </button>
    </div>
  );
}
