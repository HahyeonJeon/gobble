import { WorkspaceDocumentV20Schema } from './workspace-document-v20';
import { WorkspaceDocumentV19Schema } from './workspace-document-v19';
import { WorkspaceDocumentV18Schema } from './workspace-document-v18';
import { WorkspaceDocumentV17Schema } from './workspace-document-v17';
import { WorkspaceDocumentV16Schema } from './workspace-document-v16';
import { WorkspaceDocumentV15Schema } from './workspace-document-v15';
import { WorkspaceDocumentV14Schema } from './workspace-document-v14';
import { WorkspaceDocumentV13Schema } from './workspace-document-v13';
import { WorkspaceDocumentV12Schema } from './workspace-document-v12';
import { WorkspaceDocumentV11Schema } from './workspace-document-v11';
import {
  FrozenWorkspaceV9Schema as WorkspaceDocumentV9Schema,
  WorkspaceDocumentV10Schema,
} from './workspace-document-v10';
import { WorkspaceDocumentV8Schema } from './workspace-document-v8';
import { WorkspaceDocumentV7Schema } from './workspace-document-v7';
import { WorkspaceDocumentV3Schema } from './workspace-document-v3';
import { WorkspaceDocumentV4Schema } from './workspace-document-v4';
import { WorkspaceDocumentV5Schema } from './workspace-document-v5';
import { WorkspaceDocumentV6Schema } from './workspace-document-v6';
import { parse } from './validation';
import { WorkspaceDocumentV1Schema } from './workspace-document-v1';
import { WorkspaceDocumentV2Schema } from './workspace-document-v2';
import type { EvidenceRefV1 } from './reference-v1';
import type { EvidenceRef } from './reference-target';
import type { SelectionV2 as Selection } from './selection-v2';
import { parseWorkspaceDocument, type WorkspaceDocument } from './workspace-document';

function migrateReference({ surfaceId, selection, ...target }: EvidenceRefV1): EvidenceRef {
  let selector: Selection | undefined;
  if (selection?.kind === 'text') selector = { ...selection, coordinateSpace: 'utf16-line-column' };
  else if (selection?.kind === 'table')
    selector = { ...selection, coordinateSpace: 'revision-row-column-keys' };
  else if (selection?.kind === 'image')
    selector = { ...selection, coordinateSpace: 'normalized-original-image' };
  return {
    ...target,
    schemaVersion: 2,
    origin: { surfaceId },
    ...(selector ? { selection: selector } : {}),
  };
}

