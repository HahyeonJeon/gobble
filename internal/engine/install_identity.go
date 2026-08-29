package engine

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
)

const installGobbleModule = "github.com/HahyeonJeon/gobble"

// InstallIdentity records the Gobble and pipeline bytes allowed to occupy or
// recover a workspace.
type InstallIdentity struct {
	GobbleModule           string `json:"gobble_module"`
	GobbleVersion          string `json:"gobble_version"`
	GobbleVCSRevision      string `json:"gobble_vcs_revision"`
	GobbleVCSModified      bool   `json:"gobble_vcs_modified"`
	GobbleSourceRealpath   string `json:"gobble_source_realpath"`
	GobbleExecutableSHA256 string `json:"gobble_executable_sha256"`
	PipelineModule         string `json:"pipeline_module"`
	PipelineImport         string `json:"pipeline_import"`
	PipelineVersion        string `json:"pipeline_version"`
	PipelineVCSRevision    string `json:"pipeline_vcs_revision"`
	PipelineVCSModified    bool   `json:"pipeline_vcs_modified"`
	PipelineSourceRealpath string `json:"pipeline_source_realpath"`
	GOOS                   string `json:"goos"`
	GOARCH                 string `json:"goarch"`
	InstallKind            string `json:"install_kind"`
	IdentityMode           string `json:"identity_mode"`
}

type identityVerb string

const (
	identityInspect identityVerb = "inspect"
	identityRelease identityVerb = "release"
	identityResume  identityVerb = "resume"
)

var installReadBuildInfo = debug.ReadBuildInfo
var installExecutableDigest = executableDigest

// ValidateInstallIdentity checks the complete occupy identity contract.
func ValidateInstallIdentity(id *InstallIdentity) []Defect {
	if id == nil {
		return invalidInstallIdentity("missing WithIdentity")
	}
	if id.GobbleModule != installGobbleModule {
		return invalidInstallIdentity("invalid gobble_module")
	}
	if id.GobbleVersion == "" {
		return invalidInstallIdentity("empty gobble_version")
	}
	if id.PipelineModule == "" || id.PipelineImport == "" || id.PipelineVersion == "" {
		return invalidInstallIdentity("incomplete pipeline identity")
	}
	if id.GOOS != "linux" || id.GOARCH != "amd64" {
		return invalidInstallIdentity("unsupported goos/goarch")
	}
	if id.InstallKind != "module" && id.InstallKind != "packed-runner" {
		return invalidInstallIdentity("invalid install_kind")
	}
	if id.IdentityMode != "exact-tag" && id.IdentityMode != "local-pin" {
		return invalidInstallIdentity("invalid identity_mode")
	}
	if (id.GobbleSourceRealpath != "") != id.GobbleVCSModified {
		return invalidInstallIdentity("invalid gobble_source_realpath")
	}
	if (id.PipelineSourceRealpath != "") != id.PipelineVCSModified {
		return invalidInstallIdentity("invalid pipeline_source_realpath")
	}
	if id.PipelineVCSRevision == "" {
		return invalidInstallIdentity("empty pipeline_vcs_revision")
	}
	if id.IdentityMode == "exact-tag" {
		if id.GobbleVCSRevision != "" || id.GobbleVCSModified || id.GobbleSourceRealpath != "" || id.GobbleExecutableSHA256 != "" {
			return invalidInstallIdentity("invalid exact-tag gobble identity")
		}
		return nil
	}
	if id.GobbleVCSRevision == "" {
		return invalidInstallIdentity("empty gobble_vcs_revision")
	}
	if id.InstallKind == "module" {
		if !validExecutableDigest(id.GobbleExecutableSHA256) {
			return invalidInstallIdentity("invalid gobble_executable_sha256")
		}
	} else if id.GobbleExecutableSHA256 != "" {
		return invalidInstallIdentity("packed-runner has gobble_executable_sha256")
	}
	return nil
}

func invalidInstallIdentity(message string) []Defect {
	return []Defect{{Code: DefectInvalidRequest, Unit: "identity", Message: message}}
}

func validExecutableDigest(s string) bool {
	if len(s) != sha256.Size*2 || strings.ToLower(s) != s {
		return false
	}
	_, err := hex.DecodeString(s)
	return err == nil
}

func cloneInstallIdentity(id *InstallIdentity) *InstallIdentity {
	if id == nil {
		return nil
	}
	out := *id
	return &out
}

