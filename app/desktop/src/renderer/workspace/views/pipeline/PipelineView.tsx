import { RunPreparation } from './RunPreparation';
import type { FlowNavigation } from './flow-navigation';
import { useEffect, useRef, useState } from 'react';
import type { PipelineArtifact, PipelineInspection } from '@gobble/contracts';
import { requestId } from '../../useWorkspace';
import { PipelineFlow } from './PipelineFlow';
import { PipelineDetails } from './PipelineDetails';
import type { FlowTarget } from './flow-layout';
import '../../../styles/pipeline.css';

type DiscussionProps = {
  selected: FlowTarget | null;
  onSelect: (target: FlowTarget | null) => void;
  onDiscuss: () => void;
  canDiscuss: boolean;
  marks: Array<{ target: FlowTarget; label: string; author: 'agent' | 'user' }>;
  feedback: string;
  navigation: FlowNavigation;
};
function CheckedFlow({
  artifact,
  selected,
  onSelect: setSelected,
  onDiscuss,
  canDiscuss,
  marks,
  feedback,
  navigation,
}: { artifact: PipelineArtifact } & DiscussionProps) {
  const [list, setListState] = useState(() => {
    if (navigation.artifactId !== artifact.artifactId)
      Object.assign(navigation, {
        artifactId: artifact.artifactId,
        list: false,
        zoom: null,
        left: 0,
        top: 0,
      });
    return navigation.list;
  });
  const setList = (value: boolean) => {
    navigation.list = value;
    setListState(value);
  };
  const dense =
    artifact.flow.steps.length + artifact.flow.inputs.length > 80 ||
    artifact.flow.connections.length > 200;
  return (
    <>
      <div className="pipeline-view-options">
        <span>
          {artifact.flow.steps.length} {artifact.flow.steps.length === 1 ? 'step' : 'steps'} ·{' '}
          {artifact.flow.inputs.length} {artifact.flow.inputs.length === 1 ? 'input' : 'inputs'} ·{' '}
          {artifact.flow.connections.length} connections
        </span>
        <div role="group" aria-label="Flow representation">
          <button
            aria-pressed={!list && !dense}
            disabled={dense}
            onClick={() => {
              setList(false);
              setSelected(null);
            }}
          >
            Flow
          </button>
          <button
            aria-pressed={list || dense}
            onClick={() => {
              setList(true);
              setSelected(null);
            }}
          >
            Step list
          </button>
        </div>
      </div>
      {dense && (
        <p className="pipeline-scope-note">
          This analysis is shown as connected steps so every declared relationship stays available.
        </p>
      )}
      {artifact.flow.steps.length === 0 && artifact.flow.inputs.length === 0 ? (
        <div className="empty-state">
          <h3>No processing steps yet</h3>
          <p>This checked version does not declare any steps or inputs.</p>
        </div>
      ) : (
        <PipelineFlow
          flow={artifact.flow}
          selected={selected}
          onSelect={setSelected}
          list={list || dense}
          marks={marks}
          navigation={navigation}
        />
      )}
      {selected ? (
        <PipelineDetails
          flow={artifact.flow}
          selected={selected}
          onClose={() => setSelected(null)}
          onDiscuss={onDiscuss}
          canDiscuss={canDiscuss}
          feedback={feedback}
          marks={marks}
          onSelect={setSelected}
        />
      ) : (
        <div className="pipeline-selection-hint">
          <span className="pipeline-legend" aria-label="Flow colors">
            <span>
              <i className="legend-input" />
              Input
            </span>
            <span>
              <i className="legend-step" />
              Step
            </span>
            <span>
              <i className="legend-selected" />
              Selected
            </span>
          </span>
          <span>Design view · select to inspect</span>
        </div>
      )}
    </>
  );
}

/** Inspection controls never author source or launch analysis. Semantic rendering
 * only uses the version returned through the Workspace's acknowledged load. */
