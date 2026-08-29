package gobble

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"runtime/debug"
	"strings"

	"github.com/HahyeonJeon/gobble/internal/engine"
)

const identityGobbleModule = "github.com/HahyeonJeon/gobble"

var exactTagPattern = regexp.MustCompile(`^v0\.[0-9]+\.[0-9]+$`)

// Identity names the Gobble and pipeline bytes allowed to occupy or recover a
// workspace.
type Identity struct {
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

// OccupyOption configures the identity used by Run, Resume, Inspect, or
// Release. Its fields are intentionally opaque.
type OccupyOption struct {
	identity *Identity
}

// WithIdentity supplies the install identity for an occupy or recovery call.
func WithIdentity(id Identity) OccupyOption {
	copy := id
	return OccupyOption{identity: &copy}
}

func parseOccupyOptions(op string, required bool, opts []OccupyOption) (*Identity, error) {
	var identity *Identity
	for _, opt := range opts {
		if opt.identity == nil {
			continue
		}
		if identity != nil {
			return nil, &Error{Op: op, Defects: []Defect{{
				Code:    DefectInvalidRequest,
				Unit:    "identity",
				Message: "repeated WithIdentity",
			}}}
		}
		copy := *opt.identity
		identity = &copy
	}
	if identity == nil {
		if required {
			return nil, &Error{Op: op, Defects: []Defect{{
				Code:    DefectInvalidRequest,
				Unit:    "identity",
				Message: "missing WithIdentity",
			}}}
		}
		return nil, nil
	}
	if err := publicError(op, engine.ValidateInstallIdentity(toEngineIdentity(identity))); err != nil {
		return nil, err
	}
	if err := confirmLinkedIdentity(op, identity); err != nil {
		return nil, err
	}
	return identity, nil
}

func toEngineIdentity(id *Identity) *engine.InstallIdentity {
	if id == nil {
		return nil
	}
	return &engine.InstallIdentity{
		GobbleModule:           id.GobbleModule,
		GobbleVersion:          id.GobbleVersion,
		GobbleVCSRevision:      id.GobbleVCSRevision,
		GobbleVCSModified:      id.GobbleVCSModified,
		GobbleSourceRealpath:   id.GobbleSourceRealpath,
		GobbleExecutableSHA256: id.GobbleExecutableSHA256,
		PipelineModule:         id.PipelineModule,
		PipelineImport:         id.PipelineImport,
		PipelineVersion:        id.PipelineVersion,
		PipelineVCSRevision:    id.PipelineVCSRevision,
		PipelineVCSModified:    id.PipelineVCSModified,
		PipelineSourceRealpath: id.PipelineSourceRealpath,
		GOOS:                   id.GOOS,
		GOARCH:                 id.GOARCH,
		InstallKind:            id.InstallKind,
		IdentityMode:           id.IdentityMode,
	}
}

type listedModule struct {
	Path    string        `json:"Path"`
	Version string        `json:"Version"`
	Main    bool          `json:"Main"`
	Dir     string        `json:"Dir"`
	Replace *listedModule `json:"Replace"`
}

type sourceIdentity struct {
	revision string
	modified bool
	realpath string
}

var identityListModule = listIdentityModule
var identitySource = readSourceIdentity
var identityReadBuildInfo = debug.ReadBuildInfo
var identityExecutableDigest = readExecutableDigest

// IdentityFromBuildInfo constructs a module install identity for the current
// process and the package import path that supplied Pipeline.
func IdentityFromBuildInfo(pipelineImport string) (Identity, error) {
	if strings.TrimSpace(pipelineImport) == "" {
		return Identity{}, identityConstructionError("empty pipeline import")
	}
	info, ok := identityReadBuildInfo()
	if !ok || info == nil || info.Main.Path == "" {
		return Identity{}, identityConstructionError("build info unavailable")
	}
	gobbleModule, err := identityListModule(identityGobbleModule)
	if err != nil {
		return Identity{}, identityConstructionError(err.Error())
	}
	if gobbleModule.Path != identityGobbleModule {
		return Identity{}, identityConstructionError("unexpected gobble module")
	}
	pipelineModule, err := identityListModule("")
	if err != nil {
		return Identity{}, identityConstructionError(err.Error())
	}
	mode, err := partitionIdentityMode(gobbleModule)
	if err != nil {
		return Identity{}, identityConstructionError(err.Error())
	}
	pipelineSource, err := identitySource(moduleSourceDir(pipelineModule))
	if err != nil {
		return Identity{}, identityConstructionError(err.Error())
	}
	id := Identity{
		GobbleModule:        gobbleModule.Path,
		GobbleVersion:       moduleVersion(gobbleModule),
		PipelineModule:      info.Main.Path,
		PipelineImport:      pipelineImport,
		PipelineVersion:     buildModuleVersion(info.Main.Version, pipelineModule),
		PipelineVCSRevision: pipelineSource.revision,
		PipelineVCSModified: pipelineSource.modified,
		GOOS:                runtime.GOOS,
		GOARCH:              runtime.GOARCH,
		InstallKind:         "module",
		IdentityMode:        mode,
	}
	if pipelineSource.modified {
		id.PipelineSourceRealpath = pipelineSource.realpath
	}
	if mode == "local-pin" {
		gobbleSource, err := identitySource(moduleSourceDir(gobbleModule))
		if err != nil {
			return Identity{}, identityConstructionError(err.Error())
		}
		id.GobbleVCSRevision = gobbleSource.revision
		id.GobbleVCSModified = gobbleSource.modified
		if gobbleSource.modified {
			id.GobbleSourceRealpath = gobbleSource.realpath
		}
		id.GobbleExecutableSHA256, err = identityExecutableDigest()
		if err != nil {
			return Identity{}, identityConstructionError(err.Error())
		}
	}
	if defects := engine.ValidateInstallIdentity(toEngineIdentity(&id)); len(defects) > 0 {
		return Identity{}, publicError("identity", defects)
	}
	return id, nil
}

func identityConstructionError(message string) error {
	return &Error{Op: "identity", Defects: []Defect{{
		Code:    DefectInvalidRequest,
		Unit:    "identity",
		Message: message,
	}}}
}

func listIdentityModule(path string) (listedModule, error) {
	goBin, err := exec.LookPath("go")
	if err != nil {
		return listedModule{}, err
	}
	args := []string{"list", "-m", "-json"}
	if path != "" {
		args = append(args, path)
	}
	cmd := exec.Command(goBin, args...)
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && len(bytes.TrimSpace(exitErr.Stderr)) > 0 {
			return listedModule{}, errors.New(strings.TrimSpace(string(exitErr.Stderr)))
		}
		return listedModule{}, err
	}
	var module listedModule
	if err := json.Unmarshal(out, &module); err != nil {
		return listedModule{}, err
	}
	return module, nil
}

