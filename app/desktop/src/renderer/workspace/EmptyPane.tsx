import { Icon } from './Icon';

export function EmptyPane({
  views,
  onBrowse,
  onMove,
}: {
  views: { id: string; title: string }[];
  onBrowse: () => void;
  onMove: (surfaceId: string) => void;
}) {
  return (
    <div className="empty-state pane-empty">
      <span className="empty-icon">
        <Icon name="grid" />
      </span>
      <h2>What would you like to open?</h2>
      <p>Choose a pipeline, file or Run from this Project.</p>
      <button onClick={onBrowse}>
        <Icon name="folder" />
        Browse files
      </button>
      {views.length > 0 && (
        <details className="empty-pane-move">
          <summary>Move a view here</summary>
          <div>
            {views.map((view) => (
              <button key={view.id} onClick={() => onMove(view.id)}>
                {view.title}
              </button>
            ))}
          </div>
        </details>
      )}
    </div>
  );
}
