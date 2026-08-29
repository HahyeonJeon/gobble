package gobble

import (
	"runtime/debug"
	"testing"
)

func TestOccupyOptions(t *testing.T) {
	id := validPublicIdentity()
	got, err := parseOccupyOptions("run", true, []OccupyOption{{}, WithIdentity(id), {}})
	if err != nil || got == nil || *got != id {
		t.Fatalf("parseOccupyOptions() = %#v, %v, want identity", got, err)
	}
	if _, err := parseOccupyOptions("run", true, nil); !isPublicDefect(err, DefectInvalidRequest, "identity") {
		t.Fatalf("missing Run identity error = %v, want invalid-request", err)
	}
	if got, err := parseOccupyOptions("inspect", false, []OccupyOption{{}}); err != nil || got != nil {
		t.Fatalf("omitted Inspect identity = %#v, %v, want nil", got, err)
	}
	if _, err := parseOccupyOptions("run", true, []OccupyOption{WithIdentity(id), WithIdentity(id)}); !isPublicDefect(err, DefectInvalidRequest, "identity") {
		t.Fatalf("repeated identity error = %v, want invalid-request", err)
	}
}

func TestIdentityFromBuildInfoCurrentModule(t *testing.T) {
	id, err := IdentityFromBuildInfo(identityGobbleModule)
	if err != nil {
		t.Fatalf("IdentityFromBuildInfo() error = %v", err)
	}
	if id.PipelineImport != identityGobbleModule {
		t.Fatalf("pipeline_import = %q, want %q", id.PipelineImport, identityGobbleModule)
	}
	if id.IdentityMode != "local-pin" || id.GobbleVCSRevision == "" || id.PipelineVCSRevision == "" || id.GobbleExecutableSHA256 == "" {
		t.Fatalf("IdentityFromBuildInfo() = %#v, want complete local-pin identity", id)
	}
}

func TestIdentityFromBuildInfoEmptyImport(t *testing.T) {
	if _, err := IdentityFromBuildInfo(""); !isPublicDefect(err, DefectInvalidRequest, "identity") {
		t.Fatalf("IdentityFromBuildInfo(empty) error = %v, want invalid-request", err)
	}
}

func TestIdentityFromBuildInfoExactTagSkipsGobbleSource(t *testing.T) {
	origList := identityListModule
	origSource := identitySource
	origBuildInfo := identityReadBuildInfo
	origDigest := identityExecutableDigest
	t.Cleanup(func() {
		identityListModule = origList
		identitySource = origSource
		identityReadBuildInfo = origBuildInfo
		identityExecutableDigest = origDigest
	})
	identityListModule = func(path string) (listedModule, error) {
		if path == identityGobbleModule {
			return listedModule{Path: identityGobbleModule, Version: "v0.1.0", Dir: "/must-not-read/module-cache/gobble@v0.1.0"}, nil
		}
		return listedModule{Path: "example.test/pipeline", Main: true, Dir: "/pipeline"}, nil
	}
	sourceCalls := 0
	identitySource = func(dir string) (sourceIdentity, error) {
		sourceCalls++
		if dir != "/pipeline" {
			t.Fatalf("identity source dir = %q, want pipeline only", dir)
		}
		return sourceIdentity{revision: "pipeline-revision"}, nil
	}
	identityReadBuildInfo = func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{Main: debug.Module{Path: "example.test/pipeline", Version: "(devel)"}}, true
	}
	identityExecutableDigest = func() (string, error) {
		t.Fatal("exact-tag identity called executable digest")
		return "", nil
	}
	id, err := IdentityFromBuildInfo("example.test/pipeline")
	if err != nil {
		t.Fatalf("IdentityFromBuildInfo(exact-tag) error = %v", err)
	}
	if id.IdentityMode != "exact-tag" || id.GobbleVersion != "v0.1.0" || id.GobbleVCSRevision != "" || id.GobbleExecutableSHA256 != "" {
		t.Fatalf("exact-tag identity = %#v", id)
	}
	if sourceCalls != 1 {
		t.Fatalf("source calls = %d, want pipeline only", sourceCalls)
	}
}

func TestWithIdentityRejectsLinkedGobbleMismatch(t *testing.T) {
	origBuildInfo := identityReadBuildInfo
	t.Cleanup(func() { identityReadBuildInfo = origBuildInfo })
	identityReadBuildInfo = func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{
			Main: debug.Module{Path: "example.test/pipeline", Version: "(devel)"},
			Deps: []*debug.Module{{Path: identityGobbleModule, Version: "v0.2.0"}},
		}, true
	}
	id := validPublicIdentity()
	id.GobbleVersion = "v0.1.0"
	if _, err := parseOccupyOptions("run", true, []OccupyOption{WithIdentity(id)}); !isPublicDefect(err, DefectIdentityMismatch, "gobble") {
		t.Fatalf("linked mismatch error = %v, want identity-mismatch", err)
	}
}

func validPublicIdentity() Identity {
	return Identity{
		GobbleModule:           identityGobbleModule,
		GobbleVersion:          "(devel)",
		GobbleVCSRevision:      "gobble-revision",
		GobbleExecutableSHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		PipelineModule:         identityGobbleModule,
		PipelineImport:         identityGobbleModule,
		PipelineVersion:        "(devel)",
		PipelineVCSRevision:    "pipeline-revision",
		GOOS:                   "linux",
		GOARCH:                 "amd64",
		InstallKind:            "module",
		IdentityMode:           "local-pin",
	}
}

func isPublicDefect(err error, code DefectCode, unit string) bool {
	ge, ok := err.(*Error)
	if !ok {
		return false
	}
	for _, defect := range ge.Defects {
		if defect.Code == code && defect.Unit == unit {
			return true
		}
	}
	return false
}
