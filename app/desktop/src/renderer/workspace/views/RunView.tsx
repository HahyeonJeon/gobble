import { RetainedRunFlow } from '../../run-launch/RetainedRunFlow';
import { useEffect, useCallback, useRef, useState } from 'react';
import type { ComponentProps } from 'react';
import type {
  DependencyBrowse,
  DependencyCamera,
  DependencyObservation,
  Surface,
  SavedReportRecord,
} from '@gobble/contracts';
import { ObservationNotice } from './ObservationNotice';
import { RunTasksView } from './RunTasksView';
import { DependencyView } from './dependencies/DependencyView';
import '../../styles/run-dependencies.css';

type Props = Omit<ComponentProps<typeof RunTasksView>, 'filter'> & {
  savedReports: SavedReportRecord[];
  onOpenReport: (instance: string, attempt: number) => Promise<boolean>;
  onOpenSavedReport: (saved: SavedReportRecord) => Promise<boolean>;
  onOpenCurrent: (pipelineId: string) => Promise<boolean>;
  onReady: () => void;
  refreshProblem: string | undefined;
  state: Extract<Surface, { view: 'run' }>['runView'];
  dependencies: DependencyObservation | undefined;
  dependencyProblem: string | undefined;
  onMode: (mode: 'tasks' | 'dependencies') => Promise<boolean>;
  onCamera: (camera: DependencyCamera) => Promise<boolean>;
  onNavigate: (navigation: DependencyBrowse) => Promise<boolean>;
};
export function RunView({
  state,
  savedReports,
  onOpenReport,
  onOpenSavedReport,
  dependencies,
  dependencyProblem,
  onMode,
  onNavigate,
  onCamera,
  onReady,
  onOpenCurrent,
  refreshProblem,
  ...tasks
}: Props) {
  useEffect(onReady, [onReady]);
  const [selecting, setSelecting] = useState(false);
  const selectionFlight = useRef(false);
  const select = useCallback(
    async (value: Parameters<typeof tasks.onSelect>[0]) => {
      if (selectionFlight.current) return false;
      selectionFlight.current = true;
      setSelecting(true);
      try {
        return await tasks.onSelect(value);
      } finally {
        selectionFlight.current = false;
        setSelecting(false);
      }
    },
    [tasks.onSelect],
  );
  // Do not let a fast Discuss click reuse the previous step while selection is
  // being acknowledged by the Workspace owner.
  const ready = tasks.ready && !selecting;
  const controls = { ...tasks, ready, onSelect: select };
  const mode = state?.mode ?? 'tasks';
  const dependencySelected =
    tasks.selection?.kind === 'run-group' || tasks.selection?.kind === 'run-dependency';
  return (
    <div className="run-composite">
      <RetainedRunFlow
        savedReports={savedReports}
        onOpenReport={onOpenReport}
        onOpenSavedReport={onOpenSavedReport}
        {...controls}
        onOpenCurrent={onOpenCurrent}
      />
      <div className="run-modes" role="group" aria-label="Run view">
        <button
          aria-pressed={mode === 'tasks'}
          disabled={!ready}
          onClick={() => void onMode('tasks')}
        >
          Tasks
        </button>
        <button
          aria-pressed={mode === 'dependencies'}
          disabled={!ready}
          onClick={() => void onMode('dependencies')}
        >
          Dependencies
        </button>
      </div>
      <ObservationNotice
        observedAt={tasks.value.observedAt}
        problem={refreshProblem}
        ready={ready}
      />
      <div className="run-mode-content" hidden={mode !== 'tasks'}>
        {dependencySelected && (
          <div className="dependency-retained-selection">
            Dependency target remains selected.{' '}
            <button disabled={!ready} onClick={() => void onMode('dependencies')}>
              Show in Dependencies
            </button>
          </div>
        )}
        <RunTasksView {...controls} filter={state} />
      </div>
      <div className="run-mode-content" hidden={mode !== 'dependencies'}>
        <DependencyView
          value={tasks.value}
          marks={tasks.marks ?? []}
          observation={dependencies}
          problem={dependencyProblem}
          navigation={state?.dependencies}
          selection={tasks.selection}
          ready={ready && mode === 'dependencies'}
          onNavigate={onNavigate}
          onCamera={onCamera}
          onSelect={select}
          onDiscuss={tasks.onDiscuss}
          onOpenLogs={tasks.onOpenLogs}
          onTasks={() => void onMode('tasks')}
          onRefresh={tasks.onRefresh}
        />
      </div>
    </div>
  );
}
