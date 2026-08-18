# Engine design review after WGS e2e

Two prior-art studies disagreed. nf-core-as-written maps Nextflow’s task. Isolate-keep keeps Gobble’s isolate. They answer different questions. Keep isolate. Take from nf-core only declared regular files plus declared literals.

This session kept: regular-file publish, isolate copy, Docker network-none, static From graph, D5 A from-path wait, literal Command, host-env exclusion, structured errors.

Ranks 1–7 to adapt next session: sidecar group of regular files; plan-time edge wait path; apply recorded Resources; control-plane image digest and pull hygiene; static per-task env map of declared literals; named Script XOR Command; explicit resume API with occupy still the default second Run.

Rejected now: directory ports, channels/wildcards/checkpoints, implicit hash-resume, publishDir, conda/host-env, in-process nf-core, scatter/gather identity, auto-delete temp.

Not a copy of the evaluator or partner reports. Source: session `tmp/p2-review/synthesis.md`.
