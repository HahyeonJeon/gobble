# WGS joint-germline testdata

`manifest.json` is the sole WGS fixture authority. It transcribes the exact
Planning-bound 24-entry byte inventory without discovering or substituting hashes:
16 staged nf-core/test-datasets bytes, five Sarek benchmark selectors, and
three license/provenance records. Every URL contains the accepted commit. The
manifest also records F and J coverage scenarios, the complete selected-stage
trace, and immutable linux/amd64 image facts.

`wgs-samplesheet.csv` localizes Sarek's multi-lane shape as two typed germline
samples. `testN` has two declared lanes over the official `test` pair. `testT`
uses the official tumor-derived `test2` pair only as engineering data. The
sheet never turns that lineage into a biological or clinical claim.

Live test-only preparation downloads staged entries into `cache/`, verifies
their declared size and SHA-256, and copies them into a caller-created
workspace. It splits the exact two-line `WG-INTERVALS` byte into stable
`interval_001.bed` and `interval_002.bed` Scatter members. Product source and
`Build` never fetch, inspect, or stage fixture bytes. The ignored cache is not
committed or redistributed.
