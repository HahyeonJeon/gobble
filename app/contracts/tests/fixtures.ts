import type { AddressedContext, EvidenceRefV3, Workspace } from '../src';

export function workspaceFixture(): Workspace {
  return {
    schemaVersion: 1,
    projectId: 'prj_atlas',
    revision: 4,
    agents: ['qc', 'methods'].map((id) => ({
      projectId: 'prj_atlas',
      agentId: `agt_${id}`,
      name: id,
      instructionProfile: 'reader',
      provider: { kind: 'codex', threadId: null },
    })),
    surfaces: [
      {
        projectId: 'prj_atlas',
        surfaceId: 'srf_table',
        resource: { kind: 'file', resourceId: 'res_samples' },
        view: 'table',
        pinned: true,
        openedBy: { kind: 'user' },
      },
      {
        projectId: 'prj_atlas',
        surfaceId: 'srf_plot',
        resource: { kind: 'file', resourceId: 'res_plot' },
        view: 'image',
        pinned: false,
        openedBy: { kind: 'agent', agentId: 'agt_qc' },
      },
    ],
    layout: {
      kind: 'split',
      primary: { tabs: ['srf_table'], activeSurfaceId: 'srf_table' },
      secondary: { tabs: ['srf_plot'], activeSurfaceId: 'srf_plot' },
      primaryFraction: 0.5,
    },
    decisions: [
      {
        projectId: 'prj_atlas',
        decisionId: 'dec_review',
        requestedBy: 'agt_qc',
        question: 'Which sample should we inspect?',
        evidence: contextFixture().evidence,
        state: { kind: 'pending' },
      },
    ],
  };
}

export function contextFixture(): Omit<AddressedContext, 'evidence'> & {
  evidence: EvidenceRefV3[];
} {
  return {
    schemaVersion: 2,
    projectId: 'prj_atlas',
    recipientAgentId: 'agt_methods',
    evidence: [
      {
        projectId: 'prj_atlas',
        schemaVersion: 2 as const,
        origin: { surfaceId: 'srf_table' },
        resource: { kind: 'file', resourceId: 'res_samples' },
        dataRevision: 'content-v4',
        selection: {
          coordinateSpace: 'revision-row-column-keys',
          kind: 'table',
          rowKeys: ['S03'],
          columns: ['sample', 'condition'],
        },
      },
    ],
  };
}
