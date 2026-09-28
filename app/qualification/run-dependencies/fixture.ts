import { projectDependencies, type DependencyRead } from '../../contracts/src/index';
export function fixture(size: number) {
  const tasks = Array.from({ length: size }, (_, i) => ({
    instanceId: 'instance-' + i,
    taskId: 'task-' + i,
    name: 'Task ' + i,
    status: i % 7 === 0 ? 'failed' : 'completed',
    attempt: 1,
    reason: null,
    template: false,
    expanded: false,
  }));
  const edges: { fromTaskId: string; toTaskId: string }[] = [];
  const pairs = new Set<string>();
  const width = Math.max(1, Math.ceil(size / 6));
  for (let i = 0; i < size; i++)
    for (let offset = 0; offset < 3; offset++) {
      const to = (Math.floor(i / width) + 1) * width + ((i + offset) % width);
      const key = i + ',' + to;
      if (to < size && edges.length < 200 && !pairs.has(key)) {
        pairs.add(key);
        edges.push({ fromTaskId: 'task-' + i, toTaskId: 'task-' + to });
      }
    }
  const read: DependencyRead = {
    schemaVersion: 1,
    run: {
      projectId: 'prj_qualification',
      runRef: 'run_fixture',
      engineRevision: 'fixture',
      observedAt: 0,
      imageId: 'synthetic',
      runId: 'fixture',
      pipelineName: 'Synthetic qualification',
      status: 'observed',
      preview: { availableTasks: size, returnedTasks: size, truncated: false },
      tasks,
    },
    topology: {
      kind: 'reported',
      availableEntries: edges.length,
      inspectedEntries: edges.length,
      invalidEntries: 0,
      duplicateEntries: 0,
      truncated: false,
      edges,
    },
  };
  return projectDependencies(read);
}
