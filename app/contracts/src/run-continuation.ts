import { Type, type Static } from '@sinclair/typebox';
import {
  closed,
  ProjectIdSchema,
  PipelineIdSchema,
  RequestIdSchema,
  RunRefSchema,
} from './identity';
import { serviceResult } from './result';
import { ContractValidationError } from './validation-error';

const digest = Type.String({ pattern: '^sha256:[a-f0-9]{64}$' });
const lease = Type.String({ pattern: '^[a-f0-9]{32,128}$' });
const text = Type.String({ maxLength: 2000 });
const choices = <T extends string>(...values: T[]) =>
  Type.Union(values.map((v) => Type.Literal(v)));
export const ContinuationProjectInputSchema = Type.Object({ projectId: ProjectIdSchema }, closed);
export const ContinuationSupportInputSchema = Type.Object(
  { projectId: ProjectIdSchema, launchReviewId: RequestIdSchema },
  closed,
);
export const ContinuationInputSchema = Type.Object(
  { projectId: ProjectIdSchema, reviewId: RequestIdSchema },
  closed,
);
export const ContinuationCheckInputSchema = Type.Object(
  {
    ...ContinuationSupportInputSchema.properties,
    requestId: RequestIdSchema,
    runRef: RunRefSchema,
  },
  closed,
);
export const ContinuationConfirmInputSchema = Type.Object(
  { ...ContinuationInputSchema.properties, requestId: RequestIdSchema, reviewDigest: digest },
  closed,
);
export const ContinuationStopInputSchema = Type.Object(
  { ...ContinuationInputSchema.properties, requestId: RequestIdSchema, expectedLease: lease },
  closed,
);
export const ContinuationSupportSchema = Type.Object(
  { ...ContinuationSupportInputSchema.properties, supported: Type.Boolean(), reason: text },
  closed,
);
export const ContinuationEvidenceSchema = Type.Object(
  {
    schemaVersion: Type.Literal(1),
    workspaceId: RequestIdSchema,
    originDigest: digest,
    previousHead: digest,
    snapshot: lease,
    stateDigest: digest,
    steps: Type.Array(
      Type.Object(
        {
          taskId: Type.String({ minLength: 1, maxLength: 100 }),
          priorAttempt: Type.Integer({ minimum: 1, maximum: 33 }),
          plannedAttempt: Type.Integer({ minimum: 1, maximum: 33 }),
          action: choices('reuse', 'restart', 'start'),
        },
        closed,
      ),
      { minItems: 2, maxItems: 2 },
    ),
    files: Type.Array(
      Type.Object(
        {
          path: Type.String({ minLength: 1, maxLength: 4096 }),
          sha256: Type.String({ pattern: '^[a-f0-9]{64}$' }),
        },
        closed,
      ),
      { minItems: 1, maxItems: 5 },
    ),
    digest,
  },
  closed,
);
export const ContinuationIntentSchema = Type.Object(
  {
    schemaVersion: Type.Literal(1),
    requestId: RequestIdSchema,
    workspaceId: RequestIdSchema,
    originDigest: digest,
    reviewDigest: digest,
    previousHead: digest,
  },
  closed,
);
export const ContinuationReceiptSchema = Type.Object(
  { intent: ContinuationIntentSchema, intentDigest: digest, lease, snapshot: lease },
  closed,
);
export const ContinuationObservationSchema = Type.Object(
  {
    schemaVersion: Type.Literal(2),
    admission: Type.Object(
      {
        schemaVersion: Type.Literal(2),
        intentDigest: digest,
        requestId: RequestIdSchema,
        preparedDigest: digest,
        workspaceId: RequestIdSchema,
        lease,
      },
      closed,
    ),
    status: choices('running', 'stopping', 'stopped', 'interrupted', 'succeeded', 'failed'),
    snapshot: lease,
    ownerActive: Type.Boolean(),
    ownerLive: Type.Boolean(),
    epoch: Type.Object({ head: digest, lease }, closed),
  },
  closed,
);
export const ContinuationOperationSchema = Type.Object(
  {
    requestId: RequestIdSchema,
    state: choices('accepted', 'dispatching', 'admitted', 'unknown', 'rejected'),
    issue: text,
    receipt: Type.Optional(ContinuationReceiptSchema),
    observation: Type.Optional(ContinuationObservationSchema),
    stopRequestId: Type.Union([RequestIdSchema, Type.Literal('')]),
    stopLease: Type.Union([lease, Type.Literal('')]),
    stopState: choices('', 'requested', 'settled', 'owner-changed', 'recovery-required'),
  },
  closed,
);
export const ContinuationSchema = Type.Object(
  {
    projectId: ProjectIdSchema,
    pipelineId: PipelineIdSchema,
    launchReviewId: RequestIdSchema,
    runRef: RunRefSchema,
    requestId: RequestIdSchema,
    createdAt: Type.String({ minLength: 1, maxLength: 64 }),
    state: choices('checking', 'ready', 'blocked', 'interrupted', 'accepted'),
    issue: text,
    review: Type.Optional(ContinuationEvidenceSchema),
    operation: Type.Optional(ContinuationOperationSchema),
  },
  closed,
);
export type Continuation = Static<typeof ContinuationSchema>;
export type ContinuationInput = Static<typeof ContinuationInputSchema>;
export type ContinuationCheckInput = Static<typeof ContinuationCheckInputSchema>;
export type ContinuationConfirmInput = Static<typeof ContinuationConfirmInputSchema>;
export type ContinuationStopInput = Static<typeof ContinuationStopInputSchema>;
export type ContinuationSupportInput = Static<typeof ContinuationSupportInputSchema>;
export const ContinuationResultSchema = serviceResult(ContinuationSchema);
export const ContinuationsResultSchema = serviceResult(
  Type.Array(ContinuationSchema, { maxItems: 64 }),
);
export const ContinuationSupportResultSchema = serviceResult(ContinuationSupportSchema);

