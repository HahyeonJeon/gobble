package engine

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	osexec "os/exec"
	"path/filepath"
	"sync"
	"testing"
)

var checkpointBoundaries = []string{PlanFile, TasksFile, RunIdentityFile, "generation", "pointer"}

func seedCheckpoint(t *testing.T) (Request, string) {
	t.Helper()
	dir := t.TempDir()
	writeCheckFile(t, filepath.Join(dir, "in", "sample.txt"), "reads")
	req := Request{Identity: testInstallIdentity(), Workspace: dir,
		Document: sampleDoc("", "", "in/sample.txt", "out/sample.txt")}
	if d := Run(t.Context(), req); len(d) > 0 {
		t.Fatal(d)
	}
	DropHeldLease(dir)
	t.Cleanup(func() { DropHeldLease(dir) })
	run, _, _, _, _, d := readCoherentControl(dir)
	if len(d) > 0 {
		t.Fatal(d)
	}
	return req, run.Snapshot
}

func copyCheckpoint(workspace, snapshot string, after func(string) error) error {
	run, plan, _, tasks, _, d := readCoherentControl(workspace)
	if len(d) > 0 {
		return fmt.Errorf("read checkpoint: %v", d)
	}
	run.Snapshot, plan.Snapshot, tasks.Snapshot = snapshot, snapshot, snapshot
	r, err := json.Marshal(run)
	if err != nil {
		return err
	}
	p, err := json.Marshal(plan)
	if err != nil {
		return err
	}
	s, err := json.Marshal(tasks)
	if err != nil {
		return err
	}
	return commitCheckpointAt(workspace, snapshot, p, s, r, after)
}

func requireSnapshot(t *testing.T, workspace, want string) {
	t.Helper()
	run, plan, hasPlan, tasks, hasTasks, d := readCoherentControl(workspace)
	if len(d) > 0 || !hasPlan || !hasTasks {
		t.Fatalf("checkpoint is not readable: %v; plan=%v tasks=%v", d, hasPlan, hasTasks)
	}
	if run.Snapshot != want || plan.Snapshot != want || tasks.Snapshot != want {
		t.Fatalf("mixed or unexpected checkpoint: run=%q plan=%q tasks=%q, want %q",
			run.Snapshot, plan.Snapshot, tasks.Snapshot, want)
	}
	if _, d := Inspect(workspace, viewRun, "", testInstallIdentity()); len(d) > 0 {
		t.Fatalf("Inspect: %v", d)
	}
}

// A subprocess exits without defers: partial generations remain exactly as they
// would after controller death, unlike a returned-error test that cleans up.
func TestCheckpointSurvivesWriterDeath(t *testing.T) {
	for _, boundary := range checkpointBoundaries {
		t.Run(boundary, func(t *testing.T) {
			req, old := seedCheckpoint(t)
			next := newOccupancyID()
			cmd := osexec.Command(os.Args[0], "-test.run=^TestCheckpointCrashWriter$")
			cmd.Env = append(os.Environ(),
				"GOBBLE_CHECKPOINT_CRASH_WORKSPACE="+req.Workspace,
				"GOBBLE_CHECKPOINT_CRASH_BOUNDARY="+boundary,
				"GOBBLE_CHECKPOINT_CRASH_SNAPSHOT="+next)
			out, err := cmd.CombinedOutput()
			var exited *osexec.ExitError
			if !errors.As(err, &exited) || exited.ExitCode() != 73 {
				t.Fatalf("crash writer: %v\n%s", err, out)
			}
			want := old
			if boundary == "pointer" {
				want = next
			}
			requireSnapshot(t, req.Workspace, want)
			if d := Release(req.Workspace, req.Identity); len(d) > 0 {
				t.Fatalf("Release after writer death: %v", d)
			}
			if d := Resume(t.Context(), req); len(d) > 0 {
				t.Fatalf("Resume after writer death: %v", d)
			}
			st := taskStates(t, req.Workspace)["copy"]
			if st.Attempt != 1 || st.Status != StatusSucceeded || st.Decision != reuseReused {
				t.Fatalf("completed work was not reused: %+v", st)
			}
			entries, err := os.ReadDir(filepath.Join(req.Workspace, ControlDir, checkpointDirectory))
			if err != nil || len(entries) != 2 {
				t.Fatalf("retention after recovery: %d generations, error %v", len(entries), err)
			}
		})
	}
}

func TestCheckpointCrashWriter(t *testing.T) {
	workspace := os.Getenv("GOBBLE_CHECKPOINT_CRASH_WORKSPACE")
	if workspace == "" {
		return
	}
	err := copyCheckpoint(workspace, os.Getenv("GOBBLE_CHECKPOINT_CRASH_SNAPSHOT"), func(boundary string) error {
		if boundary == os.Getenv("GOBBLE_CHECKPOINT_CRASH_BOUNDARY") {
			os.Exit(73)
		}
		return nil
	})
	t.Fatalf("crash boundary was not reached: %v", err)
}

