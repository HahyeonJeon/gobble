import { parentPort } from 'node:worker_threads';
import { parseNotebook } from './parser';
import { NotebookProblem } from './model';
parentPort?.once('message', (input: unknown) => {
  try {
    if (!(input instanceof Uint8Array))
      throw new NotebookProblem('invalid', 'Expected Notebook bytes.');
    parentPort?.postMessage({ ok: true, value: parseNotebook(input) });
  } catch (error) {
    parentPort?.postMessage({
      ok: false,
      code: error instanceof NotebookProblem ? error.code : 'invalid',
      message: error instanceof Error ? error.message : 'Notebook decoding failed.',
    });
  }
});
