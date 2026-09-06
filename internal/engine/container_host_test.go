package engine

import (
	"os"
	"testing"

	"github.com/HahyeonJeon/gobble/internal/containerenv"
)

func TestContainerCommandsShareDaemonHostIdentity(t *testing.T) {
	t.Setenv("GOBBLE_DAEMON_ID", "engine-one")
	t.Setenv("GOBBLE_CONTROLLER", "first-controller")
	first, _ := currentHost()
	t.Setenv("GOBBLE_CONTROLLER", "second-controller")
	second, _ := currentHost()
	if first != second || first != containerenv.Host("engine-one") {
		t.Fatal("container replacement changed host identity")
	}
	t.Setenv("GOBBLE_DAEMON_ID", "engine-two")
	other, _ := currentHost()
	if first == other {
		t.Fatal("different daemons share an identity")
	}
	t.Setenv("GOBBLE_CONTROLLER", "")
	native, _ := currentHost()
	want, _ := os.Hostname()
	if native != want {
		t.Fatal("native host identity changed")
	}
}
