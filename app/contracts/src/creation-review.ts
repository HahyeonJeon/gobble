import { Type, type Static } from '@sinclair/typebox';
import { closed, ProjectIdSchema, RequestIdSchema, AgentIdSchema } from './identity';
import {
  CreationDraftIdSchema,
  CreationDraftSchema,
  CreationInputObservationSchema,
  PipelineDraftInputSchema,
} from './pipeline-creation';
import {
  PipelineFlowSchema,
  PipelineSettingSchema,
  validatePipelineFlow,
} from './pipeline-inspection';
import { PipelineProposalFileSchema } from './pipeline-proposal';
import { serviceResult } from './result';
import { ContractValidationError } from './validation-error';
const digest = Type.String({ pattern: '^sha256:[a-f0-9]{64}$' });
const generation = Type.Integer({ minimum: 1, maximum: Number.MAX_SAFE_INTEGER });
const id = Type.String({ minLength: 1, maxLength: 256 });
export const CreationTargetSchema = Type.Union([
  Type.Object(
    {
      kind: Type.Union([Type.Literal('input'), Type.Literal('step'), Type.Literal('connection')]),
      id,
    },
    closed,
  ),
  Type.Object({ kind: Type.Literal('setting'), id, name: id }, closed),
  Type.Object(
    {
      kind: Type.Literal('port'),
      id,
      name: id,
      direction: Type.Union([Type.Literal('input'), Type.Literal('output')]),
    },
    closed,
  ),
]);
export const CreationContextSchema = Type.Union([
  Type.Object({ kind: Type.Literal('draft'), draftId: CreationDraftIdSchema, generation }, closed),
  Type.Object(
    {
      kind: Type.Literal('candidate'),
      base: Type.Literal('none'),
      draftId: CreationDraftIdSchema,
      generation,
      candidateId: RequestIdSchema,
      artifactId: digest,
      target: Type.Optional(CreationTargetSchema),
    },
    closed,
  ),
]);
export const CreationMarkSchema = Type.Object(
  {
    context: CreationContextSchema,
    agentId: AgentIdSchema,
    submissionId: RequestIdSchema,
    note: Type.String({ maxLength: 500 }),
  },
  closed,
);
const definition = Type.Object(
  {
    schemaVersion: Type.Literal(1),
    context: digest,
    steps: Type.Array(
      Type.Object(
        {
          id,
          fingerprint: digest,
          residual: digest,
          recipe: Type.String({ maxLength: 100 }),
          settings: Type.Array(PipelineSettingSchema, { maxItems: 32 }),
        },
        closed,
      ),
      { maxItems: 500 },
    ),
    edges: Type.Array(
      Type.Object(
        {
          fromTask: Type.String({ maxLength: 256 }),
          fromPort: id,
          toTask: id,
          toPort: id,
          wait: Type.Array(Type.String({ maxLength: 4096 }), { maxItems: 500 }),
        },
        closed,
      ),
      { maxItems: 2000 },
    ),
  },
  closed,
);
export const CreationCandidateSchema = Type.Object(
  {
    projectId: ProjectIdSchema,
    draftId: CreationDraftIdSchema,
    candidateId: RequestIdSchema,
    generation,
    digest,
    state: Type.Union(
      ['checking', 'ready', 'unsupported', 'failed', 'cancelled'].map((v) => Type.Literal(v)),
    ),
    issue: Type.String({ maxLength: 4000 }),
    summary: Type.String({ maxLength: 2000 }),
    createdAt: Type.String({ maxLength: 64 }),
    artifact: Type.Optional(
      Type.Object(
        {
          artifactId: digest,
          sourceRevision: digest,
          input: CreationInputObservationSchema,
          runtimeId: digest,
          checkedAt: Type.String({ maxLength: 64 }),
          check: Type.Object(
            {
              schemaVersion: Type.Literal(1),
              scope: Type.Literal('single-end-trim-fastqc-v1'),
              inputPath: Type.String({ pattern: '^inputs/reads\\.(fastq|fq)(\\.gz)?$' }),
              review: Type.Object(
                { schemaVersion: Type.Literal(1), flow: PipelineFlowSchema.anyOf[1], definition },
                closed,
              ),
              gaps: Type.Array(Type.String({ maxLength: 2000 }), { maxItems: 100 }),
            },
            closed,
          ),
        },
        closed,
      ),
    ),
  },
  closed,
);
export const CreationScopeSchema = Type.Object(
  {
    inputPath: Type.String({ maxLength: 256 }),
    scopeId: digest,
    draft: CreationDraftSchema,
    runtimeId: digest,
    guidance: Type.String({ maxLength: 8000 }),
    files: Type.Array(PipelineProposalFileSchema, { minItems: 1, maxItems: 1 }),
  },
  closed,
);
export const CreationCandidateInputSchema = Type.Object(
  { ...PipelineDraftInputSchema.properties, candidateId: RequestIdSchema },
  closed,
);
export const CreationSourceInputSchema = Type.Object(
  { ...PipelineDraftInputSchema.properties, generation },
  closed,
);
export const CreationProposeInputSchema = Type.Object(
  {
    ...PipelineDraftInputSchema.properties,
    requestId: RequestIdSchema,
    expectedGeneration: generation,
    scopeId: digest,
    summary: Type.String({ maxLength: 2000 }),
    files: Type.Array(PipelineProposalFileSchema, { minItems: 1, maxItems: 1 }),
  },
  closed,
);
export const CreationSelectSchema = Type.Object(
  { projectId: ProjectIdSchema, context: Type.Union([CreationContextSchema, Type.Null()]) },
  closed,
);
export const CreationStateSchema = Type.Object(
  {
    draft: CreationDraftSchema,
    candidates: Type.Array(CreationCandidateSchema, { maxItems: 8 }),
    marks: Type.Array(CreationMarkSchema, { maxItems: 64 }),
  },
  closed,
);
export const CreationEnginesSchema = Type.Object(
  {
    ready: Type.Boolean(),
    issue: Type.String({ maxLength: 2000 }),
    engines: Type.Array(
      Type.Object({ engineId: digest, label: Type.String({ maxLength: 200 }) }, closed),
      { maxItems: 8 },
    ),
  },
  closed,
);
export const CreationConnectSchema = Type.Object({ engineId: digest }, closed);
export const CreationCandidateResultSchema = serviceResult(CreationCandidateSchema);
export const CreationCandidatesResultSchema = serviceResult(
  Type.Array(CreationCandidateSchema, { maxItems: 8 }),
);
export const CreationScopeResultSchema = serviceResult(CreationScopeSchema);
export const CreationStateResultSchema = serviceResult(CreationStateSchema);
export const CreationEnginesResultSchema = serviceResult(CreationEnginesSchema);
export type CreationCandidate = Static<typeof CreationCandidateSchema>;
export type CreationScope = Static<typeof CreationScopeSchema>;
export type CreationContext = Static<typeof CreationContextSchema>;
export type CreationTarget = Static<typeof CreationTargetSchema>;
export type CreationMark = Static<typeof CreationMarkSchema>;
export type CreationState = Static<typeof CreationStateSchema>;
export type CreationCandidateInput = Static<typeof CreationCandidateInputSchema>;
export type CreationSourceInput = Static<typeof CreationSourceInputSchema>;
export type CreationProposeInput = Static<typeof CreationProposeInputSchema>;
export type CreationSelect = Static<typeof CreationSelectSchema>;
export function validateCreationCandidate(c: CreationCandidate) {
  if (c.state === 'ready' && (!c.artifact || c.artifact.check.gaps.length))
    throw new ContractValidationError('Incomplete creation check.');
  if (c.artifact) validatePipelineFlow(c.artifact.check.review.flow);
}
/** Resolve immutable scientific objects. No source text or inferred command semantics. */
export function creationFacts(
  c: CreationCandidate,
  context: Extract<CreationContext, { kind: 'candidate' }>,
) {
  if (
    !c.artifact ||
    c.draftId !== context.draftId ||
    c.generation !== context.generation ||
    c.candidateId !== context.candidateId ||
    c.artifact.artifactId !== context.artifactId
  )
    throw new ContractValidationError('This reference belongs to another creation artifact.');
  const flow = c.artifact.check.review.flow,
    t = context.target;
  if (!t) return { label: 'Proposed pipeline', flow, input: c.artifact.input };
  const step = flow.steps.find((s) => s.id === t.id);
  const object =
    t.kind === 'step'
      ? step
      : t.kind === 'input'
        ? flow.inputs.find((p) => p.name === t.id)
        : t.kind === 'connection'
          ? flow.connections.find((e) => e.id === t.id)
          : t.kind === 'setting'
            ? step?.settings.find((s) => s.key === t.name)
            : t.kind === 'port'
              ? step?.[t.direction === 'input' ? 'inputs' : 'outputs'].find(
                  (p) => p.name === t.name,
                )
              : undefined;
  if (!object)
    throw new ContractValidationError('This addition is unavailable in the checked design.');
  const label =
    t.kind === 'input'
      ? c.artifact.input.relativePath
      : t.kind === 'step'
        ? step?.display.stage || step?.name || 'Step'
        : t.kind === 'setting' && 'label' in object
          ? String(object.label)
          : t.kind === 'connection'
            ? 'Data connection'
            : t.kind === 'port'
              ? t.name
              : 'Setting';
  return { label, target: t, object, input: c.artifact.input };
}
