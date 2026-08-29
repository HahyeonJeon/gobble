package engine

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"testing"
)

func testInstallIdentity() *InstallIdentity {
	return &InstallIdentity{
		GobbleModule:           installGobbleModule,
		GobbleVersion:          "(devel)",
		GobbleVCSRevision:      "gobble-revision",
		GobbleVCSModified:      true,
		GobbleSourceRealpath:   "/source/gobble",
		GobbleExecutableSHA256: strings.Repeat("a", 64),
		PipelineModule:         installGobbleModule,
		PipelineImport:         installGobbleModule + "/internal/engine",
		PipelineVersion:        "(devel)",
		PipelineVCSRevision:    "pipeline-revision",
		PipelineVCSModified:    true,
		PipelineSourceRealpath: "/source/pipeline",
		GOOS:                   "linux",
		GOARCH:                 "amd64",
		InstallKind:            "module",
		IdentityMode:           "local-pin",
	}
}

func TestValidateInstallIdentityModes(t *testing.T) {
	local := testInstallIdentity()
	if defects := ValidateInstallIdentity(local); len(defects) != 0 {
		t.Fatalf("ValidateInstallIdentity(local) = %v, want none", defects)
	}
	exact := *local
	exact.GobbleVersion = "v0.1.0"
	exact.GobbleVCSRevision = ""
	exact.GobbleVCSModified = false
	exact.GobbleSourceRealpath = ""
	exact.GobbleExecutableSHA256 = ""
	exact.IdentityMode = "exact-tag"
	if defects := ValidateInstallIdentity(&exact); len(defects) != 0 {
		t.Fatalf("ValidateInstallIdentity(exact) = %v, want none", defects)
	}
	packed := *local
	packed.InstallKind = "packed-runner"
	packed.GobbleExecutableSHA256 = ""
	if defects := ValidateInstallIdentity(&packed); len(defects) != 0 {
		t.Fatalf("ValidateInstallIdentity(packed) = %v, want none", defects)
	}
	local.GobbleVCSRevision = ""
	if defects := ValidateInstallIdentity(local); !hasDefect(defects, DefectInvalidRequest, "identity") {
		t.Fatalf("ValidateInstallIdentity(empty revision) = %v, want invalid-request", defects)
	}
}

func TestIdentityMatchRejectsCrossFamilyWithEqualReducedFields(t *testing.T) {
	required := testInstallIdentity()
	required.GobbleVersion = "v0.1.0"
	have := *required
	have.IdentityMode = "exact-tag"
	have.GobbleVCSRevision = ""
	have.GobbleVCSModified = false
	have.GobbleSourceRealpath = ""
	have.GobbleExecutableSHA256 = ""
	match, unit := matchInstallIdentity(required, &have, identityInspect)
	if match || unit != "identity-mode" {
		t.Fatalf("matchInstallIdentity() = %v, %q, want false identity-mode", match, unit)
	}
}

func TestIdentityMatchTables(t *testing.T) {
	required := testInstallIdentity()

	inspectHave := *required
	inspectHave.GobbleVCSRevision = "other"
	inspectHave.GobbleVCSModified = false
	inspectHave.GobbleSourceRealpath = ""
	inspectHave.PipelineModule = "other/module"
	inspectHave.PipelineImport = "other/import"
	if match, unit := matchInstallIdentity(required, &inspectHave, identityInspect); !match {
		t.Fatalf("local-pin Inspect match = false %q, want ignored git and pipeline", unit)
	}
	if match, unit := matchInstallIdentity(required, &inspectHave, identityResume); match || unit != "gobble" {
		t.Fatalf("local-pin Resume git match = %v %q, want false gobble", match, unit)
	}

	resumeHave := *required
	resumeHave.PipelineVCSRevision = "changed"
	resumeHave.PipelineVCSModified = false
	resumeHave.PipelineSourceRealpath = ""
	if match, unit := matchInstallIdentity(required, &resumeHave, identityResume); !match {
		t.Fatalf("module Resume pipeline VCS match = false %q, want ignored", unit)
	}
	resumeHave.PipelineImport = "other/import"
	if match, unit := matchInstallIdentity(required, &resumeHave, identityResume); match || unit != "pipeline" {
		t.Fatalf("module Resume import match = %v %q, want false pipeline", match, unit)
	}

	dirtyHave := *required
	dirtyHave.GobbleSourceRealpath = "/source/other"
	if match, unit := matchInstallIdentity(required, &dirtyHave, identityResume); match || unit != "gobble" {
		t.Fatalf("dirty realpath match = %v %q, want false gobble", match, unit)
	}

	packed := *required
	packed.InstallKind = "packed-runner"
	packed.GobbleExecutableSHA256 = ""
	packedHave := packed
	packedHave.PipelineVCSRevision = "other"
	if match, unit := matchInstallIdentity(&packed, &packedHave, identityInspect); match || unit != "pipeline" {
		t.Fatalf("packed pipeline match = %v %q, want false pipeline", match, unit)
	}
}

func TestProcessIdentityExactTagSkipsExecutableDigest(t *testing.T) {
	origBuildInfo := installReadBuildInfo
	origDigest := installExecutableDigest
	t.Cleanup(func() {
		installReadBuildInfo = origBuildInfo
		installExecutableDigest = origDigest
	})
	installReadBuildInfo = func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{Main: debug.Module{Path: installGobbleModule, Version: "v0.1.0"}}, true
	}
	installExecutableDigest = func() (string, error) {
		t.Fatal("exact-tag process identity called executable digest")
		return "", errors.New("unreachable")
	}
	required := testInstallIdentity()
	required.IdentityMode = "exact-tag"
	have := processInstallIdentity(required)
	if have.GobbleModule != installGobbleModule || have.GobbleVersion != "v0.1.0" || have.GobbleExecutableSHA256 != "" {
		t.Fatalf("processInstallIdentity(exact) = %#v", have)
	}
}

