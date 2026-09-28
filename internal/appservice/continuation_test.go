package appservice

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/HahyeonJeon/gobble/internal/preparation"
)

type continuationRuntimeFixture struct {
	mu              sync.Mutex
	review          preparation.ContinuationReview
	receipt         *preparation.ContinuationAdmission
	original        launchRecord
	commands        [][]string
	support         bool
	controller      map[string]any
	intended        *preparation.ContinuationIntent
	beforeCreate    func()
	rejectAdmission bool
}

func (f *continuationRuntimeFixture) command(ctx context.Context, args, env []string) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.commands = append(f.commands, append([]string(nil), args...))
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	joined := strings.Join(args, " ")
	switch {
	case args[0] == "info":
		return []byte(f.original.Runtime.DaemonID + " linux"), nil
	case args[0] == "image" && strings.Contains(joined, "continuation.scope"):
		if f.support {
			return []byte("single-end-trim-fastqc-v1"), nil
		}
		return []byte("<no value>"), nil
	case args[0] == "image":
		return []byte(f.original.Runtime.ImageID + " linux/amd64"), nil
	case strings.Contains(joined, "prepared-capabilities"):
		return []byte(`{"schemaVersion":1,"launchSchemaVersion":2,"continuationVersion":1}`), nil
	case strings.Contains(joined, "prepared-continuation-review"):
		return json.Marshal(f.review)
	case strings.Contains(joined, "prepared-continuation-receipt"):
		return json.Marshal(struct {
			SchemaVersion int                                `json:"schemaVersion"`
			Found         bool                               `json:"found"`
			Receipt       *preparation.ContinuationAdmission `json:"receipt,omitempty"`
		}{1, f.receipt != nil, f.receipt})
	case strings.Contains(joined, "prepared-status"):
		origin := preparation.Admission{SchemaVersion: 2, IntentDigest: launchIntentDigest(f.original.Intent), RequestID: f.original.Intent.RequestID, PreparedDigest: f.original.Intent.PreparedDigest, WorkspaceID: f.original.Intent.WorkspaceID, Lease: strings.Repeat("a", 32)}
		epoch := &preparation.ExecutionEpoch{Head: origin.IntentDigest, Lease: origin.Lease}
		if f.receipt != nil {
			epoch = &preparation.ExecutionEpoch{Head: requestDigest(*f.receipt), Lease: f.receipt.Lease}
		}
		return json.Marshal(admittedState{SchemaVersion: 2, Admission: origin, Status: "running", Snapshot: strings.Repeat("d", 32), OwnerActive: true, OwnerLive: true, Epoch: epoch})
	case strings.Contains(joined, "prepared-stop"):
		return json.Marshal(map[string]string{"status": "settled", "lease": f.receipt.Lease})
	case args[0] == "rm":
		return nil, nil
	case args[0] == "create":
		if f.beforeCreate != nil {
			f.beforeCreate()
		}
		if f.controller != nil {
			return []byte("controller-id"), nil
		}
		return nil, errors.New("lost create acknowledgement")
	case args[0] == "inspect" && f.controller != nil:
		return json.Marshal(f.controller)
	case args[0] == "start" && f.intended != nil:
		if f.rejectAdmission {
			f.controller["State"] = map[string]string{"Status": "exited"}
			return []byte("exited without admission"), nil
		}
		f.receipt = &preparation.ContinuationAdmission{Intent: *f.intended, IntentDigest: requestDigest(*f.intended), Lease: strings.Repeat("b", 32), Snapshot: strings.Repeat("c", 32)}
		return []byte("started"), nil
	}
	return nil, errors.New("unexpected runtime command")
}
func continuationFixture(t *testing.T) (*Service, Project, launchRecord, *continuationRuntimeFixture) {
	t.Helper()
	s, p, original := readyLaunchFixture(t)
	original.Intent.SchemaVersion = 2
	original.Intent.RequestID = "req_start"
	raw := []byte(`{"sealed":"fixture"}`)
	original.Intent.PreparedDigest = digest(raw)
	original.Value.State = "accepted"
	original.Value.Fresh = false
	original.Value.Operation = &launchOperation{RequestID: "req_start", State: "admitted", RunRef: "run_saved", RunStatus: "stopped", OwnerLease: strings.Repeat("a", 32)}
	if err := atomicWrite(s.launchBundle(original), "payload.json", raw); err != nil {
		t.Fatal(err)
	}
	if err := s.saveLaunch(original); err != nil {
		t.Fatal(err)
	}
	review := preparation.ContinuationReview{SchemaVersion: 1, WorkspaceID: original.Intent.WorkspaceID, OriginDigest: launchIntentDigest(original.Intent), PreviousHead: launchIntentDigest(original.Intent), Snapshot: strings.Repeat("a", 32), StateDigest: digest([]byte("saved state")), Steps: []preparation.ContinuationStep{{TaskID: "trim", PriorAttempt: 1, PlannedAttempt: 1, Action: "reuse"}, {TaskID: "qc", PriorAttempt: 1, PlannedAttempt: 2, Action: "restart"}}, Files: []preparation.ContinuationFile{{Path: original.Intent.Binding.InputPath, SHA256: strings.TrimPrefix(original.Intent.InputSHA256, "sha256:")}}}
	review.Digest = requestDigest(review)
	f := &continuationRuntimeFixture{original: original, review: review, support: true}
	s.runtime.command = f.command
	return s, p, original, f
}
func awaitContinuation(t *testing.T, s *Service, project, review string) continuationView {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		s.mu.Lock()
		busy := s.continuationFlights[project+"/"+review] != nil
		s.mu.Unlock()
		if !busy {
			v, e := s.continuation(project, review)
			if e != nil {
				t.Fatal(e)
			}
			return v
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("continuation worker timed out")
	return continuationView{}
}
func checkedContinuationFixture(t *testing.T) (*Service, Project, launchRecord, *continuationRuntimeFixture, continuationView) {
	t.Helper()
	s, p, original, f := continuationFixture(t)
	if _, e := s.checkContinuation(p.ProjectID, continuationCheckInput{"req_review", original.Value.RequestID, "run_saved"}); e != nil {
		t.Fatal(e)
	}
	v := awaitContinuation(t, s, p.ProjectID, "req_review")
	if v.State != "ready" {
		t.Fatalf("review: %+v", v)
	}
	return s, p, original, f, v
}
func TestContinuationCheckStoresExactReadOnlyReview(t *testing.T) {
	s, p, original, f, v := checkedContinuationFixture(t)
	if !reflect.DeepEqual(*v.Review, f.review) {
		t.Fatal("review changed")
	}
	for _, args := range f.commands {
		if args[0] == "create" || args[0] == "start" {
			t.Fatal("check dispatched", args)
		}
		if strings.Contains(strings.Join(args, " "), "prepared-continuation-review") {
			target := filepath.Join(mustProject(t, s, p.ProjectID).Root, original.Value.OutputPath)
			if !strings.Contains(strings.Join(args, " "), readOnlyMount(target, original.Runtime.WorkspacePath)) {
				t.Fatal("review workspace writable")
			}
		}
	}
	_, e := s.checkContinuation(p.ProjectID, continuationCheckInput{"req_review", original.Value.RequestID, "run_other"})
	wantCode(t, e, "request_conflict")
	_, e = s.confirmContinuation(p.ProjectID, "req_review", continuationConfirmInput{"req_continue", digest([]byte("different"))})
	wantCode(t, e, "stale_revision")
}
func mustProject(t *testing.T, s *Service, id string) projectRecord {
	t.Helper()
	p, e := s.store.project(id)
	if e != nil {
		t.Fatal(e)
	}
	return p
}
func TestContinuationUnknownAcknowledgementRestoresWithoutDispatch(t *testing.T) {
	s, p, original, f, v := checkedContinuationFixture(t)
	accepted, e := s.confirmContinuation(p.ProjectID, v.RequestID, continuationConfirmInput{"req_continue", v.Review.Digest})
	if e != nil || accepted.Operation.State != "accepted" {
		t.Fatal(accepted, e)
	}
	unknown := awaitContinuation(t, s, p.ProjectID, v.RequestID)
	if unknown.Operation.State != "unknown" {
		t.Fatal(unknown)
	}
	var creates int
	for _, args := range f.commands {
		if args[0] == "create" {
			creates++
		}
	}
	if creates != 1 {
		t.Fatal("expected single dispatch", creates)
	}
	profile := filepath.Dir(s.store.dir)
	if e = s.Close(); e != nil {
		t.Fatal(e)
	}
	reopened, e := Open(t.Context(), profile)
	if e != nil {
		t.Fatal(e)
	}
	defer reopened.Close()
	reopened.runtime.command = f.command
	restored, e := reopened.continuations(p.ProjectID)
	if e != nil || len(restored) != 1 || restored[0].Operation.RequestID != "req_continue" {
		t.Fatal(restored, e)
	}
	for range 2 {
		if _, e = reopened.refreshContinuation(p.ProjectID, v.RequestID); e != nil {
			t.Fatal(e)
		}
		awaitContinuation(t, reopened, p.ProjectID, v.RequestID)
	}
	_, e = reopened.confirmContinuation(p.ProjectID, v.RequestID, continuationConfirmInput{"req_continue", v.Review.Digest})
	if e != nil {
		t.Fatal(e)
	}
	afterCreates := 0
	for _, args := range f.commands {
		if args[0] == "create" || args[0] == "start" {
			afterCreates++
		}
	}
	if afterCreates != 1 {
		t.Fatal("restoration dispatched again", afterCreates)
	}
	rec, e := reopened.readContinuation(p.ProjectID, v.RequestID)
	if e != nil {
		t.Fatal(e)
	}
	f.receipt = &preparation.ContinuationAdmission{Intent: *rec.Intent, IntentDigest: requestDigest(*rec.Intent), Lease: strings.Repeat("b", 32), Snapshot: strings.Repeat("c", 32)}
	if _, e = reopened.refreshContinuation(p.ProjectID, v.RequestID); e != nil {
		t.Fatal(e)
	}
	admitted := awaitContinuation(t, reopened, p.ProjectID, v.RequestID)
	if admitted.Operation.State != "admitted" || admitted.Operation.Observation.Epoch.Lease != f.receipt.Lease {
		t.Fatal(admitted)
	}
	before, e := reopened.readLaunch(p.ProjectID, original.Value.RequestID)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = reopened.stopContinuation(p.ProjectID, v.RequestID, launchStopInput{"req_stop", original.Value.Operation.OwnerLease}); e == nil {
		t.Fatal("old lease Stop accepted")
	}
	if _, e = reopened.stopContinuation(p.ProjectID, v.RequestID, launchStopInput{"req_stop", f.receipt.Lease}); e != nil {
		t.Fatal(e)
	}
	stopped := awaitContinuation(t, reopened, p.ProjectID, v.RequestID)
	if stopped.Operation.StopState != "settled" {
		t.Fatal(stopped)
	}
	after, e := reopened.readLaunch(p.ProjectID, original.Value.RequestID)
	if e != nil || !reflect.DeepEqual(before, after) {
		t.Fatal("continuation changed original Start", e)
	}
	retained := *stopped.Operation.Receipt
	f.receipt = nil
	if _, e = reopened.refreshContinuation(p.ProjectID, v.RequestID); e != nil {
		t.Fatal(e)
	}
	unavailable := awaitContinuation(t, reopened, p.ProjectID, v.RequestID)
	if unavailable.Operation.State != "admitted" || unavailable.Operation.Receipt == nil || *unavailable.Operation.Receipt != retained || unavailable.Operation.Issue == "" {
		t.Fatal("temporary observation erased admitted authority", unavailable)
	}
}
func TestContinuationRejectsUnavailableAndCrossRunEvidence(t *testing.T) {
	for _, scenario := range []string{"earlier", "unsupported", "different review", "corrupt"} {
		t.Run(scenario, func(t *testing.T) {
			s, p, original, f := continuationFixture(t)
			switch scenario {
			case "earlier":
				original.Intent.SchemaVersion = 1
				if e := s.saveLaunch(original); e != nil {
					t.Fatal(e)
				}
			case "unsupported":
				f.support = false
			case "different review":
				f.review.WorkspaceID = "req_other"
				f.review.Digest = ""
				f.review.Digest = requestDigest(f.review)
			case "corrupt":
				f.review.Digest = digest([]byte("not review"))
			}
			if _, e := s.checkContinuation(p.ProjectID, continuationCheckInput{"req_review", original.Value.RequestID, "run_saved"}); e != nil {
				t.Fatal(e)
			}
			v := awaitContinuation(t, s, p.ProjectID, "req_review")
			if v.State != "blocked" || v.Review != nil {
				t.Fatal(v)
			}
			for _, args := range f.commands {
				if args[0] == "create" || args[0] == "start" {
					t.Fatal("blocked review executed")
				}
			}
		})
	}
}
func TestContinuationCorruptSavedRecordNeverDispatches(t *testing.T) {
	s, p, _, _, v := checkedContinuationFixture(t)
	path := filepath.Join(s.continuationDirectory(p.ProjectID, v.RequestID), "record.json")
	b, e := os.ReadFile(path)
	if e != nil {
		t.Fatal(e)
	}
	b = []byte(strings.Replace(string(b), `"run_saved"`, `"run_other"`, 1))
	if e = os.WriteFile(path, b, 0600); e != nil {
		t.Fatal(e)
	}
	if _, e = s.confirmContinuation(p.ProjectID, v.RequestID, continuationConfirmInput{"req_continue", v.Review.Digest}); e == nil {
		t.Fatal("corrupt intent accepted")
	}
}

func TestContinuationDispatchBindsExactControllerBeforeStart(t *testing.T) {
	for _, scenario := range []string{"exact", "bundle-replaced", "rejected"} {
		t.Run(scenario, func(t *testing.T) {
			changed := scenario == "bundle-replaced"
			s, p, original, f, v := checkedContinuationFixture(t)
			in := continuationIntentFor(v, "req_continue")
			f.intended = &in
			f.rejectAdmission = scenario == "rejected"
			rec := continuationRecord{Value: v, Intent: &in}
			spec := s.continuationController(rec)
			root := mustProject(t, s, p.ProjectID).Root
			mounts := []map[string]any{{"Source": filepath.Join(root, original.Value.OutputPath), "Destination": original.Runtime.WorkspacePath, "RW": true}, {"Source": filepath.Join(root, original.Value.OutputPath, original.Intent.Binding.InputPath), "Destination": original.Runtime.WorkspacePath + "/" + original.Intent.Binding.InputPath, "RW": false}, {"Source": spec.bundle, "Destination": "/gobble/launch", "RW": false}, {"Source": launchSocket(original.Runtime.Endpoint), "Destination": "/var/run/docker.sock", "RW": true}}
			if changed {
				mounts[2]["Source"] = "/other/bundle"
			}
			f.controller = map[string]any{"Image": original.Runtime.ImageID, "Config": map[string]any{"Image": original.Runtime.ImageID, "User": "0:0", "Env": []string{"GOBBLE_CONTAINER_BOOTSTRAP=1", "GOBBLE_CONTROLLER=" + spec.name, "DOCKER_HOST=unix:///var/run/docker.sock"}, "WorkingDir": original.Runtime.WorkspacePath, "Entrypoint": []string{"/usr/local/bin/gobble"}, "Cmd": []string{"prepared-continue", "--workspace", original.Runtime.WorkspacePath}, "Labels": map[string]string{"io.gobble.launch": spec.digest}}, "HostConfig": map[string]any{"NetworkMode": "none", "ReadonlyRootfs": true, "CapDrop": []string{"ALL"}, "CapAdd": []string{"CAP_SETUID", "CAP_SETGID"}, "SecurityOpt": []string{"no-new-privileges"}}, "State": map[string]string{"Status": "created"}, "Mounts": mounts}
			f.beforeCreate = func() {
				saved, e := s.readContinuation(p.ProjectID, v.RequestID)
				if e != nil || saved.Intent == nil || *saved.Intent != in || saved.Value.Operation.State != "dispatching" {
					t.Error("create before durable exact intent", e)
				}
			}
			if _, e := s.confirmContinuation(p.ProjectID, v.RequestID, continuationConfirmInput{in.RequestID, v.Review.Digest}); e != nil {
				t.Fatal(e)
			}
			got := awaitContinuation(t, s, p.ProjectID, v.RequestID)
			starts := 0
			for _, args := range f.commands {
				if args[0] == "start" {
					starts++
				}
				if args[0] == "create" {
					all := strings.Join(args, " ")
					if !strings.Contains(all, spec.digest) || !strings.Contains(all, spec.bundle) || !strings.Contains(all, "--network=none") || !strings.Contains(all, "prepared-continue") {
						t.Error("controller lost exact binding")
					}
				}
			}
			if changed {
				if starts != 0 || got.Operation.State != "unknown" {
					t.Fatal("changed controller started", got)
				}
			} else if scenario == "rejected" {
				if starts != 1 || got.Operation.State != "rejected" {
					t.Fatal("terminal non-admission stayed uncertain", got)
				}
				if _, e := s.checkContinuation(p.ProjectID, continuationCheckInput{"req_recheck", original.Value.RequestID, "run_saved"}); e != nil {
					t.Fatal("proved rejection blocked recheck", e)
				}
				awaitContinuation(t, s, p.ProjectID, "req_recheck")
			} else {
				if starts != 1 || got.Operation.State != "admitted" {
					t.Fatal("exact controller not admitted", got)
				}
			}
		})
	}
}

func TestContinuationRejectsProtocolAndEpochMismatch(t *testing.T) {
	for _, scenario := range []string{"capability-version", "missing-epoch", "schema-downgrade"} {
		t.Run(scenario, func(t *testing.T) {
			s, p, original, f := continuationFixture(t)
			s.runtime.command = func(ctx context.Context, args, env []string) ([]byte, error) {
				joined := strings.Join(args, " ")
				if scenario == "capability-version" && strings.Contains(joined, "prepared-capabilities") {
					return []byte(`{"schemaVersion":1,"launchSchemaVersion":1,"continuationVersion":1}`), nil
				}
				raw, e := f.command(ctx, args, env)
				if e == nil && strings.Contains(joined, "prepared-status") {
					var state admittedState
					if e = json.Unmarshal(raw, &state); e != nil {
						return nil, e
					}
					if scenario == "missing-epoch" {
						state.Epoch = nil
					} else {
						state.SchemaVersion = 1
						state.Admission.SchemaVersion = 1
						state.Epoch = nil
					}
					return json.Marshal(state)
				}
				return raw, e
			}
			if scenario == "capability-version" {
				if _, e := s.engineContinuationSupport(t.Context(), original.Runtime); e == nil {
					t.Fatal("mismatched capability accepted")
				}
			} else {
				if _, e := s.queryLaunchState(t.Context(), mustProject(t, s, p.ProjectID), original); e == nil {
					t.Fatal("mismatched epoch accepted")
				}
			}
		})
	}
}
