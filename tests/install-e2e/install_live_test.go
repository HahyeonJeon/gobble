//go:build live

package install_e2e_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type apiResult struct {
	Op              string         `json:"op"`
	RunningIdentity string         `json:"running_identity"`
	Remaining       []string       `json:"remaining"`
	Identity        identityFields `json:"identity"`
}

type assayEvidence struct {
	WorktreeHead      string          `json:"worktree_head"`
	CandidateManifest string          `json:"candidate_manifest_sha256"`
	SnapshotCommit    string          `json:"snapshot_commit"`
	ConsumerCommit    string          `json:"consumer_commit"`
	SelectedGobble    string          `json:"selected_gobble_revision"`
	InstalledGobble   string          `json:"installed_gobble_path"`
	InstalledSHA256   string          `json:"installed_gobble_sha256"`
	PackedWGS         string          `json:"packed_wgs_path"`
	PackedWGSSHA256   string          `json:"packed_wgs_sha256"`
	APIWorkspace      string          `json:"api_workspace"`
	CLIWorkspace      string          `json:"cli_workspace"`
	PackedWorkspace   string          `json:"packed_workspace"`
	APIRemaining      []string        `json:"api_remaining"`
	CLIRemaining      []string        `json:"cli_remaining"`
	PackedRemaining   []string        `json:"packed_remaining"`
	DockerObserved    map[string]bool `json:"docker_observed"`
	Recovered         map[string]bool `json:"recovered"`
}

