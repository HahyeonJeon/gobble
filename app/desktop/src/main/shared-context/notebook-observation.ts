import {
  notebookPart,
  notebookSelectionContains,
  validateNotebookViewport,
  type NotebookTarget,
  type NotebookViewport,
  type Selection,
  type Surface,
  type SurfaceData,
} from '@gobble/contracts';
import { AppProblem } from '../problem';
export type NotebookObservationView = {
  surface: Surface;
  data: SurfaceData;
  notebookViewport?: { generation: number; viewport: NotebookViewport } | undefined;
};
export type NotebookObservation = {
  targets: NotebookTarget[];
  textParts: Array<{ evidence: NotebookTarget; text: string }>;
  imagesAvailable: NotebookTarget[];
  image?: NotebookTarget;
  scope: 'view' | 'source-preview';
};
export function notebookObservation(
  view: NotebookObservationView,
  selection: Selection | undefined,
  scope: 'view' | 'source-preview',
): NotebookObservation {
  if (
    view.surface.view !== 'notebook' ||
    view.surface.resource.kind !== 'file' ||
    view.data.kind !== 'file' ||
    view.data.value.content.kind !== 'notebook'
  )
    throw new AppProblem('invalid_request', 'A Notebook view is required.');
  if (view.data.value.content.reference)
    throw new AppProblem('unsupported', 'Return to the Notebook reader before observing it.');
  if (!view.notebookViewport)
    throw new AppProblem(
      'stale_revision',
      'The visible Notebook content has not been confirmed. Observe again after it is displayed.',
    );
  if (selection && selection.kind !== 'notebook')
    throw new AppProblem(
      'invalid_request',
      'Notebook observations require a cell, part and text or image selector.',
    );
  if (scope === 'source-preview' && !selection)
    throw new AppProblem(
      'invalid_request',
      'Source-preview requires one exact Notebook part selector.',
    );
  const document = view.data.value.content.document,
    viewport = view.notebookViewport.viewport;
  validateNotebookViewport(document, viewport);
  if (selection) {
    notebookPart(document, selection);
    if (
      scope === 'view' &&
      !viewport.parts.some((part) => notebookSelectionContains(part, selection))
    )
      throw new AppProblem(
        'invalid_request',
        'This Notebook range is not fully visible. Reveal it first, or request source-preview explicitly.',
      );
  }
  const result: NotebookObservation = { targets: [], textParts: [], imagesAvailable: [], scope };
  for (const part of selection ? [selection] : viewport.parts) {
    const evidence: NotebookTarget = {
      schemaVersion: 6,
      projectId: view.surface.projectId,
      resource: view.surface.resource,
      origin: { surfaceId: view.surface.surfaceId },
      dataRevision: view.data.value.revision,
      selection: part,
    };
    const content = notebookPart(document, part);
    if (content.kind === 'text' && part.selector.kind === 'text') {
      const text = content.text.slice(part.selector.start, part.selector.end);
      if (Buffer.byteLength(text) > 48 * 1024)
        throw new AppProblem('unsupported', 'Request a smaller Notebook text range.');
      result.textParts.push({ evidence, text });
      result.targets.push(evidence);
    } else if (selection) {
      result.image = evidence;
      result.targets.push(evidence);
    } else result.imagesAvailable.push(evidence);
  }
  return result;
}
