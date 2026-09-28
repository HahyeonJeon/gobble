import { Type, type Static } from '@sinclair/typebox';
import { closed, RequestIdSchema } from './identity';
import {
  PipelineFlowSchema,
  validatePipelineFlow,
  PipelineInspectionInputSchema,
} from './pipeline-inspection';
import { CreationInputObservationSchema } from './pipeline-creation';
import { serviceResult } from './result';
import { ContractValidationError } from './validation-error';
const digest = Type.String({ pattern: '^sha256:[a-f0-9]{64}$' });
export const PreparePipelineInputSchema = Type.Object(
  {
    ...PipelineInspectionInputSchema.properties,
    requestId: RequestIdSchema,
    artifactId: digest,
    engineId: digest,
  },
  closed,
);
export const PreparationInputSchema = Type.Object(
  { ...PipelineInspectionInputSchema.properties, requestId: RequestIdSchema },
  closed,
);
const prepared = Type.Object(
  {
    flow: PipelineFlowSchema,
    digest,
    binding: Type.Object(
      {
        sourceRevision: digest,
        runtimeId: digest,
        inputIdentity: digest,
        inputPath: Type.String({ pattern: '^inputs/reads\\.(fastq|fq)(\\.gz)?$' }),
        cap: Type.Literal(1),
      },
      closed,
    ),
    input: CreationInputObservationSchema,
    steps: Type.Array(
      Type.Object(
        {
          id: Type.String({ minLength: 1, maxLength: 256 }),
          label: Type.String({ maxLength: 200 }),
          cpu: Type.Number({ minimum: 0 }),
          memory: Type.String({ maxLength: 50 }),
          image: Type.String({ maxLength: 4096 }),
          outputs: Type.Array(Type.String({ maxLength: 4096 }), { maxItems: 10 }),
        },
        closed,
      ),
      { minItems: 2, maxItems: 2 },
    ),
    checkedAt: Type.String({ minLength: 1, maxLength: 64 }),
  },
  closed,
);
const common = {
  engineId: digest,
  createdAt: Type.String({ minLength: 1, maxLength: 64 }),
  ...PreparationInputSchema.properties,
  artifactId: digest,
  issue: Type.String({ maxLength: 2000 }),
};
export const PipelinePreparationSchema = Type.Union([
  Type.Object({ ...common, state: Type.Literal('ready'), prepared, fresh: Type.Boolean() }, closed),
  Type.Object(
    {
      ...common,
      state: Type.Union([
        Type.Literal('preparing'),
        Type.Literal('failed'),
        Type.Literal('cancelled'),
      ]),
      fresh: Type.Literal(false),
    },
    closed,
  ),
]);
export type PipelinePreparation = Static<typeof PipelinePreparationSchema>;
export type PreparationInput = Static<typeof PreparationInputSchema>;
export type PreparePipelineInput = Static<typeof PreparePipelineInputSchema>;
export const PreparationResultSchema = serviceResult(PipelinePreparationSchema);
export const PreparationsResultSchema = serviceResult(
  Type.Array(PipelinePreparationSchema, { maxItems: 32 }),
);
export function validatePreparation(v: PipelinePreparation) {
  if (v.state === 'ready') {
    validatePipelineFlow(v.prepared.flow);
    if (
      v.prepared.flow.inputs.length !== 1 ||
      v.prepared.flow.inputs[0]?.path !== v.prepared.binding.inputPath ||
      v.prepared.flow.steps.length !== v.prepared.steps.length ||
      v.prepared.steps.some((step, i) => step.id !== v.prepared.flow.steps[i]?.id)
    )
      throw new ContractValidationError('The prepared review and processing steps disagree.');
  }
  if (
    v.state === 'ready' &&
    new Set(v.prepared.steps.map((s) => s.id)).size !== v.prepared.steps.length
  )
    throw new ContractValidationError('The prepared steps have duplicate identities.');
}
export const PreparationEnginesInputSchema = Type.Object({}, closed);
export const PreparationEnginesResultSchema = serviceResult(
  Type.Array(Type.Object({ engineId: digest, label: Type.String({ maxLength: 200 }) }, closed), {
    maxItems: 8,
  }),
);
export const PREPARATION_CHANNELS = {
  engines: 'gobble:preparations:engines:v1',
  list: 'gobble:preparations:list:v1',
  read: 'gobble:preparations:read:v1',
  prepare: 'gobble:preparations:prepare:v1',
  cancel: 'gobble:preparations:cancel:v1',
} as const;
export type PreparationBridge = Readonly<{
  engines: () => Promise<Static<typeof PreparationEnginesResultSchema>>;
  list: (
    input: Static<typeof PipelineInspectionInputSchema>,
  ) => Promise<Static<typeof PreparationsResultSchema>>;
  read: (input: PreparationInput) => Promise<Static<typeof PreparationResultSchema>>;
  prepare: (input: PreparePipelineInput) => Promise<Static<typeof PreparationResultSchema>>;
  cancel: (input: PreparationInput) => Promise<Static<typeof PreparationResultSchema>>;
}>;
