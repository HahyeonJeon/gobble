package gobble

import "fmt"

// IntegerSetting is module-authored inspection metadata, never an executable
// parameter. Value nil means the tool chooses its default. Key is stable within
// the declaring step; Label and Unit are presentation. Modules must derive Value
// from the same validated option used to construct their command.
type IntegerSetting struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Unit  string `json:"unit"`
	Value *int   `json:"value"`
}

func cloneIntegerSettings(in []IntegerSetting) []IntegerSetting {
	out := append([]IntegerSetting{}, in...)
	for i := range out {
		if out[i].Value != nil {
			value := *out[i].Value
			out[i].Value = &value
		}
	}
	return out
}

func validateIntegerSettings(in []IntegerSetting) error {
	if len(in) > 32 {
		return fmt.Errorf("inspection settings exceed 32 entries")
	}
	keys := make(map[string]bool, len(in))
	for _, field := range in {
		if field.Key == "" || len(field.Key) > 128 || keys[field.Key] || field.Label == "" || len(field.Label) > 256 || len(field.Unit) > 64 || (field.Value != nil && (*field.Value < -9007199254740991 || *field.Value > 9007199254740991)) {
			return fmt.Errorf("invalid inspection setting %q", field.Key)
		}
		keys[field.Key] = true
	}
	return nil
}
