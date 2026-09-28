import {
  getDocument,
  PDFWorker,
  AnnotationMode,
  type PDFDocumentProxy,
  type PDFDocumentLoadingTask,
  type RenderTask,
} from 'pdfjs-dist';
import {
  LIMITS,
  PROFILE,
  type Job,
  type Reply,
  type PageResult,
  type PageModel,
  type Rect,
  type Matrix,
} from '@gobble/contracts/pdf-decoder';
import { assertRegion, pixelRegion } from '@gobble/contracts/pdf-geometry';

declare global {
  interface Window {
    pdfJob: { listen(callback: (job: Job) => void): void; reply(value: Reply): void };
  }
}
let documentProxy: PDFDocumentProxy | undefined;
let loadingTask: PDFDocumentLoadingTask | undefined;
let current: { page: PageResult; canvas: HTMLCanvasElement } | undefined;
let renderTask: RenderTask | undefined;
let sourceRevision = '';
let busy = false;
const diagnostics: string[] = [];
const workerPort = new Worker('pdf-host://decoder/worker.js', { type: 'module' });
let ready: () => void;
const workerReady = new Promise<void>((resolve) => {
  ready = resolve;
});
workerPort.addEventListener('message', (event: MessageEvent) => {
  if (event.data?.type === 'gobble-pdf-ready') {
    event.stopImmediatePropagation();
    ready();
    return;
  }
  if (event.data?.type !== 'gobble-pdf-warning') return;
  event.stopImmediatePropagation();
  if (diagnostics.length < 20) diagnostics.push(String(event.data.message).slice(0, 300));
});
let pdfWorker: PDFWorker | undefined;
const warn = console.warn.bind(console);
console.warn = (...args: unknown[]) => {
  const text = args.map(String).join(' ');
  if (text.startsWith('Warning:')) {
    if (diagnostics.length < 20) diagnostics.push(text.slice(0, 300));
  } else warn(...args);
};
async function sha(bytes: Uint8Array) {
  const digest = await crypto.subtle.digest('SHA-256', bytes.slice().buffer);
  return (
    'sha256:' + Array.from(new Uint8Array(digest), (n) => n.toString(16).padStart(2, '0')).join('')
  );
}
const modelHash = (model: PageModel) => sha(new TextEncoder().encode(JSON.stringify(model)));
function png(canvas: HTMLCanvasElement, limit: number) {
  const encoded = canvas.toDataURL('image/png').split(',')[1]!;
  if (encoded.length > limit)
    throw new Error('Rendered image exceeds the evidence byte limit. Select a smaller region.');
  return { width: canvas.width, height: canvas.height, png: encoded };
}
async function run(job: Job) {
  diagnostics.length = 0;
  if (job.kind === 'open') {
    if (job.bytes.byteLength > LIMITS.fileBytes)
      throw new Error('PDF exceeds the 8 MiB qualification limit.');
    sourceRevision = await sha(job.bytes);
    if (sourceRevision !== job.revision) throw new Error('Source hash mismatch.');
    if (loadingTask) await loadingTask.destroy();
    current = undefined;
    await workerReady;
    pdfWorker ??= PDFWorker.create({ port: workerPort });
    const loading = getDocument({
      data: job.bytes,
      worker: pdfWorker,
      standardFontDataUrl: 'pdf-host://decoder/vendor/standard_fonts/',
      cMapUrl: 'pdf-host://decoder/vendor/cmaps/',
      cMapPacked: true,
      useSystemFonts: false,
      disableFontFace: true,
      useWasm: false,
      enableXfa: false,
      // This tuple can resolve a partial operator stream before propagating its error.
      // Its best-effort mode reports recoverable omissions as warnings instead; the
      // worker bridge and the display warning hook make every such warning fatal here.
      stopAtErrors: false,
      maxImageSize: LIMITS.sourceImagePixels,
      disableAutoFetch: true,
      disableStream: true,
    });
    loadingTask = loading;
    // No password callback means an encrypted document rejects instead of waiting for input.
    documentProxy = await loading.promise;
    if (documentProxy.numPages > LIMITS.pages)
      throw new Error('PDF exceeds the 200-page qualification limit.');
    if (diagnostics.length) throw new Error('Incomplete PDF decode: ' + diagnostics[0]);
    return {
      kind: 'opened',
      revision: sourceRevision,
      profile: PROFILE,
      pageCount: documentProxy.numPages,
    } as const;
  }
  if (!documentProxy) throw new Error('Open a PDF first.');
  if (job.kind === 'page') {
    current = undefined;
    if (
      !Number.isInteger(job.pageIndex) ||
      job.pageIndex < 0 ||
      job.pageIndex >= documentProxy.numPages
    )
      throw new Error('Page is outside the document.');
    if (
      !Number.isFinite(job.scale) ||
      job.scale < 0.1 ||
      job.scale > 4 ||
      ![1, 2].includes(job.dpr) ||
      ![0, 90, 180, 270].includes(job.rotation)
    )
      throw new Error('Invalid page presentation.');
    const start = performance.now(),
      page = await documentProxy.getPage(job.pageIndex + 1);
    const viewBox = page.view as Rect;
    const viewport = page.getViewport({
      scale: job.scale * job.dpr,
      rotation: (page.rotate + job.rotation) % 360,
    });
    const width = Math.ceil(viewport.width),
      height = Math.ceil(viewport.height);
    if (
      !Number.isSafeInteger(width) ||
      !Number.isSafeInteger(height) ||
      width < 1 ||
      height < 1 ||
      width * height > LIMITS.rasterPixels
    )
      throw new Error('PDF page exceeds the 4-megapixel raster limit.');
    const text = await page.getTextContent({ disableNormalization: true });
    const items: PageModel['text']['items'] = [];
    let chars = 0,
      truncated = false;
    for (const item of text.items) {
      if (!('str' in item)) continue;
      if (items.length === LIMITS.textItems || chars + item.str.length > LIMITS.textUtf16) {
        truncated = true;
        break;
      }
      items.push({
        id: items.length,
        text: item.str,
        transform: item.transform as Matrix,
        width: item.width,
        height: item.height,
        direction: item.dir,
      });
      chars += item.str.length;
    }
    const model: PageModel = {
      revision: sourceRevision,
      profile: PROFILE,
      pageIndex: job.pageIndex,
      viewBox,
      userUnit: page.userUnit,
      intrinsicRotation: page.rotate as 0 | 90 | 180 | 270,
      text: { capability: 'diagnostic-only', items, truncated },
    };
    const hash = await modelHash(model);
    const modelMs = performance.now() - start;
    const canvas = document.createElement('canvas');
    canvas.width = width;
    canvas.height = height;
    const renderStart = performance.now();
    renderTask = page.render({
      canvas,
      viewport,
      annotationMode: AnnotationMode.DISABLE,
      background: 'rgb(255,255,255)',
    });
    try {
      await renderTask.promise;
    } finally {
      renderTask = undefined;
    }
    if (diagnostics.length) throw new Error('Incomplete PDF render: ' + diagnostics[0]);
    const result: PageResult = {
      kind: 'page',
      model,
      modelHash: hash,
      renditionId: job.jobId,
      viewport: viewport.transform as Matrix,
      raster: png(canvas, LIMITS.pagePngBase64),
      timings: { modelMs, renderMs: performance.now() - renderStart },
    };
    current = { page: result, canvas };
    page.cleanup();
    return result;
  }
  if (
    !current ||
    current.page.renditionId !== job.renditionId ||
    current.page.modelHash !== job.modelHash
  )
    throw new Error('The PDF rendition is no longer current.');
  assertRegion(job.region, current.page.model.viewBox);
  const pixelRect = pixelRegion(
    job.region,
    current.page.viewport,
    current.canvas.width,
    current.canvas.height,
  );
  const canvas = document.createElement('canvas');
  canvas.width = pixelRect.width;
  canvas.height = pixelRect.height;
  canvas
    .getContext('2d')!
    .drawImage(
      current.canvas,
      pixelRect.x,
      pixelRect.y,
      pixelRect.width,
      pixelRect.height,
      0,
      0,
      pixelRect.width,
      pixelRect.height,
    );
  return {
    kind: 'capture',
    revision: sourceRevision,
    modelHash: current.page.modelHash,
    renditionId: current.page.renditionId,
    pageIndex: current.page.model.pageIndex,
    region: job.region,
    pixelRect,
    raster: png(canvas, LIMITS.capturePngBase64),
  } as const;
}
window.pdfJob.listen((job) => {
  if (busy) {
    window.pdfJob.reply({ jobId: job.jobId, ok: false, message: 'PDF host already has a job.' });
    return;
  }
  busy = true;
  void run(job)
    .then(
      (value) => window.pdfJob.reply({ jobId: job.jobId, ok: true, value }),
      (error) =>
        window.pdfJob.reply({
          jobId: job.jobId,
          ok: false,
          message: String(error instanceof Error ? error.message : error).slice(0, 500),
        }),
    )
    .finally(() => {
      busy = false;
    });
});
window.addEventListener('pagehide', () => {
  renderTask?.cancel();
  void loadingTask?.destroy();
  pdfWorker?.destroy();
  workerPort.terminate();
});
