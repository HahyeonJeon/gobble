import { mkdtemp, rm, readFile } from 'node:fs/promises';
import { join } from 'node:path';
import { tmpdir } from 'node:os';
import { afterEach, describe, expect, it } from 'vitest';
import type {
  SharedToolExecutor,
  ToolHandler,
  ToolContext,
} from '../src/main/shared-context/ports';
import type { AgentAttachment, Submission } from '@gobble/contracts';
import { CollaborationCoordinator } from '../src/main/collaboration/coordinator';
import type {
  ConversationProvider,
  ProviderEvent,
  ProviderTurn,
} from '../src/main/collaboration/provider';
import { WorkspaceController } from '../src/main/workspace/controller';
import { WorkspaceStorage } from '../src/main/workspace/storage';
import type { WorkspaceResources } from '../src/main/workspace/service';

const paths: string[] = [];
afterEach(async () => {
  await Promise.all(paths.splice(0).map((path) => rm(path, { recursive: true, force: true })));
});
class Provider implements ConversationProvider {
  toolHandler: ToolHandler | undefined;
  onTool(handler: ToolHandler) {
    this.toolHandler = handler;
    return () => {
      this.toolHandler = undefined;
    };
  }

  sent: Submission[] = [];
  interrupted: string[] = [];
  listeners = new Set<(event: ProviderEvent) => void>();
  disconnected = new Set<() => void>();
  history: ProviderTurn[] = [];
  fail = false;
  beforeStart: (submission: Submission) => Promise<void> = async () => {};
  bind = async (agent: AgentAttachment) => agent.provider.threadId ?? 'thread_' + agent.agentId;
  start = async (submission: Submission): Promise<ProviderTurn> => {
    await this.beforeStart(submission);
    this.sent.push(structuredClone(submission));
    if (this.fail) throw new Error('Lost acknowledgment.');
    return {
      id: 'turn_' + submission.requestId,
      status: 'inProgress',
      clientIds: [submission.requestId],
      messages: [],
    };
  };
  interrupt = async (_threadId: string, turnId: string) => {
    this.interrupted.push(turnId);
  };
  read = async () => structuredClone(this.history);
  onEvent = (listener: (event: ProviderEvent) => void) => {
    this.listeners.add(listener);
    return () => this.listeners.delete(listener);
  };
  onDisconnected = (listener: () => void) => {
    this.disconnected.add(listener);
    return () => this.disconnected.delete(listener);
  };
  disconnect = async () => {
    for (const listener of this.disconnected) listener();
  };
  emit(event: ProviderEvent) {
    for (const listener of this.listeners) listener(event);
  }
}
async function setup(sharedTools?: SharedToolExecutor) {
  const path = await mkdtemp(join(tmpdir(), 'gobble-collaboration-'));
  paths.push(path);
  const resources: WorkspaceResources = {
    projects: async () => [
      { projectId: 'prj_one', rootResourceId: 'res_one', name: 'One' },
      { projectId: 'prj_two', rootResourceId: 'res_two', name: 'Two' },
    ],
    read: async () => {
      throw new Error('Files are not agent input.');
    },
    describe: async () => {
      throw new Error('Files are not agent input.');
    },
  };
  const store = new WorkspaceController(new WorkspaceStorage(path), resources);
  await store.initialize();
  await store.connect();
  await store.openProject('prj_one');
  const provider = new Provider();
  const account = {
    session: 'session_one',
    require() {
      return this.session;
    },
  };
  const problems: string[] = [];
  const coordinator = new CollaborationCoordinator(
    store,
    provider,
    account,
    () => {},
    (message) => problems.push(message),
    sharedTools,
  );
  const attach = (name: string) =>
    coordinator.configure({
      projectId: 'prj_one',
      agentId: null,
      requestId: 'req_' + name,
      name,
      configuration: { model: 'fixture', effort: 'low', instructions: '' },
    });
  await attach('One');
  await attach('Two');
  const message = (name: string, request = name) => ({
    projectId: 'prj_one',
    agentId: 'agt_' + name,
    requestId: 'req_message_' + request,
    text: 'Hello ' + name,
  });
  return {
    path,
    store,
    resources,
    coordinator,
    provider,
    account,
    attach,
    message,
    problems,
    submissions: async () => (await store.readCollaboration('prj_one')).history.submissions,
  };
}
describe('Project collaboration ownership and delivery', () => {
  it('commits first response order across delayed peers and retains it through later text and restoration', async () => {
    const { coordinator, provider, message, submissions, path } = await setup();
    await coordinator.send(message('One'));
    await coordinator.send(message('Two'));
    expect((await submissions()).map((item) => item.responseStartedAt)).toEqual([null, null]);
    const delta = (name: string, text: string) =>
      provider.emit({
        kind: 'delta',
        threadId: 'thread_agt_' + name,
        turnId: 'turn_req_message_' + name,
        itemId: 'answer',
        text,
      });
    delta('Two', 'First to arrive');
    await expect.poll(async () => (await submissions())[1]?.responseStartedAt).toBeTypeOf('number');
    delta('One', 'Second to arrive');
    await expect.poll(async () => (await submissions())[0]?.responseStartedAt).toBeTypeOf('number');
    const times = (await submissions()).map((item) => item.responseStartedAt!);
    expect(times[1]).toBeLessThan(times[0]!);
    delta('Two', ' and more text');
    await coordinator.stop();
    const saved = JSON.parse(await readFile(join(path, 'projects/prj_one.json'), 'utf8'));
    expect(
      saved.collaboration.submissions.map((item: Submission) => item.responseStartedAt),
    ).toEqual(times);
    expect(provider.sent).toHaveLength(2);
  });
  it('does not manufacture historical response times when reconciling a legacy submission', async () => {
    const { coordinator, provider, message, store, submissions } = await setup();
    await coordinator.send(message('One'));
    await store.changeCollaboration('prj_one', (current) => {
      delete current.history.submissions[0]!.responseStartedAt;
      return current;
    });
    provider.history = [
      {
        id: 'turn_req_message_One',
        status: 'completed',
        clientIds: ['req_message_One'],
        messages: [{ id: 'answer', text: 'Recovered legacy response' }],
      },
    ];
    await coordinator.agentAction({
      projectId: 'prj_one',
      agentId: 'agt_One',
      action: 'reconcile',
    });
    expect((await submissions())[0]).not.toHaveProperty('responseStartedAt');
    await coordinator.stop();
  });
  it.each(['sharedViews', 'messages'] as const)(
    'requires explicit renewal of an old tool binding even with %s access',
    async (access) => {
      const { coordinator, provider, store, message, submissions } = await setup();
      await store.changeCollaboration('prj_one', (current) => {
        const agent = current.agents[0]!;
        agent.access = access;
        agent.provider = {
          kind: 'codex',
          threadId: 'old',
          accountSessionId: 'session_one',
          toolsetVersion: 'shared-views-v1',
        };
        return current;
      });
      provider.bind = async () => {
        throw new Error('Unexpected provider resume');
      };
      await expect(coordinator.send(message('One'))).rejects.toThrow(
        'Shared-view tools have changed.',
      );
      expect(await submissions()).toHaveLength(0);
      expect(provider.sent).toHaveLength(0);
      await coordinator.stop();
    },
  );
  it('keeps a queued terminal outcome when transport loss follows it immediately', async () => {
    const { coordinator, provider, message, submissions } = await setup();
    await coordinator.send(message('One'));
    provider.emit({
      kind: 'turn',
      threadId: 'thread_agt_One',
      turn: {
        id: 'turn_req_message_One',
        status: 'completed',
        clientIds: [],
        messages: [{ id: 'answer', text: 'Finished' }],
      },
    });
    await provider.disconnect();
    await coordinator.agentAction({
      projectId: 'prj_one',
      agentId: 'agt_One',
      action: 'reconcile',
    });
    expect((await submissions())[0]).toMatchObject({ state: 'completed', response: 'Finished' });
  });
  it('retains ownership of a turn when saving its acknowledgment fails, so quit still interrupts it', async () => {
    const { coordinator, provider, message, path } = await setup();
    provider.beforeStart = async () => {
      await rm(join(path, 'projects/prj_one.json'));
    };
    await expect(coordinator.send(message('One'))).rejects.toThrow('storage is unavailable');
    expect(coordinator.hasActive()).toBe(true);
    await expect(coordinator.stop()).rejects.toThrow('storage is unavailable');
    expect(provider.interrupted).toEqual(['turn_req_message_One']);
  });
  it('saves acceptance and consumes the exact draft before provider submission; duplicate requests do not resend', async () => {
    const { store, coordinator, provider, message, path } = await setup();
    const input = message('One');
    await store.updateDraft('prj_one', input.text);
    provider.beforeStart = async (submission) => {
      const saved = JSON.parse(await readFile(join(path, 'projects/prj_one.json'), 'utf8'));
      expect(saved.collaboration.submissions[0]).toMatchObject({
        requestId: submission.requestId,
        state: 'submitting',
      });
      expect(saved.chat.draft).toBe('');
    };
    await Promise.all([coordinator.send(input), coordinator.send(input)]);
    expect(provider.sent).toHaveLength(1);
    expect(provider.sent[0]?.text).toBe(input.text);
    await expect(coordinator.send({ ...input, text: 'Different message' })).rejects.toThrow(
      'another message',
    );
  });
  it('keeps two streams and interruptions independent and rejects late/foreign turn events', async () => {
    const { coordinator, provider, message, submissions } = await setup();
    await Promise.all([coordinator.send(message('One')), coordinator.send(message('Two'))]);
    for (const name of ['One', 'Two'])
      provider.emit({
        kind: 'delta',
        threadId: 'thread_agt_' + name,
        turnId: 'turn_req_message_' + name,
        itemId: 'item',
        text: 'Answer ' + name,
      });
    await expect
      .poll(() =>
        coordinator
          .streams()
          .map((stream) => stream.text)
          .sort(),
      )
      .toEqual(['Answer One', 'Answer Two']);
    provider.emit({
      kind: 'delta',
      threadId: 'thread_agt_One',
      turnId: 'foreign',
      itemId: 'item',
      text: 'leak',
    });
    await coordinator.agentAction({
      projectId: 'prj_one',
      agentId: 'agt_One',
      action: 'interrupt',
    });
    expect(provider.interrupted).toEqual(['turn_req_message_One']);
    expect((await submissions()).every((item) => item.state === 'running')).toBe(true);
    provider.emit({
      kind: 'turn',
      threadId: 'thread_agt_One',
      turn: { id: 'turn_req_message_One', status: 'interrupted', clientIds: [], messages: [] },
    });
    await expect.poll(async () => (await submissions())[0]?.state).toBe('interrupted');
    expect((await submissions())[1]?.state).toBe('running');
    expect(coordinator.streams()[0]?.text).toBe('Answer Two');
  });
  it('reserves two slots before slow preparation and prevents same-agent steering', async () => {
    const { coordinator, attach, message, provider } = await setup();
    await attach('Three');
    await Promise.all([coordinator.send(message('One')), coordinator.send(message('Two'))]);
    await expect(coordinator.send(message('Three'))).rejects.toThrow('Two agents are working');
    await expect(coordinator.send(message('One', 'again'))).rejects.toThrow('pending message');
    expect(provider.sent).toHaveLength(2);
  });
  it('persists uncertainty and reconciles the original client ID after restart without replay', async () => {
    const { coordinator, provider, message, submissions, store, account } = await setup();
    provider.fail = true;
    await expect(coordinator.send(message('One'))).rejects.toThrow('Lost acknowledgment');
    expect((await submissions())[0]?.state).toBe('uncertain');
    await coordinator.send(message('One'));
    expect(provider.sent).toHaveLength(1);
    const next = new Provider();
    next.history = [
      {
        id: 'accepted-turn',
        status: 'completed',
        clientIds: [message('One').requestId],
        messages: [{ id: 'answer', text: 'Recovered response' }],
      },
    ];
    const restored = new CollaborationCoordinator(
      store,
      next,
      account,
      () => {},
      () => {},
    );
    await restored.agentAction({ projectId: 'prj_one', agentId: 'agt_One', action: 'reconcile' });
    expect((await submissions())[0]).toMatchObject({
      state: 'completed',
      response: 'Recovered response',
      turnId: 'accepted-turn',
    });
    expect(next.sent).toHaveLength(0);
  });
  it('does not infer non-delivery from absent history; explicit new conversation preserves uncertainty', async () => {
    const { coordinator, provider, message, submissions } = await setup();
    provider.fail = true;
    await expect(coordinator.send(message('One'))).rejects.toThrow();
    await coordinator.agentAction({
      projectId: 'prj_one',
      agentId: 'agt_One',
      action: 'reconcile',
    });
    expect((await submissions())[0]?.state).toBe('uncertain');
    await coordinator.agentAction({
      projectId: 'prj_one',
      agentId: 'agt_One',
      action: 'newConversation',
    });
    expect((await submissions())[0]).toMatchObject({
      state: 'abandoned',
      problem: expect.stringContaining('unconfirmed'),
    });
    expect(provider.sent).toHaveLength(1);
  });
  it('checks Project membership and account-session ownership before provider input', async () => {
    const { coordinator, provider, message, submissions, account } = await setup();
    await expect(coordinator.send({ ...message('One'), projectId: 'prj_two' })).rejects.toThrow(
      'not attached',
    );
    await coordinator.send(message('One'));
    provider.emit({
      kind: 'turn',
      threadId: 'thread_agt_One',
      turn: { id: 'turn_req_message_One', status: 'completed', clientIds: [], messages: [] },
    });
    await expect.poll(async () => (await submissions())[0]?.state).toBe('completed');
    account.session = 'different_session';
    await expect(coordinator.send(message('One', 'new'))).rejects.toThrow('earlier sign-in');
    expect(provider.sent).toHaveLength(1);
  });
  it('keeps unsent edits intact and sends nothing if the Project state cannot be saved', async () => {
    const { path, coordinator, provider, message, store } = await setup();
    await store.updateDraft('prj_one', 'A newer draft');
    provider.beforeStart = async () => {
      expect(
        JSON.parse(await readFile(join(path, 'projects/prj_one.json'), 'utf8')).chat.draft,
      ).toBe('A newer draft');
    };
    await coordinator.send(message('One'));
    await rm(join(path, 'projects/prj_one.json'));
    await expect(coordinator.send(message('Two'))).rejects.toThrow('could not be saved');
    expect(provider.sent).toHaveLength(1);
  });
  it('quit requests each owned interruption and records uncertainty until terminal evidence exists', async () => {
    const { coordinator, provider, message, submissions } = await setup();
    await Promise.all([coordinator.send(message('One')), coordinator.send(message('Two'))]);
    await coordinator.stop();
    expect(provider.interrupted.sort()).toEqual(['turn_req_message_One', 'turn_req_message_Two']);
    expect((await submissions()).map((item) => item.state)).toEqual(['uncertain', 'uncertain']);
    expect(provider.listeners.size).toBe(0);
    expect(coordinator.streams()).toEqual([]);
    expect(() => coordinator.send(message('One'))).toThrow('shutting down');
  });
});

