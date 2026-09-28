import type { FlowNavigation } from './flow-navigation';
import { FollowUpOrigin } from '../../../run-feedback/FollowUpOrigin';
import { useEffect, useRef, useState } from 'react';
import {
  type PipelineReviewState,
  type PipelineProposal,
  type PipelineChange,
  type PipelineReviewContext,
} from '@gobble/contracts';
import { PipelineFlow } from './PipelineFlow';
import { PipelineChangeDetails } from './PipelineChangeDetails';
import '../../../styles/pipeline-review.css';

function contextFor(p: PipelineProposal, change?: PipelineChange): PipelineReviewContext {
  return {
    pipelineId: p.pipelineId,
    proposalId: p.proposalId,
    baseArtifactId: p.base.artifactId,
    proposedArtifactId: p.proposed!.artifactId,
    ...(change ? { changeId: change.id, side: 'both' as const } : {}),
  };
}
export function PipelineReviewPanel({
  projectId,
  pipelineId,
  onReturn,
  onAdopted,
  navigation,
}: {
  projectId: string;
  pipelineId: string;
  onReturn: () => void;
  onAdopted: () => void;
  navigation: FlowNavigation;
}) {
  const [state, setState] = useState<PipelineReviewState | null>(null);
  const [chosen, setChosen] = useState<string | null>(navigation.proposalId);
  const [selection, setSelection] = useState<string | null>(navigation.changeId);
  const [issue, setIssue] = useState('');
  const [busy, setBusy] = useState(false);
  const [confirm, setConfirm] = useState(false);
  const [uncertain, setUncertain] = useState(false);
  const active = useRef(true);
  useEffect(() => {
    active.current = true;
    let cancelled = false;
    let timer: ReturnType<typeof setTimeout>;
    const poll = async () => {
      try {
        const result = await window.gobble.pipelineReviews.list({ projectId, pipelineId });
        if (cancelled) return;
        if (result.ok) {
          setState(result.value);
          navigation.proposalId ??= result.value.proposals.at(-1)?.proposalId ?? null;
          const proposal = result.value.proposals.find(
            (p) => p.proposalId === navigation.proposalId,
          );
          navigation.changeId ??= proposal?.comparison?.changes[0]?.id ?? null;
          setChosen(navigation.proposalId);
          setSelection(navigation.changeId);
        } else setIssue(result.error.message);
      } catch {
        if (!cancelled) setIssue('Comparison history is unavailable. Return and reconnect.');
      } finally {
        if (!cancelled) timer = setTimeout(() => void poll(), 2000);
      }
    };
    void poll();
    return () => {
      cancelled = true;
      active.current = false;
      clearTimeout(timer);
    };
  }, [projectId, pipelineId, navigation]);
  const proposal = state?.proposals.find((p) => p.proposalId === chosen);
  const changes = proposal?.comparison?.changes ?? [];
  const selected = changes.find((c) => c.id === selection);
  const adopted = !!proposal?.proposed && state?.currentArtifactId === proposal.proposed.artifactId;
  const outdated = !!proposal && state?.currentArtifactId !== proposal.base.artifactId && !adopted;
  const canAdopt =
    proposal?.state === 'ready' &&
    !!proposal.proposed &&
    !!changes.length &&
    !proposal.comparison?.gaps.length &&
    !outdated &&
    !adopted;
  const choose = (id: string) => {
    navigation.changeId = id;
    setSelection(id);
    setConfirm(false);
  };
  async function discuss() {
    if (!proposal?.proposed || !selected) return;
    setBusy(true);
    try {
      const result = await window.gobble.pipelineReviews.select({
        projectId,
        context: contextFor(proposal, selected),
      });
      if (active.current)
        setIssue(
          result.ok
            ? 'Added this exact change to Chat. Write your message there.'
            : result.error.message,
        );
    } finally {
      if (active.current) setBusy(false);
    }
  }
  async function adopt(query = false) {
    if (!proposal?.proposed || busy) return;
    setBusy(true);
    setIssue('');
    const requestId = 'req_adopt_' + proposal.proposalId.slice(4);
    try {
      const result = query
        ? await window.gobble.pipelineReviews.outcome({ projectId, pipelineId, requestId })
        : await window.gobble.pipelineReviews.adopt({
            projectId,
            pipelineId,
            requestId,
            proposalId: proposal.proposalId,
            artifactId: proposal.proposed.artifactId,
          });
      if (!active.current) return;
      if (result.ok && result.value.state === 'adopted') {
        setUncertain(false);
        setConfirm(false);
        setIssue('Adopted as current. No Run was started.');
        onAdopted();
      } else if (result.ok) {
        setUncertain(false);
        setConfirm(false);
        setIssue('No adoption was recorded. Check the current version before trying again.');
      } else {
        setUncertain(true);
        setIssue(result.error.message + ' Check the saved outcome before continuing.');
      }
    } catch {
      if (active.current) {
        setUncertain(true);
        setIssue('Adoption could not be confirmed. Check the saved outcome.');
      }
    } finally {
      if (active.current) setBusy(false);
    }
  }
  return (
    <section className="pipeline-review" aria-label="Pipeline change review">
      <header className="review-header">
        <button onClick={onReturn}>← Current flow</button>
        <strong>Changes</strong>
        <span>
          {adopted
            ? 'Adopted'
            : outdated
              ? 'Earlier proposal'
              : proposal?.state === 'checking'
                ? 'Checking…'
                : 'Review before adopting'}
        </span>
      </header>
      {state && (state.proposals.length > 1 || (!!chosen && !proposal)) && (
        <label className="review-history">
          Proposal{' '}
          <select
            aria-label="Proposal history"
            value={proposal?.proposalId ?? ''}
            onChange={(e) => {
              navigation.proposalId = e.target.value;
              navigation.changeId =
                state.proposals.find((p) => p.proposalId === e.target.value)?.comparison?.changes[0]
                  ?.id ?? null;
              setChosen(navigation.proposalId);
              setSelection(navigation.changeId);
              setConfirm(false);
              setUncertain(false);
            }}
          >
            {!proposal && (
              <option value="" disabled>
                Choose a proposal
              </option>
            )}
            {state.proposals.map((p, i) => (
              <option key={p.proposalId} value={p.proposalId}>
                {i + 1}. {p.summary.slice(0, 70) || 'Pipeline proposal'} · {p.state}
              </option>
            ))}
          </select>
        </label>
      )}
      {!proposal ? (
        <div className="empty-state">
          <h3>
            {!state
              ? 'Loading comparisons…'
              : chosen
                ? 'The selected proposal is unavailable'
                : 'No proposals yet'}
          </h3>
          <p>
            {chosen && state?.proposals.length
              ? 'Choose another proposal from the history above, or return to the current flow.'
              : 'Return to the current flow, choose Discuss changes, and allow the Agent to propose an update in Chat.'}
          </p>
        </div>
      ) : (
        <>
          {proposal.followUp && (
            <FollowUpOrigin key={proposal.proposalId} value={proposal.followUp} />
          )}
          <div className="review-summary">
            <span className="review-eyebrow">AGENT PROPOSAL</span>
            <p>{proposal.summary || 'Pipeline update'}</p>
          </div>
          {proposal.state === 'checking' ? (
            <div className="empty-state">
              <h3>Checking the proposed analysis</h3>
              <p>The current Pipeline remains available. You can keep talking in Chat.</p>
            </div>
          ) : proposal.state === 'failed' ? (
            <div className="pipeline-issue" role="status">
              {proposal.issue}
            </div>
          ) : (
            proposal.proposed && (
              <>
                <div className="review-flow-heading">
                  <strong>Proposed</strong>
                  <span className="review-key">
                    ◆ Changed <span>＋ Added</span>
                    <small>Gray steps are unchanged</small>
                  </span>
                </div>
                <div className="review-flow">
                  <PipelineFlow
                    flow={proposal.proposed.flow}
                    selected={selected ? { kind: 'step', id: selected.stepId } : null}
                    onSelect={(target) => {
                      const change = changes.find((c) => c.stepId === target.id);
                      if (change) choose(change.id);
                    }}
                    list={
                      proposal.proposed.flow.steps.length > 80 ||
                      proposal.proposed.flow.connections.length > 200
                    }
                    changes={changes.map((c, i) => ({
                      stepId: c.stepId,
                      kind: c.kind,
                      number: i + 1,
                    }))}
                  />
                </div>
                <nav className="review-change-nav" aria-label="Changes">
                  {changes.map((c, i) => (
                    <button
                      key={c.id}
                      aria-pressed={selected?.id === c.id}
                      onClick={() => choose(c.id)}
                    >
                      <b>{i + 1}</b>
                      {c.kind === 'added-step' ? 'Add quality check' : c.label}
                    </button>
                  ))}
                </nav>
                {selection && !selected && (
                  <p role="status">The selected change is unavailable. Choose another change.</p>
                )}
                {selected && (
                  <PipelineChangeDetails
                    flow={proposal.proposed.flow}
                    change={selected}
                    index={changes.indexOf(selected)}
                    total={changes.length}
                    busy={busy}
                    onDiscuss={() => void discuss()}
                  />
                )}
                {!!proposal.comparison?.gaps.length && (
                  <div className="review-gaps" role="status">
                    <strong>More review support is needed before adoption</strong>
                    <ul>
                      {proposal.comparison.gaps.map((gap, i) => (
                        <li key={i}>{gap}</li>
                      ))}
                    </ul>
                  </div>
                )}
                {state?.marks
                  .filter(
                    (m) =>
                      m.context.proposalId === proposal.proposalId &&
                      m.context.proposedArtifactId === proposal.proposed?.artifactId,
                  )
                  .map((m, i) => (
                    <button
                      className="review-reference"
                      key={i}
                      onClick={() => m.context.changeId && choose(m.context.changeId)}
                    >
                      Agent reference · {m.note || 'Show change'}
                    </button>
                  ))}
              </>
            )
          )}
        </>
      )}
      {issue && (
        <div className="review-feedback" role="status">
          {issue}
        </div>
      )}
      {adopted && (
        <button className="review-next" onClick={onReturn}>
          Review a new analysis →
        </button>
      )}
      <footer className="review-adoption">
        <span>
          {adopted
            ? 'This is the current version. No Run was started.'
            : outdated
              ? 'Current changed. This comparison remains available as history.'
              : confirm
                ? proposal?.managed
                  ? 'Replace the managed current version with this exact checked proposal?'
                  : 'Create an app-managed source copy as current? The imported folder remains separate.'
                : 'Your current analysis stays unchanged until you adopt.'}
        </span>
        {uncertain ? (
          <button disabled={busy} onClick={() => void adopt(true)}>
            Check adoption outcome
          </button>
        ) : confirm ? (
          <>
            <button disabled={busy} onClick={() => setConfirm(false)}>
              Cancel
            </button>
            <button
              className="primary-button"
              disabled={busy || !canAdopt}
              onClick={() => void adopt()}
            >
              Confirm adoption
            </button>
          </>
        ) : (
          <button
            className="primary-button"
            disabled={!canAdopt || busy}
            onClick={() => setConfirm(true)}
          >
            Adopt proposal
          </button>
        )}
      </footer>
    </section>
  );
}
