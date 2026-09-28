import { useEffect, useRef, useState } from 'react';

type Props = {
  label: string;
  controls: string;
  orientation: 'horizontal' | 'vertical';
  value: number;
  min: number;
  max: number;
  step: number;
  unit: 'fraction' | 'pixels';
  edge?: 'start' | 'end';
  onPreview: (value: number | null) => void;
  onCommit: (value: number) => Promise<boolean>;
};

/** Accessible separator. Geometry callers supply units and the controlled edge explicitly. */
export function Splitter({
  label,
  controls,
  orientation,
  value,
  min,
  max,
  step,
  unit,
  edge = 'start',
  onPreview,
  onCommit,
}: Props) {
  const [display, setDisplay] = useState(value);
  const current = useRef(value);
  const dragging = useRef(false);
  const pending = useRef(0);
  const [settled, setSettled] = useState(0);
  useEffect(() => {
    if (!dragging.current && pending.current === 0) {
      setDisplay(value);
      current.current = value;
    }
  }, [value, settled]);
  const clamp = (next: number) =>
    Math.max(min, Math.min(max, unit === 'pixels' ? Math.round(next) : next));
  function preview(next: number) {
    current.current = clamp(next);
    setDisplay(current.current);
    onPreview(current.current);
    return current.current;
  }
  async function commit(next: number) {
    dragging.current = false;
    pending.current += 1;
    try {
      await onCommit(clamp(next));
    } finally {
      pending.current -= 1;
      // Earlier save acknowledgments must not replace newer keyboard/pointer intent.
      if (pending.current === 0 && !dragging.current) onPreview(null);
      setSettled((version) => version + 1);
    }
  }
  function cancel() {
    if (!dragging.current) return;
    dragging.current = false;
    current.current = value;
    setDisplay(value);
    onPreview(null);
  }
  const scale = unit === 'fraction' ? 100 : 1;
  return (
    <div
      className="workspace-splitter"
      role="separator"
      aria-label={label}
      aria-orientation={orientation}
      aria-controls={controls}
      aria-valuemin={min * scale}
      aria-valuemax={max * scale}
      aria-valuenow={Math.round(display * scale)}
      aria-valuetext={Math.round(display * scale) + (unit === 'fraction' ? ' percent' : ' pixels')}
      tabIndex={0}
      onKeyDown={(event) => {
        const backward = orientation === 'horizontal' ? 'ArrowUp' : 'ArrowLeft';
        const forward = orientation === 'horizontal' ? 'ArrowDown' : 'ArrowRight';
        const delta = edge === 'start' ? step : -step;
        const next =
          event.key === backward
            ? current.current - delta
            : event.key === forward
              ? current.current + delta
              : event.key === 'Home'
                ? min
                : event.key === 'End'
                  ? max
                  : null;
        if (next === null) return;
        event.preventDefault();
        void commit(preview(next));
      }}
      onPointerDown={(event) => {
        if (event.button !== 0) return;
        event.preventDefault();
        event.currentTarget.focus();
        dragging.current = true;
        event.currentTarget.setPointerCapture(event.pointerId);
      }}
      onPointerMove={(event) => {
        if (!dragging.current) return;
        const bounds = event.currentTarget.parentElement?.getBoundingClientRect();
        if (!bounds) return;
        const length = orientation === 'horizontal' ? bounds.height : bounds.width;
        const offset =
          orientation === 'horizontal' ? event.clientY - bounds.top : event.clientX - bounds.left;
        const amount = edge === 'start' ? offset : length - offset;
        preview(unit === 'fraction' ? amount / length : amount);
      }}
      onPointerUp={(event) => {
        if (!dragging.current) return;
        void commit(current.current);
        event.currentTarget.releasePointerCapture(event.pointerId);
      }}
      onPointerCancel={cancel}
      onLostPointerCapture={cancel}
    >
      <span />
    </div>
  );
}
