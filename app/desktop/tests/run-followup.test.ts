import { describe, it, expect, vi } from 'vitest';
import {
  parse,
  EvidenceManifestSchema,
  RunFollowUpSchema,
  validateRunFollowUp,
  followUpDataRelation,
  toolSchemas,
} from '@gobble/contracts';
import { buildRunFollowUp } from '../src/main/pipeline-review/run-followup';
import type { ToolContext } from '../src/main/shared-context/ports';
import type { ProjectService } from '../src/main/service/project-service';
const hash = 'sha256:' + 'a'.repeat(64);
function manifest(runRef = 'run_origin') {
  const asset = { hash, byteLength: 10, mediaType: 'application/json' };
  const representation = { kind: 'run', truncated: true };
  return parse(EvidenceManifestSchema, {
    attachmentId: 'att_one',
    label: 'FastQC · attempt 1',
    createdAt: 1,
    capturedAt: 1,
    evidence: {
      schemaVersion: 3,
      projectId: 'prj_one',
      resource: { kind: 'run', runRef },
      dataRevision: 'snapshot_original',
      selection: {
        kind: 'run-task',
        coordinateSpace: 'observed-instance-attempt',
        instanceId: 'fastqc',
        attempt: 1,
      },
    },
    asset,
    representation,
    capture: { kind: 'observed', capturedAt: 1, asset, representation },
  });
}
function fixture() {
  const evidence = manifest();
  const context = {
    agent: { projectId: 'prj_one' },
    submission: {
      requestId: 'req_sent',
      pipelineReview: { pipelineId: 'pip_one', baseArtifactId: hash },
      evidence: [evidence],
    },
  } as unknown as ToolContext;
  const list = vi
    .fn()
    .mockResolvedValue([
      { requestId: 'req_launch', pipelineId: 'pip_one', operation: { runRef: 'run_origin' } },
    ]);
  const service = { launches: { list } } as unknown as ProjectService;
  const workspace = { readSentEvidence: vi.fn().mockResolvedValue(evidence) };
  const verify = vi.fn().mockResolvedValue(undefined);
  return { context, service, workspace, verify, list };
}
describe('follow-up origin authority', () => {
  it('uses actual sent evidence and checks its stored bytes before proposing', async () => {
    const f = fixture();
    const origin = await buildRunFollowUp(f.context, f.service, f.workspace, f.verify, () => {});
    expect(origin).toMatchObject({ launchReviewId: 'req_launch', submissionId: 'req_sent' });
    expect(f.workspace.readSentEvidence).toHaveBeenCalledWith('prj_one', 'req_sent', 'att_one');
    expect(f.verify).toHaveBeenCalledWith('prj_one', manifest());
  });
  it('does not invent lineage for a message without Run evidence', async () => {
    const f = fixture();
    delete f.context.submission.evidence;
    expect(
      await buildRunFollowUp(f.context, f.service, f.workspace, f.verify, () => {}),
    ).toBeUndefined();
    expect(f.list).not.toHaveBeenCalled();
  });
  it('rejects mixed Runs and another Pipeline before authoring', async () => {
    const f = fixture();
    f.context.submission.evidence!.push({ ...manifest('run_other'), attachmentId: 'att_two' });
    await expect(
      buildRunFollowUp(f.context, f.service, f.workspace, f.verify, () => {}),
    ).rejects.toThrow('one analysis');
    f.context.submission.evidence = [manifest()];
    f.list.mockResolvedValue([
      { requestId: 'req_launch', pipelineId: 'pip_other', operation: { runRef: 'run_origin' } },
    ]);
    await expect(
      buildRunFollowUp(f.context, f.service, f.workspace, f.verify, () => {}),
    ).rejects.toThrow('Current design');
  });
  it('refuses unavailable, altered or stale-session evidence', async () => {
    const f = fixture();
    f.verify.mockRejectedValue(new Error('Captured bytes missing'));
    await expect(
      buildRunFollowUp(f.context, f.service, f.workspace, f.verify, () => {}),
    ).rejects.toThrow('missing');
    f.verify.mockResolvedValue(undefined);
    f.workspace.readSentEvidence.mockResolvedValue({ ...manifest(), label: 'another' });
    await expect(
      buildRunFollowUp(f.context, f.service, f.workspace, f.verify, () => {}),
    ).rejects.toThrow('does not match');
    await expect(
      buildRunFollowUp(f.context, f.service, f.workspace, f.verify, () => {
        throw Error('Project switched');
      }),
    ).rejects.toThrow('switched');
  });
  it('does not expose origin injection to the Agent tool', () => {
    expect(() =>
      parse(toolSchemas.gobble_pipeline_propose, {
        summary: 'change',
        files: [{ path: 'pipeline.go', content: 'source' }],
        followUp: { launchReviewId: 'req_forged' },
      }),
    ).toThrow();
  });
  it('keeps capture coordinates, association and data difference truthful', () => {
    const value = parse(RunFollowUpSchema, {
      schemaVersion: 1,
      projectId: 'prj_one',
      pipelineId: 'pip_one',
      launchReviewId: 'req_launch',
      preparationId: 'req_prep',
      runRef: 'run_origin',
      runName: 'Analysis 12',
      artifactId: hash,
      inputSHA256: hash,
      submissionId: 'req_sent',
      evidence: [manifest()],
    });
    validateRunFollowUp(value, 'prj_one', 'pip_one');
    expect(value.evidence[0]!.evidence.selection).toMatchObject({
      instanceId: 'fastqc',
      attempt: 1,
    });
    expect(() =>
      validateRunFollowUp({ ...value, evidence: [manifest('run_other')] }, 'prj_one', 'pip_one'),
    ).toThrow();
    expect(() => validateRunFollowUp(value, 'prj_other', 'pip_one')).toThrow();
    expect(() =>
      validateRunFollowUp({ ...value, evidence: [manifest(), manifest()] }, 'prj_one', 'pip_one'),
    ).toThrow();
    const altered = manifest();
    altered.capturedAt = 2;
    expect(() =>
      validateRunFollowUp({ ...value, evidence: [altered] }, 'prj_one', 'pip_one'),
    ).toThrow();
    expect(followUpDataRelation('', value)).toContain('pending');
    expect(followUpDataRelation(hash, value)).toContain('Same input');
    expect(followUpDataRelation('sha256:' + 'b'.repeat(64), value)).toContain('differs');
  });
});
