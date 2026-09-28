// Command gobble-service is the app-owned native Project query service.
package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"syscall"
	"time"

	"github.com/HahyeonJeon/gobble/internal/appservice"
)

type startup struct {
	SchemaVersion int    `json:"schemaVersion"`
	Token         string `json:"token"`
	ProfilePath   string `json:"profilePath"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "Gobble Project service:", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	reader := bufio.NewReaderSize(os.Stdin, 16<<10)
	ready := make(chan []byte, 1)
	go func() {
		line, err := reader.ReadSlice('\n')
		if err != nil {
			ready <- nil
			return
		}
		ready <- append([]byte{}, line...)
	}()
	var line []byte
	select {
	case line = <-ready:
	case <-time.After(5 * time.Second):
		return errors.New("startup handshake timed out")
	case <-ctx.Done():
		return ctx.Err()
	}
	var input startup
	if len(line) > 16<<10 || json.Unmarshal(line, &input) != nil || input.SchemaVersion != 1 || !regexp.MustCompile(`^[a-f0-9]{64}$`).MatchString(input.Token) || !filepath.IsAbs(input.ProfilePath) {
		return errors.New("invalid startup handshake")
	}
	service, err := appservice.Open(ctx, input.ProfilePath)
	if err != nil {
		return err
	}
	defer service.Close()
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return err
	}
	defer listener.Close()
	server := &http.Server{Handler: service.Handler(listener.Addr().String(), input.Token), ReadHeaderTimeout: 3 * time.Second, ReadTimeout: 5 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 10 * time.Second, MaxHeaderBytes: 16 << 10, DisableGeneralOptionsHandler: true, BaseContext: func(net.Listener) context.Context { return ctx }}
	input.Token = ""
	if err = json.NewEncoder(os.Stdout).Encode(map[string]any{"schemaVersion": 1, "instanceId": "svc_" + rand.Text(), "port": listener.Addr().(*net.TCPAddr).Port}); err != nil {
		return err
	}
	go func() { _, _ = io.Copy(io.Discard, reader); cancel() }()
	served := make(chan error, 1)
	go func() { served <- server.Serve(listener) }()
	select {
	case err = <-served:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case <-ctx.Done():
	}
	shutdown, stop := context.WithTimeout(context.Background(), 8*time.Second)
	defer stop()
	if err = server.Shutdown(shutdown); err != nil {
		_ = server.Close()
	}
	return nil
}
