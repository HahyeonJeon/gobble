import { mkdtemp, readFile, rm, writeFile, symlink, readdir } from 'node:fs/promises';
import { join } from 'node:path';
import { tmpdir } from 'node:os';
import { afterEach, describe, expect, it } from 'vitest';
import {
  parseWorkspaceDocument,
  type EvidenceRefV2 as EvidenceRef,
  type SelectionV2 as Selection,
  type SurfaceData,
  type WorkspaceAction,
  type Submission,
} from '@gobble/contracts';
import { WorkspaceController } from '../src/main/workspace/controller';
import { WorkspaceStorage } from '../src/main/workspace/storage';
import type { WorkspaceResources } from '../src/main/workspace/service';
import { EvidenceStorage } from '../src/main/evidence/storage';
import { EvidenceService } from '../src/main/evidence/service';
import {
  contentHash,
  materializeEvidence,
  evidencePreview,
} from '../src/main/evidence/materialize';
import { CollaborationCoordinator } from '../src/main/collaboration/coordinator';
import type { ConversationProvider, ProviderInput } from '../src/main/collaboration/provider';
import type { ImageRenderer } from '../src/main/shared-context/observation';

const paths: string[] = [];
afterEach(async () => {
  for (const path of paths.splice(0)) await rm(path, { recursive: true, force: true });
});
const projectId = 'prj_one';
const textData = (text = 'Alpha\nBeta 😀 gamma\n'): SurfaceData => ({
  kind: 'file',
  value: {
    projectId,
    resourceId: 'res_text',
    name: 'notes.txt',
    revision: contentHash(Buffer.from(text)),
    size: Buffer.byteLength(text),
    content: { kind: 'text', text },
  },
});
const revision = contentHash(Buffer.from('table'));
const renderImage: ImageRenderer = async (_content, selection) => ({
  url: 'data:image/png;base64,AA==',
  width: selection ? 25 : 100,
  height: selection ? 20 : 80,
  crop: {
    x: selection ? 50 : 0,
    y: selection ? 40 : 0,
    width: selection ? 25 : 100,
    height: selection ? 20 : 80,
  },
});

