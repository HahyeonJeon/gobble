import {
  notebookSelectionContains,
  containsObservedTarget,
  validateNotebookViewport,
  presentationRevision,
} from '@gobble/contracts';
import { notebookObservation } from '../src/main/shared-context/notebook-observation';
import { ObservedReads } from '../src/main/shared-context/observed';
import { readFile } from 'node:fs/promises';
import { describe, it, expect } from 'vitest';
import {
  parse,
  NotebookTargetSchema,
  NotebookDocumentSchema,
  resolveReferenceTarget,
  SharedToolRequestV8Schema,
  WorkspaceDocumentV13Schema,
  needsToolsetRenewal,
  SharedToolRequestSchema,
  WorkspaceDocumentV12Schema,
  readStoredWorkspace,
  parseWorkspaceDocument,
  sameNotebookTarget,
  validateManifest,
  type NotebookTarget,
  type SurfaceData,
  type EvidenceManifest,
} from '@gobble/contracts';
import { parseNotebook } from '../src/main/notebook/parser';
import { fixture } from '../../qualification/notebook/fixtures';
import { emptyWorkspace } from '../src/main/workspace/model';
const snapshot = parseNotebook(fixture('python'));
const document = parse(NotebookDocumentSchema, {
  ...snapshot,
  cells: snapshot.cells.map((c) => ({
    ...c,
    source: { kind: 'text', text: c.source.text, digest: c.source.digest },
    outputs: c.outputs.map((o) => ({
      ...o,
      part:
        o.part.kind === 'text'
          ? { kind: 'text', text: o.part.text, digest: o.part.digest }
          : o.part.kind === 'image'
            ? {
                kind: 'image',
                mime: o.part.mime,
                width: o.part.width,
                height: o.part.height,
                digest: o.part.digest,
              }
            : o.part,
    })),
  })),
});
const target: NotebookTarget = {
  schemaVersion: 6,
  projectId: 'prj_notebook',
  resource: { kind: 'file', resourceId: 'res_notebook' },
  origin: { surfaceId: 'srf_notebook' },
  dataRevision: snapshot.revision,
  selection: {
    kind: 'notebook',
    profile: snapshot.profile,
    cell: { kind: 'id', id: 'filter' },
    part: { kind: 'source' },
    selector: { kind: 'text', coordinateSpace: 'notebook-display-utf16', start: 0, end: 4 },
  },
};
const data: SurfaceData = {
  kind: 'file',
  value: {
    projectId: target.projectId,
    resourceId: target.resource.resourceId,
    revision: target.dataRevision,
    name: 'research.ipynb',
    size: snapshot.bytes,
    content: { kind: 'notebook', document },
  },
};
const source = {
  projectId: target.projectId,
  resource: target.resource,
  dataRevision: target.dataRevision,
  data,
};
describe('Notebook contracts and frozen compatibility', () => {
  it('requires explicit target version, units and closed payloads', () => {
    expect(parse(NotebookTargetSchema, target)).toEqual(target);
    for (const value of [
      { ...target, schemaVersion: 5 },
      { ...target, extra: true },
      {
        ...target,
        selection: { ...target.selection, selector: { kind: 'text', start: 0, end: 4 } },
      },
    ])
      expect(() => parse(NotebookTargetSchema, value)).toThrow();
    expect(
      sameNotebookTarget(target, {
        ...target,
        selection: { ...target.selection, cell: { id: 'filter', kind: 'id' } },
      }),
    ).toBe(true);
  });
  it('resolves only the exact resource, cell and display range', () => {
    expect(resolveReferenceTarget(target, source).kind).toBe('exact');
    expect(
      resolveReferenceTarget({ ...target, dataRevision: 'sha256:' + 'f'.repeat(64) }, source).kind,
    ).toBe('historical');
    expect(resolveReferenceTarget({ ...target, projectId: 'prj_other' }, source).kind).toBe(
      'unavailable',
    );
    for (const selection of [
      { ...target.selection, cell: { kind: 'ordinal' as const, index: 2 } },
      {
        ...target.selection,
        selector: {
          kind: 'text' as const,
          coordinateSpace: 'notebook-display-utf16' as const,
          start: 4,
          end: 4,
        },
      },
    ])
      expect(resolveReferenceTarget({ ...target, selection }, source).kind).toBe('unavailable');
    expect(
      resolveReferenceTarget(
        {
          schemaVersion: 2,
          projectId: target.projectId,
          resource: target.resource,
          dataRevision: target.dataRevision,
        },
        source,
      ).kind,
    ).toBe('unavailable');
  });
  it('binds an output to its original index, MIME and digest', () => {
    const o = document.cells[2]!.outputs[0]!;
    if (o.part.kind !== 'text') throw new Error('Expected text output');
    const selection = {
      ...target.selection,
      part: { kind: 'output' as const, index: 0, mime: o.mime, digest: o.part.digest },
    };
    expect(resolveReferenceTarget({ ...target, selection }, source).kind).toBe('exact');
    for (const part of [
      { ...selection.part, index: 1 },
      { ...selection.part, mime: 'text/html' },
      { ...selection.part, digest: 'sha256:' + 'f'.repeat(64) },
    ])
      expect(
        resolveReferenceTarget({ ...target, selection: { ...selection, part } }, source).kind,
      ).toBe('unavailable');
  });
  it('keeps the v8 Agent tool input byte-equivalent and excludes Notebook pointing', async () => {
    const frozen = JSON.parse(
      await readFile(
        new URL(
          '../../../docs/desktop-workspace/stages/r4a2-review/toolset-v8.json',
          import.meta.url,
        ),
        'utf8',
      ),
    );
    expect(JSON.parse(JSON.stringify(SharedToolRequestV8Schema))).toEqual(frozen);
    expect(() =>
      parse(SharedToolRequestV8Schema, {
        tool: 'workspace_point',
        arguments: { evidence: target, note: '' },
      }),
    ).toThrow();
  });
  it('accepts Notebook points in v9 and migrates exact v13 User state', () => {
    expect(
      parse(SharedToolRequestSchema, {
        tool: 'workspace_point',
        arguments: { evidence: target, note: 'Look here' },
      }),
    ).toBeTruthy();
    const old = { ...emptyWorkspace(target.projectId), schemaVersion: 13 };
    old.chat.draft = 'Keep my draft';
    expect(() =>
      parse(WorkspaceDocumentV13Schema, {
        ...old,
        sharedReferences: [
          {
            referenceId: 'ref_old',
            evidence: target,
            label: 'Notebook',
            note: '',
            author: { kind: 'user' },
            createdAt: 1,
            retracted: false,
          },
        ],
      }),
    ).toThrow();
    expect(
      needsToolsetRenewal({
        access: 'sharedViews',
        provider: { threadId: 'thread_old', toolsetVersion: 'shared-views-v8' },
      }),
    ).toBe(true);
    expect(readStoredWorkspace(old)).toEqual({ ...old, schemaVersion: 21 });
  });
  it('rejects missing capture and inconsistent text/image representation', () => {
    const doc = emptyWorkspace(target.projectId);
    doc.chat.attachments = [
      { attachmentId: 'att_a', evidence: target, label: 'Notebook', createdAt: 1 },
    ];
    expect(() => parseWorkspaceDocument(doc)).toThrow(/frozen/);
    const asset = {
      hash: 'sha256:' + 'a'.repeat(64),
      byteLength: 100,
      mediaType: 'application/json' as const,
    };
    const representation = {
      kind: 'text' as const,
      truncated: false as const,
      notebook: true as const,
    };
    const manifest: EvidenceManifest = {
      ...doc.chat.attachments[0]!,
      capturedAt: 1,
      asset,
      representation,
      capture: { kind: 'notebook', capturedAt: 1, asset, representation },
    };
    expect(() => validateManifest(manifest)).not.toThrow();
    expect(() =>
      validateManifest({ ...manifest, asset: { ...asset, byteLength: 65537 } }),
    ).toThrow();
  });
  it('migrates frozen v12 metadata without changing its draft or layout', () => {
    const old = { ...emptyWorkspace(target.projectId), schemaVersion: 12 };
    old.chat.draft = 'Preserve this draft';
    parse(WorkspaceDocumentV12Schema, old);
    const frozen = JSON.stringify(old),
      next = readStoredWorkspace(old);
    expect(next).toEqual({ ...old, schemaVersion: 21 });
    expect(JSON.stringify(old)).toBe(frozen);
    expect(readStoredWorkspace(next)).toEqual(next);
    expect(() =>
      readStoredWorkspace({
        ...old,
        chat: {
          ...old.chat,
          attachments: [
            {
              attachmentId: 'att_future',
              evidence: target,
              label: 'Invalid old target',
              createdAt: 1,
            },
          ],
        },
      }),
    ).toThrow();
  });
});

