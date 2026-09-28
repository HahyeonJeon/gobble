import { Type, type Static } from '@sinclair/typebox';
import { closed, TimestampSchema } from './identity';
import { EvidenceRefV3Schema as EvidenceRefSchema } from './reference-v3';
import { RunPresentationSchema } from './run-presentation';
import { LogPresentationSchema, LogStreamSchema } from './log-presentation';
import { ContractValidationError } from './validation-error';

/** Bound into the immutable asset bytes. A target alone cannot substitute this content. */
export const ObservedEvidenceSchema = Type.Object(
  {
    schemaVersion: Type.Literal(1),
    capturedAt: TimestampSchema,
    target: EvidenceRefSchema,
    source: Type.Union([
      Type.Object(
        {
          kind: Type.Literal('run'),
          value: RunPresentationSchema,
          observedTaskCount: Type.Integer({ minimum: 0, maximum: 1000 }),
        },
        closed,
      ),
      Type.Object(
        {
          kind: Type.Literal('log'),
          value: Type.Object(
            {
              ...Type.Omit(LogPresentationSchema, ['streams']).properties,
              streams: Type.Object(
                {
                  stdout: Type.Omit(LogStreamSchema, ['text']),
                  stderr: Type.Omit(LogStreamSchema, ['text']),
                },
                closed,
              ),
            },
            closed,
          ),
          selectedText: Type.String({ maxLength: 65536 }),
          selectedStream: Type.Union([Type.Literal('stdout'), Type.Literal('stderr'), Type.Null()]),
        },
        closed,
      ),
    ]),
  },
  closed,
);
export type ObservedEvidence = Static<typeof ObservedEvidenceSchema>;

/** Exact decoded text coordinates. Never an original-file line or byte range. */
export function textRange(
  text: string,
  selection: { start: { line: number; column: number }; end: { line: number; column: number } },
): string {
  const lines = text.split('\n');
  const offset = (p: { line: number; column: number }) => {
    const line = lines[p.line - 1];
    if (
      !Number.isSafeInteger(p.line) ||
      !Number.isSafeInteger(p.column) ||
      p.line < 1 ||
      p.column < 0 ||
      line === undefined ||
      p.column > line.length ||
      (p.column > 0 &&
        /[\uDC00-\uDFFF]/.test(line[p.column] ?? '') &&
        /[\uD800-\uDBFF]/.test(line[p.column - 1] ?? ''))
    )
      throw new ContractValidationError('Text coordinates do not match the decoded preview.');
    return lines.slice(0, p.line - 1).reduce((n, line) => n + line.length + 1, 0) + p.column;
  };
  const start = offset(selection.start),
    end = offset(selection.end);
  if (end <= start)
    throw new ContractValidationError('Text selection must be nonempty and ordered.');
  return text.slice(start, end);
}

/** Portable semantic preview; deriving it again detects payload/text or subject tampering. */
export function observedEvidenceText(capture: Omit<ObservedEvidence, 'schemaVersion'>): string {
  const { target, source } = capture;
  if (
    target.projectId !== source.value.projectId ||
    target.resource.kind !== source.kind ||
    target.resource.runRef !== source.value.runRef
  )
    throw new ContractValidationError('Captured observation belongs to another subject.');
  const selection = target.selection;
  if (source.kind === 'run') {
    if (
      selection &&
      (selection.kind !== 'run-task' ||
        source.value.tasks.length !== 1 ||
        source.value.tasks[0]?.instanceId !== selection.instanceId ||
        source.value.tasks[0]?.attempt !== selection.attempt)
    )
      throw new ContractValidationError('Captured task does not match its target.');
    return JSON.stringify(source.value);
  }
  if (
    target.resource.kind !== 'log' ||
    target.resource.taskId !== source.value.instance ||
    target.resource.attempt !== source.value.attempt
  )
    throw new ContractValidationError('Captured logs belong to another attempt.');
  if (selection?.kind === 'log-text' && source.selectedStream === selection.stream)
    return source.selectedText;
  if ((!selection || selection.kind === 'text') && source.selectedStream === null)
    return source.selectedText;
  throw new ContractValidationError('Captured log selection is incompatible.');
}
