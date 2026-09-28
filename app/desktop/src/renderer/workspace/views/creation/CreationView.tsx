import { CreationAdoption } from './CreationAdoption';
import { useEffect, useRef, useState } from 'react';
import type {
  CreationState,
  CreationContext,
  CreationTarget,
  WorkspaceDocument,
} from '@gobble/contracts';
import { PipelineFlow } from '../pipeline/PipelineFlow';
import { PipelineDetails } from '../pipeline/PipelineDetails';
import { CreationInput } from './CreationInput';
import { CreationEngine } from './CreationEngine';
import { requestId } from '../../useWorkspace';
import '../../../styles/creation.css';

/** Native draft lifecycle, checked candidate navigation and explicit Chat references. */
export function CreationView({
  projectId,
  draftId,
  document,
  onOpenPipeline,
}: {
  projectId: string;
  draftId: string;
  document: WorkspaceDocument;
  onOpenPipeline: (pipelineId: string) => void;
}) {
  const [state, setState] = useState<CreationState | null>(null),
    [issue, setIssue] = useState(''),
    [busy, setBusy] = useState(false),
    [choosing, setChoosing] = useState(false),
    [candidateId, setCandidateId] = useState<string | null>(null),
    [selected, setSelected] = useState<CreationTarget | null>(null),
    [list, setList] = useState(false),
    [feedback, setFeedback] = useState('');
  const active = useRef(true);
  const base = { projectId, draftId };
  useEffect(() => {
    active.current = true;
    let timer: ReturnType<typeof setTimeout>;
    async function poll() {
      try {
        const r = await window.gobble.creation.state({ projectId, draftId });
        if (!active.current) return;
        if (r.ok) setState(r.value);
        else setIssue(r.error.message);
      } finally {
        if (active.current) timer = setTimeout(() => void poll(), 1500);
      }
    }
    void poll();
    return () => {
      active.current = false;
      clearTimeout(timer);
    };
  }, [projectId, draftId]);
  async function refresh() {
    const r = await window.gobble.creation.state(base);
    if (active.current) {
      if (r.ok) {
        setState(r.value);
        setIssue('');
      } else setIssue(r.error.message);
    }
  }
  async function action(run: () => Promise<void>) {
    if (busy) return;
    setBusy(true);
    setIssue('');
    try {
      await run();
    } catch {
      if (active.current)
        setIssue('The request could not be confirmed. Refresh before trying again.');
    } finally {
      if (active.current) setBusy(false);
    }
  }
  async function attach(context: CreationContext) {
    const r = await window.gobble.creation.select({ projectId, context });
    if (!active.current) return;
    if (!r.ok) setIssue(r.error.message);
    else setFeedback('Added to your next message.');
  }
  const draft = state?.draft;
  const candidates = state?.candidates ?? [];
  const candidate = candidates.find((c) => c.candidateId === candidateId) ?? candidates.at(-1);
  const artifact = candidate?.artifact;
  const context: CreationContext | null =
    candidate && artifact
      ? {
          kind: 'candidate',
          base: 'none',
          draftId,
          generation: candidate.generation,
          candidateId: candidate.candidateId,
          artifactId: artifact.artifactId,
          ...(selected ? { target: selected } : {}),
        }
      : null;
  const historical =
    !!candidate && (candidate.generation !== draft?.generation || draft?.state !== 'draft');
  const marks = (state?.marks ?? []).flatMap((m) =>
    m.context.kind === 'candidate' &&
    m.context.candidateId === candidate?.candidateId &&
    m.context.target
      ? [
          {
            target: m.context.target,
            label: document.workspace.agents.find((a) => a.agentId === m.agentId)?.name ?? 'Agent',
            author: 'agent' as const,
            note: m.note,
          },
        ]
      : [],
  );
  function select(target: CreationTarget) {
    setSelected(target);
    setFeedback('');
  }
  return (
    <div className="creation-view">
      <header className="creation-toolbar">
        <div>
          <strong>New pipeline</strong>
          <small>
            {draft?.state === 'discarded'
              ? 'Discarded draft'
              : draft?.state === 'adopted'
                ? 'Creation history · Adopted'
                : historical
                  ? 'Earlier proposal'
                  : 'Draft · Not yet a Pipeline'}
          </small>
        </div>
        <div className="toolbar">
          <button disabled={busy} onClick={() => void refresh()}>
            Refresh
          </button>
          {draft?.state === 'draft' && (
            <button
              disabled={busy}
              onClick={() =>
                void action(async () => {
                  const r = await window.gobble.creation.discard({
                    ...base,
                    requestId: requestId(),
                    expectedGeneration: draft.generation,
                  });
                  if (!r.ok) setIssue(r.error.message);
                  else await refresh();
                })
              }
            >
              Discard draft
            </button>
          )}
        </div>
      </header>
      {issue && (
        <p className="inline-error" role="alert">
          {issue}
        </p>
      )}
      {!draft ? (
        <p>Loading draft…</p>
      ) : (
        <>
          <div className="creation-summary">
            <span>
              {draft.input?.relativePath ?? 'No data selected'}
              {draft.input ? ' · Single-end reads' : ''}
            </span>
            {draft.state === 'draft' && (
              <div className="toolbar">
                {draft.input && (
                  <button disabled={busy} onClick={() => setChoosing((v) => !v)}>
                    {choosing ? 'Keep current data' : 'Change data'}
                  </button>
                )}
                {draft.input && (
                  <button
                    disabled={busy}
                    onClick={() =>
                      void action(() =>
                        attach({ kind: 'draft', draftId, generation: draft.generation }),
                      )
                    }
                  >
                    Discuss this draft
                  </button>
                )}
              </div>
            )}
          </div>
          {draft.state === 'draft' && <CreationEngine />}
          {(choosing || !draft.input) && draft.state === 'draft' ? (
            <CreationInput
              projectId={projectId}
              busy={busy}
              onSelect={(resourceId) =>
                void action(async () => {
                  const r = await window.gobble.creation.update({
                    ...base,
                    requestId: requestId(),
                    expectedGeneration: draft.generation,
                    brief: draft.brief,
                    resourceId,
                    readLayout: 'single-end',
                  });
                  if (!r.ok) {
                    setIssue(r.error.message);
                    return;
                  }
                  await refresh();
                  setChoosing(false);
                  await attach({ kind: 'draft', draftId, generation: r.value.generation });
                })
              }
            />
          ) : (
            <>
              {candidate && (
                <CreationAdoption
                  key={candidate.candidateId}
                  draft={draft}
                  candidate={candidate}
                  onRefresh={refresh}
                  onOpen={onOpenPipeline}
                />
              )}
              {candidates.length > 0 && (
                <div className="creation-proposal-bar">
                  <label>
                    Proposal{' '}
                    <select
                      aria-label="Creation proposal"
                      value={candidate?.candidateId ?? ''}
                      onChange={(e) => {
                        setCandidateId(e.target.value);
                        setSelected(null);
                        setFeedback('');
                      }}
                    >
                      {candidates.map((c, i) => (
                        <option key={c.candidateId} value={c.candidateId}>
                          {i + 1} · {c.summary || 'New pipeline'} · {c.state}
                        </option>
                      ))}
                    </select>
                  </label>
                  <button onClick={() => setList((v) => !v)}>
                    {list ? 'Show flow' : 'Show list'}
                  </button>
                </div>
              )}
              {candidate?.state === 'checking' && (
                <div className="creation-empty" role="status">
                  <h3>Checking the proposed design…</h3>
                  <p>
                    The engine is checking its steps and data connections. Your analysis is not
                    running.
                  </p>
                  <button
                    disabled={busy}
                    onClick={() =>
                      void action(async () => {
                        const r = await window.gobble.creation.cancel({
                          ...base,
                          candidateId: candidate.candidateId,
                        });
                        if (!r.ok) setIssue(r.error.message);
                        await refresh();
                      })
                    }
                  >
                    Cancel check
                  </button>
                </div>
              )}
              {candidate && candidate.state !== 'checking' && candidate.state !== 'ready' && (
                <div className="creation-empty">
                  <h3>
                    {candidate.state === 'unsupported'
                      ? 'This proposal needs revision'
                      : candidate.state === 'cancelled'
                        ? 'Check cancelled'
                        : 'Check could not complete'}
                  </h3>
                  <p>{candidate.issue || 'Discuss the draft with your Agent to try again.'}</p>
                </div>
              )}
              {artifact && candidate?.state === 'ready' ? (
                <>
                  <div className="creation-flow-heading">
                    <strong>Proposed</strong>
                    <span>
                      + {artifact.check.review.flow.steps.length} steps · Checked design
                      {historical ? ' · Historical' : ''}
                    </span>
                  </div>
                  <div className="creation-flow">
                    <PipelineFlow
                      creation
                      flow={artifact.check.review.flow}
                      selected={selected}
                      onSelect={select}
                      list={list}
                      marks={marks}
                      changes={artifact.check.review.flow.steps.map((s, i) => ({
                        stepId: s.id,
                        kind: 'added-step',
                        number: i + 1,
                      }))}
                    />
                  </div>
                  {marks.length > 0 && (
                    <div className="creation-marks" aria-label="Agent creation references">
                      {marks.map((m, i) => (
                        <button key={i} onClick={() => select(m.target)}>
                          {m.label}: {m.note || m.target.kind}
                        </button>
                      ))}
                    </div>
                  )}
                  <div className="creation-comparison">
                    <section className="creation-current">
                      <small>{draft.state === 'adopted' ? 'Current at creation' : 'Current'}</small>
                      <h3>No current version</h3>
                      <p>
                        {draft.state === 'adopted'
                          ? 'Creation history: every step and connection was a new addition. Open the Pipeline for its current version.'
                          : 'This is a new analysis. Every step and connection is a proposed addition.'}
                      </p>
                      <small>Checking a design does not create a Pipeline or start a Run.</small>
                    </section>
                    <section className="creation-proposed">
                      <small>Proposed · Added</small>
                      {selected ? (
                        <PipelineDetails
                          flow={artifact.check.review.flow}
                          selected={selected}
                          onSelect={select}
                          onClose={() => setSelected(null)}
                          onDiscuss={() => context && void action(() => attach(context))}
                          canDiscuss={!busy}
                          feedback={feedback}
                          marks={marks}
                        />
                      ) : (
                        <div className="creation-empty">
                          <h3>Select an addition</h3>
                          <p>
                            Choose data, a step or a connection above. Select a setting below a step
                            to discuss that exact value.
                          </p>
                          <button
                            disabled={busy}
                            onClick={() => context && void action(() => attach(context))}
                          >
                            Discuss proposed flow
                          </button>
                        </div>
                      )}
                    </section>
                  </div>
                </>
              ) : (
                !candidate && (
                  <div className="creation-empty">
                    <h3>Design it together in Chat</h3>
                    <p>
                      Describe your goal to the right and enable a proposal for that message. Your
                      Agent’s checked flow will appear here.
                    </p>
                    <p>Start with trimming single-end reads and a FastQC quality report.</p>
                  </div>
                )
              )}
            </>
          )}
          {feedback && !selected && <p role="status">{feedback}</p>}
        </>
      )}
    </div>
  );
}
