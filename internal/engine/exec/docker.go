package exec

import (
	"bytes"
	"errors"
	"io"
	"os"
	osexec "os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
)

const containerWorkDir = "/work"

// DockerCLI runs docker argv without the docker token. Tests replace it.
var DockerCLI = runDockerCLI

// Docker runs each job as a container.
type Docker struct {
	mu   sync.Mutex
	done map[string]Report
}

// NewDocker returns a docker adapter.
func NewDocker() *Docker {
	return &Docker{done: make(map[string]Report)}
}

// Submit starts a detached container. Image ENTRYPOINT is unused.
func (d *Docker) Submit(job Job) (Handle, Report, error) {
	if job.Image == "" {
		return Handle{}, Report{}, errors.New("empty image")
	}
	if len(job.Argv) == 0 {
		return Handle{}, Report{}, errors.New("empty command")
	}
	if err := ensureImage(job.Image); err != nil {
		return Handle{}, Report{}, err
	}
	var idBuf, errBuf bytes.Buffer
	args := dockerRunArgs(job)
	exit, err := DockerCLI(args, &idBuf, &errBuf)
	if err != nil {
		return Handle{}, Report{}, errors.New("docker: " + err.Error())
	}
	if exit != 0 {
		msg := strings.TrimSpace(errBuf.String())
		if msg == "" {
			msg = "exit " + strconv.Itoa(exit)
		}
		return Handle{}, Report{}, errors.New("docker: " + msg)
	}
	id := strings.TrimSpace(idBuf.String())
	if id == "" {
		return Handle{}, Report{}, errors.New("docker: empty container id")
	}
	digest := imageDigest(job.Image)
	h := Handle{Identity: job.Identity, Backend: BackendDocker, RuntimeID: id}
	return h, Report{
		Identity:    job.Identity,
		RuntimeID:   id,
		ImageDigest: digest,
		Running:     true,
	}, nil
}

// Poll uses docker inspect. After exit it copies logs and removes the container.
func (d *Docker) Poll(h Handle) (Report, error) {
	d.mu.Lock()
	if r, ok := d.done[h.RuntimeID]; ok {
		d.mu.Unlock()
		return r, nil
	}
	d.mu.Unlock()
	running, exit, err := inspectContainer(h.RuntimeID)
	if err != nil {
		r := Report{Identity: h.Identity, RuntimeID: h.RuntimeID, Running: false, Message: err.Error(), Exit: -1}
		d.store(h.RuntimeID, r)
		return r, nil
	}
	if running {
		return Report{Identity: h.Identity, RuntimeID: h.RuntimeID, Running: true}, nil
	}
	writeDockerLogs(h)
	_, _ = DockerCLI([]string{"rm", "-f", h.RuntimeID}, discard(), discard())
	msg := ""
	if exit != 0 {
		msg = "exit " + strconv.Itoa(exit)
	}
	r := Report{Identity: h.Identity, RuntimeID: h.RuntimeID, Exit: exit, Message: msg, Running: false}
	d.store(h.RuntimeID, r)
	return r, nil
}

// Cancel runs docker kill.
func (d *Docker) Cancel(h Handle) error {
	_, err := DockerCLI([]string{"kill", h.RuntimeID}, discard(), discard())
	if err != nil {
		return nil
	}
	return nil
}

// Reconcile uses docker inspect. A missing container is not running.
func (d *Docker) Reconcile(h Handle) (Report, error) {
	d.mu.Lock()
	if r, ok := d.done[h.RuntimeID]; ok {
		d.mu.Unlock()
		return r, nil
	}
	d.mu.Unlock()
	running, exit, err := inspectContainer(h.RuntimeID)
	if err != nil {
		return Report{Identity: h.Identity, RuntimeID: h.RuntimeID, Running: false}, nil
	}
	if running {
		return Report{Identity: h.Identity, RuntimeID: h.RuntimeID, Running: true}, nil
	}
	return Report{Identity: h.Identity, RuntimeID: h.RuntimeID, Exit: exit, Running: false}, nil
}

func (d *Docker) store(id string, r Report) {
	d.mu.Lock()
	d.done[id] = r
	d.mu.Unlock()
}

func dockerRunArgs(job Job) []string {
	abs := job.Isolate
	if resolved, err := filepath.Abs(job.Isolate); err == nil {
		abs = resolved
	}
	args := []string{
		"run", "-d",
		"--user", strconv.Itoa(os.Getuid()) + ":" + strconv.Itoa(os.Getgid()),
		"--network=none",
		"--entrypoint", job.Argv[0],
		"-v", abs + ":" + containerWorkDir,
		"-w", containerWorkDir,
	}
	if job.CPU != 0 {
		args = append(args, "--cpus", strconv.FormatFloat(job.CPU, 'f', -1, 64))
	}
	if n, ok := parseMemory(job.Memory); ok && n > 0 {
		args = append(args, "--memory", job.Memory)
	}
	keys := make([]string, 0, len(job.Env))
	for k, v := range job.Env {
		if k == "" || v == "" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		args = append(args, "-e", k+"="+job.Env[k])
	}
	args = append(args, job.Image)
	return append(args, job.Argv[1:]...)
}

