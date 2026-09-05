package main

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// The generated runner imports the TUI even when the pipeline only imports
// the root library. Resolve that extra dependency closure in a temporary
// module file, preserving the consumer's requirements and checksum file.
func packedModuleArgs(goBin, cwd, dir string) ([]string, error) {
	cmd := exec.Command(goBin, "env", "-json", "GOMOD", "GOWORK")
	cmd.Dir = cwd
	raw, err := cmd.CombinedOutput()
	if err != nil {
		return nil, errors.New(compileMessage(raw, err))
	}
	var env struct{ GOMOD, GOWORK string }
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, err
	}
	// Workspace builds already resolve all participating module graphs and use
	// the workspace's dependency policy. Go disallows -modfile in that mode.
	if env.GOWORK != "" && env.GOWORK != "off" {
		return nil, nil
	}
	if env.GOMOD == "" || env.GOMOD == os.DevNull {
		return nil, nil
	}
	modfile := filepath.Join(dir, "runner.mod")
	if err := copyPackFile(env.GOMOD, modfile, 0600); err != nil {
		return nil, err
	}
	sumfile := strings.TrimSuffix(env.GOMOD, ".mod") + ".sum"
	if err := copyPackFile(sumfile, filepath.Join(dir, "runner.sum"), 0600); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	// -modfile retains the original module root, including relative replaces.
	return []string{"-mod=mod", "-modfile=" + modfile}, nil
}
