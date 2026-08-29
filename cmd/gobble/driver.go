package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	"github.com/HahyeonJeon/gobble"
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
	if hasInternalImport(importPath) {
		return writeErr(stderr, invalidRequest(req.command, "consumer internal/ packages are unsupported for CLI graph verbs and pack; export Pipeline from a non-internal package"), 2)
	}
	install, err := resolveInstallIdentity(goBin, cwd, importPath, req.command)
	if err != nil {
		return writeDriverSetupError(stderr, req.command, err)
	}
	dir, err := os.MkdirTemp("", driverTempPrefix)
	if err != nil {
		return writeErr(stderr, invalidRequest(req.command, err.Error()), 2)
	}
	defer func() { _ = os.RemoveAll(dir) }()

	src := filepath.Join(dir, "main.go")
	if err := os.WriteFile(src, []byte(driverSource(importPath, req, install)), 0o600); err != nil {
		return writeErr(stderr, invalidRequest(req.command, err.Error()), 2)
	}
	protocol, err := os.OpenFile(filepath.Join(dir, "protocol"), os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0o600)
	if err != nil {
		return writeErr(stderr, invalidRequest(req.command, err.Error()), 2)
	}
	defer protocol.Close()
	bin := filepath.Join(dir, "driver")
	build := exec.Command(goBin, "build", "-o", bin, src)
	build.Dir = cwd
	if out, err := build.CombinedOutput(); err != nil {
		return writeErr(stderr, invalidRequest(req.command, compileMessage(out, err)), 2)
	}
	cmd := exec.Command(bin)
	cmd.Dir = cwd
	cmd.Stdout = io.Discard
	cmd.Stderr = stderr
	cmd.ExtraFiles = []*os.File{protocol}
	if err := cmd.Start(); err != nil {
		return writeErr(stderr, invalidRequest(req.command, err.Error()), 2)
	}
	if req.command == "run" || req.command == "resume" {
		defer forwardSignals(cmd.Process)()
	}
	if err := cmd.Wait(); err != nil {
		code := driverWaitCode(err)
		if code < 0 {
			return writeErr(stderr, invalidRequest(req.command, err.Error()), 2)
		}
		return code
	}
	return copyDriverProtocol(protocol, stdout, stderr, req.command)
}

func writeDriverSetupError(stderr io.Writer, op string, err error) int {
	var ge *gobble.Error
	if errors.As(err, &ge) {
		return writeErr(stderr, ge, 2)
	}
	return writeErr(stderr, invalidRequest(op, err.Error()), 2)
}

func copyDriverProtocol(protocol *os.File, stdout, stderr io.Writer, op string) int {
	if _, err := protocol.Seek(0, io.SeekStart); err != nil {
		return writeErr(stderr, invalidRequest(op, "child protocol read failed"), 1)
	}
	data, err := io.ReadAll(protocol)
	if err != nil || !json.Valid(bytes.TrimSpace(data)) {
		return writeErr(stderr, invalidRequest(op, "invalid child protocol"), 1)
	}
	if _, err := stdout.Write(data); err != nil {
		return writeErr(stderr, invalidRequest(op, "stdout write failed"), 1)
	}
	return 0
}

func driverWaitCode(err error) int {
	var ee *exec.ExitError
	if !errors.As(err, &ee) {
		return -1
	}
	code := ee.ExitCode()
	if code == 2 {
		return 2
	}
	if code >= 0 {
		return 1
	}
	if ws, ok := ee.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
		return 1
	}
	return 1
}

func forwardSignals(proc *os.Process) func() {
	sigs := make(chan os.Signal, 2)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	done := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case s := <-sigs:
				_ = proc.Signal(s)
			case <-done:
				return
			}
		}
	}()
	return func() {
		signal.Stop(sigs)
		close(done)
		wg.Wait()
	}
}

func resolveImport(goBin, cwd, pkg string) (string, error) {
	cmd := exec.Command(goBin, "list", "-f", "{{.ImportPath}}", "--", pkg)
	cmd.Dir = cwd
	out, err := cmd.Output()
	msg := strings.TrimSpace(string(out))
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			stderr := strings.TrimSpace(string(ee.Stderr))
			if stderr != "" {
				return "", errors.New(stderr)
			}
		}
		if msg != "" {
			return "", errors.New(msg)
		}
		return "", err
	}
	if msg == "" {
		return "", errors.New("empty import path")
	}
	paths := make([]string, 0, 1)
	for _, line := range strings.Split(msg, "\n") {
		if path := strings.TrimSpace(line); path != "" {
			paths = append(paths, path)
		}
	}
	if len(paths) != 1 {
		return "", errors.New("go list matched multiple packages")
	}
	return paths[0], nil
}

