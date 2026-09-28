import { readFile, realpath, stat } from 'node:fs/promises';
import { isAbsolute, relative, resolve, sep } from 'node:path';

export const CONTENT_SECURITY_POLICY = [
  "default-src 'none'",
  "script-src 'self'",
  "style-src 'self'",
  "img-src 'self' blob:",
  "font-src 'self'",
  "connect-src 'none'",
  "frame-src 'none'",
  "frame-ancestors 'none'",
  "object-src 'none'",
  "base-uri 'none'",
  "form-action 'none'",
  "worker-src 'none'",
].join('; ');

const mimeTypes: Readonly<Record<string, string>> = {
  html: 'text/html; charset=utf-8',
  js: 'text/javascript; charset=utf-8',
  css: 'text/css; charset=utf-8',
  png: 'image/png',
  woff2: 'font/woff2',
};

export function assetPath(rawURL: string): string | undefined {
  let url: URL;
  try {
    url = new URL(rawURL);
  } catch {
    return undefined;
  }
  if (
    url.protocol !== 'app:' ||
    url.host !== 'gobble' ||
    url.username ||
    url.password ||
    url.search
  )
    return undefined;
  // The built shell has only one entry and flat hashed assets. No user files are served here.
  if (url.pathname === '/index.html') return 'index.html';
  if (/^\/assets\/[A-Za-z0-9_-]+\.(js|css|png|woff2)$/.test(url.pathname))
    return url.pathname.slice(1);
  return undefined;
}

export async function readShellAsset(root: string, rawURL: string): Promise<Response> {
  const path = assetPath(rawURL);
  if (path === undefined) return new Response(null, { status: 404 });
  try {
    const canonicalRoot = await realpath(root);
    const file = await realpath(resolve(canonicalRoot, path));
    const relativePath = relative(canonicalRoot, file);
    if (isAbsolute(relativePath) || relativePath === '..' || relativePath.startsWith(`..${sep}`)) {
      return new Response(null, { status: 404 });
    }
    const metadata = await stat(file);
    if (!metadata.isFile() || metadata.size > 8 * 1024 * 1024)
      return new Response(null, { status: 404 });
    const extension = path.split('.').at(-1) ?? '';
    const bytes = await readFile(file);
    return new Response(new Uint8Array(bytes), {
      headers: {
        'Content-Type': mimeTypes[extension] ?? 'application/octet-stream',
        'Content-Security-Policy': CONTENT_SECURITY_POLICY,
        'X-Content-Type-Options': 'nosniff',
        'Cache-Control': 'no-store',
      },
    });
  } catch {
    return new Response(null, { status: 404 });
  }
}
