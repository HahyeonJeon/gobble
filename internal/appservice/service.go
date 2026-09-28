package appservice

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Service is the sole catalog owner. Close only after its HTTP handlers have stopped.
type Service struct {
	store               *store
	runtime             runtimeAdapter
	mu                  sync.Mutex
	queries             map[string]*queryFlight
	inspections         map[string]*pipelineFlight
	proposals           map[string]*pipelineFlight
	creations           map[string]*pipelineFlight
	preparations        map[string]*pipelineFlight
	launchChecks        map[string]*pipelineFlight
	launchFlights       map[string]*pipelineFlight
	continuationFlights map[string]*pipelineFlight
	runLocks            map[string]chan struct{}
	ctx                 context.Context
	cancel              context.CancelFunc
	workers             sync.WaitGroup
	closeOnce           sync.Once
	closeErr            error
	closing             bool
}
type queryFlight struct {
	done   chan struct{}
	result snapshotResult
	err    error
}

// Open acquires the profile catalog lock; callers own Close and the context lifetime.
func Open(ctx context.Context, profile string) (*Service, error) {
	catalog, err := openStore(filepath.Join(profile, "service"))
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(ctx)
	return &Service{store: catalog, runtime: runtimeAdapter{command: runDocker}, queries: map[string]*queryFlight{}, runLocks: map[string]chan struct{}{}, ctx: ctx, cancel: cancel}, nil
}
func (s *Service) Close() error {
	s.closeOnce.Do(func() {
		s.mu.Lock()
		s.closing = true
		s.mu.Unlock()
		s.cancel()
		s.workers.Wait()
		s.closeErr = s.store.close()
	})
	return s.closeErr
}

