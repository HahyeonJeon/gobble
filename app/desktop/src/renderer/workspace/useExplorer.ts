import { useEffect, useRef, useState } from 'react';
import { requestId } from './useWorkspace';

/** Window-local navigation visibility; hiding it never destroys file navigation state. */
export function useExplorer(compact: boolean) {
  const [collapsed, setCollapsed] = useState(false);
  const [drawer, setDrawer] = useState(false);
  const panel = useRef<HTMLElement>(null);
  const trigger = useRef<HTMLButtonElement>(null);
  const open = compact ? drawer : !collapsed;
  function closeDrawer() {
    setDrawer(false);
    if (compact) trigger.current?.focus();
  }
  useEffect(() => {
    if (compact && !drawer && panel.current?.contains(document.activeElement))
      trigger.current?.focus();
    if (!compact) {
      if (drawer) setDrawer(false);
      return;
    }
    if (!drawer) return;
    const token = requestId();
    let active = true;
    void window.gobble.workspace
      .interaction({ token, blocked: true })
      .then((result) => {
        if (active && !result.ok) closeDrawer();
      })
      .catch(() => {
        if (active) closeDrawer();
      });
    const escape = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        event.preventDefault();
        closeDrawer();
      }
    };
    const leave = (event: FocusEvent) => {
      if (
        event.target instanceof Node &&
        !panel.current?.contains(event.target) &&
        !trigger.current?.contains(event.target)
      )
        setDrawer(false);
    };
    window.addEventListener('keydown', escape);
    window.addEventListener('focusin', leave);
    panel.current?.querySelector<HTMLElement>('summary')?.focus();
    return () => {
      active = false;
      window.removeEventListener('keydown', escape);
      window.removeEventListener('focusin', leave);
      void window.gobble.workspace.interaction({ token, blocked: false });
    };
  }, [compact, drawer]);
  return {
    open,
    panel,
    trigger,
    closeDrawer,
    show: () => {
      if (compact) setDrawer(true);
      else {
        setCollapsed(false);
        requestAnimationFrame(() => panel.current?.querySelector<HTMLElement>('summary')?.focus());
      }
    },
    toggle: () => {
      if (compact) setDrawer((value) => !value);
      else setCollapsed((value) => !value);
    },
  };
}
