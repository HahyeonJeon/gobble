import { Value } from '@sinclair/typebox/value';
import { samePipelineSelector } from './pipeline-reference';
import { notebookSelectionContains } from './notebook-viewport';
import { Type, type Static } from '@sinclair/typebox';
import { closed, RequestIdSchema } from './identity';
import { SharedReferenceIdSchema } from './shared-context';
import { EvidenceRefSchema, type EvidenceRef } from './reference-target';
import type { Surface } from './surface';
import { defaultDependencyNavigation } from './dependency-navigation';
import { sameResource } from './resource';

export const ObservedReadIdSchema = Type.String({
  pattern: '^obs_[A-Za-z0-9_-]{1,80}$',
  maxLength: 84,
});
/** Transient Main-owned presentation; base Surface and local selection are never overwritten. */
export const ObservedReferenceViewSchema = Type.Object(
  {
    requestId: RequestIdSchema,
    referenceId: SharedReferenceIdSchema,
    evidence: EvidenceRefSchema,
  },
  closed,
);
export type ObservedReferenceView = Static<typeof ObservedReferenceViewSchema>;

export function observedReferenceSurface(
  surface: Surface,
  reference?: ObservedReferenceView,
): Surface {
  if (!reference) return surface;
  if (surface.view === 'pdf' && reference.evidence.schemaVersion === 5)
    return {
      ...surface,
      pdf: {
        pageIndex: reference.evidence.selection.pageIndex,
        scale: 1,
        rotation: 0,
        fitWidth: true,
      },
    };
  const selection = reference.evidence.selection;
  if (
    surface.view === 'run' &&
    (selection?.kind === 'run-group' || selection?.kind === 'run-dependency')
  )
    return {
      ...surface,
      runView: {
        query: '',
        status: null,
        viewRevision: surface.runView?.viewRevision ?? 0,
        mode: 'dependencies',
        dependencies: defaultDependencyNavigation(),
      },
    };
  if (surface.view === 'run' && selection?.kind === 'run-task')
    return {
      ...surface,
      runView: {
        mode: 'tasks',
        query: '',
        status: null,
        viewRevision: surface.runView?.viewRevision ?? 0,
      },
    };
  if (surface.view === 'log' && selection?.kind === 'log-text')
    return {
      ...surface,
      logView: { stream: selection.stream, viewRevision: surface.logView?.viewRevision ?? 0 },
    };
  return surface;
}
/** Scope inclusion, independent of origin and availability. Callers still validate source coordinates. */
export function containsObservedTarget(container: EvidenceRef, target: EvidenceRef): boolean {
  if (
    container.projectId !== target.projectId ||
    container.dataRevision !== target.dataRevision ||
    !sameResource(container.resource, target.resource)
  )
    return false;
  if (container.schemaVersion === 8 && target.schemaVersion === 8)
    return Value.Equal(container.resource.saved, target.resource.saved);
  if (container.schemaVersion === 7 && target.schemaVersion === 7)
    return (
      container.sourceRevision === target.sourceRevision &&
      samePipelineSelector(container.selection.subject, target.selection.subject)
    );
  if (container.schemaVersion === 6 && target.schemaVersion === 6)
    return notebookSelectionContains(container.selection, target.selection);
  if (container.schemaVersion === 5 && target.schemaVersion === 5) {
    const a = container.selection,
      b = target.selection;
    return (
      a.profile === b.profile &&
      a.pageIndex === b.pageIndex &&
      a.modelHash === b.modelHash &&
      b.region[0] >= a.region[0] &&
      b.region[1] >= a.region[1] &&
      b.region[2] <= a.region[2] &&
      b.region[3] <= a.region[3]
    );
  }
  if (container.schemaVersion === 4 && target.schemaVersion === 4) {
    const outer = container.selection,
      inner = target.selection;
    return outer.kind === 'run-group'
      ? inner.kind === 'run-group' && outer.taskId === inner.taskId
      : inner.kind === 'run-dependency' &&
          outer.fromTaskId === inner.fromTaskId &&
          outer.toTaskId === inner.toTaskId;
  }
  if (container.schemaVersion !== 3 || target.schemaVersion !== 3) return false;
  const outer = container.selection,
    inner = target.selection;
  if (!outer) return container.resource.kind === 'run';
  if (outer.kind === 'run-task')
    return (
      inner?.kind === 'run-task' &&
      inner.instanceId === outer.instanceId &&
      inner.attempt === outer.attempt
    );
  const before = (a: { line: number; column: number }, b: { line: number; column: number }) =>
    a.line < b.line || (a.line === b.line && a.column <= b.column);
  return (
    outer.kind === 'log-text' &&
    inner?.kind === 'log-text' &&
    outer.stream === inner.stream &&
    before(outer.start, inner.start) &&
    before(inner.end, outer.end)
  );
}
