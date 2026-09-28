import { pipelineObservation } from './pipeline-observation';
import { dependencyObservation } from './dependency-observation';
import { runObservation } from './run-observation';
import { isDeepStrictEqual } from 'node:util';
import { randomUUID } from 'node:crypto';
import {
  activeLogStream,
  visibleRunTasks,
  containsObservedTarget,
  type EvidenceRef,
  type Selection,
  type Surface,
  type SurfaceData,
  type RenderAcknowledgment,
} from '@gobble/contracts';
import { AppProblem } from '../problem';
import { checkSelection, observedDataRevision } from '../workspace/selection';
import { presentLog } from '../service/log-presentation';

type View = { surface: Surface; data: SurfaceData; acknowledgment: RenderAcknowledgment };
type Scope = {
  acknowledgment: RenderAcknowledgment;
  targets: EvidenceRef[];
  assertCurrent?: () => void;
};
const position = (text: string, offset: number) => {
  const lines = text.slice(0, offset).split('\n');
  return { line: lines.length, column: lines.at(-1)?.length ?? 0 };
};
const offset = (text: string, point: { line: number; column: number }) =>
  text
    .split('\n')
    .slice(0, point.line - 1)
    .reduce((n, line) => n + line.length + 1, 0) + point.column;

/** One turn's returned semantic scopes. It holds addresses, not another observation cache. */
export class ObservedReads {
  private readonly scopes = new Map<string, Scope>();
  observe(
    view: View,
    selection: Selection | undefined,
    scope: 'view' | 'source-preview',
    requestedStream?: 'stdout' | 'stderr',
  ) {
    if (view.data.kind === 'report' || view.surface.resource.kind === 'report')
      throw new AppProblem('unsupported', 'Report observation is not available yet.');
    if (view.data.kind === 'file') return undefined;
    if (
      view.data.kind === 'pipeline' ||
      view.surface.resource.kind === 'pipeline' ||
      view.surface.resource.kind === 'creation-draft'
    )
      throw new AppProblem('unsupported', 'Pipeline discussion references are not available yet.');
    if (
      scope === 'view' &&
      view.surface.view === 'run' &&
      view.surface.runView?.mode === 'dependencies'
    )
      throw new AppProblem(
        'unsupported',
        'This representation is Dependencies. Request dependency targets, or use source-preview explicitly for task facts.',
      );
    if (this.scopes.size >= 32)
      throw new AppProblem('unsupported', 'This turn reached its observation limit.');
    if (selection && selection.kind !== 'run-task' && selection.kind !== 'log-text')
      throw new AppProblem(
        'invalid_request',
        'Run/log observations require typed task or stream selectors.',
      );
    const evidence: Extract<EvidenceRef, { schemaVersion: 3 }> = {
      schemaVersion: 3,
      projectId: view.surface.projectId,
      resource: view.surface.resource,
      origin: { surfaceId: view.surface.surfaceId },
      dataRevision: observedDataRevision(view.data),
      ...(selection ? { selection } : {}),
    };
    checkSelection(evidence, view.data);
    let content: unknown;
    let targets: EvidenceRef[] = [];
    if (view.data.kind === 'run') {
      if (requestedStream)
        throw new AppProblem('invalid_request', 'Stream choice requires a log view.');
      const value = view.data.value;
      const visible = visibleRunTasks(
        value,
        view.surface.view === 'run' ? view.surface.runView : undefined,
      );
      let tasks = scope === 'view' ? visible : value.tasks;
      if (selection?.kind === 'run-task') {
        tasks = tasks.filter(
          (task) => task.instanceId === selection.instanceId && task.attempt === selection.attempt,
        );
        if (!tasks.length)
          throw new AppProblem(
            'invalid_request',
            'This task is outside the displayed filter. Use source-preview explicitly.',
          );
      }
      const bounded = runObservation(value, tasks);
      const returned = bounded.tasks;
      if (selection && !returned.length)
        throw new AppProblem('unsupported', 'This task exceeds the observation size limit.');
      targets = returned.map((task) => ({
        ...evidence,
        selection: {
          kind: 'run-task',
          coordinateSpace: 'observed-instance-attempt',
          instanceId: task.instanceId,
          attempt: task.attempt,
        },
      }));
      content = { ...bounded, scope, displayedTasks: visible.length };
    } else {
      const value = 'streams' in view.data.value ? view.data.value : presentLog(view.data.value);
      const displayed = activeLogStream(
        view.surface.view === 'log' ? view.surface.logView?.stream : undefined,
        value,
      );
      const stream =
        requestedStream ??
        (selection?.kind === 'log-text'
          ? selection.stream
          : displayed === 'legacy'
            ? undefined
            : displayed);
      if (!stream)
        throw new AppProblem(
          'invalid_request',
          'Choose stdout or stderr explicitly for a legacy log view.',
        );
      if (scope === 'view' && stream !== displayed)
        throw new AppProblem(
          'invalid_request',
          'This stream is not displayed. Use source-preview explicitly.',
        );
      if (selection && (selection.kind !== 'log-text' || selection.stream !== stream))
        throw new AppProblem('invalid_request', 'The selection must address the requested stream.');
      const source = value.streams[stream];
      const range =
        selection?.kind === 'log-text'
          ? selection
          : source.text
            ? {
                kind: 'log-text' as const,
                coordinateSpace: 'decoded-preview-utf16-line-column' as const,
                stream,
                start: { line: 1, column: 0 },
                end: position(source.text, source.text.length),
              }
            : undefined;
      if (range) {
        evidence.selection = range;
        targets = [evidence];
      }
      content = {
        kind: 'log',
        runRef: value.runRef,
        instanceId: value.instance,
        attempt: value.attempt,
        stream,
        displayedStream: displayed,
        scope,
        engineRevision: value.engineRevision,
        observedAt: value.observedAt,
        tailLimitBytes: value.tailLimitBytes,
        error: value.error,
        ...source,
        text: range
          ? source.text.slice(offset(source.text, range.start), offset(source.text, range.end))
          : '',
        selection: range ?? null,
      };
    }
    if (Buffer.byteLength(JSON.stringify({ evidence, content })) > 64 * 1024 - 2048)
      throw new AppProblem(
        'unsupported',
        'This observation exceeds the text size limit. Request one task or smaller range.',
      );
    const id = 'obs_' + randomUUID();
    this.scopes.set(
      id,
      structuredClone({
        acknowledgment: view.acknowledgment,
        targets: scope === 'view' ? targets : [],
      }),
    );
    return {
      receipt: {
        evidence,
        observedReadId: id,
        observation: view.acknowledgment,
        source: 'loaded-preview',
        scope,
        observedAt: view.data.value.observedAt,
        pointable: scope === 'view' && targets.length > 0,
      },
      content,
    };
  }
  observePipeline(view: View, selection: Selection | undefined, scope: 'view' | 'source-preview') {
    if (view.data.kind !== 'pipeline' || view.surface.view !== 'pipeline')
      throw new AppProblem('invalid_request', 'A pipeline view is required.');
    if (selection && selection.kind !== 'pipeline')
      throw new AppProblem('invalid_request', 'Pipeline observations require pipeline selectors.');
    if (this.scopes.size >= 32)
      throw new AppProblem('unsupported', 'This turn reached its observation limit.');
    const result = pipelineObservation(view.data.value, view.surface.surfaceId, selection?.subject);
    const id = 'obs_' + randomUUID();
    this.scopes.set(
      id,
      structuredClone({
        acknowledgment: view.acknowledgment,
        targets: scope === 'view' ? result.targets : [],
      }),
    );
    return {
      receipt: {
        observedReadId: id,
        observation: view.acknowledgment,
        source: 'loaded-preview',
        scope,
        ...(selection && result.targets[0] ? { evidence: result.targets[0] } : {}),
        pointable: scope === 'view' && result.targets.length > 0,
      },
      content: result.content,
    };
  }
  observeDependencies(
    view: View,
    selection: Selection | undefined,
    scope: 'view' | 'source-preview',
  ) {
    if (view.data.kind !== 'run' || view.surface.view !== 'run' || !view.data.dependencies)
      throw new AppProblem('unsupported', 'This Run has no dependency observation.');
    if (scope === 'view' && view.surface.runView?.mode !== 'dependencies')
      throw new AppProblem(
        'invalid_request',
        'Dependencies are not displayed. Use source-preview explicitly.',
      );
    if (selection && selection.kind !== 'run-group' && selection.kind !== 'run-dependency')
      throw new AppProblem(
        'invalid_request',
        'Dependencies require an authored group or directed pair selector.',
      );
    if (this.scopes.size >= 32)
      throw new AppProblem('unsupported', 'This turn reached its observation limit.');
    const result = dependencyObservation(view.data.dependencies, view.surface, scope, selection);
    const id = 'obs_' + randomUUID();
    const targets = scope === 'view' ? result.targets : [];
    this.scopes.set(id, structuredClone({ acknowledgment: view.acknowledgment, targets }));
    return {
      receipt: {
        ...(result.evidence ? { evidence: result.evidence } : {}),
        observedReadId: id,
        observation: view.acknowledgment,
        source: 'loaded-preview',
        scope,
        observedAt: view.data.dependencies.source.observedAt,
        pointable: targets.length > 0,
      },
      content: result.content,
    };
  }
  recordReport(
    view: View,
    evidence: Extract<EvidenceRef, { schemaVersion: 8 }>,
    assertCurrent: () => void,
  ) {
    if (this.scopes.size >= 32)
      throw new AppProblem('unsupported', 'This turn reached its observation limit.');
    const id = 'obs_' + randomUUID();
    this.scopes.set(id, {
      acknowledgment: structuredClone(view.acknowledgment),
      targets: [structuredClone(evidence)],
      assertCurrent,
    });
    return {
      evidence,
      observedReadId: id,
      observation: view.acknowledgment,
      scope: 'view',
      pointable: true,
      source: 'loaded-preview',
    };
  }
  recordNotebook(
    view: View,
    targets: Extract<EvidenceRef, { schemaVersion: 6 }>[],
    scope: 'view' | 'source-preview',
    assertCurrent: () => void,
  ) {
    if (this.scopes.size >= 32)
      throw new AppProblem('unsupported', 'This turn reached its observation limit.');
    const id = 'obs_' + randomUUID();
    this.scopes.set(id, {
      acknowledgment: structuredClone(view.acknowledgment),
      targets: scope === 'view' ? structuredClone(targets) : [],
      assertCurrent,
    });
    return {
      ...(targets.length === 1 ? { evidence: targets[0] } : {}),
      observedReadId: id,
      observation: view.acknowledgment,
      scope,
      pointable: scope === 'view' && targets.length > 0,
      source: 'loaded-preview' as const,
    };
  }
  recordPdf(
    view: View,
    evidence: Extract<EvidenceRef, { schemaVersion: 5 }>,
    scope: 'view' | 'source-preview',
    assertCurrent: () => void,
  ) {
    if (this.scopes.size >= 32)
      throw new AppProblem('unsupported', 'This turn reached its observation limit.');
    const id = 'obs_' + randomUUID();
    this.scopes.set(id, {
      acknowledgment: structuredClone(view.acknowledgment),
      targets: scope === 'view' ? [structuredClone(evidence)] : [],
      assertCurrent,
    });
    return {
      evidence,
      observedReadId: id,
      observation: view.acknowledgment,
      scope,
      pointable: scope === 'view',
      source: 'loaded-preview' as const,
    };
  }
  assertPoint(
    id: string | undefined,
    evidence: EvidenceRef,
    acknowledgment: RenderAcknowledgment | undefined,
  ) {
    const record = id ? this.scopes.get(id) : undefined;
    record?.assertCurrent?.();
    if (
      !record ||
      !acknowledgment ||
      !isDeepStrictEqual(record.acknowledgment, acknowledgment) ||
      !record.targets.some((target) => containsObservedTarget(target, evidence))
    )
      throw new AppProblem(
        'stale_revision',
        'Observe this exact target in the displayed view in this turn, then pass its observedReadId and observation receipt.',
      );
  }
}
