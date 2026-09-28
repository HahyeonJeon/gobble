import { useCallback, useEffect, useRef, useState } from 'react';
import type {
  ResourceRef,
  WorkspaceAction,
  WorkspaceBootstrap,
  WorkspaceDocument,
} from '@gobble/contracts';

export type Command = (action: WorkspaceAction) => Promise<boolean>;
export const requestId = () => 'req_' + crypto.randomUUID();

export function useWorkspace() {
  const [state, setState] = useState<WorkspaceBootstrap | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [saving, setSaving] = useState(0);
  const current = useRef<WorkspaceBootstrap | null>(null);
  const queue = useRef<Promise<unknown>>(Promise.resolve());
  const drafts = useRef<Promise<unknown>>(Promise.resolve());
  const mounted = useRef(false);
  const unsavedDrafts = useRef(new Set<string>());

  const receive = useCallback((value: WorkspaceBootstrap) => {
    current.current = value;
    if (mounted.current) setState(value);
  }, []);
  const receiveDocument = useCallback(
    (document: WorkspaceDocument) => {
      const latest = current.current;
      if (
        !latest ||
        latest.document?.workspace.projectId !== document.workspace.projectId ||
        latest.document.workspace.revision > document.workspace.revision
      )
        return;
      receive({ ...latest, document });
    },
    [receive],
  );
  const report = useCallback((message: string) => {
    if (mounted.current) setError(message);
  }, []);
  useEffect(() => window.gobble.workspace.onChanged(receiveDocument), [receiveDocument]);
  const canLeaveProject = useCallback(async () => {
    await drafts.current;
    const projectId = current.current?.document?.workspace.projectId;
    if (projectId && unsavedDrafts.current.has(projectId)) {
      report(
        'Your draft is not saved. Keep this Project open and copy the draft before closing the app.',
      );
      return false;
    }
    return true;
  }, [report]);
  const connect = useCallback(async () => {
    setBusy(true);
    try {
      if (!(await canLeaveProject())) return;
      const result = await window.gobble.workspace.connect();
      if (!result.ok) {
        report(result.error.message);
        return;
      }
      receive(result.value);
      setError(null);
    } catch {
      report('The workspace could not be opened. Try again.');
    } finally {
      if (mounted.current) setBusy(false);
    }
  }, [canLeaveProject, receive, report]);
  useEffect(() => {
    let active = true;
    mounted.current = true;
    // Strict Mode may reconnect; only the latest effect's bootstrap is consumed.
    void window.gobble.workspace
      .connect()
      .then((result) => {
        if (!active) return;
        if (result.ok) receive(result.value);
        else report(result.error.message);
      })
      .catch(() => {
        if (active) report('The workspace could not be opened. Try again.');
      });
    return () => {
      active = false;
      mounted.current = false;
    };
  }, [receive, report]);

  const schedule = useCallback(<T>(work: () => Promise<T>): Promise<T> => {
    const result = queue.current.then(async () => {
      await drafts.current;
      return work();
    });
    queue.current = result.catch(() => {});
    return result;
  }, []);
  const enqueueCommand = useCallback(
    (resolveAction: (document: WorkspaceDocument) => WorkspaceAction) => {
      const projectId = current.current?.document?.workspace.projectId;
      if (!projectId) return Promise.resolve(false);
      return schedule(async () => {
        const doc = current.current?.document;
        if (!doc || doc.workspace.projectId !== projectId) return false;
        try {
          const input = {
            projectId,
            expectedRevision: doc.workspace.revision,
            requestId: requestId(),
            action: resolveAction(doc),
          };
          let result = await window.gobble.workspace.command(input);
          // Only an explicit revision rejection proves this layout command was
          // not applied. Read without invalidating render leases. Bound retries across overlapping
          // Agent/question/draft commits; each explicit rejection proves no side effect.
          for (
            let retry = 0;
            retry < 4 && !result.ok && result.error.code === 'stale_revision';
            retry++
          ) {
            const fresh = await window.gobble.workspace.read({ projectId });
            if (fresh.ok && current.current?.document?.workspace.projectId === projectId) {
              receiveDocument(fresh.value);
              result = await window.gobble.workspace.command({
                ...input,
                expectedRevision: fresh.value.workspace.revision,
              });
            }
          }
          if (!result.ok) {
            report(result.error.message);
            return false;
          }
          receiveDocument(result.value);
          return true;
        } catch {
          report('This workspace change could not be saved. Try again.');
          return false;
        }
      });
    },
    [receiveDocument, report, schedule],
  );
  const command: Command = useCallback((action) => enqueueCommand(() => action), [enqueueCommand]);
  // Resolve explorer intent after earlier Pane changes, not from a stale render.
  const openResource = useCallback(
    (resource: ResourceRef, destination: 'active' | 'other') =>
      enqueueCommand((document) => ({
        kind: 'open',
        resource,
        pane:
          destination === 'active'
            ? document.activePane
            : document.activePane === 'primary'
              ? 'secondary'
              : 'primary',
        duplicate: false,
      })),
    [enqueueCommand],
  );
  const openProject = useCallback(
    (projectId: string | null) => {
      setBusy(true);
      return schedule(async () => {
        try {
          if (!(await canLeaveProject())) return;
          const result = await window.gobble.workspace.openProject({ projectId });
          if (!result.ok) {
            report(result.error.message);
            return;
          }
          receive(result.value);
          setError(null);
        } catch {
          report('This Project could not be opened. Try again.');
        } finally {
          if (mounted.current) setBusy(false);
        }
      });
    },
    [canLeaveProject, receive, report, schedule],
  );
  const chooseFolder = useCallback(async () => {
    setBusy(true);
    try {
      if (!(await canLeaveProject())) return;
      const result = await window.gobble.projects.chooseFolder({ requestId: requestId() });
      if (!result.ok) {
        report(result.error.message);
        return;
      }
      if (result.value.kind === 'selected') await openProject(result.value.project.projectId);
    } catch {
      report('The folder could not be opened. Try again.');
    } finally {
      if (mounted.current) setBusy(false);
    }
  }, [canLeaveProject, openProject, report]);
  const updateDraft = useCallback(
    (projectId: string, draft: string) => {
      setSaving((count) => count + 1);
      // Send every input directly to the host, whose queue is flushed before normal quit.
      const pending = window.gobble.workspace
        .updateDraft({ projectId, draft })
        .then((result) => {
          if (result.ok) unsavedDrafts.current.delete(projectId);
          else unsavedDrafts.current.add(projectId);
          if (result.ok) receiveDocument(result.value);
          else report(result.error.message);
          return result.ok;
        })
        .catch(() => {
          unsavedDrafts.current.add(projectId);
          report('Your draft could not be saved. Keep it here and try again.');
          return false;
        })
        .finally(() => {
          if (mounted.current) setSaving((count) => Math.max(0, count - 1));
        });
      drafts.current = Promise.all([drafts.current, pending]).then(() => {});
      return pending;
    },
    [receiveDocument, report],
  );
  return {
    state,
    error,
    busy,
    saving: saving > 0,
    command,
    connect,
    openProject,
    openResource,
    chooseFolder,
    updateDraft,
    report,
    clearError: () => setError(null),
  };
}
