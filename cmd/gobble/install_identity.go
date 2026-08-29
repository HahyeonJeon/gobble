package main

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

	"github.com/HahyeonJeon/gobble"
)

var installExactTagPattern = regexp.MustCompile(`^v0\.[0-9]+\.[0-9]+$`)

type installModule struct {
	Path    string         `json:"Path"`
	Version string         `json:"Version"`
	Main    bool           `json:"Main"`
	Dir     string         `json:"Dir"`
	Replace *installModule `json:"Replace"`
}

type installSource struct {
	revision string
	modified bool
	realpath string
}

type installIdentityResult struct {
	Identity    gobble.Identity
	HasReplace  bool
	ReplacePath string
}

type installIdentityOps struct {
	listModule func(string) (installModule, error)
	source     func(string) (installSource, error)
	buildInfo  func() (*debug.BuildInfo, bool)
	digest     func() (string, error)
}

var installBuildInfo = debug.ReadBuildInfo

func resolveInstallIdentity(goBin, cwd, importPath, op string) (installIdentityResult, error) {
	ops := installIdentityOps{
		listModule: func(path string) (installModule, error) {
			return listInstallModule(goBin, cwd, path)
		},
		source:    readInstallSource,
		buildInfo: installBuildInfo,
		digest:    installExecutableSHA256,
	}
	return buildInstallIdentity(importPath, op, ops)
}

func buildInstallIdentity(importPath, op string, ops installIdentityOps) (installIdentityResult, error) {
	selected, err := ops.listModule(modulePath)
	if err != nil {
		return installIdentityResult{}, installInvalid(op, err.Error())
	}
	mode, err := partitionInstallIdentity(selected)
	if err != nil {
		return installIdentityResult{}, installInvalid(op, err.Error())
	}
	info, ok := ops.buildInfo()
	if !ok || info == nil {
		return installIdentityResult{}, installInvalid(op, "build info unavailable")
	}
	if selected.Path != modulePath || info.Main.Path != modulePath {
		return installIdentityResult{}, installMismatch(op, selected.Path, info.Main.Path)
	}
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		return installIdentityResult{}, installInvalid(op, "unsupported goos/goarch")
	}

	result := installIdentityResult{
		Identity: gobble.Identity{
			GobbleModule:  selected.Path,
			GobbleVersion: installModuleVersion(selected),
			GOOS:          runtime.GOOS,
			GOARCH:        runtime.GOARCH,
			InstallKind:   "module",
			IdentityMode:  mode,
		},
		HasReplace: selected.Replace != nil,
	}
	if selected.Replace != nil {
		result.ReplacePath = selected.Replace.Path
	}

	if mode == "exact-tag" {
		if info.Main.Version != selected.Version {
			return installIdentityResult{}, installMismatch(op, selected.Version, info.Main.Version)
		}
	} else {
		source, err := ops.source(installModuleSourceDir(selected))
		if err != nil {
			return installIdentityResult{}, installInvalid(op, err.Error())
		}
		if source.revision == "" {
			return installIdentityResult{}, installInvalid(op, "empty gobble revision")
		}
		result.Identity.GobbleVCSRevision = source.revision
		result.Identity.GobbleVCSModified = source.modified
		if source.modified {
			result.Identity.GobbleSourceRealpath = source.realpath
		}
		result.Identity.GobbleExecutableSHA256, err = ops.digest()
		if err != nil || !validInstallDigest(result.Identity.GobbleExecutableSHA256) {
			if err != nil {
				return installIdentityResult{}, installInvalid(op, err.Error())
			}
			return installIdentityResult{}, installInvalid(op, "invalid executable digest")
		}
	}

	pipeline, err := ops.listModule("")
	if err != nil {
		return installIdentityResult{}, installInvalid(op, err.Error())
	}
	if pipeline.Path == "" || installModuleVersion(pipeline) == "" {
		return installIdentityResult{}, installInvalid(op, "incomplete pipeline module identity")
	}
	pipelineSource, err := ops.source(installModuleSourceDir(pipeline))
	if err != nil {
		return installIdentityResult{}, installInvalid(op, err.Error())
	}
	if pipelineSource.revision == "" {
		return installIdentityResult{}, installInvalid(op, "empty pipeline revision")
	}
	result.Identity.PipelineModule = pipeline.Path
	result.Identity.PipelineImport = importPath
	result.Identity.PipelineVersion = installModuleVersion(pipeline)
	result.Identity.PipelineVCSRevision = pipelineSource.revision
	result.Identity.PipelineVCSModified = pipelineSource.modified
	if pipelineSource.modified {
		result.Identity.PipelineSourceRealpath = pipelineSource.realpath
	}
	return result, nil
}

