import {
  validatePdfSelection,
  containsObservedTarget,
  type PdfTarget,
  type Selection,
  type SurfaceData,
  type Surface,
} from '@gobble/contracts';
import type { Rect } from '@gobble/contracts/pdf-decoder';
import { AppProblem } from '../problem';

/** Addresses only returned pixels. Source-preview is the decoded page, never off-page authority. */
export function pdfObservation(
  view: {
    surface: Surface;
    data: SurfaceData;
    pdfViewport?: { generation: number; region: Rect | null } | undefined;
  },
  selection: Selection | undefined,
  scope: 'view' | 'source-preview',
) {
  if (
    view.surface.view !== 'pdf' ||
    view.surface.resource.kind !== 'file' ||
    view.data.kind !== 'file' ||
    view.data.value.content.kind !== 'pdf'
  )
    throw new AppProblem('invalid_request', 'A PDF view is required.');
  const content = view.data.value.content,
    { page } = content;
  if (!view.pdfViewport?.region)
    throw new AppProblem(
      'stale_revision',
      'The visible PDF region has not been confirmed. Reveal the page and observe again.',
    );
  if (selection && selection.kind !== 'pdf')
    throw new AppProblem('invalid_request', 'PDF observations require page or region coordinates.');
  const region = scope === 'view' ? view.pdfViewport.region : page.model.viewBox;
  const outer: PdfTarget = {
    schemaVersion: 5,
    projectId: view.surface.projectId,
    resource: view.surface.resource,
    origin: { surfaceId: view.surface.surfaceId },
    dataRevision: view.data.value.revision,
    selection: {
      kind: 'pdf',
      coordinateSpace: 'pdf-user-space',
      scope: JSON.stringify(region) === JSON.stringify(page.model.viewBox) ? 'page' : 'region',
      profile: page.model.profile,
      pageIndex: page.model.pageIndex,
      modelHash: page.modelHash,
      region,
    },
  };
  const evidence = selection ? { ...outer, selection } : outer;
  validatePdfSelection(evidence.selection, content);
  if (!containsObservedTarget(outer, evidence))
    throw new AppProblem(
      'invalid_request',
      'This PDF target is outside the observed region. Reveal it first, or use source-preview for the current page.',
    );
  return {
    evidence,
    content: {
      kind: 'pdf' as const,
      pageCount: content.pageCount,
      pageIndex: page.model.pageIndex,
      coordinateSpace: 'pdf-user-space',
      viewBox: page.model.viewBox,
      visibleRegion: view.pdfViewport.region,
      region: evidence.selection.region,
      viewport: page.viewport,
      profile: page.model.profile,
      modelHash: page.modelHash,
      textCapability: 'unsupported',
      scope,
    },
  };
}
