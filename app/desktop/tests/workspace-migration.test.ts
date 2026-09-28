import { mkdtemp, mkdir, readFile, rm, writeFile } from 'node:fs/promises';
import { join } from 'node:path';
import { tmpdir } from 'node:os';
import { afterEach, describe, expect, it } from 'vitest';
import {
  parse,
  parseWorkspaceDocument,
  readStoredWorkspace,
  WorkspaceDocumentV1Schema,
} from '@gobble/contracts';
import { WorkspaceStorage } from '../src/main/workspace/storage';

const paths: string[] = [];
afterEach(async () => {
  await Promise.all(paths.splice(0).map((path) => rm(path, { recursive: true, force: true })));
});
async function legacy() {
  const doc = parse(
    WorkspaceDocumentV1Schema,
    JSON.parse(await readFile(new URL('./fixtures/workspace-v1.json', import.meta.url), 'utf8')),
  );
  doc.workspace.layout = {
    kind: 'split',
    primary: doc.workspace.layout.primary,
    secondary: { tabs: [], activeSurfaceId: null },
    primaryFraction: 0.7,
  };
  doc.discussion.collapsed = true;
  doc.workspace.agents.push({
    projectId: 'prj_atlas',
    agentId: 'agt_reader',
    name: 'Reader',
    instructionProfile: 'discussion-v1',
    provider: { kind: 'codex', threadId: 'thread-existing' },
  });
  doc.discussion.recipientAgentId = 'agt_reader';
  doc.collaboration = {
    submissions: [
      {
        requestId: 'req_saved',
        agentId: 'agt_reader',
        text: 'Earlier message',
        model: 'model',
        effort: 'low',
        createdAt: 1,
        state: 'completed',
        threadId: 'thread-existing',
        turnId: 'turn-existing',
        response: 'Earlier response',
        problem: null,
      },
    ],
  };
  return doc;
}

describe('workspace presentation and reference migration to v4', () => {
  it('validates before migrating and retains collaboration, selections and protected views without mutating input', async () => {
    const input = await legacy();
    const original = structuredClone(input);
    const next = readStoredWorkspace(input);
    expect(input).toEqual(original);
    expect(next.chat).toEqual({
      collapsed: true,
      width: 420,
      draft: 'Keep this draft',
      recipientAgentId: 'agt_reader',
    });
    expect(next.workspace.layout).toEqual({ ...input.workspace.layout, primaryFraction: 0.5 });
    expect(next.workspace.surfaces).toEqual(input.workspace.surfaces);
    expect(next.workspace.agents).toEqual(input.workspace.agents);
    expect(next.collaboration).toEqual(input.collaboration);
    expect(next.selections[0]?.surfaceId).toBe(input.selections[0]?.surfaceId);
    expect(next.selections[0]?.evidence).toMatchObject({
      schemaVersion: 2,
      origin: { surfaceId: 'srf_legacy' },
      selection: { coordinateSpace: 'utf16-line-column' },
    });
    expect(readStoredWorkspace(next)).toEqual(next);
    expect(() => parseWorkspaceDocument(input)).toThrow(); // migration is storage-only
    const invalid = structuredClone(input);
    invalid.workspace.layout.primary.activeSurfaceId = 'srf_foreign';
    expect(() => readStoredWorkspace(invalid)).toThrow();
    expect(() => readStoredWorkspace({ ...input, extra: true })).toThrow();
    expect(() => readStoredWorkspace({ ...input, schemaVersion: 9 })).toThrow();
  });
  it('preserves exact original bytes in an immutable migration archive across later saves and restarts', async () => {
    const directory = await mkdtemp(join(tmpdir(), 'gobble-migration-'));
    paths.push(directory);
    await mkdir(join(directory, 'projects'));
    const path = join(directory, 'projects/prj_atlas.json');
    const raw = JSON.stringify(await legacy(), null, 2) + '\n';
    await writeFile(path, raw);
    const file = new WorkspaceStorage(directory).project('prj_atlas');
    const next = await file.read();
    expect(await readFile(path, 'utf8')).toBe(raw); // reading never writes
    if (!next) throw new Error('Missing document');
    await file.write(next);
    expect(await readFile(path + '.v1.backup', 'utf8')).toBe(raw);
    next.chat.width = 500;
    await file.write(next);
    const restarted = new WorkspaceStorage(directory).project('prj_atlas');
    expect(await restarted.read()).toEqual(next);
    next.chat.draft = 'New unsent draft';
    await restarted.write(next);
    expect(await readFile(path + '.v1.backup', 'utf8')).toBe(raw);
    expect(JSON.parse(await readFile(path, 'utf8')).schemaVersion).toBe(21);
  });
  it('refuses migration when another archive or a foreign Project would be overwritten', async () => {
    const directory = await mkdtemp(join(tmpdir(), 'gobble-migration-'));
    paths.push(directory);
    await mkdir(join(directory, 'projects'));
    const path = join(directory, 'projects/prj_atlas.json');
    const raw = JSON.stringify(await legacy());
    await writeFile(path, raw);
    await writeFile(path + '.v1.backup', 'Existing archive');
    const file = new WorkspaceStorage(directory).project('prj_atlas');
    const next = await file.read();
    if (!next) throw new Error('Missing document');
    await expect(file.write(next)).rejects.toMatchObject({ code: 'internal' });
    expect(await readFile(path, 'utf8')).toBe(raw);
    expect(await readFile(path + '.v1.backup', 'utf8')).toBe('Existing archive');
    const foreign = join(directory, 'projects/prj_other.json');
    await writeFile(foreign, raw);
    await expect(new WorkspaceStorage(directory).project('prj_other').read()).rejects.toMatchObject(
      { code: 'internal' },
    );
    expect(await readFile(foreign, 'utf8')).toBe(raw);
  });
});