func partitionIdentityMode(module listedModule) (string, error) {
	version := moduleVersion(module)
	if module.Replace != nil || version == "(devel)" || version == "v0.0.0" {
		return "local-pin", nil
	}
	if module.Replace == nil && version != "v0.0.0" && exactTagPattern.MatchString(version) {
		return "exact-tag", nil
	}
	return "", fmt.Errorf("unsupported gobble module version %q", version)
}

func moduleVersion(module listedModule) string {
	if module.Version == "" && module.Main {
		return "(devel)"
	}
	return module.Version
}

func buildModuleVersion(version string, module listedModule) string {
	if version != "" {
		return version
	}
	return moduleVersion(module)
}

func moduleSourceDir(module listedModule) string {
	if module.Replace != nil {
		return module.Replace.Dir
	}
	return module.Dir
}

func readSourceIdentity(dir string) (sourceIdentity, error) {
	if dir == "" {
		return sourceIdentity{}, errors.New("empty module source directory")
	}
	realpath, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return sourceIdentity{}, err
	}
	realpath = filepath.Clean(realpath)
	if realpath == "" {
		return sourceIdentity{}, errors.New("empty module source realpath")
	}
	revision, err := runIdentityGit(realpath, "rev-parse", "HEAD")
	if err != nil {
		return sourceIdentity{}, err
	}
	if revision == "" {
		return sourceIdentity{}, errors.New("empty module revision")
	}
	status, err := runIdentityGit(realpath, "status", "--porcelain")
	if err != nil {
		return sourceIdentity{}, err
	}
	return sourceIdentity{revision: revision, modified: status != "", realpath: realpath}, nil
}

func runIdentityGit(dir string, args ...string) (string, error) {
	gitBin, err := exec.LookPath("git")
	if err != nil {
		return "", err
	}
	cmd := exec.Command(gitBin, append([]string{"-C", dir}, args...)...)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func readExecutableDigest() (string, error) {
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

func confirmLinkedIdentity(op string, required *Identity) error {
	info, ok := identityReadBuildInfo()
	if !ok || info == nil {
		return linkedIdentityError(op, *required, Identity{})
	}
	if info.Main.Path == identityGobbleModule {
		return nil
	}
	for _, dep := range info.Deps {
		if dep != nil && dep.Path == identityGobbleModule {
			have := Identity{GobbleModule: dep.Path, GobbleVersion: dep.Version}
			if dep.Path == required.GobbleModule && dep.Version == required.GobbleVersion {
				return nil
			}
			return linkedIdentityError(op, *required, have)
		}
	}
	return linkedIdentityError(op, *required, Identity{})
}

func linkedIdentityError(op string, required, have Identity) error {
	requiredJSON, _ := json.Marshal(required)
	haveJSON, _ := json.Marshal(have)
	return &Error{Op: op, Defects: []Defect{{
		Code:    DefectIdentityMismatch,
		Unit:    "gobble",
		Message: "required " + string(requiredJSON) + "; have " + string(haveJSON),
	}}}
}
