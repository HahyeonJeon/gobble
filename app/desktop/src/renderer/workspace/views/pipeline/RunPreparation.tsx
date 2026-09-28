import { LaunchReviewControls } from '../../../run-launch/LaunchReviewControls';
import { useState } from 'react';
import type { PipelineArtifact } from '@gobble/contracts';
import { useRunPreparation, type PreparationSection as Section } from './useRunPreparation';
import '../../../styles/run-preparation.css';
/** Presentation only. Jobs, private payloads and freshness belong to the service;
 * selecting a discussion section never mutates Current or the User's flow selection. */
export function RunPreparation({
  projectId,
  pipelineId,
  artifact,
}: {
  projectId: string;
  pipelineId: string;
  artifact: PipelineArtifact;
}) {
  const [expanded, setExpanded] = useState(false);
  const [section, setSection] = useState<Section>('settings');
  const {
    value,
    values,
    marks,
    issue,
    busy,
    checking,
    exact,
    engines,
    engineId,
    setSelected,
    chooseEngine,
    refreshEngines,
    prepare,
    cancel,
    discuss,
  } = useRunPreparation(projectId, pipelineId, artifact.artifactId);
  function pointers(section: Section) {
    return marks
      .filter(
        (m) =>
          m.context.preparationId === value?.requestId && m.context.preparationSection === section,
      )
      .slice(-2)
      .map((m, i) => (
        <p className="preparation-mark" key={m.submissionId + i}>
          <strong>Agent · {section}</strong> {m.note}
        </p>
      ));
  }
  return (
    <section className="run-preparation" aria-label="Run preparation">
      <header>
        <button
          className="preparation-toggle"
          aria-expanded={expanded}
          onClick={() => setExpanded((v) => !v)}
        >
          {expanded ? '▾' : '▸'} Run review
        </button>
        <span className="preparation-status">
          {checking
            ? 'Preparing…'
            : exact
              ? 'Prepared review'
              : value?.state === 'ready'
                ? 'Earlier review'
                : 'Review before running'}
        </span>
        <select
          aria-label="Preparation engine"
          disabled={busy || checking}
          value={engineId}
          onChange={(e) => chooseEngine(e.target.value)}
        >
          {engines.length ? (
            engines.map((engine) => (
              <option key={engine.engineId} value={engine.engineId}>
                {engine.label}
              </option>
            ))
          ) : (
            <option value="">No preparation engine available</option>
          )}
        </select>
        <button
          aria-label="Refresh preparation engines"
          disabled={busy || checking}
          onClick={() => void refreshEngines()}
        >
          ↻
        </button>
        <button
          className="primary"
          disabled={busy || checking || !engineId}
          onClick={() => {
            setExpanded(true);
            void prepare();
          }}
        >
          {busy ? 'Connecting…' : value ? 'Prepare again' : 'Prepare run'}
        </button>
      </header>
      {expanded && (
        <div className="preparation-body">
          <p className="preparation-intro">
            Review this flow, its selected data and analysis environment together. Preparing does
            not start the analysis.
          </p>
          {values.length > 1 && (
            <label className="preparation-history">
              Review history{' '}
              <select
                aria-label="Run review history"
                value={value?.requestId ?? ''}
                onChange={(e) => setSelected(e.target.value)}
              >
                {values.map((v) => (
                  <option value={v.requestId} key={v.requestId}>
                    {new Date(v.createdAt).toLocaleTimeString('en-US')} · {v.state}
                  </option>
                ))}
              </select>
            </label>
          )}
          {(issue || value?.issue) && (
            <p role="status" className="preparation-feedback">
              {issue || value?.issue}
            </p>
          )}
          {value?.state === 'preparing' && (
            <div className="preparation-progress">
              <span>
                Gobble is validating and saving the exact plan. You can keep talking in Chat.
              </span>
              <button disabled={busy} onClick={() => void cancel()}>
                Cancel preparation
              </button>
            </div>
          )}
          {value?.state === 'ready' && (
            <>
              {!exact && (
                <p role="status">
                  This review belongs to an earlier observation. Prepare the current flow again
                  before planning a Run.
                </p>
              )}
              <div role="group" aria-label="Run review section" className="preparation-sections">
                {(['data', 'settings', 'environment'] as const).map((part) => (
                  <button
                    key={part}
                    aria-pressed={section === part}
                    onClick={() => setSection(part)}
                  >
                    {part.charAt(0).toUpperCase() + part.slice(1)}
                  </button>
                ))}
              </div>
              <div className="preparation-cards">
                <article hidden={section !== 'data'}>
                  <h3>
                    Data <span>Single-end reads</span>
                  </h3>
                  <strong className="preparation-file">{value.prepared.input.relativePath}</strong>
                  <p>
                    {new Intl.NumberFormat('en-US').format(value.prepared.input.size)} bytes ·
                    observed {new Date(value.prepared.checkedAt).toLocaleString('en-US')}
                  </p>
                  <p>
                    File metadata checked. Data contents and FASTQ validity have not been checked or
                    copied.
                  </p>
                  <button onClick={() => void discuss('data')}>Discuss data</button>
                  {pointers('data')}
                </article>
                <article hidden={section !== 'settings'}>
                  <h3>
                    Settings <span>{value.prepared.steps.length} steps</span>
                  </h3>
                  {value.prepared.flow.steps.map((step) => (
                    <div className="preparation-step" key={step.id}>
                      <strong>{step.display?.stage || step.name}</strong>
                      {value.prepared.flow.schemaVersion === 2 &&
                        'settings' in step &&
                        step.settings.map((s) => (
                          <p key={s.key}>
                            {s.label}
                            <b>{s.value === null ? 'Tool default' : `${s.value} ${s.unit}`}</b>
                          </p>
                        ))}
                    </div>
                  ))}
                  <button onClick={() => void discuss('settings')}>Discuss settings</button>
                  {pointers('settings')}
                </article>
                <article hidden={section !== 'environment'}>
                  <h3>
                    Environment <span>Local analysis engine</span>
                  </h3>
                  <p>One processing task at a time.</p>
                  {value.prepared.steps.map((step) => (
                    <p key={step.id}>
                      {step.label}
                      <b>
                        {step.cpu} CPU · {step.memory}
                      </b>
                    </p>
                  ))}
                  <p>
                    The engine identity is pinned. Tool availability, data staging and an available
                    output location will be checked before Start.
                  </p>
                  <button onClick={() => void discuss('environment')}>Discuss environment</button>
                  {pointers('environment')}
                </article>
              </div>
              <details className="preparation-outputs">
                <summary>Expected results</summary>
                {value.prepared.steps
                  .flatMap((s) => s.outputs)
                  .map((p) => (
                    <p key={p}>{p}</p>
                  ))}
              </details>
              <LaunchReviewControls key={value.requestId} preparation={value} />
            </>
          )}
        </div>
      )}
    </section>
  );
}
