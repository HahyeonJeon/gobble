import { Type, type Static } from '@sinclair/typebox';
import { closed } from './identity';
import { EvidenceRefSchema } from './reference-target';
import { SharedReferenceV3Schema } from './shared-reference-v3';

export const SHARED_TOOLSET = 'shared-views-v15';
export const AgentAccessSchema = Type.Union([
  Type.Literal('messages'),
  Type.Literal('sharedViews'),
]);
export { SharedReferenceIdSchema } from './shared-reference-v3';
export const SharedReferenceSchema = Type.Object(
  { ...SharedReferenceV3Schema.properties, evidence: EvidenceRefSchema },
  closed,
);
export type SharedReference = Static<typeof SharedReferenceSchema>;
export type AgentAccess = Static<typeof AgentAccessSchema>;
/** Revoking access does not rewrite a provider thread's registered tool definitions. */
export function needsToolsetRenewal(agent: {
  access?: AgentAccess;
  provider: { threadId: string | null; toolsetVersion?: string | null };
}): boolean {
  return (
    !!agent.provider.threadId &&
    (agent.access === 'sharedViews' || !!agent.provider.toolsetVersion) &&
    agent.provider.toolsetVersion !== SHARED_TOOLSET
  );
}
export const referenceAuthor = (reference: SharedReference): string =>
  reference.author.kind === 'user' ? 'You' : reference.author.name;
