import { Type, type Static } from '@sinclair/typebox';
import {
  closed,
  ProjectIdSchema,
  PipelineIdSchema,
  SurfaceIdSchema,
  TimestampSchema,
} from './identity';
import {
  PipelineStepSchema,
  PipelinePortSchema,
  PipelineConnectionSchema,
  PipelineSettingSchema,
  type PipelineFlow,
  type PipelineInspection,
} from './pipeline-inspection';
import { ContractValidationError } from './validation-error';

const id = Type.String({ minLength: 1, maxLength: 256 });
const digest = Type.String({ pattern: '^sha256:[a-f0-9]{64}$' });
const endpoint = Type.Object({ stepId: Type.String({ maxLength: 256 }), portName: id }, closed);
export const PipelineSubjectSelectorSchema = Type.Union([
  Type.Object({ kind: Type.Literal('step'), stepId: id }, closed),
  Type.Object({ kind: Type.Literal('input'), portName: id }, closed),
  Type.Object(
    {
      kind: Type.Literal('port'),
      stepId: id,
      direction: Type.Union([Type.Literal('input'), Type.Literal('output')]),
      portName: id,
    },
    closed,
  ),
  Type.Object(
    { kind: Type.Literal('connection'), connectionId: id, from: endpoint, to: endpoint },
    closed,
  ),
  Type.Object(
    {
      kind: Type.Literal('setting'),
      stepId: id,
      key: Type.String({ minLength: 1, maxLength: 128 }),
    },
    closed,
  ),
]);
export const PipelineSelectionSchema = Type.Object(
  {
    kind: Type.Literal('pipeline'),
    coordinateSpace: Type.Literal('checked-pipeline-artifact'),
    subject: PipelineSubjectSelectorSchema,
  },
  closed,
);
export const PipelineTargetSchema = Type.Object(
  {
    schemaVersion: Type.Literal(7),
    projectId: ProjectIdSchema,
    resource: Type.Object({ kind: Type.Literal('pipeline'), pipelineId: PipelineIdSchema }, closed),
    dataRevision: digest,
    sourceRevision: digest,
    origin: Type.Optional(Type.Object({ surfaceId: SurfaceIdSchema }, closed)),
    selection: PipelineSelectionSchema,
  },
  closed,
);
export const PipelineSubjectSchema = Type.Union([
  Type.Object({ kind: Type.Literal('step'), step: PipelineStepSchema }, closed),
  Type.Object({ kind: Type.Literal('input'), port: PipelinePortSchema }, closed),
  Type.Object(
    {
      kind: Type.Literal('port'),
      stepId: id,
      stepLabel: Type.String({ maxLength: 4096 }),
      direction: Type.Union([Type.Literal('input'), Type.Literal('output')]),
      port: PipelinePortSchema,
    },
    closed,
  ),
  Type.Object(
    {
      kind: Type.Literal('connection'),
      connection: PipelineConnectionSchema,
      fromLabel: Type.String({ maxLength: 4096 }),
      toLabel: Type.String({ maxLength: 4096 }),
    },
    closed,
  ),
  Type.Object(
    {
      kind: Type.Literal('setting'),
      stepId: id,
      stepLabel: Type.String({ maxLength: 4096 }),
      setting: PipelineSettingSchema,
    },
    closed,
  ),
]);
export const PipelineCaptureSchema = Type.Object(
  {
    schemaVersion: Type.Literal(1),
    capturedAt: TimestampSchema,
    target: PipelineTargetSchema,
    pipelineName: id,
    checkedAt: Type.String({ minLength: 1, maxLength: 100 }),
    subject: PipelineSubjectSchema,
  },
  closed,
);
export type PipelineSelector = Static<typeof PipelineSubjectSelectorSchema>;
export type PipelineSelection = Static<typeof PipelineSelectionSchema>;
export type PipelineTarget = Static<typeof PipelineTargetSchema>;
export type PipelineSubject = Static<typeof PipelineSubjectSchema>;
export type PipelineCapture = Static<typeof PipelineCaptureSchema>;

export const pipelineStepLabel = (step: PipelineFlow['steps'][number]) =>
  step.display.stage || step.name;