func ensureImage(image string) error {
	if exit, err := DockerCLI([]string{"image", "inspect", image}, discard(), discard()); err == nil && exit == 0 {
		return nil
	} else if err != nil {
		return errors.New("docker: " + err.Error())
	}
	var buf bytes.Buffer
	exit, err := DockerCLI([]string{"pull", image}, discard(), &buf)
	if err != nil {
		return errors.New("docker pull: " + err.Error())
	}
	if exit != 0 {
		msg := strings.TrimSpace(buf.String())
		if msg == "" {
			msg = "exit " + strconv.Itoa(exit)
		}
		return errors.New("docker pull: " + msg)
	}
	return nil
}

func imageDigest(image string) string {
	var buf bytes.Buffer
	if exit, err := DockerCLI([]string{"image", "inspect", "--format", "{{with .RepoDigests}}{{index . 0}}{{end}}", image}, &buf, discard()); err != nil || exit != 0 {
		return ""
	}
	s := strings.TrimSpace(buf.String())
	if s != "" {
		return s
	}
	buf.Reset()
	if exit, err := DockerCLI([]string{"image", "inspect", "--format", "{{.Id}}", image}, &buf, discard()); err != nil || exit != 0 {
		return ""
	}
	return strings.TrimSpace(buf.String())
}

func inspectContainer(id string) (running bool, exit int, err error) {
	var buf, errBuf bytes.Buffer
	code, cliErr := DockerCLI([]string{"inspect", "--format", "{{.State.Running}} {{.State.ExitCode}}", id}, &buf, &errBuf)
	if cliErr != nil {
		return false, -1, errors.New("docker inspect: " + cliErr.Error())
	}
	if code != 0 {
		msg := strings.TrimSpace(errBuf.String())
		if msg == "" {
			msg = "exit " + strconv.Itoa(code)
		}
		return false, -1, errors.New("docker inspect: " + msg)
	}
	fields := strings.Fields(strings.TrimSpace(buf.String()))
	if len(fields) < 2 {
		return false, -1, errors.New("docker inspect: " + strings.TrimSpace(buf.String()))
	}
	running = fields[0] == "true"
	exit, _ = strconv.Atoi(fields[1])
	return running, exit, nil
}

func writeDockerLogs(h Handle) {
	// Isolate is not on Handle. Logs go to the attempt directory only when
	// the scheduler created stdout/stderr files next to work/. Callers that
	// need logs pass Isolate on Submit; Poll writes beside the volume by
	// inspecting the container's first bind mount when possible.
	var buf bytes.Buffer
	if exit, err := DockerCLI([]string{"inspect", "--format", "{{range .Mounts}}{{if eq .Destination \"" + containerWorkDir + "\"}}{{.Source}}{{end}}{{end}}", h.RuntimeID}, &buf, discard()); err != nil || exit != 0 {
		return
	}
	src := strings.TrimSpace(buf.String())
	if src == "" {
		return
	}
	attempt := filepath.Dir(src)
	outf, err := os.Create(filepath.Join(attempt, "stdout"))
	if err != nil {
		return
	}
	errf, err := os.Create(filepath.Join(attempt, "stderr"))
	if err != nil {
		outf.Close()
		return
	}
	_, _ = DockerCLI([]string{"logs", h.RuntimeID}, outf, errf)
	outf.Close()
	errf.Close()
}

func parseMemory(s string) (int64, bool) {
	if s == "" {
		return 0, true
	}
	if n, err := strconv.ParseUint(s, 10, 63); err == nil {
		return int64(n), true
	}
	if len(s) < 2 {
		return 0, false
	}
	var mul int64
	switch s[len(s)-1] {
	case 'b', 'B':
		mul = 1
	case 'k', 'K':
		mul = 1024
	case 'm', 'M':
		mul = 1024 * 1024
	case 'g', 'G':
		mul = 1024 * 1024 * 1024
	default:
		return 0, false
	}
	num := s[:len(s)-1]
	if n, err := strconv.ParseUint(num, 10, 63); err == nil {
		return int64(n) * mul, true
	}
	return 0, false
}

func runDockerCLI(args []string, stdout, stderr io.Writer) (int, error) {
	cmd := osexec.Command("docker", args...)
	cmd.Env = []string{"PATH=/usr/bin:/bin"}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	err := cmd.Run()
	if err == nil {
		return 0, nil
	}
	var ee *osexec.ExitError
	if errors.As(err, &ee) {
		return ee.ExitCode(), nil
	}
	return -1, err
}
