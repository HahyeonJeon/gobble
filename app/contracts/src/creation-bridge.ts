import type { Static } from '@sinclair/typebox';
import type * as C from './creation-review';
import type * as D from './pipeline-creation';
import type { PipelineReviewSelectedResultSchema } from './pipeline-proposal';
export type CreationBridge = Readonly<{
  adopt: (input: D.AdoptCreationInput) => Promise<Static<typeof D.CreationAdoptionResultSchema>>;
  outcome: (
    input: D.CreationAdoptionInput,
  ) => Promise<Static<typeof D.CreationAdoptionResultSchema>>;
  list: (input: { projectId: string }) => Promise<Static<typeof D.CreationDraftsResultSchema>>;
  create: (
    input: D.CreatePipelineDraftInput,
  ) => Promise<Static<typeof D.CreationDraftResultSchema>>;
  update: (
    input: D.UpdatePipelineDraftInput,
  ) => Promise<Static<typeof D.CreationDraftResultSchema>>;
  discard: (
    input: D.DiscardPipelineDraftInput,
  ) => Promise<Static<typeof D.CreationDraftResultSchema>>;
  state: (input: D.PipelineDraftInput) => Promise<Static<typeof C.CreationStateResultSchema>>;
  select: (input: C.CreationSelect) => Promise<Static<typeof PipelineReviewSelectedResultSchema>>;
  cancel: (
    input: C.CreationCandidateInput,
  ) => Promise<Static<typeof C.CreationCandidateResultSchema>>;
  engines: () => Promise<Static<typeof C.CreationEnginesResultSchema>>;
  connect: (
    input: Static<typeof C.CreationConnectSchema>,
  ) => Promise<Static<typeof C.CreationEnginesResultSchema>>;
}>;
export const CREATION_CHANNELS = {
  adopt: 'gobble:creation:adopt:v1',
  outcome: 'gobble:creation:outcome:v1',
  list: 'gobble:creation:list:v1',
  create: 'gobble:creation:create:v1',
  update: 'gobble:creation:update:v1',
  discard: 'gobble:creation:discard:v1',
  state: 'gobble:creation:state:v1',
  select: 'gobble:creation:select:v1',
  cancel: 'gobble:creation:cancel:v1',
  engines: 'gobble:creation:engines:v1',
  connect: 'gobble:creation:connect:v1',
} as const;
