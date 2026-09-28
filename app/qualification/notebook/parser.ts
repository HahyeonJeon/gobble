import { LIMITS, NotebookProblem, PROFILE, type Snapshot, type Cell, type Output } from './model';
import { digest, projectText } from './text';
import { imageSource } from './image-source';
function record(value: unknown): Record<string, unknown> {
  if (!value || typeof value !== 'object' || Array.isArray(value))
    throw new NotebookProblem('invalid', 'Expected a JSON object.');
  return value as Record<string, unknown>; /* Object shape checked; field values remain unknown. */
}
function string(value: unknown): string {
  if (typeof value !== 'string') throw new NotebookProblem('invalid', 'Expected text.');
  return value;
}
function multiline(value: unknown): string {
  if (typeof value === 'string') return value;
  if (Array.isArray(value) && value.every((s): s is string => typeof s === 'string'))
    return value.join('');
  throw new NotebookProblem('invalid', 'Expected text or an array of text segments.');
}
function count(value: unknown): void {
  if (value !== null && (!Number.isSafeInteger(value) || typeof value !== 'number' || value < 0))
    throw new NotebookProblem('invalid', 'Invalid saved execution count.');
}
/** Cheap preflight bounds nested JSON before JSON.parse allocates an object tree. */
function preflight(json: string): void {
  let depth = 0,
    nodes = 0,
    quoted = false,
    escaped = false;
  for (const c of json) {
    if (quoted) {
      if (escaped) escaped = false;
      else if (c === '\\') escaped = true;
      else if (c === '"') quoted = false;
      continue;
    }
    if (c === '"') {
      quoted = true;
      nodes++;
    } else if (c === '{' || c === '[') {
      depth++;
      nodes++;
    } else if (c === '}' || c === ']') depth--;
    else if (c === ',') nodes++;
    if (depth > LIMITS.depth || nodes > LIMITS.nodes)
      throw new NotebookProblem('limit', 'Notebook JSON exceeds depth or item limits.');
  }
}
function output(value: unknown, index: number): Output {
  const o = record(value),
    type = string(o.output_type);
  const base = { index, type, mime: '', alternatives: [] as string[], notice: '' };
  if (type === 'stream') {
    if (o.name !== 'stdout' && o.name !== 'stderr')
      throw new NotebookProblem('invalid', 'Invalid stream name.');
    return {
      ...base,
      mime: 'text/plain',
      notice: String(o.name),
      part: projectText(multiline(o.text)),
    };
  }
  if (type === 'error') {
    if (!Array.isArray(o.traceback) || !o.traceback.every((t) => typeof t === 'string'))
      throw new NotebookProblem('invalid', 'Invalid error traceback.');
    return {
      ...base,
      mime: 'text/plain',
      notice: 'Saved error',
      part: projectText(string(o.ename) + ': ' + string(o.evalue) + '\n' + o.traceback.join('\n')),
    };
  }
  if (type !== 'display_data' && type !== 'execute_result')
    return {
      ...base,
      part: { kind: 'unavailable', reason: 'Unknown output type: ' + type.slice(0, 80) },
    };
  if (type === 'execute_result') count(o.execution_count);
  record(o.metadata);
  const data = record(o.data),
    alternatives = Object.keys(data);
  const failures: string[] = [];
  for (const mime of ['image/png', 'image/jpeg', 'text/plain'] as const) {
    if (!(mime in data)) continue;
    try {
      const text = multiline(data[mime]);
      const part = mime === 'text/plain' ? projectText(text) : imageSource(mime, text);
      return {
        ...base,
        mime,
        alternatives,
        part,
        notice: failures.length
          ? 'Static fallback: ' + failures.join(' ')
          : alternatives.some((m) => !['image/png', 'image/jpeg', 'text/plain'].includes(m))
            ? 'Static representation; active alternatives are not rendered.'
            : '',
      };
    } catch (error) {
      failures.push(error instanceof Error ? error.message : 'Invalid representation.');
    }
  }
  return {
    ...base,
    alternatives,
    part: {
      kind: 'unavailable',
      reason: failures.join(' ') || 'No supported saved image or plain-text alternative.',
    },
  };
}
export function parseNotebook(bytes: Uint8Array): Snapshot {
  if (bytes.byteLength > LIMITS.bytes)
    throw new NotebookProblem('limit', 'Notebook exceeds 8 MiB.');
  let json: string;
  try {
    json = new TextDecoder('utf-8', { fatal: true }).decode(bytes);
  } catch {
    throw new NotebookProblem('invalid', 'Notebook is not valid UTF-8.');
  }
  preflight(json);
  let value: unknown;
  try {
    value = JSON.parse(json);
  } catch {
    throw new NotebookProblem('invalid', 'Notebook JSON is malformed.');
  }
  const nb = record(value),
    minor = nb.nbformat_minor;
  if (
    nb.nbformat !== 4 ||
    typeof minor !== 'number' ||
    !Number.isInteger(minor) ||
    minor < 0 ||
    minor > 5
  )
    throw new NotebookProblem('unsupported', 'Supported Notebook formats are 4.0–4.5.');
  const metadata = record(nb.metadata);
  if (!Array.isArray(nb.cells) || nb.cells.length > LIMITS.cells)
    throw new NotebookProblem('limit', 'Notebook requires an array of at most 1,000 cells.');
  const ids = new Set<string>();
  let textBytes = 0,
    outputCount = 0;
  const cells: Cell[] = nb.cells.map((value, index) => {
    const c = record(value);
    record(c.metadata);
    const declaredType = string(c.cell_type);
    let address: Cell['address'];
    if (c.id !== undefined) {
      const id = string(c.id);
      if (!/^[a-zA-Z0-9_-]{1,64}$/.test(id) || ids.has(id))
        throw new NotebookProblem('invalid', 'Cell IDs must be valid and unique.');
      ids.add(id);
      address = { kind: 'id', id };
    } else if (minor === 5) throw new NotebookProblem('invalid', 'Notebook 4.5 requires cell IDs.');
    else address = { kind: 'ordinal', index };
    const type =
      declaredType === 'code' || declaredType === 'markdown' || declaredType === 'raw'
        ? declaredType
        : 'unknown';
    const source = projectText(multiline(c.source));
    textBytes += Buffer.byteLength(source.text);
    let outputs: Output[] = [];
    if (type === 'code') {
      count(c.execution_count);
      if (!Array.isArray(c.outputs) || c.outputs.length > LIMITS.outputsPerCell)
        throw new NotebookProblem('limit', 'Cell outputs exceed 100 or are malformed.');
      outputCount += c.outputs.length;
      if (outputCount > LIMITS.outputs)
        throw new NotebookProblem('limit', 'Notebook exceeds 1,000 outputs.');
      outputs = c.outputs.map(output);
      for (const o of outputs)
        if (o.part.kind === 'text') textBytes += Buffer.byteLength(o.part.text);
    }
    if (textBytes > LIMITS.text)
      throw new NotebookProblem('limit', 'Decoded source and output text exceed 1 MiB.');
    return {
      index,
      address,
      type,
      source,
      outputs,
      notice:
        type === 'unknown'
          ? 'Unknown cell type; source only.'
          : address.kind === 'ordinal'
            ? 'Legacy cell: address is scoped to this file version.'
            : c.attachments
              ? 'Cell attachments are not displayed in this profile.'
              : '',
    };
  });
  const language =
    metadata.language_info &&
    typeof metadata.language_info === 'object' &&
    !Array.isArray(metadata.language_info) &&
    'name' in metadata.language_info &&
    typeof metadata.language_info.name === 'string'
      ? metadata.language_info.name.slice(0, 80)
      : 'Unspecified';
  return {
    profile: PROFILE,
    revision: digest(bytes),
    bytes: bytes.byteLength,
    minor,
    language,
    cells,
    textBytes,
  };
}
