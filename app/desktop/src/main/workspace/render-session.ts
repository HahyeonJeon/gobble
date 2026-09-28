import { assertRegion } from '@gobble/contracts/pdf-geometry';
import type { Rect } from '@gobble/contracts/pdf-decoder';
import {
  validateNotebookViewport,
  type NotebookViewport,
  runPresentationState,
} from '@gobble/contracts';
import { dependencyRevision } from '../evidence/dependency-capture';
import { randomUUID } from 'node:crypto';
import {
  presentationRevision,
  observedReferenceSurface,
  type ObservedReferenceView,
  effectiveReferencePresentation,
  type ReferenceView,
  samePresentation,
  filterTable,
  validateTableSettings,
  sameResource,
  type ViewLink,
  type RenderAcknowledgment,
  type Surface,
  type SurfaceData,
  type SurfaceLoad,
} from '@gobble/contracts';
import { AppProblem } from '../problem';
import { dataRevision, observedDataRevision } from './selection';

export type SurfaceRequest = {
  projectId: string;
  surfaceId: string;
  rendererSessionId: string;
  refresh?: boolean;
};
type Ticket = {
  readonly request: SurfaceRequest;
  readonly surface: Surface;
  readonly generation: number;
  readonly link?: ViewLink;
  readonly referenceView?: ReferenceView;
  readonly observedReferenceView?: ObservedReferenceView;
  readonly fingerprint: string;
};
type Lease = {
  ticket: Ticket;
  acknowledgment?: RenderAcknowledgment;
  data?: SurfaceData;
  ready: boolean;
  pdfViewport?: { generation: number; region: Rect | null };
  notebookViewport?: { generation: number; viewport: NotebookViewport };
  observedBytes?: number;
  invalidated?: boolean;
  refreshProblem?: string;
};

function matches(a: RenderAcknowledgment, b: RenderAcknowledgment): boolean {
  return (
    a.schemaVersion === b.schemaVersion &&
    a.projectId === b.projectId &&
    a.surfaceId === b.surfaceId &&
    a.rendererSessionId === b.rendererSessionId &&
    a.requestId === b.requestId &&
    a.generation === b.generation &&
    a.dataRevision === b.dataRevision &&
    a.referenceViewRequestId === b.referenceViewRequestId &&
    samePresentation(a.presentation, b.presentation)
  );
}

const fingerprint = (
  surface: Surface,
  link?: ViewLink,
  referenceView?: ReferenceView,
  observedReferenceView?: ObservedReferenceView,
): string =>
  JSON.stringify([
    referenceView ?? null,
    observedReferenceView ?? null,
    surface.resource,
    surface.view,
    surface.view === 'scatter' ? surface.scatter : null,
    surface.view === 'table' ? (surface.table ?? null) : null,
    surface.view === 'run' ? (runPresentationState(surface.runView) ?? null) : null,
    surface.view === 'log' ? (surface.logView ?? null) : null,
    surface.view === 'pdf' ? (surface.pdf ?? null) : null,
    surface.view === 'notebook' ? (surface.notebook ?? null) : null,
    surface.view === 'report' ? (surface.reportModuleId ?? null) : null,
    link ? [link.linkId, link.dataRevision, link.filterRevision, link.filter] : null,
  ]);

/** Ephemeral renderer observations only. The controller owns Project authorization and durable state. */
export class RenderSession {
  private sessionId = 'rnd_' + randomUUID();
  private generation = 0;
  private viewportGeneration = 0;
  private inFlight = new Set<Ticket>();
  private visible = new Set<string | null>();
  private leases = new Map<string, Lease>();
  get id(): string {
    return this.sessionId;
  }