async function versionTwo() {
  return JSON.parse(
    await readFile(new URL('./fixtures/workspace-v2.json', import.meta.url), 'utf8'),
  ) as unknown;
}

describe('v2 durable evidence migration', () => {
  it('keeps the old reader identical to the published v2 schema including nested limits', async () => {
    const { WorkspaceDocumentV2Schema } = await import('@gobble/contracts');
    const published = JSON.parse(
      await readFile(new URL('../../contracts/schema/v2.json', import.meta.url), 'utf8'),
    );
    const normalize = (value: unknown, key = ''): unknown => {
      if (Array.isArray(value))
        return key === 'required' ? [...value].sort() : value.map((item) => normalize(item));
      if (value !== null && typeof value === 'object')
        return Object.fromEntries(
          Object.entries(value).map(([key, item]) => [key, normalize(item, key)]),
        );
      return value;
    };
    expect(normalize(JSON.parse(JSON.stringify(WorkspaceDocumentV2Schema)))).toEqual(
      normalize(published.definitions.WorkspaceDocumentSchema),
    );
  });
  it('migrates every reference owner without changing captured assets or requiring closed origin views', async () => {
    const { WorkspaceDocumentV2Schema, isQuestion } = await import('@gobble/contracts');
    const old = parse(WorkspaceDocumentV2Schema, await versionTwo());
    const original = structuredClone(old);
    const next = readStoredWorkspace(old);
    expect(old).toEqual(original);
    expect(next.schemaVersion).toBe(21);
    expect(
      next.selections.map((item) =>
        item.evidence.selection && 'coordinateSpace' in item.evidence.selection
          ? item.evidence.selection.coordinateSpace
          : undefined,
      ),
    ).toEqual(['utf16-line-column', 'revision-row-column-keys', 'normalized-original-image']);
    const question = next.workspace.decisions.find(isQuestion)!;
    const legacy = next.workspace.decisions.find((item) => !isQuestion(item))!;
    const refs = [
      ...next.selections.map((item) => item.evidence),
      ...next.sharedReferences!.map((item) => item.evidence),
      ...next.chat.attachments!.map((item) => item.evidence),
      ...next.collaboration!.submissions[0]!.evidence!.map((item) => item.evidence),
      ...question.evidence.map((item) => item.evidence),
      ...(!isQuestion(legacy) ? legacy.evidence : []),
    ];
    expect(refs).toHaveLength(9);
    for (const target of refs) {
      expect(target.schemaVersion).toBe(2);
      expect(target).not.toHaveProperty('surfaceId');
      if (target.schemaVersion === 4) throw new Error('Legacy fixture required');
      expect(target.origin?.surfaceId).toMatch(/^srf_/);
    }
    const pointer = next.sharedReferences![0]!.evidence;
    if (pointer.schemaVersion === 4) throw new Error('Legacy fixture required');
    expect(pointer.origin?.surfaceId).toBe('srf_closed_original');
    expect(
      next.workspace.surfaces.some((surface) => surface.surfaceId === 'srf_closed_original'),
    ).toBe(false);
    expect(next.chat.draft).toBe(old.chat.draft);
    expect(next.workspace.agents).toEqual(old.workspace.agents); // old provider definitions need explicit renewal, not silent reuse
    expect(next.collaboration!.submissions[0]!.evidence![0]!.asset).toEqual(
      old.collaboration!.submissions[0]!.evidence![0]!.asset,
    );
    expect(question.evidence[0]!.representation).toEqual({
      kind: 'image',
      width: 1,
      height: 1,
      originalWidth: 1,
      originalHeight: 1,
      crop: { x: 0, y: 0, width: 1, height: 1 },
    });
    expect(readStoredWorkspace(next)).toEqual(next);
    // Portable targets may shed origin independently of saved local Surface bindings.
    for (const target of refs) {
      if (target.schemaVersion === 4) throw new Error('Legacy fixture required');
      delete target.origin;
    }
    expect(parseWorkspaceDocument(next)).toEqual(next);
  });
  it('keeps an immutable v2 archive and reads the same historical blob after migration and restart', async () => {
    const { EvidenceStorage } = await import('../src/main/evidence/storage');
    const directory = await mkdtemp(join(tmpdir(), 'gobble-v2-migration-'));
    paths.push(directory);
    await mkdir(join(directory, 'projects'));
    const path = join(directory, 'projects/prj_atlas.json');
    const raw = JSON.stringify(await versionTwo(), null, 2) + '\n';
    await writeFile(path, raw);
    const file = new WorkspaceStorage(directory).project('prj_atlas');
    const next = (await file.read())!;
    const captured = next.collaboration!.submissions[0]!.evidence![0]!;
    const bytes = await readFile(new URL('./fixtures/evidence-v2-table.json', import.meta.url));
    const blobDir = join(directory, 'evidence/prj_atlas');
    await mkdir(blobDir, { recursive: true });
    await writeFile(join(blobDir, captured.asset.hash.slice(7) + '.blob'), bytes);
    await file.write(next);
    expect(await readFile(path + '.v2.backup', 'utf8')).toBe(raw);
    const restarted = new WorkspaceStorage(directory).project('prj_atlas');
    const restored = (await restarted.read())!;
    restored.chat.draft = 'Later draft';
    await restarted.write(restored);
    expect(await readFile(path + '.v2.backup', 'utf8')).toBe(raw);
    const asset = await new EvidenceStorage(join(directory, 'evidence')).read(
      'prj_atlas',
      restored.collaboration!.submissions[0]!.evidence![0]!,
    );
    expect(asset.bytes).toEqual(bytes);
    expect(asset.manifest.asset).toEqual(captured.asset);
  });
  it('preserves invalid, future and archive-conflicting v2 inputs without resetting them', async () => {
    const { WorkspaceDocumentV2Schema } = await import('@gobble/contracts');
    const old = parse(WorkspaceDocumentV2Schema, await versionTwo());
    for (const invalid of [
      { ...old, schemaVersion: 4 },
      { ...old, unexpected: true },
      { ...old, selections: [...old.selections, old.selections[0]] },
      {
        ...old,
        sharedReferences: [
          {
            ...old.sharedReferences![0],
            evidence: { ...old.sharedReferences![0]!.evidence, projectId: 'prj_foreign' },
          },
        ],
      },
      {
        ...old,
        selections: [
          {
            ...old.selections[0],
            selection: { kind: 'text', start: { line: 2, column: 0 }, end: { line: 1, column: 0 } },
          },
        ],
      },
    ]) {
      const directory = await mkdtemp(join(tmpdir(), 'gobble-invalid-migration-'));
      paths.push(directory);
      await mkdir(join(directory, 'projects'));
      const path = join(directory, 'projects/prj_atlas.json');
      const raw = JSON.stringify(invalid);
      await writeFile(path, raw);
      await expect(
        new WorkspaceStorage(directory).project('prj_atlas').read(),
      ).rejects.toMatchObject({ code: 'internal' });
      expect(await readFile(path, 'utf8')).toBe(raw);
    }
    const directory = await mkdtemp(join(tmpdir(), 'gobble-conflicting-migration-'));
    paths.push(directory);
    await mkdir(join(directory, 'projects'));
    const path = join(directory, 'projects/prj_atlas.json');
    const raw = JSON.stringify(old);
    await writeFile(path, raw);
    await writeFile(path + '.v2.backup', 'Existing archive');
    const file = new WorkspaceStorage(directory).project('prj_atlas');
    await expect(file.write((await file.read())!)).rejects.toMatchObject({ code: 'internal' });
    expect(await readFile(path, 'utf8')).toBe(raw);
    expect(await readFile(path + '.v2.backup', 'utf8')).toBe('Existing archive');
  });
});

