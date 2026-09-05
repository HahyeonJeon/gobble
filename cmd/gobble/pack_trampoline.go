package main

import (
	"go/format"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const packInnerName = "inner"

func buildPackedTrampoline(goBin, cwd, dir, innerPath string, forceFallback bool) (string, error) {
	if err := copyPackFile(innerPath, filepath.Join(dir, packInnerName), 0o600); err != nil {
		return "", err
	}
	source, err := packTrampolineSource(forceFallback)
	if err != nil {
		return "", err
	}
	sourcePath := filepath.Join(dir, "main.go")
	if err := os.WriteFile(sourcePath, []byte(source), 0o600); err != nil {
		return "", err
	}
	output := filepath.Join(dir, "runner")
	if err := buildPacked(goBin, cwd, sourcePath, output); err != nil {
		return "", err
	}
	return output, nil
}

func packTrampolineSource(forceFallback bool) (string, error) {
	src := strings.ReplaceAll(packTrampolineTemplate, "__FORCE_FALLBACK__", strconv.FormatBool(forceFallback))
	formatted, err := format.Source([]byte(src))
	if err != nil {
		return "", err
	}
	return string(formatted), nil
}

const packTrampolineTemplate = `package main

import (
	"bytes"
	_ "embed"
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
	"unsafe"
)

const forceFallback = __FORCE_FALLBACK__

//go:embed inner
var inner []byte

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	protocol, err := openProtocol()
	if err != nil {
		return writeFail(args, "child protocol create failed", 1)
	}
	defer protocol.Close()
	cmd, err := startInner(protocol, args)
	if err != nil {
		return writeFail(args, err.Error(), 1)
	}
	stopForwarding := forwardSignals(cmd.Process)
	waitErr := cmd.Wait()
	stopForwarding()
	if waitErr != nil {
		code := waitCode(waitErr)
		if code < 0 {
			return writeFail(args, waitErr.Error(), 1)
		}
		return code
	}
	return copyProtocol(protocol, args)
}

func openProtocol() (*os.File, error) {
	file, err := os.CreateTemp(os.TempDir(), "gobble-packed-protocol-*")
	if err != nil {
		return nil, err
	}
	if err := os.Remove(file.Name()); err != nil {
		file.Close()
		_ = os.Remove(file.Name())
		return nil, err
	}
	return file, nil
}

func startInner(protocol *os.File, args []string) (*exec.Cmd, error) {
	var memfdErr error
	if !forceFallback {
		cmd, err := startMemfd(protocol, args)
		if err == nil {
			return cmd, nil
		}
		memfdErr = err
	} else {
		memfdErr = errors.New("memfd disabled")
	}
	cmd, err := startTemp(protocol, args)
	if err != nil {
		return nil, fmt.Errorf("memfd extraction failed: %v; temp fallback failed: %w", memfdErr, err)
	}
	return cmd, nil
}

func startMemfd(protocol *os.File, args []string) (*exec.Cmd, error) {
	name, err := syscall.BytePtrFromString("gobble-packed-inner")
	if err != nil {
		return nil, err
	}
	fd, _, errno := syscall.Syscall(319, uintptr(unsafe.Pointer(name)), 1, 0)
	if errno != 0 {
		return nil, errno
	}
	file := os.NewFile(fd, "gobble-packed-inner")
	if file == nil {
		_ = syscall.Close(int(fd))
		return nil, errors.New("memfd unavailable")
	}
	defer file.Close()
	if err := file.Chmod(0o700); err != nil {
		return nil, err
	}
	if err := writeAll(file, inner); err != nil {
		return nil, err
	}
	cmd := innerCommand(fmt.Sprintf("/proc/self/fd/%d", fd), protocol, args)
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return cmd, nil
}

func startTemp(protocol *os.File, args []string) (*exec.Cmd, error) {
	file, err := os.CreateTemp(os.TempDir(), "gobble-packed-inner-*")
	if err != nil {
		return nil, err
	}
	path := file.Name()
	remove := true
	defer func() {
		file.Close()
		if remove {
			_ = os.Remove(path)
		}
	}()
	if err := file.Chmod(0o700); err != nil {
		return nil, err
	}
	if err := writeAll(file, inner); err != nil {
		return nil, err
	}
	if err := file.Close(); err != nil {
		return nil, err
	}
	cmd := innerCommand(path, protocol, args)
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	if err := os.Remove(path); err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return nil, err
	}
	remove = false
	return cmd, nil
}

func writeAll(file *os.File, data []byte) error {
	written, err := file.Write(data)
	if err != nil {
		return err
	}
	if written != len(data) {
		return io.ErrShortWrite
	}
	return nil
}

func innerCommand(path string, protocol *os.File, args []string) *exec.Cmd {
	cmd := exec.Command(path, args...)
	cmd.Stdout = io.Discard
	cmd.Stderr = os.Stderr
	if emptyWatch(args) {
		cmd.Stdin = os.Stdin
	}
	cmd.ExtraFiles = []*os.File{protocol}
	return cmd
}

func forwardSignals(process *os.Process) func() {
	signals := make(chan os.Signal, 2)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	done := make(chan struct{})
	var wait sync.WaitGroup
	wait.Add(1)
	go func() {
		defer wait.Done()
		for {
			select {
			case sig := <-signals:
				_ = process.Signal(sig)
			case <-done:
				return
			}
		}
	}()
	return func() {
		signal.Stop(signals)
		close(done)
		wait.Wait()
	}
}

func waitCode(err error) int {
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		return -1
	}
	if exitErr.ExitCode() == 2 {
		return 2
	}
	return 1
}

func copyProtocol(protocol *os.File, args []string) int {
	if _, err := protocol.Seek(0, io.SeekStart); err != nil {
		return writeFail(args, "child protocol read failed", 1)
	}
	data, err := io.ReadAll(protocol)
	if err != nil || !validProtocol(data, args) {
		return writeFail(args, "invalid child protocol", 1)
	}
	if _, err := os.Stdout.Write(data); err != nil {
		return writeFail(args, "stdout write failed", 1)
	}
	return 0
}

func validProtocol(data []byte, args []string) bool {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return len(data) == 0 && (emptyInspectJSONL(args) || emptyWatch(args))
	}
	if json.Valid(trimmed) {
		return true
	}
	lines := bytes.Split(data, []byte{'\n'})
	if len(lines) > 1 {
		valid := true
		for i, line := range lines {
			if i == len(lines)-1 && len(line) == 0 {
				continue
			}
			line = bytes.TrimSuffix(line, []byte{'\r'})
			if len(line) == 0 || !json.Valid(bytes.TrimSpace(line)) {
				valid = false
				break
			}
		}
		if valid {
			return true
		}
	}
	return helpShape(args)
}

func emptyInspectJSONL(args []string) bool { return emptyView(args, false) }

func emptyWatch(args []string) bool { return emptyView(args, true) }

func emptyView(args []string, watch bool) bool {
	positionals := make([]string, 0, 2)
	options := true
	workspaceSet := false
	instanceSet := false
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if options {
			if arg == "--" {
				options = false
				continue
			}
			if arg == "-h" || arg == "--help" || arg == "--version" {
				return false
			}
			if strings.HasPrefix(arg, "--") {
				name, value, hasValue := strings.Cut(arg[2:], "=")
				if (name != "workspace" && name != "instance") || (watch && name == "instance") {
					return false
				}
				if !hasValue {
					if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
						return false
					}
					i++
					value = args[i]
				}
				if name == "workspace" {
					if workspaceSet || value == "" {
						return false
					}
					workspaceSet = true
				} else {
					if instanceSet {
						return false
					}
					instanceSet = true
				}
				continue
			}
			if strings.HasPrefix(arg, "-") {
				return false
			}
		}
		positionals = append(positionals, arg)
		if len(positionals) > 2 {
			return false
		}
	}
	if watch {
		return workspaceSet && len(positionals) == 1 && positionals[0] == "watch"
	}
	return workspaceSet && len(positionals) == 2 && positionals[0] == "inspect" &&
		(positionals[1] == "remaining" || positionals[1] == "reuse")
}

func helpShape(args []string) bool {
	if len(args) == 0 || args[0] == "help" {
		return true
	}
	for _, arg := range args {
		if arg == "-h" || arg == "--help" {
			return true
		}
	}
	return false
}

func writeFail(args []string, message string, code int) int {
	data, err := json.Marshal(map[string]any{
		"op": operation(args),
		"defects": []map[string]any{{
			"code": "invalid-request",
			"message": message,
		}},
	})
	if err == nil {
		_, _ = os.Stderr.Write(append(data, '\n'))
	}
	return code
}

func operation(args []string) string {
	if len(args) == 0 || args[0] == "-h" || args[0] == "--help" {
		return "cli"
	}
	if args[0] == "--version" {
		return "version"
	}
	return filepath.Base(args[0])
}
`