func partitionInstallIdentity(module installModule) (string, error) {
	version := installModuleVersion(module)
	if module.Replace != nil || version == "(devel)" || version == "v0.0.0" {
		return "local-pin", nil
	}
	if module.Replace == nil && version != "v0.0.0" && installExactTagPattern.MatchString(version) {
		return "exact-tag", nil
	}
	return "", fmt.Errorf("unsupported gobble module version %q", version)
}

func installModuleVersion(module installModule) string {
	if module.Version == "" && module.Main {
		return "(devel)"
	}
	return module.Version
}

func installModuleSourceDir(module installModule) string {
	if module.Replace != nil {
		return module.Replace.Dir
	}
	return module.Dir
}

func listInstallModule(goBin, cwd, path string) (installModule, error) {
	args := []string{"list", "-m", "-json"}
	if path != "" {
		args = append(args, path)
	}
	cmd := exec.Command(goBin, args...)
	cmd.Dir = cwd
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && len(bytes.TrimSpace(exitErr.Stderr)) > 0 {
			return installModule{}, errors.New(strings.TrimSpace(string(exitErr.Stderr)))
		}
		return installModule{}, err
	}
	var module installModule
	if err := json.Unmarshal(out, &module); err != nil {
		return installModule{}, err
	}
	return module, nil
}

func readInstallSource(dir string) (installSource, error) {
	if dir == "" {
		return installSource{}, errors.New("empty module source directory")
	}
	realpath, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return installSource{}, err
	}
	realpath = filepath.Clean(realpath)
	if realpath == "" {
		return installSource{}, errors.New("empty module source realpath")
	}
	revision, err := runInstallGit(realpath, "rev-parse", "HEAD")
	if err != nil {
		return installSource{}, err
	}
	if revision == "" {
		return installSource{}, errors.New("empty module revision")
	}
	status, err := runInstallGit(realpath, "status", "--porcelain")
	if err != nil {
		return installSource{}, err
	}
	return installSource{revision: revision, modified: status != "", realpath: realpath}, nil
}

func runInstallGit(dir string, args ...string) (string, error) {
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

func installExecutableSHA256() (string, error) {
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

func validInstallDigest(digest string) bool {
	if len(digest) != sha256.Size*2 || digest != strings.ToLower(digest) {
		return false
	}
	for _, r := range digest {
		if !strings.ContainsRune("0123456789abcdef", r) {
			return false
		}
	}
	return true
}

func vcsSetting(settings []debug.BuildSetting, key string) string {
	for _, setting := range settings {
		if setting.Key == key {
			return setting.Value
		}
	}
	return ""
}

func hasInternalImport(importPath string) bool {
	for _, part := range strings.Split(importPath, "/") {
		if part == "internal" {
			return true
		}
	}
	return false
}

func installInvalid(op, message string) error {
	return &gobble.Error{Op: op, Defects: []gobble.Defect{{
		Code:    gobble.DefectInvalidRequest,
		Unit:    "identity",
		Message: message,
	}}}
}

func installMismatch(op, required, have string) error {
	return &gobble.Error{Op: op, Defects: []gobble.Defect{{
		Code:    gobble.DefectIdentityMismatch,
		Unit:    "gobble",
		Message: "required " + required + "; have " + have,
	}}}
}
