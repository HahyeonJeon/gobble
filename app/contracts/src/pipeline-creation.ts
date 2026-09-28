import { Type, type Static } from '@sinclair/typebox';
import {
  closed,
  ProjectIdSchema,
  PipelineIdSchema,
  RequestIdSchema,
  ResourceIdSchema,
} from './identity';
import { serviceResult } from './result';
import { ContractValidationError } from './validation-error';

export const CreationDraftIdSchema = Type.String({
  pattern: '^drf_[A-Za-z0-9_-]{1,80}$',
  maxLength: 90,
});
const GenerationSchema = Type.Integer({ minimum: 1, maximum: Number.MAX_SAFE_INTEGER });
const BriefSchema = Type.String({ maxLength: 8000, pattern: '^[^\\u0000]*$' });
const DigestSchema = Type.String({ pattern: '^sha256:[a-f0-9]{64}$' });
/** Metadata observation only: neither content digest nor FASTQ validation. */
export const CreationInputObservationSchema = Type.Object(
  {
    resourceId: ResourceIdSchema,
    relativePath: Type.String({ minLength: 1, maxLength: 4096 }),
    readLayout: Type.Literal('single-end'),
    size: Type.Integer({ minimum: 0, maximum: Number.MAX_SAFE_INTEGER }),
    modifiedAt: Type.String({ minLength: 1, maxLength: 64 }),
    identity: DigestSchema,
  },
  closed,
);
export const CreationAdoptionSchema = Type.Object(
  {
    requestId: RequestIdSchema,
    pipelineId: PipelineIdSchema,
    candidateId: RequestIdSchema,
    artifactId: DigestSchema,
    generation: GenerationSchema,
    name: Type.String({ minLength: 1, maxLength: 200 }),
  },
  closed,
);
export const CreationDraftSchema = Type.Object(
  {
    projectId: ProjectIdSchema,
    draftId: CreationDraftIdSchema,
    generation: GenerationSchema,
    state: Type.Union([Type.Literal('draft'), Type.Literal('discarded'), Type.Literal('adopted')]),
    brief: BriefSchema,
    input: Type.Optional(CreationInputObservationSchema),
    adoption: Type.Optional(CreationAdoptionSchema),
  },
  closed,
);
export type CreationDraft = Static<typeof CreationDraftSchema>;
export const CreatePipelineDraftInputSchema = Type.Object(
  {
    projectId: ProjectIdSchema,
    requestId: RequestIdSchema,
    brief: BriefSchema,
  },
  closed,
);
export const PipelineDraftInputSchema = Type.Object(
  { projectId: ProjectIdSchema, draftId: CreationDraftIdSchema },
  closed,
);
/** Empty resourceId/readLayout explicitly clear binding. Omission never clears it. */
export const UpdatePipelineDraftInputSchema = Type.Union([
  Type.Object(
    {
      ...PipelineDraftInputSchema.properties,
      requestId: RequestIdSchema,
      expectedGeneration: GenerationSchema,
      brief: BriefSchema,
      resourceId: ResourceIdSchema,
      readLayout: Type.Literal('single-end'),
    },
    closed,
  ),
  Type.Object(
    {
      ...PipelineDraftInputSchema.properties,
      requestId: RequestIdSchema,
      expectedGeneration: GenerationSchema,
      brief: BriefSchema,
      resourceId: Type.Literal(''),
      readLayout: Type.Literal(''),
    },
    closed,
  ),
]);
export const DiscardPipelineDraftInputSchema = Type.Object(
  {
    ...PipelineDraftInputSchema.properties,
    requestId: RequestIdSchema,
    expectedGeneration: GenerationSchema,
  },
  closed,
);
export const CreationDraftResultSchema = serviceResult(CreationDraftSchema);
export const CreationDraftsResultSchema = serviceResult(
  Type.Object(
    { projectId: ProjectIdSchema, drafts: Type.Array(CreationDraftSchema, { maxItems: 1000 }) },
    closed,
  ),
);
/** Prepared identity contract. Live Pane support arrives with the creation UI. */
export const PipelineViewSubjectSchema = Type.Union([
  Type.Object({ kind: Type.Literal('pipeline'), pipelineId: PipelineIdSchema }, closed),
  Type.Object({ kind: Type.Literal('creation-draft'), draftId: CreationDraftIdSchema }, closed),
]);
export type PipelineViewSubject = Static<typeof PipelineViewSubjectSchema>;

export type CreatePipelineDraftInput = Static<typeof CreatePipelineDraftInputSchema>;
export type UpdatePipelineDraftInput = Static<typeof UpdatePipelineDraftInputSchema>;
export type DiscardPipelineDraftInput = Static<typeof DiscardPipelineDraftInputSchema>;
export type PipelineDraftInput = Static<typeof PipelineDraftInputSchema>;

export const AdoptCreationInputSchema = Type.Object(
  {
    ...PipelineDraftInputSchema.properties,
    requestId: RequestIdSchema,
    candidateId: RequestIdSchema,
    artifactId: DigestSchema,
    expectedGeneration: GenerationSchema,
    name: Type.String({ minLength: 1, maxLength: 200 }),
  },
  closed,
);
export const CreationAdoptionInputSchema = Type.Object(
  { ...PipelineDraftInputSchema.properties, requestId: RequestIdSchema },
  closed,
);
export const CreationAdoptionOutcomeSchema = Type.Union([
  Type.Object(
    { ...CreationAdoptionInputSchema.properties, state: Type.Literal('not-recorded') },
    closed,
  ),
  Type.Object(
    {
      ...CreationAdoptionInputSchema.properties,
      state: Type.Literal('adopted'),
      adoption: CreationAdoptionSchema,
    },
    closed,
  ),
]);
export const CreationAdoptionResultSchema = serviceResult(CreationAdoptionOutcomeSchema);
export type AdoptCreationInput = Static<typeof AdoptCreationInputSchema>;
export type CreationAdoptionInput = Static<typeof CreationAdoptionInputSchema>;
export type CreationAdoption = Static<typeof CreationAdoptionSchema>;
/** Adoption is a closed lifecycle state, never an editable draft with a hidden Pipeline. */
export function validateCreationDraft(draft: CreationDraft) {
  if (
    draft.state === 'adopted'
      ? !draft.adoption || !draft.input || draft.adoption.generation !== draft.generation - 1
      : !!draft.adoption
  )
    throw new ContractValidationError('Invalid creation adoption state.');
}
