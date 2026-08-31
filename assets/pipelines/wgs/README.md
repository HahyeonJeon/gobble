# WGS joint-germline product

`assets/pipelines/wgs` owns Gobble graph generation
`wgs-joint-germline-v1`, benchmarked against nf-core/sarek 3.10.0. Pre-lift
two-sample alignment/QC workspaces are not resumable with this generation. The
temporary `assets.WGS` constructor now points here.

## Typed entry points

- `Parse` and `Load` accept exact columns
  `patient,sample,lane,fastq_1,fastq_2` plus optional `sex`. Unknown columns
  are errors. Repeated rows become ordered paired `Lane` values. At least two
  distinct patient/sample pairs are required. A sample token may occur under
  different patients. Patient, sample, and lane identity comes from typed
  cells, never filenames. When present, `sex` is experiment-data identity. It
  is recorded in sample-task params, input names, read groups, cohort identity,
  and cohort-scoped work destinations.
- `DefaultConfig` returns fresh typed reference, known-site, stable interval,
  output, command, immutable image, resource, argv, and publication policy. It
  selects BWA-MEM, GATK HaplotypeCaller gVCFs, GenomicsDB, and
  GenotypeGVCFs. It exposes no caller selector or VQSR resources.
- `Build(samples, config)` copies caller data and reads no process state,
  filesystem content, current working directory, or network location. Invalid
  cohorts, reference members, intervals, paths, images, and protected option
  prefixes or short aliases are structured compose defects.
- `Pipeline` is only the process-exclusive CLI adapter. It loads the injected
  sheet path and delegates to `Build` with fresh defaults.

## Selected path and outputs

The graph prepares a truthful BWA sidecar Group, runs raw FastQC and paired
FastP per lane, performs read-grouped BWA-MEM, lane sorting/indexing, sample
merge, GATK duplicate marking, interval BQSR with samtools BAM gather, alignment
QC, interval HaplotypeCaller and per-sample gVCF gather, complete-cohort
GenomicsDB import,
interval GenotypeGVCFs and sort, final VCF gather, callset statistics, and
MultiQC.

The exact interval Group drives native Gobble Scatter/Gather for BQSR,
HaplotypeCaller, GenomicsDB import, and joint genotyping. Every gathered
command names every expected interval path. Each GenomicsDB Scatter member
derives one Tree below `work/joint/cohort-<cohort-sha256>/genomicsdb` and passes
that exact member Tree to the matching joint-genotype instance. The digest is
over the ordered cohort identity recorded in task params. A missing sample,
interval, index, Tree manifest, or gathered member fails closed.

Required results are indexed recalibrated BAMs and per-sample gVCFs below
`results/wgs/samples/<patient>/<sample>/`, indexed unfiltered
`results/wgs/joint/joint_germline.vcf.gz`, command metrics, callset statistics,
and MultiQC HTML/data. Prepared reads and a generated BWA index are optional
publication categories. Required categories cannot be disabled.

## Lifecycle and support boundary

A changed patient, sample, present sex, lane, or read invalidates that sample's
consuming branch, gVCF, and all cohort work. Unchanged reference preparation and
unrelated sample preprocessing stay eligible for reuse. A caller-option change
affects its interval command and cohort descendants. Any sample-membership or
interval-membership change invalidates the applicable strict gathers and joint
result. Stop, failure, Inspect, Release, and Resume use the shared engine
contract; this product adds no retry, fallback, repair, cleanup, or migration
verb.

The final callset is unfiltered engineering output. The product does not claim
variant quality, study suitability, clinical validity, filtering, annotation,
or nf-core endorsement. It does not support somatic or tumor-normal calling,
VQSR, WES, structural variants, copy-number analysis, alternate callers, UMI,
or GPU routes. The official small data and tumor-derived `test2` lineage are
used only to prove tool, graph, artifact, provenance, and recovery behavior.
