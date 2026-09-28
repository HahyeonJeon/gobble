import { describe, it, expect, vi } from 'vitest';
import {
  parse,
  CreationContextSchema,
  CreationConnectSchema,
  readStoredWorkspace,
  parseWorkspaceDocument,
} from '@gobble/contracts';
import { PipelineCreationHost } from '../src/main/pipeline-review/creation-host';
import { emptyWorkspace } from '../src/main/workspace/model';
import type { WorkspaceController } from '../src/main/workspace/controller';
import type { ProjectService } from '../src/main/service/project-service';
import type { ToolContext } from '../src/main/shared-context/ports';
const hash = 'sha256:' + 'a'.repeat(64);
function fixture() {
  const scope = { kind: 'draft' as const, draftId: 'drf_new', generation: 2 };
  const doc = emptyWorkspace('prj_one');
  doc.chat.pipelineCreation = scope;
  const source = vi.fn(async () => ({ scopeId: hash })),
    propose = vi.fn(async () => ({ candidateId: 'req_proposed' }));
  const host = new PipelineCreationHost(
    { readShared: async () => doc } as unknown as WorkspaceController,
    {
      drafts: {
        read: async () => ({
          draftId: 'drf_new',
          state: 'draft',
          generation: 2,
          input: { resourceId: 'res_reads' },
        }),
      },
      creation: { source, propose },
    } as unknown as ProjectService,
  );
  const context = {
    agent: { projectId: 'prj_one', agentId: 'agt_one', access: 'sharedViews' },
    submission: { requestId: 'req_message', pipelineCreation: scope, allowPipelineCreation: true },
    signal: new AbortController().signal,
    assert: () => {},
  } as unknown as ToolContext;
  return { host, context, source, propose, scope, doc };
}
describe('creation authority', () => {
  it('requires explicit message opt-in and source-read before authoring; revokes at turn end', async () => {
    const { host, context, source, propose } = fixture();
    const call = {
      threadId: 'thread',
      turnId: 'turn',
      callId: 'one',
      tool: 'gobble_creation_source',
      arguments: {},
    };
    await expect(
      host.execute(
        { ...context, submission: { ...context.submission, allowPipelineCreation: false } },
        call,
        () => {},
      ),
    ).rejects.toMatchObject({ code: 'forbidden' });
    expect(source).not.toHaveBeenCalled();
    const write = {
      ...call,
      tool: 'gobble_creation_propose',
      arguments: {
        summary: 'Read quality',
        files: [{ path: 'pipeline/pipe.go', content: 'package pipeline' }],
      },
    };
    await expect(host.execute(context, write, () => {})).rejects.toMatchObject({
      code: 'forbidden',
    });
    await host.execute(context, call, () => {});
    await host.execute(context, write, () => {});
    expect(propose).toHaveBeenCalledWith(
      expect.objectContaining({
        draftId: 'drf_new',
        expectedGeneration: 2,
        scopeId: hash,
        requestId: expect.stringMatching(/^req_[a-f0-9]{64}$/),
      }),
    );
    host.release('prj_one', 'agt_one', 'req_message');
    await expect(host.execute(context, write, () => {})).rejects.toMatchObject({
      code: 'forbidden',
    });
  });
  it('rejects stale selections and candidate discussion authoring before provider delivery', async () => {
    const { host, context, scope, source } = fixture();
    const input = {
      projectId: 'prj_one',
      agentId: 'agt_one',
      requestId: 'req_send',
      text: 'Design this',
      pipelineCreation: scope,
    };
    await expect(
      host.prepare(
        { ...input, pipelineCreation: { ...scope, generation: 1 } },
        context.agent,
        () => {},
      ),
    ).rejects.toMatchObject({ code: 'stale_revision' });
    await expect(host.prepare(input, context.agent, () => {})).resolves.toHaveLength(1);
    expect(source).not.toHaveBeenCalled();
    await host.prepare({ ...input, allowPipelineCreation: true }, context.agent, () => {});
    expect(source).toHaveBeenCalledTimes(1);
    await expect(
      host.prepare(input, { ...context.agent, access: 'messages' }, () => {}),
    ).rejects.toMatchObject({ code: 'forbidden' });
  });
  it('closes creation references and renderer engine selection; preserves v17 history', () => {
    const context = {
      kind: 'candidate',
      base: 'none',
      draftId: 'drf_new',
      generation: 2,
      candidateId: 'req_one',
      artifactId: hash,
      target: { kind: 'setting', id: 'trim', name: 'quality' },
    } as const;
    expect(parse(CreationContextSchema, context)).toEqual(context);
    for (const bad of [
      { ...context, base: 'current' },
      { ...context, pipelineId: 'pip_fake' },
      { ...context, target: { kind: 'code', line: 3 } },
    ])
      expect(() => parse(CreationContextSchema, bad)).toThrow();
    expect(() =>
      parse(CreationConnectSchema, { engineId: hash, endpoint: 'unix:///other.sock' }),
    ).toThrow();
    const old = { ...emptyWorkspace('prj_one'), schemaVersion: 17 };
    expect(readStoredWorkspace(old)).toEqual({ ...old, schemaVersion: 21 });
    expect(() =>
      readStoredWorkspace({ ...old, chat: { ...old.chat, pipelineCreation: context } }),
    ).toThrow();
    const doc = emptyWorkspace('prj_one');
    doc.creationMarks = [
      { context, agentId: 'agt_missing', submissionId: 'req_missing', note: 'Here' },
    ];
    expect(() => parseWorkspaceDocument(doc)).toThrow();
  });
});
