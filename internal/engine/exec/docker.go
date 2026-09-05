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
	mu      sync.Mutex
	done    map[string]Report
	streams map[string]*logStream
}

// NewDocker returns a docker adapter.
func NewDocker() *Docker {
	return &Docker{done: make(map[string]Report)}
}

func (d *Docker) DurableDocker() bool { return true }

// Submit creates a stopped container, durably records its ID through the
// scheduler, then starts it. Image ENTRYPOINT is unused.
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
	if job.Submission == nil || !validSubmissionToken(job.Submission.Token) || job.Record == nil {
		return Handle{}, Report{}, errors.New("docker: durable submission recorder is required")
	}
	submission := *job.Submission
	endpoint, err := selectedDockerEndpoint(ctx)
	if err != nil {
		return Handle{}, Report{}, err
	}
	submission.Endpoint = endpoint
	ctx = bindDockerEndpoint(ctx, endpoint)
	submission.DaemonID, err = dockerDaemonID(ctx)
	if err != nil {
		return Handle{}, Report{}, err
	}
	h := Handle{Identity: job.Identity, Backend: BackendDocker, Isolate: job.Isolate, Submission: &submission}
	if err := job.Record(ctx, h, Report{Identity: job.Identity}); err != nil {
		return Handle{}, Report{}, err
	}
	if err := ensureImage(ctx, job.Image); err != nil {
		return Handle{}, Report{}, err
	}
	if err := checkDockerDaemon(ctx, submission.DaemonID); err != nil {
		return Handle{}, Report{}, err
	}
	var idBuf, errBuf bytes.Buffer
	bindJob := job
	bindJob.Isolate, err = daemonIsolate(ctx, job.Isolate)
	if err != nil {
		return h, Report{}, err
	}
	args := dockerCreateArgs(bindJob)
	args = append(args[:1], append([]string{"--name", submissionName(submission.Token),
		"--label", submissionLabel + "=" + submission.Token}, args[1:]...)...)
	exit, err := dockerCLI(ctx, args, &idBuf, &errBuf)
	if err != nil {
		return h, Report{}, dockerErr(ctx, "docker create", err)
	}
	if exit != 0 {
		msg := strings.TrimSpace(errBuf.String())
		if msg == "" {
			msg = "exit " + strconv.Itoa(exit)
		}
		return h, Report{}, dockerErr(ctx, "docker create", errors.New(msg))
	}
	id := strings.TrimSpace(idBuf.String())
	if id == "" {
		return h, Report{}, errors.New("docker: empty container id")
	}
	digest := containerImageID(ctx, id)
	created := submission
	created.Created = true
	h.RuntimeID = id
	h.Submission = &created
	r := Report{
		Identity:    job.Identity,
		RuntimeID:   id,
		ImageDigest: digest,
	}
	if err := job.Record(ctx, h, r); err != nil {
		return h, r, err
	}
	if err := ctx.Err(); err != nil {
		return h, r, err
	}
	if err := checkDockerDaemon(ctx, created.DaemonID); err != nil {
		return h, r, err
	}
	errBuf.Reset()
	exit, err = dockerCLI(ctx, []string{"start", id}, discard(), &errBuf)
	if err != nil {
		return h, r, dockerErr(ctx, "docker start", err)
	}
	if exit != 0 {
		return h, r, fmt.Errorf("docker start: exit %d: %s", exit, strings.TrimSpace(errBuf.String()))
	}
	r.Running = true
	return h, r, nil
}

// Poll uses docker inspect. After a proved stop it caches the exit before
// treating log copy or container removal as cleanup.
func (d *Docker) Poll(ctx context.Context, h Handle) (Report, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if h.Submission != nil {
		var err error
		ctx, h, err = resolveSubmission(ctx, h)
		if err != nil {
			return Report{Identity: h.Identity, RuntimeID: h.RuntimeID}, err
		}
		if h.RuntimeID == "" {
			return missingSubmissionReport(h), nil
		}
	}
	if r, ok := d.cached(ctx, h.RuntimeID); ok {
		return r, nil
	}
	running, exit, err := inspectContainer(ctx, h.RuntimeID)
	if err != nil {
		return Report{Identity: h.Identity, RuntimeID: h.RuntimeID}, err
	}
	if running {
		d.followLogs(ctx, h)
		return Report{Identity: h.Identity, RuntimeID: h.RuntimeID, Running: true}, nil
	}
	return d.finishStopped(ctx, h, exit)
}

