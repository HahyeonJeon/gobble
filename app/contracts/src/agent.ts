import { Type, type Static } from '@sinclair/typebox';
import { AgentIdSchema, closed, LabelSchema, ProjectIdSchema } from './identity';
import { AgentAccessSchema } from './shared-context';
import { AgentConfigurationSchema, ProviderIdSchema } from './collaboration';

export const AgentAttachmentSchema = Type.Object(
  {
    projectId: ProjectIdSchema,
    agentId: AgentIdSchema,
    name: LabelSchema,
    instructionProfile: LabelSchema,
    configuration: Type.Optional(AgentConfigurationSchema),
    access: Type.Optional(AgentAccessSchema),
    provider: Type.Object(
      {
        kind: Type.Literal('codex'),
        toolsetVersion: Type.Optional(
          Type.Union([Type.String({ minLength: 1, maxLength: 64 }), Type.Null()]),
        ),
        threadId: Type.Union([Type.Null(), Type.String({ minLength: 1, maxLength: 256 })]),
        accountSessionId: Type.Optional(Type.Union([ProviderIdSchema, Type.Null()])),
      },
      closed,
    ),
  },
  closed,
);

export type AgentAttachment = Static<typeof AgentAttachmentSchema>;
