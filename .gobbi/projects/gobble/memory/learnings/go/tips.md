# Go Tips

## Keep From identity on the pipeline pointer

**Context:** Gobble snapshots binds into an id-keyed engine graph.

**Tip:** A `From` handle belongs to a pipeline only when `h.task.pipe` or `h.pipe` is the composed pipeline. Task-id equality is not enough. Module reuse produces colliding dotted ids across pipelines.

**Application:** Apply one `foreignFrom` predicate at every site that records a spec or a DAG identity, including output binds.
