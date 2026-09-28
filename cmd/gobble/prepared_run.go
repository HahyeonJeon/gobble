package main

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/HahyeonJeon/gobble"
	"io"
	"os"
	"time"
)

// The mount is fixed by the trusted native host. No Project source or driver.
func readLaunchFile(name string, limit int64, out any) error {
	f, err := os.Open("/gobble/launch/" + name)
	if err != nil {
		return err
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil {
		return err
	}
	if int64(len(raw)) > limit {
		return io.ErrShortBuffer
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if err = d.Decode(out); err != nil {
		return err
	}
	if d.Decode(new(any)) != io.EOF {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func runPreparedControl(req *request, stdout, stderr io.Writer) int {
	if req.command == "prepared-status" {
		value, err := gobble.ReadPreparedRun(req.workspace)
		if err != nil {
			return writeErr(stderr, invalidRequest(req.command, err.Error()), 1)
		}
		return writeJSON(stdout, stderr, req.command, value)
	}
	if req.command == "prepared-stop" {
		var in struct {
			ExpectedLease string `json:"expectedLease"`
		}
		if err := readLaunchFile("stop.json", 4096, &in); err != nil {
			return writeErr(stderr, invalidRequest(req.command, "invalid Stop intent"), 2)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
		defer cancel()
		value, err := gobble.StopPreparedRun(ctx, req.workspace, in.ExpectedLease)
		if err != nil && value.Status != "recovery-required" {
			return writeLibraryErr(stderr, err)
		}
		return writeJSON(stdout, stderr, req.command, value)
	}
	var intent gobble.LaunchIntent
	var payload json.RawMessage
	if err := readLaunchFile("intent.json", 16384, &intent); err != nil {
		return writeErr(stderr, invalidRequest(req.command, "invalid launch intent"), 2)
	}
	if err := readLaunchFile("payload.json", 3<<20, &payload); err != nil {
		return writeErr(stderr, invalidRequest(req.command, "invalid prepared payload"), 2)
	}
	if err := gobble.RunPreparedPipeline(context.Background(), req.workspace, payload, intent); err != nil {
		return writeLibraryErr(stderr, err)
	}
	state, err := gobble.ReadPreparedRun(req.workspace)
	if err != nil {
		return writeErr(stderr, invalidRequest(req.command, err.Error()), 1)
	}
	return writeJSON(stdout, stderr, req.command, state)
}
