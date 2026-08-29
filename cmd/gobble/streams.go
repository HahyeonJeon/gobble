package main

import (
	"encoding/json"
	"errors"
	"io"
	"runtime"
	"runtime/debug"

	"github.com/HahyeonJeon/gobble"
)

const modulePath = "github.com/HahyeonJeon/gobble"

type versionResult struct {
	Op               string `json:"op"`
	Module           string `json:"module"`
	Version          string `json:"version"`
	VCSRevision      string `json:"vcs_revision"`
	VCSModified      bool   `json:"vcs_modified"`
	Process          string `json:"process"`
	InstallKind      string `json:"install_kind"`
	IdentityMode     string `json:"identity_mode"`
	ExecutableSHA256 string `json:"executable_sha256"`
	GOOS             string `json:"goos"`
	GOARCH           string `json:"goarch"`
}

func invalidRequest(op, message string) *gobble.Error {
	return &gobble.Error{
		Op: op,
		Defects: []gobble.Defect{{
			Code:    gobble.DefectInvalidRequest,
			Message: message,
		}},
	}
}

func writeErr(stderr io.Writer, ge *gobble.Error, code int) int {
	if ge == nil {
		return code
	}
	data, err := json.Marshal(ge)
	if err != nil {
		return code
	}
	_, _ = stderr.Write(append(data, '\n'))
	return code
}

func writeLibraryErr(stderr io.Writer, err error) int {
	var ge *gobble.Error
	if !errors.As(err, &ge) {
		return writeErr(stderr, invalidRequest("cli", err.Error()), 1)
	}
	return writeErr(stderr, ge, 1)
}

func writeJSON(stdout, stderr io.Writer, op string, v any) int {
	data, err := json.Marshal(v)
	if err != nil {
		return writeErr(stderr, invalidRequest(op, "invalid request"), 1)
	}
	if _, err := stdout.Write(append(data, '\n')); err != nil {
		return writeErr(stderr, invalidRequest(op, "stdout write failed"), 1)
	}
	return 0
}

func writeVersion(stdout, stderr io.Writer) int {
	info, ok := debug.ReadBuildInfo()
	version := "(devel)"
	mode := "local-pin"
	var settings []debug.BuildSetting
	if ok && info != nil {
		settings = info.Settings
		if info.Main.Version != "" {
			version = info.Main.Version
		}
		if info.Main.Replace == nil && version != "v0.0.0" && installExactTagPattern.MatchString(version) {
			mode = "exact-tag"
		}
	}
	digest := ""
	if mode == "local-pin" {
		var err error
		digest, err = installExecutableSHA256()
		if err != nil {
			return writeErr(stderr, invalidRequest("version", err.Error()), 1)
		}
	}
	return writeJSON(stdout, stderr, "version", versionResult{
		Op:               "version",
		Module:           modulePath,
		Version:          version,
		VCSRevision:      vcsSetting(settings, "vcs.revision"),
		VCSModified:      vcsSetting(settings, "vcs.modified") == "true",
		Process:          "generic-cli",
		InstallKind:      "module",
		IdentityMode:     mode,
		ExecutableSHA256: digest,
		GOOS:             runtime.GOOS,
		GOARCH:           runtime.GOARCH,
	})
}

func buildVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok || info.Main.Version == "" {
		return "(devel)"
	}
	return info.Main.Version
}
