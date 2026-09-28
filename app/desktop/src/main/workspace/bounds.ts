import type { WindowState } from '@gobble/contracts';

type Bounds = NonNullable<WindowState['bounds']>;
export function clampBounds(saved: Bounds | null, workArea: Bounds): Bounds {
  const width = Math.min(workArea.width, Math.max(720, saved?.width ?? 1280));
  const height = Math.min(workArea.height, Math.max(520, saved?.height ?? 840));
  return {
    width,
    height,
    x: Math.min(
      workArea.x + workArea.width - width,
      Math.max(workArea.x, saved?.x ?? workArea.x + Math.round((workArea.width - width) / 2)),
    ),
    y: Math.min(
      workArea.y + workArea.height - height,
      Math.max(workArea.y, saved?.y ?? workArea.y + Math.round((workArea.height - height) / 2)),
    ),
  };
}
