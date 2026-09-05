package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSupportedHostsAndDesktopSocket(t *testing.T) {
	for _, host := range [][2]string{{"darwin", "amd64"}, {"darwin", "arm64"}, {"linux", "amd64"}, {"windows", "amd64"}} {
		if !supportedHost(host[0], host[1]) {
			t.Errorf("rejected %v", host)
		}
	}
	for _, host := range [][2]string{{"linux", "arm64"}, {"windows", "arm64"}, {"darwin", "386"}, {"freebsd", "amd64"}} {
		if supportedHost(host[0], host[1]) {
			t.Errorf("accepted %v", host)
		}
	}
	if got := controllerSocket("darwin", "unix:///Users/researcher/.docker/run/docker.sock"); got != "/var/run/docker.sock" {
		t.Fatal(got)
	}
	if got := controllerSocket("linux", "unix:///run/user/1000/docker.sock"); got != "/run/user/1000/docker.sock" {
		t.Fatal(got)
	}
}

func TestRuntimePlatformCheckedBeforePinningAndOnReuse(t *testing.T) {
	original := dockerOutput
	t.Cleanup(func() { dockerOutput = original })
	id := "sha256:" + strings.Repeat("a", 64)
	platform := "linux/arm64"
	dockerOutput = func(_ context.Context, args ...string) (string, error) { return id + " " + platform, nil }
	t.Setenv("GOBBLE_RUNTIME_IMAGE", "runtime:preview")
	root := t.TempDir()
	if _, err := selectRuntime(t.Context(), root, "daemon"); err == nil || !strings.Contains(err.Error(), "--platform linux/amd64") {
		t.Fatalf("arm runtime: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, runtimeLock)); !os.IsNotExist(err) {
		t.Fatal("incompatible runtime pinned")
	}
	platform = "linux/amd64"
	cfg, err := selectRuntime(t.Context(), root, "daemon")
	if err != nil || cfg.Image != id {
		t.Fatalf("pin: %v %v", cfg, err)
	}
	t.Setenv("GOBBLE_RUNTIME_IMAGE", "changed:tag")
	dockerOutput = func(_ context.Context, args ...string) (string, error) {
		if args[len(args)-1] != id {
			t.Fatalf("reused mutable tag: %v", args)
		}
		return id + " " + platform, nil
	}
	if _, err := selectRuntime(t.Context(), root, "daemon"); err != nil {
		t.Fatal(err)
	}
	platform = "linux/arm64"
	if _, err := selectRuntime(t.Context(), root, "daemon"); err == nil {
		t.Fatal("locked image platform not verified")
	}
	if _, err := selectRuntime(t.Context(), root, "different-daemon"); err == nil {
		t.Fatal("daemon mismatch accepted")
	}
}

func TestMacStyleSymlinkProjectPaths(t *testing.T) {
	root := t.TempDir()
	real := filepath.Join(root, "private", "project")
	if err := os.MkdirAll(filepath.Join(real, "runs", "demo"), 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "project")
	if err := os.Symlink(real, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	args, mounts, err := translateArgs(link, []string{"run", ".", "--workspace", filepath.Join(link, "runs", "demo")})
	if err != nil || len(mounts) != 0 || args[3] != "/gobble/project/runs/demo" {
		t.Fatalf("%v %v %v", args, mounts, err)
	}
}

func TestScaffoldMustStayOnHostMount(t *testing.T) {
	root := t.TempDir()
	for _, args := range [][]string{{"init", "new"}, {"demo", "rnaseq", "new"}} {
		translated, _, err := translateArgs(root, args)
		if err != nil || translated[len(translated)-1] != "./new" {
			t.Fatalf("%v: %v %v", args, translated, err)
		}
	}
	for _, args := range [][]string{{"init", "../lost"}, {"demo", "rnaseq", "../lost"}} {
		if _, _, err := translateArgs(root, args); err == nil {
			t.Fatalf("project outside host mount accepted: %v", args)
		}
	}
	link := filepath.Join(root, "external")
	if err := os.Symlink(t.TempDir(), link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, _, err := translateArgs(root, []string{"demo", "rnaseq", filepath.Join(link, "new")}); err == nil {
		t.Fatal("new project through external symlink accepted")
	}
}

func TestScaffoldHelpAndEndOfOptions(t *testing.T) {
	root := t.TempDir()
	args, _, err := translateArgs(root, []string{"init", "--help"})
	if err != nil || args[1] != "--help" {
		t.Fatalf("help became a directory: %v %v", args, err)
	}
	args, _, err = translateArgs(root, []string{"demo", "--", "rnaseq", "--example"})
	if err != nil || args[3] != "./--example" {
		t.Fatalf("literal directory: %v %v", args, err)
	}
}
