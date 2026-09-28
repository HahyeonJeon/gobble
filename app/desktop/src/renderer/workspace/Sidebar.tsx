import { useState, type Ref } from 'react';
import { PipelinesBrowser } from './navigation/PipelinesBrowser';
import type { ProjectInfo } from '@gobble/contracts';
import { FilesBrowser } from './navigation/FilesBrowser';
import { RunsBrowser } from './navigation/RunsBrowser';
import type { OpenResource, ReportError } from './navigation/types';

export function Sidebar({
  project,
  hidden,
  panelRef,
  onOpen,
  report,
}: {
  project: ProjectInfo;
  hidden: boolean;
  panelRef: Ref<HTMLElement>;
  onOpen: OpenResource;
  report: ReportError;
}) {
  const [pipelineRevision, setPipelineRevision] = useState(0);
  return (
    <aside
      ref={panelRef}
      id="project-explorer"
      className="sidebar"
      aria-label="Project navigation"
      hidden={hidden}
    >
      <div className="project-resources">
        <PipelinesBrowser
          projectId={project.projectId}
          revision={pipelineRevision}
          onOpen={onOpen}
        />
        <FilesBrowser
          project={project}
          onOpen={onOpen}
          report={report}
          onPipelineRegistered={() => setPipelineRevision((value) => value + 1)}
        />
        <RunsBrowser projectId={project.projectId} onOpen={onOpen} report={report} />
      </div>
    </aside>
  );
}
