import { useEffect, useRef, useState } from 'react';
import type { PaneId, Surface, WorkspaceAction } from '@gobble/contracts';
import { requestId, type Command } from './useWorkspace';
import { Icon } from './Icon';
import '../styles/pane-actions.css';

/** A transient, source-specific popover. Durable view state remains in the host. */
export function PaneActions({
  surface,
  title,
  linked,
  otherPane,
  command,
  onRefresh,
  referenceActive = false,
}: {
  surface: Surface;
  title: string;
  linked: boolean;
  otherPane: PaneId;
  command: Command;
  onRefresh: () => void;
  referenceActive?: boolean;
}) {
  const [open, setOpen] = useState(false);
  const trigger = useRef<HTMLButtonElement>(null);
  const popover = useRef<HTMLDivElement>(null);
  useEffect(() => {
    if (!open) return;
    const token = requestId();
    const panel = popover.current;
    let current = true;
    void window.gobble.workspace
      .interaction({ token, blocked: true })
      .then((result) => {
        if (!current) return;
        if (!result.ok) {
          setOpen(false);
          return;
        }
        const bounds = trigger.current!.getBoundingClientRect();
        panel!.style.top = Math.min(bounds.bottom + 6, window.innerHeight - 190) + 'px';
        panel!.style.left =
          Math.max(8, Math.min(bounds.right - 248, window.innerWidth - 256)) + 'px';
        panel?.showPopover();
        panel?.querySelector<HTMLButtonElement>('button')?.focus();
      })
      .catch(() => {
        if (current) setOpen(false);
      });
    const dismiss = () => setOpen(false);
    window.addEventListener('resize', dismiss);
    return () => {
      current = false;
      window.removeEventListener('resize', dismiss);
      const returnFocus =
        panel?.contains(document.activeElement) || document.activeElement === document.body;
      panel?.hidePopover();
      void window.gobble.workspace.interaction({ token, blocked: false });
      if (returnFocus) trigger.current?.focus();
    };
  }, [open]);
  function act(action: WorkspaceAction) {
    setOpen(false);
    void command(action);
  }
  return (
    <>
      <button
        ref={trigger}
        className="pane-more"
        aria-label={'More actions for ' + title}
        aria-expanded={open}
        aria-controls={'actions-' + surface.surfaceId}
        onClick={() => setOpen((value) => !value)}
      >
        More <Icon name="chevron" />
      </button>
      <div
        ref={popover}
        id={'actions-' + surface.surfaceId}
        popover="auto"
        className="pane-action-popover"
        role="group"
        aria-label={'Actions for ' + title}
        onBlur={(event) => {
          if (
            event.relatedTarget instanceof Node &&
            !event.currentTarget.contains(event.relatedTarget) &&
            !trigger.current?.contains(event.relatedTarget)
          )
            setOpen(false);
        }}
        onToggle={(event) => {
          if (event.newState === 'closed') setOpen(false);
        }}
      >
        <button
          aria-label={(surface.pinned ? 'Unpin ' : 'Pin ') + title}
          aria-pressed={surface.pinned}
          onClick={() =>
            act({ kind: 'pin', surfaceId: surface.surfaceId, pinned: !surface.pinned })
          }
        >
          <Icon name="pin" />
          {surface.pinned ? 'Unpin view' : 'Pin view'}
        </button>
        <button
          aria-label={'Move ' + title + ' to other pane'}
          onClick={() => act({ kind: 'move', surfaceId: surface.surfaceId, pane: otherPane })}
        >
          <Icon name="arrow" />
          Move to other pane
        </button>
        <button
          aria-label={'Duplicate ' + title + ' in other pane'}
          onClick={() =>
            act({ kind: 'duplicateView', surfaceId: surface.surfaceId, pane: otherPane })
          }
        >
          <Icon name="copy" />
          Duplicate in other pane
        </button>
        <button
          aria-label={'Refresh ' + title}
          disabled={referenceActive}
          title={
            referenceActive
              ? 'Return to your view before refreshing the source.'
              : linked
                ? 'Refresh both linked views. Changed source data clears local selection and view settings.'
                : undefined
          }
          onClick={() => {
            setOpen(false);
            onRefresh();
          }}
        >
          <Icon name="refresh" />
          {linked ? 'Refresh linked views' : 'Refresh view'}
        </button>
      </div>
    </>
  );
}
