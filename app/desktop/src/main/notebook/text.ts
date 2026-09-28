import { createHash } from 'node:crypto';
import { NotebookProblem, PROFILE, type TextPart } from './model';
export const digest = (bytes: string | Uint8Array): string =>
  'sha256:' + createHash('sha256').update(bytes).digest('hex');
/** Browser text uses LF; control characters are literal escapes, never terminal commands. */
function segment(raw: string, index: number): { text: string; length: number } {
  const point = raw.codePointAt(index);
  if (point === undefined) throw new NotebookProblem('invalid', 'Missing text position.');
  if (point >= 0xd800 && point <= 0xdfff)
    throw new NotebookProblem('invalid', 'Unpaired Unicode surrogate.');
  if (point === 13) return { text: '\n', length: raw[index + 1] === '\n' ? 2 : 1 };
  if ((point < 32 && point !== 9 && point !== 10) || (point >= 127 && point <= 159))
    return { text: '\\u' + point.toString(16).padStart(4, '0'), length: 1 };
  return { text: String.fromCodePoint(point), length: point > 0xffff ? 2 : 1 };
}
export function projectText(raw: string): TextPart {
  const chunks: string[] = [];
  let from = 0;
  for (let i = 0; i < raw.length;) {
    const s = segment(raw, i);
    if (s.text !== raw.slice(i, i + s.length)) {
      chunks.push(raw.slice(from, i), s.text);
      from = i + s.length;
    }
    i += s.length;
  }
  chunks.push(raw.slice(from));
  const text = chunks.join('');
  return { kind: 'text', raw, text, digest: digest(PROFILE + '\0' + text) };
}
/** Selection offsets address displayed UTF-16; raw quotes retain original line endings. */
export function rawRange(part: TextPart, start: number, end: number): string {
  if (
    !Number.isSafeInteger(start) ||
    !Number.isSafeInteger(end) ||
    start < 0 ||
    end <= start ||
    end > part.text.length
  )
    throw new NotebookProblem('invalid', 'Select a non-empty, in-bounds text range.');
  let display = 0,
    rawStart: number | undefined,
    rawEnd: number | undefined;
  for (let i = 0; i <= part.raw.length;) {
    if (display === start) rawStart = i;
    if (display === end) {
      rawEnd = i;
      break;
    }
    if (i === part.raw.length) break;
    const s = segment(part.raw, i);
    display += s.text.length;
    i += s.length;
  }
  if (rawStart === undefined || rawEnd === undefined)
    throw new NotebookProblem(
      'invalid',
      'The range splits a Unicode character or displayed control escape.',
    );
  return part.raw.slice(rawStart, rawEnd);
}
