package moduleevidence

import "github.com/HahyeonJeon/gobble/assets/modules"

// ImagePin records the evidence that makes one module default image a
// supported platform tuple. Values are source-adjacent to that module; no
// repository-wide or assay manifest is a second pin authority.
type ImagePin struct {
	Image             modules.Image
	BenchmarkPipeline string
	BenchmarkRelease  string
	ModuleSource      string
	Command           string
	Version           string
	Provenance        string
	License           string
	LicenseSource     string
	GOOS              string
	GOARCH            string
}
