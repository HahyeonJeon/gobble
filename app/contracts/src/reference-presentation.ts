import { Type, type Static } from '@sinclair/typebox';
import { closed, RequestIdSchema } from './identity';
import { SharedReferenceIdSchema } from './shared-context';
import {
  ReferencePresentationSchema,
  type ReferencePresentation,
  type ViewLink,
} from './tabular-view';
import type { EvidenceRef } from './reference-target';
import type { Surface } from './surface';
import { ContractValidationError } from './validation-error';
import {
  filterTable,
  projectScatter,
  scatterColumns,
  scatterViewport,
  validateScatterSpec,
  validateViewport,
  type TableContent,
} from './tabular-projection';

/** Effective, temporary display state. Never written into the durable Workspace document. */
export const ReferenceViewSchema = Type.Object(
  {
    requestId: RequestIdSchema,
    referenceId: SharedReferenceIdSchema,
    presentation: ReferencePresentationSchema,
    adjustments: Type.Array(Type.Union([Type.Literal('filter'), Type.Literal('viewport')]), {
      maxItems: 2,
      uniqueItems: true,
    }),
  },
  closed,
);
export type ReferenceView = Static<typeof ReferenceViewSchema>;

/** Structural relationships independent of source availability; safe for archived references. */
export function validateReferencePresentation(
  evidence: EvidenceRef,
  presentation?: ReferencePresentation,
): void {
  if (!presentation) return;
  validateViewport(presentation.viewport);
  if (
    evidence.resource.kind !== 'file' ||
    evidence.selection?.kind !== 'table' ||
    presentation.spec.xColumnId === presentation.spec.yColumnId ||
    scatterColumns(presentation.spec).some(
      (id) =>
        !evidence.selection ||
        evidence.selection.kind !== 'table' ||
        !evidence.selection.columns.includes(id),
    )
  )
    throw new ContractValidationError(
      'Plot context requires an exact table target including its axis and label columns.',
    );
}
export function validatePresentationSource(
  content: TableContent,
  presentation: ReferencePresentation,
): void {
  validateScatterSpec(content, presentation.spec);
  filterTable(content, presentation.filter);
  validateViewport(presentation.viewport);
}
export function captureScatterPresentation(
  surface: Surface,
  link: ViewLink | undefined,
  content: TableContent,
): ReferencePresentation | undefined {
  if (surface.view !== 'scatter') return undefined;
  const filter = link?.filter ?? { kind: 'all' as const };
  const points = projectScatter(content, surface.scatter.spec, filter).points;
  return copyPresentation({
    schemaVersion: 1,
    kind: 'scatter',
    spec: surface.scatter.spec,
    filter,
    viewport: surface.scatter.viewport ?? scatterViewport(points),
    revision: {
      spec: surface.scatter.specRevision,
      view: surface.scatter.viewRevision,
      filter: link?.filterRevision ?? 0,
    },
  });
}

/** Expand only temporary presentation when a reference is outside the captured filter/range. */
export function revealPlotTarget(
  content: TableContent,
  presentation: ReferencePresentation,
  rows: string[],
): Pick<ReferenceView, 'presentation' | 'adjustments'> {
  validatePresentationSource(content, presentation);
  const next = copyPresentation(presentation);
  const adjustments: ReferenceView['adjustments'] = [];
  const visible = new Set(filterTable(content, next.filter).map((row) => row.key));
  if (rows.some((id) => !visible.has(id))) {
    next.filter = { kind: 'all' };
    adjustments.push('filter');
  }
  const points = projectScatter(content, next.spec, next.filter).points;
  const targeted = points.filter((point) => rows.includes(point.rowKey));
  if (
    targeted.some(
      (point) =>
        point.x < next.viewport.x[0] ||
        point.x > next.viewport.x[1] ||
        point.y < next.viewport.y[0] ||
        point.y > next.viewport.y[1],
    )
  ) {
    const bounds = scatterViewport(targeted);
    next.viewport = {
      x: [Math.min(next.viewport.x[0], bounds.x[0]), Math.max(next.viewport.x[1], bounds.x[1])],
      y: [Math.min(next.viewport.y[0], bounds.y[0]), Math.max(next.viewport.y[1], bounds.y[1])],
    };
    adjustments.push('viewport');
  }
  return { presentation: next, adjustments };
}

/** A derived override; never mutate the user's base Surface or link. */
export function effectiveReferencePresentation(
  surface: Surface,
  link: ViewLink | undefined,
  reference?: ReferenceView,
): { surface: Surface; link: ViewLink | undefined } {
  if (!reference) return { surface, link };
  const p = reference.presentation;
  return {
    surface:
      surface.view === 'scatter'
        ? {
            ...surface,
            scatter: {
              spec: p.spec,
              specRevision: p.revision.spec,
              viewRevision: p.revision.view,
              viewport: p.viewport,
            },
          }
        : surface,
    link: link ? { ...link, filter: p.filter, filterRevision: p.revision.filter } : undefined,
  };
}

function copyPresentation(p: ReferencePresentation): ReferencePresentation {
  return {
    ...p,
    spec: { ...p.spec },
    filter: { ...p.filter },
    viewport: { x: [...p.viewport.x], y: [...p.viewport.y] },
    revision: { ...p.revision },
  };
}
