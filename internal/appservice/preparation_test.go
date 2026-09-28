package appservice

import (
	"context"
	"encoding/json"
	"github.com/HahyeonJeon/gobble/internal/preparation"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func preparationFixture(t *testing.T, delayed bool) (*Service, Project, string, string) {
	t.Helper()
	s, p, scope, in := firstAdoptionFixture(t)
	adopted, err := s.adoptCreation(context.Background(), p.ProjectID, scope.Draft.DraftID, in)
	if err != nil {
		t.Fatal(err)
	}
	s.runtime.command = inspectionCommand(func(ctx context.Context, args []string) ([]byte, error) {
		if delayed {
			<-ctx.Done()
			return nil, ctx.Err()
		}
		var dir string
		for _, a := range args {
			if strings.HasPrefix(a, "type=bind,src=") {
				dir = strings.Split(strings.TrimPrefix(a, "type=bind,src="), ",dst=")[0]
			}
		}
		if args[len(args)-1] == "prepared-review" {
			var intent struct {
				Digest  string              `json:"digest"`
				Binding preparation.Binding `json:"binding"`
			}
			if e := readProposalFile(filepath.Join(dir, "intent.json"), &intent); e != nil {
				return nil, e
			}
			var review creationCheck
			_ = json.Unmarshal(mockCreationCheck(), &review)
			return json.Marshal(verifiedPreparation{1, intent.Digest, intent.Binding, review.Review.Flow, []preparation.Step{{ID: "trim", Label: "Trim reads", Outputs: []string{}}, {ID: "qc", Label: "Quality check", Outputs: []string{}}}})
		}
		var b preparation.Binding
		if err := readProposalFile(filepath.Join(dir, "gobble-preparation.json"), &b); err != nil {
			return nil, err
		}
		var review creationCheck
		_ = json.Unmarshal(mockCreationCheck(), &review)
		payload, _ := json.Marshal(map[string]any{"schemaVersion": 1, "binding": b, "document": map[string]any{"private": "execution value"}})
		return json.Marshal(preparedResponse{1, digest(payload), payload, b, review, []preparation.Step{{ID: "trim", Label: "Trim reads", Outputs: []string{}}, {ID: "qc", Label: "Quality check", Outputs: []string{}}}})
	})
	return s, p, adopted.Adoption.PipelineID, in.ArtifactID
}
func awaitPreparation(t *testing.T, s *Service, p, pipeline, request string) pipelinePreparation {
	t.Helper()
	deadline := time.Now().Add(inspectionTimeout + 10*time.Second)
	for time.Now().Before(deadline) {
		v, e := s.preparation(p, pipeline, request)
		if e != nil {
			t.Fatal(e)
		}
		if v.State != "preparing" {
			return v
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("preparation did not finish")
	return pipelinePreparation{}
}
func TestPreparationReplayPrivacyFreshnessAndRestart(t *testing.T) {
	s, p, id, artifact := preparationFixture(t, false)
	v, e := s.preparePipeline(p.ProjectID, id, prepareInput{RequestID: "req_prepare", ArtifactID: artifact})
	if e != nil || v.State != "preparing" {
		t.Fatal(v, e)
	}
	same, e := s.preparePipeline(p.ProjectID, id, prepareInput{RequestID: "req_prepare", ArtifactID: artifact})
	if e != nil || same.RequestID != v.RequestID {
		t.Fatal("retry", e)
	}
	_, e = s.preparePipeline(p.ProjectID, id, prepareInput{RequestID: "req_prepare", ArtifactID: digest([]byte("other"))})
	wantCode(t, e, "request_conflict")
	ready := awaitPreparation(t, s, p.ProjectID, id, v.RequestID)
	if ready.State != "ready" || !ready.Fresh {
		t.Fatal(ready)
	}
	raw, _ := json.Marshal(ready)
	if strings.Contains(string(raw), "private") || strings.Contains(string(raw), "payload") {
		t.Fatal("private bytes leaked")
	}
	info, e := os.Stat(filepath.Join(s.preparationDirectory(id, v.RequestID), "payload.json"))
	if e != nil || info.Mode().Perm()&0077 != 0 {
		t.Fatal("private payload permission", e)
	}
	if len(s.store.data.Runs) != 0 {
		t.Fatal("preparation launched Run")
	}
	s.store.mu.Lock()
	project := s.store.data.Projects[0]
	s.store.mu.Unlock()
	writeTestFile(t, filepath.Join(project.Root, ready.Prepared.Input.RelativePath), []byte("new data contents"))
	earlier, e := s.preparation(p.ProjectID, id, v.RequestID)
	if e != nil || earlier.Fresh || earlier.Prepared.Digest != ready.Prepared.Digest {
		t.Fatal("old observation lost", e)
	}
	profile := filepath.Dir(s.store.dir)
	if e = s.Close(); e != nil {
		t.Fatal(e)
	}
	reopened, e := Open(context.Background(), profile)
	if e != nil {
		t.Fatal(e)
	}
	defer reopened.Close()
	saved, e := reopened.preparation(p.ProjectID, id, v.RequestID)
	if e != nil || saved.State != "ready" || saved.Fresh {
		t.Fatal("restart", saved, e)
	}
	_, e = reopened.preparePipeline(p.ProjectID, id, prepareInput{RequestID: v.RequestID, ArtifactID: artifact})
	if e != nil {
		t.Fatal("replay after restart", e)
	}
	writeTestFile(t, filepath.Join(reopened.preparationDirectory(id, v.RequestID), "payload.json"), []byte("tampered"))
	_, e = reopened.preparation(p.ProjectID, id, v.RequestID)
	wantCode(t, e, "internal")
}
func TestPreparationCancellationAndForeignProject(t *testing.T) {
	s, p, id, artifact := preparationFixture(t, true)
	other := testProject(t, s, t.TempDir())
	_, e := s.preparePipeline(other.ProjectID, id, prepareInput{RequestID: "req_foreign", ArtifactID: artifact})
	wantCode(t, e, "not_found")
	_, e = s.preparePipeline(p.ProjectID, id, prepareInput{RequestID: "req_cancel", ArtifactID: artifact})
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.cancelPreparation(p.ProjectID, id, "req_cancel"); e != nil {
		t.Fatal(e)
	}
	v := awaitPreparation(t, s, p.ProjectID, id, "req_cancel")
	if v.State != "cancelled" || v.Prepared != nil {
		t.Fatal(v)
	}
	if len(s.store.data.Runs) != 0 {
		t.Fatal("cancel created run")
	}
}

func TestPreparationRefusesSourceDrift(t *testing.T) {
	s, p, id, artifact := preparationFixture(t, false)
	s.mu.Lock()
	base, _, _, err := s.preparationSourceLocked(p.ProjectID, id, artifact)
	s.mu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	sourcePath := filepath.Join(base.Directory, base.Manifest.Files[0].Path)
	if err = os.Chmod(sourcePath, 0600); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, sourcePath, []byte("changed source"))
	_, err = s.preparePipeline(p.ProjectID, id, prepareInput{RequestID: "req_drift", ArtifactID: artifact})
	wantCode(t, err, "stale_revision")
	if len(s.store.data.Runs) != 0 {
		t.Fatal("source drift launched a Run")
	}
}

func TestPreparationClosePreservesCancelledReceipt(t *testing.T) {
	s, p, id, artifact := preparationFixture(t, true)
	_, err := s.preparePipeline(p.ProjectID, id, prepareInput{RequestID: "req_close", ArtifactID: artifact})
	if err != nil {
		t.Fatal(err)
	}
	profile := filepath.Dir(s.store.dir)
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(context.Background(), profile)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	v, err := reopened.preparation(p.ProjectID, id, "req_close")
	if err != nil || v.State != "cancelled" {
		t.Fatal(v, err)
	}
	retry, err := reopened.preparePipeline(p.ProjectID, id, prepareInput{RequestID: "req_close", ArtifactID: artifact})
	if err != nil || retry.State != "cancelled" {
		t.Fatal(retry, err)
	}
}

func TestPreparationRejectsEvaluatorReviewMismatch(t *testing.T) {
	s, p, id, artifact := preparationFixture(t, false)
	original := s.runtime.command
	s.runtime.command = func(ctx context.Context, args, env []string) ([]byte, error) {
		raw, err := original(ctx, args, env)
		if err == nil && len(args) > 0 && args[len(args)-1] == "prepared-review" {
			var v verifiedPreparation
			if err = json.Unmarshal(raw, &v); err != nil {
				return nil, err
			}
			v.Steps[0].CPU = 99
			return json.Marshal(v)
		}
		return raw, err
	}
	if _, err := s.preparePipeline(p.ProjectID, id, prepareInput{RequestID: "req_forged", ArtifactID: artifact}); err != nil {
		t.Fatal(err)
	}
	v := awaitPreparation(t, s, p.ProjectID, id, "req_forged")
	if v.State != "failed" || v.Prepared != nil || !strings.Contains(v.Issue, "does not match") {
		t.Fatal(v)
	}
	if len(s.store.data.Runs) != 0 {
		t.Fatal("mismatched review launched a Run")
	}
}
