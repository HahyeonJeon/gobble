import {
  LogPresentationSchema,
  parse,
  type RunLogs,
  type LogPresentation,
} from '@gobble/contracts';
import { AppProblem } from '../problem';

/** Validate subject before normalizing. Native log paths never enter a display observation. */
export function presentLog(value: RunLogs): LogPresentation {
  const record = value.logs[0];
  const invalid = () =>
    new AppProblem('incompatible_runtime', 'The runtime returned invalid log metadata.');
  if (value.logs.length !== 1 || !record || record.identity !== value.instance) throw invalid();
  function stream(name: 'stdout' | 'stderr') {
    const text = record?.[name + '_tail'] ?? '';
    const size = record?.[name + '_size'] ?? null;
    if (
      typeof text !== 'string' ||
      (size !== null && (typeof size !== 'number' || !Number.isSafeInteger(size) || size < 0))
    )
      throw invalid();
    return {
      text,
      sourceBytesReported: size,
      decodedUtf8Bytes: Buffer.byteLength(text),
      earlierBytesOmitted: typeof size === 'number' && size > value.tailLimitBytes,
      completeness: 'unknown',
      availability: text ? 'text-returned' : 'no-text-returned',
    };
  }
  try {
    return parse(LogPresentationSchema, {
      schemaVersion: 1,
      projectId: value.projectId,
      runRef: value.runRef,
      engineRevision: value.engineRevision,
      observedAt: value.observedAt,
      instance: value.instance,
      attempt: value.attempt,
      tailLimitBytes: value.tailLimitBytes,
      error: record.error ?? null,
      streams: { stdout: stream('stdout'), stderr: stream('stderr') },
    });
  } catch {
    throw invalid();
  }
}
