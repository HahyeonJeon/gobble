package engine

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"strconv"
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

func applyLegacyTaskSlots(doc *jsonTasksFile) {
	if doc.SchemaVersion != 0 {
		return
	}
	for i := range doc.Tasks {
		applyTaskStateDefaults(&doc.Tasks[i])
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

func sha256File(path string) (string, error) {
	f, err := openReadFile(path)
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

func fileHashes(workspace string, ios []IO) ([]jsonFileHash, error) {
	var out []jsonFileHash
	for _, io := range ios {
		for _, f := range namedIOFiles(io) {
			src := fileSource(f)
			path := workspaceFile(workspace, src)
			if !regularFile(path) {
				out = append(out, jsonFileHash{Path: f.path})
				continue
			}
			sum, err := sha256File(path)
			if err != nil {
				return nil, err
			}
			out = append(out, jsonFileHash{Path: f.path, SHA256: sum})
		}
	}
	return out, nil
}

func hashByPath(hashes []jsonFileHash) map[string]string {
	out := make(map[string]string, len(hashes))
	for _, h := range hashes {
		out[h.Path] = h.SHA256
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
		for _, f := range namedIOFiles(in) {
			out = append(out, jsonLineage{
				Producer: producer,
				Path:     f.path,
				Checksum: inSum[f.path],
				Consumer: consumer,
			})
		}
	}
	for _, pub := range task.Outputs {
		for _, f := range namedIOFiles(pub) {
			out = append(out, jsonLineage{
				Producer: consumer,
				Path:     f.path,
				Checksum: outSum[f.path],
			})
		}
	}
	return out
}
