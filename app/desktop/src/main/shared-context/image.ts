import { nativeImage } from 'electron';
import { AppProblem } from '../problem';
import { IMAGE_BYTES, type ImageRenderer } from './observation';

/** Materializes only validated loaded ImageView bytes; never screenshots or paths. */
export const renderObservationImage: ImageRenderer = async (content, selection) => {
  let image = nativeImage.createFromBuffer(Buffer.from(content.base64, 'base64'));
  const size = image.getSize();
  if (image.isEmpty() || size.width !== content.width || size.height !== content.height)
    throw new AppProblem('stale_revision', 'The loaded image dimensions do not match its preview.');
  const x = selection ? Math.floor(selection.x * size.width) : 0;
  const y = selection ? Math.floor(selection.y * size.height) : 0;
  const right = selection
    ? Math.min(size.width, Math.ceil((selection.x + selection.width) * size.width))
    : size.width;
  const bottom = selection
    ? Math.min(size.height, Math.ceil((selection.y + selection.height) * size.height))
    : size.height;
  const crop = { x, y, width: right - x, height: bottom - y };
  image = image.crop(crop);
  let scale = Math.min(1, 1536 / Math.max(crop.width, crop.height));
  for (let attempt = 0; attempt < 16; attempt++) {
    const width = Math.max(1, Math.floor(crop.width * scale));
    const height = Math.max(1, Math.floor(crop.height * scale));
    const base64 = image.resize({ width, height, quality: 'best' }).toPNG().toString('base64');
    if (base64.length <= IMAGE_BYTES)
      return { url: 'data:image/png;base64,' + base64, width, height, crop };
    scale *= 0.75;
  }
  throw new AppProblem(
    'unsupported',
    'This image cannot fit the observation limit. Select a smaller region.',
  );
};
