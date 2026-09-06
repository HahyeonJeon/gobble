package containerenv

import (
"errors"
"path/filepath"
"strings"
)

type Mount struct {
Type string
Source string
Destination string
RW bool
}

func MapPath(path string, mounts []Mount) (string, error) {
	abs, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", err
	}
	abs, err = filepath.Abs(abs)
	if err != nil {
		return "", err
	}
	var selected *Mount
	var relative string
	for i := range mounts {
		m := &mounts[i]
		rel, err := filepath.Rel(m.Destination, abs)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			continue
		}
		if selected == nil || len(m.Destination) > len(selected.Destination) {
			selected, relative = m, rel
		}
	}
	if selected == nil || selected.Type != "bind" || !selected.RW || !filepath.IsAbs(selected.Source) {
		return "", errors.New("attempt directory requires a writable controller bind mount")
	}
	return filepath.Join(selected.Source, relative), nil
}
