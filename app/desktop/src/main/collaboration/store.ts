import type { AgentAttachment, CollaborationHistory, WorkspaceDocument } from '@gobble/contracts';
import type { ReplyCommit } from '../questions/ports';
import type { DraftConsumption } from '../evidence/ports';

// Collaboration can change only this slice of the Project aggregate.
export type ProjectCollaboration = {
  agents: AgentAttachment[];
  history: CollaborationHistory;
};
export interface CollaborationStore {
  readCollaboration(projectId: string): Promise<ProjectCollaboration>;
  changeCollaboration(
    projectId: string,
    change: (current: ProjectCollaboration) => ProjectCollaboration,
    consumedDraft?: string | DraftConsumption,
    reply?: ReplyCommit,
  ): Promise<void>;
}
export const collaborationOf = (doc: WorkspaceDocument): ProjectCollaboration =>
  structuredClone({
    agents: doc.workspace.agents,
    history: doc.collaboration ?? { submissions: [] },
  });