async function setup(quota?: number) {
  const path = await mkdtemp(join(tmpdir(), 'gobble-evidence-'));
  paths.push(path);
  const data = new Map<string, SurfaceData>([
    ['res_text', textData()],
    [
      'res_table',
      {
        kind: 'file',
        value: {
          projectId,
          resourceId: 'res_table',
          name: 'samples.csv',
          revision,
          size: 30,
          content: {
            kind: 'table',
            columns: [
              { id: 'sample', name: 'Sample' },
              { id: 'group', name: 'Group' },
              { id: 'secret', name: 'Unselected' },
            ],
            rows: [
              { key: 'r1', cells: ['S01', 'Control', 'private-1'] },
              { key: 'r2', cells: ['S02', 'Treatment', 'private-2'] },
            ],
            truncated: false,
          },
        },
      },
    ],
    [
      'res_image',
      {
        kind: 'file',
        value: {
          projectId,
          resourceId: 'res_image',
          name: 'plot.png',
          revision,
          size: 1,
          content: {
            kind: 'image',
            mediaType: 'image/png',
            width: 100,
            height: 80,
            base64: 'AA==',
          },
        },
      },
    ],
  ]);
  const resources: WorkspaceResources = {
    projects: async () => [
      { projectId, rootResourceId: 'res_root', name: 'One' },
      { projectId: 'prj_two', rootResourceId: 'res_two', name: 'Two' },
    ],
    read: async (_id, ref) => {
      const found = ref.kind === 'file' && data.get(ref.resourceId);
      if (!found) throw new Error('Resource missing');
      return structuredClone(found);
    },
    describe: async (id, ref) => {
      const value = await resources.read(id, ref);
      if (value.kind !== 'file') throw new Error('Expected file');
      return { title: value.value.name, view: value.value.content.kind };
    },
  };
  const workspace = new WorkspaceController(
    new WorkspaceStorage(join(path, 'workspace')),
    resources,
  );
  await workspace.initialize();
  const session = await workspace.connect();
  await workspace.openProject(projectId);
  const account = {
    sessionId: 'account-one',
    images: true,
    require: () => account.sessionId,
    supportsImages: () => account.images,
  };
  const storage = new EvidenceStorage(join(path, 'evidence'), quota);
  let now = Date.now();
  const evidence = new EvidenceService(
    workspace,
    resources,
    storage,
    renderImage,
    account,
    () => now,
  );
  const sent: { submission: Submission; input: ProviderInput[] }[] = [];
  let beforeStart = async (_submission: Submission) => {};
  const provider: ConversationProvider = {
    bind: async (agent) => agent.provider.threadId ?? 'thread_' + agent.agentId,
    start: async (submission, input = []) => {
      await beforeStart(submission);
      sent.push({ submission: structuredClone(submission), input: structuredClone(input) });
      return {
        id: 'turn_' + submission.requestId,
        status: 'completed',
        clientIds: [submission.requestId],
        messages: [{ id: 'reply', text: 'Done' }],
      };
    },
    interrupt: async () => {},
    read: async () => [],
    onEvent: () => () => {},
    onDisconnected: () => () => {},
    disconnect: async () => {},
  };
  const coordinator = new CollaborationCoordinator(
    workspace,
    provider,
    account,
    () => {},
    () => {},
    undefined,
    evidence,
  );
  const configure = async (name: string, model = 'image-model') =>
    coordinator.configure({
      projectId,
      agentId: name === 'One' || name === 'Two' ? null : 'agt_One',
      requestId: 'req_' + name,
      name,
      configuration: { model, effort: 'low', instructions: '' },
    });
  await configure('One');
  await configure('Two');
  let sequence = 0;
  const command = async (action: WorkspaceAction) =>
    workspace.command({
      projectId,
      expectedRevision: (await workspace.read(projectId)).workspace.revision,
      requestId: 'req_action_' + ++sequence,
      action,
    });
  await command({ kind: 'recipient', agentId: 'agt_One' });
  const attach = async (resourceId: string, selection?: Selection) => {
    const opened = await command({
      kind: 'open',
      resource: { kind: 'file', resourceId },
      pane: 'primary',
      duplicate: false,
    });
    const surfaceId = opened.workspace.layout.primary.activeSurfaceId!;
    await workspace.present({
      projectId,
      rendererSessionId: session.rendererSessionId,
      visiblePanes: ['primary'],
    });
    const load = await workspace.loadSurface({
      projectId,
      surfaceId,
      rendererSessionId: session.rendererSessionId,
    });
    await workspace.acknowledge(load.acknowledgment);
    const ref: EvidenceRef = {
      projectId,
      schemaVersion: 2 as const,
      origin: { surfaceId: surfaceId },
      resource: { kind: 'file', resourceId },
      dataRevision: load.acknowledgment.dataRevision,
      ...(selection ? { selection } : {}),
    };
    const doc = await command({ surfaceId: ref.origin!.surfaceId, kind: 'attach', evidence: ref });
    return { ...doc.chat.attachments!.at(-1)!, evidence: ref };
  };
  const prepare = async () => {
    const doc = await workspace.read(projectId);
    return evidence.prepare({
      projectId,
      agentId: doc.chat.recipientAgentId!,
      attachmentRevision: doc.chat.attachmentRevision ?? 0,
    });
  };
  const send = async (
    preparedEvidenceId: string,
    text = 'Inspect the attachment',
    requestId = 'req_send',
  ) => {
    await workspace.updateDraft(projectId, text);
    return coordinator.send({ projectId, agentId: 'agt_One', requestId, text, preparedEvidenceId });
  };
  return {
    path,
    data,
    resources,
    workspace,
    account,
    storage,
    evidence,
    coordinator,
    sent,
    command,
    attach,
    prepare,
    send,
    configure,
    beforeStart: (callback: typeof beforeStart) => {
      beforeStart = callback;
    },
    advance: () => {
      now += 11 * 60 * 1000;
    },
  };
}

