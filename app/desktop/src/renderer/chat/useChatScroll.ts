import { useLayoutEffect, useRef, useState } from 'react';

type Anchor = { id: string; offset: number };

/** Transient reading state. Hidden regions retain their last visible anchor. */
export function useChatScroll(version: string) {
  const list = useRef<HTMLDivElement>(null);
  const content = useRef<HTMLDivElement>(null);
  const following = useRef(true);
  const anchor = useRef<Anchor | null>(null);
  const programmedTop = useRef<number | null>(null);
  const [unread, setUnread] = useState(false);

  function entries() {
    return [...(content.current?.querySelectorAll<HTMLElement>('[data-chat-item]') ?? [])];
  }
  function capture() {
    const element = list.current;
    if (!element?.clientHeight) return;
    const top = element.getBoundingClientRect().top;
    const item = entries().find((item) => item.getBoundingClientRect().bottom > top);
    if (item)
      anchor.current = {
        id: item.dataset.chatItem!,
        offset: item.getBoundingClientRect().top - top,
      };
  }
  function restore() {
    const element = list.current;
    if (!element?.clientHeight) return;
    if (following.current) element.scrollTop = element.scrollHeight;
    else if (anchor.current) {
      const item = entries().find((item) => item.dataset.chatItem === anchor.current!.id);
      if (item)
        element.scrollTop +=
          item.getBoundingClientRect().top -
          element.getBoundingClientRect().top -
          anchor.current.offset;
    }
    programmedTop.current = element.scrollTop;
    capture();
  }
  useLayoutEffect(() => {
    restore();
    if (!following.current) setUnread(true);
  }, [version]);
  useLayoutEffect(() => {
    const observer = new ResizeObserver(restore);
    if (list.current) observer.observe(list.current);
    if (content.current) observer.observe(content.current);
    return () => observer.disconnect();
  }, []);
  return {
    list,
    content,
    unread,
    onScroll: () => {
      const element = list.current;
      if (!element?.clientHeight) return;
      if (programmedTop.current !== null && Math.abs(element.scrollTop - programmedTop.current) < 1)
        return;
      programmedTop.current = null;
      following.current = element.scrollHeight - element.scrollTop - element.clientHeight < 48;
      if (following.current) setUnread(false);
      capture();
    },
    latest: () => {
      following.current = true;
      setUnread(false);
      restore();
    },
    reveal: (id: string) => {
      const item = entries().find((item) => item.dataset.chatItem === id);
      if (!item || !list.current?.clientHeight) return;
      following.current = false;
      anchor.current = { id, offset: 12 };
      restore();
      item.focus({ preventScroll: true });
    },
  };
}
