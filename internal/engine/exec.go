package engine

import (
	"os"
	"path/filepath"

	"github.com/HahyeonJeon/gobble/internal/engine/exec"
)

func executeArgv(task TaskPlan) []string {
	if task.Script != "" {
		return []string{"sh", "-c", "set -eu\n" + task.Script}
	}
	return task.Command
}

func prepareIsolate(workspace, isolate string, task TaskPlan) error {
	for _, in := range task.Inputs {
		if isTreeIO(in) {
			if err := os.MkdirAll(workspaceFile(isolate, treeDir(in)), 0o755); err != nil {
				return err
			}
			continue
		}
		for _, f := range namedIOFiles(in) {
			if err := mkdirPlanParent(isolate, f.path); err != nil {
				return err
			}
		}
	}
	for _, out := range task.Outputs {
		if isTreeIO(out) {
			if err := os.MkdirAll(workspaceFile(isolate, treeDir(out)), 0o755); err != nil {
				return err
			}
			continue
		}
		for _, f := range namedIOFiles(out) {
			if err := mkdirPlanParent(isolate, f.path); err != nil {
				return err
			}
		}
	}
	allowSymlink := task.Image == ""
	for _, in := range task.Inputs {
		if isTreeIO(in) {
			if err := stageTree(workspace, isolate, in, allowSymlink); err != nil {
				return err
			}
			continue
		}
		for _, f := range namedIOFiles(in) {
			src := workspaceFile(workspace, fileSource(f))
			dst := workspaceFile(isolate, f.path)
			if err := exec.StageFile(src, dst, allowSymlink); err != nil {
				return err
			}
		}
	}
	return nil
}

func mkdirPlanParent(root, planPath string) error {
	dir := filepath.Dir(filepath.FromSlash(planPath))
	if dir == "." || dir == "" {
		return nil
	}
	return os.MkdirAll(filepath.Join(root, dir), 0o755)
}

func inspectOutputs(isolate string, task TaskPlan) error {
	for _, out := range task.Outputs {
		if isTreeIO(out) {
			if err := checkTreeOutput(isolate, out); err != nil {
				return err
			}
			continue
		}
		for _, f := range namedIOFiles(out) {
			if !regularFile(workspaceFile(isolate, f.path)) {
				return errTreeMissing
			}
		}
	}
	return nil
}

func publishAll(workspace, isolate string, task TaskPlan) error {
	var wrote []string
	rollback := func() {
		for _, p := range wrote {
			os.Remove(p)
		}
	}
	for _, out := range task.Outputs {
		if isTreeIO(out) {
			added, err := publishTree(workspace, isolate, out, false)
			if err != nil {
				rollback()
				for _, p := range added {
					os.Remove(p)
				}
				return err
			}
			wrote = append(wrote, added...)
			continue
		}
		for _, f := range namedIOFiles(out) {
			src := workspaceFile(isolate, f.path)
			dst := workspaceFile(workspace, f.path)
			if err := exec.PublishFile(src, dst); err != nil {
				rollback()
				return err
			}
			wrote = append(wrote, dst)
		}
	}
	return nil
}

func publishReplace(workspace, isolate string, task TaskPlan) error {
	for _, out := range task.Outputs {
		if isTreeIO(out) {
			if _, err := publishTree(workspace, isolate, out, true); err != nil {
				return err
			}
			continue
		}
		for _, f := range namedIOFiles(out) {
			src := workspaceFile(isolate, f.path)
			dst := workspaceFile(workspace, f.path)
			if !pathPresent(dst) {
				if err := exec.PublishFile(src, dst); err != nil {
					return err
				}
				continue
			}
			if err := exec.StagedReplace(src, dst); err != nil {
				return err
			}
		}
	}
	return nil
}

func copyFile(src, dst string) error {
	return exec.CopyFile(src, dst)
}

func stagedReplace(src, dst string) error {
	return exec.StagedReplace(src, dst)
}

type namedFile struct {
	name   string
	path   string
	source string
}

func namedIOFiles(io IO) []namedFile {
	if isTreeIO(io) {
		return nil
	}
	if io.Members != nil {
		out := make([]namedFile, 0, len(io.Members))
		for _, m := range io.Members {
			found, ok := findIOMember(io.Members, m.Name)
			if !ok || found.Path == "" {
				continue
			}
			out = append(out, namedFile{name: found.Name, path: found.Path, source: found.Source})
		}
		return out
	}
	if io.Path == "" {
		return nil
	}
	return []namedFile{{name: io.Name, path: io.Path, source: io.Source}}
}

func fileSource(f namedFile) string {
	if f.source != "" {
		return f.source
	}
	return f.path
}

func findIOMember(members []IOMember, name string) (IOMember, bool) {
	for _, m := range members {
		if m.Name == name {
			return m, true
		}
	}
	return IOMember{}, false
}
