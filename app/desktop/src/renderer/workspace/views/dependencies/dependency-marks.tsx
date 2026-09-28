import { referenceAuthor, type DependencyTarget, type SharedReference } from '@gobble/contracts';

export function dependencyMarks(marks: SharedReference[], target: DependencyTarget['selection']) {
  return marks.filter((mark) => {
    if (mark.retracted || mark.evidence.schemaVersion !== 4) return false;
    const selected = mark.evidence.selection;
    return target.kind === 'run-group'
      ? selected.kind === 'run-group' && selected.taskId === target.taskId
      : selected.kind === 'run-dependency' &&
          selected.fromTaskId === target.fromTaskId &&
          selected.toTaskId === target.toTaskId;
  });
}
export function DependencyMarkLabel({ marks }: { marks: SharedReference[] }) {
  if (!marks.length) return null;
  const authors = [...new Set(marks.map(referenceAuthor))].join(', ');
  return (
    <span className="dependency-mark-label" title={authors + ' pointed here'}>
      {authors} · mark
    </span>
  );
}
