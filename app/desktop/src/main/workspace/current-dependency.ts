import { randomUUID } from 'node:crypto';
import {
  defaultDependencyNavigation,
  sameResource,
  type SharedReference,
  type WorkspaceDocument,
} from '@gobble/contracts';
import type { WorkspaceResources } from './service';
import { AppProblem } from '../problem';
import { transition } from './model';

/** Explicit current-source search. It never rebinds or copies the historical pointer into selection. */
export async function findCurrentDependency(
  current: WorkspaceDocument,
  reference: SharedReference,
  resources: Pick<WorkspaceResources, 'describe'>,
) {
  if (reference.retracted || reference.evidence.schemaVersion !== 4)
    throw new AppProblem('unsupported', 'This reference has no current dependency target to find.');
  const { resource, selection } = reference.evidence;
  const existing = current.workspace.surfaces.find((surface) =>
    sameResource(surface.resource, resource),
  );
  const metadata = existing
    ? undefined
    : {
        ...(await resources.describe(current.workspace.projectId, resource)),
        surfaceId: 'srf_' + randomUUID(),
      };
  const document = transition(
    current,
    { kind: 'open', resource, pane: current.activePane, duplicate: false },
    metadata,
  );
  const surface = document.workspace.surfaces.find((item) =>
    sameResource(item.resource, resource),
  )!;
  if (surface.view !== 'run') throw new AppProblem('unsupported', 'This source has no Run view.');
  surface.runView = {
    query: '',
    status: null,
    ...surface.runView,
    mode: 'dependencies',
    viewRevision: (surface.runView?.viewRevision ?? 0) + 1,
    dependencies: {
      ...(surface.runView?.dependencies ?? defaultDependencyNavigation()),
      representation: 'list',
      query: (selection.kind === 'run-group' ? selection.taskId : selection.fromTaskId).slice(
        0,
        200,
      ),
    },
  };
  document.selections = document.selections.filter((item) => item.surfaceId !== surface.surfaceId);
  delete document.referenceReveal;
  return { document, surfaceId: surface.surfaceId };
}
