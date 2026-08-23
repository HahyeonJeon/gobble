package exec

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	osexec "os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const containerWorkDir = "/work"

// DockerCLI runs docker argv without the docker token. env is the
// engine-owned client environment for that call. Tests replace it.
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
func (d *Docker) Submit(ctx context.Context, job Job) (Handle, Report, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if job.Image == "" {
		return Handle{}, Report{}, errors.New("empty image")
	}
	if len(job.Argv) == 0 {
		return Handle{}, Report{}, errors.New("empty command")
	}
	if err := ensureImage(ctx, job.Image); err != nil {
		return Handle{}, Report{}, err
	}
	var idBuf, errBuf bytes.Buffer
	args := dockerRunArgs(job)
	exit, err := dockerCLI(ctx, args, &idBuf, &errBuf)
	if err != nil {
		return Handle{}, Report{}, dockerErr(ctx, "docker", err)
	}
	if exit != 0 {
		msg := strings.TrimSpace(errBuf.String())
		if msg == "" {
			msg = "exit " + strconv.Itoa(exit)
		}
		return Handle{}, Report{}, dockerErr(ctx, "docker", errors.New(msg))
	}
	id := strings.TrimSpace(idBuf.String())
	if id == "" {
		return Handle{}, Report{}, errors.New("docker: empty container id")
	}
	digest := containerImageID(ctx, id)
	h := Handle{Identity: job.Identity, Backend: BackendDocker, RuntimeID: id}
	return h, Report{
		Identity:    job.Identity,
		RuntimeID:   id,
		ImageDigest: digest,
		Running:     true,
	}, nil
}

// Poll uses docker inspect. After exit it copies logs and removes the
// container. A cleanup failure is returned and is not cached as success.
func (d *Docker) Poll(ctx context.Context, h Handle) (Report, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	d.mu.Lock()
	if r, ok := d.done[h.RuntimeID]; ok {
		d.mu.Unlock()
		return r, nil
	}
	d.mu.Unlock()
	running, exit, err := inspectContainer(ctx, h.RuntimeID)
	if err != nil {
		return Report{Identity: h.Identity, RuntimeID: h.RuntimeID}, err
	}
	if running {
		return Report{Identity: h.Identity, RuntimeID: h.RuntimeID, Running: true}, nil
	}
	return d.finishStopped(ctx, h, exit)
}

// Cancel runs docker kill.
func (d *Docker) Cancel(ctx context.Context, h Handle) error {
	if ctx == nil {
		ctx = context.Background()
	}
	exit, err := dockerCLI(ctx, []string{"kill", h.RuntimeID}, discard(), discard())
	if err != nil {
		return dockerErr(ctx, "docker kill", err)
	}
	if exit != 0 {
		return dockerErr(ctx, "docker kill", errors.New("exit "+strconv.Itoa(exit)))
	}
	return nil
}

// Reconcile uses docker inspect. Inspect or cleanup errors leave disposition
// unproved.
func (d *Docker) Reconcile(ctx context.Context, h Handle) (Report, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	d.mu.Lock()
	if r, ok := d.done[h.RuntimeID]; ok {
		d.mu.Unlock()
		return r, nil
	}
	d.mu.Unlock()
	running, exit, err := inspectContainer(ctx, h.RuntimeID)
	if err != nil {
		return Report{Identity: h.Identity, RuntimeID: h.RuntimeID}, err
	}
	if running {
		return Report{Identity: h.Identity, RuntimeID: h.RuntimeID, Running: true}, nil
	}
	return d.finishStopped(ctx, h, exit)
}

func (d *Docker) finishStopped(ctx context.Context, h Handle, exit int) (Report, error) {
	msg := ""
	if exit != 0 {
		msg = "exit " + strconv.Itoa(exit)
	}
	r := Report{Identity: h.Identity, RuntimeID: h.RuntimeID, Exit: exit, Message: msg, Running: false}
	logErr := writeDockerLogs(ctx, h)
	rmErr := removeDockerContainer(ctx, h.RuntimeID)
	if err := errors.Join(logErr, rmErr); err != nil {
		return r, err
	}
	d.store(h.RuntimeID, r)
	return r, nil
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
	if job.MemoryBytes > 0 {
		args = append(args, "--memory", strconv.FormatInt(job.MemoryBytes, 10))
	}
	for _, k := range envKeys(job.Env) {
		args = append(args, "-e", k+"="+job.Env[k])
	}
	args = append(args, job.Image)
	return append(args, job.Argv[1:]...)
}

