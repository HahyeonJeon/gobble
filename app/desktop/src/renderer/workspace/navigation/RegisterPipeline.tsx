import { useEffect, useRef, useState } from 'react';
import { Icon } from '../Icon';
import { requestId } from '../useWorkspace';

/** One request identity per displayed folder, including retries of uncertain delivery. */
export function RegisterPipeline({
  projectId,
  packageResourceId,
  name,
  onRegistered,
}: {
  projectId: string;
  packageResourceId: string;
  name: string;
  onRegistered: () => void;
}) {
  const [id] = useState(requestId);
  const [state, setState] = useState<'ready' | 'pending' | 'registered'>('ready');
  const [problem, setProblem] = useState<string | null>(null);
  const active = useRef(true);
  const submitting = useRef(false);
  useEffect(() => {
    active.current = true;
    return () => {
      active.current = false;
    };
  }, []);

  async function register() {
    if (submitting.current || state === 'registered') return;
    submitting.current = true;
    setState('pending');
    setProblem(null);
    try {
      const result = await window.gobble.pipelines.register({
        projectId,
        packageResourceId,
        name,
        requestId: id,
      });
      // The folder may have closed while its accepted registration completed.
      // Its Project list still needs to refresh; the callback is Project-owned.
      if (result.ok) onRegistered();
      if (!active.current) return;
      if (result.ok) setState('registered');
      else {
        setState('ready');
        setProblem(result.error.message);
      }
    } catch {
      if (active.current) {
        setState('ready');
        setProblem('Registration could not be confirmed. Retry to check the same request.');
      }
    } finally {
      submitting.current = false;
    }
  }

  return (
    <div className="pipeline-registration">
      <button className="text-button" disabled={state !== 'ready'} onClick={() => void register()}>
        <Icon name="plus" />
        {state === 'pending'
          ? 'Registering…'
          : state === 'registered'
            ? 'Pipeline registered'
            : 'Register pipeline'}
      </button>
      {problem && (
        <p role="alert" className="inline-error">
          {problem}
        </p>
      )}
    </div>
  );
}
