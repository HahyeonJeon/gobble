import { useEffect, useRef, useState } from 'react';
import type { PipelinePreparation, PipelineReviewMark } from '@gobble/contracts';
import { requestId } from '../../useWorkspace';
export type PreparationSection = 'data' | 'settings' | 'environment';
/** Owns local pending-operation identity and lifecycle; the native service owns durable work. */
export function useRunPreparation(projectId: string, pipelineId: string, artifactId: string) {
  const [values, setValues] = useState<PipelinePreparation[]>([]),
    [selected, setSelected] = useState<string | null>(null),
    [issue, setIssue] = useState(''),
    [busy, setBusy] = useState(false),
    [marks, setMarks] = useState<PipelineReviewMark[]>([]),
    [engines, setEngines] = useState<Array<{ engineId: string; label: string }>>([]),
    [engineId, setEngineId] = useState('');
  const operation = useRef<string | null>(null),
    alive = useRef(true);
  const value = values.find((v) => v.requestId === selected) ?? values.at(-1);
  useEffect(() => {
    let stopped = false;
    alive.current = true;
    let timer: ReturnType<typeof setTimeout>;
    async function poll() {
      try {
        const [result, reviews] = await Promise.all([
          window.gobble.preparations.list({ projectId, pipelineId }),
          window.gobble.pipelineReviews.list({ projectId, pipelineId }),
        ]);
        if (stopped) return;
        if (result.ok) {
          setValues([...result.value].sort((a, b) => a.createdAt.localeCompare(b.createdAt)));
        } else setIssue(result.error.message);
        if (reviews.ok) setMarks(reviews.value.marks.filter((m) => !!m.context.preparationId));
      } catch {
        if (!stopped) setIssue('Run reviews are unavailable. Reopen this view to reconnect.');
      }
      if (!stopped) timer = setTimeout(() => void poll(), 2000);
    }
    void poll();
    return () => {
      stopped = true;
      alive.current = false;
      clearTimeout(timer);
    };
  }, [projectId, pipelineId]);
  useEffect(() => {
    let stopped = false;
    void window.gobble.preparations.engines().then((result) => {
      if (stopped) return;
      if (result.ok) {
        setEngines(result.value);
        setEngineId((current) => current || result.value[0]?.engineId || '');
      } else setIssue(result.error.message);
    });
    return () => {
      stopped = true;
    };
  }, [projectId, pipelineId]);
  async function prepare() {
    if (busy || !engineId) return;
    setBusy(true);
    setIssue('');
    operation.current ??= requestId();
    try {
      const result = await window.gobble.preparations.prepare({
        projectId,
        pipelineId,
        requestId: operation.current,
        artifactId: artifactId,
        engineId,
      });
      if (!alive.current) return;
      if (result.ok) {
        setSelected(result.value.requestId);
        setValues((old) => [
          ...old.filter((v) => v.requestId !== result.value.requestId),
          result.value,
        ]);
        operation.current = null;
      } else {
        if (!['internal', 'runtime_unavailable'].includes(result.error.code))
          operation.current = null;
        setIssue(result.error.message);
      }
    } catch {
      if (alive.current)
        setIssue('The outcome is uncertain. Retry to reconnect to the same preparation.');
    } finally {
      if (alive.current) setBusy(false);
    }
  }
  async function cancel() {
    if (!value || busy) return;
    setBusy(true);
    const result = await window.gobble.preparations.cancel({
      projectId,
      pipelineId,
      requestId: value.requestId,
    });
    if (alive.current) {
      setBusy(false);
      setIssue(result.ok ? 'Cancellation requested. No analysis started.' : result.error.message);
    }
  }
  async function discuss(section: PreparationSection) {
    if (!value || value.state !== 'ready') return;
    const result = await window.gobble.pipelineReviews.select({
      projectId,
      context: {
        pipelineId,
        baseArtifactId: value.artifactId,
        preparationId: value.requestId,
        preparationSection: section,
      },
    });
    if (alive.current)
      setIssue(
        result.ok
          ? `${section.charAt(0).toUpperCase() + section.slice(1)} added to Chat. Your selected flow step is unchanged.`
          : result.error.message,
      );
  }
  const checking = values.some((v) => v.state === 'preparing');
  const exact = value?.state === 'ready' && value.fresh && value.artifactId === artifactId;

  async function refreshEngines() {
    const result = await window.gobble.preparations.engines();
    if (!alive.current) return;
    if (result.ok) {
      setEngines(result.value);
      setEngineId((current) =>
        result.value.some((e) => e.engineId === current)
          ? current
          : result.value[0]?.engineId || '',
      );
      setIssue(
        result.value.length
          ? ''
          : 'No preparation engine is available. Start the local engine and refresh.',
      );
    } else setIssue(result.error.message);
  }
  function chooseEngine(id: string) {
    setEngineId(id);
    operation.current = null;
  }
  return {
    value,
    values,
    marks,
    issue,
    busy,
    checking,
    exact,
    engines,
    engineId,
    setSelected,
    chooseEngine,
    refreshEngines,
    prepare,
    cancel,
    discuss,
  };
}
