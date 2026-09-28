import { describe, it, expect, vi } from 'vitest';
import {
  parse,
  PipelineReviewContextSchema,
  PipelineProposeInputSchema,
  validatePipelineReviewContext,
  readStoredWorkspace,
  SendMessageSchema,
} from '@gobble/contracts';
import { PipelineReviewHost } from '../src/main/pipeline-review/host';
import { emptyWorkspace } from '../src/main/workspace/model';
import type { WorkspaceController } from '../src/main/workspace/controller';
import type { ProjectService } from '../src/main/service/project-service';
import type { ToolContext } from '../src/main/shared-context/ports';
import { WorkspaceDocumentV16Schema } from '../../contracts/src/workspace-document-v16';
const hash = 'sha256:' + 'a'.repeat(64);
const scope = { pipelineId: 'pip_one', baseArtifactId: hash };
function fixture() {
  const source = vi.fn(async () => ({
    projectId: 'prj_one',
    ...scope,
    files: [{ path: 'pipeline.go', content: 'package example' }],
  }));
  const propose = vi.fn(async () => ({ accepted: true }));
  const doc = emptyWorkspace('prj_one');
  doc.chat.pipelineReview = scope;
  const host = new PipelineReviewHost(
    { readShared: async () => doc } as unknown as WorkspaceController,
    {
      pipelineInspection: async () => ({ state: 'ready', artifact: { artifactId: hash } }),
      proposals: { source, propose },
    } as unknown as ProjectService,
  );
  const context = {
    agent: { projectId: 'prj_one', agentId: 'agt_one', access: 'sharedViews' },
    submission: { requestId: 'req_turn', pipelineReview: scope, allowPipelineProposal: true },
    signal: new AbortController().signal,
    assert: () => {},
  } as unknown as ToolContext;
  return { host, source, propose, context };
}
describe('Pipeline proposal authority and immutable comparison references', () => {
  it('keeps source permission per message, separate from read-only shared views', async () => {
    const { host, source, context } = fixture();
    const call = {
      threadId: 'thread',
      turnId: 'turn',
      callId: 'read',
      tool: 'gobble_pipeline_source',
      arguments: {},
    };
    await expect(
      host.execute(
        { ...context, submission: { ...context.submission, allowPipelineProposal: false } },
        call,
        () => {},
      ),
    ).rejects.toMatchObject({ code: 'forbidden' });
    expect(source).not.toHaveBeenCalled();
    await expect(host.execute(context, call, () => {})).resolves.toMatchObject({
      baseArtifactId: hash,
    });
    expect(source).toHaveBeenCalledWith({ projectId: 'prj_one', ...scope });
    await expect(
      host.execute(context, call, () => {
        throw new Error('expired');
      }),
    ).rejects.toThrow('expired');
    expect(source).toHaveBeenCalledTimes(1);
  });
  it('requires an observed source scope and revokes it at the turn boundary', async () => {
    const { host, context, propose } = fixture();
    const call = {
      threadId: 'thread',
      turnId: 'turn',
      callId: 'propose',
      tool: 'gobble_pipeline_propose',
      arguments: { summary: 'A change', files: [{ path: 'pipeline.go', content: 'package next' }] },
    };
    await expect(host.execute(context, call, () => {})).rejects.toMatchObject({
      code: 'forbidden',
    });
    await host.execute(
      context,
      { ...call, tool: 'gobble_pipeline_source', arguments: {} },
      () => {},
    );
    await expect(host.execute(context, call, () => {})).resolves.toEqual({ accepted: true });
    expect(propose).toHaveBeenCalledWith(
      expect.objectContaining({
        projectId: 'prj_one',
        ...scope,
        requestId: expect.stringMatching(/^req_[a-f0-9]{64}$/),
      }),
    );
    host.release('prj_one', 'agt_one', 'req_turn');
    await expect(host.execute(context, call, () => {})).rejects.toMatchObject({
      code: 'forbidden',
    });
  });
  it('validates the selected User context before message delivery', async () => {
    const { host, context, source } = fixture();
    const agent = context.agent;
    await expect(
      host.prepare(
        {
          projectId: 'prj_one',
          agentId: 'agt_one',
          requestId: 'req_one',
          text: 'Refine this',
          allowPipelineProposal: true,
        },
        agent,
        () => {},
      ),
    ).rejects.toMatchObject({ code: 'forbidden' });
    await expect(
      host.prepare(
        {
          projectId: 'prj_one',
          agentId: 'agt_one',
          requestId: 'req_one',
          text: 'Refine this',
          pipelineReview: { ...scope, baseArtifactId: 'sha256:' + 'b'.repeat(64) },
          allowPipelineProposal: true,
        },
        agent,
        () => {},
      ),
    ).rejects.toMatchObject({ code: 'stale_revision' });
    expect(source).not.toHaveBeenCalled();
  });
  it('rejects foreign authoring fields and incomplete comparison coordinates', () => {
    expect(() =>
      parse(PipelineReviewContextSchema, { ...scope, path: '/private/source' }),
    ).toThrow();
    expect(() =>
      parse(SendMessageSchema, {
        projectId: 'prj_one',
        agentId: 'agt_one',
        requestId: 'req_one',
        text: 'x',
        pipelineReview: { arbitrary: 'untyped' },
      }),
    ).toThrow();
    expect(() => validatePipelineReviewContext({ ...scope, proposalId: 'req_proposal' })).toThrow();
    expect(() =>
      parse(PipelineProposeInputSchema, {
        projectId: 'prj_one',
        ...scope,
        requestId: 'req_one',
        summary: 'x',
        files: [],
        run: true,
      }),
    ).toThrow();
  });
  it('freezes v16 without inventing source authority during migration', () => {
    const old = { ...emptyWorkspace('prj_one'), schemaVersion: 16 };
    expect(() => parse(WorkspaceDocumentV16Schema, old)).not.toThrow();
    expect(() =>
      parse(WorkspaceDocumentV16Schema, { ...old, chat: { ...old.chat, pipelineReview: scope } }),
    ).toThrow();
    const next = readStoredWorkspace(old);
    expect(next.schemaVersion).toBe(21);
    expect(next.chat.pipelineReview).toBeUndefined();
    expect(next.chat.draft).toBe(old.chat.draft);
  });
});