describe('immutable addressed evidence', () => {
  it('sends exact Unicode text and selected table cells, after assets and addressed acceptance are durable', async () => {
    const s = await setup();
    const text = await s.attach('res_text', {
      coordinateSpace: 'utf16-line-column',
      kind: 'text',
      start: { line: 2, column: 5 },
      end: { line: 2, column: 7 },
    });
    const table = await s.attach('res_table', {
      coordinateSpace: 'revision-row-column-keys',
      kind: 'table',
      rowKeys: ['r2'],
      columns: ['sample', 'group'],
    });
    const prepared = await s.prepare();
    expect(
      await s.evidence.preview({
        kind: 'prepared',
        projectId,
        preparedId: prepared.preparedId,
        attachmentId: text.attachmentId,
      }),
    ).toEqual({ kind: 'text', text: '😀' });
    expect(
      await s.evidence.preview({
        kind: 'prepared',
        projectId,
        preparedId: prepared.preparedId,
        attachmentId: table.attachmentId,
      }),
    ).toMatchObject({ kind: 'table', rows: [{ key: 'r2', cells: ['S02', 'Treatment'] }] });
    s.beforeStart(async (submission) => {
      const saved = JSON.parse(
        await readFile(join(s.path, 'workspace/projects/prj_one.json'), 'utf8'),
      );
      expect(saved.chat).toMatchObject({ draft: '', attachments: [] });
      expect(saved.collaboration.submissions[0].state).toBe('submitting');
      for (const item of submission.evidence!)
        await expect(s.storage.read(projectId, item)).resolves.toBeDefined();
    });
    await s.send(prepared.preparedId);
    expect(s.sent).toHaveLength(1);
    expect(s.sent[0]!.input).toHaveLength(2);
    expect(JSON.stringify(s.sent[0]!.input)).not.toContain('private-2');
    expect(JSON.stringify(s.sent[0]!.input)).not.toContain('Control');
    expect(s.sent[0]!.submission.agentId).toBe('agt_One');
    await s.coordinator.send({
      projectId,
      agentId: 'agt_One',
      text: 'Inspect the attachment',
      requestId: 'req_send',
      preparedEvidenceId: prepared.preparedId,
    });
    expect(s.sent).toHaveLength(1);
    await expect(
      s.coordinator.send({
        projectId,
        agentId: 'agt_One',
        text: 'Inspect the attachment',
        requestId: 'req_send',
      }),
    ).rejects.toThrow('another message');
  });
  it('retains sent content after source changes, Surface close and normal store reconstruction', async () => {
    const s = await setup();
    const attachment = await s.attach('res_text');
    const prepared = await s.prepare();
    await s.send(prepared.preparedId, '');
    s.data.set('res_text', textData('Replaced content'));
    await s.command({ kind: 'close', surfaceId: attachment.evidence.origin!.surfaceId });
    const request = {
      kind: 'sent' as const,
      projectId,
      requestId: 'req_send',
      attachmentId: attachment.attachmentId,
    };
    expect(await s.evidence.preview(request)).toEqual({
      kind: 'text',
      text: 'Alpha\nBeta 😀 gamma\n',
    });
    await s.workspace.stop();
    const restored = new WorkspaceController(
      new WorkspaceStorage(join(s.path, 'workspace')),
      s.resources,
    );
    await restored.initialize();
    await restored.connect();
    const evidence = new EvidenceService(restored, s.resources, s.storage, renderImage, s.account);
    expect(await evidence.preview(request)).toEqual({
      kind: 'text',
      text: 'Alpha\nBeta 😀 gamma\n',
    });
    await expect(
      evidence.preview({ ...request, kind: 'prepared', preparedId: prepared.preparedId }),
    ).rejects.toThrow('prepared again');
    expect(s.sent).toHaveLength(1);
  });
  it('keeps local selection changes separate from attachment intent and rejects forged attachment capture', async () => {
    const s = await setup();
    const item = await s.attach('res_text', {
      coordinateSpace: 'utf16-line-column',
      kind: 'text',
      start: { line: 1, column: 0 },
      end: { line: 1, column: 5 },
    });
    const prepared = await s.prepare();
    await s.command({
      kind: 'select',
      surfaceId: item.evidence.origin!.surfaceId,
      evidence: {
        ...item.evidence,
        selection: {
          coordinateSpace: 'utf16-line-column',
          kind: 'text',
          start: { line: 2, column: 0 },
          end: { line: 2, column: 4 },
        },
      },
    });
    expect(
      await s.evidence.preview({
        kind: 'prepared',
        projectId,
        preparedId: prepared.preparedId,
        attachmentId: item.attachmentId,
      }),
    ).toEqual({ kind: 'text', text: 'Alpha' });
    await expect(
      s.command({
        surfaceId: item.evidence.origin!.surfaceId,
        kind: 'attach',
        evidence: { ...item.evidence, projectId: 'prj_two' },
      }),
    ).rejects.toThrow();
    await expect(
      s.command({
        surfaceId: item.evidence.origin!.surfaceId,
        kind: 'attach',
        evidence: {
          ...item.evidence,
          selection: {
            coordinateSpace: 'utf16-line-column',
            kind: 'text',
            start: { line: 2, column: 6 },
            end: { line: 2, column: 7 },
          },
        },
      }),
    ).rejects.toThrow('no longer matches');
  });
  it('invalidates preparation on recipient/model round trips, account change, Project switch and expiry', async () => {
    const s = await setup();
    await s.attach('res_text');
    let prepared = await s.prepare();
    await s.command({ kind: 'recipient', agentId: 'agt_Two' });
    await s.command({ kind: 'recipient', agentId: 'agt_One' });
    await expect(s.send(prepared.preparedId)).rejects.toThrow('prepared again');
    prepared = await s.prepare();
    await s.configure('Renamed', 'text-model');
    await s.configure('Renamed', 'image-model');
    await expect(s.send(prepared.preparedId)).rejects.toThrow('prepared again');
    prepared = await s.prepare();
    s.account.sessionId = 'new-account';
    await expect(s.send(prepared.preparedId)).rejects.toThrow('prepared again');
    prepared = await s.prepare();
    await s.workspace.openProject('prj_two');
    await s.workspace.openProject(projectId);
    await expect(s.send(prepared.preparedId)).rejects.toThrow('prepared again');
    prepared = await s.prepare();
    s.advance();
    await expect(s.send(prepared.preparedId)).rejects.toThrow('prepared again');
    expect(s.sent).toHaveLength(0);
  });
  it('rejects a stale source before delivery, preserving the entire draft and all attachments', async () => {
    const s = await setup();
    await s.attach('res_text');
    await s.attach('res_table');
    const prepared = await s.prepare();
    s.data.set('res_text', textData('New revision'));
    await expect(s.send(prepared.preparedId, 'Keep this draft')).rejects.toThrow(
      'no longer matches',
    );
    const doc = await s.workspace.read(projectId);
    expect(doc.chat.draft).toBe('Keep this draft');
    expect(doc.chat.attachments).toHaveLength(2);
    expect(doc.collaboration?.submissions).toHaveLength(0);
    expect(s.sent).toHaveLength(0);
  });
  it('checks intent again inside the writer after asynchronous source validation', async () => {
    const s = await setup();
    await s.attach('res_text');
    const prepared = await s.prepare();
    const read = s.resources.read;
    s.resources.read = async (...args) => {
      const result = await read(...args);
      await s.workspace.updateDraft(projectId, 'Newer draft');
      return result;
    };
    await expect(s.send(prepared.preparedId)).rejects.toThrow('draft or recipient changed');
    expect((await s.workspace.read(projectId)).chat.draft).toBe('Newer draft');
    expect(s.sent).toHaveLength(0);
  });
  it('rejects foreign recipients/tokens and a text-only send that would silently omit draft attachments', async () => {
    const s = await setup();
    await s.attach('res_text');
    const prepared = await s.prepare();
    await expect(
      s.coordinator.send({
        projectId,
        agentId: 'agt_Two',
        requestId: 'req_foreign',
        text: 'Hello',
        preparedEvidenceId: prepared.preparedId,
      }),
    ).rejects.toThrow('prepared again');
    await expect(
      s.evidence.preview({
        kind: 'prepared',
        projectId: 'prj_two',
        preparedId: prepared.preparedId,
        attachmentId: 'att_other',
      }),
    ).rejects.toThrow('prepared again');
    await expect(
      s.coordinator.send({
        projectId,
        agentId: 'agt_One',
        requestId: 'req_no_token',
        text: 'Hello',
      }),
    ).rejects.toThrow('Prepare the draft attachments');
    expect(s.sent).toHaveLength(0);
  });
  it('rejects unsupported image models without dropping the image and emits actual image input for supported models', async () => {
    const s = await setup();
    const item = await s.attach('res_image', {
      coordinateSpace: 'normalized-original-image',
      kind: 'image',
      x: 0.5,
      y: 0.5,
      width: 0.25,
      height: 0.25,
      originalWidth: 100,
      originalHeight: 80,
      contentHash: revision,
    });
    s.account.images = false;
    await expect(s.prepare()).rejects.toThrow('This model cannot receive images');
    s.account.images = true;
    const prepared = await s.prepare();
    expect(prepared.items[0]?.representation).toMatchObject({
      kind: 'image',
      width: 25,
      height: 20,
      crop: { x: 50, y: 40, width: 25, height: 20 },
    });
    s.account.images = false;
    await expect(s.send(prepared.preparedId)).rejects.toThrow('cannot receive images');
    s.account.images = true;
    await s.send(prepared.preparedId, '');
    expect(s.sent[0]!.input.at(-1)).toEqual({ type: 'image', url: 'data:image/png;base64,AA==' });
    expect(
      (await s.workspace.read(projectId)).collaboration?.submissions[0]?.evidence?.[0]
        ?.attachmentId,
    ).toBe(item.attachmentId);
  });
  it('bounds attachment count, image count and aggregate selected text without truncating a selection', async () => {
    const s = await setup();
    const item = await s.attach('res_text');
    for (let i = 1; i < 16; i++)
      await s.command({
        surfaceId: item.evidence.origin!.surfaceId,
        kind: 'attach',
        evidence: item.evidence,
      });
    await expect(
      s.command({
        surfaceId: item.evidence.origin!.surfaceId,
        kind: 'attach',
        evidence: item.evidence,
      }),
    ).rejects.toThrow('up to 16');
    const t = await setup();
    for (let i = 0; i < 3; i++) await t.attach('res_image');
    await expect(t.prepare()).rejects.toThrow('at most two images');
    const u = await setup();
    u.data.set('res_text', textData('x'.repeat(40000)));
    await u.attach('res_text');
    await u.attach('res_text');
    await expect(u.prepare()).rejects.toThrow('combined text');
    const long = textData('😀'.repeat(20000));
    if (long.kind !== 'file') throw new Error('Wrong fixture');
    await expect(
      materializeEvidence(
        {
          ...item,
          evidence: {
            ...item.evidence,
            dataRevision: long.value.revision,
            selection: {
              coordinateSpace: 'utf16-line-column',
              kind: 'text',
              start: { line: 1, column: 0 },
              end: { line: 1, column: 40000 },
            },
          },
        },
        long,
        renderImage,
      ),
    ).rejects.toThrow('selected text exceeds');
  });
  it('refuses quota exhaustion before send and never evicts earlier assets', async () => {
    const s = await setup(30);
    await s.attach('res_text');
    await expect(s.prepare()).rejects.toThrow('evidence quota');
    expect(s.sent).toHaveLength(0);
    expect((await s.workspace.read(projectId)).chat.attachments).toHaveLength(1);
  });
  it('preserves the draft when asset publication or Project acceptance fails', async () => {
    const s = await setup();
    await s.attach('res_text');
    const prepared = await s.prepare();
    const put = s.storage.put.bind(s.storage);
    s.storage.put = async () => {
      throw new Error('Simulated disk failure');
    };
    await expect(s.send(prepared.preparedId, 'Saved draft')).rejects.toThrow('disk failure');
    expect(s.sent).toHaveLength(0);
    expect((await s.workspace.read(projectId)).chat.draft).toBe('Saved draft');
    s.storage.put = put;
    const configPath = join(s.path, 'workspace/projects/prj_one.json');
    const original = await readFile(configPath, 'utf8');
    s.resources.read = async (...args) => {
      await writeFile(configPath, original + ' ');
      const ref = args[1];
      return structuredClone(s.data.get(ref.kind === 'file' ? ref.resourceId : '')!);
    };
    await expect(s.send(prepared.preparedId, 'Saved draft')).rejects.toThrow('could not be saved');
    expect(s.sent).toHaveLength(0);
    expect((await s.workspace.read(projectId)).chat.draft).toBe('Saved draft');
  });
  it('detects corrupt/missing historical assets and refuses symlinks without source fallback', async () => {
    const s = await setup();
    const item = await s.attach('res_text');
    const prepared = await s.prepare();
    await s.send(prepared.preparedId);
    const asset = prepared.items[0]!;
    const path = join(s.path, 'evidence', projectId, asset.asset.hash.slice(7) + '.blob');
    const input = {
      kind: 'sent' as const,
      projectId,
      requestId: 'req_send',
      attachmentId: item.attachmentId,
    };
    const bytes = await readFile(path);
    await writeFile(path, Buffer.alloc(bytes.length, 65));
    await expect(s.evidence.preview(input)).rejects.toThrow('Evidence unavailable');
    await rm(path);
    await expect(s.evidence.preview(input)).rejects.toThrow('Evidence unavailable');
    const outside = join(s.path, 'unrelated');
    await writeFile(outside, bytes);
    await symlink(outside, path);
    await expect(s.evidence.preview(input)).rejects.toThrow('Evidence unavailable');
    expect(await readFile(outside)).toEqual(bytes);
  });
  it('deduplicates bytes while retaining distinct attachment manifests and validates saved associations', async () => {
    const s = await setup();
    await s.attach('res_text');
    await s.attach('res_text');
    const prepared = await s.prepare();
    await s.send(prepared.preparedId);
    expect(await readdir(join(s.path, 'evidence', projectId))).toHaveLength(1);
    const doc = await s.workspace.read(projectId);
    expect(doc.collaboration!.submissions[0]!.evidence).toHaveLength(2);
    doc.collaboration!.submissions[0]!.evidence![0]!.evidence.projectId = 'prj_two';
    expect(() => parseWorkspaceDocument(doc)).toThrow('another Project');
    expect(() =>
      evidencePreview({
        manifest: prepared.items[0]!,
        bytes: Buffer.from('{"kind":"image","base64":"AA=="}'),
      }),
    ).toThrow();
  });
});
