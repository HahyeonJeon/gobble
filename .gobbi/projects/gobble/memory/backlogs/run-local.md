# run-local backlog

## Related-file output From on Run

**Backlogged at:** 2026-08-23T04:56:42Z

**What:** Schedule a task whose input is a related-file `From` of another task's output (BAM to BAI).

**Why backlogged:** D5 locked related-file output `From` and forbade dropping `From` or using thin-slice `sh -c`. That task did not edit the scheduler.

**Context:** A Bind with only `Ext` set, and optionally `Dir`, is a related file of `From`. `Run` `upstreamReady` requires that `ToPort` be an existing input file, so thin-slice task `bai` stays `not-started`. The WGS product proof uses a separate samtools index `TaskSpec` instead. Evidence: `tests/local-e2e/wgs_e2e_thin_test.go` D5 joint-hole stop text.
