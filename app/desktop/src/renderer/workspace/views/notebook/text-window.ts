/** Bound DOM work independently of the full document/text transport limit. */
export function textWindow(
  text: string,
  start: number,
): { text: string; start: number; end: number } {
  let end = Math.min(text.length, start + 2048),
    line = start;
  for (let i = 0; i < 200; i++) {
    const next = text.indexOf('\n', line);
    if (next < 0 || next >= end) break;
    line = next + 1;
    if (i === 199) end = line;
  }
  if (end < text.length) {
    const char = text.charCodeAt(end);
    if (char >= 0xdc00 && char <= 0xdfff) end--;
    const escape = text.slice(Math.max(start, end - 5), end).match(/\\u[0-9a-f]{0,3}$/i);
    if (escape) end -= escape[0].length;
  }
  return { text: text.slice(start, end), start, end };
}