func TestCheckpointCommitErrorKeepsCompleteState(t *testing.T) {
	for _, boundary := range checkpointBoundaries {
		t.Run(boundary, func(t *testing.T) {
			req, old := seedCheckpoint(t)
			next := newOccupancyID()
			fault := errors.New("storage unavailable")
			err := copyCheckpoint(req.Workspace, next, func(at string) error {
				if at == boundary {
					return fault
				}
				return nil
			})
			if !errors.Is(err, fault) {
				t.Fatalf("commit error = %v, want %v", err, fault)
			}
			want := old
			if boundary == "pointer" {
				want = next
			}
			requireSnapshot(t, req.Workspace, want)
		})
	}
}

func TestCheckpointReaderAndRetention(t *testing.T) {
	req, _ := seedCheckpoint(t)
	start := make(chan struct{})
	errs := make(chan error, 2)
	var workers sync.WaitGroup
	workers.Add(2)
	go func() {
		defer workers.Done()
		<-start
		for range 40 {
			if err := copyCheckpoint(req.Workspace, newOccupancyID(), nil); err != nil {
				errs <- err
				return
			}
		}
	}()
	go func() {
		defer workers.Done()
		<-start
		for range 120 {
			_, _, _, _, _, d := readCoherentControl(req.Workspace)
			if len(d) > 0 {
				errs <- fmt.Errorf("reader observed invalid/pruned generation: %v", d)
				return
			}
		}
	}()
	close(start)
	workers.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
}

func TestCheckpointLegacyMigration(t *testing.T) {
	req, snapshot := seedCheckpoint(t)
	root := filepath.Join(req.Workspace, ControlDir)
	legacy := map[string][]byte{}
	for _, name := range []string{RunIdentityFile, PlanFile, TasksFile} {
		raw, err := os.ReadFile(filepath.Join(root, checkpointDirectory, snapshot, name))
		if err != nil {
			t.Fatal(err)
		}
		legacy[name] = raw
		if err := os.WriteFile(filepath.Join(root, name), raw, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Remove(filepath.Join(root, checkpointPointerFile)); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(root, checkpointDirectory)); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(root, checkpointLockFile)); err != nil {
		t.Fatal(err)
	}
	requireSnapshot(t, req.Workspace, snapshot)
	if _, err := os.Stat(filepath.Join(root, checkpointLockFile)); !os.IsNotExist(err) {
		t.Fatal("Inspect wrote into a legacy workspace")
	}
	if d := Release(req.Workspace, req.Identity); len(d) > 0 {
		t.Fatal(d)
	}
	if _, err := os.Stat(filepath.Join(root, checkpointPointerFile)); err != nil {
		t.Fatalf("Release did not migrate the checkpoint: %v", err)
	}
	for name, want := range legacy {
		got, err := os.ReadFile(filepath.Join(root, name))
		if err != nil || !bytes.Equal(got, want) {
			t.Fatalf("migration changed legacy %s: %v", name, err)
		}
	}
	if d := Resume(t.Context(), req); len(d) > 0 {
		t.Fatal(d)
	}
}

func TestCheckpointRefusesDamagedCommittedState(t *testing.T) {
	for _, damage := range []string{"pointer", "missing-generation", "missing-member", "snapshot", "schema", "generation-symlink", "member-symlink", "lock-symlink"} {
		t.Run(damage, func(t *testing.T) {
			req, snapshot := seedCheckpoint(t)
			root := filepath.Join(req.Workspace, ControlDir)
			generation := filepath.Join(root, checkpointDirectory, snapshot)
			// Valid old controls must not hide corruption of the committed state.
			for _, name := range []string{RunIdentityFile, PlanFile, TasksFile} {
				raw, err := os.ReadFile(filepath.Join(generation, name))
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(root, name), raw, 0o600); err != nil {
					t.Fatal(err)
				}
			}
			var err error
			switch damage {
			case "pointer":
				err = os.WriteFile(filepath.Join(root, checkpointPointerFile), []byte("{"), 0o600)
			case "missing-generation":
				err = os.RemoveAll(generation)
			case "missing-member":
				err = os.Remove(filepath.Join(generation, TasksFile))
			case "snapshot", "schema":
				run, _, _, _, _, d := readCoherentControl(req.Workspace)
				if len(d) > 0 {
					t.Fatal(d)
				}
				if damage == "snapshot" {
					run.Snapshot = newOccupancyID()
				} else {
					run.SchemaVersion++
				}
				raw, e := json.Marshal(run)
				if e != nil {
					t.Fatal(e)
				}
				err = os.WriteFile(filepath.Join(generation, RunIdentityFile), raw, 0o600)
			default:
				path := generation
				if damage == "member-symlink" {
					path = filepath.Join(generation, TasksFile)
				} else if damage == "lock-symlink" {
					path = filepath.Join(root, checkpointLockFile)
				}
				outside := filepath.Join(t.TempDir(), "target")
				if err := os.Rename(path, outside); err != nil {
					t.Fatal(err)
				}
				err = os.Symlink(outside, path)
			}
			if err != nil {
				t.Fatal(err)
			}
			if _, d := Inspect(req.Workspace, viewRun, "", req.Identity); len(d) == 0 {
				t.Fatal("Inspect silently accepted damaged committed state")
			}
			if d := Release(req.Workspace, req.Identity); len(d) == 0 {
				t.Fatal("Release silently accepted damaged committed state")
			}
		})
	}
}