func TestProcessIdentityLocalPinRejectsDifferentExecutable(t *testing.T) {
	origBuildInfo := installReadBuildInfo
	origDigest := installExecutableDigest
	t.Cleanup(func() {
		installReadBuildInfo = origBuildInfo
		installExecutableDigest = origDigest
	})
	installReadBuildInfo = func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{Main: debug.Module{Path: installGobbleModule, Version: "(devel)"}}, true
	}
	installExecutableDigest = func() (string, error) { return strings.Repeat("b", 64), nil }
	required := testInstallIdentity()
	have := processInstallIdentity(required)
	if match, unit := matchInstallIdentity(required, have, identityInspect); match || unit != "gobble" {
		t.Fatalf("different executable match = %v %q, want false gobble", match, unit)
	}
}

func TestInspectIdentityMissingAndMismatch(t *testing.T) {
	t.Run("missing identity", func(t *testing.T) {
		dir := t.TempDir()
		run := jsonRun{
			SchemaVersion: SchemaVersion,
			ID:            "run-1",
			Status:        StatusRunning,
			Started:       "2026-01-01T00:00:00Z",
			Occupancy:     &jsonOccupancy{Active: true, Lease: "lease"},
		}
		data, err := json.Marshal(run)
		if err != nil {
			t.Fatal(err)
		}
		writeCheckFile(t, filepath.Join(dir, ControlDir, RunIdentityFile), string(data))
		raw, defects := Inspect(dir, viewIdentity, "", testInstallIdentity())
		if len(defects) != 0 {
			t.Fatalf("Inspect(identity) defects = %v, want none", defects)
		}
		var header inspectIdentityDoc
		if err := json.Unmarshal(raw, &header); err != nil {
			t.Fatalf("Inspect(identity) JSON: %v", err)
		}
		if header.SchemaVersion != SchemaVersion || header.View != viewIdentity || header.Match || header.Required != nil {
			t.Fatalf("Inspect(identity) header = %#v, want missing false match", header)
		}
		if _, defects := Inspect(dir, viewRun, "", testInstallIdentity()); !hasDefect(defects, DefectIdentityMismatch, "identity") {
			t.Fatalf("Inspect(run) defects = %v, want identity-mismatch", defects)
		}
	})

	t.Run("mismatched digest", func(t *testing.T) {
		dir := t.TempDir()
		writeCheckFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
		req := Request{
			Identity:  testInstallIdentity(),
			Workspace: dir,
			Document:  sampleDoc("", "", "in/sample.txt", "out/sample.txt"),
		}
		if defects := Run(t.Context(), req); len(defects) != 0 {
			t.Fatalf("Run() defects = %v", defects)
		}
		wrong := testInstallIdentity()
		wrong.GobbleExecutableSHA256 = strings.Repeat("b", 64)
		before := snapshotDir(t, dir)
		raw, defects := Inspect(dir, viewIdentity, "", wrong)
		if len(defects) != 0 {
			t.Fatalf("Inspect(identity) defects = %v, want none", defects)
		}
		var header inspectIdentityDoc
		if err := json.Unmarshal(raw, &header); err != nil || header.Match {
			t.Fatalf("Inspect(identity) header = %#v error=%v, want false match", header, err)
		}
		if _, defects := Inspect(dir, viewRun, "", wrong); !hasDefect(defects, DefectIdentityMismatch, "gobble") {
			t.Fatalf("Inspect(run) defects = %v, want identity-mismatch", defects)
		}
		if defects := Release(dir, wrong); !hasDefect(defects, DefectIdentityMismatch, "gobble") {
			t.Fatalf("Release() defects = %v, want identity-mismatch", defects)
		}
		if after := snapshotDir(t, dir); after != before {
			t.Fatal("identity mismatch mutated workspace")
		}
		if _, err := os.Stat(filepath.Join(dir, ControlDir, RunIdentityFile)); err != nil {
			t.Fatalf("run identity disappeared: %v", err)
		}
	})
}

func TestResumeIdentityMismatchDoesNotOccupy(t *testing.T) {
	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
	req := Request{
		Identity:  testInstallIdentity(),
		Workspace: dir,
		Document:  sampleDoc("", "", "in/sample.txt", "out/sample.txt"),
	}
	if defects := Run(t.Context(), req); len(defects) != 0 {
		t.Fatalf("Run() defects = %v", defects)
	}
	if defects := Release(dir, testInstallIdentity()); len(defects) != 0 {
		t.Fatalf("Release() defects = %v", defects)
	}
	wrong := testInstallIdentity()
	wrong.PipelineImport = "example.test/other"
	before := snapshotDir(t, dir)
	if defects := Resume(t.Context(), Request{
		Identity:  wrong,
		Workspace: dir,
		Document:  req.Document,
	}); !hasDefect(defects, DefectIdentityMismatch, "pipeline") {
		t.Fatalf("Resume() defects = %v, want pipeline identity-mismatch", defects)
	}
	if after := snapshotDir(t, dir); after != before {
		t.Fatal("Resume identity mismatch mutated workspace")
	}
}