describe('Notebook observation authority', () => {
  const surface: import('@gobble/contracts').Surface = {
    projectId: target.projectId,
    surfaceId: 'srf_notebook',
    resource: target.resource,
    view: 'notebook',
    openedBy: { kind: 'user' },
    pinned: false,
  };
  const view = {
    surface,
    data,
    notebookViewport: { generation: 1, viewport: { parts: [target.selection] } },
  };
  it('returns visible text with exact addresses and no authority for hidden parts', () => {
    const result = notebookObservation(view, undefined, 'view');
    expect(result.textParts).toEqual([
      {
        evidence: target,
        text: document.cells
          .find((c) => c.address.kind === 'id' && c.address.id === 'filter')!
          .source.text.slice(0, 4),
      },
    ]);
    expect(result.targets).toEqual([target]);
    expect(() =>
      notebookObservation(
        view,
        {
          ...target.selection,
          selector: { kind: 'text', coordinateSpace: 'notebook-display-utf16', start: 0, end: 5 },
        },
        'view',
      ),
    ).toThrow(/not fully visible/);
    expect(() =>
      notebookObservation({ ...view, notebookViewport: undefined }, undefined, 'view'),
    ).toThrow(/not been confirmed/);
    expect(() => notebookObservation(view, undefined, 'source-preview')).toThrow(/one exact/);
    expect(
      notebookObservation(
        view,
        {
          ...target.selection,
          selector: { kind: 'text', coordinateSpace: 'notebook-display-utf16', start: 0, end: 5 },
        },
        'source-preview',
      ).scope,
    ).toBe('source-preview');
  });
  it('compares identity values, rejects foreign parts and bounds viewport size', () => {
    expect(
      notebookSelectionContains(target.selection, {
        ...target.selection,
        cell: { id: 'filter', kind: 'id' },
      }),
    ).toBe(true);
    expect(
      notebookSelectionContains(target.selection, {
        ...target.selection,
        cell: { kind: 'id', id: 'load' },
      }),
    ).toBe(false);
    expect(containsObservedTarget(target, { ...target, projectId: 'prj_other' })).toBe(false);
    expect(
      containsObservedTarget(target, { ...target, dataRevision: 'sha256:' + 'a'.repeat(64) }),
    ).toBe(false);
    expect(() =>
      validateNotebookViewport(document, {
        parts: Array.from({ length: 65 }, () => target.selection),
      }),
    ).toThrow(/part limit/);
  });
  it('separates advertised images from returned pixels', () => {
    const imageDoc = structuredClone(document),
      cell = imageDoc.cells[2]!;
    const part = {
      kind: 'image' as const,
      mime: 'image/png' as const,
      width: 500,
      height: 300,
      digest: target.dataRevision,
    };
    cell.outputs = [
      { index: 0, type: 'display_data', mime: 'image/png', alternatives: [], notice: '', part },
    ];
    const selection: import('@gobble/contracts').NotebookSelection = {
      ...target.selection,
      cell: cell.address,
      part: { kind: 'output', index: 0, mime: 'image/png', digest: part.digest },
      selector: {
        kind: 'image',
        coordinateSpace: 'natural-image-pixels',
        rect: { x: 20, y: 10, width: 100, height: 80 },
      },
    };
    const imageData: SurfaceData = {
      ...data,
      value: { ...data.value, content: { kind: 'notebook', document: imageDoc } },
    };
    const imageView = {
      surface,
      data: imageData,
      notebookViewport: { generation: 2, viewport: { parts: [selection] } },
    };
    const available = notebookObservation(imageView, undefined, 'view');
    expect(available.targets).toEqual([]);
    expect(available.imagesAvailable).toHaveLength(1);
    expect(available.image).toBeUndefined();
    const observed = notebookObservation(imageView, selection, 'view');
    expect(observed.image?.selection).toEqual(selection);
    expect(observed.targets).toHaveLength(1);
    expect(() =>
      notebookObservation(
        imageView,
        {
          ...selection,
          selector: {
            kind: 'image',
            coordinateSpace: 'natural-image-pixels',
            rect: { x: 0, y: 0, width: 500, height: 300 },
          },
        },
        'view',
      ),
    ).toThrow(/not fully visible/);
  });
  it('revokes receipts after viewport change and refuses preview or missing receipts', () => {
    const reads = new ObservedReads(),
      ack: import('@gobble/contracts').RenderAcknowledgment = {
        schemaVersion: 3,
        projectId: target.projectId,
        surfaceId: surface.surfaceId,
        rendererSessionId: 'rnd_test',
        requestId: 'req_read',
        generation: 1,
        dataRevision: target.dataRevision,
        presentation: presentationRevision(surface),
      };
    let current = true;
    const receipt = reads.recordNotebook({ ...view, acknowledgment: ack }, [target], 'view', () => {
      if (!current) throw new Error('Viewport changed');
    });
    expect(() => reads.assertPoint(receipt.observedReadId, target, ack)).not.toThrow();
    expect(() => reads.assertPoint(undefined, target, ack)).toThrow();
    const preview = reads.recordNotebook(
      { ...view, acknowledgment: ack },
      [target],
      'source-preview',
      () => {},
    );
    expect(preview.pointable).toBe(false);
    expect(() => reads.assertPoint(preview.observedReadId, target, ack)).toThrow();
    current = false;
    expect(() => reads.assertPoint(receipt.observedReadId, target, ack)).toThrow(
      /Viewport changed/,
    );
  });
});
