import { useCallback, useEffect, useRef, useState } from 'react';
import type { SurfaceLoad } from '@gobble/contracts';

type State =
  | { kind: 'loading' }
  | { kind: 'error'; message: string }
  | { kind: 'loaded'; load: SurfaceLoad; identity: string };
/** Owns asynchronous observation and acknowledgment, independent of any presenter. */
export function useSurfaceLoad({
  projectId,
  surfaceId,
  rendererSessionId,
  presentationKey,
  linkedRevision,
  refreshVersion,
}: {
  projectId: string;
  surfaceId: string;
  rendererSessionId: string;
  presentationKey: string;
  linkedRevision: string | undefined;
  refreshVersion: number;
}) {
  const [state, setState] = useState<State>({ kind: 'loading' });
  const [retry, setRetry] = useState(0);
  const [acknowledged, setReady] = useState(false);
  const acknowledgmentIsCurrent = useRef<SurfaceLoad | null>(null);
  const previousRefresh = useRef(`${refreshVersion}:0`);
  const ready = acknowledged && state.kind === 'loaded' && state.identity === presentationKey;
  useEffect(() => {
    let active = true;
    const refreshKey = `${refreshVersion}:${retry}`;
    const refresh = previousRefresh.current !== refreshKey;
    previousRefresh.current = refreshKey;
    setState((current) =>
      current.kind === 'loaded' &&
      (!linkedRevision || linkedRevision === current.load.acknowledgment.dataRevision)
        ? current
        : { kind: 'loading' },
    );
    setReady(false);
    acknowledgmentIsCurrent.current = null;
    void window.gobble.workspace
      .loadSurface({
        projectId,
        surfaceId,
        rendererSessionId,
        ...(refresh ? { refresh: true } : {}),
      })
      .then((result) => {
        if (!active) return;
        acknowledgmentIsCurrent.current = result.ok ? result.value : null;
        setState(
          result.ok
            ? { kind: 'loaded', load: result.value, identity: presentationKey }
            : { kind: 'error', message: result.error.message },
        );
      })
      .catch(() => {
        if (active)
          setState({ kind: 'error', message: 'This view could not be loaded. Try again.' });
      });
    return () => {
      active = false;
      acknowledgmentIsCurrent.current = null;
    };
  }, [projectId, surfaceId, rendererSessionId, retry, refreshVersion, presentationKey]);
  const loaded = state.kind === 'loaded' ? state.load : null;
  const acknowledge = useCallback(() => {
    if (
      !loaded ||
      acknowledgmentIsCurrent.current !== loaded ||
      state.kind !== 'loaded' ||
      state.identity !== presentationKey
    )
      return;
    void window.gobble.workspace
      .acknowledge(loaded.acknowledgment)
      .then((result) => {
        if (acknowledgmentIsCurrent.current !== loaded) return;
        setReady(result.ok);
        if (!result.ok)
          setState((current) =>
            current.kind === 'loaded' && current.load === loaded
              ? { kind: 'error', message: result.error.message }
              : current,
          );
      })
      .catch(() => {
        if (acknowledgmentIsCurrent.current !== loaded) return;
        setState((current) =>
          current.kind === 'loaded' && current.load === loaded
            ? { kind: 'error', message: 'The displayed view could not be confirmed. Try again.' }
            : current,
        );
      });
  }, [loaded, presentationKey]);
  const failRender = useCallback(
    (message: string) => {
      if (!loaded || acknowledgmentIsCurrent.current !== loaded) return;
      acknowledgmentIsCurrent.current = null;
      setReady(false);
      const failure: State = { kind: 'error', message };
      setState(failure);
      void window.gobble.workspace.invalidate(loaded.acknowledgment).catch(() => {
        setState((current) =>
          current === failure
            ? {
                kind: 'error',
                message: message + ' Reload the workspace before using this view again.',
              }
            : current,
        );
      });
    },
    [loaded],
  );
  async function changePresentation(change: () => Promise<boolean>) {
    setReady(false);
    const accepted = await change();
    if (!accepted && acknowledgmentIsCurrent.current === loaded) setReady(true);
    return accepted;
  }
  return {
    state,
    loaded,
    ready,
    acknowledge,
    failRender,
    changePresentation,
    refresh: () => {
      acknowledgmentIsCurrent.current = null;
      setReady(false);
      setRetry((value) => value + 1);
    },
  };
}
