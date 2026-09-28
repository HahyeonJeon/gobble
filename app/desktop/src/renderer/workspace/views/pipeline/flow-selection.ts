import {
  pipelineStepLabel as stepLabel,
  connectionSelector,
  type PipelineFlow,
  type PipelineSelector,
} from '@gobble/contracts';
import type { FlowTarget } from './flow-layout';

/** Layout addresses are transient; discussion addresses include exact directional ports. */
export function flowSelector(flow: PipelineFlow, target: FlowTarget): PipelineSelector {
  switch (target.kind) {
    case 'step':
      return { kind: 'step', stepId: target.id };
    case 'input':
      return { kind: 'input', portName: target.id };
    case 'connection': {
      const edge = flow.connections.find((e) => e.id === target.id);
      if (!edge) throw new Error('Connection unavailable.');
      return connectionSelector(edge);
    }
    case 'port':
      return {
        kind: 'port',
        stepId: target.id,
        direction: target.direction,
        portName: target.name,
      };
    case 'setting':
      return { kind: 'setting', stepId: target.id, key: target.name };
  }
}
export function flowTarget(subject: PipelineSelector): FlowTarget {
  switch (subject.kind) {
    case 'step':
      return { kind: 'step', id: subject.stepId };
    case 'input':
      return { kind: 'input', id: subject.portName };
    case 'connection':
      return { kind: 'connection', id: subject.connectionId };
    case 'port':
      return {
        kind: 'port',
        id: subject.stepId,
        direction: subject.direction,
        name: subject.portName,
      };
    case 'setting':
      return { kind: 'setting', id: subject.stepId, name: subject.key };
  }
}
export function isFlowParent(target: FlowTarget, kind: FlowTarget['kind'], id: string): boolean {
  return (
    target.id === id &&
    (target.kind === kind ||
      (kind === 'step' && (target.kind === 'port' || target.kind === 'setting')))
  );
}

export function connectionLabel(flow: PipelineFlow, id: string) {
  const edge = flow.connections.find((value) => value.id === id)!;
  const label = (id: string) => {
    const step = flow.steps.find((value) => value.id === id);
    return step ? stepLabel(step) : 'Pipeline input';
  };
  return `${label(edge.fromTask)} · ${edge.fromPort} → ${label(edge.toTask)} · ${edge.toPort}`;
}