func TestDualInstallFirstHorizon(t *testing.T) {
	a := newAssay(t)

	t.Run("agent installed path", func(t *testing.T) {
		requireCommand(t, runCommand(a.apiAssay, a.consumer, a.agentEnv, "workflow"), "API workflow compose/validate/plan")
		for _, command := range []string{"compose", "validate", "plan"} {
			result := requireCommand(t, runCommand(a.gobble, a.consumer, a.agentEnv, command, "./workflowcase"), "installed workflow "+command)
			if len(bytes.TrimSpace(result.Stdout)) == 0 || !json.Valid(bytes.TrimSpace(result.Stdout)) {
				t.Fatalf("installed workflow %s stdout is not protocol JSON: %q", command, result.Stdout)
			}
		}
		proveHandshakeFailure(t, a)
	})

	t.Run("human packed path", func(t *testing.T) {
		provePackFailures(t, a)
		packRunner(t, a, "./workflowcase", a.packedWorkflow)
		packRunner(t, a, "./wgs", a.packedWGS)
		packRunner(t, a, "./printpipe", a.packedPrint)
		a.packedWGSSHA256 = fileSHA256(t, a.packedWGS)
		provePackedMetadata(t, a)
		for _, command := range []string{"compose", "validate", "plan"} {
			result := requireCommand(t, runCommand(a.packedWorkflow, t.TempDir(), a.neutralEnv, command), "packed workflow "+command)
			if len(bytes.TrimSpace(result.Stdout)) == 0 || !json.Valid(bytes.TrimSpace(result.Stdout)) {
				t.Fatalf("packed workflow %s stdout is not protocol JSON: %q", command, result.Stdout)
			}
		}
		printResult := requireCommand(t, runCommand(a.packedPrint, t.TempDir(), a.neutralEnv, "compose"), "packed printpipe compose")
		for _, leak := range [][]byte{[]byte("user init output"), []byte("user Pipeline output")} {
			if bytes.Contains(printResult.Stdout, leak) || bytes.Contains(printResult.Stderr, leak) {
				t.Fatalf("packed compose leaked %q: stdout=%q stderr=%q", leak, printResult.Stdout, printResult.Stderr)
			}
		}
		requireStructuredFailure(t, runCommand(a.packedWorkflow, t.TempDir(), a.neutralEnv, "pack"), "packed pack refusal", 2, "invalid-request", "")
		proveHelpHonesty(t, a)
		requireNoGoInvocation(t, a.goStubMarker, "packed workflow and help")
	})

	apiWorkspace := t.TempDir()
	cliWorkspace := t.TempDir()
	packedWorkspace := t.TempDir()
	if apiWorkspace == cliWorkspace || apiWorkspace == packedWorkspace || cliWorkspace == packedWorkspace {
		t.Fatalf("WGS workspaces are not distinct: api=%s cli=%s packed=%s", apiWorkspace, cliWorkspace, packedWorkspace)
	}

	var apiPrepare apiResult
	var apiRemaining, cliRemaining, packedRemaining []string
	t.Run("remaining work", func(t *testing.T) {
		stageWGSWorkspace(t, a, apiWorkspace)
		prepare := requireCommand(t, runCommand(a.apiAssay, a.consumer, a.agentEnv, "prepare", apiWorkspace), "API WGS prepare cancellation")
		if err := json.Unmarshal(prepare.Stdout, &apiPrepare); err != nil {
			t.Fatalf("API prepare JSON: %v\n%s", err, prepare.Stdout)
		}
		if apiPrepare.Op != "prepare" || apiPrepare.RunningIdentity == "" || len(apiPrepare.Remaining) == 0 {
			t.Fatalf("API prepare evidence incomplete: %#v", apiPrepare)
		}
		if apiPrepare.Identity.GobbleVCSRevision != a.snapshotCommit || apiPrepare.Identity.PipelineVCSRevision != a.consumerCommit || apiPrepare.Identity.IdentityMode != "local-pin" || apiPrepare.Identity.InstallKind != "module" {
			t.Fatalf("API install identity = %#v, want snapshot=%s consumer=%s", apiPrepare.Identity, a.snapshotCommit, a.consumerCommit)
		}
		apiRemaining = append([]string(nil), apiPrepare.Remaining...)
		proveAPIDigestMismatch(t, a, apiWorkspace, apiPrepare.Identity)
		resume := requireCommand(t, runCommand(a.apiAssay, a.consumer, a.agentEnv, "resume", apiWorkspace), "API WGS Release/Resume")
		var apiResume apiResult
		if err := json.Unmarshal(resume.Stdout, &apiResume); err != nil || apiResume.Op != "resume" || len(apiResume.Remaining) == 0 {
			t.Fatalf("API Resume evidence: err=%v result=%#v raw=%s", err, apiResume, resume.Stdout)
		}
		requireWGSOutputs(t, apiWorkspace)

		cliRemaining = runInstalledWGSRecovery(t, a, cliWorkspace)

		packedRemaining = runPackedWGSRecovery(t, a, packedWorkspace)
		packedInspectDir := t.TempDir()
		packedInspect := func(view string) commandResult {
			return runCommand(a.packedWGS, packedInspectDir, a.neutralEnv, "inspect", view, "--workspace", packedWorkspace)
		}
		provePackedCrossKindMismatch(t, a, packedWorkspace)
		finishPackedWGSRecovery(t, a, packedWorkspace, packedInspect)
	})

	requireCleanGit(t, a.gitBin, a.snapshot)
	requireCleanGit(t, a.gitBin, a.consumer)
	requireNoGoInvocation(t, a.goStubMarker, "all packed and neutral commands")
	evidence := assayEvidence{
		WorktreeHead:      a.worktreeHead,
		CandidateManifest: a.manifestSHA256,
		SnapshotCommit:    a.snapshotCommit,
		ConsumerCommit:    a.consumerCommit,
		SelectedGobble:    apiPrepare.Identity.GobbleVCSRevision,
		InstalledGobble:   a.gobble,
		InstalledSHA256:   a.gobbleSHA256,
		PackedWGS:         a.packedWGS,
		PackedWGSSHA256:   a.packedWGSSHA256,
		APIWorkspace:      apiWorkspace,
		CLIWorkspace:      cliWorkspace,
		PackedWorkspace:   packedWorkspace,
		APIRemaining:      apiRemaining,
		CLIRemaining:      cliRemaining,
		PackedRemaining:   packedRemaining,
		DockerObserved:    map[string]bool{"api": true, "installed-cli": true, "packed": true},
		Recovered:         map[string]bool{"api": true, "installed-cli": true, "packed": true},
	}
	encoded, err := json.Marshal(evidence)
	if err != nil {
		t.Fatalf("marshal evidence: %v", err)
	}
	t.Logf("install evidence: %s", encoded)
}