  disconnect(): void {
    this.sessionId = 'rnd_' + randomUUID();
    this.leases.clear();
    this.visible.clear();
  }
  clear(): void {
    this.leases.clear();
  }
  forget(surfaceId: string): void {
    this.leases.delete(surfaceId);
  }
  reveal(surfaceIds: (string | null)[]): void {
    this.visible = new Set(surfaceIds);
    for (const id of this.leases.keys()) if (!this.visible.has(id)) this.leases.delete(id);
  }
  reconcile(
    surfaces: Surface[],
    links: ViewLink[],
    referenceFor: (id: string) => ReferenceView | undefined = () => undefined,
    observedFor: (id: string) => ObservedReferenceView | undefined = () => undefined,
  ): void {
    for (const [id, lease] of this.leases) {
      const surface = surfaces.find((item) => item.surfaceId === id);
      const link = links.find((item) => item.surfaceIds.includes(id));
      if (
        !surface ||
        lease.ticket.fingerprint !== fingerprint(surface, link, referenceFor(id), observedFor(id))
      ) {
        if (
          surface &&
          lease.data &&
          lease.data.kind !== 'file' &&
          sameResource(surface.resource, lease.ticket.surface.resource)
        ) {
          lease.ready = false;
          lease.invalidated = true;
          delete lease.acknowledgment;
        } else this.leases.delete(id);
      }
    }
  }
  /** Reserve before asynchronous I/O; callers release the returned ticket in finally (release is idempotent). */
  begin(
    request: SurfaceRequest,
    surface: Surface,
    link?: ViewLink,
    referenceView?: ReferenceView,
    observedReferenceView?: ObservedReferenceView,
  ): Ticket {
    if (surface.view === 'scatter')
      throw new AppProblem('unsupported', 'Chart views have been retired. Open the source table.');
    if (request.rendererSessionId !== this.sessionId)
      throw new AppProblem('stale_revision', 'This window session is no longer current.');
    if (!this.visible.has(request.surfaceId))
      throw new AppProblem('stale_revision', 'Reveal this view before loading it.');
    if (this.inFlight.size >= 4)
      throw new AppProblem(
        'runtime_unavailable',
        'Other views are still loading. Try again shortly.',
      );
    const ticket: Ticket = {
      request,
      surface: structuredClone(surface),
      generation: ++this.generation,
      fingerprint: fingerprint(surface, link, referenceView, observedReferenceView),
      ...(observedReferenceView
        ? { observedReferenceView: structuredClone(observedReferenceView) }
        : {}),
      ...(referenceView ? { referenceView: structuredClone(referenceView) } : {}),
      ...(link ? { link: structuredClone(link) } : {}),
    };
    const prior = this.leases.get(request.surfaceId);
    const retained =
      prior?.data &&
      prior.data.kind !== 'file' &&
      sameResource(prior.ticket.surface.resource, surface.resource)
        ? prior.data
        : undefined;
    this.leases.set(request.surfaceId, {
      ticket,
      ready: false,
      ...(retained
        ? {
            data: retained,
            observedBytes: prior?.observedBytes ?? 0,
            ...(prior?.refreshProblem ? { refreshProblem: prior.refreshProblem } : {}),
          }
        : {}),
    });
    this.inFlight.add(ticket);
    return ticket;
  }
  release(ticket: Ticket): void {
    this.inFlight.delete(ticket);
  }
  /** Presentation changes reuse one retained observation; only Refresh reads a newer source. */
  retainedData(ticket: Ticket): SurfaceData | undefined {
    const lease = this.leases.get(ticket.request.surfaceId);
    return lease?.ticket === ticket && lease.data?.kind !== 'file' && lease.data
      ? structuredClone(lease.data)
      : undefined;
  }
  retainedProblem(ticket: Ticket): string | undefined {
    const lease = this.leases.get(ticket.request.surfaceId);
    return lease?.ticket === ticket ? lease.refreshProblem : undefined;
  }
  complete(ticket: Ticket, data: SurfaceData, refreshProblem?: string): SurfaceLoad {
    const input = ticket.request;
    const lease = this.leases.get(input.surfaceId);
    if (
      input.rendererSessionId !== this.sessionId ||
      lease?.ticket !== ticket ||
      lease.invalidated ||
      !this.visible.has(input.surfaceId)
    )
      throw new AppProblem('stale_revision', 'A newer view replaced this loading request.');
    if (refreshProblem) lease.refreshProblem = refreshProblem;
    else delete lease.refreshProblem;
    let observedBytes = data.kind === 'file' ? 0 : Buffer.byteLength(JSON.stringify(data));
    // Optional dependency context must not displace an otherwise admissible Tasks observation.
    if (observedBytes > 1024 * 1024 && data.kind === 'run' && data.dependencies) {
      data = {
        kind: 'run',
        value: data.value,
        dependencyProblem:
          'The dependency preview exceeds the view memory limit. Tasks remain available.',
      };
      observedBytes = Buffer.byteLength(JSON.stringify(data));
    }
    const retainedBytes = [...this.leases.values()].reduce(
      (sum, item) => sum + (item === lease ? 0 : (item.observedBytes ?? 0)),
      0,
    );
    if (observedBytes > 1024 * 1024 || retainedBytes + observedBytes > 2 * 1024 * 1024)
      throw new AppProblem('unsupported', 'This observation exceeds the view memory limit.');
    const view = data.kind === 'file' ? data.value.content.kind : data.kind;
    if (view !== ticket.surface.view)
      throw new AppProblem(
        'stale_revision',
        'The file format changed. Close and reopen this view.',
      );
    if (
      ticket.link &&
      (data.kind !== 'file' ||
        data.value.content.kind !== 'table' ||
        dataRevision(data) !== ticket.link.dataRevision)
    )
      throw new AppProblem(
        'stale_revision',
        'The linked source changed. Refresh the linked views together.',
      );
    const effective = effectiveReferencePresentation(
      observedReferenceSurface(ticket.surface, ticket.observedReferenceView),
      ticket.link,
      ticket.referenceView,
    );
    if (effective.link && data.kind === 'file' && data.value.content.kind === 'table')
      filterTable(data.value.content, effective.link.filter);
    if (
      effective.surface.view === 'table' &&
      effective.surface.table &&
      data.kind === 'file' &&
      data.value.content.kind === 'table'
    )
      validateTableSettings(data.value.content, effective.surface.table);
    const acknowledgment: RenderAcknowledgment = {
      schemaVersion: 3,
      projectId: input.projectId,
      surfaceId: input.surfaceId,
      rendererSessionId: input.rendererSessionId,
      ...(ticket.referenceView
        ? { referenceViewRequestId: ticket.referenceView.requestId }
        : ticket.observedReferenceView
          ? { referenceViewRequestId: ticket.observedReferenceView.requestId }
          : {}),
      requestId: 'req_' + randomUUID(),
      generation: ticket.generation,
      dataRevision: dataRevision(data),
      presentation: presentationRevision(effective.surface, effective.link),
    };
    lease.acknowledgment = acknowledgment;
    lease.data = structuredClone(data);
    lease.observedBytes = observedBytes;
    return {
      acknowledgment,
      data: structuredClone(data),
      ...(data.kind === 'file' ? {} : { observedRevision: observedDataRevision(data) }),
      ...(data.kind === 'run' && data.dependencies
        ? { dependencyRevision: dependencyRevision(data.dependencies) }
        : {}),
      ...(ticket.referenceView ? { referenceView: ticket.referenceView } : {}),
      ...(ticket.observedReferenceView
        ? { observedReferenceView: ticket.observedReferenceView }
        : {}),
    };
  }
  acknowledge(input: RenderAcknowledgment): { accepted: true } {
    const lease = this.leases.get(input.surfaceId);
    if (
      !lease?.acknowledgment ||
      !matches(lease.acknowledgment, input) ||
      input.rendererSessionId !== this.sessionId ||
      !this.visible.has(input.surfaceId)
    )
      throw new AppProblem('stale_revision', 'This rendered view is no longer current.');
    lease.ready = true;
    return { accepted: true };
  }
  invalidate(input: RenderAcknowledgment): { accepted: true } {
    const lease = this.leases.get(input.surfaceId);
    if (!lease?.acknowledgment || !matches(lease.acknowledgment, input))
      throw new AppProblem('stale_revision', 'This render has already been replaced.');
    this.leases.delete(input.surfaceId);
    return { accepted: true };
  }
  presentation(surfaceId: string): 'ready' | 'loading' | 'hidden' {
    return !this.visible.has(surfaceId)
      ? 'hidden'
      : this.leases.get(surfaceId)?.ready
        ? 'ready'
        : 'loading';
  }
  setPdfViewport(input: import('@gobble/contracts').PdfViewportInput) {
    this.assertAcknowledgment(input.acknowledgment);
    const lease = this.leases.get(input.acknowledgment.surfaceId)!;
    if (lease.data?.kind !== 'file' || lease.data.value.content.kind !== 'pdf')
      throw new AppProblem('invalid_request', 'A PDF page is required.');
    if (input.region) assertRegion(input.region, lease.data.value.content.page.model.viewBox);
    if (JSON.stringify(lease.pdfViewport?.region) !== JSON.stringify(input.region))
      lease.pdfViewport = { generation: ++this.viewportGeneration, region: input.region };
    return { accepted: true as const };
  }
  setNotebookViewport(input: import('@gobble/contracts').NotebookViewportInput) {
    this.assertAcknowledgment(input.acknowledgment);
    const lease = this.leases.get(input.acknowledgment.surfaceId)!;
    if (
      lease.data?.kind !== 'file' ||
      lease.data.value.content.kind !== 'notebook' ||
      lease.ticket.observedReferenceView
    )
      throw new AppProblem('invalid_request', 'A Notebook reader is required.');
    validateNotebookViewport(lease.data.value.content.document, input.viewport);
    if (JSON.stringify(lease.notebookViewport?.viewport) !== JSON.stringify(input.viewport))
      lease.notebookViewport = {
        generation: ++this.viewportGeneration,
        viewport: structuredClone(input.viewport),
      };
    return { accepted: true as const };
  }
  assertNotebookViewport(ack: RenderAcknowledgment, generation: number | undefined) {
    this.assertAcknowledgment(ack);
    if (
      generation === undefined ||
      this.leases.get(ack.surfaceId)?.notebookViewport?.generation !== generation
    )
      throw new AppProblem(
        'stale_revision',
        'The visible Notebook content changed. Observe it again.',
      );
  }
  assertPdfViewport(ack: RenderAcknowledgment, generation: number | undefined) {
    this.assertAcknowledgment(ack);
    if (
      generation === undefined ||
      this.leases.get(ack.surfaceId)?.pdfViewport?.generation !== generation
    )
      throw new AppProblem('stale_revision', 'The visible PDF region changed. Observe it again.');
  }
  snapshot(surfaceId: string) {
    const data = this.selectableData(surfaceId);
    const lease = this.leases.get(surfaceId)!;
    return {
      acknowledgment: structuredClone(lease.acknowledgment!),
      notebookViewport: lease.notebookViewport
        ? structuredClone(lease.notebookViewport)
        : undefined,
      pdfViewport: lease.pdfViewport ? structuredClone(lease.pdfViewport) : undefined,
      data,
      ...effectiveReferencePresentation(
        observedReferenceSurface(lease.ticket.surface, lease.ticket.observedReferenceView),
        lease.ticket.link,
        lease.ticket.referenceView,
      ),
      ...(lease.ticket.observedReferenceView
        ? { observedReferenceView: structuredClone(lease.ticket.observedReferenceView) }
        : {}),
      ...(lease.ticket.referenceView
        ? { referenceView: structuredClone(lease.ticket.referenceView) }
        : {}),
    };
  }
  assertAcknowledgment(acknowledgment: RenderAcknowledgment): void {
    // Validate identity without cloning a multi-megabyte PDF raster on every scroll report.
    const current = this.readyLease(acknowledgment.surfaceId);
    if (!current.acknowledgment || !matches(current.acknowledgment, acknowledgment))
      throw new AppProblem(
        'stale_revision',
        'The observed view changed. Observe its current version again.',
      );
  }
  private readyLease(surfaceId: string): Lease & { data: SurfaceData } {
    const lease = this.leases.get(surfaceId);
    if (!lease?.ready || !lease.data || !this.visible.has(surfaceId))
      throw new AppProblem(
        'stale_revision',
        'Wait for this view to finish loading before selecting its content.',
      );
    return lease as Lease & { data: SurfaceData };
  }
  selectableData(surfaceId: string): SurfaceData {
    return structuredClone(this.readyLease(surfaceId).data);
  }
}
