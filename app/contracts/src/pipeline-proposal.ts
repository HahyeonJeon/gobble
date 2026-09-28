import { RunFollowUpSchema, RunFollowUpInputSchema, validateRunFollowUp } from './run-followup';
import { Type, type Static } from '@sinclair/typebox';
import { closed, ProjectIdSchema, RequestIdSchema, AgentIdSchema } from './identity';
import { PipelineIdSchema } from './identity';
import { PipelineArtifactSchema, PipelineInspectionInputSchema } from './pipeline-inspection';
import { serviceResult } from './result';
import { ContractValidationError } from './validation-error';
import { validatePipelineFlow } from './pipeline-inspection';
const ReviewDigestSchema = Type.String({ pattern: '^sha256:[a-f0-9]{64}$' });
const text = (maxLength: number) => Type.String({ maxLength });
export const PipelineProposalFileSchema = Type.Object(
  { path: text(4096), content: Type.String({ minLength: 1, maxLength: 32768 }) },
  closed,
);
export const PipelineChangeSchema = Type.Object(
  {
    id: Type.String({ minLength: 1, maxLength: 600 }),
    kind: Type.Union([Type.Literal('setting'), Type.Literal('added-step')]),
    stepId: text(256),
    key: text(100),
    label: text(200),
    unit: text(100),
    before: Type.Union([Type.Integer(), Type.Null()]),
    after: Type.Union([Type.Integer(), Type.Null()]),
  },
  closed,
);
export const PipelineProposalSchema = Type.Object(
  {
    followUp: Type.Optional(RunFollowUpSchema),
    createdAt: Type.String({ minLength: 1, maxLength: 64 }),
    projectId: ProjectIdSchema,
    pipelineId: PipelineIdSchema,
    proposalId: RequestIdSchema,
    digest: ReviewDigestSchema,
    state: Type.Union([Type.Literal('checking'), Type.Literal('ready'), Type.Literal('failed')]),
    summary: text(2000),
    issue: text(2000),
    base: PipelineArtifactSchema,
    proposed: Type.Optional(PipelineArtifactSchema),
    managed: Type.Boolean(),
    comparison: Type.Optional(
      Type.Object(
        {
          schemaVersion: Type.Literal(1),
          changes: Type.Array(PipelineChangeSchema, { maxItems: 1000 }),
          gaps: Type.Array(text(1000), { maxItems: 5000 }),
        },
        closed,
      ),
    ),
  },
  closed,
);
/** Immutable comparison coordinates. Separate from current-flow v7 targets: a
 * comparison always binds both checked artifacts and the selected change. */
