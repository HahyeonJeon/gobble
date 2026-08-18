# Design Tips

## Sidecar group is not related-file Ext

**Context:** Five BWA siblings and BAM+BAI still need one bind each. Same-task `From` is a cycle.

**Tip:** A sidecar group is N declared regular files, not a directory and not related-file `Ext`.

## A static env map is not an env port

**Context:** Containers inherit image `ENV`. Host inheritance stays forbidden.

**Tip:** `Env: {"HOME":"/work"}` is an author literal, not a Nextflow env port.
