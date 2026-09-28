import { ReportTargetSchema } from './run-report';
import { PipelineTargetSchema } from './pipeline-reference';
import { NotebookTargetSchema, validateNotebookSelection } from './notebook';
import { PdfTargetSchema, validatePdfSelection } from './pdf';
import { DependencyTargetSchema } from './run-dependencies';
import { type Static, type TSchema } from '@sinclair/typebox';
import { Value } from '@sinclair/typebox/value';
import type { EvidenceRef, Selection } from './context';
import { AddressedContextSchema } from './context';
import { isQuestion } from './question';
import { validateViewport } from './tabular-projection';
import { WorkspaceSchema } from './workspace';

import { ContractValidationError } from './validation-error';
export { ContractValidationError } from './validation-error';

// Interpret schemas without generated code: this also works under renderer CSP.
export function parse<T extends TSchema>(schema: T, input: unknown): Static<T> {
  if (!Value.Check(schema, input)) throw new ContractValidationError();
  return input;
}

export function validateSelection(selection: Selection): void {
  if (selection.kind === 'notebook') {
    validateNotebookSelection(selection);
    return;
  }
  if (selection.kind === 'pdf') {
    validatePdfSelection(selection);
    return;
  }
  if (
    selection.kind === 'image' &&
    (selection.x + selection.width > 1 || selection.y + selection.height > 1)
  ) {
    throw new ContractValidationError('Image selection exceeds the image bounds.');
  }
  if (
    (selection.kind === 'text' || selection.kind === 'log-text') &&
    (selection.start.line > selection.end.line ||
      (selection.start.line === selection.end.line &&
        selection.start.column >= selection.end.column))
  ) {
    throw new ContractValidationError('Text selection must be nonempty and ordered.');
  }
}

export function validateEvidence(evidence: EvidenceRef, projectId: string): void {
  if (evidence.projectId !== projectId)
    throw new ContractValidationError('Evidence belongs to another Project.');
  if (evidence.schemaVersion === 8) {
    parse(ReportTargetSchema, evidence);
    if (
      evidence.projectId !== evidence.resource.saved.projectId ||
      evidence.dataRevision !== evidence.resource.saved.asset.hash
    )
      throw new ContractValidationError('Report identity does not match its saved content.');
    return;
  }
  if (evidence.schemaVersion === 7) {
    parse(PipelineTargetSchema, evidence);
    return;
  }
  if (evidence.schemaVersion === 6) {
    parse(NotebookTargetSchema, evidence);
    validateNotebookSelection(evidence.selection);
    return;
  }
  if (evidence.schemaVersion === 5) {
    parse(PdfTargetSchema, evidence);
    validatePdfSelection(evidence.selection);
    return;
  }
  if (evidence.schemaVersion === 4) {
    parse(DependencyTargetSchema, evidence);
    return;
  }
  const kind = evidence.selection?.kind;
  if (evidence.schemaVersion === 3 && evidence.resource.kind === 'file')
    throw new ContractValidationError('Observed targets require a Run or log resource.');
  if (
    kind &&
    ((evidence.resource.kind === 'log' && kind !== 'text' && kind !== 'log-text') ||
      (evidence.resource.kind === 'run' && kind !== 'run-task') ||
      (evidence.resource.kind === 'file' && (kind === 'run-task' || kind === 'log-text')))
  ) {
    throw new ContractValidationError('Selection is incompatible with this resource.');
  }
  if (evidence.selection) validateSelection(evidence.selection);
}

export function parseAddressedContext(input: unknown) {
  const context = parse(AddressedContextSchema, input);
  context.evidence.forEach((evidence) => validateEvidence(evidence, context.projectId));
  return context;
}

export function parseWorkspace(input: unknown) {
  const workspace = parse(WorkspaceSchema, input);
  const agents = new Set(workspace.agents.map((agent) => agent.agentId));
  const surfaces = new Set(workspace.surfaces.map((surface) => surface.surfaceId));
  const decisions = new Set(workspace.decisions.map((decision) => decision.decisionId));
  if (
    agents.size !== workspace.agents.length ||
    surfaces.size !== workspace.surfaces.length ||
    decisions.size !== workspace.decisions.length
  ) {
    throw new ContractValidationError('Workspace identifiers must be unique.');
  }
  for (const record of [...workspace.agents, ...workspace.surfaces, ...workspace.decisions]) {
    if (record.projectId !== workspace.projectId)
      throw new ContractValidationError('Record belongs to another Project.');
  }
  for (const surface of workspace.surfaces) {
    if (surface.openedBy.kind === 'agent' && !agents.has(surface.openedBy.agentId)) {
      throw new ContractValidationError('Opening agent is missing.');
    }
    if (surface.view === 'scatter') validateViewport(surface.scatter.viewport);
    const compatible =
      surface.resource.kind === 'file'
        ? ['text', 'table', 'image', 'scatter', 'pdf', 'notebook'].includes(surface.view)
        : surface.resource.kind === surface.view;
    if (!compatible) throw new ContractValidationError('View is incompatible with this resource.');
  }
  const panes =
    workspace.layout.kind === 'single'
      ? [workspace.layout.primary]
      : [workspace.layout.primary, workspace.layout.secondary];
  const tabs = panes.flatMap((pane) => pane.tabs);
  if (
    tabs.length !== surfaces.size ||
    new Set(tabs).size !== tabs.length ||
    tabs.some((tab) => !surfaces.has(tab))
  ) {
    throw new ContractValidationError('Every open surface must appear in exactly one pane.');
  }
  for (const pane of panes) {
    if (
      pane.tabs.length === 0
        ? pane.activeSurfaceId !== null
        : !pane.tabs.some((tab) => tab === pane.activeSurfaceId)
    ) {
      throw new ContractValidationError('Active surface must belong to its pane.');
    }
  }
  for (const decision of workspace.decisions) {
    if (!agents.has(decision.requestedBy))
      throw new ContractValidationError('Requesting agent is missing.');
    // Evidence may outlive a closed surface; service checks current resource ownership/revision on use.
    const evidence = isQuestion(decision)
      ? decision.evidence.map((item) => item.evidence)
      : decision.evidence;
    evidence.forEach((item) => validateEvidence(item, workspace.projectId));
  }
  return workspace;
}
