import {
  pipelineSubject,
  pipelineTarget,
  connectionSelector,
  type PipelineSelector,
  type PipelineTarget,
  type PipelineFlow,
  type PipelineInspection,
} from '@gobble/contracts';
import { AppProblem } from '../problem';

/** Bounded semantic inventory; only complete returned subjects authorize a mark. */
export function pipelineObservation(
  value: PipelineInspection,
  surfaceId: string,
  selection?: PipelineSelector,
) {
  const artifact = value.artifact;
  if (!artifact)
    throw new AppProblem('unsupported', 'Check this pipeline before observing its steps.');
  const flow: PipelineFlow = artifact.flow;
  const selectors: PipelineSelector[] = selection
    ? [selection]
    : [
        ...flow.inputs.map((p) => ({ kind: 'input' as const, portName: p.name })),
        ...flow.steps.flatMap((s) => [
          { kind: 'step' as const, stepId: s.id },
          ...s.inputs.map((p) => ({
            kind: 'port' as const,
            stepId: s.id,
            direction: 'input' as const,
            portName: p.name,
          })),
          ...s.outputs.map((p) => ({
            kind: 'port' as const,
            stepId: s.id,
            direction: 'output' as const,
            portName: p.name,
          })),
          ...(s.settings ?? []).map((f) => ({
            kind: 'setting' as const,
            stepId: s.id,
            key: f.key,
          })),
        ]),
        ...flow.connections.map(connectionSelector),
      ];
  const targets: PipelineTarget[] = [];
  const content = {
    kind: 'pipeline' as const,
    name: flow.name,
    artifactId: artifact.artifactId,
    sourceRevision: artifact.sourceRevision,
    checkedAt: artifact.checkedAt,
    inspectionState: value.state,
    scope: 'checked-artifact' as const,
    requestedSubjects: selectors.length,
    returnedSubjects: 0,
    truncated: false,
    subjects: [] as Array<{ target: PipelineTarget; facts: ReturnType<typeof pipelineSubject> }>,
  };
  let attempted = 0;
  for (const selector of selectors) {
    if (targets.length >= 128 || attempted++ >= 256) break;
    const target = pipelineTarget(value, selector, surfaceId);
    const entry = { target, facts: pipelineSubject(flow, selector) };
    content.subjects.push(entry);
    if (Buffer.byteLength(JSON.stringify(content)) > 48 * 1024) {
      content.subjects.pop();
      if (selection)
        throw new AppProblem(
          'unsupported',
          'This step exceeds the observation limit. Request a port or setting.',
        );
      continue;
    }
    targets.push(target);
  }
  content.returnedSubjects = targets.length;
  content.truncated = targets.length < selectors.length;
  return { content, targets };
}
