package exec

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"
)

const submissionLabel = "io.gobble.submission"

type dockerEndpointKey struct{}

func validSubmissionToken(token string) bool {
	if len(token) < 32 || len(token) > 128 {
		return false
	}
	_, err := hex.DecodeString(token)
	return err == nil
}

func submissionName(token string) string { return "gobble-" + token }

func bindDockerEndpoint(ctx context.Context, endpoint string) context.Context {
	return context.WithValue(ctx, dockerEndpointKey{}, endpoint)
}

func selectedDockerEndpoint(ctx context.Context) (string, error) {
	endpoint := os.Getenv("DOCKER_HOST")
	if endpoint == "" || os.Getenv("DOCKER_CONTEXT") != "" {
		var out bytes.Buffer
		code, err := dockerCLI(ctx, []string{"context", "inspect", "--format", "{{.Endpoints.docker.Host}}"}, &out, discard())
		if err != nil {
			return "", dockerErr(ctx, "docker context inspect", err)
		}
		if code != 0 {
			return "", fmt.Errorf("docker context inspect: exit %d", code)
		}
		endpoint = strings.TrimSpace(out.String())
	}
	if !strings.HasPrefix(endpoint, "unix:///") {
		return "", errors.New("docker: select a local Unix-socket Docker engine")
	}
	return endpoint, nil
}

func dockerDaemonID(ctx context.Context) (string, error) {
	var out bytes.Buffer
	code, err := dockerCLI(ctx, []string{"info", "--format", "{{.ID}}"}, &out, discard())
	if err != nil {
		return "", dockerErr(ctx, "docker info", err)
	}
	id := strings.TrimSpace(out.String())
	if code != 0 || id == "" {
		return "", errors.New("docker: cannot determine owning daemon identity")
	}
	return id, nil
}

func checkDockerDaemon(ctx context.Context, expected string) error {
	id, err := dockerDaemonID(ctx)
	if err != nil {
		return err
	}
	if id != expected {
		return errors.New("docker: owning daemon changed; restore the original Docker engine")
	}
	return nil
}

// resolveSubmission searches only the recorded daemon and attempt label. An
// empty successful listing proves absence; an inspect/connection error does not.
func resolveSubmission(ctx context.Context, h Handle) (context.Context, Handle, error) {
	s := h.Submission
	if s == nil || !validSubmissionToken(s.Token) {
		return ctx, h, errors.New("docker: invalid submission identity")
	}
	if s.Endpoint == "" && s.DaemonID == "" && !s.Created && h.RuntimeID == "" {
		// Admission was recorded, but no create was authorized yet.
		return ctx, h, nil
	}
	if !strings.HasPrefix(s.Endpoint, "unix:///") || s.DaemonID == "" {
		return ctx, h, errors.New("docker: incomplete owning daemon identity")
	}
	ctx = bindDockerEndpoint(ctx, s.Endpoint)
	if err := checkDockerDaemon(ctx, s.DaemonID); err != nil {
		return ctx, h, err
	}
	var out bytes.Buffer
	filter := "label=" + submissionLabel + "=" + s.Token
	if h.RuntimeID != "" {
		filter = "id=" + h.RuntimeID
	}
	args := []string{"container", "ls", "--all", "--no-trunc", "--filter", filter, "--format", "{{.ID}}"}
	code, err := dockerCLI(ctx, args, &out, discard())
	if err != nil {
		return ctx, h, dockerErr(ctx, "docker container ls", err)
	}
	if code != 0 {
		return ctx, h, fmt.Errorf("docker container ls: exit %d", code)
	}
	ids := strings.Fields(out.String())
	if len(ids) > 1 {
		return ctx, h, errors.New("docker: multiple containers match one submission")
	}
	if len(ids) == 0 {
		h.RuntimeID = ""
		return ctx, h, nil
	}
	if h.RuntimeID != "" && ids[0] != h.RuntimeID {
		return ctx, h, errors.New("docker: container identity does not match the recorded attempt")
	}
	out.Reset()
	code, err = dockerCLI(ctx, []string{"inspect", "--format", "{{index .Config.Labels \"" + submissionLabel + "\"}}", ids[0]}, &out, discard())
	if err != nil || code != 0 || strings.TrimSpace(out.String()) != s.Token {
		return ctx, h, errors.New("docker: cannot verify container submission ownership")
	}
	h.RuntimeID = ids[0]
	return ctx, h, nil
}

func missingSubmissionReport(h Handle) Report {
	return Report{Identity: h.Identity, Exit: -1, Reason: "container-missing",
		Message: "no container remains for the recorded submission"}
}
