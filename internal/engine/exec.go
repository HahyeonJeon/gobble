package engine

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
)

var errNotRegular = errors.New("not a regular file")

// report is the executor result. Executors do not write state files.
type report struct {
	ID        string
	Exit      int
	Message   string
	Stdout    string
	Stderr    string
	Published bool
}

type runner func(cwd string, argv []string, stdout, stderr io.Writer) (int, error)

func executeTask(workspace string, task TaskPlan) report {
	if task.Image != "" {
		return isolatedExecute(workspace, task, dockerRunner(task.Image))
	}
	return isolatedExecute(workspace, task, runProcess)
}

func isolatedExecute(workspace string, task TaskPlan, run runner) report {
	r := report{ID: task.ID}
	taskDir := filepath.Join(workspace, ControlDir, "tasks", task.ID)
	isolate := filepath.Join(taskDir, "work")
	r.Stdout = ControlDir + "/tasks/" + task.ID + "/stdout"
	r.Stderr = ControlDir + "/tasks/" + task.ID + "/stderr"
	if err := os.MkdirAll(isolate, 0o755); err != nil {
		r.Message = err.Error()
		return r
	}
	if err := prepareIsolate(workspace, isolate, task); err != nil {
		r.Message = err.Error()
		return r
	}
	stdoutPath := filepath.Join(workspace, r.Stdout)
	stderrPath := filepath.Join(workspace, r.Stderr)
	outf, err := os.Create(stdoutPath)
	if err != nil {
		r.Message = err.Error()
		return r
	}
	errf, err := os.Create(stderrPath)
	if err != nil {
		outf.Close()
		r.Message = err.Error()
		return r
	}
	exit, runErr := run(isolate, task.Command, outf, errf)
	outf.Close()
	errf.Close()
	r.Exit = exit
	if runErr != nil {
		r.Message = runErr.Error()
		return r
	}
	if exit != 0 {
		r.Message = "exit " + strconv.Itoa(exit)
		return r
	}
	if missing := missingOutputs(isolate, task); missing != "" {
		r.Message = "missing output"
		return r
	}
	if err := publishAll(workspace, isolate, task); err != nil {
		r.Message = err.Error()
		return r
	}
	r.Published = true
	return r
}

func prepareIsolate(workspace, isolate string, task TaskPlan) error {
	for _, in := range task.Inputs {
		if err := mkdirPlanParent(isolate, in.Path); err != nil {
			return err
		}
	}
	for _, out := range task.Outputs {
		if err := mkdirPlanParent(isolate, out.Path); err != nil {
			return err
		}
	}
	for _, in := range task.Inputs {
		src := workspaceFile(workspace, in.Path)
		dst := workspaceFile(isolate, in.Path)
		if err := copyFile(src, dst); err != nil {
			return err
		}
		if err := os.Chmod(dst, 0o444); err != nil {
			return err
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

func runProcess(cwd string, argv []string, stdout, stderr io.Writer) (int, error) {
	if len(argv) == 0 {
		return -1, errors.New("empty command")
	}
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Dir = cwd
	cmd.Env = []string{"PATH=/usr/bin:/bin"}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	err := cmd.Run()
	if err == nil {
		return 0, nil
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return ee.ExitCode(), nil
	}
	return -1, err
}

func missingOutputs(isolate string, task TaskPlan) string {
	for _, out := range task.Outputs {
		if !regularFile(workspaceFile(isolate, out.Path)) {
			return out.Path
		}
	}
	return ""
}

func publishAll(workspace, isolate string, task TaskPlan) error {
	var wrote []string
	for _, out := range task.Outputs {
		src := workspaceFile(isolate, out.Path)
		dst := workspaceFile(workspace, out.Path)
		if err := copyFile(src, dst); err != nil {
			for _, p := range wrote {
				os.Remove(p)
			}
			return err
		}
		wrote = append(wrote, dst)
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := openReadFile(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		os.Remove(dst)
		return copyErr
	}
	if closeErr != nil {
		os.Remove(dst)
		return closeErr
	}
	return nil
}

func openReadFile(path string) (*os.File, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, errNotRegular
	}
	return os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW, 0)
}
