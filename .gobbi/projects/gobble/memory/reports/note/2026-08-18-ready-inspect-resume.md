# Ready inspect and resume

The session shipped public `Inspect`, `Resume`, and `Release` on `13c42a68fed6e01a9ffbe46b6e3a5273c5696ddd`. Occupancy is an owner record on `.gobble/run.json`. `Release` closes occupancy, marks in-flight work `incomplete`, and is not deletion.

`Inspect(workspace, view, instance)` is read-only. Views are `run`, `instances`, `errors`, `logs`, `timing`, `DAG`, `lineage`, `remaining`, and `reuse`. Remaining is classified on all latest attempts, then instance-filtered for emit.

`Resume(graph, workspace, cap)` re-validates, treats wait-only edge change as plan-drift, scopes `output-exists` to dests this Resume would publish that are not authorized replace dests, attributes dests by checksum or producer lineage, and uses staged replace plus per-attempt isolate. Script and env persist on the attempt. Dest rename is a reuse miss.

First-check is `go test ./...`. Live proof at 2026-08-18T15:06:50Z: gobble 2.399s, engine 2.560s, wgs-e2e 34.153s; extra `go test ./tests/wgs-e2e -count=1` 36.898s; Docker 28.3.3. There is no CLI. Publication is local.

See [Design Memory](../../design/README.md) and [history](../../history/2026-08-18-ready-inspect-resume.md).
