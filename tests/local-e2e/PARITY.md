# Scenario live parity map

Old test → new test → assertions that moved. Replaced copies are deleted after this map.

| Old test | New test | Assertions that moved |
|---|---|---|
| `run_local_live_test.go` `TestRunLocalFixture` | `tests/local-e2e` `TestRunLocalFixture` | Docker and process published fixture text; container cwd `/work`; image/host task status, executor, image, and resources; second `Run` is `occupied-workspace`. Extended with remaining empty, `Release`, `Resume`, and reuse `reused-identity-matched`. |
| `run_local_live_test.go` `TestRunLocalBadImage` | `tests/local-e2e` `TestRunLocalBadImage` | Named docker failure not skip; failed docker dest unpublished; independent process dest published; failed isolate work directory present; image `failed` and host `succeeded`. |
| `assets/run_live_test.go` `TestRNASeqRun` | `tests/local-e2e` `TestRNASeqRecover` | STAR uniquely mapped reads ≥ 10 and splices on at least one sample; MultiQC html exists. Paths now per-sample. Added remaining empty, occupy, `Release`, `Resume`, reuse, `work/deseq2/results.csv` gene id and `log2FoldChange`, MultiQC inputs omit BAM. |
| `assets/run_live_test.go` `TestMethylSeqRun` | `tests/local-e2e` `TestMethylSeqRecover` | No `in/Bisulfite_Genome`; unique PE alignment floor; methylation call rows. Paths now per-sample extractor dests. Added remaining empty, occupy, `Release`, `Resume`, reuse, published extractor outputs per sample, merged MultiQC html. |
| `tests/wgs-e2e` `TestWGSSuccessInspectReleaseResume` | `tests/local-e2e` `TestWGSSuccessInspectReleaseResume` | `assets.WGS()` success; remaining empty; MultiQC html and two sample BAMs; occupy; `Release` occupancy inactive; `Resume`; reuse `reused-identity-matched`. |
| `tests/wgs-e2e` `TestThinFailFixtureInspectReleaseResume` | `tests/local-e2e` `TestThinFailFixtureInspectReleaseResume` | Contained fail unpublished dest; independent dest published; remaining `fail`; `Release`; resume reruns `fail`. Now calls `requireDocker`. |
| `tests/wgs-e2e` `TestWGSSpinePlan` | `tests/local-e2e` `TestWGSSpinePlan` | Hermetic Strelka and BCFTools plan shape. Untagged. |
| `tests/wgs-e2e` `TestWGSThinSlicePlan` | `tests/local-e2e` `TestWGSThinSlicePlan` | Hermetic four-task thin plan. Untagged. |
| `tests/wgs-e2e` `TestWGSFixtureManifestPins` | `tests/local-e2e` `TestWGSFixtureManifestPins` | Manifest URL, bytes, SHA256. Untagged. |
| `tests/cli-valid/runlocal_live_test.go` `TestRunLocalCLIRecover` | `tests/local-e2e` `TestRunLocalCLIRecover` | Compose/validate/plan golden; run stdout `{"op":"run"}`; fixture outputs; remaining empty; occupy; release; resume; reuse. Pipe stays `tests/cli-valid/runlocal`. |
| `tests/cli-valid/wgs_live_test.go` `TestWGSCLIRecover` | `tests/local-e2e` `TestWGSCLIRecover` | Compose/validate/plan; run stdout `{"op":"run"}`; MultiQC and two BAMs; remaining empty; occupy; release; resume; reuse. Pipe moved to `tests/local-e2e/wgs`. |
| `tests/cli-valid/rnaseq_live_test.go` `TestRNASeqCLIRun` | `tests/local-e2e` `TestRNASeqCLIRecover` | Compose/validate/plan; run stdout `{"op":"run"}`; STAR uniquely mapped and splices; MultiQC html; remaining empty. Extended with occupy, release, resume, reuse, `--sample`, DESeq2 results shape. Pipe moved to `tests/local-e2e/rnaseq`. |
| `tests/cli-valid/methylseq_live_test.go` `TestMethylSeqCLIRun` | `tests/local-e2e` `TestMethylSeqCLIRecover` | Compose/validate/plan; run stdout `{"op":"run"}`; no `in/Bisulfite_Genome`; remaining empty. Extended with occupy, release, resume, reuse, `--sample`, per-sample extractor outputs, MultiQC html. Pipe moved to `tests/local-e2e/methylseq`. |

Omitted `--sample` cwd `samplesheet.csv` is `TestRNASeqCLIOmittedSampleReadsCwdSheet` (hermetic compose).

GSE110004 fallback fired: reused-FASTQ DESeq2 exited with all genes equal across samples. Live RNA sheet and pins use four distinct pairs SRR6357070–SRR6357073.
