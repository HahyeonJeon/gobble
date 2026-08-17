# Go Tips

## Keep From identity on the pipeline pointer

**Context:** Gobble snapshots binds into an id-keyed engine graph.

**Tip:** A `From` handle belongs to a pipeline only when `h.task.pipe` or `h.pipe` is the composed pipeline. Task-id equality is not enough. Module reuse produces colliding dotted ids across pipelines.

**Application:** Apply one `foreignFrom` predicate at every site that records a spec or a DAG identity, including output binds.

## FromIn readiness uses the downstream input path

**Context:** A task becomes ready from a `From` bind, including `FromIn`.

**Tip:** Readiness looks at the downstream task’s rendered input path. The upstream output-port path is the wrong file when the bind names an input port.

**Application:** Require the named upstream task to have succeeded, then check the consuming input path. Do not resolve `FromIn` only against upstream outputs.

## Leftover not-started is not success

**Context:** Public `Run` returns nil only when every task succeeded.

**Tip:** A leftover `not-started` task is a failed defect. Do not treat unfinished or unpublished work as success.

**Application:** After the scheduler stops, any task that is not `succeeded` must become a `failed` defect so `Run` cannot return nil.
