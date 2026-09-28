import { useState, type ReactNode, type Ref } from 'react';
import type { ProjectInfo } from '@gobble/contracts';
import { ModalDialog } from '../ui/ModalDialog';
import { Icon } from './Icon';

export function ProjectBar({
  projects,
  project,
  busy,
  onProject,
  onFolder,
  explorerOpen,
  onToggleExplorer,
  explorerTriggerRef,
  compact,
  compactView,
  onRegion,
  chatCollapsed,
  showChatRef,
  onShowChat,
  layout,
  onLayout,
  account,
  selections,
}: {
  projects: ProjectInfo[];
  project: ProjectInfo | undefined;
  busy: boolean;
  onProject: (id: string | null) => void;
  onFolder: () => void;
  explorerOpen: boolean;
  onToggleExplorer: () => void;
  explorerTriggerRef: Ref<HTMLButtonElement>;
  compact: boolean;
  compactView: 'workspace' | 'chat';
  onRegion: (region: 'workspace' | 'chat') => void;
  chatCollapsed: boolean;
  showChatRef: Ref<HTMLButtonElement>;
  onShowChat: () => void;
  layout: 'single' | 'split' | undefined;
  onLayout: (layout: 'single' | 'split') => void;
  account: ReactNode;
  selections: ReactNode;
}) {
  const [projectsOpen, setProjectsOpen] = useState(false);
  function choose(id: string | null) {
    setProjectsOpen(false);
    onProject(id);
  }
  return (
    <header className="project-bar">
      {project && (
        <button
          ref={explorerTriggerRef}
          className="icon-button"
          aria-label="Toggle files"
          title="Toggle files"
          aria-expanded={explorerOpen}
          aria-controls="project-explorer"
          onClick={onToggleExplorer}
        >
          <Icon name="sidebar" />
        </button>
      )}
      <h1 aria-label={project?.name ?? 'Projects'}>
        <button
          className="project-switcher"
          aria-label="Switch Project"
          aria-description={
            project ? 'Current Project: ' + project.name : 'Choose or open a Project'
          }
          title={project?.name ?? 'Projects'}
          aria-haspopup="dialog"
          aria-expanded={projectsOpen}
          disabled={busy}
          onClick={() => setProjectsOpen(true)}
        >
          <span className="truncate">{project?.name ?? 'Projects'}</span>
          <Icon name="chevron" />
        </button>
      </h1>
      <div className="project-bar-actions">
        {busy && (
          <small className="muted" role="status">
            Opening…
          </small>
        )}
        {compact && project && (
          <nav className="compact-view-switch" aria-label="Project regions">
            <button
              aria-pressed={compactView === 'workspace'}
              onClick={() => onRegion('workspace')}
            >
              Workspace
            </button>
            <button aria-pressed={compactView === 'chat'} onClick={() => onRegion('chat')}>
              Chat
            </button>
          </nav>
        )}
        {!compact && project && chatCollapsed && (
          <button ref={showChatRef} onClick={onShowChat}>
            Show chat
          </button>
        )}
        {project ? (
          <div className="layout-toggle" aria-label="Workspace layout">
            <button
              className="icon-button"
              aria-label="Single pane"
              title="Single pane"
              aria-pressed={layout === 'single'}
              onClick={() => onLayout('single')}
            >
              <Icon name="single" />
            </button>
            <button
              className="icon-button"
              aria-label="Two panes"
              title="Two panes"
              aria-pressed={layout === 'split'}
              onClick={() => onLayout('split')}
            >
              <Icon name="split" />
            </button>
          </div>
        ) : (
          <button onClick={onFolder} disabled={busy}>
            <Icon name="folder" /> Open folder
          </button>
        )}
        {selections}
        {account}
      </div>
      {projectsOpen && (
        <ModalDialog
          title="Projects"
          className="project-picker"
          onClose={() => setProjectsOpen(false)}
        >
          <nav aria-label="Projects" className="project-list">
            {projects.map((item) => (
              <button
                key={item.projectId}
                aria-current={item.projectId === project?.projectId ? 'page' : undefined}
                onClick={() => choose(item.projectId)}
                disabled={busy}
              >
                <Icon name="folder" />
                <span>{item.name}</span>
              </button>
            ))}
            {projects.length === 0 && <p className="muted">Your folders will appear here.</p>}
          </nav>
          <footer>
            <button
              disabled={busy}
              onClick={() => {
                setProjectsOpen(false);
                onFolder();
              }}
            >
              <Icon name="plus" /> Open folder
            </button>
            <button disabled={busy} onClick={() => choose(null)}>
              All projects
            </button>
          </footer>
        </ModalDialog>
      )}
    </header>
  );
}
