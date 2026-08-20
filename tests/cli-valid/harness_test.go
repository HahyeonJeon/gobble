package cli_valid_test

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

type gobbleResult struct {
	stdout []byte
	stderr []byte
	code   int
}

func moduleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("go.mod not found from %s", dir)
		}
		dir = parent
	}
}

func buildGobble(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "gobble")
	cmd := exec.Command("go", "build", "-o", bin, "./cmd/gobble")
	cmd.Dir = moduleRoot(t)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build gobble: %v\n%s", err, out)
	}
	return bin
}

func gobbleCommand(t *testing.T, bin string, args ...string) *exec.Cmd {
	t.Helper()
	cmd := exec.Command(bin, args...)
	cmd.Dir = moduleRoot(t)
	tmp := t.TempDir()
	env := os.Environ()
	out := make([]string, 0, len(env)+1)
	for _, e := range env {
		if strings.HasPrefix(e, "TMPDIR=") {
			continue
		}
		out = append(out, e)
	}
	cmd.Env = append(out, "TMPDIR="+tmp)
	return cmd
}

func runGobble(t *testing.T, bin string, args ...string) gobbleResult {
	t.Helper()
	cmd := gobbleCommand(t, bin, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return gobbleResult{
		stdout: stdout.Bytes(),
		stderr: stderr.Bytes(),
		code:   exitCode(err),
	}
}

func exitCode(err error) int {
	if err == nil {
		return 0
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return ee.ExitCode()
	}
	return -1
}

func readGolden(t *testing.T, rel string) []byte {
	t.Helper()
	path := filepath.Join(moduleRoot(t), filepath.FromSlash(rel))
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", path, err)
	}
	return data
}
