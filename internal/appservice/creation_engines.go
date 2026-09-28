package appservice

import (
	"context"
	"net/http"
	"os"
	"strings"
	"time"
)

const creationEngineLabel = "io.gobble.creation.scope=single-end-trim-fastqc-v1"

type creationEngine struct {
	EngineID string `json:"engineId"`
	Label    string `json:"label"`
}
type creationEngines struct {
	Ready   bool             `json:"ready"`
	Issue   string           `json:"issue"`
	Engines []creationEngine `json:"engines"`
}

// Discovery returns display identities only. The host resolves and verifies the local binding.
func (s *Service) creationEngineBindings(ctx context.Context) ([]RuntimeBinding, error) {
	return s.analysisEngineBindings(ctx, creationEngineLabel)
}

func (s *Service) analysisEngineBindings(ctx context.Context, label string) ([]RuntimeBinding, error) {
	endpoint := os.Getenv("DOCKER_HOST")
	if endpoint == "" || os.Getenv("DOCKER_CONTEXT") != "" {
		raw, err := s.runtime.command(ctx, []string{"context", "inspect", "--format", "{{.Endpoints.docker.Host}}"}, os.Environ())
		if err != nil {
			return nil, err
		}
		endpoint = strings.TrimSpace(string(raw))
	}
	if !strings.HasPrefix(endpoint, "unix:///") || strings.ContainsAny(endpoint, "\x00\r\n") {
		return nil, problem("incompatible_runtime", "Connect a local Docker engine.")
	}
	raw, err := s.runtime.commandAt(ctx, endpoint, "info", "--format", "{{.ID}} {{.OSType}}")
	if err != nil {
		return nil, err
	}
	info := strings.Fields(string(raw))
	if len(info) != 2 || info[1] != "linux" {
		return nil, problem("incompatible_runtime", "A local Linux engine is required.")
	}
	raw, err = s.runtime.commandAt(ctx, endpoint, "image", "ls", "--no-trunc", "--quiet", "--filter", "label="+label)
	if err != nil {
		return nil, err
	}
	out := []RuntimeBinding{}
	seen := map[string]bool{}
	for _, id := range strings.Fields(string(raw)) {
		if !digestPattern.MatchString(id) || seen[id] {
			continue
		}
		seen[id] = true
		b := RuntimeBinding{endpoint, info[0], id, "linux/amd64", "/gobble/project", "/gobble/project/workspace"}
		if s.runtime.verify(ctx, b) == nil {
			out = append(out, b)
		}
		if len(out) == 8 {
			break
		}
	}
	return out, nil
}
func (s *Service) creationEnginesStatus(ctx context.Context) creationEngines {
	result := creationEngines{Engines: []creationEngine{}}
	if saved, err := s.readCreationRuntime(); err == nil {
		if s.runtime.verify(ctx, saved.Binding) == nil {
			result.Ready = true
			result.Engines = append(result.Engines, creationEngine{saved.Binding.ImageID, "Connected analysis engine"})
			return result
		}
	}
	bindings, err := s.creationEngineBindings(ctx)
	if err != nil {
		result.Issue = "The local analysis engine is unavailable. Start Docker and refresh."
		return result
	}
	for _, b := range bindings {
		result.Engines = append(result.Engines, creationEngine{b.ImageID, "Single-end · Trim + FastQC · " + b.ImageID[7:19]})
	}
	if len(result.Engines) == 0 {
		result.Issue = "No compatible analysis engine is installed on this computer."
	} else {
		result.Issue = "Connect an installed engine to check Agent proposals."
	}
	return result
}
func (s *Service) connectCreationEngine(ctx context.Context, id string) (creationEngines, error) {
	if !digestPattern.MatchString(id) {
		return creationEngines{}, problem("invalid_request", "Choose an installed engine.")
	}
	bindings, err := s.creationEngineBindings(ctx)
	if err != nil {
		return creationEngines{}, err
	}
	for _, b := range bindings {
		if b.ImageID == id {
			if _, err = s.bindCreationRuntime(ctx, b); err != nil {
				return creationEngines{}, err
			}
			return s.creationEnginesStatus(ctx), nil
		}
	}
	return creationEngines{}, problem("stale_revision", "This engine is no longer available. Refresh the engine list.")
}
func (s *Service) registerCreationEngines(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/creation-engines", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()
		reply(w, s.creationEnginesStatus(ctx), nil)
	})
	mux.HandleFunc("POST /v1/creation-runtime/connect", func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			EngineID string `json:"engineId"`
		}
		if err := decodeDraftRequest(w, r, &input, "engineId"); err != nil {
			reply(w, nil, err)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()
		value, err := s.connectCreationEngine(ctx, input.EngineID)
		reply(w, value, err)
	})
}
