package engine

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"regexp"

	"github.com/HahyeonJeon/gobble/internal/preparation"
)

// preparedV1 encodes the COMPLETE engine Document, including environment values,
// execution/control fields and the engine-owned Path representation. It is NOT
// public Plan JSON. Changing any nested engine representation requires version review.
type preparedV1 struct {
	SchemaVersion int                 `json:"schemaVersion"`
	Binding       preparation.Binding `json:"binding"`
	Document      Document            `json:"document"`
}

const maxPreparedBytes = 3 << 20

var preparedDigestPattern = regexp.MustCompile(`^sha256:[a-f0-9]{64}$`)

func validPreparationBinding(b preparation.Binding) bool {
	return preparedDigestPattern.MatchString(b.SourceRevision) && preparedDigestPattern.MatchString(b.RuntimeID) && preparedDigestPattern.MatchString(b.InputIdentity) && b.InputPath != "" && b.Cap == 1
}

// Prepare seals one validated, qualified document without evaluating or executing
// tasks. Only the bounded single-end creation design is admitted in format 1.
func Prepare(doc Document, b preparation.Binding) ([]byte, string, error) {
	if !validPreparationBinding(b) {
		return nil, "", errors.New("invalid preparation binding")
	}
	if len(Validate(doc)) != 0 || len(QualifyCreation(doc, b.InputPath)) != 0 {
		return nil, "", errors.New("unsupported or invalid preparation design")
	}
	raw, err := json.Marshal(preparedV1{1, b, doc})
	if err != nil || len(raw) > maxPreparedBytes {
		return nil, "", errors.New("prepared document exceeds encoding bounds")
	}
	return raw, preparedDigest(raw), nil
}
func preparedDigest(raw []byte) string {
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// DecodePrepared validates exact bytes and expected external facts without Project
// code evaluation or filesystem effects. P4 must call this BEFORE occupancy and
// additionally bind staged input content and runtime/controller identity.
func DecodePrepared(raw []byte, expectedDigest string, expected preparation.Binding) (Document, error) {
	bad := func() (Document, error) {
		return Document{}, errors.New("prepared document or binding mismatch; review required")
	}
	if len(raw) == 0 || len(raw) > maxPreparedBytes || !preparedDigestPattern.MatchString(expectedDigest) || preparedDigest(raw) != expectedDigest {
		return bad()
	}
	var p preparedV1
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if d.Decode(&p) != nil || d.Decode(new(any)) != io.EOF || p.SchemaVersion != 1 || p.Binding != expected {
		return bad()
	}
	// Canonical equality also rejects duplicate keys and lossy JSON decoding.
	canonical, id, err := Prepare(p.Document, p.Binding)
	if err != nil || id != expectedDigest || !bytes.Equal(canonical, raw) {
		return bad()
	}
	return p.Document, nil
}

// PreparedSteps is safe only after complete qualification; no argv, scripts or
// environment values enter the review. It derives resource/output facts from doc.
func PreparedSteps(doc Document) []preparation.Step {
	out := make([]preparation.Step, 0, len(doc.Tasks))
	for _, t := range doc.Tasks {
		label := "Trim reads"
		if t.Name == "fastqc" {
			label = "Check read quality"
		}
		outputs := make([]string, 0, len(t.Outputs))
		for _, o := range t.Outputs {
			outputs = append(outputs, o.Path)
		}
		out = append(out, preparation.Step{t.ID, label, t.Resources.CPU, t.Resources.Memory, t.Image, outputs})
	}
	return out
}
