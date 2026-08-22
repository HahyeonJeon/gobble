package engine

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/HahyeonJeon/gobble/internal/engine/exec"
)

// occupyLockFile is the local exclusive used to claim occupancy
// without deleting run.json. The occupying process retains the
// flock as private liveness until occupancy closes or the process dies.
const occupyLockFile = "occupy.lock"

type jsonOccupancy struct {
	Active     bool     `json:"active"`
	Host       string   `json:"host,omitempty"`
	PID        int      `json:"pid,omitempty"`
	Lease      string   `json:"lease,omitempty"`
	Started    string   `json:"started,omitempty"`
	Closed     string   `json:"closed,omitempty"`
	Incomplete []string `json:"incomplete,omitempty"`
	Unknown    []string `json:"unknown,omitempty"`
}

type lateSubmit struct {
	ident string
	done  chan struct{}
	once  sync.Once
	h     exec.Handle
	sub   exec.Report
	err   error
}

type heldLease struct {
	file *os.File
	exec exec.Executor
	mu   sync.Mutex
	late map[string]*lateSubmit
}

var (
	leaseMu     sync.Mutex
	heldLeases  = map[string]*heldLease{}
	forgottenFD []*os.File
)

func occupancyIsActive(r jsonRun) bool {
	if r.Occupancy == nil {
		return false
	}
	return r.Occupancy.Active
}

func schemaUnsupported(version int) bool {
	return version != SchemaVersion
}

func pidOnlyOccupancy(run jsonRun) bool {
	return occupancyIsActive(run) && (run.Occupancy == nil || run.Occupancy.Lease == "")
}

func readRunIdentity(workspace string) (jsonRun, bool, error) {
	path := filepath.Join(workspace, ControlDir, RunIdentityFile)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return jsonRun{}, false, nil
		}
		return jsonRun{}, false, err
	}
	var r jsonRun
	if err := json.Unmarshal(data, &r); err != nil {
		return jsonRun{}, true, nil
	}
	return r, true, nil
}

