package engine

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"syscall"
)

// occupyLockFile is the local exclusive used to claim occupancy
// without deleting run.json.
const occupyLockFile = "occupy.lock"

type jsonOccupancy struct {
	Active     bool     `json:"active"`
	Host       string   `json:"host,omitempty"`
	PID        int      `json:"pid,omitempty"`
	Started    string   `json:"started,omitempty"`
	Closed     string   `json:"closed,omitempty"`
	Incomplete []string `json:"incomplete,omitempty"`
}

func occupancyIsActive(r jsonRun) bool {
	if r.Occupancy == nil {
		return false
	}
	return r.Occupancy.Active
}

func schemaUnsupported(version int) bool {
	return version != SchemaVersion
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

func pidExists(pid int) bool {
	if pid <= 0 {
		return false
	}
	err := syscall.Kill(pid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}

func currentHost() (string, error) {
	return os.Hostname()
}
