/** Read-only textareas scroll on arrow keys on macOS; give the communication view explicit caret selection. */
export function moveTextRange(
  text: string,
  start: number,
  end: number,
  direction: 'forward' | 'backward' | 'none',
  key: string,
  extend: boolean,
  wholeDocument: boolean,
) {
  if (!['ArrowLeft', 'ArrowRight', 'ArrowUp', 'ArrowDown', 'Home', 'End'].includes(key))
    return null;
  const anchor = direction === 'backward' ? end : start;
  const focus = direction === 'backward' ? start : end;
  const lineStart = focus === 0 ? 0 : text.lastIndexOf('\n', focus - 1) + 1;
  const newline = text.indexOf('\n', focus);
  const lineEnd = newline < 0 ? text.length : newline;
  let next = focus;
  if (key === 'ArrowLeft') next = !extend && start !== end ? start : Math.max(0, focus - 1);
  if (key === 'ArrowRight')
    next =
      !extend && start !== end
        ? end
        : Math.min(text.length, focus + (text.codePointAt(focus)! > 0xffff ? 2 : 1));
  if (key === 'Home') next = wholeDocument ? 0 : lineStart;
  if (key === 'End') next = wholeDocument ? text.length : lineEnd;
  if (key === 'ArrowUp')
    next =
      lineStart === 0
        ? 0
        : Math.min(lineStart - 1, text.lastIndexOf('\n', lineStart - 2) + 1 + focus - lineStart);
  if (key === 'ArrowDown') {
    const nextEnd = text.indexOf('\n', lineEnd + 1);
    next =
      lineEnd === text.length
        ? text.length
        : Math.min(nextEnd < 0 ? text.length : nextEnd, lineEnd + 1 + focus - lineStart);
  }
  // UTF-16 addresses must not bisect a displayed code point.
  if (
    next > 0 &&
    /[\uDC00-\uDFFF]/.test(text[next] ?? '') &&
    /[\uD800-\uDBFF]/.test(text[next - 1] ?? '')
  )
    next--;
  return extend
    ? {
        start: Math.min(anchor, next),
        end: Math.max(anchor, next),
        direction: next < anchor ? ('backward' as const) : ('forward' as const),
      }
    : { start: next, end: next, direction: 'none' as const };
}
