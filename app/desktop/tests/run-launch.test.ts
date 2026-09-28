import { describe, it, expect, vi } from 'vitest';
import { readFileSync } from 'node:fs';
import {
  parse,
  PipelinePreparationSchema,
  LaunchReviewSchema,
  validateLaunchReview,
  type PipelinePreparation,
  type LaunchReview,
} from '@gobble/contracts';
import { RunLaunchService } from '../src/main/service/run-launch';
const hash = 'sha256:' + 'a'.repeat(64);
const fixture = JSON.parse(
  readFileSync(new URL('./fixtures/creation-candidate.json', import.meta.url), 'utf8'),
);
function preparation(): PipelinePreparation {
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

function review(): LaunchReview {
  return parse(LaunchReviewSchema, {
    projectId: 'prj_one',
    pipelineId: 'pip_one',
    requestId: 'req_review',
    preparationId: 'req_prepared',
    createdAt: '2026-09-12T00:00:00Z',
    state: 'ready',
    issue: '',
    copiedBytes: 100,
    totalBytes: 100,
    readCount: 1,
    outputPath: 'runs/analysis-test',
    inputSHA256: hash,
    fresh: true,
    preparation: preparation(),
  });
}
const result = (v: unknown) => ({ schemaVersion: 1, ok: true, value: v });
describe('run launch trust boundary', () => {
  it('rejects private fields, mismatched preparation, and unsealed ready state', () => {
    validateLaunchReview(review());
    expect(() => parse(LaunchReviewSchema, { ...review(), payload: 'private' })).toThrow();
    for (const bad of [
      { ...review(), preparationId: 'req_other' },
      { ...review(), copiedBytes: 99 },
      { ...review(), state: 'accepted' },
      { ...review(), inputSHA256: '' },
    ]) {
      expect(() => validateLaunchReview(parse(LaunchReviewSchema, bad))).toThrow();
    }
  });
  it('refuses cross-project and duplicate retained reviews', async () => {
    const request = vi.fn().mockResolvedValue(result([review(), review()]));
    const service = new RunLaunchService({ request });
    await expect(service.list({ projectId: 'prj_one' })).rejects.toThrow('Duplicate');
    request.mockResolvedValue(result([review()]));
    await expect(service.list({ projectId: 'prj_other' })).rejects.toThrow('another');
  });
  it('sends typed Start only to exact review and rejects another acknowledgement', async () => {
    const request = vi.fn().mockResolvedValue(result(review()));
    const service = new RunLaunchService({ request });
    await expect(
      service.start({ projectId: 'prj_one', reviewId: 'req_review', requestId: 'req_start' }),
    ).rejects.toThrow('acknowledgement');
    expect(request).toHaveBeenCalledWith(
      '/v1/projects/prj_one/launch-reviews/req_review/start',
      'POST',
      { requestId: 'req_start' },
    );
  });
  it('keeps execution methods out of the Agent tool inventory', () => {
    const tools = readFileSync(
      new URL('../src/main/shared-context/catalog.ts', import.meta.url),
      'utf8',
    );
    expect(tools).not.toContain('start_launch');
    expect(tools).not.toContain('stop_launch');
  });
});
