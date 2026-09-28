import { Worker } from 'node:worker_threads';
import { LIMITS, NotebookProblem, type Snapshot } from './model';
import { parseWorkerReply } from './worker-protocol';
let active = 0;
export const activeWorkers = (): number => active;
/** One cancellable worker per read; at most two reads across the App. */
export async function readNotebook(
  bytes: Uint8Array,
  signal: AbortSignal,
  workerPath: string,
  timeoutMs: number = LIMITS.jobMs,
): Promise<Snapshot> {
  if (bytes.byteLength > LIMITS.bytes)
    throw new NotebookProblem('limit', 'Notebook exceeds 8 MiB.');
  if (signal.aborted) throw new NotebookProblem('cancelled', 'Notebook read was cancelled.');
  if (active >= 2) throw new NotebookProblem('busy', 'Two Notebook reads are already active.');
  const worker = new Worker(workerPath, {
    resourceLimits: { maxOldGenerationSizeMb: 128, maxYoungGenerationSizeMb: 16 },
  });
  active++;
  let timer: ReturnType<typeof setTimeout> | undefined;
  let abort = () => {};
  try {
    return await new Promise<Snapshot>((resolve, reject) => {
      abort = () => reject(new NotebookProblem('cancelled', 'Notebook read was cancelled.'));
      signal.addEventListener('abort', abort, { once: true });
      timer = setTimeout(
        () => reject(new NotebookProblem('timeout', 'Notebook read exceeded its deadline.')),
        timeoutMs,
      );
      worker.once('message', (message: unknown) => {
        try {
          const reply = parseWorkerReply(message);
          if (reply.ok) resolve(reply.value);
          else reject(new NotebookProblem(reply.code, reply.message));
        } catch (error) {
          reject(error);
        }
      });
      worker.once('error', reject);
      worker.once('exit', () =>
        reject(new NotebookProblem('invalid', 'Notebook worker stopped before completing.')),
      );
      worker.postMessage(bytes);
    });
  } finally {
    clearTimeout(timer);
    signal.removeEventListener('abort', abort);
    await worker.terminate();
    worker.removeAllListeners();
    active--;
  }
}