func ensureImage(ctx context.Context, image string) error {
	exit, err := dockerCLI(ctx, []string{"image", "inspect", image}, discard(), discard())
	if err == nil && exit == 0 {
		return nil
	}
	if err != nil {
		return dockerErr(ctx, "docker", err)
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	var buf bytes.Buffer
	exit, err = dockerCLI(ctx, []string{"pull", image}, discard(), &buf)
	if err != nil {
		return dockerErr(ctx, "docker pull", err)
	}
	if exit != 0 {
		msg := strings.TrimSpace(buf.String())
		if msg == "" {
			msg = "exit " + strconv.Itoa(exit)
		}
		return dockerErr(ctx, "docker pull", errors.New(msg))
	}
	return nil
}

func imageDigest(ctx context.Context, image string) string {
	var buf bytes.Buffer
	if exit, err := dockerCLI(ctx, []string{"image", "inspect", "--format", "{{.Id}}", image}, &buf, discard()); err != nil || exit != 0 {
		return ""
	}
	return strings.TrimSpace(buf.String())
}

func containerImageID(ctx context.Context, id string) string {
	if id == "" {
		return ""
	}
	var buf bytes.Buffer
	if exit, err := dockerCLI(ctx, []string{"inspect", "--format", "{{.Image}}", id}, &buf, discard()); err != nil || exit != 0 {
		return ""
	}
	return strings.TrimSpace(buf.String())
}

// LookupImageID returns the local image .Id without pulling. Empty means miss.
func LookupImageID(image string) string {
	if image == "" {
		return ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return imageDigest(ctx, image)
}

func envKeys(env map[string]string) []string {
	keys := make([]string, 0, len(env))
	for k, v := range env {
		if k == "" || v == "" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func dockerClientEnv() []string {
	return []string{"PATH=/usr/bin:/bin"}
}

func inspectContainer(ctx context.Context, id string) (running bool, exit int, err error) {
	var buf, errBuf bytes.Buffer
	code, cliErr := dockerCLI(ctx, []string{"inspect", "--format", "{{.State.Running}} {{.State.ExitCode}}", id}, &buf, &errBuf)
	if cliErr != nil {
		return false, -1, dockerErr(ctx, "docker inspect", cliErr)
	}
	if code != 0 {
		msg := strings.TrimSpace(errBuf.String())
		if msg == "" {
			msg = "exit " + strconv.Itoa(code)
		}
		return false, -1, dockerErr(ctx, "docker inspect", errors.New(msg))
	}
	fields := strings.Fields(strings.TrimSpace(buf.String()))
	if len(fields) < 2 {
		return false, -1, errors.New("docker inspect: " + strings.TrimSpace(buf.String()))
	}
	running = fields[0] == "true"
	exit, _ = strconv.Atoi(fields[1])
	return running, exit, nil
}

func writeDockerLogs(ctx context.Context, h Handle) error {
	// Isolate is not on Handle. Logs go to the attempt directory only when
	// the scheduler created stdout/stderr files next to work/. Callers that
	// need logs pass Isolate on Submit; Poll writes beside the volume by
	// inspecting the container's first bind mount when possible.
	var buf, errBuf bytes.Buffer
	exit, err := dockerCLI(ctx, []string{"inspect", "--format", "{{range .Mounts}}{{if eq .Destination \"" + containerWorkDir + "\"}}{{.Source}}{{end}}{{end}}", h.RuntimeID}, &buf, &errBuf)
	if err != nil {
		return dockerErr(ctx, "docker logs inspect", err)
	}
	if exit != 0 {
		msg := strings.TrimSpace(errBuf.String())
		if msg == "" {
			msg = "exit " + strconv.Itoa(exit)
		}
		return dockerErr(ctx, "docker logs inspect", errors.New(msg))
	}
	src := strings.TrimSpace(buf.String())
	if src == "" {
		return errors.New("docker logs: work mount not found")
	}
	attempt := filepath.Dir(src)
	outf, err := createAttemptFile(filepath.Join(attempt, "stdout"))
	if err != nil {
		return err
	}
	errf, err := createAttemptFile(filepath.Join(attempt, "stderr"))
	if err != nil {
		if closeErr := outf.Close(); closeErr != nil {
			return errors.Join(err, fmt.Errorf("close docker stdout: %w", closeErr))
		}
		return err
	}
	logExit, logErr := dockerCLI(ctx, []string{"logs", h.RuntimeID}, outf, errf)
	if logErr != nil {
		logErr = dockerErr(ctx, "docker logs", logErr)
	} else if logExit != 0 {
		logErr = dockerErr(ctx, "docker logs", errors.New("exit "+strconv.Itoa(logExit)))
	}
	if closeErr := outf.Close(); closeErr != nil {
		logErr = errors.Join(logErr, fmt.Errorf("close docker stdout: %w", closeErr))
	}
	if closeErr := errf.Close(); closeErr != nil {
		logErr = errors.Join(logErr, fmt.Errorf("close docker stderr: %w", closeErr))
	}
	return logErr
}

func removeDockerContainer(ctx context.Context, id string) error {
	var errBuf bytes.Buffer
	exit, err := dockerCLI(ctx, []string{"rm", "-f", id}, discard(), &errBuf)
	if err != nil {
		return dockerErr(ctx, "docker rm", err)
	}
	if exit == 0 {
		return nil
	}
	msg := strings.TrimSpace(errBuf.String())
	if msg == "" {
		msg = "exit " + strconv.Itoa(exit)
	}
	return dockerErr(ctx, "docker rm", errors.New(msg))
}

func dockerCLI(ctx context.Context, args []string, stdout, stderr io.Writer) (int, error) {
	return DockerCLI(ctx, args, dockerClientEnv(), stdout, stderr)
}

func runDockerCLI(ctx context.Context, args, env []string, stdout, stderr io.Writer) (int, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if len(env) == 0 {
		env = dockerClientEnv()
	}
	cmd := osexec.CommandContext(ctx, "docker", args...)
	cmd.Env = env
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	err := cmd.Run()
	if err == nil {
		return 0, nil
	}
	if cerr := ctx.Err(); cerr != nil {
		return -1, cerr
	}
	var ee *osexec.ExitError
	if errors.As(err, &ee) {
		return ee.ExitCode(), nil
	}
	return -1, err
}

func dockerErr(ctx context.Context, op string, err error) error {
	if err == nil {
		return nil
	}
	if ctx != nil {
		if cerr := ctx.Err(); cerr != nil {
			return cerr
		}
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return err
	}
	return fmt.Errorf("%s: %w", op, err)
}