/** Validate the association and state meaning that JSON field shapes cannot express. */
export function validateContinuation(v: Continuation): void {
  const bad = () => {
    throw new ContractValidationError(
      'The continuation does not match its saved review or execution.',
    );
  };
  const r = v.review,
    op = v.operation;
  if ((v.state === 'accepted') !== !!op || (v.state === 'ready' || v.state === 'accepted') !== !!r)
    bad();
  if (!r) return;
  if (r.workspaceId !== v.launchReviewId) bad();
  const ids = new Set<string>();
  let unfinished = false;
  for (const step of r.steps) {
    if (
      ids.has(step.taskId) ||
      (step.action === 'reuse' && unfinished) ||
      step.plannedAttempt !== step.priorAttempt + (step.action === 'restart' ? 1 : 0)
    )
      bad();
    ids.add(step.taskId);
    if (step.action !== 'reuse') unfinished = true;
  }
  if (!unfinished || new Set(r.files.map((f) => f.path)).size !== r.files.length) bad();
  for (const f of r.files)
    if (
      f.path.startsWith('/') ||
      f.path.includes('\\') ||
      /[\x00-\x1f]/.test(f.path) ||
      f.path.split('/').some((p) => p === '' || p === '.' || p === '..')
    )
      bad();
  if (!op) return;
  if ((op.state === 'admitted') !== !!op.receipt || (!!op.observation && !op.receipt)) bad();
  const receipt = op.receipt;
  if (
    receipt &&
    (receipt.intent.requestId !== op.requestId ||
      receipt.intent.workspaceId !== r.workspaceId ||
      receipt.intent.originDigest !== r.originDigest ||
      receipt.intent.reviewDigest !== r.digest ||
      receipt.intent.previousHead !== r.previousHead)
  )
    bad();
  if (
    op.observation &&
    (op.observation.admission.intentDigest !== r.originDigest ||
      op.observation.admission.workspaceId !== r.workspaceId)
  )
    bad();
  if (
    !!op.stopRequestId !== !!op.stopLease ||
    !!op.stopRequestId !== !!op.stopState ||
    (op.stopLease && op.stopLease !== receipt?.lease)
  )
    bad();
}
export const CONTINUATION_CHANNELS = {
  support: 'gobble:continuation:support:v1',
  list: 'gobble:continuation:list:v1',
  read: 'gobble:continuation:read:v1',
  check: 'gobble:continuation:check:v1',
  confirm: 'gobble:continuation:confirm:v1',
  refresh: 'gobble:continuation:refresh:v1',
  stop: 'gobble:continuation:stop:v1',
} as const;
export type ContinuationBridge = Readonly<{
  support: (v: ContinuationSupportInput) => Promise<Static<typeof ContinuationSupportResultSchema>>;
  list: (
    v: Static<typeof ContinuationProjectInputSchema>,
  ) => Promise<Static<typeof ContinuationsResultSchema>>;
  read: (v: ContinuationInput) => Promise<Static<typeof ContinuationResultSchema>>;
  check: (v: ContinuationCheckInput) => Promise<Static<typeof ContinuationResultSchema>>;
  confirm: (v: ContinuationConfirmInput) => Promise<Static<typeof ContinuationResultSchema>>;
  refresh: (v: ContinuationInput) => Promise<Static<typeof ContinuationResultSchema>>;
  stop: (v: ContinuationStopInput) => Promise<Static<typeof ContinuationResultSchema>>;
}>;
