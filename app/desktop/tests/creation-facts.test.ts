import { it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import {
  parse,
  CreationCandidateSchema,
  creationFacts,
  validateCreationCandidate,
  type CreationContext,
} from '@gobble/contracts';
import { PipelineCreationService } from '../src/main/service/pipeline-creation';
const candidate = parse(
  CreationCandidateSchema,
  JSON.parse(readFileSync(new URL('./fixtures/creation-candidate.json', import.meta.url), 'utf8')),
);
const context: Extract<CreationContext, { kind: 'candidate' }> = {
  kind: 'candidate',
  base: 'none',
  draftId: candidate.draftId,
  generation: candidate.generation,
  candidateId: candidate.candidateId,
  artifactId: candidate.artifact!.artifactId,
};
it('resolves exact native checked input, setting, port and connection identities', () => {
  validateCreationCandidate(candidate);
  expect(
    creationFacts(candidate, {
      ...context,
      target: { kind: 'setting', id: 'trim_galore', name: 'quality' },
    }),
  ).toMatchObject({ label: 'Quality threshold', object: { value: 25, unit: 'Phred' } });
  expect(
    creationFacts(candidate, { ...context, target: { kind: 'input', id: 'reads' } }),
  ).toMatchObject({ label: 'sample.fastq.gz' });
  const edge = candidate.artifact!.check.review.flow.connections[0]!;
  expect(
    creationFacts(candidate, { ...context, target: { kind: 'connection', id: edge.id } }),
  ).toMatchObject({ object: edge });
  for (const invalid of [
    { ...context, generation: 1 },
    { ...context, artifactId: 'sha256:' + 'f'.repeat(64) },
    { ...context, target: { kind: 'setting' as const, id: 'fastqc', name: 'quality' } },
  ])
    expect(() => creationFacts(candidate, invalid)).toThrow();
});
it('rejects a native response crossing candidate or Project association', async () => {
  const service = new PipelineCreationService({
    request: async () => ({ schemaVersion: 1, ok: true, value: candidate }),
  });
  await expect(
    service.read({
      projectId: candidate.projectId,
      draftId: candidate.draftId,
      candidateId: candidate.candidateId,
    }),
  ).resolves.toEqual(candidate);
  await expect(
    service.read({
      projectId: 'prj_foreign',
      draftId: candidate.draftId,
      candidateId: candidate.candidateId,
    }),
  ).rejects.toThrow();
  await expect(
    service.read({
      projectId: candidate.projectId,
      draftId: candidate.draftId,
      candidateId: 'req_other',
    }),
  ).rejects.toThrow();
});
