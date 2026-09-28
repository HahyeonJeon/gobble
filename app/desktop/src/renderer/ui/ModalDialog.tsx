import { useEffect, useRef, type ReactNode } from 'react';
import { requestId } from '../workspace/useWorkspace';
import { createPortal } from 'react-dom';
import '../styles/dialog.css';

export function ModalDialog({
  title,
  children,
  onClose,
  className,
  onAfterClose,
}: {
  title: string;
  children: ReactNode;
  onClose: () => void;
  className?: string;
  onAfterClose?: () => void;
}) {
  const ref = useRef<HTMLDialogElement>(null);
  useEffect(() => {
    const dialog = ref.current;
    const trigger = document.activeElement;
    const token = requestId();
    let mounted = true;
    void window.gobble.workspace.interaction({ token, blocked: true }).then((result) => {
      if (mounted && result.ok) dialog?.showModal();
      if (mounted && !result.ok) onClose();
    });
    return () => {
      mounted = false;
      dialog?.close();
      void window.gobble.workspace.interaction({ token, blocked: false });
      if (trigger instanceof HTMLElement && trigger.isConnected) trigger.focus();
      onAfterClose?.();
    };
  }, []);
  // Native modality does not detach a dialog from ancestor CSS. Own its DOM
  // placement here so navigation styles cannot override account/agent controls.
  return createPortal(
    <dialog
      ref={ref}
      className={'modal-dialog ' + (className ?? '')}
      aria-label={title}
      onCancel={(event) => {
        event.preventDefault();
        onClose();
      }}
    >
      <header>
        <h2>{title}</h2>
        <button type="button" onClick={onClose} aria-label={'Close ' + title}>
          Close
        </button>
      </header>
      {children}
    </dialog>,
    document.body,
  );
}
