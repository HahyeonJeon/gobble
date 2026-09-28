import { Type } from '@sinclair/typebox';
import { Value } from '@sinclair/typebox/value';
import { codexToolSchema } from '../src/main/codex/tool-schema';
import { restoreSchema } from '../../contracts/src/storage/restore-schema';
import { mkdtemp, mkdir, chmod, copyFile, rm, readFile } from 'node:fs/promises';
import { join, resolve } from 'node:path';
import { tmpdir } from 'node:os';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { CodexTransport } from '../src/main/codex/transport';
import { CodexAccount } from '../src/main/codex/account';
import { CodexConversations } from '../src/main/codex/conversations';
import { loginURL } from '../src/main/codex/policy';
import { decodeEvent, decodeTurn } from '../src/main/codex/protocol';
import type { AgentAttachment } from '@gobble/contracts';

const cleanup: (() => Promise<unknown>)[] = [];
afterEach(async () => {
  for (const clean of cleanup.splice(0).reverse()) await clean();
});
async function setup(timeout = 1000, callbackTimeout = 15000) {
  const path = await mkdtemp(join(tmpdir(), 'gobble-codex-test-'));
  cleanup.push(() => rm(path, { recursive: true, force: true }));
  const executable = join(path, 'codex');
  await copyFile(resolve('desktop/tests/fixtures/codex.cjs'), executable);
  await chmod(executable, 0o700);
  const cwd = join(path, 'discussion');
  await mkdir(cwd);
  const rpc = new CodexTransport(executable, join(path, 'home'), cwd, timeout, callbackTimeout);
  cleanup.push(() => rpc.stop());
  return { path, rpc, provider: new CodexConversations(rpc, cwd) };
}
const agent: AgentAttachment = {
  projectId: 'prj_one',
  agentId: 'agt_one',
  name: 'One',
  instructionProfile: 'discussion-v1',
  configuration: { model: 'fixture-model', effort: 'low', instructions: '' },
  provider: { kind: 'codex', threadId: null },
};

describe('pinned Codex protocol boundary', () => {
  it('adapts addressed text/image input without local paths or unregistered tools', async () => {
    const { rpc, provider, path } = await setup();
    await rpc.start();
    const threadId = await provider.bind(agent);
    await provider.start(
      {
        requestId: 'req_attached',
        agentId: agent.agentId,
        text: 'Inspect',
        model: 'fixture-model',
        effort: 'low',
        createdAt: 1,
        state: 'submitting',
        threadId,
        turnId: null,
        response: '',
        problem: null,
      },
      [
        { type: 'text', text: 'Exact selected receipt' },
        { type: 'image', url: 'data:image/png;base64,AA==' },
      ],
    );
    const state = JSON.parse(await readFile(join(path, 'home/fixture-state.json'), 'utf8'));
    expect(state.threads[threadId].tools).toEqual([]);
    expect(state.threads[threadId].turns[0].items[0].content).toEqual([
      { type: 'text', text: 'Inspect', text_elements: [] },
      { type: 'text', text: 'Exact selected receipt', text_elements: [] },
      { type: 'image', url: 'data:image/png;base64,AA==' },
    ]);
  });
  it('clears a dead browser login so reconnect can start a fresh ceremony', async () => {
    const { rpc, path } = await setup();
    const account = new CodexAccount(
      rpc,
      join(path, 'session.json'),
      async () => {},
      () => {},
    );
    cleanup.push(() => account.stop());
    await account.action('signIn');
    expect(account.snapshot().account.status).toBe('signingIn');
    await rpc.stop();
    expect(account.snapshot().account.status).toBe('signedOut');
    await account.action('signIn');
    await expect.poll(() => account.snapshot().account.status).toBe('signedIn');
  });
  it('starts an isolated process, pins read-only parameters and keeps provider tools out of the app bridge', async () => {
    const { rpc, provider } = await setup();
    await rpc.start();
    expect(await rpc.request('account/read')).toMatchObject({ account: null });
    const threadId = await provider.bind(agent);
    expect(threadId).toBe('thread_1');
    expect(await rpc.request('fixture/request')).toEqual({});
    expect(await provider.read(threadId)).toEqual([]);
  });
  it.each(['fixture/no-response', 'fixture/malformed'])(
    'fails pending requests on %s and allows a clean reconnect',
    async (method) => {
      const { rpc } = await setup();
      await rpc.start();
      let disconnected = 0;
      rpc.onDisconnected(() => disconnected++);
      if (method === 'fixture/no-response') {
        // Exercise the deadline deterministically after the real process handshake.
        // A tiny wall-clock deadline also races startup under unrelated CPU load.
        vi.useFakeTimers();
        try {
          const rejected = expect(rpc.request(method)).rejects.toThrow('connection was lost');
          await vi.advanceTimersByTimeAsync(1000);
          await rejected;
        } finally {
          vi.useRealTimers();
        }
      } else {
        await expect(rpc.request(method)).rejects.toThrow('connection was lost');
      }
      expect(disconnected).toBe(1);
      await rpc.start();
      expect(await rpc.request('account/read')).toMatchObject({ account: null });
    },
  );
  it('uses official login status and catalog, persists only a local session reference, and invalidates it on logout', async () => {
    const { rpc, path } = await setup();
    await rpc.start();
    await rpc.request('fixture/model-delay', { ms: 100 });
    const urls: string[] = [];
    const account = new CodexAccount(
      rpc,
      join(path, 'session.json'),
      async (url) => {
        urls.push(url);
      },
      () => {},
    );
    cleanup.push(() => account.stop());
    await account.action('signIn');
    expect(urls).toEqual(['https://auth.openai.com/authorize?fixture=true']);
    await expect.poll(() => account.snapshot().account.status).toBe('signedIn');
    const session = account.require('fixture-model', 'high');
    expect(account.snapshot().models[0]?.name).toBe('Fixture model');
    const saved = JSON.parse(await readFile(join(path, 'session.json'), 'utf8'));
    expect(saved).toEqual({ schemaVersion: 1, sessionId: session });
    expect(() => account.require('unknown', 'high')).toThrow('Choose a model');
    await account.action('signOut');
    expect(() => account.require()).toThrow('Connect your ChatGPT');
    await account.action('signIn');
    await expect.poll(() => account.snapshot().account.status).toBe('signedIn');
    expect(account.require()).not.toBe(session);
  });
  it.each([
    'https://auth.openai.com.attacker.test/login',
    'file:///secret',
    'http://auth.openai.com/login',
    'https://user@auth.openai.com/login',
    'https://chatgpt.com:444/login',
  ])('rejects untrusted login target %s', (url) => {
    expect(() => loginURL(url)).toThrow();
  });
  it('projects only public agent text, preserves client submission IDs, and rejects malformed turn identity', () => {
    expect(
      decodeTurn({
        id: 'turn',
        status: 'completed',
        items: [
          { type: 'reasoning', text: 'private' },
          { type: 'userMessage', clientId: 'req_one' },
          { type: 'agentMessage', id: 'answer', text: 'Public response' },
        ],
      }),
    ).toEqual({
      id: 'turn',
      status: 'completed',
      clientIds: ['req_one'],
      messages: [{ id: 'answer', text: 'Public response' }],
    });
    expect(() => decodeEvent('turn/completed', { threadId: '', turn: {} })).toThrow();
    expect(() => decodeTurn({ id: 'x', status: 'invented', items: [] })).toThrow();
  });
});

