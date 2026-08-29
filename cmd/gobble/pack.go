package main

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/HahyeonJeon/gobble"
)

const packTempPrefix = "gobble-pack-"

var resolvePackInstallIdentity = resolveInstallIdentity

func runPack(req *request, stderr io.Writer) int {
	cwd, err := os.Getwd()
	if err != nil {
		return writeErr(stderr, invalidRequest("pack", err.Error()), 2)
	}
	output, err := resolvePackOutput(cwd, req.output)
	if err != nil {
		return writeErr(stderr, invalidRequest("pack", err.Error()), 2)
	}
	goBin, err := exec.LookPath("go")
	if err != nil {
		return writeErr(stderr, invalidRequest("pack", err.Error()), 2)
	}
	importPath, err := resolveImport(goBin, cwd, req.pkg)
	if err != nil {
		return writeErr(stderr, invalidRequest("pack", err.Error()), 2)
	}
	if hasInternalImport(importPath) {
		return writeErr(stderr, invalidRequest("pack", "consumer internal/ packages are unsupported for CLI graph verbs and pack; export Pipeline from a non-internal package"), 2)
	}
	install, err := resolvePackInstallIdentity(goBin, cwd, importPath, "pack")
	if err != nil {
		return writeDriverSetupError(stderr, "pack", err)
	}
	install, err = makePackedIdentity(install)
	if err != nil {
		return writeErr(stderr, packIdentityError(err), 2)
	}
	innerDir, err := os.MkdirTemp("", packTempPrefix+"inner-")
	if err != nil {
		return writeErr(stderr, invalidRequest("pack", err.Error()), 2)
	}
	defer func() { _ = os.RemoveAll(innerDir) }()
	innerBin, err := buildPackedInner(goBin, cwd, innerDir, importPath, install)
	if err != nil {
		return writeErr(stderr, invalidRequest("pack", err.Error()), 2)
	}
	trampolineDir, err := os.MkdirTemp("", packTempPrefix+"trampoline-")
	if err != nil {
		return writeErr(stderr, invalidRequest("pack", err.Error()), 2)
	}
	defer func() { _ = os.RemoveAll(trampolineDir) }()
	trampolineBin, err := buildPackedTrampoline(goBin, cwd, trampolineDir, innerBin, false)
	if err != nil {
		return writeErr(stderr, invalidRequest("pack", err.Error()), 2)
	}
	if err := persistPacked(trampolineBin, output); err != nil {
		return writeErr(stderr, invalidRequest("pack", err.Error()), 1)
	}
	return 0
}

func resolvePackOutput(cwd, output string) (string, error) {
	if !filepath.IsAbs(output) {
		output = filepath.Join(cwd, output)
	}
	output = filepath.Clean(output)
	info, err := os.Lstat(output)
	if err == nil {
		if info.IsDir() {
			return "", errors.New("output path is a directory")
		}
		if !info.Mode().IsRegular() {
			return "", errors.New("output path is not a regular file")
		}
		return output, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	return output, nil
}

func buildPacked(goBin, cwd, source, output string) error {
	cmd := exec.Command(goBin, "build", "-o", output, source)
	cmd.Dir = cwd
	cmd.Env = packedBuildEnv(os.Environ())
	out, err := cmd.CombinedOutput()
	if err != nil {
		return errors.New(compileMessage(out, err))
	}
	return nil
}

func packedBuildEnv(base []string) []string {
	env := make([]string, 0, len(base)+3)
	for _, value := range base {
		if strings.HasPrefix(value, "GOOS=") || strings.HasPrefix(value, "GOARCH=") || strings.HasPrefix(value, "CGO_ENABLED=") {
			continue
		}
		env = append(env, value)
	}
	return append(env, "GOOS=linux", "GOARCH=amd64", "CGO_ENABLED=0")
}

func copyPackFile(source, destination string, mode os.FileMode) error {
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	remove := true
	defer func() {
		_ = out.Close()
		if remove {
			_ = os.Remove(destination)
		}
	}()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	remove = false
	return nil
}

func persistPacked(source, output string) error {
	dir := filepath.Dir(output)
	stage, err := os.CreateTemp(dir, ".gobble-pack-*")
	if err != nil {
		return err
	}
	stagePath := stage.Name()
	remove := true
	defer func() {
		_ = stage.Close()
		if remove {
			_ = os.Remove(stagePath)
		}
	}()
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	if _, err := io.Copy(stage, in); err != nil {
		_ = in.Close()
		return err
	}
	if err := in.Close(); err != nil {
		return err
	}
	if err := stage.Chmod(0o755); err != nil {
		return err
	}
	if err := stage.Sync(); err != nil {
		return err
	}
	if err := stage.Close(); err != nil {
		return err
	}
	if err := os.Rename(stagePath, output); err != nil {
		return err
	}
	remove = false
	return nil
}

func makePackedIdentity(install installIdentityResult) (installIdentityResult, error) {
	id := install.Identity
	switch {
	case id.GobbleVCSModified:
		return installIdentityResult{}, installInvalid("pack", "dirty Gobble source")
	case id.PipelineVCSModified:
		return installIdentityResult{}, installInvalid("pack", "dirty pipeline source")
	case id.PipelineVCSRevision == "":
		return installIdentityResult{}, installInvalid("pack", "empty pipeline revision")
	case id.GobbleSourceRealpath != "":
		return installIdentityResult{}, installInvalid("pack", "unexpected Gobble source realpath")
	case id.PipelineSourceRealpath != "":
		return installIdentityResult{}, installInvalid("pack", "unexpected pipeline source realpath")
	case id.GOOS != "linux" || id.GOARCH != "amd64":
		return installIdentityResult{}, installInvalid("pack", "unsupported goos/goarch")
	case id.IdentityMode != "exact-tag" && id.IdentityMode != "local-pin":
		return installIdentityResult{}, installInvalid("pack", "unsupported identity mode")
	case id.GobbleModule == "" || id.GobbleVersion == "" || id.PipelineModule == "" || id.PipelineImport == "" || id.PipelineVersion == "":
		return installIdentityResult{}, installInvalid("pack", "incomplete packed identity")
	}
	if id.IdentityMode == "local-pin" && id.GobbleVCSRevision == "" {
		return installIdentityResult{}, installInvalid("pack", "empty gobble revision")
	}
	id.GobbleExecutableSHA256 = ""
	id.InstallKind = "packed-runner"
	return installIdentityResult{
		Identity:    id,
		HasReplace:  install.HasReplace,
		ReplacePath: install.ReplacePath,
	}, nil
}

func packIdentityError(err error) *gobble.Error {
	var ge *gobble.Error
	if errors.As(err, &ge) {
		return ge
	}
	return invalidRequest("pack", err.Error())
}
