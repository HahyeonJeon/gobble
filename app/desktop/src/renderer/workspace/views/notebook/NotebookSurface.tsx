import { useRef, type ComponentProps } from 'react';
import type { SharedReference, SurfaceLoad } from '@gobble/contracts';
import { NotebookView } from './NotebookView';
import { ObservedReferencePanel } from '../ObservedReferencePanel';
const ignore = () => {};
/** Preserve the mounted reader, including unsaved image gestures, throughout Show/Return. */
export function NotebookSurface({
  load,
  reference,
  onFailure,
  ...props
}: ComponentProps<typeof NotebookView> & {
  load: SurfaceLoad;
  reference: SharedReference | undefined;
  onFailure: (message: string) => void;
}) {
  const base = useRef({
    document: props.document,
    surface: props.surface,
    acknowledgment: props.acknowledgment,
  });
  const showing = load.observedReferenceView && reference;
  if (!showing)
    base.current = {
      document: props.document,
      surface: props.surface,
      acknowledgment: props.acknowledgment,
    };
  return (
    <div className="surface-view notebook-surface" data-ready={props.ready}>
      <div className="observed-base" hidden={!!showing}>
        <NotebookView
          {...props}
          {...base.current}
          key={base.current.document.revision}
          ready={props.ready && !showing}
          onReady={showing ? ignore : props.onReady}
        />
      </div>
      {showing && (
        <ObservedReferencePanel
          view={load.observedReferenceView!}
          reference={reference}
          data={load.data}
          onReady={props.onReady}
          onFailure={onFailure}
          command={props.command}
          acknowledgment={load.acknowledgment}
          ready={props.ready}
        />
      )}
    </div>
  );
}