export function PipelineView({
  value,
  onReady,
  onRefresh,
  ready,
  onReview,
  onRequestChanges,
  ...discussion
}: {
  value: PipelineInspection;
  onReview: () => void;
  onRequestChanges: () => void;
  ready: boolean;
  onReady: () => void;
  onRefresh: () => void;
} & DiscussionProps) {
  const [busy, setBusy] = useState(false);
  const [issue, setIssue] = useState<string | null>(null);
  const request = useRef<string | null>(null);
  const active = useRef(true);
  const refresh = useRef(onRefresh);
  refresh.current = onRefresh;
  useEffect(() => {
    active.current = true;
    return () => {
      active.current = false;
    };
  }, []);
  useEffect(onReady, [onReady, value]);
  useEffect(() => {
    if (value.state !== 'checking') return;
    let stopped = false;
    let timer: ReturnType<typeof setTimeout>;
    async function poll() {
      try {
        const response = await window.gobble.pipelines.inspection({
          projectId: value.projectId,
          pipelineId: value.pipelineId,
        });
        if (stopped) return;
        if (!response.ok) {
          setIssue(response.error.message);
          return;
        }
        if (response.value.state !== value.state || response.value.jobId !== value.jobId) {
          refresh.current();
          return;
        }
        timer = setTimeout(() => void poll(), 2000);
      } catch {
        if (!stopped) setIssue('The check status is unavailable. Refresh to reconnect.');
      }
    }
    timer = setTimeout(() => void poll(), 1000);
    return () => {
      stopped = true;
      clearTimeout(timer);
    };
  }, [value.state, value.jobId, value.projectId, value.pipelineId]);
  async function change(cancel: boolean) {
    if (busy) return;
    setBusy(true);
    setIssue(null);
    try {
      request.current ??= requestId();
      const input = { projectId: value.projectId, pipelineId: value.pipelineId };
      const response =
        cancel && value.jobId
          ? await window.gobble.pipelines.cancelCheck({ ...input, jobId: value.jobId })
          : await window.gobble.pipelines.check({ ...input, requestId: request.current });
      if (!active.current) return;
      if (response.ok) {
        request.current = null;
        refresh.current();
      } else {
        request.current = null;
        setIssue(response.error.message);
      }
    } catch {
      if (active.current)
        setIssue('The check could not be confirmed. Retry to reconnect to the same check.');
    } finally {
      if (active.current) setBusy(false);
    }
  }
  return (
    <div className="surface-view pipeline-view" data-ready={ready} data-testid="pipeline-view">
      <header className="pipeline-toolbar">
        <div>
          <strong>{value.artifact ? 'Pipeline flow' : 'Analysis preview'}</strong>
          <span className="muted">
            {value.state === 'checking'
              ? 'Checking analysis…'
              : value.artifact
                ? `${value.managed ? 'Managed current · ' : ''}Checked ${new Date(value.artifact.checkedAt).toLocaleString('en-US', { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })}`
                : 'No checked version yet'}
          </span>
        </div>
        <div className="pipeline-toolbar-actions">
          <button onClick={onReview}>Changes</button>
          <button
            disabled={!ready || !value.artifact || value.state !== 'ready'}
            onClick={onRequestChanges}
          >
            Discuss changes
          </button>
          <button
            disabled={busy}
            onClick={() => refresh.current()}
            aria-label="Refresh pipeline view"
          >
            Refresh
          </button>
          {value.state === 'checking' ? (
            <button disabled={busy} onClick={() => void change(true)}>
              Cancel check
            </button>
          ) : (
            <button
              className="primary"
              disabled={busy || value.managed}
              onClick={() => void change(false)}
            >
              {busy ? 'Connecting…' : 'Check flow'}
            </button>
          )}
        </div>
      </header>
      {discussion.feedback && !discussion.selected && (
        <div className="pipeline-issue" role="status">
          {discussion.feedback}
        </div>
      )}
      {(issue || value.issue) && (
        <div className="pipeline-issue" role="status" aria-label="Pipeline check status">
          {issue || value.issue}
          {value.artifact && <span> Showing the earlier checked version.</span>}
        </div>
      )}
      {value.artifact ? (
        <CheckedFlow key={value.artifact.artifactId} artifact={value.artifact} {...discussion} />
      ) : (
        <div className="empty-state pipeline-empty">
          <h2>
            {value.state === 'checking' ? 'Preparing your flow' : 'See how this analysis works'}
          </h2>
          <p>
            {value.state === 'checking'
              ? 'Checking the declared inputs and processing steps. Your conversation stays available while this completes.'
              : 'Check this pipeline to see its processing steps, inputs and connections. This does not run the analysis.'}
          </p>
        </div>
      )}
      {value.artifact && (
        <RunPreparation
          key={value.pipelineId}
          projectId={value.projectId}
          pipelineId={value.pipelineId}
          artifact={value.artifact}
        />
      )}
    </div>
  );
}