export function pipelineSubject(flow: PipelineFlow, selector: PipelineSelector): PipelineSubject {
  const step = 'stepId' in selector ? flow.steps.find((s) => s.id === selector.stepId) : undefined;
  if (selector.kind === 'step' && step) return { kind: 'step', step };
  if (selector.kind === 'input') {
    const port = flow.inputs.find((p) => p.name === selector.portName);
    if (port) return { kind: 'input', port };
  }
  if (selector.kind === 'port' && step) {
    const port = (selector.direction === 'input' ? step.inputs : step.outputs).find(
      (p) => p.name === selector.portName,
    );
    if (port)
      return {
        kind: 'port',
        stepId: step.id,
        stepLabel: pipelineStepLabel(step),
        direction: selector.direction,
        port,
      };
  }
  if (selector.kind === 'setting' && step) {
    const setting = step.settings?.find((s) => s.key === selector.key);
    if (setting)
      return { kind: 'setting', stepId: step.id, stepLabel: pipelineStepLabel(step), setting };
  }
  if (selector.kind === 'connection') {
    const edge = flow.connections.find((e) => e.id === selector.connectionId);
    if (
      edge &&
      edge.fromTask === selector.from.stepId &&
      edge.fromPort === selector.from.portName &&
      edge.toTask === selector.to.stepId &&
      edge.toPort === selector.to.portName
    ) {
      const label = (id: string) => {
        const s = flow.steps.find((s) => s.id === id);
        return s ? pipelineStepLabel(s) : 'Pipeline input';
      };
      return {
        kind: 'connection',
        connection: edge,
        fromLabel: label(edge.fromTask),
        toLabel: label(edge.toTask),
      };
    }
  }
  throw new ContractValidationError(
    'This subject is not declared in the checked pipeline version.',
  );
}
export function pipelineSubjectLabel(subject: PipelineSubject): string {
  switch (subject.kind) {
    case 'step':
      return pipelineStepLabel(subject.step);
    case 'input':
      return 'Pipeline input · ' + subject.port.name;
    case 'port':
      return subject.stepLabel + ' · ' + subject.direction + ' · ' + subject.port.name;
    case 'setting':
      return subject.stepLabel + ' · ' + subject.setting.label;
    case 'connection':
      return (
        subject.fromLabel +
        ' · ' +
        subject.connection.fromPort +
        ' → ' +
        subject.toLabel +
        ' · ' +
        subject.connection.toPort
      );
  }
}
export function pipelineTarget(
  value: PipelineInspection,
  subject: PipelineSelector,
  surfaceId?: string,
): PipelineTarget {
  if (!value.artifact)
    throw new ContractValidationError('Check the pipeline before discussing its flow.');
  pipelineSubject(value.artifact.flow, subject);
  return {
    schemaVersion: 7,
    projectId: value.projectId,
    resource: { kind: 'pipeline', pipelineId: value.pipelineId },
    dataRevision: value.artifact.artifactId,
    sourceRevision: value.artifact.sourceRevision,
    ...(surfaceId ? { origin: { surfaceId } } : {}),
    selection: { kind: 'pipeline', coordinateSpace: 'checked-pipeline-artifact', subject },
  };
}
export function samePipelineSelector(a: PipelineSelector, b: PipelineSelector): boolean {
  if (a.kind !== b.kind) return false;
  switch (a.kind) {
    case 'step':
      return b.kind === 'step' && a.stepId === b.stepId;
    case 'input':
      return b.kind === 'input' && a.portName === b.portName;
    case 'port':
      return (
        b.kind === 'port' &&
        a.stepId === b.stepId &&
        a.direction === b.direction &&
        a.portName === b.portName
      );
    case 'setting':
      return b.kind === 'setting' && a.stepId === b.stepId && a.key === b.key;
    case 'connection':
      return (
        b.kind === 'connection' &&
        a.connectionId === b.connectionId &&
        a.from.stepId === b.from.stepId &&
        a.from.portName === b.from.portName &&
        a.to.stepId === b.to.stepId &&
        a.to.portName === b.to.portName
      );
  }
}
export function selectorForSubject(subject: PipelineSubject): PipelineSelector {
  switch (subject.kind) {
    case 'step':
      return { kind: 'step', stepId: subject.step.id };
    case 'input':
      return { kind: 'input', portName: subject.port.name };
    case 'port':
      return {
        kind: 'port',
        stepId: subject.stepId,
        direction: subject.direction,
        portName: subject.port.name,
      };
    case 'setting':
      return { kind: 'setting', stepId: subject.stepId, key: subject.setting.key };
    case 'connection':
      return connectionSelector(subject.connection);
  }
}
export function connectionSelector(edge: PipelineFlow['connections'][number]): PipelineSelector {
  return {
    kind: 'connection',
    connectionId: edge.id,
    from: { stepId: edge.fromTask, portName: edge.fromPort },
    to: { stepId: edge.toTask, portName: edge.toPort },
  };
}
export function validatePipelineCapture(capture: PipelineCapture): void {
  if (!samePipelineSelector(capture.target.selection.subject, selectorForSubject(capture.subject)))
    throw new ContractValidationError('Captured pipeline facts do not match their exact target.');
}
