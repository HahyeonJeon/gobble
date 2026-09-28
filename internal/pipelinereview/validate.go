package pipelinereview

import (
	"errors"
	"regexp"
)

var sha256Pattern = regexp.MustCompile(`^sha256:[a-f0-9]{64}$`)

// Validate bounds portable checked facts before comparison. Invalid fingerprints
// cannot accidentally compare as equal, and endpoints must belong to this graph.
func Validate(d Definition) error {
	bad := func() error { return errors.New("invalid checked review definition") }
	if d.SchemaVersion != 1 || !sha256Pattern.MatchString(d.Context) || d.Steps == nil || d.Edges == nil || len(d.Steps) > 500 || len(d.Edges) > 2000 {
		return bad()
	}
	ids := map[string]bool{}
	for _, s := range d.Steps {
		if s.ID == "" || len(s.ID) > 256 || ids[s.ID] || !sha256Pattern.MatchString(s.Fingerprint) || !sha256Pattern.MatchString(s.Residual) || len(s.Settings) > 32 {
			return bad()
		}
		ids[s.ID] = true
		if s.Recipe != "" && s.Recipe != "trim-galore-v1" && s.Recipe != "fastqc-v1" {
			return bad()
		}
		keys := map[string]bool{}
		for _, v := range s.Settings {
			if v.Key == "" || len(v.Key) > 100 || keys[v.Key] || len(v.Label) > 200 || len(v.Unit) > 100 {
				return bad()
			}
			keys[v.Key] = true
		}
	}
	for _, e := range d.Edges {
		if (e.FromTask != "" && !ids[e.FromTask]) || !ids[e.ToTask] || e.FromPort == "" || e.ToPort == "" || len(e.FromPort) > 256 || len(e.ToPort) > 256 || len(e.Wait) > 500 {
			return bad()
		}
	}
	return nil
}
