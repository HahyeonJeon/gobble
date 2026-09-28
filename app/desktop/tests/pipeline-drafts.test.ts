import { describe, expect, it } from 'vitest';
import {
  parse,
  CreationDraftSchema,
  PipelineViewSubjectSchema,
  UpdatePipelineDraftInputSchema,
  PipelineDefinitionSchema,
} from '@gobble/contracts';
import { PipelineDraftService } from '../src/main/service/pipeline-drafts';
import { ProjectService } from '../src/main/service/project-service';
const draft = {
  projectId: 'prj_study',
  draftId: 'drf_new',
  generation: 1,
  state: 'draft' as const,
  brief: 'Read quality',
};
const success = (value: unknown) => ({ schemaVersion: 1, ok: true, value });
describe('creation storage contracts', () => {
  it('separates drafts, registered Pipelines and managed origins', () => {
    expect(parse(CreationDraftSchema, draft)).toEqual(draft);
    expect(
      parse(PipelineViewSubjectSchema, { kind: 'creation-draft', draftId: 'drf_new' }),
    ).toEqual({ kind: 'creation-draft', draftId: 'drf_new' });
    for (const value of [
      { ...draft, pipelineId: 'pip_phantom' },
      { ...draft, state: 'ready' },
      { ...draft, generation: 0 },
    ])
      expect(() => parse(CreationDraftSchema, value)).toThrow();
    expect(() =>
      parse(PipelineViewSubjectSchema, { kind: 'pipeline', pipelineId: 'drf_new' }),
    ).toThrow();
    const managed = {
      projectId: 'prj_study',
      pipelineId: 'pip_managed',
      name: 'Read quality',
      origin: { kind: 'managed' },
    };
    expect(parse(PipelineDefinitionSchema, managed)).toEqual(managed);
    expect(() =>
      parse(PipelineDefinitionSchema, {
        ...managed,
        origin: { kind: 'managed', sourceResourceId: 'res_fake' },
      }),
    ).toThrow();
  });
  it('requires an explicit input clear and binds both sides of single-end selection', () => {
    const input = {
      projectId: draft.projectId,
      draftId: draft.draftId,
      requestId: 'req_update',
      expectedGeneration: 1,
      brief: 'Goal',
    };
    expect(() => parse(UpdatePipelineDraftInputSchema, input)).toThrow();
    expect(() =>
      parse(UpdatePipelineDraftInputSchema, {
        ...input,
        resourceId: 'res_reads',
        readLayout: 'paired-end',
      }),
    ).toThrow();
    expect(
      parse(UpdatePipelineDraftInputSchema, { ...input, resourceId: '', readLayout: '' }),
    ).toMatchObject({ resourceId: '', readLayout: '' });
  });
  it('uses scoped native routes, rejects cross-Project and duplicate list results', async () => {
    const calls: unknown[][] = [];
    const service = new PipelineDraftService({
      request: async (...args: unknown[]) => {
        calls.push(args);
        return success(draft);
      },
    });
    await expect(
      service.create({ projectId: 'prj_study', requestId: 'req_new', brief: 'Read quality' }),
    ).resolves.toEqual(draft);
    expect(calls).toEqual([
      [
        '/v1/projects/prj_study/pipeline-drafts',
        'POST',
        { requestId: 'req_new', brief: 'Read quality' },
      ],
    ]);
    const other = new PipelineDraftService({
      request: async () => success({ ...draft, projectId: 'prj_other' }),
    });
    await expect(
      other.read({ projectId: draft.projectId, draftId: draft.draftId }),
    ).rejects.toThrow('different creation draft');
    for (const drafts of [
      [draft, draft],
      [{ ...draft, state: 'discarded' }],
      [{ ...draft, projectId: 'prj_other' }],
    ]) {
      const list = new PipelineDraftService({
        request: async () => success({ projectId: draft.projectId, drafts }),
      });
      await expect(list.list({ projectId: draft.projectId })).rejects.toThrow();
    }
  });
  it('allows several managed Pipelines without inventing duplicate package IDs', async () => {
    const pipelines = ['pip_a', 'pip_b'].map((pipelineId) => ({
      projectId: 'prj_study',
      pipelineId,
      name: 'Analysis',
      origin: { kind: 'managed' },
    }));
    const service = new ProjectService({
      request: async () => success({ projectId: 'prj_study', pipelines }),
    });
    await expect(service.listPipelines({ projectId: 'prj_study' })).resolves.toEqual({
      projectId: 'prj_study',
      pipelines,
    });
  });
});