// Handler accepts only the authenticated native host at its exact loopback authority.
func (s *Service) Handler(authority, token string) http.Handler {
	mux := http.NewServeMux()
	s.proposalRoutes(mux)
	s.preparationRoutes(mux)
	s.launchRoutes(mux)
	s.continuationRoutes(mux)
	s.creationRoutes(mux)
	s.creationCheckRoutes(mux)
	mux.HandleFunc("GET /v1/capabilities", func(w http.ResponseWriter, r *http.Request) {
		writeValue(w, map[string]any{"protocolVersion": 1, "queries": []string{"projects", "files", "pipelines", "pipeline_inspection", "pipeline_preparations", "launch_reviews", "continuations", "continuation_support", "preparation_engines", "pipeline_proposals", "proposal_source", "adoption_outcome", "pipeline_drafts", "creation_runtime", "creation_source", "creation_candidate", "runs", "snapshot", "logs"}, "mutations": []string{"register_project", "register_pipeline", "import_pipeline", "check_pipeline", "prepare_pipeline", "cancel_preparation", "check_launch", "cancel_launch_check", "start_launch", "refresh_launch", "stop_launch", "check_continuation", "confirm_continuation", "refresh_continuation", "stop_continuation", "cancel_pipeline_check", "propose_pipeline", "adopt_pipeline", "create_pipeline_draft", "update_pipeline_draft", "discard_pipeline_draft", "bind_creation_runtime", "submit_creation_candidate", "cancel_creation_check", "attach_run"}})
	})
	mux.HandleFunc("GET /v1/projects", func(w http.ResponseWriter, r *http.Request) { writeValue(w, s.store.projects()) })
	mux.HandleFunc("POST /v1/projects", func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			RequestID string `json:"requestId"`
			Root      string `json:"root"`
			Name      string `json:"name"`
		}
		if err := decodeRequest(w, r, &input, "requestId", "root", "name"); err != nil {
			writeProblem(w, err)
			return
		}
		value, err := s.store.registerProject(input.RequestID, input.Root, input.Name)
		reply(w, value, err)
	})
	mux.HandleFunc("GET /v1/projects/{project}/files", func(w http.ResponseWriter, r *http.Request) {
		value, err := s.store.listFiles(r.PathValue("project"), r.URL.Query().Get("directoryId"))
		reply(w, value, err)
	})
	mux.HandleFunc("GET /v1/projects/{project}/files/{resource}", func(w http.ResponseWriter, r *http.Request) {
		value, err := s.store.readFile(r.PathValue("project"), r.PathValue("resource"), r.URL.Query().Get("expectedRevision"))
		reply(w, value, err)
	})
	mux.HandleFunc("GET /v1/projects/{project}/pipelines", func(w http.ResponseWriter, r *http.Request) {
		value, err := s.store.listPipelines(r.PathValue("project"))
		reply(w, value, err)
	})
	mux.HandleFunc("POST /v1/projects/{project}/pipelines", func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			RequestID         string `json:"requestId"`
			PackageResourceID string `json:"packageResourceId"`
			Name              string `json:"name"`
		}
		if err := decodeRequest(w, r, &input, "requestId", "packageResourceId", "name"); err != nil {
			writeProblem(w, err)
			return
		}
		value, err := s.store.registerPipeline(r.PathValue("project"), input.PackageResourceID, input.RequestID, input.Name)
		reply(w, value, err)
	})
	mux.HandleFunc("POST /v1/projects/{project}/pipelines/import", func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			RequestID string `json:"requestId"`
			Path      string `json:"path"`
		}
		if err := decodeRequest(w, r, &input, "requestId", "path"); err != nil {
			writeProblem(w, err)
			return
		}
		value, err := s.store.importPipeline(r.PathValue("project"), input.RequestID, input.Path)
		reply(w, value, err)
	})
	mux.HandleFunc("GET /v1/projects/{project}/pipelines/{pipeline}/inspection", func(w http.ResponseWriter, r *http.Request) {
		value, err := s.pipelineInspection(r.PathValue("project"), r.PathValue("pipeline"))
		reply(w, value, err)
	})
	mux.HandleFunc("POST /v1/projects/{project}/pipelines/{pipeline}/check", func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			RequestID string `json:"requestId"`
		}
		if err := decodeRequest(w, r, &input, "requestId"); err != nil {
			writeProblem(w, err)
			return
		}
		value, err := s.checkPipeline(r.PathValue("project"), r.PathValue("pipeline"), input.RequestID)
		reply(w, value, err)
	})
	mux.HandleFunc("POST /v1/projects/{project}/pipelines/{pipeline}/cancel", func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			JobID string `json:"jobId"`
		}
		if err := decodeRequest(w, r, &input, "jobId"); err != nil {
			writeProblem(w, err)
			return
		}
		value, err := s.cancelPipeline(r.PathValue("project"), r.PathValue("pipeline"), input.JobID)
		reply(w, value, err)
	})
	mux.HandleFunc("GET /v1/projects/{project}/runs", func(w http.ResponseWriter, r *http.Request) {
		value, err := s.store.listRuns(r.PathValue("project"))
		reply(w, value, err)
	})
	mux.HandleFunc("POST /v1/projects/{project}/runs/attach", func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			RequestID           string `json:"requestId"`
			WorkspaceResourceID string `json:"workspaceResourceId"`
		}
		if err := decodeRequest(w, r, &input, "requestId", "workspaceResourceId"); err != nil {
			writeProblem(w, err)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
		defer cancel()
		value, err := s.attachRun(ctx, r.PathValue("project"), input.WorkspaceResourceID, input.RequestID)
		reply(w, value, err)
	})
	mux.HandleFunc("GET /v1/projects/{project}/runs/{run}/snapshot", func(w http.ResponseWriter, r *http.Request) {
		value, err := s.coalescedQuery(r.Context(), r.PathValue("project"), r.PathValue("run"), "", 0)
		reply(w, value, err)
	})
	mux.HandleFunc("GET /v1/projects/{project}/runs/{run}/report", func(w http.ResponseWriter, r *http.Request) {
		instance := r.URL.Query().Get("instance")
		attempt, err := strconv.Atoi(r.URL.Query().Get("attempt"))
		if err != nil || attempt < 1 || !validInstance(instance) {
			writeProblem(w, problem("invalid_request", "A task instance and positive attempt are required."))
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()
		value, err := s.readRunReport(ctx, r.PathValue("project"), r.PathValue("run"), instance, attempt)
		reply(w, value, err)
	})
	mux.HandleFunc("GET /v1/projects/{project}/runs/{run}/logs", func(w http.ResponseWriter, r *http.Request) {
		instance := r.URL.Query().Get("instance")
		attempt, err := strconv.Atoi(r.URL.Query().Get("attempt"))
		if err != nil || attempt < 1 || !validInstance(instance) {
			writeProblem(w, problem("invalid_request", "A task instance and positive attempt are required."))
			return
		}
		snapshot, err := s.coalescedQuery(r.Context(), r.PathValue("project"), r.PathValue("run"), instance, attempt)
		if err != nil {
			writeProblem(w, err)
			return
		}
		header, err := readMonitor(snapshot.Snapshot)
		reply(w, logsResult{snapshot.ProjectID, snapshot.RunRef, snapshot.EngineRevision, snapshot.ObservedAt, instance, attempt, 4096, header.Logs}, err)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		writeProblem(w, problem("unsupported", "This service route or method is not supported."))
	})
	semaphore := make(chan struct{}, 16)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		if s.closing {
			s.mu.Unlock()
			writeProblem(w, problem("runtime_unavailable", "The Project service is shutting down."))
			return
		}
		s.workers.Add(1)
		s.mu.Unlock()
		defer s.workers.Done()
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		if r.Host != authority || len(r.Header.Values("Origin")) != 0 || len(r.Header.Values("Authorization")) != 1 || subtle.ConstantTimeCompare([]byte(r.Header.Get("Authorization")), []byte("Bearer "+token)) != 1 {
			writeProblem(w, problem("forbidden", "This request is not authorized."))
			return
		}
		if len(r.URL.RequestURI()) > 8192 {
			writeProblem(w, problem("invalid_request", "The request URL is too long."))
			return
		}
		select {
		case semaphore <- struct{}{}:
			defer func() { <-semaphore }()
		default:
			writeProblem(w, problem("runtime_unavailable", "The service is busy. Try again shortly."))
			return
		}
		s.store.mu.Lock()
		failed := s.store.failed
		s.store.mu.Unlock()
		if failed {
			writeProblem(w, problem("internal", "Catalog storage is unavailable. Restart the app before continuing."))
			return
		}
		mux.ServeHTTP(w, r)
	})
}

