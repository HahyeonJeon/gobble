package main

import (
	"errors"
	"runtime/debug"
	"strings"
	"testing"

	"github.com/HahyeonJeon/gobble"
)

func init() {
	installBuildInfo = func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{Main: debug.Module{Path: modulePath, Version: "(devel)"}}, true
	}
}

func TestPartitionVZeroWithReplaceIsLocalPin(t *testing.T) {
	mode, err := partitionInstallIdentity(installModule{
		Path:    modulePath,
		Version: "v0.0.0",
		Replace: &installModule{Path: "/source/gobble", Dir: "/source/gobble"},
	})
	if err != nil || mode != "local-pin" {
		t.Fatalf("partition = %q, %v, want local-pin", mode, err)
	}
}

func TestPartitionVZeroWithoutReplaceIsLocalPin(t *testing.T) {
	mode, err := partitionInstallIdentity(installModule{Path: modulePath, Version: "v0.0.0"})
	if err != nil || mode != "local-pin" {
		t.Fatalf("partition = %q, %v, want local-pin", mode, err)
	}
}

func TestParentHandshakeExactTagEqualSkipsGobbleGit(t *testing.T) {
	const cacheDir = "/must-not-read/module-cache/gobble@v0.1.0"
	sourceCalls := []string{}
	ops := installIdentityOps{
		listModule: func(path string) (installModule, error) {
			if path == modulePath {
				return installModule{Path: modulePath, Version: "v0.1.0", Dir: cacheDir}, nil
			}
			return installModule{Path: "example.test/pipeline", Main: true, Dir: "/pipeline"}, nil
		},
		source: func(dir string) (installSource, error) {
			sourceCalls = append(sourceCalls, dir)
			if dir == cacheDir {
				t.Fatalf("exact-tag handshake read Gobble module-cache dir")
			}
			return installSource{revision: "pipeline-revision"}, nil
		},
		buildInfo: func() (*debug.BuildInfo, bool) {
			return &debug.BuildInfo{Main: debug.Module{Path: modulePath, Version: "v0.1.0"}}, true
		},
		digest: func() (string, error) {
			t.Fatal("exact-tag handshake hashed executable")
			return "", nil
		},
	}
	result, err := buildInstallIdentity("example.test/pipeline", "run", ops)
	if err != nil {
		t.Fatalf("buildInstallIdentity() error = %v", err)
	}
	if result.Identity.IdentityMode != "exact-tag" || result.Identity.GobbleVCSRevision != "" || result.Identity.GobbleExecutableSHA256 != "" {
		t.Fatalf("exact-tag identity = %#v", result.Identity)
	}
	if len(sourceCalls) != 1 || sourceCalls[0] != "/pipeline" {
		t.Fatalf("source calls = %v, want pipeline only", sourceCalls)
	}
}

func TestParentHandshakeExactTagMismatchSkipsAllGit(t *testing.T) {
	listCalls := 0
	ops := installIdentityOps{
		listModule: func(path string) (installModule, error) {
			listCalls++
			return installModule{Path: modulePath, Version: "v0.2.0", Dir: "/must-not-read/module-cache/gobble@v0.2.0"}, nil
		},
		source: func(string) (installSource, error) {
			t.Fatal("exact-tag mismatch called git source")
			return installSource{}, nil
		},
		buildInfo: func() (*debug.BuildInfo, bool) {
			return &debug.BuildInfo{Main: debug.Module{Path: modulePath, Version: "v0.1.0"}}, true
		},
		digest: func() (string, error) {
			t.Fatal("exact-tag mismatch hashed executable")
			return "", nil
		},
	}
	_, err := buildInstallIdentity("example.test/pipeline", "run", ops)
	if !hasCLIIdentityDefect(err, gobble.DefectIdentityMismatch) {
		t.Fatalf("buildInstallIdentity() error = %v, want identity-mismatch", err)
	}
	if listCalls != 1 {
		t.Fatalf("list calls = %d, want selected Gobble only", listCalls)
	}
}

func TestParentHandshakeExactTagMismatchDoesNotFallThrough(t *testing.T) {
	_, err := buildInstallIdentity("example.test/pipeline", "run", installIdentityOps{
		listModule: func(string) (installModule, error) {
			return installModule{Path: modulePath, Version: "v0.2.0"}, nil
		},
		source: func(string) (installSource, error) {
			return installSource{}, errors.New("local-pin fallthrough")
		},
		buildInfo: func() (*debug.BuildInfo, bool) {
			return &debug.BuildInfo{Main: debug.Module{Path: modulePath, Version: "v0.1.0"}}, true
		},
		digest: func() (string, error) { return "", errors.New("local-pin fallthrough") },
	})
	if !hasCLIIdentityDefect(err, gobble.DefectIdentityMismatch) || strings.Contains(err.Error(), "fallthrough") {
		t.Fatalf("buildInstallIdentity() error = %v, want exact-tag mismatch without fallthrough", err)
	}
}

