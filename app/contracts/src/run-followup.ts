import { Type, type Static } from '@sinclair/typebox';
import {
  closed,
  ProjectIdSchema,
  PipelineIdSchema,
  RequestIdSchema,
  RunRefSchema,
} from './identity';
import { EvidenceManifestSchema } from './evidence';
import { ContractValidationError } from './validation-error';
const digest = Type.String({ pattern: '^sha256:[a-f0-9]{64}$' });
/** Sent evidence metadata only; captured bytes remain owned by Main's evidence store. */
export const RunFollowUpInputSchema = Type.Object(
  {
    launchReviewId: RequestIdSchema,
    submissionId: RequestIdSchema,
    evidence: Type.Array(EvidenceManifestSchema, { minItems: 1, maxItems: 16 }),
  },
  closed,
);
export const RunFollowUpSchema = Type.Object(
  {
    ...RunFollowUpInputSchema.properties,
    schemaVersion: Type.Literal(1),
    projectId: ProjectIdSchema,
    pipelineId: PipelineIdSchema,
    runRef: RunRefSchema,
    runName: Type.String({ minLength: 1, maxLength: 200 }),
    preparationId: RequestIdSchema,
    artifactId: digest,
    inputSHA256: digest,
  },
  closed,
);
export type RunFollowUpInput = Static<typeof RunFollowUpInputSchema>;
export type RunFollowUp = Static<typeof RunFollowUpSchema>;
export function validateRunFollowUp(v: RunFollowUp, projectId: string, pipelineId: string) {
  if (v.projectId !== projectId || v.pipelineId !== pipelineId)
    throw new ContractValidationError('The follow-up belongs to another Project or Pipeline.');
  const ids = new Set<string>();
  for (const item of v.evidence) {
    const target = item.evidence;
    if (
      ids.has(item.attachmentId) ||
      target.projectId !== projectId ||
      !['run', 'log'].includes(target.resource.kind) ||
      !('runRef' in target.resource) ||
      target.resource.runRef !== v.runRef ||
      !item.capture ||
      item.capture.kind !== 'observed' ||
      item.representation.kind !== target.resource.kind ||
      item.capture.capturedAt !== item.capturedAt ||
      JSON.stringify(item.capture.asset) !== JSON.stringify(item.asset) ||
      JSON.stringify(item.capture.representation) !== JSON.stringify(item.representation)
    )
      throw new ContractValidationError('Follow-up evidence must identify one exact observed Run.');
    ids.add(item.attachmentId);
  }
}
export function followUpDataRelation(inputSHA256: string, followUp: RunFollowUp): string {
  return !inputSHA256
    ? 'Data comparison pending'
    : inputSHA256 === followUp.inputSHA256
      ? 'Same input content as the earlier analysis'
      : 'Data differs from the earlier analysis';
}