func (s *Service) coalescedQuery(ctx context.Context, projectID, runRef, instance string, attempt int) (snapshotResult, error) {
	if ctx.Err() != nil {
		return snapshotResult{}, problem("runtime_unavailable", "The Run query was interrupted.")
	}
	// Verify ownership before allocating a per-run queue.
	s.store.mu.Lock()
	exists := false
	for _, run := range s.store.data.Runs {
		if run.ProjectID == projectID && run.RunRef == runRef {
			exists = true
			break
		}
	}
	s.store.mu.Unlock()
	if !exists {
		return snapshotResult{}, problem("not_found", "Run not found in this Project.")
	}
	key := requestDigest([]string{projectID, runRef, instance, strconv.Itoa(attempt)})
	s.mu.Lock()
	if s.closing {
		s.mu.Unlock()
		return snapshotResult{}, problem("runtime_unavailable", "The Project service is shutting down.")
	}
	flight := s.queries[key]
	if flight == nil {
		if len(s.queries) >= 16 {
			s.mu.Unlock()
			return snapshotResult{}, problem("runtime_unavailable", "The Run query queue is full. Try again shortly.")
		}
		flight = &queryFlight{done: make(chan struct{})}
		s.queries[key] = flight
		lock := s.runLocks[runRef]
		if lock == nil {
			lock = make(chan struct{}, 1)
			s.runLocks[runRef] = lock
		}
		s.workers.Add(1)
		go func() {
			defer s.workers.Done()
			queryCtx, cancel := context.WithTimeout(s.ctx, 20*time.Second)
			defer cancel()
			select {
			case lock <- struct{}{}:
				flight.result, flight.err = s.queryRun(queryCtx, projectID, runRef, instance, attempt)
				<-lock
			case <-queryCtx.Done():
				flight.err = problem("runtime_unavailable", "The Run query timed out.")
			}
			s.mu.Lock()
			delete(s.queries, key)
			close(flight.done)
			s.mu.Unlock()
		}()
	}
	s.mu.Unlock()
	select {
	case <-ctx.Done():
		return snapshotResult{}, problem("runtime_unavailable", "The Run query was interrupted.")
	case <-flight.done:
		return flight.result, flight.err
	}
}

func decodeRequest(w http.ResponseWriter, r *http.Request, target any, fields ...string) error {
	if r.Header.Get("Content-Type") != "application/json" {
		return problem("invalid_request", "Use application/json for this request.")
	}
	raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 16<<10))
	if err != nil {
		return problem("invalid_request", "The request body exceeds its limit.")
	}
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	token, err := decoder.Token()
	if err != nil || token != json.Delim('{') {
		return problem("invalid_request", "A JSON object is required.")
	}
	allowed := map[string]bool{}
	for _, field := range fields {
		allowed[field] = true
	}
	seen := map[string]bool{}
	for decoder.More() {
		token, err = decoder.Token()
		if err != nil {
			return problem("invalid_request", "The request is not valid JSON.")
		}
		key, ok := token.(string)
		if !ok || !allowed[key] || seen[key] {
			return problem("invalid_request", "The request contains an unknown or duplicate field.")
		}
		seen[key] = true
		token, err = decoder.Token()
		if _, ok := token.(string); err != nil || !ok {
			return problem("invalid_request", "Request fields must be strings.")
		}
	}
	if _, err = decoder.Token(); err != nil || decoder.Decode(new(any)) != io.EOF {
		return problem("invalid_request", "The request is not valid JSON.")
	}
	if json.Unmarshal(raw, target) != nil {
		return problem("invalid_request", "The request is not valid JSON.")
	}
	return nil
}

func reply(w http.ResponseWriter, value any, err error) {
	if err != nil {
		writeProblem(w, err)
	} else {
		writeValue(w, value)
	}
}
func writeValue(w http.ResponseWriter, value any) {
	writeJSON(w, 200, map[string]any{"schemaVersion": schemaVersion, "ok": true, "value": value})
}
func writeProblem(w http.ResponseWriter, err error) {
	var api *apiError
	if !errors.As(err, &api) {
		api = problem("internal", "The Project service could not complete this request.")
	}
	status := 400
	switch api.Code {
	case "forbidden":
		status = 403
	case "not_found":
		status = 404
	case "request_conflict", "stale_revision":
		status = 409
	case "runtime_unavailable":
		status = 503
	case "internal":
		status = 500
	case "unsupported":
		status = 422
	}
	writeJSON(w, status, map[string]any{"schemaVersion": schemaVersion, "ok": false, "error": api})
}
func writeJSON(w http.ResponseWriter, status int, value any) {
	data, err := json.Marshal(value)
	if err != nil || len(data) > 16<<20 {
		status = 500
		data = []byte(`{"schemaVersion":1,"ok":false,"error":{"code":"internal","message":"The response exceeds its limit.","retry":"never"}}`)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(data)
}
