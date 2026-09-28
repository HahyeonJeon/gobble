import type { CSSProperties } from 'react';

type Name =
  | 'sidebar'
  | 'folder'
  | 'file'
  | 'table'
  | 'image'
  | 'pipeline'
  | 'run'
  | 'agents'
  | 'grid'
  | 'plus'
  | 'close'
  | 'pin'
  | 'split'
  | 'single'
  | 'expand'
  | 'arrow'
  | 'refresh'
  | 'chevron'
  | 'chat'
  | 'copy';
const paths: Record<Name, string> = {
  sidebar: 'M3 4h18v16H3V4Z M9 4v16',
  folder: 'M3 6h6l2 2h10v11H3V6Z M3 6V4h7l2 2h7v2',
  file: 'M6 3h8l4 4v14H6V3Z M14 3v5h4 M9 12h6 M9 16h6',
  table: 'M3 4h18v16H3V4Z M3 9h18 M9 9v11',
  image: 'M3 4h18v16H3V4Z M4 18l5-6 4 4 3-3 5 5 M8 8h.01',
  pipeline: 'M3 3h6v6H3V3Z M15 15h6v6h-6v-6Z M6 9v9h9 M15 3h6v6h-6V3Z M9 6h6',
  run: 'M8 4l12 8-12 8V4Z',
  agents:
    'M8 10a3 3 0 1 0 0-6 3 3 0 0 0 0 6Z M2 20v-3a6 6 0 0 1 12 0v3 M17 5a3 3 0 0 1 0 6 M17 14a5 5 0 0 1 5 5',
  grid: 'M3 3h7v7H3V3Z M14 3h7v7h-7V3Z M3 14h7v7H3v-7Z M14 14h7v7h-7v-7Z',
  plus: 'M12 4v16 M4 12h16',
  close: 'M6 6l12 12 M18 6 6 18',
  pin: 'M8 3h8 M9 3v7l-3 4h12l-3-4V3 M12 14v7',
  split: 'M3 4h18v16H3V4Z M3 12h18',
  single: 'M3 4h18v16H3V4Z',
  expand: 'M14 3h7v7 M21 3l-7 7 M10 21H3v-7 M3 21l7-7',
  arrow: 'M4 12h16 M14 6l6 6-6 6',
  refresh: 'M20 10a8 8 0 1 0-2 8 M20 4v6h-6',
  chevron: 'M9 5l7 7-7 7',
  chat: 'M3 4h18v13H9l-6 4V4Z',
  copy: 'M8 8h13v13H8V8Z M16 8V3H3v13h5',
};
export function Icon({
  name,
  className,
  style,
}: {
  name: Name;
  className?: string;
  style?: CSSProperties;
}) {
  return (
    <svg
      aria-hidden="true"
      viewBox="0 0 24 24"
      className={'icon ' + (className ?? '')}
      style={style}
      fill="none"
      stroke="currentColor"
      strokeWidth="1.6"
      strokeLinecap="round"
      strokeLinejoin="round"
    >
      <path d={paths[name]} />
    </svg>
  );
}
