import { useRef, type ComponentProps } from 'react';
import type { SharedReference, SurfaceLoad } from '@gobble/contracts';
import { PdfView } from './PdfView';
import { ObservedReferencePanel } from '../ObservedReferencePanel';
const ignore = () => {};
/** Keeps the base reader mounted during temporary Show; its native scroll and edit state survive. */
export function PdfSurface({
  load,
  reference,
  ...props
}: ComponentProps<typeof PdfView> & {
  load: SurfaceLoad;
  reference?: SharedReference | undefined;
}) {
  const base = useRef({ content: props.content, surface: props.surface });
  const showing = load.observedReferenceView && reference;
  if (!showing) base.current = { content: props.content, surface: props.surface };
  return (
    <div className="surface-view pdf-surface" data-ready={props.ready}>
      <div className="observed-base" hidden={!!showing}>
        <PdfView
          {...props}
          {...base.current}
          key={JSON.stringify([base.current.content.page.modelHash, base.current.surface.pdf])}
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
          onFailure={props.onFailure}
          command={props.command}
          acknowledgment={load.acknowledgment}
          ready={props.ready}
        />
      )}
    </div>
  );
}
