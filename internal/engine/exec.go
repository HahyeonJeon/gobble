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
		return isolatedExecute(workspace, task, dockerRunner(task))
	}
	return isolatedExecute(workspace, task, processRunner(task))
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
	exit, runErr := run(isolate, executeArgv(task), outf, errf)
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

func executeArgv(task TaskPlan) []string {
	if task.Script != "" {
		return []string{"sh", "-c", "set -eu\n" + task.Script}
	}
	return task.Command
}

func processRunner(task TaskPlan) runner {
	return func(cwd string, argv []string, stdout, stderr io.Writer) (int, error) {
		return runProcessWithEnv(cwd, argv, processEnv(task.Env), stdout, stderr)
	}
}

func processEnv(env map[string]string) []string {
	out := make([]string, 0, 1+len(env))
	if _, ok := env["PATH"]; !ok {
		out = append(out, "PATH=/usr/bin:/bin")
	}
	for k, v := range env {
		if k == "" || v == "" {
			continue
		}
		out = append(out, k+"="+v)
	}
	return out
}

func prepareIsolate(workspace, isolate string, task TaskPlan) error {
	for _, in := range task.Inputs {
		for _, f := range namedIOFiles(in) {
			if err := mkdirPlanParent(isolate, f.path); err != nil {
				return err
			}
		}
	}
	for _, out := range task.Outputs {
		for _, f := range namedIOFiles(out) {
			if err := mkdirPlanParent(isolate, f.path); err != nil {
				return err
			}
		}
	}
	for _, in := range task.Inputs {
		for _, f := range namedIOFiles(in) {
			src := workspaceFile(workspace, f.path)
			dst := workspaceFile(isolate, f.path)
			if err := copyFile(src, dst); err != nil {
				return err
			}
			if err := os.Chmod(dst, 0o444); err != nil {
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

func runProcess(cwd string, argv []string, stdout, stderr io.Writer) (int, error) {
	return runProcessWithEnv(cwd, argv, []string{"PATH=/usr/bin:/bin"}, stdout, stderr)
}

func runProcessWithEnv(cwd string, argv []string, env []string, stdout, stderr io.Writer) (int, error) {
	if len(argv) == 0 {
		return -1, errors.New("empty command")
	}
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Dir = cwd
	cmd.Env = env
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
		for _, f := range namedIOFiles(out) {
			if !regularFile(workspaceFile(isolate, f.path)) {
				return f.path
			}
		}
	}
	return ""
}

func publishAll(workspace, isolate string, task TaskPlan) error {
	var wrote []string
	for _, out := range task.Outputs {
		for _, f := range namedIOFiles(out) {
			src := workspaceFile(isolate, f.path)
			dst := workspaceFile(workspace, f.path)
			if err := copyFile(src, dst); err != nil {
				for _, p := range wrote {
					os.Remove(p)
				}
				return err
			}
			wrote = append(wrote, dst)
		}
	}
	return nil
}

type namedFile struct {
	name string
	path string
}

func namedIOFiles(io IO) []namedFile {
	if io.Members != nil {
		out := make([]namedFile, 0, len(io.Members))
		for _, m := range io.Members {
			found, ok := findIOMember(io.Members, m.Name)
			if !ok || found.Path == "" {
				continue
			}
			out = append(out, namedFile{name: found.Name, path: found.Path})
		}
		return out
	}
	if io.Path == "" {
		return nil
	}
	return []namedFile{{name: io.Name, path: io.Path}}
}

func findIOMember(members []IOMember, name string) (IOMember, bool) {
	for _, m := range members {
		if m.Name == name {
			return m, true
		}
	}
	return IOMember{}, false
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
