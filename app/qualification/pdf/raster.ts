import { LIMITS } from './protocol';

/** Check decoded dimensions before allowing the UI to decompress a host-supplied PNG. */
export function assertPngHeader(base64: string, width: number, height: number): void {
  const bytes = Buffer.from(base64, 'base64');
  if (
    bytes.length < 33 ||
    bytes.subarray(0, 8).toString('hex') !== '89504e470d0a1a0a' ||
    bytes.readUInt32BE(8) !== 13 ||
    bytes.toString('ascii', 12, 16) !== 'IHDR' ||
    bytes.readUInt32BE(16) !== width ||
    bytes.readUInt32BE(20) !== height ||
    width * height > LIMITS.rasterPixels
  )
    throw new Error('PDF raster header does not match its bounded dimensions.');
}
