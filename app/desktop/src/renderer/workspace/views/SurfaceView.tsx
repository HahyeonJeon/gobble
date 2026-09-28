import { reportTarget } from '@gobble/contracts';
import { ReportView } from './ReportView';
import type { FlowNavigation } from './pipeline/flow-navigation';
import { PipelineSurface } from './pipeline/PipelineSurface';
import { NotebookSurface } from './notebook/NotebookSurface';
import { PdfSurface } from './pdf/PdfSurface';
import { runPresentationState } from '@gobble/contracts';
import { useEffect, useRef, useState } from 'react';
import {
  logPreviewText,
  effectiveReferencePresentation,
  scatterColumns,
  sortTable,
  filterTable,
  type ViewLink,
  sameResource,
  referenceAuthor,
  resolveReferenceTarget,
  type SharedReference,
  type WorkspaceDocument,
  type EvidenceRef,
  type LogTarget,
  type Selection,
  type Surface,
  activeLogStream,
} from '@gobble/contracts';
import type { Command } from '../useWorkspace';
import { TabularControls } from './TabularControls';
import { LinkedSelectionToolbar } from '../../selections/LinkedSelectionToolbar';
import { TableView } from './TableView';
import { TextView } from './TextView';
import { ImageView } from './ImageView';
import { evidenceLabel, markStyle } from '../../shared-context/marks';
import '../../styles/shared-context.css';
import { SelectionToolbar } from '../../selections/SelectionToolbar';
import { ObservedReferencePanel } from './ObservedReferencePanel';
import { RunView } from './RunView';
import { LogView } from './LogView';
import { useSurfaceLoad } from './useSurfaceLoad';
import '../../styles/run-log.css';
import { ReferenceViewBanner } from './ReferenceViewBanner';

const ignoreReady = () => {};

