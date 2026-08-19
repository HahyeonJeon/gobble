# Go Mistakes

## Compose and Run must share one edge contract

**Context:** Related-file output `From` is legal at compose. Thin-slice BAI used it.

**Mistake:** `upstreamReady` only looked up inputs. Output-port `From` composed cleanly and deadlocked at run (`not-started`).

**Correction:** Resolve wait paths at `BuildPlan`. Run reads `Edge.Wait` only. Do not guess from `ToPort`. Add a process-level regression.

## WGS FASTQ is not RNA or bisulfite proof

**Context:** First-party RNA-seq and Methyl-seq live Runs need assay-shaped reads.

**Mistake:** Reusing WGS homo_sapiens FASTQ and FASTA as STAR or Bismark stand-ins proves graph shape only. Non-bisulfite WGS reads aligned 0.00% through Bismark.

**Correction:** Pin official nf-core rnaseq and methylseq test-profile files. Drop WGS stand-in from RNA and Methyl proofs.

## Basename-only pin cache collides

**Context:** Two pins can share `Pin.Name` (`test_1.fastq.gz` for WGS and SARS-CoV-2).

**Mistake:** Writing `CacheDir/<Name>` lets the second pin overwrite or reuse the first file.

**Correction:** Use `CacheDir/<sha256[:16]>/<Name>` and a shared fetch helper. Check size and sha256 after download.

## Bismark image swap needs a CLI study

**Context:** Gobble first used `quay.io/biocontainers/bismark:3.1.0`. Official methylseq 4.2.0 uses Seqera Bismark 0.25.1.

**Mistake:** Swapping the image without mapping argv. A rumor that Perl 0.25.1 wants `--output` instead of `--output_dir` is not evidence.

**Correction:** Study the 0.25.1 CLI (and live `--help` when available) before the swap. v0.25.1 documents `-o/--output_dir`. Keep `--basename aligned`. Do not combine `--basename` with `--multicore`. Re-run live after the image change.

## Do not classify from a record this operation will overwrite

**Context:** Resume occupy overwrites `plan.json`. Inspect remaining can take an instance filter. Dest replace needs an owner.

**Mistake:** Classifying resume identity from the plan after occupy overwrites it. Filtering latest attempts before remaining classification. Treating any executed identity as dest owner.

**Correction:** Persist script and env on the attempt, or classify before replacing `plan.json`. Classify remaining on all latest attempts, then instance-filter emit. Attribute dests by checksum or producer lineage, not executed identity.
