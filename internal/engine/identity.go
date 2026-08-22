package engine

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"sort"
	"strconv"
	"syscall"

	"github.com/HahyeonJeon/gobble/internal/engine/exec"
)

const (
	// DefaultShardIndex is the first-horizon shard index.
	DefaultShardIndex = 0
	// DefaultShardCount is the first-horizon shard count.
	DefaultShardCount = 1
	// DefaultAttempt is the first execution attempt.
	DefaultAttempt = 1

	emptyInstanceSeg = "_"
)

type jsonFileHash struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
	Mtime  int64  `json:"mtime"`
	Dev    uint64 `json:"dev"`
	Inode  uint64 `json:"inode"`
}

type jsonLineage struct {
	Producer string `json:"producer"`
	Path     string `json:"path"`
	Checksum string `json:"checksum"`
	Consumer string `json:"consumer"`
}

func applyReservedDefaults(t *TaskPlan) {
	if t.ShardCount == 0 {
		t.ShardCount = DefaultShardCount
	}
	if t.Attempt == 0 {
		t.Attempt = DefaultAttempt
	}
}

func applyTaskStateDefaults(st *jsonTaskState) {
	if st.ShardCount == 0 {
		st.ShardCount = DefaultShardCount
	}
	if st.Attempt == 0 {
		st.Attempt = DefaultAttempt
	}
}

func instanceSeg(instance string) string {
	if instance == "" {
		return emptyInstanceSeg
	}
	return instance
}

func reservedIdentity(t TaskPlan) string {
	if t.Instance == "" && t.ShardIndex == DefaultShardIndex {
		return t.ID
	}
	return t.ID + "/" + instanceSeg(t.Instance) + "/" + strconv.Itoa(t.ShardIndex)
}

func isolateRel(t TaskPlan) string {
	applyReservedDefaults(&t)
	return ControlDir + "/tasks/" + t.ID + "/" + instanceSeg(t.Instance) + "/" +
		strconv.Itoa(t.ShardIndex) + "/" + strconv.Itoa(t.Attempt)
}

func envDigest(env map[string]string) string {
	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	h := sha256.New()
	var lenBuf [8]byte
	for _, k := range keys {
		v := env[k]
		binary.BigEndian.PutUint64(lenBuf[:], uint64(len(k)))
		h.Write(lenBuf[:])
		h.Write([]byte(k))
		binary.BigEndian.PutUint64(lenBuf[:], uint64(len(v)))
		h.Write(lenBuf[:])
		h.Write([]byte(v))
	}
	return hex.EncodeToString(h.Sum(nil))
}

func planEnvDigest(t TaskPlan) string {
	if t.Env != nil {
		return envDigest(t.Env)
	}
	if t.EnvDigest != "" {
		return t.EnvDigest
	}
	return envDigest(nil)
}

func sha256File(path string) (string, error) {
	f, err := exec.OpenReadFile(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func cheapKey(path string) (jsonFileHash, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return jsonFileHash{}, err
	}
	if !info.Mode().IsRegular() {
		return jsonFileHash{}, exec.ErrNotRegular
	}
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return jsonFileHash{}, errors.New("stat_t unavailable")
	}
	return jsonFileHash{
		Size:  info.Size(),
		Mtime: info.ModTime().UnixNano(),
		Dev:   uint64(st.Dev),
		Inode: uint64(st.Ino),
	}, nil
}

func hasCheap(h jsonFileHash) bool {
	return h.Dev != 0 || h.Inode != 0
}

func sameCheap(a, b jsonFileHash) bool {
	return a.Size == b.Size && a.Mtime == b.Mtime && a.Dev == b.Dev && a.Inode == b.Inode
}

func fileRecord(abs, planPath string) (jsonFileHash, error) {
	rec, err := cheapKey(abs)
	if err != nil {
		return jsonFileHash{}, err
	}
	sum, err := sha256File(abs)
	if err != nil {
		return jsonFileHash{}, err
	}
	rec.Path = planPath
	rec.SHA256 = sum
	return rec, nil
}

func inputRecords(workspace string, ios []IO) ([]jsonFileHash, error) {
	return fileRecords(workspace, ios, false)
}

func destRecords(workspace string, ios []IO) ([]jsonFileHash, error) {
	return fileRecords(workspace, ios, true)
}

func fileRecords(workspace string, ios []IO, dest bool) ([]jsonFileHash, error) {
	var out []jsonFileHash
	for _, io := range ios {
		files := namedIOFiles(io)
		if isTreeIO(io) {
			probe := io
			if !dest {
				probe.Path = treeSourceDir(io)
			}
			files = treeDestMemberPaths(workspace, probe)
		}
		for _, f := range files {
			statRel := f.path
			if !dest {
				statRel = fileSource(f)
			}
			path := workspaceFile(workspace, statRel)
			if !regularFile(path) {
				out = append(out, jsonFileHash{Path: f.path})
				continue
			}
			rec, err := fileRecord(path, f.path)
			if err != nil {
				return nil, err
			}
			out = append(out, rec)
		}
	}
	return out, nil
}

func ioFiles(workspace string, io IO) []namedFile {
	if isTreeIO(io) {
		probe := io
		probe.Path = treeSourceDir(io)
		return treeDestMemberPaths(workspace, probe)
	}
	return namedIOFiles(io)
}

func hashByPath(hashes []jsonFileHash) map[string]string {
	out := make(map[string]string, len(hashes))
	for _, h := range hashes {
		out[h.Path] = h.SHA256
	}
	return out
}

func recordByPath(hashes []jsonFileHash) map[string]jsonFileHash {
	out := make(map[string]jsonFileHash, len(hashes))
	for _, h := range hashes {
		out[h.Path] = h
	}
	return out
}

func successLineage(s *sched, task TaskPlan, inputs, outputs []jsonFileHash) []jsonLineage {
	consumer := reservedIdentity(task)
	inSum := hashByPath(inputs)
	outSum := hashByPath(outputs)
	var out []jsonLineage
	for _, in := range task.Inputs {
		producer := ""
		for _, e := range s.doc.Edges {
			if e.ToTask != task.ID || e.ToPort != in.Name || e.FromTask == "" {
				continue
			}
			if up, ok := s.taskByID(e.FromTask); ok {
				producer = reservedIdentity(up)
			} else {
				producer = e.FromTask
			}
			break
		}
		for _, f := range ioFiles(s.workspace, in) {
			out = append(out, jsonLineage{
				Producer: producer,
				Path:     f.path,
				Checksum: inSum[f.path],
				Consumer: consumer,
			})
		}
	}
	for _, pub := range task.Outputs {
		for _, f := range ioFiles(s.workspace, pub) {
			out = append(out, jsonLineage{
				Producer: consumer,
				Path:     f.path,
				Checksum: outSum[f.path],
			})
		}
	}
	return out
}