func compileMessage(out []byte, err error) string {
	msg := strings.TrimSpace(string(out))
	if msg == "" {
		return err.Error()
	}
	return msg
}

func driverSource(importPath string, req *request, install installIdentityResult) string {
	identityJSON, _ := json.Marshal(install.Identity)
	return fmt.Sprintf(driverTemplate, importPath, req.command, req.workspace, req.cap, req.sample, string(identityJSON), install.HasReplace, install.ReplacePath)
}

const driverTemplate = `package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"

	userpipe %q
	"github.com/HahyeonJeon/gobble"
)

const (
	verb      = %q
	workspace = %q
	cap       = %d
	sample    = %q
	embeddedIdentity = %q
	expectReplace = %t
	expectedReplacePath = %q
)

var protocol = os.NewFile(3, "gobble-protocol")

func main() {
	os.Exit(run())
}

func run() int {
	identity, err := linkedIdentity()
	if err != nil {
		return writeIdentityFail(err)
	}
	gobble.SetSampleSheetPath(sample)
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
		if err := writeProtocol(buf.Bytes()); err != nil {
			return writeFail("stdout write failed")
		}
		return 0
	case "run", "resume":
		return occupy(g, identity)
	default:
		return writeFail("invalid request")
	}
}

func occupy(g *gobble.Graph, identity gobble.Identity) int {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	var err error
	if verb == "run" {
		err = gobble.Run(ctx, g, workspace, cap, gobble.WithIdentity(identity))
	} else {
		err = gobble.Resume(ctx, g, workspace, cap, gobble.WithIdentity(identity))
	}
	if err != nil {
		return writeLibErr(err)
	}
	return writeJSON(struct {
		Op string ` + "`json:\"op\"`" + `
	}{Op: verb})
}

func linkedIdentity() (gobble.Identity, error) {
	var identity gobble.Identity
	if err := json.Unmarshal([]byte(embeddedIdentity), &identity); err != nil {
		return identity, err
	}
	info, ok := debug.ReadBuildInfo()
	if !ok || info == nil {
		return identity, errors.New("linked Gobble build info unavailable")
	}
	for _, dep := range info.Deps {
		if dep == nil || dep.Path != identity.GobbleModule {
			continue
		}
		if dep.Version != identity.GobbleVersion {
			return identity, fmt.Errorf("required Gobble %%s@%%s; have %%s@%%s", identity.GobbleModule, identity.GobbleVersion, dep.Path, dep.Version)
		}
		if expectReplace && (dep.Replace == nil || dep.Replace.Path != expectedReplacePath) {
			have := ""
			if dep.Replace != nil {
				have = dep.Replace.Path
			}
			return identity, fmt.Errorf("required Gobble replace %%s; have %%s", expectedReplacePath, have)
		}
		return identity, nil
	}
	return identity, fmt.Errorf("required Gobble %%s@%%s; linked dependency missing", identity.GobbleModule, identity.GobbleVersion)
}

func writeIdentityFail(err error) int {
	return writeErrJSON(&gobble.Error{
		Op: verb,
		Defects: []gobble.Defect{{
			Code: gobble.DefectIdentityMismatch,
			Unit: "gobble",
			Message: err.Error(),
		}},
	}, 2)
}

func writeJSON(v any) int {
	data, err := json.Marshal(v)
	if err != nil {
		return writeFail(err.Error())
	}
	if err := writeProtocol(append(data, '\n')); err != nil {
		return writeFail("stdout write failed")
	}
	return 0
}

func writeProtocol(data []byte) error {
	if protocol == nil {
		return errors.New("protocol unavailable")
	}
	_, err := protocol.Write(data)
	return err
}

func writeLibErr(err error) int {
	var ge *gobble.Error
	if !errors.As(err, &ge) {
		return writeFail(err.Error())
	}
	code := 1
	if gobble.IsSampleSheetError(ge) {
		code = 2
	}
	return writeErrJSON(ge, code)
}

func writeFail(message string) int {
	return writeErrJSON(&gobble.Error{
		Op: verb,
		Defects: []gobble.Defect{{
			Code:    gobble.DefectInvalidRequest,
			Message: message,
		}},
	}, 1)
}

func writeErrJSON(ge *gobble.Error, code int) int {
	data, err := json.Marshal(ge)
	if err != nil {
		return 1
	}
	_, _ = os.Stderr.Write(append(data, '\n'))
	return code
}
`
