import { mkdtemp, mkdir, readFile, writeFile, rm } from 'node:fs/promises';
import { join } from 'node:path';
import { tmpdir } from 'node:os';
import { WorkspaceStorage } from '../src/main/workspace/storage';
import { describe, it, expect, vi } from 'vitest';
import { readFileSync } from 'node:fs';
import {
  parse,
  PipelinePreparationSchema,
  validatePreparation,
  validatePipelineReviewContext,
  readStoredWorkspace,
  parseWorkspaceDocument,
  type PipelinePreparation,
  type WorkspaceDocument,
  type AgentAttachment,
  type SendMessage,
} from '@gobble/contracts';
import { RunPreparationService } from '../src/main/service/run-preparation';
import { PipelineReviewHost } from '../src/main/pipeline-review/host';
import { emptyWorkspace } from '../src/main/workspace/model';
import type { WorkspaceController } from '../src/main/workspace/controller';
import type { ProjectService } from '../src/main/service/project-service';
import type { ToolContext } from '../src/main/shared-context/ports';
const hash = 'sha256:' + 'a'.repeat(64);
const fixture = JSON.parse(
  readFileSync(new URL('./fixtures/creation-candidate.json', import.meta.url), 'utf8'),
);
const scope = {
  pipelineId: 'pip_one',
  baseArtifactId: hash,
  preparationId: 'req_prepared',
  preparationSection: 'settings' as const,
};
function value(): PipelinePreparation {
  const a = fixture.artifact;
  return parse(PipelinePreparationSchema, {
    projectId: 'prj_one',
    pipelineId: 'pip_one',
    requestId: 'req_prepared',
    engineId: hash,
    artifactId: hash,
    state: 'ready',
    fresh: true,
    issue: '',
    createdAt: a.checkedAt,
    prepared: {
      digest: hash,
      flow: a.check.review.flow,
      binding: {
        sourceRevision: a.sourceRevision,
        runtimeId: hash,
        inputIdentity: hash,
        inputPath: a.check.inputPath,
        cap: 1,
      },
      input: a.input,
      steps: a.check.review.flow.steps.map((s: { id: string }) => ({
        id: s.id,
        label: 'Processing step',
        cpu: 2,
        memory: '1g',
        image: 'image',
        outputs: [],
      })),
      checkedAt: a.checkedAt,
    },
  });
}
describe('run preparation contracts and ownership', () => {
  it('refuses private bytes, incomplete ready results and conflicting reference kinds', () => {
    const v = value();
    validatePreparation(v);
    for (const bad of [
      { ...v, payload: 'private' },
      { ...v, prepared: undefined },
      { ...v, state: 'preparing' },
      { ...v, state: 'cancelled' },
    ])
      expect(() => parse(PipelinePreparationSchema, bad)).toThrow();
    expect(() =>
      validatePipelineReviewContext({
        ...scope,
        proposalId: 'req_proposal',
        proposedArtifactId: hash,
      }),
    ).toThrow();
    expect(() =>
      validatePipelineReviewContext({
        pipelineId: 'pip_one',
        baseArtifactId: hash,
        preparationSection: 'data',
      }),
    ).toThrow();
    validatePipelineReviewContext(scope);
  });
  it('checks all native associations before exposing a run review', async () => {
    const v = value(),
      request = vi.fn(async () => ({ schemaVersion: 1, ok: true, value: v }));
    const service = new RunPreparationService({ request });
    await service.read({ projectId: 'prj_one', pipelineId: 'pip_one', requestId: 'req_prepared' });
    await expect(
      service.read({ projectId: 'prj_other', pipelineId: 'pip_one', requestId: 'req_prepared' }),
    ).rejects.toMatchObject({ code: 'internal' });
    await expect(
      service.read({ projectId: 'prj_one', pipelineId: 'pip_one', requestId: 'req_other' }),
    ).rejects.toMatchObject({ code: 'internal' });
    await expect(
      service.prepare({
        projectId: 'prj_one',
        pipelineId: 'pip_one',
        requestId: 'req_prepared',
        artifactId: 'sha256:' + 'b'.repeat(64),
        engineId: hash,
      }),
    ).rejects.toMatchObject({ code: 'internal' });
  });
  it('freezes v18 and preserves exact new discussion coordinates in v19', () => {
    const old = { ...emptyWorkspace('prj_one'), schemaVersion: 18 };
    expect(readStoredWorkspace(old)).toEqual({ ...old, schemaVersion: 21 });
    expect(() =>
      readStoredWorkspace({ ...old, chat: { ...old.chat, pipelineReview: scope } }),
    ).toThrow();
    const doc = emptyWorkspace('prj_one');
    doc.chat.pipelineReview = scope;
    expect(parseWorkspaceDocument(doc).chat.pipelineReview).toEqual(scope);
  });
  it('requires read-before-point and keeps User selection, Chat and Current independent', async () => {
    const doc = emptyWorkspace('prj_one');
    doc.chat.pipelineReview = scope;
    doc.chat.draft = 'My unsent question';
    doc.workspace.agents.push({
      projectId: 'prj_one',
      agentId: 'agt_one',
      name: 'Researcher',
      instructionProfile: 'discussion-v1',
      access: 'sharedViews',
      provider: { kind: 'codex', threadId: 'thread' },
    });
    doc.collaboration = {
      submissions: [
        {
          requestId: 'req_message',
          agentId: 'agt_one',
          text: 'Review these settings',
          pipelineReview: scope,
          model: 'model',
          effort: 'low',
          createdAt: 1,
          state: 'completed',
          threadId: 'thread',
          turnId: 'turn',
          response: '',
          problem: null,
        },
      ],
    };
    const prepared = value();
    const host = new PipelineReviewHost(
      {
        readShared: async () => doc,
        changeShared: async (
          _p: string,
          guard: () => void,
          change: (d: WorkspaceDocument) => void,
        ) => {
          guard();
          change(doc);
          parseWorkspaceDocument(doc);
        },
      } as unknown as WorkspaceController,
      { preparations: { read: async () => prepared } } as unknown as ProjectService,
    );
    const context = {
      agent: { projectId: 'prj_one', agentId: 'agt_one', access: 'sharedViews' },
      submission: { requestId: 'req_message', pipelineReview: scope },
      signal: new AbortController().signal,
      assert: () => {},
    } as unknown as ToolContext;
    const call = {
      threadId: 'thread',
      turnId: 'turn',
      callId: 'call',
      tool: 'gobble_pipeline_point',
      arguments: { context: scope, note: 'Discuss this exact preparation setting.' },
    };
    await expect(host.execute(context, call, () => {})).rejects.toMatchObject({
      code: 'forbidden',
    });
    await host.execute(
      context,
      { ...call, tool: 'gobble_pipeline_review', arguments: { context: scope } },
      () => {},
    );
    await host.execute(context, call, () => {});
    expect(doc.pipelineReviewMarks?.[0]?.context).toEqual(scope);
    expect(doc.chat.draft).toBe('My unsent question');
    expect(doc.selections).toEqual([]);
    host.release('prj_one', 'agt_one', 'req_message');
    await expect(host.execute(context, call, () => {})).rejects.toMatchObject({
      code: 'forbidden',
    });
  });
  it('delivers the selected saved review, including earlier status, without silently choosing a newer preparation', async () => {
    const doc = emptyWorkspace('prj_one');
    doc.chat.pipelineReview = scope;
    const v = value();
    if (v.state === 'ready') v.fresh = false;
    const read = vi.fn(async () => v);
    const host = new PipelineReviewHost(
      { readShared: async () => doc } as unknown as WorkspaceController,
      { preparations: { read } } as unknown as ProjectService,
    );
    const input = { projectId: 'prj_one', pipelineReview: scope } as SendMessage;
    const agent = { access: 'sharedViews' } as AgentAttachment;
    const delivery = await host.prepare(input, agent, () => {});
    expect(JSON.parse(delivery[0]!.text).facts.preparation).toEqual(v);
    await expect(
      host.prepare({ ...input, allowPipelineProposal: true }, agent, () => {}),
    ).rejects.toMatchObject({ code: 'forbidden' });
    expect(read).toHaveBeenCalledWith({
      projectId: 'prj_one',
      pipelineId: 'pip_one',
      requestId: 'req_prepared',
    });
  });
});

it('archives the exact v18 document before writing preparation references in v19', async () => {
  const directory = await mkdtemp(join(tmpdir(), 'gobble-preparation-migration-'));
  try {
    await mkdir(join(directory, 'projects'));
    const path = join(directory, 'projects/prj_one.json');
    const old = { ...emptyWorkspace('prj_one'), schemaVersion: 18 };
    old.chat.draft = 'Keep this draft';
    const raw = JSON.stringify(old, null, 2) + '\n';
    await writeFile(path, raw);
    const file = new WorkspaceStorage(directory).project('prj_one');
    const next = await file.read();
    expect(await readFile(path, 'utf8')).toBe(raw);
    if (!next) throw Error('Missing workspace');
    next.chat.pipelineReview = scope;
    await file.write(next);
    expect(await readFile(path + '.v18.backup', 'utf8')).toBe(raw);
    const reopened = await new WorkspaceStorage(directory).project('prj_one').read();
    expect(reopened?.chat.pipelineReview).toEqual(scope);
    expect(reopened?.chat.draft).toBe('Keep this draft');
  } finally {
    await rm(directory, { recursive: true, force: true });
  }
});
