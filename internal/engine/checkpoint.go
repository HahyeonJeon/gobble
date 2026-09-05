package engine

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

const (
	checkpointPointerFile = "current.json"
	checkpointDirectory   = "checkpoints"
	checkpointLockFile    = "checkpoint.lock"
	checkpointFormat      = 1
)

type checkpointPointer struct {
	Format   int    `json:"format"`
	Current  string `json:"current"`
	Previous string `json:"previous,omitempty"`
}

var errCheckpointSchema = errors.New("unsupported checkpoint schema")

func checkpointDefects(err error) []Defect {
	if errors.Is(err, errCheckpointSchema) {
		return schemaDefect(ControlDir + "/" + checkpointPointerFile)
	}
	return pathDefects(err)
}

func validCheckpointID(id string) bool {
	if len(id) < 32 || len(id) > 128 {
		return false
	}
	_, err := hex.DecodeString(id)
	return err == nil
}

// checkpointRoot resolves a committed generation; a missing pointer selects
// the legacy flat layout. A malformed or unsafe pointer never falls back.
func checkpointRoot(workspace string) (string, bool, error) {
	p, present, err := containedRel(workspace, ControlDir+"/"+checkpointPointerFile, false)
	if err != nil {
		return "", false, err
	}
	if !present {
		return ControlDir, false, nil
	}
	raw, err := os.ReadFile(p)
	if err != nil {
		return "", true, err
	}
	var ptr checkpointPointer
	if err := json.Unmarshal(raw, &ptr); err != nil {
		return "", true, fmt.Errorf("invalid checkpoint pointer: %w", err)
	}
	if ptr.Format != checkpointFormat || !validCheckpointID(ptr.Current) || (ptr.Previous != "" && !validCheckpointID(ptr.Previous)) {
		return "", true, errors.New("unsupported or invalid checkpoint pointer")
	}
	rel := ControlDir + "/" + checkpointDirectory + "/" + ptr.Current
	path, present, err := containedRel(workspace, rel, false)
	if err != nil {
		return "", true, err
	}
	if !present {
		return "", true, errors.New("committed checkpoint is missing")
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", true, err
	}
	if !info.IsDir() {
		return "", true, errors.New("committed checkpoint is not a directory")
	}
	return rel, true, nil
}

func checkpointLock(workspace string, write bool) (*os.File, error) {
	rel := ControlDir + "/" + checkpointLockFile
	path, present, err := containedRel(workspace, rel, false)
	if err != nil {
		return nil, err
	}
	if !write && !present {
		return nil, nil
	}
	flag := os.O_RDONLY
	if write {
		flag = os.O_CREATE | os.O_RDWR
	}
	f, err := os.OpenFile(path, flag|syscall.O_NOFOLLOW, 0o600)
	if err != nil {
		return nil, err
	}
	op := syscall.LOCK_SH
	if write {
		op = syscall.LOCK_EX
	}
	if err := syscall.Flock(int(f.Fd()), op); err != nil {
		f.Close()
		return nil, err
	}
	return f, nil
}

func closeCheckpointLock(f *os.File) {
	if f != nil {
		_ = f.Close()
	}
}

// openCheckpoint pins one generation until the returned lock is closed. Legacy
// workspaces need no new file for a read; their flat controls are never changed
// by a generation writer. Retry the lock if the first commit raced this read.
func openCheckpoint(workspace string) (*os.File, string, bool, error) {
	lock, err := checkpointLock(workspace, false)
	if err != nil {
		return nil, "", false, err
	}
	if lock == nil {
		_, present, err := containedRel(workspace, ControlDir+"/"+checkpointPointerFile, false)
		if err != nil {
			return nil, "", false, err
		}
		if !present {
			return nil, ControlDir, false, nil
		}
		lock, err = checkpointLock(workspace, false)
		if err == nil && lock == nil {
			err = errors.New("committed checkpoint lock is missing")
		}
		if err != nil {
			return nil, "", true, err
		}
	}
	root, committed, err := checkpointRoot(workspace)
	if err != nil {
		closeCheckpointLock(lock)
		return nil, "", committed, err
	}
	return lock, root, committed, nil
}

// readControlFile is for individual metadata reads. Full Inspect reads pin one
// generation under a shared checkpoint lock in readCoherentControl.
func readControlFile(workspace, name string) ([]byte, bool, error) {
	lock, root, committed, err := openCheckpoint(workspace)
	if err != nil {
		return nil, false, err
	}
	defer closeCheckpointLock(lock)
	return readCheckpointMember(workspace, root, name, committed)
}