func proveHandshakeFailure(t *testing.T, a *assay) {
	t.Helper()
	clone := filepath.Join(t.TempDir(), "consumer")
	requireCommand(t, runCommand(a.gitBin, t.TempDir(), nil, "clone", "-q", a.consumer, clone), "clone handshake consumer")
	requireCleanGit(t, a.gitBin, clone)
	swapped := filepath.Join(t.TempDir(), "go.mod")
	goMod := strings.ReplaceAll(string(mustReadFile(t, filepath.Join(clone, "go.mod"))), a.snapshot, a.secondSnapshot)
	writeFile(t, swapped, 0o644, goMod)
	wrapperDir := t.TempDir()
	writeFile(t, filepath.Join(wrapperDir, "go"), 0o755, "#!/bin/sh\nif [ \"$1\" = build ]; then\n  /bin/cp \"$ASSAY_SWAP_GO_MOD\" \"$ASSAY_CLONE/go.mod\" || exit 98\nfi\nexec \"$ASSAY_REAL_GO\" \"$@\"\n")
	marker := filepath.Join(t.TempDir(), "pipeline-called")
	workspace := t.TempDir()
	before := workspaceInventory(t, workspace)
	pathValue := joinPATH(wrapperDir, a.gobin, filepath.Dir(a.gitBin), filepath.Dir(a.dockerBin), "/bin")
	env := replaceEnv(os.Environ(), map[string]string{
		"ASSAY_CLONE":           clone,
		"ASSAY_PIPELINE_MARKER": marker,
		"ASSAY_REAL_GO":         a.goBin,
		"ASSAY_SWAP_GO_MOD":     swapped,
		"PATH":                  pathValue,
	})
	result := runCommand(a.gobble, clone, env, "run", "./markerpipe", "--workspace", workspace, "--cap", "1")
	requireStructuredFailure(t, result, "generated-child handshake mismatch", 2, "identity-mismatch", "gobble")
	if _, err := os.Stat(marker); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Pipeline ran before handshake failure: %v", err)
	}
	if _, err := os.Stat(filepath.Join(workspace, ".gobble")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("handshake failure wrote workspace control state: %v", err)
	}
	after := workspaceInventory(t, workspace)
	if after != before {
		t.Fatalf("handshake failure mutated workspace: before=%s after=%s", before, after)
	}
}

func provePackFailures(t *testing.T, a *assay) {
	t.Helper()
	requireStructuredFailure(t, runCommand(a.gobble, a.consumer, a.agentEnv, "pack", "./wgs"), "pack without --output", 2, "invalid-request", "")
	requireStructuredFailure(t, runCommand(a.gobble, a.consumer, a.agentEnv, "pack", "./wgs", "-o", filepath.Join(t.TempDir(), "bad")), "pack with -o", 2, "invalid-request", "")
	requireStructuredFailure(t, runCommand(a.gobble, a.consumer, a.agentEnv, "pack", "./internal/hidden", "--output", filepath.Join(t.TempDir(), "internal")), "pack internal package", 2, "invalid-request", "")
	hiddenOutput := filepath.Join(t.TempDir(), "hidden-go")
	hiddenEnv := replaceEnv(os.Environ(), map[string]string{"PATH": t.TempDir()})
	requireStructuredFailure(t, runCommand(a.gobble, a.consumer, hiddenEnv, "pack", "./wgs", "--output", hiddenOutput), "pack with go hidden", 2, "invalid-request", "")
	if _, err := os.Stat(hiddenOutput); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("pack with go hidden created output: %v", err)
	}
}

func packRunner(t *testing.T, a *assay, pkg, output string) {
	t.Helper()
	requireCommand(t, runCommand(a.gobble, a.consumer, a.agentEnv, "pack", pkg, "--output", output), "pack "+pkg)
	requireExecutable(t, output)
	requireCleanGit(t, a.gitBin, a.consumer)
}

func provePackedMetadata(t *testing.T, a *assay) {
	t.Helper()
	version := requireCommand(t, runCommand(a.packedWGS, t.TempDir(), a.neutralEnv, "version"), "packed WGS version")
	var body map[string]any
	if err := json.Unmarshal(version.Stdout, &body); err != nil {
		t.Fatalf("packed version JSON: %v\n%s", err, version.Stdout)
	}
	want := map[string]any{
		"process": "packed-runner", "install_kind": "packed-runner", "identity_mode": "local-pin",
		"vcs_revision": a.snapshotCommit, "pipeline_vcs_revision": a.consumerCommit,
		"goos": "linux", "goarch": "amd64", "executable_sha256": "",
	}
	for key, value := range want {
		if body[key] != value {
			t.Fatalf("packed version %s = %#v, want %#v; body=%#v", key, body[key], value, body)
		}
	}
}