export const PipelineReviewContextSchema = Type.Object(
  {
    pipelineId: PipelineIdSchema,
    baseArtifactId: ReviewDigestSchema,
    preparationId: Type.Optional(RequestIdSchema),
    preparationSection: Type.Optional(
      Type.Union([Type.Literal('data'), Type.Literal('settings'), Type.Literal('environment')]),
    ),
    proposalId: Type.Optional(RequestIdSchema),
    proposedArtifactId: Type.Optional(ReviewDigestSchema),
    changeId: Type.Optional(text(600)),
    side: Type.Optional(
      Type.Union([Type.Literal('current'), Type.Literal('proposed'), Type.Literal('both')]),
    ),
  },
  closed,
);
export const PipelineReviewMarkSchema = Type.Object(
  {
    context: PipelineReviewContextSchema,
    agentId: AgentIdSchema,
    submissionId: RequestIdSchema,
    note: text(500),
  },
  closed,
);
export const PipelineProposalListSchema = Type.Object(
  {
    projectId: ProjectIdSchema,
    pipelineId: PipelineIdSchema,
    currentArtifactId: Type.Union([ReviewDigestSchema, Type.Literal('')]),
    managed: Type.Boolean(),
    proposals: Type.Array(PipelineProposalSchema, { maxItems: 8 }),
  },
  closed,
);
export const PipelineReviewStateSchema = Type.Object(
  {
    ...PipelineProposalListSchema.properties,
    marks: Type.Array(PipelineReviewMarkSchema, { maxItems: 64 }),
  },
  closed,
);
export const PipelineProposalSourceSchema = Type.Object(
  {
    guidance: Type.String({ maxLength: 4000 }),
    projectId: ProjectIdSchema,
    pipelineId: PipelineIdSchema,
    baseArtifactId: ReviewDigestSchema,
    files: Type.Array(PipelineProposalFileSchema, { minItems: 1, maxItems: 16 }),
  },
  closed,
);
export const PipelineProposeInputSchema = Type.Object(
  {
    followUp: Type.Optional(RunFollowUpInputSchema),
    ...PipelineInspectionInputSchema.properties,
    requestId: RequestIdSchema,
    baseArtifactId: ReviewDigestSchema,
    summary: text(2000),
    files: Type.Array(PipelineProposalFileSchema, { minItems: 1, maxItems: 16 }),
  },
  closed,
);
export const PipelineProposalSourceInputSchema = Type.Object(
  { ...PipelineInspectionInputSchema.properties, baseArtifactId: ReviewDigestSchema },
  closed,
);
export const PipelineAdoptInputSchema = Type.Object(
  {
    ...PipelineInspectionInputSchema.properties,
    requestId: RequestIdSchema,
    proposalId: RequestIdSchema,
    artifactId: ReviewDigestSchema,
  },
  closed,
);
export const PipelineAdoptionInputSchema = Type.Object(
  { ...PipelineInspectionInputSchema.properties, requestId: RequestIdSchema },
  closed,
);
export const PipelineAdoptionSchema = Type.Object(
  {
    ...PipelineAdoptionInputSchema.properties,
    proposalId: Type.Union([RequestIdSchema, Type.Literal('')]),
    artifactId: Type.Union([ReviewDigestSchema, Type.Literal('')]),
    state: Type.Union([Type.Literal('adopted'), Type.Literal('not-recorded')]),
  },
  closed,
);
export const PipelineReviewSelectSchema = Type.Object(
  { projectId: ProjectIdSchema, context: Type.Union([PipelineReviewContextSchema, Type.Null()]) },
  closed,
);
export const PipelineReviewSelectedResultSchema = serviceResult(
  Type.Object({ selected: Type.Literal(true) }, closed),
);
export const PipelineProposalResultSchema = serviceResult(PipelineProposalSchema);
export const PipelineProposalListResultSchema = serviceResult(PipelineProposalListSchema);
export const PipelineReviewStateResultSchema = serviceResult(PipelineReviewStateSchema);
export const PipelineProposalSourceResultSchema = serviceResult(PipelineProposalSourceSchema);
export const PipelineAdoptionResultSchema = serviceResult(PipelineAdoptionSchema);
export type PipelineProposal = Static<typeof PipelineProposalSchema>;
export type PipelineChange = Static<typeof PipelineChangeSchema>;
export type PipelineReviewContext = Static<typeof PipelineReviewContextSchema>;
export type PipelineReviewMark = Static<typeof PipelineReviewMarkSchema>;
export type PipelineReviewState = Static<typeof PipelineReviewStateSchema>;
export type PipelineProposeInput = Static<typeof PipelineProposeInputSchema>;
export type PipelineAdoptInput = Static<typeof PipelineAdoptInputSchema>;
export type PipelineAdoptionInput = Static<typeof PipelineAdoptionInputSchema>;
export type PipelineProposalSourceInput = Static<typeof PipelineProposalSourceInputSchema>;
export type PipelineReviewSelect = Static<typeof PipelineReviewSelectSchema>;
export function validatePipelineProposal(value: PipelineProposal): void {
  const fail = (): never => {
    throw new ContractValidationError('The checked comparison has inconsistent identities.');
  };
  if (value.followUp) validateRunFollowUp(value.followUp, value.projectId, value.pipelineId);
  validatePipelineFlow(value.base.flow);
  if (value.proposed) validatePipelineFlow(value.proposed.flow);
  if (value.state === 'ready' && (!value.proposed || !value.comparison)) fail();
  const ids = new Set<string>();
  for (const c of value.comparison?.changes ?? []) {
    const next = value.proposed?.flow.steps.find((s) => s.id === c.stepId);
    const old = value.base.flow.steps.find((s) => s.id === c.stepId);
    if (ids.has(c.id) || !next || (c.kind === 'added-step' ? !!old : !old)) fail();
    ids.add(c.id);
    if (c.kind === 'setting') {
      const before =
        value.base.flow.schemaVersion === 2
          ? value.base.flow.steps
              .find((s) => s.id === c.stepId)
              ?.settings.find((s) => s.key === c.key)
          : undefined;
      const after =
        value.proposed?.flow.schemaVersion === 2
          ? value.proposed.flow.steps
              .find((s) => s.id === c.stepId)
              ?.settings.find((s) => s.key === c.key)
          : undefined;
      if (
        !before ||
        !after ||
        before.value !== c.before ||
        after.value !== c.after ||
        before.label !== c.label ||
        after.label !== c.label ||
        before.unit !== c.unit ||
        after.unit !== c.unit
      )
        fail();
    }
  }
}
export function reviewChange(
  value: PipelineProposal,
  context: PipelineReviewContext,
): PipelineChange | undefined {
  if (
    value.pipelineId !== context.pipelineId ||
    value.proposalId !== context.proposalId ||
    value.base.artifactId !== context.baseArtifactId ||
    value.proposed?.artifactId !== context.proposedArtifactId
  )
    throw new ContractValidationError('This reference belongs to another comparison.');
  const change = value.comparison?.changes.find((c) => c.id === context.changeId);
  if (context.changeId && !change)
    throw new ContractValidationError('This change is unavailable in the comparison.');
  return change;
}

export function validatePipelineReviewContext(context: PipelineReviewContext): void {
  if (
    (!!context.preparationId && !!context.proposalId) ||
    (!!context.preparationSection && !context.preparationId) ||
    !!context.proposalId !== !!context.proposedArtifactId ||
    (!context.proposalId && (!!context.changeId || !!context.side)) ||
    (context.changeId !== undefined && !context.changeId)
  )
    throw new ContractValidationError(
      'A comparison reference requires both artifacts and an exact change identity.',
    );
}