func readCheckpointMember(workspace, root, name string, required bool) ([]byte, bool, error) {
	path, present, err := containedRel(workspace, root+"/"+name, false)
	if err != nil {
		return nil, false, err
	}
	if !present {
		if required {
			return nil, false, fmt.Errorf("committed checkpoint member %s is missing", name)
		}
		return nil, false, nil
	}
	raw, err := os.ReadFile(path)
	return raw, true, err
}

func readCommittedControl(workspace, root string) (jsonRun, jsonPlan, jsonTasksFile, error) {
	var run jsonRun
	var plan jsonPlan
	var tasks jsonTasksFile
	for _, member := range []struct {
		name string
		dest any
	}{{RunIdentityFile, &run}, {PlanFile, &plan}, {TasksFile, &tasks}} {
		raw, _, err := readCheckpointMember(workspace, root, member.name, true)
		if err != nil {
			return run, plan, tasks, err
		}
		if err := json.Unmarshal(raw, member.dest); err != nil {
			return run, plan, tasks, fmt.Errorf("invalid checkpoint %s: %w", member.name, err)
		}
	}
	if schemaUnsupported(run.SchemaVersion) || schemaUnsupported(plan.SchemaVersion) || schemaUnsupported(tasks.SchemaVersion) || pidOnlyOccupancy(run) {
		return run, plan, tasks, errCheckpointSchema
	}
	if run.Snapshot != filepath.Base(root) || plan.Snapshot != run.Snapshot || tasks.Snapshot != run.Snapshot {
		return run, plan, tasks, errors.New("control snapshot is not coherent")
	}
	return run, plan, tasks, nil
}

// commitCheckpoint publishes all controls through one durable pointer update.
// The occupying writer serializes commits. The checkpoint lock also protects
// readers while obsolete generations are pruned.
func commitCheckpoint(workspace, snapshot string, plan, tasks, run []byte) error {
	return commitCheckpointAt(workspace, snapshot, plan, tasks, run, nil)
}

// after is a per-call fault boundary used by recovery tests, never a global seam.
func commitCheckpointAt(workspace, snapshot string, plan, tasks, run []byte, after func(string) error) error {
	if !validCheckpointID(snapshot) {
		return errors.New("invalid checkpoint identity")
	}
	if d := checkControlContainment(workspace); len(d) > 0 {
		return errors.New(d[0].Message)
	}
	lock, err := checkpointLock(workspace, true)
	if err != nil {
		return err
	}
	defer closeCheckpointLock(lock)
	oldRoot, hasOld, err := checkpointRoot(workspace)
	if err != nil {
		return err
	}
	base := filepath.Join(workspace, ControlDir, checkpointDirectory)
	if err := os.MkdirAll(base, 0o700); err != nil {
		return err
	}
	dir := filepath.Join(base, snapshot)
	if err := os.Mkdir(dir, 0o700); err != nil {
		return err
	}
	published := false
	defer func() {
		if !published {
			_ = os.RemoveAll(dir)
		}
	}()
	step := func(name string) error {
		if after != nil {
			return after(name)
		}
		return nil
	}
	for _, member := range []struct {
		name string
		data []byte
	}{{PlanFile, plan}, {TasksFile, tasks}, {RunIdentityFile, run}} {
		if err := writeAtomic(filepath.Join(dir, member.name), member.data); err != nil {
			return err
		}
		if err := step(member.name); err != nil {
			return err
		}
	}
	if err := syncDirectory(base); err != nil {
		return err
	}
	if err := syncDirectory(filepath.Dir(base)); err != nil {
		return err
	}
	// The first checkpoint must also persist the workspace's .gobble entry.
	if err := syncDirectory(workspace); err != nil {
		return err
	}
	if err := step("generation"); err != nil {
		return err
	}
	ptr := checkpointPointer{Format: checkpointFormat, Current: snapshot}
	if hasOld {
		ptr.Previous = filepath.Base(oldRoot)
	}
	data, err := json.MarshalIndent(ptr, "", "  ")
	if err != nil {
		return err
	}
	// If rename succeeds but directory sync fails, the new pointer can already
	// be visible. Never delete its complete generation on that error path.
	published = true
	if err := writeAtomic(filepath.Join(workspace, ControlDir, checkpointPointerFile), append(data, '\n')); err != nil {
		return err
	}
	if err := step("pointer"); err != nil {
		return err
	}
	// Old flat controls are kept as migration evidence, but are no longer read.
	// Retention is best effort and cannot turn a committed write into a failure.
	entries, err := os.ReadDir(base)
	if err == nil {
		for _, entry := range entries {
			name := entry.Name()
			if entry.IsDir() && validCheckpointID(name) && name != ptr.Current && name != ptr.Previous {
				_ = os.RemoveAll(filepath.Join(base, name))
			}
		}
	}
	return nil
}
