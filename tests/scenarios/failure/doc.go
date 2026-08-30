// Package failure owns evidence that command, input, backend, and publication
// failures remain contained and expose the failed unit, logs, reusable work,
// blocked downstream work, and remaining state. Recovery uses Inspect,
// Release, and Resume where backend disposition is proved. It adds no fallback
// route, keep-going fan-in, retry verb, or artifact deletion. It does not own an
// assay graph or fixture.
package failure