describe('shared-view conversation authority', () => {
  it('requires explicit opt-in and a bound active thread/turn/account; disabling revokes immediately', async () => {
    const called: ToolContext[] = [];
    const s = await setup({
      begin: () => {},
      release: () => {},
      execute: async (context) => {
        called.push(context);
        return { success: true, content: [{ type: 'text', text: 'ok' }] };
      },
    });
    const invoke = (threadId = 'thread_agt_One', turnId = 'turn_req_message_One') =>
      s.provider.toolHandler!(
        { threadId, turnId, callId: 'call_one', tool: 'workspace_list', arguments: {} },
        new AbortController().signal,
      );
    await s.coordinator.send(s.message('One'));
    expect((await invoke()).success).toBe(false);
    s.provider.emit({
      kind: 'turn',
      threadId: 'thread_agt_One',
      turn: {
        id: 'turn_req_message_One',
        status: 'completed',
        clientIds: ['req_message_One'],
        messages: [],
      },
    });
    await expect.poll(() => s.coordinator.hasActive()).toBe(false);
    await s.coordinator.agentAction({
      projectId: 'prj_one',
      agentId: 'agt_One',
      action: 'enableSharedViews',
    });
    await s.coordinator.send(s.message('One', 'shared'));
    expect((await invoke('thread_agt_One', 'turn_req_message_shared')).success).toBe(true);
    expect(called[0]?.agent.provider.toolsetVersion).toBe('shared-views-v15');
    expect((await invoke('thread_foreign', 'turn_req_message_shared')).success).toBe(false);
    expect((await invoke('thread_agt_One', 'turn_foreign')).success).toBe(false);
    s.account.session = 'different';
    expect((await invoke('thread_agt_One', 'turn_req_message_shared')).success).toBe(false);
    s.account.session = 'session_one';
    const disable = s.coordinator.agentAction({
      projectId: 'prj_one',
      agentId: 'agt_One',
      action: 'disableSharedViews',
    });
    expect((await invoke('thread_agt_One', 'turn_req_message_shared')).success).toBe(false);
    await disable;
    const saved = await s.store.readCollaboration('prj_one');
    expect(saved.agents[0]?.access).toBe('messages');
    expect(saved.history.submissions).toHaveLength(2);
  });
  it('rejects a callback result when Stop arrives while the tool is waiting', async () => {
    let finish!: () => void;
    let began!: () => void;
    const started = new Promise<void>((resolve) => {
      began = resolve;
    });
    const s = await setup({
      begin: () => {},
      release: () => {},
      execute: async () => {
        began();
        await new Promise<void>((resolve) => {
          finish = resolve;
        });
        return { success: true, content: [{ type: 'text', text: 'late content' }] };
      },
    });
    await s.coordinator.agentAction({
      projectId: 'prj_one',
      agentId: 'agt_One',
      action: 'enableSharedViews',
    });
    await s.coordinator.send(s.message('One'));
    const result = s.provider.toolHandler!(
      {
        threadId: 'thread_agt_One',
        turnId: 'turn_req_message_One',
        callId: 'call_one',
        tool: 'workspace_list',
        arguments: {},
      },
      new AbortController().signal,
    );
    await started;
    await s.coordinator.agentAction({
      projectId: 'prj_one',
      agentId: 'agt_One',
      action: 'interrupt',
    });
    finish();
    expect((await result).success).toBe(false);
  });
});
