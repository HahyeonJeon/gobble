import { describe, expect, it } from 'vitest';
import {
  parse,
  CreationDraftSchema,
  validateCreationDraft,
  CreationAdoptionOutcomeSchema,
} from '@gobble/contracts';
import { PipelineCreationService } from '../src/main/service/pipeline-creation';
const digest = 'sha256:' + 'a'.repeat(64);
const input = {
  projectId: 'prj_study',
  draftId: 'drf_new',
  requestId: 'req_confirm',
  candidateId: 'req_candidate',
  artifactId: digest,
  expectedGeneration: 2,
  name: 'Read quality',
};
const adoption = {
  requestId: input.requestId,
  pipelineId: 'pip_created',
  candidateId: input.candidateId,
  artifactId: digest,
  generation: 2,
  name: input.name,
};
const outcome = {
  projectId: input.projectId,
  draftId: input.draftId,
  requestId: input.requestId,
  state: 'adopted',
  adoption,
};
const service = (value: unknown) =>
  new PipelineCreationService({ request: async () => ({ schemaVersion: 1, ok: true, value }) });
describe('first adoption contract boundary', () => {
  it('rejects absent and mixed adoption state', () => {
    const draft = {
      projectId: input.projectId,
      draftId: input.draftId,
      generation: 3,
      state: 'adopted',
      brief: 'Read quality',
      adoption,
      input: {
        resourceId: 'res_reads',
        relativePath: 'reads.fastq.gz',
        readLayout: 'single-end',
        size: 100,
        modifiedAt: '2026-09-12',
        identity: digest,
      },
    };
    expect(() => validateCreationDraft(parse(CreationDraftSchema, draft))).not.toThrow();
    for (const invalid of [
      { ...draft, adoption: undefined },
      { ...draft, input: undefined },
      { ...draft, generation: 2 },
      { ...draft, state: 'draft' },
      { ...draft, state: 'discarded' },
    ])
      expect(() => validateCreationDraft(parse(CreationDraftSchema, invalid))).toThrow();
    for (const invalid of [
      { ...outcome, adoption: undefined },
      { ...outcome, state: 'not-recorded' },
      { ...outcome, runRef: 'run_forbidden' },
    ])
      expect(() => parse(CreationAdoptionOutcomeSchema, invalid)).toThrow();
  });
  it('posts the exact User intent and scopes outcome lookup', async () => {
    const calls: unknown[][] = [];
    const api = new PipelineCreationService({
      request: async (...args: unknown[]) => {
        calls.push(args);
        return { schemaVersion: 1, ok: true, value: outcome };
      },
    });
    await expect(api.adopt(input)).resolves.toEqual(outcome);
    await expect(
      api.outcome({
        projectId: input.projectId,
        draftId: input.draftId,
        requestId: input.requestId,
      }),
    ).resolves.toEqual(outcome);
    const { projectId, draftId, ...payload } = input;
    expect(calls).toEqual([
      [`/v1/projects/${projectId}/pipeline-drafts/${draftId}/adopt`, 'POST', payload],
      [`/v1/projects/${projectId}/pipeline-drafts/${draftId}/adoptions/${input.requestId}`],
    ]);
  });
  it('refuses mismatched Project, draft, operation, candidate and checked facts', async () => {
    for (const value of [
      { ...outcome, projectId: 'prj_other' },
      { ...outcome, draftId: 'drf_other' },
      { ...outcome, requestId: 'req_other' },
      ...Object.entries({
        candidateId: 'req_other',
        artifactId: 'sha256:' + 'b'.repeat(64),
        generation: 1,
        name: 'Other',
      }).map(([key, value]) => ({ ...outcome, adoption: { ...adoption, [key]: value } })),
    ])
      await expect(service(value).adopt(input)).rejects.toThrow();
    await expect(
      service({ ...outcome, requestId: 'req_other' }).outcome({
        projectId: input.projectId,
        draftId: input.draftId,
        requestId: input.requestId,
      }),
    ).rejects.toThrow();
  });
});