/** Storage-only compatibility boundary; IPC accepts current documents only. Never mutates input. */
export function readStoredWorkspace(input: unknown): WorkspaceDocument {
  if (
    input !== null &&
    typeof input === 'object' &&
    'schemaVersion' in input &&
    input.schemaVersion === 1
  ) {
    const { discussion, ...legacy } = parse(WorkspaceDocumentV1Schema, input);

    return readStoredWorkspace({
      ...legacy,
      workspace: {
        ...legacy.workspace,
        layout:
          legacy.workspace.layout.kind === 'split'
            ? { ...legacy.workspace.layout, primaryFraction: 0.5 }
            : legacy.workspace.layout,
      },
      schemaVersion: 2,
      paneOrientation: 'vertical',
      chat: {
        collapsed: discussion.collapsed,
        width: 420,
        draft: discussion.draft,
        recipientAgentId: discussion.recipientAgentId,
      },
    });
  }
  if (
    input !== null &&
    typeof input === 'object' &&
    'schemaVersion' in input &&
    input.schemaVersion === 2
  ) {
    const old = parse(WorkspaceDocumentV2Schema, input);
    const manifest = <T extends { evidence: EvidenceRefV1 }>(item: T) => ({
      ...item,
      evidence: migrateReference(item.evidence),
    });
    return readStoredWorkspace({
      ...old,
      schemaVersion: 3,
      selections: old.selections.map((evidence) => ({
        surfaceId: evidence.surfaceId,
        evidence: migrateReference(evidence),
      })),
      workspace: {
        ...old.workspace,
        decisions: old.workspace.decisions.map((decision) => ({
          ...decision,
          evidence: decision.evidence.map((item) =>
            'evidence' in item ? manifest(item) : migrateReference(item),
          ),
        })),
      },
      ...(old.sharedReferences ? { sharedReferences: old.sharedReferences.map(manifest) } : {}),
      chat: {
        ...old.chat,
        ...(old.chat.attachments ? { attachments: old.chat.attachments.map(manifest) } : {}),
      },
      ...(old.collaboration
        ? {
            collaboration: {
              submissions: old.collaboration.submissions.map((submission) => ({
                ...submission,
                ...(submission.evidence ? { evidence: submission.evidence.map(manifest) } : {}),
              })),
            },
          }
        : {}),
    });
  }
  if (
    input !== null &&
    typeof input === 'object' &&
    'schemaVersion' in input &&
    input.schemaVersion === 3
  ) {
    const old = parse(WorkspaceDocumentV3Schema, input);
    return readStoredWorkspace({ ...old, schemaVersion: 4, viewLinks: [] });
  }
  if (
    input !== null &&
    typeof input === 'object' &&
    'schemaVersion' in input &&
    input.schemaVersion === 4
  ) {
    const old = parse(WorkspaceDocumentV4Schema, input);
    return readStoredWorkspace({ ...old, schemaVersion: 5 });
  }
  if (
    input !== null &&
    typeof input === 'object' &&
    'schemaVersion' in input &&
    input.schemaVersion === 5
  ) {
    const old = parse(WorkspaceDocumentV5Schema, input);
    return readStoredWorkspace({ ...old, schemaVersion: 6 });
  }
  if (
    input !== null &&
    typeof input === 'object' &&
    'schemaVersion' in input &&
    input.schemaVersion === 6
  ) {
    const old = parse(WorkspaceDocumentV6Schema, input);
    return readStoredWorkspace({ ...old, schemaVersion: 7 });
  }
  if (
    input !== null &&
    typeof input === 'object' &&
    'schemaVersion' in input &&
    input.schemaVersion === 7
  ) {
    const old = parse(WorkspaceDocumentV7Schema, input);
    return readStoredWorkspace({ ...old, schemaVersion: 8 });
  }
  if (
    input !== null &&
    typeof input === 'object' &&
    'schemaVersion' in input &&
    input.schemaVersion === 8
  ) {
    const old = parse(WorkspaceDocumentV8Schema, input);
    return readStoredWorkspace({ ...old, schemaVersion: 9 });
  }
  if (
    input !== null &&
    typeof input === 'object' &&
    'schemaVersion' in input &&
    input.schemaVersion === 9
  ) {
    const old = parse(WorkspaceDocumentV9Schema, input);
    return readStoredWorkspace({ ...old, schemaVersion: 10 });
  }
  if (
    input !== null &&
    typeof input === 'object' &&
    'schemaVersion' in input &&
    input.schemaVersion === 10
  ) {
    const old = parse(WorkspaceDocumentV10Schema, input);
    return readStoredWorkspace({ ...old, schemaVersion: 11 });
  }
  if (
    input !== null &&
    typeof input === 'object' &&
    'schemaVersion' in input &&
    input.schemaVersion === 11
  ) {
    const old = parse(WorkspaceDocumentV11Schema, input);
    return readStoredWorkspace({ ...old, schemaVersion: 12 });
  }
  if (
    input !== null &&
    typeof input === 'object' &&
    'schemaVersion' in input &&
    input.schemaVersion === 12
  ) {
    const old = parse(WorkspaceDocumentV12Schema, input);
    return readStoredWorkspace({ ...old, schemaVersion: 13 });
  }
  if (
    input !== null &&
    typeof input === 'object' &&
    'schemaVersion' in input &&
    input.schemaVersion === 13
  ) {
    const old = parse(WorkspaceDocumentV13Schema, input);
    return readStoredWorkspace({ ...old, schemaVersion: 14 });
  }
  if (
    input !== null &&
    typeof input === 'object' &&
    'schemaVersion' in input &&
    input.schemaVersion === 14
  ) {
    const old = parse(WorkspaceDocumentV14Schema, input);
    return readStoredWorkspace({ ...old, schemaVersion: 15 });
  }
  if (
    input &&
    typeof input === 'object' &&
    'schemaVersion' in input &&
    input.schemaVersion === 15
  ) {
    const old = parse(WorkspaceDocumentV15Schema, input);
    return readStoredWorkspace({ ...old, schemaVersion: 16 });
  }
  if (
    input &&
    typeof input === 'object' &&
    'schemaVersion' in input &&
    input.schemaVersion === 16
  ) {
    const old = parse(WorkspaceDocumentV16Schema, input);
    return readStoredWorkspace({ ...old, schemaVersion: 17 });
  }
  if (
    input &&
    typeof input === 'object' &&
    'schemaVersion' in input &&
    input.schemaVersion === 17
  ) {
    const old = parse(WorkspaceDocumentV17Schema, input);
    return readStoredWorkspace({ ...old, schemaVersion: 18 });
  }
  if (
    input &&
    typeof input === 'object' &&
    'schemaVersion' in input &&
    input.schemaVersion === 18
  ) {
    const old = parse(WorkspaceDocumentV18Schema, input);
    return readStoredWorkspace({ ...old, schemaVersion: 19 });
  }
  if (
    typeof input === 'object' &&
    input !== null &&
    'schemaVersion' in input &&
    input.schemaVersion === 19
  ) {
    const old = parse(WorkspaceDocumentV19Schema, input);
    return readStoredWorkspace({ ...old, schemaVersion: 20 });
  }
  if (
    input &&
    typeof input === 'object' &&
    'schemaVersion' in input &&
    input.schemaVersion === 20
  ) {
    const old = parse(WorkspaceDocumentV20Schema, input);
    return parseWorkspaceDocument({ ...old, schemaVersion: 21 });
  }
  return parseWorkspaceDocument(input);
}