func workspaceIdentityDefects(required, have *InstallIdentity, verb identityVerb) []Defect {
	match, unit := matchInstallIdentity(required, have, verb)
	if match {
		return nil
	}
	return []Defect{{
		Code:    DefectIdentityMismatch,
		Unit:    unit,
		Message: identityMismatchMessage(required, have),
		Paths:   []string{ControlDir + "/" + RunIdentityFile},
	}}
}

func matchInstallIdentity(required, have *InstallIdentity, verb identityVerb) (bool, string) {
	if required == nil || have == nil {
		return false, "identity"
	}
	if required.InstallKind != have.InstallKind {
		return false, "install-kind"
	}
	if required.IdentityMode != have.IdentityMode {
		return false, "identity-mode"
	}
	if required.GOOS != have.GOOS || required.GOARCH != have.GOARCH {
		return false, "gobble"
	}
	if required.InstallKind == "module" {
		if !matchModuleGobble(required, have, verb) {
			return false, "gobble"
		}
		if verb == identityResume && (required.PipelineModule != have.PipelineModule || required.PipelineImport != have.PipelineImport) {
			return false, "pipeline"
		}
		return true, ""
	}
	if !matchPackedGobble(required, have) {
		return false, "gobble"
	}
	if !matchPackedPipeline(required, have) {
		return false, "pipeline"
	}
	return true, ""
}

func matchModuleGobble(required, have *InstallIdentity, verb identityVerb) bool {
	if required.GobbleModule != have.GobbleModule {
		return false
	}
	if required.IdentityMode == "exact-tag" {
		return required.GobbleVersion == have.GobbleVersion
	}
	if required.GobbleExecutableSHA256 != have.GobbleExecutableSHA256 {
		return false
	}
	if verb != identityResume {
		return true
	}
	return required.GobbleVersion == have.GobbleVersion &&
		required.GobbleVCSRevision == have.GobbleVCSRevision &&
		required.GobbleVCSModified == have.GobbleVCSModified &&
		(!required.GobbleVCSModified || required.GobbleSourceRealpath == have.GobbleSourceRealpath)
}

func matchPackedGobble(required, have *InstallIdentity) bool {
	if required.GobbleModule != have.GobbleModule || required.GobbleVersion != have.GobbleVersion {
		return false
	}
	if required.IdentityMode == "exact-tag" {
		return true
	}
	return required.GobbleVCSRevision == have.GobbleVCSRevision &&
		required.GobbleVCSModified == have.GobbleVCSModified &&
		(!required.GobbleVCSModified || required.GobbleSourceRealpath == have.GobbleSourceRealpath)
}

func matchPackedPipeline(required, have *InstallIdentity) bool {
	return required.PipelineModule == have.PipelineModule &&
		required.PipelineImport == have.PipelineImport &&
		required.PipelineVersion == have.PipelineVersion &&
		required.PipelineVCSRevision == have.PipelineVCSRevision &&
		required.PipelineVCSModified == have.PipelineVCSModified &&
		(!required.PipelineVCSModified || required.PipelineSourceRealpath == have.PipelineSourceRealpath)
}

func identityMismatchMessage(required, have *InstallIdentity) string {
	return "required " + identityJSON(required) + "; have " + identityJSON(have)
}

func identityJSON(id *InstallIdentity) string {
	data, err := json.Marshal(id)
	if err != nil {
		return "null"
	}
	return string(data)
}

func installIdentityForWorkspace(required, supplied *InstallIdentity) *InstallIdentity {
	if supplied != nil {
		return cloneInstallIdentity(supplied)
	}
	return processInstallIdentity(required)
}

func processInstallIdentity(required *InstallIdentity) *InstallIdentity {
	have := &InstallIdentity{GOOS: runtime.GOOS, GOARCH: runtime.GOARCH, InstallKind: "module"}
	if required != nil {
		have.IdentityMode = required.IdentityMode
	}
	if info, ok := installReadBuildInfo(); ok && info != nil {
		have.GobbleModule = info.Main.Path
		if have.IdentityMode == "exact-tag" {
			have.GobbleVersion = info.Main.Version
		}
	}
	if have.IdentityMode == "local-pin" {
		if digest, err := installExecutableDigest(); err == nil {
			have.GobbleExecutableSHA256 = digest
		}
	}
	return have
}

func executableDigest() (string, error) {
	path, err := os.Executable()
	if err != nil {
		return "", err
	}
	path, err = filepath.EvalSymlinks(path)
	if err != nil {
		return "", err
	}
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

type inspectIdentityDoc struct {
	SchemaVersion int              `json:"schema_version"`
	View          string           `json:"view"`
	Match         bool             `json:"match"`
	Required      *InstallIdentity `json:"required"`
	Have          *InstallIdentity `json:"have"`
}
