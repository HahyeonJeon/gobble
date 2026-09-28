import { useEffect, useState } from 'react';
import type { NotebookReferenceExcerpt } from '@gobble/contracts';
/** Displays only Main's exact captured excerpt; it grants no new Agent observation authority. */
export function NotebookReferenceContent({
  excerpt,
  onReady,
  onFailure,
}: {
  excerpt: NotebookReferenceExcerpt;
  onReady: () => void;
  onFailure: (message: string) => void;
}) {
  const [url, setUrl] = useState<string>();
  useEffect(() => {
    if (excerpt.content.kind === 'text') {
      onReady();
      return;
    }
    let objectURL: string | undefined;
    try {
      const bytes = Uint8Array.from(atob(excerpt.content.base64), (c) => c.charCodeAt(0));
      objectURL = URL.createObjectURL(new Blob([bytes], { type: 'image/png' }));
      setUrl(objectURL);
    } catch {
      onFailure('The Notebook reference image could not be displayed.');
    }
    return () => {
      if (objectURL) URL.revokeObjectURL(objectURL);
    };
  }, [excerpt, onReady, onFailure]);
  return (
    <div
      className="observed-reference-scroll notebook-reference"
      tabIndex={0}
      aria-label="Notebook reference content"
    >
      {excerpt.content.kind === 'text' ? (
        <pre className="observed-reference-text" data-target="true">
          {excerpt.content.text}
        </pre>
      ) : url ? (
        <img
          src={url}
          width={excerpt.content.width}
          height={excerpt.content.height}
          alt="Exact Notebook reference region"
          onLoad={onReady}
          onError={() => onFailure('The Notebook reference image could not be displayed.')}
        />
      ) : (
        <p role="status">Loading reference image…</p>
      )}
    </div>
  );
}
