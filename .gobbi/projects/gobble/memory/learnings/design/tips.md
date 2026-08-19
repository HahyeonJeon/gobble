# Design Tips

## Sidecar group is public Group

**Context:** Five BWA siblings and BAM+BAI still need one bind each unless authored as a Group. Same-task `From` is a cycle.

**Tip:** Public `Group` is N declared regular files on one Bind. It is not a directory and not related-file `Ext`.

**Application:** Use one Group port when siblings travel together. Keep single-file binds when a consumer wants one sibling.

## A static env map is not an env port

**Context:** Containers inherit image `ENV`. Host inheritance stays forbidden.

**Tip:** `Env: {"HOME":"/work"}` is an author literal, not a Nextflow env port.

## File, Group, and Tree are the artifact sum

**Context:** Directory-output tools such as STAR genomeGenerate need a declared directory. Glob Set and Nextflow wildcards were offered and rejected this horizon.

**Tip:** A Bind is File, Group, or Tree. Tree is a declared directory plus dest `.gobble-tree.json`. Directory is placement, not an artifact. Group stays named regular files. Do not add glob Set this horizon.

**Application:** STAR `--genomeDir` is Tree. BAM+BAI stays Group. Undeclared cwd files do not publish.
