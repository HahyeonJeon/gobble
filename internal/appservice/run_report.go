package appservice

import (
	"context"
	"encoding/base64"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const maxReportBytes = 1 << 20

// Native transport deliberately does not import the Linux engine implementation.
type outputEvidence struct {
	SchemaVersion int    `json:"schemaVersion"`
	RunID         string `json:"runId"`
	Snapshot      string `json:"snapshot"`
	OriginDigest  string `json:"originDigest"`
	Instance      string `json:"instance"`
	Attempt       int    `json:"attempt"`
	Port          string `json:"port"`
	Recipe        string `json:"recipe,omitempty"`
	Path          string `json:"path"`
	SHA256        string `json:"sha256"`
	Size          int64  `json:"size"`
}
type runReport struct {
	ProjectID  string         `json:"projectId"`
	RunRef     string         `json:"runRef"`
	ObservedAt int64          `json:"observedAt"`
	Evidence   outputEvidence `json:"evidence"`
	Base64     string         `json:"base64"`
}

// readRunReport returns only the same bounded bytes that matched engine evidence.
// Capturing and rendering the report belong to Electron Main, not this service.
func (s *Service) readRunReport(ctx context.Context, project, runRef, instance string, attempt int) (runReport, error) {
	if !validInstance(instance) || attempt < 1 {
		return runReport{}, problem("invalid_request", "A task instance and positive attempt are required.")
	}
	p, err := s.store.project(project)
	if err != nil {
		return runReport{}, err
	}
	s.store.mu.Lock()
	var run runRecord
	for _, v := range s.store.data.Runs {
		if v.ProjectID == project && v.RunRef == runRef {
			run = v
			break
		}
	}
	s.store.mu.Unlock()
	if run.RunRef == "" {
		return runReport{}, problem("not_found", "Run not found in this Project.")
	}
	original, err := s.reportLaunch(project, runRef)
	if err != nil {
		return runReport{}, err
	}
	workspace := p.Resources[run.WorkspaceResourceID]
	if !validRelative(workspace) || original.Value.OutputPath != workspace || run.Binding != original.Runtime || run.Binding.WorkspacePath != workspaceContainerPath(workspace) {
		return runReport{}, problem("stale_revision", "The registered result location or runtime changed.")
	}
	if err = launchTargetIdentity(p, original); err != nil {
		return runReport{}, err
	}
	if err = s.runtime.verify(ctx, run.Binding); err != nil {
		return runReport{}, err
	}
	identity, err := s.runtime.inspect(ctx, p, run.Binding, "identity", "")
	if err != nil {
		return runReport{}, err
	}
	required, err := readIdentity(identity)
	if err != nil {
		return runReport{}, err
	}
	if requestDigest(required) != requestDigest(run.Identity) {
		return runReport{}, problem("stale_revision", "The workspace engine identity has changed.")
	}
	// A label gates the new query for old engines; its versioned response is still
	// checked. Existing strict prepared-capabilities decoding remains untouched.
	label, err := s.runtime.commandAt(ctx, run.Binding.Endpoint, "image", "inspect", "--format", `{{index .Config.Labels "io.gobble.output-evidence.version"}}`, run.Binding.ImageID)
	if err != nil {
		return runReport{}, err
	}
	if strings.TrimSpace(string(label)) != "1" {
		return runReport{}, problem("unsupported", "This Run's engine does not support saved reports.")
	}
	raw, err := s.reportQuery(ctx, p, original, "output-capabilities")
	if err != nil {
		return runReport{}, err
	}
	var caps struct {
		SchemaVersion         int `json:"schemaVersion"`
		OutputEvidenceVersion int `json:"outputEvidenceVersion"`
	}
	if decodeCreationJSON(raw, &caps) != nil || caps.SchemaVersion != 1 || caps.OutputEvidenceVersion != 1 {
		return runReport{}, problem("incompatible_runtime", "The output evidence protocol is unsupported.")
	}
	read := func() (outputEvidence, error) {
		raw, err := s.reportQuery(ctx, p, original, "output-evidence", "--workspace", run.Binding.WorkspacePath, run.EngineRunID, instance, strconv.Itoa(attempt), "html")
		if err != nil {
			return outputEvidence{}, reportQueryError(err)
		}
		var v outputEvidence
		if decodeCreationJSON(raw, &v) != nil || v.SchemaVersion != 1 || v.RunID != run.EngineRunID || !continuationLeasePattern.MatchString(v.Snapshot) || v.OriginDigest != launchIntentDigest(original.Intent) || v.Instance != instance || v.Attempt != attempt || v.Port != "html" || v.Recipe != "fastqc-v1" || !validRelative(v.Path) || v.Path == "." || !digestPattern.MatchString(v.SHA256) || v.Size < 0 {
			return v, problem("incompatible_runtime", "The report evidence does not match this exact quality step.")
		}
		return v, nil
	}
	evidence, err := read()
	if err != nil {
		return runReport{}, err
	}
	data, err := readReportBytes(p, workspace, evidence)
	if err != nil {
		return runReport{}, err
	}
	// A newer task attempt must not be mistaken for the producer just captured.
	// Status-only checkpoints are allowed when the entire producer tuple is stable.
	after, err := read()
	if err != nil {
		return runReport{}, err
	}
	before := evidence
	before.Snapshot = after.Snapshot
	if before != after {
		return runReport{}, problem("stale_revision", "The report producer changed while reading. Open the result again.")
	}
	if err = launchTargetIdentity(p, original); err != nil {
		return runReport{}, err
	}
	if err = ctx.Err(); err != nil {
		return runReport{}, problem("runtime_unavailable", "The report read was interrupted.")
	}
	return runReport{project, runRef, time.Now().UnixMilli(), evidence, base64.StdEncoding.EncodeToString(data)}, nil
}

func (s *Service) reportLaunch(project, runRef string) (launchRecord, error) {
	entries, err := os.ReadDir(s.launchDirectory(project, ""))
	if err != nil {
		return launchRecord{}, problem("unsupported", "This Run has no retained App launch for report verification.")
	}
	var found launchRecord
	for _, entry := range entries {
		if !entry.IsDir() || !validID(entry.Name(), "req") {
			continue
		}
		rec, err := s.readLaunch(project, entry.Name())
		if err != nil {
			return launchRecord{}, err
		}
		if op := rec.Value.Operation; op != nil && op.RunRef == runRef && op.State == "admitted" {
			if found.Value.RequestID != "" {
				return launchRecord{}, problem("internal", "The Run has ambiguous launch records.")
			}
			found = rec
		}
	}
	if found.Value.RequestID == "" {
		return found, problem("unsupported", "This Run has no retained App launch for report verification.")
	}
	return found, nil
}

func (s *Service) reportQuery(ctx context.Context, p projectRecord, rec launchRecord, args ...string) ([]byte, error) {
	name := "gobble-output-query-" + newID("req")
	defer s.removeRuntimeQuery(rec.Runtime, name)
	command := []string{"run", "--rm", "--pull=never", "--name", name, "--platform", "linux/amd64", "--network=none", "--read-only", "--cap-drop=ALL", "--security-opt=no-new-privileges", "--user", "0:0", "--pids-limit=32", "--memory=256m", "--cpus=1", "--env", "GOBBLE_CONTAINER_BOOTSTRAP=0", "--mount", readOnlyMount(filepath.Join(p.Root, filepath.FromSlash(rec.Value.OutputPath)), rec.Runtime.WorkspacePath), "--mount", readOnlyMount(s.launchBundle(rec), "/gobble/launch"), "--entrypoint", "/usr/local/bin/gobble", rec.Runtime.ImageID}
	return s.runtime.commandAt(ctx, rec.Runtime.Endpoint, append(command, args...)...)
}

func readReportBytes(p projectRecord, workspace string, e outputEvidence) ([]byte, error) {
	if e.Size > maxReportBytes {
		return nil, problem("unsupported", "This report exceeds the 1 MiB limit.")
	}
	root, err := openProject(p)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	// Narrow the file handle's containment to this Run, not the entire Project.
	runRoot, err := root.OpenRoot(workspace)
	if err != nil {
		return nil, fileAccessError(err)
	}
	defer runRoot.Close()
	file, err := openReportSource(runRoot, e.Path)
	if err != nil {
		return nil, fileAccessError(err)
	}
	defer file.Close()
	before, err := file.Stat()
	if err != nil {
		return nil, fileAccessError(err)
	}
	if !before.Mode().IsRegular() {
		return nil, problem("unsupported", "The report source must be a regular file.")
	}
	if before.Size() != e.Size {
		return nil, problem("stale_revision", "The report file has changed since execution.")
	}
	data, err := io.ReadAll(io.LimitReader(file, maxReportBytes+1))
	if err != nil {
		return nil, fileAccessError(err)
	}
	if len(data) > maxReportBytes {
		return nil, problem("unsupported", "This report exceeds the 1 MiB limit.")
	}
	return finishReportRead(runRoot, file, before, e, data)
}

// finishReportRead checks the open descriptor and current no-follow pathname
// after acquisition. Callers receive only bytes that passed both revision checks.
func finishReportRead(root *os.Root, file *os.File, before os.FileInfo, e outputEvidence, data []byte) ([]byte, error) {
	after, err := file.Stat()
	if err != nil {
		return nil, fileAccessError(err)
	}
	named, err := openReportSource(root, e.Path)
	if err != nil {
		return nil, problem("stale_revision", "The report file changed while reading. Open the result again.")
	}
	defer named.Close()
	current, err := named.Stat()
	if err != nil || !current.Mode().IsRegular() || !os.SameFile(before, current) || !os.SameFile(before, after) || before.Size() != after.Size() || !before.ModTime().Equal(after.ModTime()) || before.Size() != current.Size() || !before.ModTime().Equal(current.ModTime()) {
		return nil, problem("stale_revision", "The report file changed while reading. Open the result again.")
	}
	if int64(len(data)) != e.Size || digest(data) != e.SHA256 {
		return nil, problem("stale_revision", "The report content has changed since execution.")
	}
	return data, nil
}

func reportQueryError(err error) error {
	var failure *runtimeCommandError
	if !errors.As(err, &failure) {
		return err
	}
	var diagnostic struct {
		Op      string `json:"op"`
		Defects []struct {
			Code    string   `json:"code"`
			Unit    string   `json:"unit"`
			Message string   `json:"message"`
			Paths   []string `json:"paths"`
		} `json:"defects"`
	}
	if decodeCreationJSON(failure.diagnostic, &diagnostic) != nil || diagnostic.Op != "cli" || len(diagnostic.Defects) != 1 {
		return err
	}
	d := diagnostic.Defects[0]
	if d.Code != "invalid-request" || d.Unit != "" || len(d.Paths) != 0 {
		return err
	}
	switch d.Message {
	case "output attempt is no longer current", "output Run origin changed", "output saved plan changed", "output task identity changed", "output engine identity changed":
		return problem("stale_revision", "The selected report's execution evidence changed. Refresh the Run.")
	case "output attempt has not succeeded":
		return problem("invalid_request", "The selected step has not completed successfully.")
	case "output task is unavailable", "output port is unavailable", "output checksum is unavailable", "output state is unavailable":
		return problem("not_found", "The selected report has no available recorded output evidence.")
	case "output attempts are ambiguous", "output declaration is ambiguous", "output checksums are ambiguous", "output declaration is unsupported":
		return problem("incompatible_runtime", "The recorded report evidence is unsupported or inconsistent.")
	}
	return err
}
