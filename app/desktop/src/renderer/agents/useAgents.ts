import { useCallback, useEffect, useRef, useState } from 'react';
import type {
  AccountAction,
  AgentAction,
  CollaborationStatus,
  ConfigureAgent,
  SendMessage,
} from '@gobble/contracts';

export function useAgents(report: (message: string) => void) {
  const [status, setStatus] = useState<CollaborationStatus | null>(null);
  const current = useRef<CollaborationStatus | null>(null);
  const mounted = useRef(false);
  const receive = useCallback((next: CollaborationStatus) => {
    if (!mounted.current || (current.current && current.current.sequence > next.sequence)) return;
    current.current = next;
    setStatus(next);
  }, []);
  useEffect(() => {
    mounted.current = true;
    const unsubscribe = window.gobble.collaboration.onChanged(receive);
    window.gobble.collaboration
      .status()
      .then((result) => {
        if (result.ok) receive(result.value);
        else if (mounted.current) report(result.error.message);
      })
      .catch(() => {
        if (mounted.current) report('Agent status is unavailable.');
      });
    return () => {
      mounted.current = false;
      unsubscribe();
    };
  }, [receive, report]);
  const perform = useCallback(
    async (
      work: () => Promise<{ ok: true } | { ok: false; error: { message: string } }>,
    ): Promise<boolean> => {
      try {
        const result = await work();
        if (!result.ok && mounted.current) report(result.error.message);
        return result.ok;
      } catch {
        if (mounted.current)
          report('The agent connection is unavailable. Check message status before trying again.');
        return false;
      }
    },
    [report],
  );
  return {
    status,
    account: (input: AccountAction) =>
      perform(async () => {
        const result = await window.gobble.collaboration.account(input);
        if (result.ok) receive(result.value);
        return result;
      }),
    configure: (input: ConfigureAgent) =>
      perform(() => window.gobble.collaboration.configure(input)),
    send: (input: SendMessage) => perform(() => window.gobble.collaboration.send(input)),
    agent: (input: AgentAction) => perform(() => window.gobble.collaboration.agent(input)),
  };
}
export type AgentsModel = ReturnType<typeof useAgents>;