func proveHelpHonesty(t *testing.T, a *assay) {
	t.Helper()
	packedRoot := requireCommand(t, runCommand(a.packedWorkflow, t.TempDir(), a.neutralEnv, "help"), "packed root help")
	packedCommand := requireCommand(t, runCommand(a.packedWorkflow, t.TempDir(), a.neutralEnv, "help", "compose"), "packed command help")
	genericRoot := requireCommand(t, runCommand(a.gobble, t.TempDir(), a.neutralEnv, "help"), "generic root help")
	genericPack := requireCommand(t, runCommand(a.gobble, t.TempDir(), a.neutralEnv, "help", "pack"), "generic pack help")
	packedRootLower := strings.ToLower(strings.Join(strings.Fields(string(packedRoot.Stdout)), " "))
	genericRootLower := strings.ToLower(strings.Join(strings.Fields(string(genericRoot.Stdout)), " "))
	if strings.Contains(packedRootLower, "[package]") || strings.Contains(packedRootLower, "  pack ") || strings.Contains(packedRootLower, "gobble pack") {
		t.Fatalf("packed help exposes package operand or pack: %s", packedRoot.Stdout)
	}
	licenseText := mustReadFile(t, filepath.Join(a.root, "LICENSE"))
	if !bytes.Contains(packedRoot.Stdout, licenseText) {
		t.Fatalf("packed root help omits Gobble's exact MIT notice:\n%s", packedRoot.Stdout)
	}
	for name, output := range map[string][]byte{
		"generic pack":   genericPack.Stdout,
		"packed root":    packedRoot.Stdout,
		"packed command": packedCommand.Stdout,
	} {
		lower := strings.ToLower(strings.Join(strings.Fields(string(output)), " "))
		for _, want := range []string{"gobble portions", "mit", "embedded pipeline", "different license"} {
			if !strings.Contains(lower, want) {
				t.Fatalf("%s help omits %q:\n%s", name, want, output)
			}
		}
		for _, forbidden := range []string{"license is unset", "runner is licensed under mit", "it is licensed under mit"} {
			if strings.Contains(lower, forbidden) {
				t.Fatalf("%s help contains %q:\n%s", name, forbidden, output)
			}
		}
	}
	if !strings.Contains(genericRootLower, "gobble is licensed under mit") || strings.Contains(genericRootLower, "license is unset") {
		t.Fatalf("generic root help misstates Gobble's license:\n%s", genericRoot.Stdout)
	}
}

func proveAPIDigestMismatch(t *testing.T, a *assay, workspace string, required identityFields) {
	t.Helper()
	neutralDir := t.TempDir()
	before := workspaceInventory(t, workspace)
	identityResult := runCommand(a.gobble, neutralDir, a.neutralEnv, "inspect", "identity", "--workspace", workspace)
	identity := inspectIdentity(t, identityResult, "installed inspect identity on API workspace")
	if identity.Match || identity.Required.GobbleExecutableSHA256 != required.GobbleExecutableSHA256 || identity.Have.GobbleExecutableSHA256 != a.gobbleSHA256 {
		t.Fatalf("API digest mismatch header = %#v, required API=%s installed=%s", identity, required.GobbleExecutableSHA256, a.gobbleSHA256)
	}
	requireStructuredFailure(t, runCommand(a.gobble, neutralDir, a.neutralEnv, "inspect", "run", "--workspace", workspace), "installed full Inspect on API workspace", 1, "identity-mismatch", "gobble")
	requireStructuredFailure(t, runCommand(a.gobble, neutralDir, a.neutralEnv, "release", "--workspace", workspace), "installed Release on API workspace", 1, "identity-mismatch", "gobble")
	after := workspaceInventory(t, workspace)
	if after != before {
		t.Fatalf("API digest mismatch mutated workspace: before=%s after=%s", before, after)
	}
	requireNoGoInvocation(t, a.goStubMarker, "API digest mismatch checks")
}

func provePackedCrossKindMismatch(t *testing.T, a *assay, workspace string) {
	t.Helper()
	neutralDir := t.TempDir()
	before := workspaceInventory(t, workspace)
	identity := inspectIdentity(t, runCommand(a.gobble, neutralDir, a.neutralEnv, "inspect", "identity", "--workspace", workspace), "installed inspect identity on packed workspace")
	if identity.Match || identity.Required.InstallKind != "packed-runner" || identity.Have.InstallKind != "module" {
		t.Fatalf("packed cross-kind header = %#v", identity)
	}
	requireStructuredFailure(t, runCommand(a.gobble, neutralDir, a.neutralEnv, "inspect", "run", "--workspace", workspace), "installed full Inspect on packed workspace", 1, "identity-mismatch", "install-kind")
	requireStructuredFailure(t, runCommand(a.gobble, neutralDir, a.neutralEnv, "release", "--workspace", workspace), "installed Release on packed workspace", 1, "identity-mismatch", "install-kind")
	after := workspaceInventory(t, workspace)
	if after != before {
		t.Fatalf("packed cross-kind mismatch mutated workspace: before=%s after=%s", before, after)
	}
	requireNoGoInvocation(t, a.goStubMarker, "packed cross-kind checks")
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return data
}