export function SurfaceView({
  surface,
  savedReports,
  refreshVersion,
  surfaces,
  link,
  showLinkedActions,
  title,
  rendererSessionId,
  flowNavigation,
  evidence,
  command,
  onOpenLogs,
  references,
  reveal,
}: {
  surface: Exclude<Surface, { view: 'creation-draft' }>;
  savedReports: NonNullable<WorkspaceDocument['savedReports']>;
  refreshVersion: number;
  surfaces: Surface[];
  link: ViewLink | undefined;
  showLinkedActions: boolean;
  title: string;
  rendererSessionId: string;
  flowNavigation: (surfaceId: string) => FlowNavigation;
  evidence: EvidenceRef | null;
  command: Command;
  references: SharedReference[];
  reveal: WorkspaceDocument['referenceReveal'];
  onOpenLogs: (target: LogTarget) => void;
}) {
  const root = useRef<HTMLDivElement>(null);
  const wasReference = useRef(false);
  const [attaching, setAttaching] = useState(false);
  const { projectId, surfaceId } = surface;
  const revealKey = references.some(
    (item) =>
      item.referenceId === reveal?.referenceId &&
      sameResource(item.evidence.resource, surface.resource),
  )
    ? reveal?.requestId
    : undefined;
  const presentationKey = JSON.stringify([
    revealKey,
    surface.view,
    surface.view === 'scatter' ? surface.scatter : null,
    surface.view === 'table' ? (surface.table ?? null) : null,
    surface.view === 'run' ? (runPresentationState(surface.runView) ?? null) : null,
    surface.view === 'log' ? (surface.logView ?? null) : null,
    surface.view === 'report' ? surface.reportModuleId : null,
    surface.view === 'pdf' ? (surface.pdf ?? null) : null,
    surface.view === 'notebook' ? (surface.notebook ?? null) : null,
    link ? [link.linkId, link.dataRevision, link.filterRevision, link.filter] : null,
  ]);
  const { state, loaded, ready, acknowledge, failRender, changePresentation, refresh } =
    useSurfaceLoad({
      projectId,
      surfaceId,
      rendererSessionId,
      presentationKey,
      linkedRevision: link?.dataRevision,
      refreshVersion,
    });
  useEffect(() => {
    if (!ready || !loaded) return;
    const showing = !!loaded.observedReferenceView;
    if (wasReference.current && !showing)
      root.current
        ?.querySelector<HTMLElement>(
          '.observed-base .log-text, .observed-base input:checked, .observed-base input[type="search"]',
        )
        ?.focus();
    wasReference.current = showing;
  }, [ready, loaded]);
  function reference(selection: Selection): EvidenceRef {
    if (!loaded) throw new Error('An observation is required.');
    if (surface.resource.kind === 'report')
      throw new Error('Report discussion is not available yet.');
    if (surface.resource.kind === 'pipeline')
      throw new Error('Pipeline discussion references are not available yet.');
    if (selection.kind === 'pipeline') throw new Error('Use the checked pipeline selection path.');
    if (selection.kind === 'notebook') {
      if (surface.resource.kind !== 'file') throw new Error('A Notebook file is required.');
      return {
        schemaVersion: 6,
        projectId,
        resource: surface.resource,
        origin: { surfaceId },
        dataRevision: loaded.acknowledgment.dataRevision,
        selection,
      };
    }
    if (selection.kind === 'pdf') {
      if (surface.resource.kind !== 'file') throw new Error('A PDF file is required.');
      return {
        schemaVersion: 5,
        projectId,
        resource: surface.resource,
        origin: { surfaceId },
        dataRevision: loaded.acknowledgment.dataRevision,
        selection,
      };
    }
    if (selection?.kind === 'run-group' || selection?.kind === 'run-dependency') {
      if (!loaded?.dependencyRevision || surface.resource.kind !== 'run')
        throw new Error('A dependency observation is required.');
      return {
        schemaVersion: 4,
        projectId,
        resource: surface.resource,
        dataRevision: loaded.dependencyRevision,
        selection,
      };
    }
    const base = { projectId, origin: { surfaceId }, resource: surface.resource };
    if (selection.kind === 'run-task' || selection.kind === 'log-text') {
      if (!loaded.observedRevision) throw new Error('An observed content revision is required.');
      return { ...base, schemaVersion: 3, dataRevision: loaded.observedRevision, selection };
    }
    return {
      ...base,
      schemaVersion: 2,
      dataRevision: loaded.acknowledgment.dataRevision,
      selection,
    };
  }
  async function select(selection: Selection | null): Promise<boolean> {
    if (!loaded || !ready) return false;
    return command({
      kind: 'select',
      surfaceId,
      acknowledgment: loaded.acknowledgment,
      evidence: selection ? reference(selection) : null,
    });
  }
  async function discussObserved(selection: Selection) {
    if (!loaded || !ready || attaching) return;
    setAttaching(true);
    try {
      if (await select(selection))
        await command({
          kind: 'attach',
          surfaceId,
          acknowledgment: loaded.acknowledgment,
          evidence: reference(selection),
        });
    } finally {
      setAttaching(false);
    }
  }
  if (surface.view === 'scatter')
    return (
      <div className="empty-state" role="status">
        <h2>Chart view retired</h2>
        <p>CSV files are shown as tables. Saved discussion evidence is still available.</p>
        <button
          onClick={() =>
            void command({
              kind: 'open',
              resource: discussionResource,
              pane: 'primary',
              duplicate: false,
            })
          }
        >
          Open source table
        </button>
      </div>
    );
  if (state.kind === 'loading')
    return (
      <div className="empty-state">
        <p>Loading {title}…</p>
      </div>
    );
  if (state.kind === 'error')
    return (
      <div className="empty-state" role="alert">
        <h2>This view is unavailable</h2>
        <p>{state.message}</p>
        <button onClick={refresh}>Retry view</button>
      </div>
    );
  const load = state.load;
  if (surface.resource.kind === 'pipeline') {
    return load.data.kind === 'pipeline' ? (
      <PipelineSurface
        navigation={flowNavigation(surfaceId)}
        load={load}
        surface={surface}
        evidence={evidence}
        references={references}
        ready={ready}
        onReady={acknowledge}
        onRefresh={refresh}
        command={command}
      />
    ) : (
      <div role="alert">The pipeline view is unavailable.</div>
    );
  }
  if (surface.resource.kind === 'report') {
    const saved = surface.resource.saved;
    return load.data.kind === 'report' ? (
      <ReportView
        ready={ready}
        moduleId={surface.view === 'report' ? surface.reportModuleId : undefined}
        onNavigate={(moduleId) =>
          changePresentation(() =>
            command({
              kind: 'reportNavigate',
              surfaceId,
              moduleId,
              acknowledgment: load.acknowledgment,
            }),
          )
        }
        onAttach={() =>
          void command({
            kind: 'attach',
            surfaceId,
            evidence: reportTarget(saved, surfaceId),
            acknowledgment: load.acknowledgment,
          })
        }
        onShare={() =>
          void command({
            kind: 'share',
            surfaceId,
            evidence: reportTarget(saved, surfaceId),
            note: 'Whole saved report',
          })
        }
        marks={references.filter(
          (ref) =>
            !ref.retracted &&
            ref.evidence.schemaVersion === 8 &&
            sameResource(ref.evidence.resource, surface.resource),
        )}
        report={load.data.value}
        onReady={acknowledge}
        onFailure={failRender}
        onOpenLogs={onOpenLogs}
      />
    ) : (
      <p role="alert">The saved report is unavailable.</p>
    );
  }
  if (load.data.kind === 'report') return <p role="alert">This report does not match the View.</p>;
  const referenceView = load.referenceView;
  const observedReference = load.observedReferenceView;
  const observedMark = references.find(
    (item) => item.referenceId === observedReference?.referenceId,
  );
  const effective = effectiveReferencePresentation(surface, link, referenceView);
  if (effective.surface.view === 'report')
    return <p role="alert">Report discussion is not available yet.</p>;
  if (effective.surface.view === 'creation-draft')
    return <div role="alert">Use the creation review to discuss this draft.</div>;
  surface = effective.surface;
  link = effective.link;
  if (surface.resource.kind === 'pipeline' || load.data.kind === 'pipeline')
    return <div role="alert">The reference is incompatible with this pipeline.</div>;
  const discussionResource = surface.resource;
  const source = (ref: EvidenceRef) => ({
    projectId,
    resource: discussionResource,
    dataRevision:
      ref.schemaVersion === 4
        ? (load.dependencyRevision ?? 'unavailable')
        : ref.schemaVersion === 3
          ? load.observedRevision!
          : load.acknowledgment.dataRevision,
    data: load.data,
  });
  if (
    load.data.kind === 'file' &&
    load.data.value.content.kind === 'notebook' &&
    surface.view === 'notebook'
  )
    return (
      <NotebookSurface
        load={load}
        reference={observedMark}
        onFailure={failRender}
        marks={references}
        document={load.data.value.content.document}
        surface={surface}
        title={title}
        acknowledgment={load.acknowledgment}
        ready={ready && !attaching}
        onReady={acknowledge}
        evidence={evidence}
        command={command}
        onSelect={select}
        onRefresh={refresh}
        onNavigate={(page) =>
          changePresentation(() =>
            command({
              kind: 'notebookNavigate',
              surfaceId,
              navigation: { page },
              acknowledgment: load.acknowledgment,
            }),
          )
        }
      />
    );
  if (load.data.kind === 'file' && load.data.value.content.kind === 'pdf' && surface.view === 'pdf')
    return (
      <PdfSurface
        load={load}
        reference={observedMark}
        marks={references.filter((item) => sameResource(item.evidence.resource, surface.resource))}
        content={load.data.value.content}
        surface={surface}
        title={title}
        evidence={evidence}
        acknowledgment={load.acknowledgment}
        ready={ready && !attaching}
        onReady={acknowledge}
        onFailure={failRender}
        onSelect={select}
        onDiscuss={discussObserved}
        onRefresh={refresh}
        command={command}
        onNavigate={(navigation) =>
          changePresentation(() =>
            command({
              kind: 'pdfNavigate',
              surfaceId,
              navigation,
              acknowledgment: load.acknowledgment,
            }),
          )
        }
      />
    );
  const table =
    load.data.kind === 'file' && load.data.value.content.kind === 'table'
      ? load.data.value.content
      : null;
  if (link && table)
    evidence = link.rowKeys.length
      ? {
          schemaVersion: 2,
          projectId,
          resource: discussionResource,
          dataRevision: link.dataRevision,
          origin: { surfaceId },
          selection: {
            kind: 'table',
            coordinateSpace: 'revision-row-column-keys',
            rowKeys: link.rowKeys,
            columns:
              surface.view === 'scatter'
                ? scatterColumns(surface.scatter.spec)
                : table.columns.map((column) => column.id),
          },
        }
      : null;
  const selection =
    evidence && resolveReferenceTarget(evidence, source(evidence)).kind === 'exact'
      ? evidence.selection
      : undefined;
  const relevant = references.filter(
    (item) => !item.retracted && sameResource(item.evidence.resource, surface.resource),
  );
  const focused = relevant.find((item) => item.referenceId === reveal?.referenceId);
  const recent = relevant.filter((item) => item !== focused).slice(-3);
  const shown = focused ? [...recent, focused] : relevant.slice(-4);
  const marks = shown.filter(
    (item) => resolveReferenceTarget(item.evidence, source(item.evidence)).kind === 'exact',
  );
  const resolution = focused
    ? resolveReferenceTarget(focused.evidence, source(focused.evidence))
    : undefined;
  const stale = resolution?.kind === 'historical';
  const data = load.data;
  const modernLog =
    surface.view === 'log' &&
    data.kind === 'log' &&
    'streams' in data.value &&
    activeLogStream(surface.logView?.stream, data.value) !== 'legacy';
  const observedView = data.kind === 'run' || modernLog;

  const controls =
    table && !referenceView ? (
      <TabularControls
        surface={surface}
        content={table}
        link={link}
        acknowledgment={load.acknowledgment}
        ready={ready && !referenceView}
        command={command}
      />
    ) : null;
  return (
    <div ref={root} className="surface-view" data-ready={ready} aria-label={title + ' preview'}>
      {referenceView && (!link || showLinkedActions) && (
        <ReferenceViewBanner
          view={referenceView}
          reference={references.find((item) => item.referenceId === referenceView.referenceId)}
          command={command}
        />
      )}
      {link && table && showLinkedActions && (
        <LinkedSelectionToolbar
          surfaces={surfaces}
          link={link}
          content={table}
          acknowledgment={load.acknowledgment}
          ready={ready}
          command={command}
        />
      )}
      {!observedView && !link && selection && evidence && (
        <SelectionToolbar
          key={JSON.stringify(evidence)}
          evidence={evidence}
          surfaceId={surfaceId}
          title={title}
          ready={ready}
          command={command}
          acknowledgment={load.acknowledgment}
        />
      )}
      {!observedView && !link && !selection && (
        <div className="view-attachment-action">
          <button
            className="text-button"
            disabled={!ready || attaching}
            aria-label={'Add view preview of ' + title + ' to message'}
            onClick={() => {
              setAttaching(true);
              void command({
                kind: 'attach',
                surfaceId,
                acknowledgment: load.acknowledgment,
                evidence: {
                  schemaVersion: 2,
                  projectId,
                  origin: { surfaceId },
                  resource: discussionResource,
                  dataRevision: load.acknowledgment.dataRevision,
                },
              }).finally(() => setAttaching(false));
            }}
          >
            Add view preview
          </button>
        </div>
      )}
      {shown.some((item) => item.referenceId !== referenceView?.referenceId) && (
        <div className="reference-strip" aria-label="Shared references in this view">
          {shown
            .filter((item) => item.referenceId !== referenceView?.referenceId)
            .map((item) => (
              <button
                key={item.referenceId}
                style={markStyle(item)}
                onClick={() => void command({ kind: 'reveal', referenceId: item.referenceId })}
                title={item.note || evidenceLabel(item.evidence)}
              >
                {referenceAuthor(item)} · {evidenceLabel(item.evidence)}
                {resolveReferenceTarget(item.evidence, source(item.evidence)).kind === 'historical'
                  ? ' · older version'
                  : resolveReferenceTarget(item.evidence, source(item.evidence)).kind ===
                      'unavailable'
                    ? ' · unavailable'
                    : ''}
              </button>
            ))}
        </div>
      )}
      {resolution?.kind === 'unavailable' && (
        <div className="selection-warning" role="status">
          This reference cannot be located in this preview. No highlight has been applied.
        </div>
      )}
      {stale && (
        <div className="selection-warning" role="status">
          This shared reference targets an older version. Its highlight is not applied to the
          current content.
        </div>
      )}
      {evidence && evidence.dataRevision !== source(evidence).dataRevision && (
        <div className="selection-warning" role="status">
          Selection changed. Select from this version again.
          <button
            onClick={() =>
              void command({
                kind: 'select',
                surfaceId,
                evidence: null,
                acknowledgment: load.acknowledgment,
              })
            }
          >
            Clear old selection
          </button>
        </div>
      )}
      {observedReference && observedMark && (
        <ObservedReferencePanel
          view={observedReference}
          data={data}
          reference={observedMark}
          onReady={acknowledge}
          command={command}
        />
      )}
      <div className="observed-base" hidden={!!observedReference}>
        {data.kind === 'file' ? (
          data.value.content.kind === 'table' ? (
            <>
              {controls && <div className="tabular-controls">{controls}</div>}
              <TableView
                referenceRequestId={referenceView?.requestId}
                content={{
                  ...data.value.content,
                  rows: sortTable(
                    data.value.content,
                    filterTable(data.value.content, link?.filter ?? { kind: 'all' }),
                    surface.view === 'table' ? (surface.table?.sort ?? null) : null,
                  ),
                }}
                sourceRows={data.value.content.rows.length}
                includedColumns={
                  surface.view === 'table' ? (surface.table?.columns ?? undefined) : undefined
                }
                onColumnsChange={
                  surface.view === 'table' && !referenceView
                    ? (columns) =>
                        changePresentation(() =>
                          command({
                            kind: 'tableSettings',
                            surfaceId,
                            acknowledgment: load.acknowledgment,
                            settings: {
                              columns,
                              sort: surface.view === 'table' ? (surface.table?.sort ?? null) : null,
                            },
                          }),
                        )
                    : undefined
                }
                emptyLabel={
                  link?.filter.kind === 'equals' ? 'No rows match this filter.' : undefined
                }
                selection={selection}
                onReady={acknowledge}
                onSelect={select}
                canSelect={ready}
                canChooseColumns={!link && !referenceView}
                marks={marks}
                reveal={reveal}
              />
            </>
          ) : data.value.content.kind === 'text' ? (
            <TextView
              text={data.value.content.text}
              label={title + ' text'}
              footer="Text preview · Read only"
              selection={selection}
              onReady={acknowledge}
              onSelect={select}
              canSelect={ready}
              marks={marks}
              reveal={reveal}
            />
          ) : data.value.content.kind === 'notebook' ? (
            <p role="status">Reopen this file as a Notebook view.</p>
          ) : data.value.content.kind === 'pdf' ? (
            <p role="alert">Reopen this file as a PDF view.</p>
          ) : (
            <ImageView
              content={data.value.content}
              revision={data.value.revision}
              title={title}
              selection={selection}
              onReady={acknowledge}
              onSelect={select}
              canSelect={ready}
              marks={marks}
              reveal={reveal}
            />
          )
        ) : data.kind === 'run' ? (
          <RunView
            savedReports={savedReports}
            onOpenReport={(instance, attempt) =>
              command({ kind: 'openReport', runRef: data.value.runRef, instance, attempt })
            }
            onOpenSavedReport={(saved) =>
              command({
                kind: 'open',
                resource: { kind: 'report', saved },
                pane: 'secondary',
                duplicate: false,
              })
            }
            onOpenCurrent={(pipelineId) =>
              command({
                kind: 'open',
                resource: { kind: 'pipeline', pipelineId },
                pane: 'primary',
                duplicate: false,
              })
            }
            marks={marks}
            value={data.value}
            state={surface.view === 'run' ? surface.runView : undefined}
            dependencies={data.dependencies}
            dependencyProblem={data.dependencyProblem}
            onMode={(mode) =>
              changePresentation(() => command({ kind: 'runMode', surfaceId, mode }))
            }
            onCamera={(camera) => command({ kind: 'dependencyCamera', surfaceId, camera })}
            onNavigate={({ query, representation }) => {
              const previous = surface.view === 'run' ? surface.runView?.dependencies : undefined;
              if (
                (previous?.query ?? '') === query &&
                (previous?.representation ?? 'graph') === representation
              )
                return Promise.resolve(true);
              return changePresentation(() =>
                command({
                  kind: 'dependencyNavigation',
                  surfaceId,
                  navigation: { query, representation },
                }),
              );
            }}
            selection={selection}
            onReady={observedReference ? ignoreReady : acknowledge}
            onOpenLogs={onOpenLogs}
            ready={ready && !attaching && !observedReference}
            onSelect={select}
            onDiscuss={discussObserved}
            onRefresh={refresh}
            refreshProblem={load.refreshProblem}
            onFilter={(filter) =>
              changePresentation(() => command({ kind: 'runFilter', surfaceId, filter }))
            }
          />
        ) : modernLog && surface.view === 'log' && 'streams' in data.value ? (
          <LogView
            marks={marks}
            value={data.value}
            stream={activeLogStream(surface.logView?.stream, data.value) as 'stdout' | 'stderr'}
            revision={load.observedRevision!}
            selection={selection}
            onReady={observedReference ? ignoreReady : acknowledge}
            ready={ready && !attaching && !observedReference}
            onSelect={select}
            onDiscuss={discussObserved}
            onRefresh={refresh}
            refreshProblem={load.refreshProblem}
            onStream={(stream) =>
              changePresentation(() => command({ kind: 'logStream', surfaceId, stream }))
            }
          />
        ) : (
          <>
            <div className="observation-toolbar">
              <span>Legacy combined preview</span>
              <button
                disabled={!ready}
                onClick={() =>
                  void changePresentation(() =>
                    command({ kind: 'logStream', surfaceId, stream: 'auto' }),
                  )
                }
              >
                Use stream view
              </button>
            </div>
            <TextView
              text={logPreviewText(data.value)}
              label={title + ' logs'}
              footer={
                'Attempt ' +
                data.value.attempt +
                ' · Up to 4096 bytes per stream · Observed ' +
                new Intl.DateTimeFormat('en', { timeStyle: 'medium' }).format(data.value.observedAt)
              }
              selection={selection}
              onReady={acknowledge}
              onSelect={select}
              canSelect={ready}
              marks={marks}
              reveal={reveal}
            />
          </>
        )}
      </div>
    </div>
  );
}