describe('dynamic-tool protocol adapter', () => {
  it('registers a versioned shared toolset and returns the exact text/image wire envelope', async () => {
    const { rpc, provider, path } = await setup();
    await rpc.start();
    const shared = { ...agent, access: 'sharedViews' as const };
    const threadId = await provider.bind(shared);
    const state = JSON.parse(await readFile(join(path, 'home/fixture-state.json'), 'utf8'));
    expect(state.threads[threadId].tools.map((tool: { name: string }) => tool.name)).toContain(
      'workspace_point',
    );
    let received: unknown;
    provider.onTool(async (call) => {
      received = call;
      return {
        success: true,
        content: [
          { type: 'text', text: 'receipt' },
          { type: 'image', url: 'data:image/png;base64,AA==' },
        ],
      };
    });
    const params = {
      threadId,
      turnId: 'turn_live',
      callId: 'call_live',
      namespace: null,
      tool: 'workspace_observe',
      arguments: { surfaceId: 'srf_one' },
    };
    expect(await rpc.request('fixture/tool', { params })).toEqual({
      success: true,
      contentItems: [
        { type: 'inputText', text: 'receipt' },
        { type: 'inputImage', imageUrl: 'data:image/png;base64,AA==' },
      ],
    });
    expect(received).toMatchObject(params);
    expect(
      await rpc.request('fixture/tool', { params: { ...params, namespace: 'foreign' } }),
    ).toMatchObject({ success: false });
    expect(
      await rpc.request('fixture/tool', {
        method: 'command/exec',
        params: { command: 'forbidden' },
      }),
    ).toMatchObject({ success: false });
  });
  it('rejects an unsupported saved toolset before resuming a conversation', async () => {
    const { rpc, provider } = await setup();
    await rpc.start();
    await expect(
      provider.bind({
        ...agent,
        access: 'sharedViews',
        provider: { kind: 'codex', threadId: 'thread_prior', toolsetVersion: 'future' },
      }),
    ).rejects.toThrow('unsupported response');
  });
});

it('bounds inbound callbacks, cancels them on disconnect and expires unresolved handlers', async () => {
  const { rpc } = await setup(1000, 100);
  const signals: AbortSignal[] = [];
  rpc.onRequest('item/tool/call', async (_params, signal) => {
    signals.push(signal);
    return new Promise(() => {});
  });
  await rpc.start();
  const requests = Array.from({ length: 4 }, () => rpc.request('fixture/tool', { params: {} }));
  await expect.poll(() => signals.length).toBe(4);
  expect(await rpc.request('fixture/tool', { params: {} })).toMatchObject({ success: false });
  const expired = await Promise.all(requests);
  expect(expired.every((value) => (value as { success: boolean }).success === false)).toBe(true);
  expect(signals.every((signal) => signal.aborted)).toBe(true);
  const pending = rpc.request('fixture/tool', { params: {} }).catch(() => null);
  await expect.poll(() => signals.length).toBe(5);
  await rpc.stop();
  await pending;
  expect(signals[4]?.aborted).toBe(true);
});

it('adapts fixed numeric PDF coordinate tuples to the pinned provider without widening validation', () => {
  const tuple = Type.Object(
    { region: Type.Tuple([Type.Number(), Type.Number(), Type.Number(), Type.Number()]) },
    { additionalProperties: false },
  );
  const original = JSON.stringify(tuple),
    wire = codexToolSchema(tuple);
  expect(wire).toMatchObject({
    properties: { region: { type: 'array', minItems: 4, maxItems: 4, items: { type: 'number' } } },
  });
  const restored = Type.Unsafe(restoreSchema(wire as Record<string, unknown>));
  for (const region of [[1, 2, 3, 4], [1, 2, 3], [1, 2, 3, 4, 5], [1, 2, '3', 4], null])
    expect(Value.Check(restored, { region })).toBe(Value.Check(tuple, { region }));
  expect(JSON.stringify(tuple)).toBe(original);
  expect(() => codexToolSchema(Type.Tuple([Type.Number(), Type.String()]))).toThrow(/homogeneous/);
});
