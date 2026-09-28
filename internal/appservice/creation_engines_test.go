package appservice

import (
	"context"
	"strings"
	"testing"
)

func TestCreationEngineDiscoveryIsLocalBoundedAndVerified(t *testing.T) {
	t.Setenv("DOCKER_HOST", "unix:///test.sock")
	t.Setenv("DOCKER_CONTEXT", "")
	id := "sha256:" + strings.Repeat("a", 64)
	other := "sha256:" + strings.Repeat("b", 64)
	s := &Service{}
	s.runtime.command = func(_ context.Context, args, env []string) ([]byte, error) {
		switch strings.Join(args, " ") {
		case "info --format {{.ID}} {{.OSType}}":
			return []byte("daemon linux"), nil
		case "image ls --no-trunc --quiet --filter label=" + creationEngineLabel:
			return []byte(id + "\n" + id + "\ninvalid\n" + other), nil
		case "image inspect --format {{.Id}} {{.Os}}/{{.Architecture}} " + id:
			return []byte(id + " linux/amd64"), nil
		case "image inspect --format {{.Id}} {{.Os}}/{{.Architecture}} " + other:
			return []byte(other + " linux/arm64"), nil
		default:
			t.Fatalf("unexpected invocation %v", args)
			return nil, nil
		}
	}
	bindings, err := s.creationEngineBindings(context.Background())
	if err != nil || len(bindings) != 1 || bindings[0].ImageID != id || bindings[0].DaemonID != "daemon" {
		t.Fatalf("discovery: %v %v", bindings, err)
	}
	if _, err = s.connectCreationEngine(context.Background(), other); err == nil {
		t.Fatal("accepted unqualified engine")
	}
	t.Setenv("DOCKER_HOST", "tcp://remote:2375")
	if _, err = s.creationEngineBindings(context.Background()); err == nil {
		t.Fatal("accepted remote endpoint")
	}
}