func readSchemaFile(path string) (int, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, false, nil
		}
		return 0, false, err
	}
	var doc struct {
		SchemaVersion int `json:"schema_version"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return 0, true, nil
	}
	return doc.SchemaVersion, true, nil
}

func occupiedDefect() []Defect {
	return []Defect{{
		Code:    DefectOccupiedWorkspace,
		Message: "occupied workspace",
		Paths:   []string{ControlDir + "/" + RunIdentityFile},
	}}
}

func liveOccupancyDefect() []Defect {
	return []Defect{{
		Code:    DefectLiveOccupancy,
		Message: "live occupancy",
		Paths:   []string{ControlDir + "/" + RunIdentityFile},
	}}
}

func unknownBackendDefects(units []string) []Defect {
	out := make([]Defect, 0, len(units))
	for _, unit := range units {
		out = append(out, Defect{
			Code:    DefectUnknownBackend,
			Unit:    unit,
			Message: "unknown backend",
			Paths:   []string{ControlDir + "/" + RunIdentityFile},
		})
	}
	if len(out) == 0 {
		return []Defect{{
			Code:    DefectUnknownBackend,
			Message: "unknown backend",
			Paths:   []string{ControlDir + "/" + RunIdentityFile},
		}}
	}
	return out
}

func claimOccupy(root string) (*os.File, []Defect) {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, pathDefects(err)
	}
	f, err := os.OpenFile(filepath.Join(root, occupyLockFile), os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, pathDefects(err)
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		f.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) || errors.Is(err, syscall.EAGAIN) {
			return nil, occupiedDefect()
		}
		return nil, pathDefects(err)
	}
	return f, nil
}

func occupancyKey(workspace string) string {
	abs, err := filepath.Abs(workspace)
	if err != nil {
		return filepath.Clean(workspace)
	}
	return abs
}

func retainLease(workspace string, f *os.File, ex exec.Executor) {
	key := occupancyKey(workspace)
	leaseMu.Lock()
	defer leaseMu.Unlock()
	if old := heldLeases[key]; old != nil && old.file != nil && old.file != f {
		old.file.Close()
	}
	heldLeases[key] = &heldLease{file: f, exec: ex, late: map[string]*lateSubmit{}}
}

func registerLateSubmit(workspace, ident string) *lateSubmit {
	ls := &lateSubmit{ident: ident, done: make(chan struct{})}
	key := occupancyKey(workspace)
	leaseMu.Lock()
	h := heldLeases[key]
	leaseMu.Unlock()
	if h == nil {
		return ls
	}
	h.mu.Lock()
	if h.late == nil {
		h.late = map[string]*lateSubmit{}
	}
	h.late[ident] = ls
	h.mu.Unlock()
	return ls
}

func finishLateSubmit(ls *lateSubmit, handle exec.Handle, sub exec.Report, err error) {
	if ls == nil {
		return
	}
	ls.once.Do(func() {
		ls.h = handle
		ls.sub = sub
		ls.err = err
		close(ls.done)
	})
}

func awaitLateSubmit(workspace, ident string) (exec.Handle, exec.Report, bool) {
	key := occupancyKey(workspace)
	leaseMu.Lock()
	h := heldLeases[key]
	leaseMu.Unlock()
	if h == nil {
		return exec.Handle{}, exec.Report{}, false
	}
	h.mu.Lock()
	ls := h.late[ident]
	h.mu.Unlock()
	if ls == nil {
		return exec.Handle{}, exec.Report{}, false
	}
	select {
	case <-ls.done:
		if ls.err != nil || ls.h.RuntimeID == "" {
			return exec.Handle{}, exec.Report{}, false
		}
		return ls.h, ls.sub, true
	case <-time.After(currentBound()):
		return exec.Handle{}, exec.Report{}, false
	}
}

func DropHeldLease(workspace string) {
	key := occupancyKey(workspace)
	leaseMu.Lock()
	h := heldLeases[key]
	delete(heldLeases, key)
	leaseMu.Unlock()
	if h != nil && h.file != nil {
		h.file.Close()
	}
}

func ForgetHeldLease(workspace string) {
	key := occupancyKey(workspace)
	leaseMu.Lock()
	if h := heldLeases[key]; h != nil && h.file != nil {
		forgottenFD = append(forgottenFD, h.file)
	}
	delete(heldLeases, key)
	leaseMu.Unlock()
}

func holdsLease(workspace string) bool {
	key := occupancyKey(workspace)
	leaseMu.Lock()
	defer leaseMu.Unlock()
	_, ok := heldLeases[key]
	return ok
}

func heldExecutor(workspace string) exec.Executor {
	key := occupancyKey(workspace)
	leaseMu.Lock()
	defer leaseMu.Unlock()
	if h := heldLeases[key]; h != nil {
		return h.exec
	}
	return nil
}

func ownerLive(workspace string) bool {
	if holdsLease(workspace) {
		return true
	}
	return flockHeld(workspace)
}

func flockHeld(workspace string) bool {
	path := filepath.Join(workspace, ControlDir, occupyLockFile)
	f, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return false
	}
	defer f.Close()
	err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		return errors.Is(err, syscall.EWOULDBLOCK) || errors.Is(err, syscall.EAGAIN)
	}
	_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	return false
}

func newOccupancyID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return hex.EncodeToString([]byte(time.Now().UTC().Format(time.RFC3339Nano)))
	}
	return hex.EncodeToString(b[:])
}

func currentHost() (string, error) {
	return os.Hostname()
}

func cloneDocument(doc Document) Document {
	out := doc
	if doc.Tasks != nil {
		out.Tasks = make([]TaskPlan, len(doc.Tasks))
		for i, t := range doc.Tasks {
			out.Tasks[i] = cloneTaskPlan(t)
		}
	}
	if doc.Edges != nil {
		out.Edges = make([]Edge, len(doc.Edges))
		for i, e := range doc.Edges {
			e.Wait = copyStrings(e.Wait)
			out.Edges[i] = e
		}
	}
	return out
}

func cloneTaskPlan(t TaskPlan) TaskPlan {
	t.Command = copyStrings(t.Command)
	if t.Params != nil {
		t.Params = append([]ParamPlan(nil), t.Params...)
	}
	t.Env = copyStringMap(t.Env)
	t.Inputs = cloneIOs(t.Inputs)
	t.Outputs = cloneIOs(t.Outputs)
	return t
}

func cloneIOs(in []IO) []IO {
	if in == nil {
		return nil
	}
	out := make([]IO, len(in))
	for i, io := range in {
		io.Spec.Suffixes = copyStrings(io.Spec.Suffixes)
		if io.Members != nil {
			mem := make([]IOMember, len(io.Members))
			copy(mem, io.Members)
			for j := range mem {
				mem[j].Spec.Suffixes = copyStrings(mem[j].Spec.Suffixes)
			}
			io.Members = mem
		}
		out[i] = io
	}
	return out
}
