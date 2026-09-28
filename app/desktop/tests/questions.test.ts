import { mkdtemp, readFile, rm, writeFile } from 'node:fs/promises';
import { join } from 'node:path';
import { tmpdir } from 'node:os';
import { afterEach, describe, expect, it } from 'vitest';
import {
  isQuestion,
  parse,
  parseWorkspaceDocument,
  QuestionInputSchema,
  WorkspaceDocumentV1Schema,
  type SurfaceData,
  type WorkspaceAction,
  type Submission,
  type AgentAttachment,
  type Question,
} from '@gobble/contracts';
import { WorkspaceController } from '../src/main/workspace/controller';
import { WorkspaceStorage } from '../src/main/workspace/storage';
import type { WorkspaceResources } from '../src/main/workspace/service';
import { EvidenceStorage } from '../src/main/evidence/storage';
import { EvidenceService } from '../src/main/evidence/service';
import { contentHash } from '../src/main/evidence/materialize';
import { QuestionService } from '../src/main/questions/service';
import { CollaborationCoordinator } from '../src/main/collaboration/coordinator';
import type { ConversationProvider, ProviderInput } from '../src/main/collaboration/provider';
import type { ToolContext } from '../src/main/shared-context/ports';

const projectId = 'prj_one';
const paths: string[] = [];
afterEach(async () => {
  for (const path of paths.splice(0)) await rm(path, { recursive: true, force: true });
});
const textData = (text: string): SurfaceData => ({
  kind: 'file',
  value: {
    projectId,
    resourceId: 'res_note',
    name: 'notes.txt',
    revision: contentHash(Buffer.from(text)),
    size: Buffer.byteLength(text),
    content: { kind: 'text', text },
  },
});
async function setup() {
  const path = await mkdtemp(join(tmpdir(), 'gobble-questions-'));
  paths.push(path);
  let data = textData('Reference: K9\n');
  const resources: WorkspaceResources = {
    projects: async () => [
      { projectId, name: 'One', rootResourceId: 'res_root' },
      { projectId: 'prj_two', name: 'Two', rootResourceId: 'res_two' },
    ],
    describe: async () => ({ title: 'notes.txt', view: 'text' }),
    read: async () => structuredClone(data),
  };
  const workspace = new WorkspaceController(
    new WorkspaceStorage(join(path, 'workspace')),
    resources,
    () => true,
  );
  await workspace.initialize();
  const session = await workspace.connect();
  await workspace.openProject(projectId);
  const storage = new EvidenceStorage(join(path, 'evidence'));
  const account = {
    connected: true,
    images: true,
    require: () => {
      if (!account.connected) throw new Error('Offline');
      return 'session-one';
    },
    supportsImages: () => account.images,
  };
  const renderImage = async () => ({
    url: 'data:image/png;base64,AA==',
    width: 100,
    height: 80,
    crop: { x: 0, y: 0, width: 100, height: 80 },
  });
  const evidence = new EvidenceService(workspace, resources, storage, renderImage, account);
  const questions = new QuestionService(workspace, resources, storage, account);
  const sent: { submission: Submission; input: ProviderInput[] }[] = [];
  let beforeStart = async (_submission: Submission) => {};
  const provider: ConversationProvider = {
    bind: async (agent) => agent.provider.threadId ?? 'thread_' + agent.agentId,
    start: async (submission, input = []) => {
      await beforeStart(submission);
      sent.push({ submission: structuredClone(submission), input });
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
    questions,
  );
  for (const name of ['One', 'Two'])
    await coordinator.configure({
      projectId,
      agentId: null,
      requestId: 'req_' + name,
      name,
      configuration: { model: 'image-model', effort: 'low', instructions: '' },
    });
  let sequence = 0;
  const command = async (action: WorkspaceAction) =>
    workspace.command({
      projectId,
      action,
      expectedRevision: (await workspace.read(projectId)).workspace.revision,
      requestId: 'req_action_' + ++sequence,
    });
  await command({ kind: 'recipient', agentId: 'agt_One' });
  const attach = async () => {
    const doc = await command({
      kind: 'open',
      resource: { kind: 'file', resourceId: 'res_note' },
      pane: 'primary',
      duplicate: false,
    });
    const surfaceId = doc.workspace.layout.primary.activeSurfaceId!;
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
    return (
      await command({
        kind: 'attach',
        surfaceId,
        evidence: {
          projectId,
          schemaVersion: 2 as const,
          origin: { surfaceId: surfaceId },
          resource: { kind: 'file', resourceId: 'res_note' },
          dataRevision: load.acknowledgment.dataRevision,
        },
      })
    ).chat.attachments!.at(-1)!;
  };
  const prepare = async () => {
    const doc = await workspace.read(projectId);
    return evidence.prepare({
      projectId,
      agentId: doc.chat.recipientAgentId!,
      attachmentRevision: doc.chat.attachmentRevision!,
    });
  };
  const attachment = await attach();
  const prepared = await prepare();
  await workspace.updateDraft(projectId, 'Inspect');
  await coordinator.send({
    projectId,
    agentId: 'agt_One',
    requestId: 'req_origin',
    text: 'Inspect',
    preparedEvidenceId: prepared.preparedId,
  });
  const read = () => workspace.read(projectId);
  const doc = await read();
  const agent = doc.workspace.agents[0] as AgentAttachment;
  const submission = doc.collaboration!.submissions[0]!;
  let authorized = true;
  const guard = () => {
    if (!authorized) throw new Error('Revoked');
  };
  const context: ToolContext = {
    agent,
    submission,
    signal: new AbortController().signal,
    assert: guard,
  };
  questions.begin(projectId, agent.agentId, submission.requestId);
  const create = async (
    callId = 'call_one',
    evidenceIds = [attachment.attachmentId],
    question = 'Should we retain K9?',
  ) =>
    questions.create(
      context,
      {
        threadId: submission.threadId,
        turnId: submission.turnId!,
        callId,
        tool: 'workspace_question',
        arguments: { question, options: ['Keep', 'Review'], evidenceIds },
      },
      guard,
    );
  const reply = async (
    questionId: string,
    requestId = 'req_answer',
    text = 'Keep K9',
    preparedEvidenceId?: string,
  ) => {
    await workspace.updateDraft(projectId, text);
    return coordinator.send({
      projectId,
      agentId: 'agt_One',
      requestId,
      text,
      replyToQuestionId: questionId,
      ...(preparedEvidenceId ? { preparedEvidenceId } : {}),
    });
  };
  const question = async (id?: string): Promise<Question> => {
    const items = (await read()).workspace.decisions.filter(isQuestion);
    return id ? items.find((item) => item.decisionId === id)! : items[0]!;
  };
  return {
    path,
    resources,
    workspace,
    storage,
    account,
    evidence,
    questions,
    coordinator,
    provider,
    sent,
    context,
    attachment,
    command,
    attach,
    prepare,
    create,
    reply,
    read,
    question,
    data: () => data,
    setData: (value: SurfaceData) => {
      data = value;
    },
    revoke: () => {
      authorized = false;
    },
    beforeStart: (fn: typeof beforeStart) => {
      beforeStart = fn;
    },
  };
}

describe('durable questions and explicit replies', () => {
  it('creates an ordinary question without changing draft/recipient; persists idempotent evidence before its record', async () => {
    const s = await setup();
    await s.command({ kind: 'recipient', agentId: 'agt_Two' });
    await s.workspace.updateDraft(projectId, 'Unsent peer note');
    const created = await s.create();
    expect(await s.create()).toEqual(created);
    expect((await s.read()).chat).toMatchObject({
      draft: 'Unsent peer note',
      recipientAgentId: 'agt_Two',
    });
    const q = await s.question();
    expect(q.state.kind).toBe('pending');
    expect((await s.read()).workspace.decisions).toHaveLength(1);
    await expect(s.storage.read(projectId, q.evidence[0]!)).resolves.toBeDefined();
    await expect(s.create('call_one', undefined, 'Changed question')).rejects.toThrow(
      'different arguments',
    );
    expect(s.sent).toHaveLength(1);
  });
  it('allows only turn-observed or this Agent’s received evidence, not a peer, draft, or fabricated ID', async () => {
    const s = await setup();
    const observed = await s.questions.observe(
      s.context,
      s.attachment.evidence,
      s.data(),
      async () => {
        throw new Error('Not image');
      },
      s.context.assert,
    );
    await s.create('observation', [observed]);
    await expect(s.create('forged', ['obs_foreign'])).rejects.toThrow('not observed or received');
    const draft = await s.attach();
    await expect(s.create('draft', [draft.attachmentId])).rejects.toThrow(
      'not observed or received',
    );
    const peer = { ...s.context, agent: { ...s.context.agent, agentId: 'agt_Two' } };
    await expect(
      s.questions.create(
        peer,
        {
          threadId: 'thread_agt_Two',
          turnId: 'turn_other',
          callId: 'peer',
          tool: 'workspace_question',
          arguments: { question: 'Question', evidenceIds: [s.attachment.attachmentId] },
        },
        peer.assert,
      ),
    ).rejects.toThrow('not observed or received');
    s.questions.release(projectId, s.context.agent.agentId, s.context.submission.requestId);
    await expect(s.create('expired', [observed])).rejects.toThrow('not observed or received');
  });
  it('stores answer link, submission and matching draft consumption together before input; duplicate send never replays', async () => {
    const s = await setup();
    const { questionId } = await s.create();
    await s.command({ kind: 'reply', questionId });
    s.beforeStart(async (submission) => {
      const saved = parseWorkspaceDocument(
        JSON.parse(await readFile(join(s.path, 'workspace/projects/prj_one.json'), 'utf8')),
      );
      expect(saved.chat.draft).toBe('');
      expect(saved.chat.replyToQuestionId).toBeUndefined();
      expect(saved.workspace.decisions.filter(isQuestion)[0]?.state).toMatchObject({
        kind: 'answered',
        submissionId: submission.requestId,
      });
      expect(saved.collaboration?.submissions.at(-1)?.state).toBe('submitting');
    });
    await s.reply(questionId);
    const q = await s.question();
    expect(q.state.kind).toBe('answered');
    expect(JSON.stringify(s.sent[1]!.input)).toContain('Reference: K9');
    expect(s.sent[1]!.submission.agentId).toBe('agt_One');
    await s.coordinator.send({
      projectId,
      agentId: 'agt_One',
      requestId: 'req_answer',
      text: 'Keep K9',
      replyToQuestionId: questionId,
    });
    expect(s.sent).toHaveLength(2);
    await expect(
      s.coordinator.send({
        projectId,
        agentId: 'agt_One',
        requestId: 'req_answer',
        text: 'Keep K9',
      }),
    ).rejects.toThrow('another message');
    await expect(s.reply(questionId, 'req_second')).rejects.toThrow('already answered');
  });
  it('keeps text and attachments across Reply/Cancel, pins recipient and rejects an unlinked or wrong-recipient send', async () => {
    const s = await setup();
    const { questionId } = await s.create();
    const attachment = await s.attach();
    await s.command({ kind: 'recipient', agentId: 'agt_Two' });
    await s.workspace.updateDraft(projectId, 'My draft');
    const prepared = await s.prepare();
    await s.command({ kind: 'reply', questionId });
    expect((await s.read()).chat).toMatchObject({
      draft: 'My draft',
      recipientAgentId: 'agt_One',
      attachments: [attachment],
    });
    await expect(s.command({ kind: 'recipient', agentId: 'agt_Two' })).rejects.toThrow(
      'Cancel the reply',
    );
    await expect(
      s.evidence.preview({
        kind: 'prepared',
        projectId,
        preparedId: prepared.preparedId,
        attachmentId: attachment.attachmentId,
      }),
    ).rejects.toThrow('prepared again');
    await expect(
      s.coordinator.send({
        projectId,
        agentId: 'agt_Two',
        requestId: 'req_wrong',
        text: 'My draft',
        replyToQuestionId: questionId,
      }),
    ).rejects.toThrow('Agent that asked');
    const current = await s.prepare();
    await expect(
      s.coordinator.send({
        projectId,
        agentId: 'agt_One',
        requestId: 'req_unlinked',
        text: 'My draft',
        preparedEvidenceId: current.preparedId,
      }),
    ).rejects.toThrow('reply target');
    await s.command({ kind: 'cancelReply' });
    expect((await s.read()).chat).toMatchObject({ draft: 'My draft', attachments: [attachment] });
    expect((await s.question()).state.kind).toBe('pending');
  });
  it('invalidates a changed dependency at Send, keeps all draft state and previews the exact original after source replacement and Surface close', async () => {
    const s = await setup();
    const { questionId } = await s.create();
    await s.command({ kind: 'reply', questionId });
    s.setData(textData('Replaced source'));
    await expect(s.reply(questionId)).rejects.toThrow('evidence changed');
    expect((await s.question()).state.kind).toBe('invalidated');
    expect((await s.read()).chat).toMatchObject({
      draft: 'Keep K9',
      replyToQuestionId: questionId,
    });
    if (s.attachment.evidence.schemaVersion === 4) throw new Error('File fixture required');
    await s.command({ kind: 'close', surfaceId: s.attachment.evidence.origin!.surfaceId });
    expect(
      await s.evidence.preview({
        kind: 'question',
        projectId,
        questionId,
        attachmentId: s.attachment.attachmentId,
      }),
    ).toEqual({ kind: 'text', text: 'Reference: K9\n' });
    expect(s.sent).toHaveLength(1);
  });
  it('preserves pending state when busy/offline and records the answer even if delivery becomes uncertain', async () => {
    const s = await setup();
    const { questionId } = await s.create();
    await s.command({ kind: 'reply', questionId });
    s.account.connected = false;
    await expect(s.reply(questionId)).rejects.toThrow('Offline');
    expect((await s.question()).state.kind).toBe('pending');
    s.account.connected = true;
    await s.workspace.changeCollaboration(projectId, (current) => {
      current.history.submissions[0]!.state = 'uncertain';
      return current;
    });
    await expect(s.reply(questionId)).rejects.toThrow('pending message');
    expect((await s.question()).state.kind).toBe('pending');
    await s.workspace.changeCollaboration(projectId, (current) => {
      current.history.submissions[0]!.state = 'completed';
      return current;
    });
    s.beforeStart(async () => {
      throw new Error('Acknowledgment lost');
    });
    await expect(s.reply(questionId)).rejects.toThrow('Acknowledgment lost');
    expect((await s.question()).state).toMatchObject({
      kind: 'answered',
      submissionId: 'req_answer',
    });
    expect((await s.read()).collaboration?.submissions.at(-1)?.state).toBe('uncertain');
    await s.coordinator.send({
      projectId,
      agentId: 'agt_One',
      requestId: 'req_answer',
      text: 'Keep K9',
      replyToQuestionId: questionId,
    });
    expect(s.sent).toHaveLength(1);
  });
  it('preserves pending question and unsent reply through normal restart without provider input or tool replay', async () => {
    const s = await setup();
    const { questionId } = await s.create();
    await s.command({ kind: 'reply', questionId });
    await s.workspace.updateDraft(projectId, 'Unsent answer');
    await s.coordinator.stop();
    await s.workspace.stop();
    const restored = new WorkspaceController(
      new WorkspaceStorage(join(s.path, 'workspace')),
      s.resources,
    );
    await restored.initialize();
    const bootstrap = await restored.connect();
    expect(bootstrap.document?.chat).toMatchObject({
      draft: 'Unsent answer',
      replyToQuestionId: questionId,
    });
    expect(bootstrap.document?.workspace.decisions.filter(isQuestion)[0]?.state.kind).toBe(
      'pending',
    );
    expect(s.sent).toHaveLength(1);
  });
  it('checks Reply/Cancel round trips, edits and dismissal inside the writer after dependency I/O', async () => {
    for (const change of ['edit', 'roundtrip', 'dismiss', 'project'] as const) {
      const s = await setup();
      const { questionId } = await s.create();
      await s.command({ kind: 'reply', questionId });
      const read = s.resources.read;
      s.resources.read = async (...args) => {
        const data = await read(...args);
        s.resources.read = read;
        if (change === 'edit') await s.workspace.updateDraft(projectId, 'Newer draft');
        if (change === 'roundtrip') {
          await s.command({ kind: 'cancelReply' });
          await s.command({ kind: 'reply', questionId });
        }
        if (change === 'dismiss') await s.command({ kind: 'dismissQuestion', questionId });
        if (change === 'project') {
          await s.workspace.openProject('prj_two');
          await s.workspace.openProject(projectId);
        }
        return data;
      };
      await expect(s.reply(questionId)).rejects.toThrow();
      expect(s.sent).toHaveLength(1);
      expect((await s.question()).state.kind).toBe(change === 'dismiss' ? 'dismissed' : 'pending');
      expect((await s.read()).chat.draft).toBe(change === 'edit' ? 'Newer draft' : 'Keep K9');
    }
  });
  it('keeps the draft and pending question if the single Project acceptance fails', async () => {
    const s = await setup();
    const { questionId } = await s.create();
    await s.command({ kind: 'reply', questionId });
    await s.workspace.updateDraft(projectId, 'Keep K9');
    const file = join(s.path, 'workspace/projects/prj_one.json');
    const original = await readFile(file, 'utf8');
    const read = s.resources.read;
    s.resources.read = async (...args) => {
      await writeFile(file, original + ' ');
      return read(...args);
    };
    await expect(s.reply(questionId)).rejects.toThrow('could not be saved');
    expect((await s.question()).state.kind).toBe('pending');
    expect((await s.read()).chat.draft).toBe('Keep K9');
    expect(s.sent).toHaveLength(1);
  });
  it('does not answer on dismissal or ordinary unlinked messages and preserves an unavailable reply link', async () => {
    const s = await setup();
    const first = await s.create();
    const second = await s.create('second');
    await s.coordinator.send({
      projectId,
      agentId: 'agt_One',
      requestId: 'req_plain',
      text: 'Keep',
    });
    expect((await s.question(first.questionId)).state.kind).toBe('pending');
    await s.command({ kind: 'reply', questionId: second.questionId });
    await s.command({ kind: 'dismissQuestion', questionId: second.questionId });
    await expect(s.reply(second.questionId)).rejects.toThrow('already answered or dismissed');
    expect((await s.read()).chat.replyToQuestionId).toBe(second.questionId);
    expect(s.sent).toHaveLength(2);
    await s.command({ kind: 'cancelReply' });
    await s.command({ kind: 'reply', questionId: first.questionId });
    await s.reply(first.questionId);
    expect((await s.question(second.questionId)).state.kind).toBe('dismissed');
  });
  it('rejects missing historical evidence, foreign previews and malformed answer associations without fresh-source fallback', async () => {
    const s = await setup();
    const { questionId } = await s.create();
    await s.command({ kind: 'reply', questionId });
    await expect(
      s.evidence.preview({ kind: 'question', projectId, questionId, attachmentId: 'att_foreign' }),
    ).rejects.toThrow('no such attachment');
    const q = await s.question();
    await rm(join(s.path, 'evidence', projectId, q.evidence[0]!.asset.hash.slice(7) + '.blob'));
    await expect(s.reply(questionId)).rejects.toThrow('Evidence unavailable');
    expect((await s.question()).state.kind).toBe('pending');
    const doc = await s.read();
    const record = doc.workspace.decisions.filter(isQuestion)[0]!;
    record.state = { kind: 'answered', submissionId: 'req_origin', answeredAt: Date.now() };
    expect(() => parseWorkspaceDocument(doc)).toThrow('addressed submission');
  });
  it('validates nonempty bounded questions/options/IDs and refuses storage failure or revoked creation', async () => {
    const s = await setup();
    for (const argumentsValue of [
      { question: 'Q', evidenceIds: [] },
      {
        question: 'Q',
        evidenceIds: ['att_x'],
        options: Array.from({ length: 7 }, (_, i) => String(i)),
      },
      { question: 'Q', evidenceIds: ['att_x'], projectId: 'prj_two' },
    ])
      expect(() => parse(QuestionInputSchema, argumentsValue)).toThrow();
    await expect(s.create('blank', undefined, '  ')).rejects.toThrow('nonempty');
    s.storage.put = async () => {
      throw new Error('Disk full');
    };
    await expect(s.create()).rejects.toThrow('Disk full');
    expect((await s.read()).workspace.decisions).toHaveLength(0);
    s.revoke();
    await expect(s.create()).rejects.toThrow('Revoked');
  });
  it('freezes the legacy document reader and rejects question metadata in v1', async () => {
    const s = await setup();
    await s.create();
    const doc = await s.read();
    const { chat, paneOrientation: _orientation, ...rest } = doc;
    expect(() =>
      parse(WorkspaceDocumentV1Schema, {
        ...rest,
        schemaVersion: 1,
        discussion: {
          collapsed: false,
          height: 200,
          draft: chat.draft,
          recipientAgentId: chat.recipientAgentId,
        },
      }),
    ).toThrow();
  });
  it('bounds combined question/new attachment text and preserves evidence on an unsupported image model', async () => {
    const s = await setup();
    s.setData(textData('x'.repeat(40000)));
    const attached = await s.attach();
    const prepared = await s.prepare();
    await s.workspace.updateDraft(projectId, 'Long evidence');
    await s.coordinator.send({
      projectId,
      agentId: 'agt_One',
      requestId: 'req_long',
      text: 'Long evidence',
      preparedEvidenceId: prepared.preparedId,
    });
    const created = await s.create('long', [attached.attachmentId]);
    await s.command({ kind: 'reply', questionId: created.questionId });
    await s.attach();
    const extra = await s.prepare();
    await expect(
      s.reply(created.questionId, 'req_large', 'Answer', extra.preparedId),
    ).rejects.toThrow('64 KiB');
    expect((await s.question(created.questionId)).state.kind).toBe('pending');
    await s.command({ kind: 'cancelReply' });
    for (const item of (await s.read()).chat.attachments ?? [])
      await s.command({ kind: 'detach', attachmentId: item.attachmentId });
    const image: SurfaceData = {
      kind: 'file',
      value: {
        projectId,
        resourceId: 'res_note',
        name: 'plot.png',
        revision: contentHash(Buffer.from('png')),
        size: 1,
        content: { kind: 'image', width: 100, height: 80, mediaType: 'image/png', base64: 'AA==' },
      },
    };
    s.setData(image);
    const ref = { ...s.attachment.evidence, dataRevision: image.value.revision };
    const id = await s.questions.observe(
      s.context,
      ref,
      image,
      async () => ({
        url: 'data:image/png;base64,AA==',
        width: 100,
        height: 80,
        crop: { x: 0, y: 0, width: 100, height: 80 },
      }),
      s.context.assert,
    );
    const imageQuestion = await s.create('image', [id]);
    await s.command({ kind: 'reply', questionId: imageQuestion.questionId });
    s.account.images = false;
    await expect(s.reply(imageQuestion.questionId, 'req_image')).rejects.toThrow(
      'question’s images',
    );
    expect((await s.read()).chat.draft).toBe('Keep K9');
  });
  it('bounds concurrent observation capture and durable question capacity without losing earlier questions', async () => {
    const s = await setup();
    const observe = () =>
      s.questions.observe(
        s.context,
        s.attachment.evidence,
        s.data(),
        async () => {
          throw new Error('Not an image');
        },
        s.context.assert,
      );
    for (let i = 0; i < 15; i++) await observe();
    const concurrent = await Promise.allSettled([observe(), observe(), observe(), observe()]);
    expect(concurrent.filter((item) => item.status === 'fulfilled')).toHaveLength(1);
    const created = await s.create();
    const original = await s.question();
    await s.workspace.changeQuestions(
      projectId,
      () => {},
      () =>
        Array.from({ length: 100 }, (_, index) => ({
          ...structuredClone(original),
          decisionId: index === 0 ? created.questionId : 'dec_capacity_' + index,
          invocation: { ...original.invocation, identity: index.toString(16).padStart(64, '0') },
        })),
    );
    await expect(s.create('excess')).rejects.toThrow('100 questions');
    expect((await s.read()).workspace.decisions).toHaveLength(100);
  });
  it('authorizes question evidence explicitly delivered in a reply to a new conversation for the same Agent', async () => {
    const s = await setup();
    const observed = await s.questions.observe(
      s.context,
      s.attachment.evidence,
      s.data(),
      async () => {
        throw new Error('Not image');
      },
      s.context.assert,
    );
    const first = await s.create('first_observed', [observed]);
    await s.coordinator.agentAction({ projectId, agentId: 'agt_One', action: 'newConversation' });
    s.provider.bind = async () => 'thread_new';
    await s.command({ kind: 'reply', questionId: first.questionId });
    await s.reply(first.questionId);
    const doc = await s.read();
    const question = await s.question();
    const context: ToolContext = {
      ...s.context,
      agent: doc.workspace.agents[0]!,
      submission: doc.collaboration!.submissions.at(-1)!,
    };
    const result = await s.questions.create(
      context,
      {
        threadId: 'thread_new',
        turnId: context.submission.turnId!,
        callId: 'followup',
        tool: 'workspace_question',
        arguments: {
          question: 'Keep this same evidence for the follow-up?',
          evidenceIds: [question.evidence[0]!.attachmentId],
        },
      },
      context.assert,
    );
    expect(result.state).toBe('pending');
    expect((await s.question(result.questionId)).evidence[0]!.asset.hash).toBe(
      question.evidence[0]!.asset.hash,
    );
  });
});
