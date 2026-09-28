package appservice

import (
	"bytes"
	"encoding/json"
	"errors"
)

// PipelineOrigin distinguishes imported Project resources from service-owned source.
// Managed origins have no Project resource IDs; their Current owns retained bytes.
type PipelineOrigin struct {
	Kind              string `json:"kind"`
	PackageResourceID string `json:"packageResourceId,omitempty"`
	SourceResourceID  string `json:"sourceResourceId,omitempty"`
}

// UnmarshalJSON keeps the two origin shapes closed, including empty fields that
// would otherwise disappear through omitempty during a catalog round trip.
func (o *PipelineOrigin) UnmarshalJSON(raw []byte) error {
	type originValue PipelineOrigin
	var value originValue
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		return err
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &fields) != nil || fields == nil {
		return errors.New("invalid pipeline origin")
	}
	switch value.Kind {
	case "imported":
		if len(fields) != 3 || value.PackageResourceID == "" || value.SourceResourceID == "" {
			return errors.New("invalid imported origin")
		}
	case "managed":
		if len(fields) != 1 {
			return errors.New("managed source has no Project resource binding")
		}
	default:
		return errors.New("unsupported pipeline origin")
	}
	*o = PipelineOrigin(value)
	return nil
}
