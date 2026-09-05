package main

import (
	"encoding/json"
	"go/format"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func buildPackedInner(goBin, cwd, dir, importPath string, install installIdentityResult) (string, error) {
	source, err := packInnerSource(importPath, install)
	if err != nil {
		return "", err
	}
	sourcePath := filepath.Join(dir, "main.go")
	if err := os.WriteFile(sourcePath, []byte(source), 0o600); err != nil {
		return "", err
	}
	output := filepath.Join(dir, packInnerName)
	if err := buildPacked(goBin, cwd, sourcePath, output); err != nil {
		return "", err
	}
	return output, nil
}

func packInnerSource(importPath string, install installIdentityResult) (string, error) {
	identityJSON, err := json.Marshal(install.Identity)
	if err != nil {
		return "", err
	}
	source := strings.NewReplacer(
		"@@PIPELINE_IMPORT@@", strconv.Quote(importPath),
		"@@EMBEDDED_IDENTITY@@", strconv.Quote(string(identityJSON)),
		"@@EXPECT_REPLACE@@", strconv.FormatBool(install.HasReplace),
		"@@REPLACE_PATH@@", strconv.Quote(install.ReplacePath),
		"@@ROOT_HELP@@", strconv.Quote(packedRootHelp),
		"@@COMMAND_HELP@@", packedCommandHelpSource(),
	).Replace(packedInnerTemplate)
	formatted, err := format.Source([]byte(source))
	if err != nil {
		return "", err
	}
	return string(formatted), nil
}

func packedCommandHelpSource() string {
	keys := make([]string, 0, len(packedCommandHelp))
	for key := range packedCommandHelp {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var out strings.Builder
	out.WriteString("map[string]string{")
	for _, key := range keys {
		out.WriteString(strconv.Quote(key))
		out.WriteByte(':')
		out.WriteString(strconv.Quote(packedCommandHelp[key]))
		out.WriteByte(',')
	}
	out.WriteByte('}')
	return out.String()
}

const packedInnerTemplate = `package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"runtime/debug"
	"strconv"
	"strings"
	"syscall"

	userpipe @@PIPELINE_IMPORT@@
	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/monitor/tui"
)

const (
	embeddedIdentity = @@EMBEDDED_IDENTITY@@
	expectReplace = @@EXPECT_REPLACE@@
	expectedReplacePath = @@REPLACE_PATH@@
	rootHelp = @@ROOT_HELP@@
)

var commandHelp = @@COMMAND_HELP@@
var protocol = os.NewFile(3, "gobble-protocol")

type request struct {
	command string
	view string
	workspace string
	cap int
	instance string
	sample string
	help bool
	version bool
}

type rawArgs struct {
	positionals []string
	workspace string
	workspaceSet bool
	workspaceRepeat bool
	capStr string
	capSet bool
	capRepeat bool
	instance string
	instanceSet bool
	instanceRepeat bool
	sample string
	sampleSet bool
	sampleRepeat bool
	help bool
	version bool
	unknown []string
	missingValue string
}

func main() {
	os.Exit(run())
}

func run() int {
	req, parseErr := parse(os.Args[1:])
	if parseErr != nil {
		return writeErrJSON(parseErr, 2)
	}
	identity, err := linkedIdentity()
	if err != nil {
		return writeIdentityFail(requestOp(req), err)
	}
	if req.help {
		return writeHelp(req.command)
	}
	if req.version {
		return writeVersion(identity)
	}
	switch req.command {
	case "watch":
		if err := tui.Watch(context.Background(), req.workspace, os.Stdin, os.Stderr, gobble.WithIdentity(identity)); err != nil {
			return writeLibErr(err)
		}
		return 0
	case "inspect":
		data, err := gobble.Inspect(req.workspace, gobble.View(req.view), req.instance, gobble.WithIdentity(identity))
		if err != nil {
			return writeLibErr(err)
		}
		if err := writeProtocol(data); err != nil {
			return writeFail("inspect", "stdout write failed")
		}
		return 0
	case "release":
		if err := gobble.Release(req.workspace, gobble.WithIdentity(identity)); err != nil {
			return writeLibErr(err)
		}
		return writeJSON("release", map[string]any{"op": "release"})
	case "compose", "validate", "plan", "run", "resume":
		return runGraph(req, identity)
	default:
		return writeErrJSON(invalid("cli", "unknown command"), 2)
	}
}

func runGraph(req *request, identity gobble.Identity) int {
	gobble.SetSampleSheetPath(req.sample)
	g, err := gobble.Compose(userpipe.Pipeline())
	if err != nil {
		return writeLibErr(err)
	}
	switch req.command {
	case "compose":
		return writeJSON("compose", map[string]any{"op": "compose", "pipeline": g.Name()})
	case "validate":
		if err := gobble.Validate(g); err != nil {
			return writeLibErr(err)
		}
		return writeJSON("validate", map[string]any{"op": "validate"})
	case "plan":
		plan, err := gobble.BuildPlan(g)
		if err != nil {
			return writeLibErr(err)
		}
		var data bytes.Buffer
		if err := plan.WriteJSON(&data); err != nil {
			return writeFail("plan", err.Error())
		}
		if err := writeProtocol(data.Bytes()); err != nil {
			return writeFail("plan", "stdout write failed")
		}
		return 0
	case "run", "resume":
		return occupy(req, g, identity)
	default:
		return writeErrJSON(invalid("cli", "unknown command"), 2)
	}
}

func occupy(req *request, g *gobble.Graph, identity gobble.Identity) int {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	var err error
	if req.command == "run" {
		err = gobble.Run(ctx, g, req.workspace, req.cap, gobble.WithIdentity(identity))
	} else {
		err = gobble.Resume(ctx, g, req.workspace, req.cap, gobble.WithIdentity(identity))
	}
	if err != nil {
		return writeLibErr(err)
	}
	return writeJSON(req.command, map[string]any{"op": req.command})
}

func parse(args []string) (*request, *gobble.Error) {
	return interpret(collectArgs(args))
}

func collectArgs(args []string) rawArgs {
	var raw rawArgs
	endOptions := false
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if endOptions {
			raw.positionals = append(raw.positionals, arg)
			continue
		}
		if arg == "--" {
			endOptions = true
			continue
		}
		if arg == "-h" || arg == "--help" {
			raw.help = true
			continue
		}
		if arg == "--version" {
			raw.version = true
			continue
		}
		if len(arg) >= 2 && arg[0] == '-' {
			name, value, hasValue, ok := cutFlag(arg)
			if !ok {
				raw.unknown = append(raw.unknown, arg)
				continue
			}
			if !hasValue {
				if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
					raw.missingValue = name
					return raw
				}
				i++
				value = args[i]
			}
			switch name {
			case "workspace":
				if raw.workspaceSet {
					raw.workspaceRepeat = true
				}
				raw.workspaceSet = true
				raw.workspace = value
			case "cap":
				if raw.capSet {
					raw.capRepeat = true
				}
				raw.capSet = true
				raw.capStr = value
			case "instance":
				if raw.instanceSet {
					raw.instanceRepeat = true
				}
				raw.instanceSet = true
				raw.instance = value
			case "sample":
				if raw.sampleSet {
					raw.sampleRepeat = true
				}
				raw.sampleSet = true
				raw.sample = value
			default:
				raw.unknown = append(raw.unknown, "--"+name)
			}
			continue
		}
		raw.positionals = append(raw.positionals, arg)
	}
	return raw
}

func cutFlag(arg string) (name, value string, hasValue, ok bool) {
	if !strings.HasPrefix(arg, "--") || len(arg) < 3 {
		return "", "", false, false
	}
	name, value, hasValue = strings.Cut(arg[2:], "=")
	if name == "" {
		return "", "", false, false
	}
	return name, value, hasValue, true
}

func interpret(raw rawArgs) (*request, *gobble.Error) {
	command := ""
	operands := raw.positionals
	if len(raw.positionals) > 0 {
		command = raw.positionals[0]
		operands = raw.positionals[1:]
	}
	op := "cli"
	if isOperate(command) {
		op = command
	}
	fail := func(message string) (*request, *gobble.Error) {
		return nil, invalid(op, message)
	}
	if raw.missingValue != "" {
		return fail("flag --" + raw.missingValue + " requires a value")
	}
	if raw.help && raw.version {
		return fail("help and version cannot be combined")
	}
	if command != "" && !isOperate(command) && command != "help" && command != "version" {
		return nil, invalid("cli", "unknown command")
	}
	allowWorkspace, allowCap, allowInstance, allowSample := flagsFor(command)
	if command == "" || command == "help" || command == "version" {
		allowWorkspace, allowCap, allowInstance, allowSample = false, false, false, false
	}
	unknown := append([]string(nil), raw.unknown...)
	if raw.workspaceSet && !allowWorkspace {
		unknown = append(unknown, "--workspace")
	}
	if raw.capSet && !allowCap {
		unknown = append(unknown, "--cap")
	}
	if raw.instanceSet && !allowInstance {
		unknown = append(unknown, "--instance")
	}
	if raw.sampleSet && !allowSample {
		unknown = append(unknown, "--sample")
	}
	if command == "help" {
		if raw.version {
			return fail("help and version cannot be combined")
		}
		if len(unknown) > 0 {
			return fail("unknown flag " + unknown[0])
		}
		if len(operands) > 1 {
			return fail("extra operand")
		}
		topic := ""
		if len(operands) == 1 {
			topic = operands[0]
			if !isOperate(topic) && topic != "help" && topic != "version" {
				return nil, invalid("cli", "unknown command")
			}
		}
		return &request{command: topic, help: true}, nil
	}
	if raw.help {
		if len(unknown) > 0 {
			return fail("unknown flag " + unknown[0])
		}
		if raw.workspaceRepeat || raw.capRepeat || raw.instanceRepeat || raw.sampleRepeat {
			return nil, repeatedFlagError(op, raw)
		}
		return &request{command: command, help: true}, nil
	}
	if raw.version && isOperate(command) {
		return fail("version cannot be combined with an operate command")
	}
	if command == "version" || raw.version {
		if len(unknown) > 0 {
			return fail("unknown flag " + unknown[0])
		}
		if len(operands) > 0 {
			return fail("extra operand")
		}
		return &request{command: "version", version: true}, nil
	}
	if command == "" {
		if len(unknown) > 0 {
			return fail("unknown flag " + unknown[0])
		}
		return &request{help: true}, nil
	}
	if len(unknown) > 0 {
		return fail("unknown flag " + unknown[0])
	}
	if raw.workspaceRepeat || raw.capRepeat || raw.instanceRepeat || raw.sampleRepeat {
		return nil, repeatedFlagError(op, raw)
	}
	req := &request{command: command}
	switch command {
	case "compose", "validate", "plan":
		if len(operands) > 0 {
			return fail("extra operand")
		}
	case "run", "resume":
		if len(operands) > 0 {
			return fail("extra operand")
		}
		if !raw.workspaceSet || raw.workspace == "" {
			return fail("missing --workspace")
		}
		req.workspace = raw.workspace
		if raw.capSet {
			value, err := strconv.Atoi(raw.capStr)
			if err != nil {
				return fail("non-integer --cap")
			}
			req.cap = value
		}
	case "inspect":
		if len(operands) == 0 {
			return fail("missing view")
		}
		if len(operands) > 1 {
			return fail("extra operand")
		}
		req.view = operands[0]
		if !raw.workspaceSet || raw.workspace == "" {
			return fail("missing --workspace")
		}
		req.workspace = raw.workspace
		if raw.instanceSet {
			req.instance = raw.instance
		}
	case "release", "watch":
		if len(operands) > 0 {
			return fail("extra operand")
		}
		if !raw.workspaceSet || raw.workspace == "" {
			return fail("missing --workspace")
		}
		req.workspace = raw.workspace
	default:
		return nil, invalid("cli", "unknown command")
	}
	if command == "compose" || command == "validate" || command == "plan" || command == "run" || command == "resume" {
		if raw.sampleSet {
			if strings.TrimSpace(raw.sample) == "" {
				return fail("empty --sample")
			}
			req.sample = raw.sample
		} else {
			req.sample = gobble.DefaultSampleSheetPath
		}
	}
	return req, nil
}

func repeatedFlagError(op string, raw rawArgs) *gobble.Error {
	switch {
	case raw.workspaceRepeat:
		return invalid(op, "repeated flag --workspace")
	case raw.capRepeat:
		return invalid(op, "repeated flag --cap")
	case raw.instanceRepeat:
		return invalid(op, "repeated flag --instance")
	default:
		return invalid(op, "repeated flag --sample")
	}
}

func isOperate(command string) bool {
	switch command {
	case "compose", "validate", "plan", "run", "inspect", "resume", "release", "watch":
		return true
	default:
		return false
	}
}

func flagsFor(command string) (workspace, cap, instance, sample bool) {
	switch command {
	case "compose", "validate", "plan":
		return false, false, false, true
	case "run", "resume":
		return true, true, false, true
	case "inspect":
		return true, false, true, false
	case "release", "watch":
		return true, false, false, false
	default:
		return false, false, false, false
	}
}

func requestOp(req *request) string {
	if req != nil && req.command != "" {
		return req.command
	}
	return "cli"
}

func linkedIdentity() (gobble.Identity, error) {
	var identity gobble.Identity
	if err := json.Unmarshal([]byte(embeddedIdentity), &identity); err != nil {
		return identity, err
	}
	info, ok := debug.ReadBuildInfo()
	if !ok || info == nil {
		return identity, errors.New("linked Gobble build info unavailable")
	}
	for _, dep := range info.Deps {
		if dep == nil || dep.Path != identity.GobbleModule {
			continue
		}
		if dep.Version != identity.GobbleVersion {
			return identity, fmt.Errorf("required Gobble %s@%s; have %s@%s", identity.GobbleModule, identity.GobbleVersion, dep.Path, dep.Version)
		}
		if expectReplace && (dep.Replace == nil || dep.Replace.Path != expectedReplacePath) {
			have := ""
			if dep.Replace != nil {
				have = dep.Replace.Path
			}
			return identity, fmt.Errorf("required Gobble replace %s; have %s", expectedReplacePath, have)
		}
		return identity, nil
	}
	return identity, fmt.Errorf("required Gobble %s@%s; linked dependency missing", identity.GobbleModule, identity.GobbleVersion)
}

func writeVersion(identity gobble.Identity) int {
	return writeJSON("version", map[string]any{
		"op": "version",
		"module": identity.GobbleModule,
		"version": identity.GobbleVersion,
		"vcs_revision": identity.GobbleVCSRevision,
		"vcs_modified": identity.GobbleVCSModified,
		"process": "packed-runner",
		"install_kind": identity.InstallKind,
		"identity_mode": identity.IdentityMode,
		"executable_sha256": "",
		"goos": identity.GOOS,
		"goarch": identity.GOARCH,
		"pipeline_module": identity.PipelineModule,
		"pipeline_import": identity.PipelineImport,
		"pipeline_version": identity.PipelineVersion,
		"pipeline_vcs_revision": identity.PipelineVCSRevision,
		"pipeline_vcs_modified": identity.PipelineVCSModified,
	})
}

func writeHelp(command string) int {
	text := rootHelp
	if command != "" {
		if commandText, ok := commandHelp[command]; ok {
			text = commandText
		}
	}
	if err := writeProtocol([]byte(text)); err != nil {
		return writeFail("help", "stdout write failed")
	}
	return 0
}

func invalid(op, message string) *gobble.Error {
	return &gobble.Error{Op: op, Defects: []gobble.Defect{{
		Code: gobble.DefectInvalidRequest,
		Message: message,
	}}}
}

func writeIdentityFail(op string, err error) int {
	return writeErrJSON(&gobble.Error{Op: op, Defects: []gobble.Defect{{
		Code: gobble.DefectIdentityMismatch,
		Unit: "gobble",
		Message: err.Error(),
	}}}, 2)
}

func writeJSON(op string, value any) int {
	data, err := json.Marshal(value)
	if err != nil {
		return writeFail(op, err.Error())
	}
	if err := writeProtocol(append(data, '\n')); err != nil {
		return writeFail(op, "stdout write failed")
	}
	return 0
}

func writeProtocol(data []byte) error {
	if protocol == nil {
		return errors.New("protocol unavailable")
	}
	_, err := protocol.Write(data)
	return err
}

func writeLibErr(err error) int {
	var ge *gobble.Error
	if !errors.As(err, &ge) {
		return writeFail("cli", err.Error())
	}
	code := 1
	if gobble.IsSampleSheetError(ge) {
		code = 2
	}
	return writeErrJSON(ge, code)
}

func writeFail(op, message string) int {
	return writeErrJSON(invalid(op, message), 1)
}

func writeErrJSON(ge *gobble.Error, code int) int {
	data, err := json.Marshal(ge)
	if err != nil {
		return 1
	}
	_, _ = os.Stderr.Write(append(data, '\n'))
	return code
}
`