describe('v3 to v4 explicit links migration', () => {
  it('freezes the published v3 shape and preserves references, captures and original bytes', async () => {
    const { WorkspaceDocumentV3Schema } = await import('@gobble/contracts');
    const published = JSON.parse(
      await readFile(new URL('../../contracts/schema/v3.json', import.meta.url), 'utf8'),
    );
    const normalize = (value: unknown, key = ''): unknown => {
      if (Array.isArray(value))
        return key === 'required' ? [...value].sort() : value.map((item) => normalize(item));
      if (value !== null && typeof value === 'object')
        return Object.fromEntries(Object.entries(value).map(([k, v]) => [k, normalize(v, k)]));
      return value;
    };
    expect(normalize(JSON.parse(JSON.stringify(WorkspaceDocumentV3Schema)))).toEqual(
      normalize(published.definitions.WorkspaceDocumentSchema),
    );
    const raw = await readFile(new URL('./fixtures/workspace-v3.json', import.meta.url), 'utf8');
    const original = parse(WorkspaceDocumentV3Schema, JSON.parse(raw));
    const directory = await mkdtemp(join(tmpdir(), 'gobble-v3-migration-'));
    paths.push(directory);
    await mkdir(join(directory, 'projects'));
    const path = join(directory, 'projects/prj_atlas.json');
    await writeFile(path, raw);
    const file = new WorkspaceStorage(directory).project('prj_atlas');
    const next = (await file.read())!;
    expect(next).toEqual({ ...original, schemaVersion: 21, viewLinks: [] });
    expect(await readFile(path, 'utf8')).toBe(raw);
    await file.write(next);
    next.chat.draft = 'New unsent draft';
    await file.write(next);
    expect(await readFile(path + '.v3.backup', 'utf8')).toBe(raw);
    expect(await new WorkspaceStorage(directory).project('prj_atlas').read()).toEqual(next);
    expect(next.collaboration).toEqual(original.collaboration);
    expect(next.sharedReferences).toEqual(original.sharedReferences);
    expect(next.selections).toEqual(original.selections);
    expect(() => readStoredWorkspace({ ...original, viewLinks: [] })).toThrow();
  });
});
