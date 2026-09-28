import { PipelineReviewPanel } from './PipelineReviewPanel';
import type { FlowNavigation } from './flow-navigation';
import { useEffect, useRef, useState } from 'react';
import {
  pipelineTarget,
  pipelineSubject,
  pipelineSubjectLabel,
  referenceAuthor,
  resolveReferenceTarget,
  sameResource,
  type EvidenceRef,
  type SurfaceLoad,
  type SharedReference,
  type Surface,
} from '@gobble/contracts';
import type { Command } from '../../useWorkspace';
import { PipelineView } from './PipelineView';
import { flowSelector, flowTarget } from './flow-selection';
import type { FlowTarget } from './flow-layout';
import { ObservedReferencePanel } from '../ObservedReferencePanel';

export function PipelineSurface({
  load,
  surface,
  evidence,
  references,
  ready,
  onReady,
  onRefresh,
  command,
  navigation,
}: {
  load: SurfaceLoad;
  surface: Surface;
  evidence: EvidenceRef | null;
  references: SharedReference[];
  ready: boolean;
  onReady: () => void;
  onRefresh: () => void;
  command: Command;
  navigation: FlowNavigation;
}) {
  const [reviewing, setReviewing] = useState(navigation.reviewing);
  const active = useRef(false);
  useEffect(() => {
    active.current = true;
    return () => {
      active.current = false;
    };
  }, []);
  const [pending, setPending] = useState(false);
  const [feedback, setFeedback] = useState('');
  if (load.data.kind !== 'pipeline') return <div role="alert">Pipeline view unavailable.</div>;
  const value = load.data.value,
    artifact = value.artifact;
  const source = {
    projectId: value.projectId,
    resource: surface.resource,
    dataRevision: artifact?.artifactId ?? 'unavailable',
    data: load.data,
  };
  const selected =
    evidence?.schemaVersion === 7 && resolveReferenceTarget(evidence, source).kind === 'exact'
      ? flowTarget(evidence.selection.subject)
      : null;
  const relevant = references
    .filter((r) => !r.retracted && sameResource(r.evidence.resource, surface.resource))
    .slice(-4);
  const exact = relevant.filter(
    (r) =>
      r.evidence.schemaVersion === 7 && resolveReferenceTarget(r.evidence, source).kind === 'exact',
  );
  const marks = exact.flatMap((r) =>
    r.evidence.schemaVersion === 7
      ? [
          {
            target: flowTarget(r.evidence.selection.subject),
            label: referenceAuthor(r),
            author: r.author.kind,
          },
        ]
      : [],
  );
  const view = load.observedReferenceView,
    reference = references.find((r) => r.referenceId === view?.referenceId);
  async function choose(target: FlowTarget | null) {
    if (!ready || view || !artifact) return;
    setFeedback('');
    await command({
      kind: 'select',
      surfaceId: surface.surfaceId,
      acknowledgment: load.acknowledgment,
      evidence: target
        ? pipelineTarget(value, flowSelector(artifact.flow, target), surface.surfaceId)
        : null,
    });
  }
  async function attach() {
    if (!ready || pending || view || !selected || !artifact) return;
    setPending(true);
    setFeedback('');
    try {
      const selector = flowSelector(artifact.flow, selected);
      const label = pipelineSubjectLabel(pipelineSubject(artifact.flow, selector)).slice(0, 160);
      const ok = await command({
        kind: 'attach',
        surfaceId: surface.surfaceId,
        acknowledgment: load.acknowledgment,
        evidence: pipelineTarget(value, selector, surface.surfaceId),
      });
      if (ok) setFeedback(`Added ${label} to message`);
    } finally {
      setPending(false);
    }
  }
  if (reviewing)
    return (
      <PipelineReviewPanel
        navigation={navigation}
        projectId={value.projectId}
        pipelineId={value.pipelineId}
        onReturn={() => {
          navigation.reviewing = false;
          setReviewing(false);
          onRefresh();
        }}
        onAdopted={onRefresh}
      />
    );
  return (
    <div className="pipeline-surface">
      {relevant.length > 0 && (
        <div className="reference-strip" aria-label="Shared pipeline references">
          {relevant.map((r) => {
            const valid = resolveReferenceTarget(r.evidence, source).kind === 'exact';
            const label =
              valid && r.evidence.schemaVersion === 7 && artifact
                ? pipelineSubjectLabel(pipelineSubject(artifact.flow, r.evidence.selection.subject))
                : r.label;
            return (
              <button
                key={r.referenceId}
                onClick={() => void command({ kind: 'reveal', referenceId: r.referenceId })}
              >
                {referenceAuthor(r)} · {label}
                {valid ? '' : ' · older or unavailable version'}
              </button>
            );
          })}
        </div>
      )}
      {view && reference && (
        <ObservedReferencePanel
          view={view}
          data={load.data}
          reference={reference}
          onReady={onReady}
          command={command}
        />
      )}
      {!view && (
        <div className="pipeline-base">
          <PipelineView
            value={value}
            onReview={() => {
              void window.gobble.workspace.invalidate(load.acknowledgment).then((result) => {
                if (!active.current) return;
                if (result.ok) {
                  navigation.reviewing = true;
                  setReviewing(true);
                } else setFeedback(result.error.message);
              });
            }}
            onRequestChanges={() => {
              if (artifact)
                void window.gobble.pipelineReviews
                  .select({
                    projectId: value.projectId,
                    context: { pipelineId: value.pipelineId, baseArtifactId: artifact.artifactId },
                  })
                  .then((result) =>
                    setFeedback(
                      result.ok ? 'Current Pipeline added to Chat.' : result.error.message,
                    ),
                  );
            }}
            ready={ready}
            onReady={onReady}
            onRefresh={onRefresh}
            selected={selected}
            onSelect={(target) => void choose(target)}
            onDiscuss={() => void attach()}
            canDiscuss={ready && !pending}
            marks={marks}
            feedback={feedback}
            navigation={navigation}
          />
        </div>
      )}
    </div>
  );
}
