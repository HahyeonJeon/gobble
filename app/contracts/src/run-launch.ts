import { RunFollowUpSchema, validateRunFollowUp } from './run-followup';
import { Type, type Static } from '@sinclair/typebox';
import {
  closed,
  ProjectIdSchema,
  PipelineIdSchema,
  RequestIdSchema,
  RunRefSchema,
} from './identity';
import { PipelinePreparationSchema, validatePreparation } from './run-preparation';
import { serviceResult } from './result';
import { ContractValidationError } from './validation-error';
const text = Type.String({ maxLength: 2000 });
const digest = Type.String({ pattern: '^sha256:[a-f0-9]{64}$' });
const choices = <T extends string>(...values: T[]) =>
  Type.Union(values.map((v) => Type.Literal(v)));
export const LaunchProjectInputSchema = Type.Object({ projectId: ProjectIdSchema }, closed);
export const LaunchReviewInputSchema = Type.Object(
  { projectId: ProjectIdSchema, reviewId: RequestIdSchema },
  closed,
);
export const LaunchCheckInputSchema = Type.Object(
  {
    projectId: ProjectIdSchema,
    pipelineId: PipelineIdSchema,
    requestId: RequestIdSchema,
    preparationId: RequestIdSchema,
  },
  closed,
);
export const LaunchStartInputSchema = Type.Object(
  { ...LaunchReviewInputSchema.properties, requestId: RequestIdSchema },
  closed,
);
export const LaunchStopInputSchema = Type.Object(
  {
    ...LaunchStartInputSchema.properties,
    expectedLease: Type.String({ minLength: 1, maxLength: 256 }),
  },
  closed,
);
export const LaunchOperationSchema = Type.Object(
  {
    requestId: RequestIdSchema,
    state: choices(
      'accepted',
      'dispatching',
      'admitted',
      'reconciling',
      'rejected',
      'recovery-required',
    ),
    runRef: Type.Union([RunRefSchema, Type.Literal('')]),
    runStatus: text,
    snapshot: text,
    ownerLease: text,
    observedAt: text,
    issue: text,
    stopRequestId: Type.Union([RequestIdSchema, Type.Literal('')]),
    stopState: choices('', 'requested', 'settled', 'owner-changed', 'recovery-required'),
    stopLease: text,
  },
  closed,
);
export const LaunchReviewSchema = Type.Object(
  {
    followUp: Type.Optional(RunFollowUpSchema),
    projectId: ProjectIdSchema,
    pipelineId: PipelineIdSchema,
    requestId: RequestIdSchema,
    preparationId: RequestIdSchema,
    createdAt: Type.String({ minLength: 1, maxLength: 64 }),
    state: choices('checking', 'ready', 'failed', 'cancelled', 'accepted'),
    issue: text,
    copiedBytes: Type.Integer({ minimum: 0, maximum: 64 * 2 ** 30 + 65536 }),
    totalBytes: Type.Integer({ minimum: 1, maximum: 64 * 2 ** 30 }),
    readCount: Type.Integer({ minimum: 0, maximum: Number.MAX_SAFE_INTEGER }),
    outputPath: Type.String({ pattern: '^runs/analysis-[A-Za-z0-9_-]{1,80}$' }),
    inputSHA256: Type.Union([digest, Type.Literal('')]),
    fresh: Type.Boolean(),
    preparation: PipelinePreparationSchema,
    operation: Type.Optional(LaunchOperationSchema),
  },
  closed,
);
export type LaunchReview = Static<typeof LaunchReviewSchema>;
export type LaunchReviewInput = Static<typeof LaunchReviewInputSchema>;
export type LaunchCheckInput = Static<typeof LaunchCheckInputSchema>;
export type LaunchStartInput = Static<typeof LaunchStartInputSchema>;
export type LaunchStopInput = Static<typeof LaunchStopInputSchema>;
export const LaunchResultSchema = serviceResult(LaunchReviewSchema);
export const LaunchReviewsResultSchema = serviceResult(
  Type.Array(LaunchReviewSchema, { maxItems: 32 }),
);
export function validateLaunchReview(v: LaunchReview) {
  if (v.followUp) {
    validateRunFollowUp(v.followUp, v.projectId, v.pipelineId);
    if (
      v.followUp.launchReviewId === v.requestId ||
      (v.operation?.runRef && v.followUp.runRef === v.operation.runRef)
    )
      throw new ContractValidationError(
        'A fresh analysis must have its own Run and launch review.',
      );
  }
  validatePreparation(v.preparation);
  if (
    v.preparation.state !== 'ready' ||
    v.preparation.projectId !== v.projectId ||
    v.preparation.pipelineId !== v.pipelineId ||
    v.preparation.requestId !== v.preparationId ||
    (v.state === 'accepted') !== !!v.operation ||
    (v.fresh && v.state !== 'ready') ||
    ((v.state === 'ready' || v.state === 'accepted') &&
      (!v.inputSHA256 || !v.readCount || v.copiedBytes !== v.totalBytes))
  )
    throw new ContractValidationError(
      'The launch review does not match its saved preparation or state.',
    );
}
export const LAUNCH_CHANNELS = {
  list: 'gobble:launch:list:v1',
  read: 'gobble:launch:read:v1',
  check: 'gobble:launch:check:v1',
  cancel: 'gobble:launch:cancel:v1',
  start: 'gobble:launch:start:v1',
  refresh: 'gobble:launch:refresh:v1',
  stop: 'gobble:launch:stop:v1',
} as const;
export type LaunchBridge = Readonly<{
  list: (
    v: Static<typeof LaunchProjectInputSchema>,
  ) => Promise<Static<typeof LaunchReviewsResultSchema>>;
  read: (v: LaunchReviewInput) => Promise<Static<typeof LaunchResultSchema>>;
  check: (v: LaunchCheckInput) => Promise<Static<typeof LaunchResultSchema>>;
  cancel: (v: LaunchReviewInput) => Promise<Static<typeof LaunchResultSchema>>;
  start: (v: LaunchStartInput) => Promise<Static<typeof LaunchResultSchema>>;
  refresh: (v: LaunchReviewInput) => Promise<Static<typeof LaunchResultSchema>>;
  stop: (v: LaunchStopInput) => Promise<Static<typeof LaunchResultSchema>>;
}>;
