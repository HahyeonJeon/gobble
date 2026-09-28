import { useEffect, useState } from 'react';
export function usePageImage(base64: string) {
  const [resource, setResource] = useState<{ source: string; url: string }>();
  useEffect(() => {
    const bytes = Uint8Array.from(atob(base64), (c) => c.charCodeAt(0));
    const url = URL.createObjectURL(new Blob([bytes], { type: 'image/png' }));
    setResource({ source: base64, url });
    return () => URL.revokeObjectURL(url);
  }, [base64]);
  return resource?.source === base64 ? resource.url : undefined;
}
