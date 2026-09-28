import { describe, expect, it } from 'vitest';
import { parse, PipelineDefinitionSchema, RegisterPipelineInputSchema } from '@gobble/contracts';
import { ProjectService } from '../src/main/service/project-service';

const pipeline = {
  projectId: 'prj_study',
  pipelineId: 'pip_rna',
  name: 'RNA analysis',
  origin: { kind: 'imported', packageResourceId: 'res_package', sourceResourceId: 'res_source' },
};
const success = (value: unknown) => ({ schemaVersion: 1, ok: true, value });

describe('pipeline service contract', () => {
  it('keeps pipeline identity distinct and accepts no paths, effects or extra state', () => {
    expect(parse(PipelineDefinitionSchema, pipeline)).toEqual(pipeline);
    for (const value of [
      { ...pipeline, pipelineId: 'run_rna' },
      { ...pipeline, plan: 'validated' },
      { ...pipeline, sourcePath: '/private/file.go' },
    ])
      expect(() => parse(PipelineDefinitionSchema, value)).toThrow();
    expect(() =>
      parse(RegisterPipelineInputSchema, {
        projectId: 'prj_study',
        requestId: 'req_one',
        packageResourceId: 'res_package',
        name: 'RNA analysis',
        command: 'go run .',
      }),
    ).toThrow();
  });

  it('routes metadata registration and validates the returned Project/package binding', async () => {
    const calls: unknown[][] = [];
    const service = new ProjectService({
      request: async (...args: unknown[]) => {
        calls.push(args);
        return success(pipeline);
      },
    });
    const input = {
      projectId: 'prj_study',
      requestId: 'req_one',
      packageResourceId: 'res_package',
      name: 'RNA analysis',
    };
    await expect(service.registerPipeline(input)).resolves.toEqual(pipeline);
    expect(calls).toEqual([
      [
        '/v1/projects/prj_study/pipelines',
        'POST',
        {
          requestId: 'req_one',
          packageResourceId: 'res_package',
          name: 'RNA analysis',
        },
      ],
    ]);
    for (const response of [
      { ...pipeline, projectId: 'prj_other' },
      { ...pipeline, origin: { ...pipeline.origin, packageResourceId: 'res_other' } },
    ]) {
      const untrusted = new ProjectService({ request: async () => success(response) });
      await expect(untrusted.registerPipeline(input)).rejects.toThrow('different resource');
    }
  });

  it('rejects cross-Project and duplicate list identities instead of presenting them', async () => {
    const candidates = [
      { projectId: 'prj_other', pipelines: [] },
      { projectId: 'prj_study', pipelines: [{ ...pipeline, projectId: 'prj_other' }] },
      { projectId: 'prj_study', pipelines: [pipeline, pipeline] },
      { projectId: 'prj_study', pipelines: [pipeline, { ...pipeline, pipelineId: 'pip_second' }] },
    ];
    for (const response of candidates) {
      const service = new ProjectService({ request: async () => success(response) });
      await expect(service.listPipelines({ projectId: 'prj_study' })).rejects.toThrow();
    }
    const service = new ProjectService({
      request: async () => success({ projectId: 'prj_study', pipelines: [pipeline] }),
    });
    await expect(service.listPipelines({ projectId: 'prj_study' })).resolves.toEqual({
      projectId: 'prj_study',
      pipelines: [pipeline],
    });
  });
});
