import { CreationContextSchema } from './creation-review';
import { PipelineReviewContextSchema } from './pipeline-proposal';
import { Type, type Static } from '@sinclair/typebox';
import {
  DecisionIdSchema,
  AgentIdSchema,
  ProjectIdSchema,
  RequestIdSchema,
  TimestampSchema,
  closed,
  LabelSchema,
} from './identity';
import { AgentAccessSchema } from './shared-context';
import { EvidenceManifestSchema, PreparedEvidenceIdSchema } from './evidence';
import { serviceResult } from './result';

export const ProviderIdSchema = Type.String({ minLength: 1, maxLength: 256 });
export const AgentConfigurationSchema = Type.Object(
  {
    model: ProviderIdSchema,
    effort: Type.String({ minLength: 1, maxLength: 32 }),
    instructions: Type.String({ maxLength: 4000 }),
  },
  closed,
);
export const SubmissionStateSchema = Type.Union([
  Type.Literal('submitting'),
  Type.Literal('running'),
  Type.Literal('uncertain'),
  Type.Literal('completed'),
  Type.Literal('interrupted'),
  Type.Literal('failed'),
  Type.Literal('abandoned'),
]);
export const SubmissionSchema = Type.Object(
  {
    requestId: RequestIdSchema,
    agentId: AgentIdSchema,
    text: Type.String({ maxLength: 16000 }),
    pipelineReview: Type.Optional(PipelineReviewContextSchema),
    pipelineCreation: Type.Optional(CreationContextSchema),
    allowPipelineCreation: Type.Optional(Type.Boolean()),
    allowPipelineProposal: Type.Optional(Type.Boolean()),
    evidence: Type.Optional(Type.Array(EvidenceManifestSchema, { minItems: 1, maxItems: 16 })),
    preparedEvidenceId: Type.Optional(PreparedEvidenceIdSchema),
    replyToQuestionId: Type.Optional(DecisionIdSchema),
    model: ProviderIdSchema,
    effort: Type.String({ minLength: 1, maxLength: 32 }),
    createdAt: TimestampSchema,
    // Absent in legacy records; null until the first response is durably received.
    responseStartedAt: Type.Optional(Type.Union([TimestampSchema, Type.Null()])),
    state: SubmissionStateSchema,
    threadId: ProviderIdSchema,
    turnId: Type.Union([ProviderIdSchema, Type.Null()]),
    response: Type.String({ maxLength: 32000 }),
    problem: Type.Union([Type.String({ maxLength: 500 }), Type.Null()]),
  },
  closed,
);
export const CollaborationHistorySchema = Type.Object(
  {
    submissions: Type.Array(SubmissionSchema, { maxItems: 40 }),
  },
  closed,
);
export const CodexModelSchema = Type.Object(
  {
    id: ProviderIdSchema,
    name: LabelSchema,
    isDefault: Type.Boolean(),
    inputModalities: Type.Optional(
      Type.Array(Type.Union([Type.Literal('text'), Type.Literal('image')]), {
        maxItems: 2,
        uniqueItems: true,
      }),
    ),
    efforts: Type.Array(Type.String({ minLength: 1, maxLength: 32 }), {
      minItems: 1,
      maxItems: 16,
    }),
    defaultEffort: Type.String({ minLength: 1, maxLength: 32 }),
  },
  closed,
);
export const AccountStatusSchema = Type.Object(
  {
    runtime: Type.Union([
      Type.Literal('disconnected'),
      Type.Literal('connecting'),
      Type.Literal('ready'),
    ]),
    version: Type.Literal('0.153.4'),
    status: Type.Union([
      Type.Literal('signedOut'),
      Type.Literal('signingIn'),
      Type.Literal('signedIn'),
    ]),
    email: Type.Union([Type.String({ maxLength: 320 }), Type.Null()]),
    plan: Type.Union([Type.String({ maxLength: 80 }), Type.Null()]),
    sessionId: Type.Union([ProviderIdSchema, Type.Null()]),
    problem: Type.Union([Type.String({ maxLength: 500 }), Type.Null()]),
  },
  closed,
);
export const CollaborationStatusSchema = Type.Object(
  {
    problem: Type.Union([Type.String({ maxLength: 500 }), Type.Null()]),
    sequence: TimestampSchema,
    account: AccountStatusSchema,
    models: Type.Array(CodexModelSchema, { maxItems: 100 }),
    streams: Type.Array(
      Type.Object(
        {
          projectId: ProjectIdSchema,
          agentId: AgentIdSchema,
          requestId: RequestIdSchema,
          text: Type.String({ maxLength: 32000 }),
        },
        closed,
      ),
      { maxItems: 2 },
    ),
  },
  closed,
);
export const AccountActionSchema = Type.Object(
  {
    action: Type.Union([
      Type.Literal('connect'),
      Type.Literal('signIn'),
      Type.Literal('cancelSignIn'),
      Type.Literal('signOut'),
    ]),
  },
  closed,
);
export const ConfigureAgentSchema = Type.Object(
  {
    projectId: ProjectIdSchema,
    agentId: Type.Union([AgentIdSchema, Type.Null()]),
    requestId: RequestIdSchema,
    name: LabelSchema,
    configuration: AgentConfigurationSchema,
    access: Type.Optional(AgentAccessSchema),
  },
  closed,
);
export const AgentActionSchema = Type.Object(
  {
    projectId: ProjectIdSchema,
    agentId: AgentIdSchema,
    action: Type.Union([
      Type.Literal('interrupt'),
      Type.Literal('reconcile'),
      Type.Literal('newConversation'),
      Type.Literal('enableSharedViews'),
      Type.Literal('disableSharedViews'),
    ]),
  },
  closed,
);
export const SendMessageSchema = Type.Object(
  {
    projectId: ProjectIdSchema,
    agentId: AgentIdSchema,
    requestId: RequestIdSchema,
    text: Type.String({ maxLength: 16000 }),
    pipelineReview: Type.Optional(PipelineReviewContextSchema),
    pipelineCreation: Type.Optional(CreationContextSchema),
    allowPipelineCreation: Type.Optional(Type.Boolean()),
    allowPipelineProposal: Type.Optional(Type.Boolean()),
    preparedEvidenceId: Type.Optional(PreparedEvidenceIdSchema),
    replyToQuestionId: Type.Optional(DecisionIdSchema),
  },
  closed,
);
export const CollaborationStatusResultSchema = serviceResult(CollaborationStatusSchema);
export const CollaborationAcceptedResultSchema = serviceResult(
  Type.Object({ accepted: Type.Literal(true) }, closed),
);
export const COLLABORATION_CHANNELS = {
  status: 'gobble:collaboration:status:v1',
  account: 'gobble:collaboration:account:v1',
  configure: 'gobble:collaboration:configure:v1',
  send: 'gobble:collaboration:send:v1',
  agent: 'gobble:collaboration:agent:v1',
  changed: 'gobble:collaboration:changed:v1',
  document: 'gobble:collaboration:document:v1',
} as const;
export type AgentConfiguration = Static<typeof AgentConfigurationSchema>;
export type Submission = Static<typeof SubmissionSchema>;
export type CollaborationHistory = Static<typeof CollaborationHistorySchema>;
export type CodexModel = Static<typeof CodexModelSchema>;
export type AccountStatus = Static<typeof AccountStatusSchema>;
export type CollaborationStatus = Static<typeof CollaborationStatusSchema>;
export type ConfigureAgent = Static<typeof ConfigureAgentSchema>;
export type SendMessage = Static<typeof SendMessageSchema>;
export type AccountAction = Static<typeof AccountActionSchema>;
export type AgentAction = Static<typeof AgentActionSchema>;
export type CollaborationBridge = Readonly<{
  collaboration: Readonly<{
    status: () => Promise<Static<typeof CollaborationStatusResultSchema>>;
    account: (input: AccountAction) => Promise<Static<typeof CollaborationStatusResultSchema>>;
    configure: (input: ConfigureAgent) => Promise<Static<typeof CollaborationAcceptedResultSchema>>;
    send: (input: SendMessage) => Promise<Static<typeof CollaborationAcceptedResultSchema>>;
    agent: (input: AgentAction) => Promise<Static<typeof CollaborationAcceptedResultSchema>>;
    onChanged: (listener: (value: CollaborationStatus) => void) => () => void;
  }>;
}>;
