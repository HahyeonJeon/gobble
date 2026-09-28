import { describe, expect, it } from 'vitest';
import { mkdtemp, mkdir, readFile, writeFile, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import {
  PdfTargetSchema,
  parse,
  validatePdfImage,
  validatePdfSelection,
  parseWorkspaceDocument,
  readStoredWorkspace,
  type PdfImageRepresentation,
  type PdfContent,
  type PdfTarget,
  SHARED_TOOLSET,
} from '@gobble/contracts';
import { PROFILE } from '@gobble/contracts/pdf-decoder';
import { emptyWorkspace } from '../src/main/workspace/model';
import { WorkspaceStorage } from '../src/main/workspace/storage';
import { checkSelection } from '../src/main/workspace/selection';
import { requireEvidence } from '../src/main/shared-context/references';
import { pdfObservation } from '../src/main/shared-context/pdf-observation';
import { ObservedReads } from '../src/main/shared-context/observed';
import { SharedToolRequestV7Schema, WorkspaceDocumentV11Schema } from '@gobble/contracts';
import { RenderSession } from '../src/main/workspace/render-session';

const hash = 'sha256:' + 'a'.repeat(64),
  other = 'sha256:' + 'b'.repeat(64);
const target: PdfTarget = {
  schemaVersion: 5,
  projectId: 'prj_pdf',
  resource: { kind: 'file', resourceId: 'res_pdf' },
  origin: { surfaceId: 'srf_pdf' },
  dataRevision: hash,
  selection: {
    kind: 'pdf',
    coordinateSpace: 'pdf-user-space',
    scope: 'region',
    profile: PROFILE,
    pageIndex: 0,
    modelHash: hash,
    region: [100, 120, 220, 180],
  },
};
const content: PdfContent = {
  kind: 'pdf',
  pageCount: 4,
  page: {
    kind: 'page',
    renditionId: 'rendition',
    modelHash: hash,
    model: {
      revision: hash,
      profile: PROFILE,
      pageIndex: 0,
      viewBox: [20, 40, 580, 800],
      userUnit: 1,
      intrinsicRotation: 0,
      text: { capability: 'diagnostic-only', items: [], truncated: false },
    },
    viewport: [1, 0, 0, -1, -20, 800],
    raster: { width: 560, height: 760, png: 'AA==' },
    timings: { modelMs: 1, renderMs: 1 },
  },
};
const view: PdfImageRepresentation = {
  kind: 'image',
  width: 120,
  height: 60,
  originalWidth: 560,
  originalHeight: 760,
  crop: { x: 80, y: 620, width: 120, height: 60 },
  pdf: {
    profile: PROFILE,
    pageIndex: 0,
    modelHash: hash,
    viewBox: [20, 40, 580, 800],
    userUnit: 1,
    intrinsicRotation: 0,
    viewport: [1, 0, 0, -1, -20, 800],
  },
};

describe('PDF evidence identity and storage boundaries', () => {
  it('invalidates old render receipts on PDF navigation, hiding, replacement and reconnect', () => {
    const rendering = new RenderSession();
    const surface = {
      projectId: 'prj_pdf',
      surfaceId: 'srf_pdf',
      resource: target.resource,
      view: 'pdf' as const,
      pinned: false,
      openedBy: { kind: 'user' as const },
    };
    const data = {
      kind: 'file' as const,
      value: {
        projectId: 'prj_pdf',
        resourceId: 'res_pdf',
        revision: hash,
        name: 'report.pdf',
        size: 10,
        content,
      },
    };
    const load = () => {
      rendering.reveal(['srf_pdf']);
      const ticket = rendering.begin(
        { projectId: 'prj_pdf', surfaceId: 'srf_pdf', rendererSessionId: rendering.id },
        surface,
      );
      const result = rendering.complete(ticket, data);
      rendering.release(ticket);
      rendering.acknowledge(result.acknowledgment);
      return result.acknowledgment;
    };
    for (const navigation of [
      { pageIndex: 1, scale: 1 as const, rotation: 0 as const },
      { pageIndex: 0, scale: 1.25 as const, rotation: 0 as const },
      { pageIndex: 0, scale: 1 as const, rotation: 90 as const },
    ]) {
      const ack = load();
      expect(() => rendering.assertAcknowledgment(ack)).not.toThrow();
      rendering.reconcile([{ ...surface, pdf: navigation }], []);
      expect(() => rendering.assertAcknowledgment(ack)).toThrow();
    }
    const hidden = load();
    rendering.reveal([]);
    expect(() => rendering.assertAcknowledgment(hidden)).toThrow();
    const replaced = load();
    load();
    expect(() => rendering.assertAcknowledgment(replaced)).toThrow();
    const disconnected = load();
    rendering.disconnect();
    expect(() => rendering.assertAcknowledgment(disconnected)).toThrow();
  });
  it('binds selection to source revision, page model and contained PDF coordinates', () => {
    expect(parse(PdfTargetSchema, target)).toEqual(target);
    expect(() => validatePdfSelection(target.selection, content)).not.toThrow();
    const data = {
      kind: 'file' as const,
      value: {
        projectId: 'prj_pdf',
        resourceId: 'res_pdf',
        revision: hash,
        name: 'report.pdf',
        size: 10,
        content,
      },
    };
    expect(() => checkSelection(target, data)).not.toThrow();
    expect(() => checkSelection({ ...target, dataRevision: other }, data)).toThrow();
    for (const selection of [
      { ...target.selection, pageIndex: 1 },
      { ...target.selection, modelHash: other },
      { ...target.selection, scope: 'page' as const },
      { ...target.selection, region: [0, 120, 220, 180] as [number, number, number, number] },
      { ...target.selection, region: [100, 120, 100, 180] as [number, number, number, number] },
    ])
      expect(() => validatePdfSelection(selection, content)).toThrow();
    expect(() =>
      parse(PdfTargetSchema, {
        ...target,
        selection: { ...target.selection, text: 'guessed text' },
      }),
    ).toThrow();
  });
  it('rejects capture geometry tampering, resampling and model mismatch', () => {
    expect(() => validatePdfImage(target, view)).not.toThrow();
    for (const invalid of [
      { ...view, width: 60 },
      { ...view, crop: { ...view.crop, x: 81 } },
      { ...view, pdf: { ...view.pdf, modelHash: other } },
      {
        ...view,
        pdf: {
          ...view.pdf,
          viewport: [0, 0, 0, 0, 0, 0] as [number, number, number, number, number, number],
        },
      },
      { ...view, originalWidth: 4_000_000 },
    ])
      expect(() => validatePdfImage(target, invalid)).toThrow();
  });
  it('requires a matching originating Surface and frozen PDF capture in durable drafts', () => {
    const doc = emptyWorkspace('prj_pdf');
    doc.workspace.surfaces.push({
      openedBy: { kind: 'user' },
      projectId: 'prj_pdf',
      surfaceId: 'srf_pdf',
      resource: target.resource,
      view: 'pdf',
      pinned: false,
    });
    doc.workspace.layout.primary = { tabs: ['srf_pdf'], activeSurfaceId: 'srf_pdf' };
    doc.titles = [{ surfaceId: 'srf_pdf', title: 'report.pdf' }];
    expect(() => requireEvidence(doc, target, 'srf_pdf')).not.toThrow();
    expect(() =>
      requireEvidence(doc, { ...target, origin: { surfaceId: 'srf_other' } }, 'srf_pdf'),
    ).toThrow();
    const attachment = {
      attachmentId: 'att_pdf',
      label: 'report.pdf',
      createdAt: 1,
      evidence: target,
    };
    doc.chat.attachments = [attachment];
    expect(() => parseWorkspaceDocument(doc)).toThrow();
    doc.chat.attachments = [
      {
        ...attachment,
        capture: {
          kind: 'pdf',
          capturedAt: 1,
          asset: { hash, byteLength: 100, mediaType: 'image/png' },
          representation: view,
        },
      },
    ];
    expect(() => parseWorkspaceDocument(doc)).not.toThrow();
    doc.chat.attachments[0]!.capture!.representation = { ...view, width: 1 };
    expect(() => parseWorkspaceDocument(doc)).toThrow();
    expect(SHARED_TOOLSET).toBe('shared-views-v15');
  });
  it('freezes v10, preserves exact migration backup and accepts PDF from v11 onwards', async () => {
    const base = await mkdtemp(join(tmpdir(), 'gobble-pdf-migration-'));
    try {
      await mkdir(join(base, 'projects'));
      const legacy = { ...emptyWorkspace('prj_pdf'), schemaVersion: 10 };
      const raw = JSON.stringify(legacy, null, 2) + '\n',
        path = join(base, 'projects/prj_pdf.json');
      await writeFile(path, raw);
      const storage = new WorkspaceStorage(base).project('prj_pdf');
      const next = (await storage.read())!;
      expect(next.schemaVersion).toBe(21);
      await storage.write(next);
      expect(await readFile(path + '.v10.backup', 'utf8')).toBe(raw);
      next.workspace.surfaces.push({
        openedBy: { kind: 'user' },
        projectId: 'prj_pdf',
        surfaceId: 'srf_pdf',
        resource: target.resource,
        view: 'pdf',
        pinned: false,
      });
      next.workspace.layout.primary = { tabs: ['srf_pdf'], activeSurfaceId: 'srf_pdf' };
      next.titles = [{ surfaceId: 'srf_pdf', title: 'report.pdf' }];
      expect(() => readStoredWorkspace({ ...next, schemaVersion: 10 })).toThrow();
      expect(() => readStoredWorkspace({ ...next, schemaVersion: 9 })).toThrow();
      await storage.write(next);
      expect((await storage.read())?.workspace.surfaces[0]?.view).toBe('pdf');
      expect(await readFile(path + '.v10.backup', 'utf8')).toBe(raw);
    } finally {
      await rm(base, { recursive: true, force: true });
    }
  });
});

function pdfViewFixture() {
  const rendering = new RenderSession();
  const surface = {
    projectId: target.projectId,
    surfaceId: 'srf_pdf',
    resource: target.resource,
    view: 'pdf' as const,
    pinned: false,
    openedBy: { kind: 'user' as const },
  };
  const data = {
    kind: 'file' as const,
    value: {
      projectId: target.projectId,
      resourceId: 'res_pdf',
      name: 'report.pdf',
      revision: hash,
      size: 10,
      content,
    },
  };
  rendering.reveal([surface.surfaceId]);
  const ticket = rendering.begin(
    { projectId: target.projectId, surfaceId: surface.surfaceId, rendererSessionId: rendering.id },
    surface,
  );
  const load = rendering.complete(ticket, data);
  rendering.release(ticket);
  rendering.acknowledge(load.acknowledgment);
  return { rendering, surface, data, acknowledgment: load.acknowledgment };
}
it('PDF viewport scopes reject unpainted, clipped-out, foreign-page and guessed text targets', () => {
  const view = pdfViewFixture();
  expect(() => pdfObservation(view, undefined, 'view')).toThrow(/visible PDF region/);
  const visible = {
    ...view,
    pdfViewport: { generation: 1, region: [80, 100, 300, 200] as [number, number, number, number] },
  };
  expect(pdfObservation(visible, target.selection, 'view').evidence).toEqual(target);
  const observed = pdfObservation(visible, undefined, 'view');
  expect(observed.evidence.selection.region).toEqual(visible.pdfViewport.region);
  expect(observed.content).toMatchObject({
    scope: 'view',
    textCapability: 'unsupported',
    visibleRegion: [80, 100, 300, 200],
  });
  for (const selection of [
    { ...target.selection, pageIndex: 1 },
    { ...target.selection, modelHash: other },
    { ...target.selection, region: [80, 90, 300, 200] as [number, number, number, number] },
    {
      kind: 'text' as const,
      coordinateSpace: 'utf16-line-column' as const,
      start: { line: 1, column: 0 },
      end: { line: 1, column: 1 },
    },
  ])
    expect(() => pdfObservation(visible, selection, 'view')).toThrow();
  expect(pdfObservation(visible, undefined, 'source-preview').evidence.selection.region).toEqual(
    content.page.model.viewBox,
  );
});
it('PDF viewport generation invalidates a read after scroll-away-and-back, including the same raster receipt', () => {
  const { rendering, acknowledgment } = pdfViewFixture();
  expect(() => rendering.assertPdfViewport(acknowledgment, undefined)).toThrow();
  const region = target.selection.region;
  rendering.setPdfViewport({ acknowledgment, region });
  const original = rendering.snapshot('srf_pdf').pdfViewport!.generation;
  rendering.setPdfViewport({ acknowledgment, region });
  expect(() => rendering.assertPdfViewport(acknowledgment, original)).not.toThrow();
  expect(() => rendering.setPdfViewport({ acknowledgment, region: [0, 0, 900, 900] })).toThrow();
  rendering.setPdfViewport({ acknowledgment, region: null });
  rendering.setPdfViewport({ acknowledgment, region });
  expect(() => rendering.assertPdfViewport(acknowledgment, original)).toThrow();
  rendering.reveal([]);
  expect(() => rendering.setPdfViewport({ acknowledgment, region })).toThrow();
});
it('PDF pointing requires a current turn-owned view scope and exact contained page identity', () => {
  const view = pdfViewFixture(),
    reads = new ObservedReads();
  let current = true;
  const assert = () => {
    if (!current) throw new Error('Viewport changed');
  };
  const receipt = reads.recordPdf(view, target, 'view', assert);
  expect(() =>
    reads.assertPoint(receipt.observedReadId, target, view.acknowledgment),
  ).not.toThrow();
  const smaller = {
    ...target,
    selection: {
      ...target.selection,
      region: [110, 130, 210, 170] as [number, number, number, number],
    },
  };
  expect(() =>
    reads.assertPoint(receipt.observedReadId, smaller, view.acknowledgment),
  ).not.toThrow();
  for (const changed of [
    { ...target, projectId: 'prj_other' },
    { ...target, dataRevision: other },
    { ...target, selection: { ...target.selection, pageIndex: 1 } },
    {
      ...target,
      selection: {
        ...target.selection,
        region: [90, 120, 220, 180] as [number, number, number, number],
      },
    },
  ])
    expect(() => reads.assertPoint(receipt.observedReadId, changed, view.acknowledgment)).toThrow();
  expect(() =>
    new ObservedReads().assertPoint(receipt.observedReadId, target, view.acknowledgment),
  ).toThrow();
  const source = reads.recordPdf(view, target, 'source-preview', assert);
  expect(source.pointable).toBe(false);
  expect(() => reads.assertPoint(source.observedReadId, target, view.acknowledgment)).toThrow();
  current = false;
  expect(() => reads.assertPoint(receipt.observedReadId, target, view.acknowledgment)).toThrow(
    /Viewport changed/,
  );
});
it('freezes v11 storage and v7 Agent tool inputs before enabling PDF authored references in v12', async () => {
  const base = await mkdtemp(join(tmpdir(), 'gobble-pdf-v12-'));
  try {
    await mkdir(join(base, 'projects'));
    const legacy = { ...emptyWorkspace('prj_pdf'), schemaVersion: 11 };
    expect(() => parse(WorkspaceDocumentV11Schema, legacy)).not.toThrow();
    const raw = JSON.stringify(legacy) + '\n',
      path = join(base, 'projects/prj_pdf.json');
    await writeFile(path, raw);
    const storage = new WorkspaceStorage(base).project('prj_pdf');
    const doc = (await storage.read())!;
    expect(doc.schemaVersion).toBe(21);
    doc.sharedReferences = [
      {
        referenceId: 'ref_pdf',
        evidence: target,
        label: 'report.pdf',
        note: 'Inspect this region.',
        author: { kind: 'user' },
        createdAt: 1,
        retracted: false,
      },
    ];
    expect(() => parseWorkspaceDocument(doc)).not.toThrow();
    expect(() => parse(WorkspaceDocumentV11Schema, { ...doc, schemaVersion: 11 })).toThrow();
    expect(() =>
      parse(SharedToolRequestV7Schema, {
        tool: 'workspace_point',
        arguments: { evidence: target, note: 'Inspect.' },
      }),
    ).toThrow();
    await storage.write(doc);
    expect(await readFile(path + '.v11.backup', 'utf8')).toBe(raw);
    expect((await storage.read())?.sharedReferences).toEqual(doc.sharedReferences);
  } finally {
    await rm(base, { recursive: true, force: true });
  }
});

it('migrates the frozen v12 PDF draft and shared reference without changing captured identity', () => {
  const old = { ...emptyWorkspace(target.projectId), schemaVersion: 12 };
  old.chat.draft = 'Keep the PDF discussion';
  old.chat.attachments = [
    {
      attachmentId: 'att_v12',
      label: 'report.pdf',
      createdAt: 1,
      evidence: target,
      capture: {
        kind: 'pdf',
        capturedAt: 1,
        asset: { hash, byteLength: 100, mediaType: 'image/png' },
        representation: view,
      },
    },
  ];
  old.sharedReferences = [
    {
      referenceId: 'ref_v12',
      evidence: target,
      author: { kind: 'user' },
      createdAt: 1,
      note: 'Review this region',
      retracted: false,
      label: 'report.pdf',
    },
  ];
  const before = JSON.stringify(old);
  const next = readStoredWorkspace(old);
  expect(next).toEqual({ ...old, schemaVersion: 21 });
  expect(JSON.stringify(old)).toBe(before);
  expect(next.chat.attachments).toEqual(old.chat.attachments);
  expect(next.sharedReferences).toEqual(old.sharedReferences);
});
