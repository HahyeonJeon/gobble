import { useEffect, useRef, useState } from 'react';
import type {
  CreationCandidate,
  CreationDraft,
  CreationAdoption as Adopted,
} from '@gobble/contracts';
/** User confirmation and outcome reconciliation. The service alone publishes Current. */
export function CreationAdoption({
  draft,
  candidate,
  onRefresh,
  onOpen,
}: {
  draft: CreationDraft;
  candidate: CreationCandidate;
  onRefresh: () => Promise<void>;
  onOpen: (pipelineId: string) => void;
}) {
  const [confirm, setConfirm] = useState(false),
    [name, setName] = useState('Read quality analysis'),
    [busy, setBusy] = useState(false),
    [uncertain, setUncertain] = useState(false),
    [issue, setIssue] = useState(''),
    [adopted, setAdopted] = useState<Adopted | null>(null);
  const active = useRef(true);
  const requestId = 'req_create_' + candidate.candidateId.slice(4);
  const input = { projectId: draft.projectId, draftId: draft.draftId, requestId };
  useEffect(() => {
    let live = true;
    active.current = true;
    void window.gobble.creation
      .outcome({ projectId: draft.projectId, draftId: draft.draftId, requestId })
      .then((r) => {
        if (!live) return;
        if (r.ok && r.value.state === 'adopted') setAdopted(r.value.adoption);
      });
    return () => {
      live = false;
      active.current = false;
    };
  }, [draft.projectId, draft.draftId, requestId]);
  async function adopt(query = false) {
    if (busy || !candidate.artifact) return;
    setBusy(true);
    setIssue('');
    try {
      const r = query
        ? await window.gobble.creation.outcome(input)
        : await window.gobble.creation.adopt({
            ...input,
            candidateId: candidate.candidateId,
            artifactId: candidate.artifact.artifactId,
            expectedGeneration: candidate.generation,
            name: name.trim(),
          });
      if (!active.current) return;
      if (r.ok && r.value.state === 'adopted') {
        setAdopted(r.value.adoption);
        setUncertain(false);
        setConfirm(false);
        await onRefresh();
      } else if (r.ok) {
        setUncertain(false);
        setIssue('No adoption was saved. Review and confirm to try again.');
      } else {
        setIssue(r.error.message);
        setUncertain(['internal', 'runtime_unavailable'].includes(r.error.code));
      }
    } catch {
      if (active.current) {
        setUncertain(true);
        setIssue('The result is unconfirmed. Check the saved outcome before trying again.');
      }
    } finally {
      if (active.current) setBusy(false);
    }
  }
  const saved = adopted ?? draft.adoption;
  if (saved)
    return (
      <section className="creation-adoption success" aria-label="Creation adoption">
        <strong>Created · No Run started.</strong>
        <span>{saved.name}</span>
        <button onClick={() => onOpen(saved.pipelineId)}>Open pipeline</button>
      </section>
    );
  const eligible =
    draft.state === 'draft' &&
    draft.generation === candidate.generation &&
    candidate.state === 'ready' &&
    !!candidate.artifact &&
    !candidate.artifact.check.gaps.length;
  if (!eligible) return null;
  return (
    <section className="creation-adoption" aria-label="Creation adoption">
      {!confirm && !uncertain ? (
        <button className="primary-button" onClick={() => setConfirm(true)}>
          Adopt as new pipeline
        </button>
      ) : (
        <>
          <div>
            <strong>Create this Pipeline?</strong>
            <p>The checked design becomes its first version. No analysis will run.</p>
          </div>
          <label>
            Pipeline name
            <input
              aria-label="Pipeline name"
              maxLength={200}
              value={name}
              disabled={busy || uncertain}
              onChange={(e) => setName(e.target.value)}
            />
          </label>
          <dl>
            <div>
              <dt>Data</dt>
              <dd>{candidate.artifact?.input.relativePath} · Single-end</dd>
            </div>
            {candidate.artifact?.check.review.flow.steps.flatMap((step) =>
              step.settings.map((setting) => (
                <div key={step.id + '/' + setting.key}>
                  <dt>{setting.label}</dt>
                  <dd>
                    {setting.value === null ? 'Tool default' : `${setting.value} ${setting.unit}`}
                  </dd>
                </div>
              )),
            )}
          </dl>
          <div className="toolbar">
            {uncertain ? (
              <button disabled={busy} onClick={() => void adopt(true)}>
                Check saved outcome
              </button>
            ) : (
              <>
                <button
                  className="primary-button"
                  disabled={busy || !name.trim()}
                  onClick={() => void adopt()}
                >
                  {busy ? 'Creating…' : 'Confirm creation'}
                </button>
                <button disabled={busy} onClick={() => setConfirm(false)}>
                  Cancel
                </button>
              </>
            )}
          </div>
        </>
      )}
      {issue && <p role="status">{issue}</p>}
    </section>
  );
}