func TestParentHandshakeTaggedReplaceUsesLocalPin(t *testing.T) {
	sourceDirs := []string{}
	result, err := buildInstallIdentity("example.test/pipeline", "run", installIdentityOps{
		listModule: func(path string) (installModule, error) {
			if path == modulePath {
				return installModule{
					Path:    modulePath,
					Version: "v0.1.0",
					Replace: &installModule{Path: "/source/gobble", Dir: "/source/gobble"},
				}, nil
			}
			return installModule{Path: "example.test/pipeline", Main: true, Dir: "/pipeline"}, nil
		},
		source: func(dir string) (installSource, error) {
			sourceDirs = append(sourceDirs, dir)
			return installSource{revision: dir + "-revision"}, nil
		},
		buildInfo: func() (*debug.BuildInfo, bool) {
			return &debug.BuildInfo{Main: debug.Module{Path: modulePath, Version: "v0.9.0"}}, true
		},
		digest: func() (string, error) { return strings.Repeat("a", 64), nil },
	})
	if err != nil {
		t.Fatalf("buildInstallIdentity() error = %v", err)
	}
	if result.Identity.IdentityMode != "local-pin" || !result.HasReplace || result.ReplacePath != "/source/gobble" {
		t.Fatalf("tagged replace result = %#v", result)
	}
	if len(sourceDirs) != 2 || sourceDirs[0] != "/source/gobble" || sourceDirs[1] != "/pipeline" {
		t.Fatalf("source dirs = %v", sourceDirs)
	}
}

func TestParentHandshakeLocalPinIgnoresStampedVCSRevision(t *testing.T) {
	const sourceDir = "/source/gobble"
	const sourceRevision = "source-revision"
	digest := strings.Repeat("a", 64)
	sourceDirs := []string{}
	result, err := buildInstallIdentity("example.test/pipeline", "run", installIdentityOps{
		listModule: func(path string) (installModule, error) {
			if path == modulePath {
				return installModule{Path: modulePath, Main: true, Dir: sourceDir}, nil
			}
			return installModule{Path: "example.test/pipeline", Main: true, Dir: "/pipeline"}, nil
		},
		source: func(dir string) (installSource, error) {
			sourceDirs = append(sourceDirs, dir)
			if dir == sourceDir {
				return installSource{revision: sourceRevision, modified: true, realpath: sourceDir}, nil
			}
			return installSource{revision: "pipeline-revision"}, nil
		},
		buildInfo: func() (*debug.BuildInfo, bool) {
			return &debug.BuildInfo{
				Main: debug.Module{Path: modulePath, Version: "(devel)"},
				Settings: []debug.BuildSetting{{
					Key:   "vcs.revision",
					Value: "unrelated-stamped-revision",
				}},
			}, true
		},
		digest: func() (string, error) { return digest, nil },
	})
	if err != nil {
		t.Fatalf("buildInstallIdentity() error = %v", err)
	}
	if result.Identity.IdentityMode != "local-pin" || result.Identity.GobbleVCSRevision != sourceRevision || !result.Identity.GobbleVCSModified || result.Identity.GobbleSourceRealpath != sourceDir || result.Identity.GobbleExecutableSHA256 != digest {
		t.Fatalf("local-pin identity = %#v", result.Identity)
	}
	if len(sourceDirs) != 2 || sourceDirs[0] != sourceDir || sourceDirs[1] != "/pipeline" {
		t.Fatalf("source dirs = %v", sourceDirs)
	}
}

func TestParentHandshakeEmptyLocalRevisionFailsClosed(t *testing.T) {
	_, err := buildInstallIdentity("example.test/pipeline", "run", installIdentityOps{
		listModule: func(string) (installModule, error) {
			return installModule{Path: modulePath, Main: true, Dir: "/source/gobble"}, nil
		},
		source: func(string) (installSource, error) { return installSource{}, nil },
		buildInfo: func() (*debug.BuildInfo, bool) {
			return &debug.BuildInfo{Main: debug.Module{Path: modulePath, Version: "(devel)"}}, true
		},
		digest: func() (string, error) { return strings.Repeat("a", 64), nil },
	})
	if !hasCLIIdentityDefect(err, gobble.DefectInvalidRequest) || !strings.Contains(err.Error(), "empty gobble revision") {
		t.Fatalf("buildInstallIdentity() error = %v, want empty-revision invalid-request", err)
	}
}

func TestParentHandshakeEmptyLocalDigestFailsClosed(t *testing.T) {
	listCalls := 0
	_, err := buildInstallIdentity("example.test/pipeline", "run", installIdentityOps{
		listModule: func(string) (installModule, error) {
			listCalls++
			return installModule{Path: modulePath, Main: true, Dir: "/source/gobble"}, nil
		},
		source: func(string) (installSource, error) {
			return installSource{revision: "revision"}, nil
		},
		buildInfo: func() (*debug.BuildInfo, bool) {
			return &debug.BuildInfo{Main: debug.Module{Path: modulePath, Version: "(devel)"}}, true
		},
		digest: func() (string, error) { return "", nil },
	})
	if !hasCLIIdentityDefect(err, gobble.DefectInvalidRequest) || !strings.Contains(err.Error(), "invalid executable digest") {
		t.Fatalf("buildInstallIdentity() error = %v, want empty-digest invalid-request", err)
	}
	if listCalls != 1 {
		t.Fatalf("list calls = %d, want failure before pipeline construction", listCalls)
	}
}

func TestInternalImportPathDetection(t *testing.T) {
	for _, path := range []string{"example.test/internal/pipeline", "internal/pipeline", "example.test/a/internal"} {
		if !hasInternalImport(path) {
			t.Fatalf("hasInternalImport(%q) = false", path)
		}
	}
	for _, path := range []string{"example.test/internalized/pipeline", "example.test/pipeline"} {
		if hasInternalImport(path) {
			t.Fatalf("hasInternalImport(%q) = true", path)
		}
	}
}

func hasCLIIdentityDefect(err error, code gobble.DefectCode) bool {
	ge, ok := err.(*gobble.Error)
	if !ok {
		return false
	}
	for _, defect := range ge.Defects {
		if defect.Code == code {
			return true
		}
	}
	return false
}