// Cancel runs docker kill.
func (d *Docker) Cancel(ctx context.Context, h Handle) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if h.Submission != nil {
		var err error
		ctx, h, err = resolveSubmission(ctx, h)
		if err != nil || h.RuntimeID == "" {
			return err
		}
		if err := d.stopLogs(ctx, h.RuntimeID); err != nil {
			return err
		}
		// Removing the exact owned container also fences an outstanding start
		// request. Observing "created" alone would not prove it cannot start.
		r := Report{Identity: h.Identity, Exit: 137, Reason: "canceled"}
		if err := writeDockerLogs(ctx, h); err != nil {
			r.Reason = "log-copy-failed"
		}
		if err := removeDockerContainer(ctx, h.RuntimeID); err != nil {
			return err
		}
		d.store(h.RuntimeID, r)
		return nil
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

// Reconcile uses docker inspect. Inspect errors leave disposition unproved.
func (d *Docker) Reconcile(ctx context.Context, h Handle) (Report, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if h.Submission != nil {
		var err error
		ctx, h, err = resolveSubmission(ctx, h)
		if err != nil {
			return Report{Identity: h.Identity, RuntimeID: h.RuntimeID}, err
		}
		if h.RuntimeID == "" {
			return missingSubmissionReport(h), nil
		}
		running, exit, err := inspectContainer(ctx, h.RuntimeID)
		if err != nil {
			return Report{Identity: h.Identity, RuntimeID: h.RuntimeID}, err
		}
		// Observed running state and the barrier against an outstanding start
		// are separate facts. Settlement removes the container in either case.
		return Report{Identity: h.Identity, RuntimeID: h.RuntimeID,
			Running: running, Exit: exit, NeedsRemoval: true}, nil
	}
	if r, ok := d.cached(ctx, h.RuntimeID); ok {
		return r, nil
	}
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
	if err := d.stopLogs(ctx, h.RuntimeID); err != nil {
		return Report{Identity: h.Identity, RuntimeID: h.RuntimeID}, err
	}
	msg := ""
	if exit != 0 {
		msg = "exit " + strconv.Itoa(exit)
	}
	r := Report{Identity: h.Identity, RuntimeID: h.RuntimeID, Exit: exit, Message: msg, Running: false}
	if err := writeDockerLogs(ctx, h); err != nil {
		r.Reason = "log-copy-failed"
	}
	if err := removeDockerContainer(ctx, h.RuntimeID); err == nil {
		r.RuntimeID = ""
	}
	d.store(h.RuntimeID, r)
	return r, nil
}

func (d *Docker) cached(ctx context.Context, id string) (Report, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	r, ok := d.done[id]
	if !ok {
		return Report{}, false
	}
	if r.RuntimeID != "" {
		if err := removeDockerContainer(ctx, r.RuntimeID); err == nil {
			r.RuntimeID = ""
			d.done[id] = r
		}
	}
	return r, true
}

func (d *Docker) store(id string, r Report) {
	d.mu.Lock()
	d.done[id] = r
	d.mu.Unlock()
}

func dockerCreateArgs(job Job) []string {
	abs := job.Isolate
	if resolved, err := filepath.Abs(job.Isolate); err == nil {
		abs = resolved
	}
	args := []string{
		"create",
		"--platform", "linux/amd64",
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
	exit, err = dockerCLI(ctx, []string{"pull", "--platform", "linux/amd64", image}, discard(), &buf)
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
	env := []string{}
	for _, key := range []string{"PATH", "HOME", "XDG_RUNTIME_DIR", "DOCKER_CONFIG", "DOCKER_CONTEXT", "DOCKER_HOST", "DOCKER_TLS_VERIFY", "DOCKER_CERT_PATH"} {
		if value, ok := os.LookupEnv(key); ok {
			env = append(env, key+"="+value)
		}
	}
	return env
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
	raw := strings.TrimSpace(buf.String())
	fields := strings.Fields(raw)
	if len(fields) != 2 {
		return false, -1, errors.New("docker inspect: " + strings.TrimSpace(buf.String()))
	}
	running, err = strconv.ParseBool(fields[0])
	if err != nil {
		return false, -1, fmt.Errorf("docker inspect running %q: %w", fields[0], err)
	}
	exit, err = strconv.Atoi(fields[1])
	if err != nil {
		return false, -1, fmt.Errorf("docker inspect exit %q: %w", fields[1], err)
	}
	return running, exit, nil
}

func writeDockerLogs(ctx context.Context, h Handle) error {
	src := h.Isolate
	// Legacy handles have no controller path. New handles always use the
	// controller's isolate, never a daemon-host mount path as a local log path.
	if src == "" {
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
		src = strings.TrimSpace(buf.String())
		if src == "" {
			return errors.New("docker logs: work mount not found")
		}
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
	if ctx == nil {
		ctx = context.Background()
	}
	return DockerCLI(ctx, args, dockerEnvForContext(ctx), stdout, stderr)
}

func dockerEnvForContext(ctx context.Context) []string {
	env := dockerClientEnv()
	if endpoint, _ := ctx.Value(dockerEndpointKey{}).(string); endpoint != "" {
		filtered := env[:0]
		for _, item := range env {
			if !strings.HasPrefix(item, "DOCKER_CONTEXT=") && !strings.HasPrefix(item, "DOCKER_HOST=") {
				filtered = append(filtered, item)
			}
		}
		env = append(filtered, "DOCKER_HOST="+endpoint)
	}
	return env
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
