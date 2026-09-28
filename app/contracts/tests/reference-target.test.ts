import { describe, it, expect } from 'vitest';
import {
  EvidenceRefSchema,
  parse,
  resolveReferenceTarget,
  type EvidenceRefV2 as EvidenceRef,
  type ReferenceSource,
  type FileContent,
} from '../src';

const ref = (selection?: EvidenceRef['selection']): EvidenceRef => ({
  schemaVersion: 2,
  projectId: 'prj_one',
  resource: { kind: 'file', resourceId: 'res_sample' },
  dataRevision: 'revision-1',
  ...(selection ? { selection } : {}),
});
function source(
  content: Exclude<FileContent['content'], { kind: 'pdf' | 'notebook' }>,
): ReferenceSource {
  return {
    projectId: 'prj_one',
    resource: ref().resource,
    dataRevision: 'revision-1',
    data: {
      kind: 'file',
      value: {
        projectId: 'prj_one',
        resourceId: 'res_sample',
        name: 'sample',
        revision: 'revision-1',
        size: 40,
        content,
      },
    },
  };
}
describe('portable reference identity and exact resolution', () => {
  it('accepts an absent origin and rejects unversioned or implicit coordinate payloads', () => {
    const evidence = ref({
      kind: 'text',
      coordinateSpace: 'utf16-line-column',
      start: { line: 1, column: 1 },
      end: { line: 1, column: 3 },
    });
    expect(parse(EvidenceRefSchema, evidence)).toEqual(evidence);
    for (const invalid of [
      { ...evidence, schemaVersion: undefined },
      { ...evidence, schemaVersion: 3 },
      { ...evidence, surfaceId: 'srf_old' },
      { ...evidence, selection: { ...evidence.selection, coordinateSpace: undefined } },
      { ...evidence, selection: { ...evidence.selection, coordinateSpace: 'utf8-bytes' } },
    ])
      expect(() => parse(EvidenceRefSchema, invalid)).toThrow();
  });
  it('uses UTF-16 half-open ranges and refuses split surrogate pairs or missing lines', () => {
    const loaded = source({ kind: 'text', text: 'α😀xyz\nend' });
    const selection = {
      kind: 'text' as const,
      coordinateSpace: 'utf16-line-column' as const,
      start: { line: 1, column: 1 },
      end: { line: 1, column: 3 },
    };
    expect(resolveReferenceTarget(ref(selection), loaded)).toEqual({ kind: 'exact' });
    for (const invalid of [
      { ...selection, start: { line: 1, column: 2 } },
      { ...selection, end: { line: 1, column: 2 } },
      { ...selection, end: { line: 3, column: 0 } },
      { ...selection, end: { line: 1, column: 99 } },
      { ...selection, end: selection.start },
    ])
      expect(resolveReferenceTarget(ref(invalid), loaded)).toEqual({
        kind: 'unavailable',
        reason: 'selection-unavailable',
      });
  });
  it('ignores origin and never reuses an ordinal row in another revision', () => {
    const loaded = source({
      kind: 'table',
      columns: [{ id: 'col_0', name: 'Gene' }],
      rows: [
        { key: 'row_2', cells: ['B'] },
        { key: 'row_1', cells: ['A'] },
      ],
      truncated: false,
    });
    const target = ref({
      kind: 'table',
      coordinateSpace: 'revision-row-column-keys',
      rowKeys: ['row_1'],
      columns: ['col_0'],
    });
    for (const evidence of [
      target,
      { ...target, origin: { surfaceId: 'srf_closed' } },
      { ...target, origin: { surfaceId: 'srf_duplicate' } },
    ])
      expect(resolveReferenceTarget(evidence, loaded)).toEqual({ kind: 'exact' });
    const newer = { ...loaded, dataRevision: 'revision-2' };
    expect(resolveReferenceTarget(target, newer)).toEqual({
      kind: 'historical',
      currentRevision: 'revision-2',
    });
    expect(
      resolveReferenceTarget(
        ref({
          kind: 'table',
          coordinateSpace: 'revision-row-column-keys',
          rowKeys: ['row_missing'],
          columns: ['col_0'],
        }),
        loaded,
      ),
    ).toEqual({ kind: 'unavailable', reason: 'selection-unavailable' });
  });
  it('distinguishes missing source and foreign identity even if revisions coincide', () => {
    const loaded = source({ kind: 'text', text: 'identical' });
    expect(resolveReferenceTarget(ref())).toEqual({
      kind: 'unavailable',
      reason: 'source-unavailable',
    });
    expect(resolveReferenceTarget({ ...ref(), projectId: 'prj_other' }, loaded)).toEqual({
      kind: 'unavailable',
      reason: 'resource-mismatch',
    });
    expect(
      resolveReferenceTarget(
        { ...ref(), resource: { kind: 'file', resourceId: 'res_other' } },
        loaded,
      ),
    ).toEqual({ kind: 'unavailable', reason: 'resource-mismatch' });
    loaded.data.value.projectId = 'prj_other';
    expect(resolveReferenceTarget(ref(), loaded)).toEqual({
      kind: 'unavailable',
      reason: 'resource-mismatch',
    });
  });
  it('anchors image geometry to original pixels and content hash', () => {
    const hash = 'sha256:' + 'a'.repeat(64);
    const loaded = source({
      kind: 'image',
      mediaType: 'image/png',
      base64: 'fixture',
      width: 100,
      height: 80,
    });
    loaded.dataRevision = hash;
    if (loaded.data.kind !== 'file') throw new Error('Missing fixture');
    loaded.data.value.revision = hash;
    const selection = {
      kind: 'image' as const,
      coordinateSpace: 'normalized-original-image' as const,
      x: 0.25,
      y: 0.25,
      width: 0.5,
      height: 0.5,
      originalWidth: 100,
      originalHeight: 80,
      contentHash: hash,
    };
    const target = { ...ref(selection), dataRevision: hash };
    expect(resolveReferenceTarget(target, loaded)).toEqual({ kind: 'exact' });
    for (const invalid of [
      { ...selection, originalWidth: 200 },
      { ...selection, x: 0.75 },
      { ...selection, contentHash: 'sha256:' + 'b'.repeat(64) },
    ])
      expect(resolveReferenceTarget({ ...target, selection: invalid }, loaded)).toEqual({
        kind: 'unavailable',
        reason: 'selection-unavailable',
      });
  });
});
