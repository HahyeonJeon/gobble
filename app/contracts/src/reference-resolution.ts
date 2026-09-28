import { Value } from '@sinclair/typebox/value';
import { pipelineSubject } from './pipeline-reference';
import { notebookPart } from './notebook';
import { validatePdfSelection } from './pdf';
import { resolveDependencyTarget } from './dependency-resolution';
import { Type, type Static } from '@sinclair/typebox';
import { closed, RevisionSchema } from './identity';
import type { ReferenceTarget } from './reference-target';
import { sameResource, type ResourceRef } from './resource';
import type { SurfaceData } from './surface-data';
import { logPreviewText } from './run';
import { validateEvidence } from './validation';
import { textRange } from './observed-evidence';

export const ReferenceResolutionSchema = Type.Union([
  Type.Object({ kind: Type.Literal('exact') }, closed),
  Type.Object({ kind: Type.Literal('historical'), currentRevision: RevisionSchema }, closed),
  Type.Object(
    {
      kind: Type.Literal('unavailable'),
      reason: Type.Union([
        Type.Literal('source-unavailable'),
        Type.Literal('resource-mismatch'),
        Type.Literal('selection-unavailable'),
      ]),
    },
    closed,
  ),
]);
export type ReferenceResolution = Static<typeof ReferenceResolutionSchema>;
export type ReferenceSource = {
  projectId: string;
  resource: ResourceRef;
  dataRevision: string;
  data: SurfaceData;
};

/** Exact identity only. No quote search, ordinal reuse across revisions or implicit remapping. */
export function resolveReferenceTarget(
  target: ReferenceTarget,
  source?: ReferenceSource,
): ReferenceResolution {
  if (!source) return { kind: 'unavailable', reason: 'source-unavailable' };
  if (target.schemaVersion === 4)
    return resolveDependencyTarget(
      target,
      source.data.kind === 'run' && source.data.dependencies
        ? { observation: source.data.dependencies, dataRevision: source.dataRevision }
        : undefined,
    );
  const data = source.data;
  if (target.schemaVersion === 8 || data.kind === 'report') {
    if (
      target.schemaVersion !== 8 ||
      data.kind !== 'report' ||
      source.projectId !== target.projectId ||
      data.saved.projectId !== target.projectId ||
      !sameResource(source.resource, target.resource)
    )
      return { kind: 'unavailable', reason: 'resource-mismatch' };
    try {
      validateEvidence(target, source.projectId);
    } catch {
      return { kind: 'unavailable', reason: 'selection-unavailable' };
    }
    return target.dataRevision === data.saved.asset.hash &&
      Value.Equal(target.resource.saved, data.saved)
      ? { kind: 'exact' }
      : { kind: 'unavailable', reason: 'resource-mismatch' };
  }
  if (target.schemaVersion === 7 || data.kind === 'pipeline') {
    if (
      target.schemaVersion !== 7 ||
      data.kind !== 'pipeline' ||
      source.projectId !== target.projectId ||
      data.value.projectId !== target.projectId ||
      data.value.pipelineId !== target.resource.pipelineId ||
      !sameResource(source.resource, target.resource)
    )
      return { kind: 'unavailable', reason: 'resource-mismatch' };
    if (!data.value.artifact) return { kind: 'unavailable', reason: 'source-unavailable' };
    if (target.dataRevision !== data.value.artifact.artifactId)
      return { kind: 'historical', currentRevision: data.value.artifact.artifactId };
    try {
      validateEvidence(target, source.projectId);
      if (target.sourceRevision !== data.value.artifact.sourceRevision) throw new Error();
      pipelineSubject(data.value.artifact.flow, target.selection.subject);
      return { kind: 'exact' };
    } catch {
      return { kind: 'unavailable', reason: 'selection-unavailable' };
    }
  }
  const resource: ResourceRef =
    data.kind === 'file'
      ? { kind: 'file', resourceId: data.value.resourceId }
      : data.kind === 'run'
        ? { kind: 'run', runRef: data.value.runRef }
        : {
            kind: 'log',
            runRef: data.value.runRef,
            taskId: data.value.instance,
            attempt: data.value.attempt,
          };
  if (
    source.projectId !== target.projectId ||
    data.value.projectId !== target.projectId ||
    !sameResource(source.resource, target.resource) ||
    !sameResource(resource, target.resource)
  )
    return { kind: 'unavailable', reason: 'resource-mismatch' };
  try {
    validateEvidence(target, source.projectId);
  } catch {
    return { kind: 'unavailable', reason: 'selection-unavailable' };
  }
  if (target.dataRevision !== source.dataRevision)
    return { kind: 'historical', currentRevision: source.dataRevision };
  return selectionMatches(target, source.data)
    ? { kind: 'exact' }
    : { kind: 'unavailable', reason: 'selection-unavailable' };
}

function selectionMatches(evidence: ReferenceTarget, data: SurfaceData): boolean {
  const selection = evidence.selection;
  if (!selection)
    return !(data.kind === 'file' && ['pdf', 'notebook'].includes(data.value.content.kind));
  if (selection.kind === 'pipeline') return false;
  if (selection.kind === 'notebook') {
    if (
      evidence.schemaVersion !== 6 ||
      data.kind !== 'file' ||
      data.value.content.kind !== 'notebook' ||
      data.value.content.document.revision !== evidence.dataRevision
    )
      return false;
    try {
      notebookPart(data.value.content.document, selection);
      return true;
    } catch {
      return false;
    }
  }
  if (selection.kind === 'pdf') {
    if (evidence.schemaVersion !== 5 || data.kind !== 'file' || data.value.content.kind !== 'pdf')
      return false;
    try {
      validatePdfSelection(selection, data.value.content);
      return true;
    } catch {
      return false;
    }
  }
  if (selection.kind === 'run-group' || selection.kind === 'run-dependency') return false;
  if (selection.kind === 'run-task')
    return (
      data.kind === 'run' &&
      data.value.tasks.some(
        (task) => task.instanceId === selection.instanceId && task.attempt === selection.attempt,
      )
    );
  if (selection.kind === 'log-text') {
    if (data.kind !== 'log' || !('schemaVersion' in data.value)) return false;
    try {
      textRange(data.value.streams[selection.stream].text, selection);
      return true;
    } catch {
      return false;
    }
  }

  if (selection.kind === 'table') {
    if (data.kind !== 'file' || data.value.content.kind !== 'table') return false;
    const content = data.value.content;
    if (
      selection.rowKeys.some((key) => !content.rows.some((row) => row.key === key)) ||
      selection.columns.some((id) => !content.columns.some((column) => column.id === id))
    )
      return false;
  } else if (selection.kind === 'image') {
    if (data.kind !== 'file' || data.value.content.kind !== 'image') return false;
    if (
      selection.contentHash !== data.value.revision ||
      selection.originalWidth !== data.value.content.width ||
      selection.originalHeight !== data.value.content.height
    )
      return false;
  } else {
    const text =
      data.kind === 'file' && data.value.content.kind === 'text'
        ? data.value.content.text
        : data.kind === 'log'
          ? logPreviewText(data.value)
          : undefined;
    if (text === undefined) return false;
    const lines = text.split('\n');
    for (const position of [selection.start, selection.end]) {
      const line = lines[position.line - 1];
      if (line === undefined || position.column > line.length) return false;
      if (
        line &&
        position.column > 0 &&
        /[\uDC00-\uDFFF]/.test(line[position.column] ?? '') &&
        /[\uD800-\uDBFF]/.test(line[position.column - 1] ?? '')
      )
        return false;
    }
  }
  return true;
}
