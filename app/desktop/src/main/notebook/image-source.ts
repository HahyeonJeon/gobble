import { NotebookProblem, LIMITS, type ImagePart } from './model';
import { digest } from './text';
/** Bound dimensions before any native decoding. Unsupported animation/orientation is explicit. */
export function imageSource(mime: 'image/png' | 'image/jpeg', base64: string): ImagePart {
  const clean = base64.replace(/[\r\n\t ]/g, '');
  if (clean.length > Math.ceil(LIMITS.imageBytes / 3) * 4 || /[^A-Za-z0-9+/=]/.test(clean))
    throw new NotebookProblem('limit', 'Image encoding is invalid or exceeds 4 MiB.');
  const bytes = Buffer.from(clean, 'base64');
  if (bytes.toString('base64') !== clean)
    throw new NotebookProblem('invalid', 'Image base64 is not canonical.');
  if (!bytes.length || bytes.length > LIMITS.imageBytes)
    throw new NotebookProblem('limit', 'Image bytes exceed the profile.');
  let width = 0,
    height = 0;
  if (mime === 'image/png') {
    if (
      bytes.length < 33 ||
      bytes.subarray(0, 8).toString('hex') !== '89504e470d0a1a0a' ||
      bytes.readUInt32BE(8) !== 13 ||
      bytes.toString('ascii', 12, 16) !== 'IHDR'
    )
      throw new NotebookProblem('invalid', 'Invalid PNG header.');
    width = bytes.readUInt32BE(16);
    height = bytes.readUInt32BE(20);
    let end = false;
    for (let p = 8; p < bytes.length;) {
      if (p + 12 > bytes.length) throw new NotebookProblem('invalid', 'Truncated PNG chunk.');
      const size = bytes.readUInt32BE(p),
        type = bytes.toString('ascii', p + 4, p + 8);
      if (size > bytes.length - p - 12)
        throw new NotebookProblem('invalid', 'Invalid PNG chunk length.');
      if (type === 'acTL')
        throw new NotebookProblem('unsupported', 'Animated PNG is not supported.');
      p += 12 + size;
      if (type === 'IEND') {
        end = p === bytes.length && size === 0;
        break;
      }
    }
    if (!end) throw new NotebookProblem('invalid', 'Invalid PNG ending.');
  } else {
    if (bytes[0] !== 255 || bytes[1] !== 216)
      throw new NotebookProblem('invalid', 'Invalid JPEG header.');
    let found = false;
    for (let p = 2; p + 4 < bytes.length;) {
      if (bytes[p] !== 255) throw new NotebookProblem('invalid', 'Invalid JPEG marker.');
      while (bytes[p] === 255) p++;
      const marker = bytes[p++];
      if (marker === 0xda || marker === 0xd9) break;
      const length = bytes.readUInt16BE(p);
      if (length < 2 || p + length > bytes.length)
        throw new NotebookProblem('invalid', 'Invalid JPEG segment.');
      if (marker === 0xe1)
        throw new NotebookProblem('unsupported', 'JPEG EXIF/orientation requires a later profile.');
      if (marker === 0xc0 || marker === 0xc1 || marker === 0xc2) {
        if (length < 8) throw new NotebookProblem('invalid', 'Invalid JPEG frame.');
        height = bytes.readUInt16BE(p + 3);
        width = bytes.readUInt16BE(p + 5);
        found = true;
      }
      p += length;
    }
    if (!found) throw new NotebookProblem('invalid', 'JPEG dimensions are unavailable.');
  }
  if (!width || !height || width * height > LIMITS.imagePixels)
    throw new NotebookProblem('limit', 'Image exceeds 4 million decoded pixels.');
  return { kind: 'image', mime, base64: clean, width, height, digest: digest(bytes) };
}
