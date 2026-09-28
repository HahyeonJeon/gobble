import { Type, type Static } from '@sinclair/typebox';
import { closed, RequestIdSchema } from './identity';
import { RunPresentationSchema } from './run-presentation';
import { ContinuationEvidenceSchema } from './run-continuation';

/** Immutable discussion context, not an execution intent or a live status. */
export const RunContinuationContextSchema = Type.Object(
  {
    schemaVersion: Type.Literal(1),
    reviewId: RequestIdSchema,
    checkedAt: Type.String({ minLength: 1, maxLength: 64 }),
    review: ContinuationEvidenceSchema,
  },
  closed,
);
export const RunPresentationV2Schema = Type.Object(
  {
    ...RunPresentationSchema.properties,
    continuation: Type.Optional(RunContinuationContextSchema),
  },
  closed,
);
export type RunPresentationV2 = Static<typeof RunPresentationV2Schema>;
export type RunContinuationContext = Static<typeof RunContinuationContextSchema>;

export function continuationStepLabel(
  step: RunContinuationContext['review']['steps'][number],
): string {
  return `${step.action === 'reuse' ? 'Can reuse' : step.action === 'restart' ? 'Will restart' : 'Will start'} · Attempt ${step.plannedAttempt}`;
}
