package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const driverTempPrefix = "gobble-driver-"

func runDriver(req *request, stdout, stderr io.Writer) int {
	goBin, err := exec.LookPath("go")
	if err != nil {
		return writeErr(stderr, invalidRequest(req.command, err.Error()), 2)
	}
	cwd, err := os.Getwd()
	if err != nil {
		return writeErr(stderr, invalidRequest(req.command, err.Error()), 2)
	}
	importPath, err := resolveImport(goBin, cwd, req.pkg)
	if err != nil {
		return writeErr(stderr, invalidRequest(req.command, err.Error()), 2)
	}
	dir, err := os.MkdirTemp("", driverTempPrefix)
	if err != nil {
		return writeErr(stderr, invalidRequest(req.command, err.Error()), 2)
	}
	defer func() { _ = os.RemoveAll(dir) }()

	src := filepath.Join(dir, "main.go")
	if err := os.WriteFile(src, []byte(driverSource(importPath, req)), 0o600); err != nil {
		return writeErr(stderr, invalidRequest(req.command, err.Error()), 2)
	}
	bin := filepath.Join(dir, "driver")
	build := exec.Command(goBin, "build", "-o", bin, src)
	build.Dir = cwd
	if out, err := build.CombinedOutput(); err != nil {
		return writeErr(stderr, invalidRequest(req.command, compileMessage(out, err)), 2)
	}
	cmd := exec.Command(bin)
	cmd.Dir = cwd
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			return ee.ExitCode()
		}
		return writeErr(stderr, invalidRequest(req.command, err.Error()), 2)
	}
	return 0
}

func resolveImport(goBin, cwd, pkg string) (string, error) {
	cmd := exec.Command(goBin, "list", "-f", "{{.ImportPath}}", "--", pkg)
	cmd.Dir = cwd
	out, err := cmd.CombinedOutput()
	msg := strings.TrimSpace(string(out))
	if err != nil {
		if msg == "" {
			return "", err
		}
		return "", errors.New(msg)
	}
	if msg == "" {
		return "", errors.New("empty import path")
	}
	if i := strings.IndexByte(msg, '\n'); i >= 0 {
		msg = strings.TrimSpace(msg[:i])
	}
	return msg, nil
}

func compileMessage(out []byte, err error) string {
	msg := strings.TrimSpace(string(out))
	if msg == "" {
		return err.Error()
	}
	return msg
}

func driverSource(importPath string, req *request) string {
	return fmt.Sprintf(driverTemplate, importPath, req.command, req.workspace, req.cap)
}

const driverTemplate = `package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"

	userpipe %q
	"github.com/HahyeonJeon/gobble"
)

const (
	verb      = %q
	workspace = %q
	cap       = %d
)

func main() {
	os.Exit(run())
}

func run() int {
	g, err := gobble.Compose(userpipe.Pipeline())
	if err != nil {
		return writeLibErr(err)
	}
	switch verb {
	case "compose":
		return writeJSON(struct {
			Op       string ` + "`json:\"op\"`" + `
			Pipeline string ` + "`json:\"pipeline\"`" + `
		}{Op: "compose", Pipeline: g.Name()})
	case "validate":
		if err := gobble.Validate(g); err != nil {
			return writeLibErr(err)
		}
		return writeJSON(struct {
			Op string ` + "`json:\"op\"`" + `
		}{Op: "validate"})
	case "plan":
		p, err := gobble.BuildPlan(g)
		if err != nil {
			return writeLibErr(err)
		}
		var buf bytes.Buffer
		if err := p.WriteJSON(&buf); err != nil {
			return writeLibErr(err)
		}
		if _, err := os.Stdout.Write(buf.Bytes()); err != nil {
			return writeFail("stdout write failed")
		}
		return 0
	default:
		return writeFail("invalid request")
	}
}

func writeJSON(v any) int {
	data, err := json.Marshal(v)
	if err != nil {
		return writeFail(err.Error())
	}
	if _, err := os.Stdout.Write(append(data, '\n')); err != nil {
		return writeFail("stdout write failed")
	}
	return 0
}

func writeLibErr(err error) int {
	var ge *gobble.Error
	if !errors.As(err, &ge) {
		return writeFail(err.Error())
	}
	return writeErrJSON(ge)
}

func writeFail(message string) int {
	return writeErrJSON(&gobble.Error{
		Op: verb,
		Defects: []gobble.Defect{{
			Code:    gobble.DefectInvalidRequest,
			Message: message,
		}},
	})
}

func writeErrJSON(ge *gobble.Error) int {
	data, err := json.Marshal(ge)
	if err != nil {
		return 1
	}
	_, _ = os.Stderr.Write(append(data, '\n'))
	return 1
}
`
