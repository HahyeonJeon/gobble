import { useEffect, useRef, useState } from 'react';
import { Sidebar } from './Sidebar';
import { ProjectChat } from '../chat/ProjectChat';
import { ResourceWorkspace } from './ResourceWorkspace';
import { Splitter } from './Splitter';
import { Icon } from './Icon';
import { useWorkspace, type Command } from './useWorkspace';
import { useAgents } from '../agents/useAgents';
import { SelectionMenu } from '../selections/SelectionMenu';
import { ProjectBar } from './ProjectBar';
import { useExplorer } from './useExplorer';
import { AccountPanel } from '../agents/AccountPanel';
import '../styles/agents.css';

export function WorkspaceApp() {
  const model = useWorkspace();
  const agents = useAgents(model.report);
  const showChatButton = useRef<HTMLButtonElement>(null);
  const collapseRequested = useRef(false);
  const [compact, setCompact] = useState(() => window.innerWidth < 1120);
  const [compactView, setCompactView] = useState<'workspace' | 'chat'>('workspace');
  const [focusSurfaceId, setFocusSurfaceId] = useState<string | null>(null);
  const [composerFocusRequest, setComposerFocusRequest] = useState(0);
  const workspaceCommand: Command = async (action) => {
    const ok = await model.command(action);
    if (ok && action.kind === 'currentTask') setCompactView('workspace');
    if (
      ok &&
      (action.kind === 'discussSelection' ||
        (action.kind === 'attach' && action.evidence.schemaVersion >= 3))
    ) {
      setCompactView('chat');
      setComposerFocusRequest((value) => value + 1);
    }
    return ok;
  };
  const [previewWidth, setPreviewWidth] = useState<number | null>(null);
  useEffect(() => {
    const query = window.matchMedia('(max-width: 1119px)');
    const update = () => setCompact(query.matches);
    query.addEventListener('change', update);
    update();
    return () => query.removeEventListener('change', update);
  }, []);
  const explorer = useExplorer(compact);
  const document = model.state?.document ?? null;
  useEffect(() => {
    if (document?.chat.collapsed && collapseRequested.current) {
      showChatButton.current?.focus();
      collapseRequested.current = false;
    }
  }, [document?.chat.collapsed]);
  useEffect(() => {
    if (document?.referenceReveal) setCompactView('workspace');
  }, [document?.referenceReveal?.requestId]);
  const layout = document?.workspace.layout;
  const active = layout
    ? document?.workspace.surfaces.find(
        (surface) =>
          surface.surfaceId ===
          (document.activePane === 'secondary' && layout.kind === 'split'
            ? layout.secondary.activeSurfaceId
            : layout.primary.activeSurfaceId),
      )
    : undefined;
  useEffect(
    () =>
      window.gobble.workspace.onShortcut((shortcut) => {
        if (shortcut === 'open-folder') {
          void model.chooseFolder();
          return;
        }
        if (!document) return;
        if (shortcut === 'split')
          void model.command({
            kind: 'arrange',
            layout: layout?.kind === 'split' ? 'single' : 'split',
          });
        if (shortcut === 'pin' && active)
          void model.command({ kind: 'pin', surfaceId: active.surfaceId, pinned: !active.pinned });
        if (shortcut === 'close-view' && active)
          void model.command({ kind: 'close', surfaceId: active.surfaceId });
      }),
    [model.chooseFolder, model.command, document, layout, active],
  );
  const project = model.state?.projects.find(
    (project) => project.projectId === document?.workspace.projectId,
  );
  const hideResources = compact && compactView === 'chat';
  const hideChat = !document || (compact ? compactView !== 'chat' : document.chat.collapsed);
  const chatWidth = previewWidth ?? document?.chat.width ?? 420;
  return (
    <div className="app-shell">
      <a className="skip-link" href="#workspace-main">
        Skip to workspace
      </a>
      <ProjectBar
        projects={model.state?.projects ?? []}
        project={project}
        busy={model.busy}
        onProject={(id) => void model.openProject(id)}
        onFolder={() => void model.chooseFolder()}
        explorerOpen={explorer.open}
        onToggleExplorer={explorer.toggle}
        explorerTriggerRef={explorer.trigger}
        compact={compact}
        compactView={compactView}
        onRegion={(region) => {
          explorer.closeDrawer();
          setCompactView(region);
        }}
        chatCollapsed={document?.chat.collapsed ?? false}
        showChatRef={showChatButton}
        onShowChat={() => void model.command({ kind: 'chat', collapsed: false })}
        layout={layout?.kind}
        onLayout={(layout) => void model.command({ kind: 'arrange', layout })}
        selections={
          document && (
            <SelectionMenu
              key={document.workspace.projectId}
              document={document}
              command={model.command}
              onReveal={(id) => {
                setFocusSurfaceId(id);
                setCompactView('workspace');
              }}
            />
          )
        }
        account={<AccountPanel model={agents} />}
      />
      <div className={'project-body' + (compact && explorer.open ? ' explorer-overlay' : '')}>
        {compact && explorer.open && (
          <button
            className="explorer-backdrop"
            aria-label="Close files"
            onClick={explorer.closeDrawer}
          />
        )}
        {project && document && (
          <Sidebar
            key={project.projectId}
            project={project}
            hidden={!explorer.open}
            panelRef={explorer.panel}
            report={model.report}
            onOpen={(resource, destination) => {
              void model.openResource(resource, destination).then((ok) => {
                if (ok) {
                  setCompactView('workspace');
                  explorer.closeDrawer();
                }
              });
            }}
          />
        )}
        <div className="project-shell">
          <div className="app-status sr-only" role="status">
            {model.state
              ? 'Workspace ready'
              : model.error
                ? 'Workspace unavailable'
                : 'Opening workspace…'}
          </div>
          {model.error && (
            <div className="error-banner" role="alert">
              <span>{model.error}</span>
              <button onClick={() => void model.connect()}>Reload workspace</button>
              <button className="icon-button" aria-label="Dismiss error" onClick={model.clearError}>
                <Icon name="close" />
              </button>
            </div>
          )}
          {model.state?.notice && <p className="notice">{model.state.notice}</p>}
          {agents.status?.problem && (
            <p role="alert" className="notice">
              {agents.status.problem}
            </p>
          )}
          <div
            className="project-columns"
            style={{
              gridTemplateColumns:
                compact || hideChat ? 'minmax(0, 1fr)' : `minmax(0, 1fr) 12px ${chatWidth}px`,
            }}
          >
            <main
              id="workspace-main"
              className="workspace-main"
              tabIndex={-1}
              hidden={hideResources}
            >
              {!model.state && !model.error ? (
                <div className="empty-state">
                  <h2>Opening your workspace…</h2>
                </div>
              ) : document && model.state ? (
                <ResourceWorkspace
                  key={document.workspace.projectId}
                  document={document}
                  rendererSessionId={model.state.rendererSessionId}
                  command={workspaceCommand}
                  hidden={hideResources}
                  report={model.report}
                  focusSurfaceId={focusSurfaceId}
                  onFocusRestored={() => setFocusSurfaceId(null)}
                  onBrowse={(pane) => {
                    void model.command({ kind: 'focusPane', pane }).then((ok) => {
                      if (ok) explorer.show();
                    });
                  }}
                />
              ) : (
                <section className="project-chooser">
                  <span className="empty-icon">
                    <Icon name="folder" />
                  </span>
                  <h2>Start with a Project</h2>
                  <p>
                    Open a folder to explore your files and results.
                    <br />
                    Your views and drafts stay together when you return.
                  </p>
                  <button
                    className="primary-button"
                    disabled={model.busy}
                    onClick={() => void model.chooseFolder()}
                  >
                    <Icon name="plus" />
                    Open a Project folder
                  </button>
                  {(model.state?.projects.length ?? 0) > 0 && (
                    <div className="recent-projects">
                      <h3>Registered projects</h3>
                      {model.state?.projects.map((project) => (
                        <button
                          key={project.projectId}
                          onClick={() => void model.openProject(project.projectId)}
                        >
                          <Icon name="folder" />
                          <span>{project.name}</span>
                          <Icon name="arrow" />
                        </button>
                      ))}
                    </div>
                  )}
                </section>
              )}
            </main>
            {document && !compact && !hideChat && (
              <Splitter
                label="Resize chat"
                controls="project-chat"
                orientation="vertical"
                value={document.chat.width}
                min={360}
                max={560}
                step={20}
                unit="pixels"
                edge="end"
                onPreview={setPreviewWidth}
                onCommit={(width) => model.command({ kind: 'resizeChat', width })}
              />
            )}
            {document && (
              <ProjectChat
                key={document.workspace.projectId}
                agents={agents}
                composerFocusRequest={composerFocusRequest}
                document={document}
                command={model.command}
                saving={model.saving}
                busy={model.busy}
                updateDraft={model.updateDraft}
                hidden={hideChat}
                compact={compact}
                onCollapse={() => {
                  collapseRequested.current = true;
                  void model.command({ kind: 'chat', collapsed: true }).then((ok) => {
                    if (!ok) collapseRequested.current = false;
                  });
                }}
              />
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
