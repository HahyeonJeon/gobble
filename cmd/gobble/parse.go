package main

import (
	"strconv"
	"strings"

	"github.com/HahyeonJeon/gobble"
)

type request struct {
	command   string
	pkg       string
	view      string
	workspace string
	cap       int
	instance  string
	sample    string
	output    string
	help      bool
	version   bool
}

type rawArgs struct {
	positionals     []string
	workspace       string
	workspaceSet    bool
	workspaceRepeat bool
	capStr          string
	capSet          bool
	capRepeat       bool
	instance        string
	instanceSet     bool
	instanceRepeat  bool
	sample          string
	sampleSet       bool
	sampleRepeat    bool
	output          string
	outputSet       bool
	outputRepeat    bool
	help            bool
	version         bool
	unknown         []string
	missingValue    string
}

func parse(args []string) (*request, *gobble.Error) {
	return interpret(collectArgs(args))
}

func collectArgs(args []string) rawArgs {
	var raw rawArgs
	endOpts := false
	for i := 0; i < len(args); i++ {
		a := args[i]
		if endOpts {
			raw.positionals = append(raw.positionals, a)
			continue
		}
		if a == "--" {
			endOpts = true
			continue
		}
		if a == "-h" || a == "--help" {
			raw.help = true
			continue
		}
		if a == "--version" {
			raw.version = true
			continue
		}
		if len(a) >= 2 && a[0] == '-' {
			name, val, hasVal, ok := cutFlag(a)
			if !ok {
				raw.unknown = append(raw.unknown, a)
				continue
			}
			if !hasVal {
				if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
					raw.missingValue = name
					return raw
				}
				i++
				val = args[i]
			}
			switch name {
			case "workspace":
				if raw.workspaceSet {
					raw.workspaceRepeat = true
				}
				raw.workspaceSet = true
				raw.workspace = val
			case "cap":
				if raw.capSet {
					raw.capRepeat = true
				}
				raw.capSet = true
				raw.capStr = val
			case "instance":
				if raw.instanceSet {
					raw.instanceRepeat = true
				}
				raw.instanceSet = true
				raw.instance = val
			case "sample":
				if raw.sampleSet {
					raw.sampleRepeat = true
				}
				raw.sampleSet = true
				raw.sample = val
			case "output":
				if raw.outputSet {
					raw.outputRepeat = true
				}
				raw.outputSet = true
				raw.output = val
			default:
				raw.unknown = append(raw.unknown, "--"+name)
			}
			continue
		}
		raw.positionals = append(raw.positionals, a)
	}
	return raw
}

func cutFlag(a string) (name, val string, hasVal, ok bool) {
	if !strings.HasPrefix(a, "--") || len(a) < 3 {
		return "", "", false, false
	}
	name, val, hasVal = strings.Cut(a[2:], "=")
	if name == "" {
		return "", "", false, false
	}
	return name, val, hasVal, true
}

func interpret(raw rawArgs) (*request, *gobble.Error) {
	cmd := ""
	operands := raw.positionals
	if len(raw.positionals) > 0 {
		cmd = raw.positionals[0]
		operands = raw.positionals[1:]
	}
	op := "cli"
	if isOperate(cmd) {
		op = cmd
	}
	inv := func(message string) *gobble.Error {
		return invalidRequest(op, message)
	}

	if raw.missingValue != "" {
		return nil, inv("flag --" + raw.missingValue + " requires a value")
	}
	if raw.help && raw.version {
		return nil, inv("help and version cannot be combined")
	}
	if cmd != "" && !isOperate(cmd) && cmd != "help" && cmd != "version" {
		return nil, invalidRequest("cli", "unknown command")
	}

	allowWS, allowCap, allowInst, allowSample, allowOutput := flagsFor(cmd)
	if cmd == "help" || cmd == "version" || cmd == "" {
		allowWS, allowCap, allowInst, allowSample, allowOutput = false, false, false, false, false
	}
	unknown := append([]string(nil), raw.unknown...)
	if raw.workspaceSet && !allowWS {
		unknown = append(unknown, "--workspace")
	}
	if raw.capSet && !allowCap {
		unknown = append(unknown, "--cap")
	}
	if raw.instanceSet && !allowInst {
		unknown = append(unknown, "--instance")
	}
	if raw.sampleSet && !allowSample {
		unknown = append(unknown, "--sample")
	}
	if raw.outputSet && !allowOutput {
		unknown = append(unknown, "--output")
	}

	if cmd == "help" {
		if raw.version {
			return nil, inv("help and version cannot be combined")
		}
		if len(unknown) > 0 {
			return nil, inv("unknown flag " + unknown[0])
		}
		if len(operands) > 1 {
			return nil, inv("extra operand")
		}
		topic := ""
		if len(operands) == 1 {
			topic = operands[0]
			if !isOperate(topic) && topic != "help" && topic != "version" {
				return nil, invalidRequest("cli", "unknown command")
			}
		}
		return &request{command: topic, help: true}, nil
	}

	if raw.help {
		if len(unknown) > 0 {
			return nil, inv("unknown flag " + unknown[0])
		}
		if raw.workspaceRepeat || raw.capRepeat || raw.instanceRepeat || raw.sampleRepeat || raw.outputRepeat {
			return nil, repeatedFlagError(op, raw)
		}
		return &request{command: cmd, help: true}, nil
	}

	if raw.version && isOperate(cmd) {
		return nil, inv("version cannot be combined with an operate command")
	}

	if cmd == "version" || raw.version {
		if len(unknown) > 0 {
			return nil, inv("unknown flag " + unknown[0])
		}
		if len(operands) > 0 {
			return nil, inv("extra operand")
		}
		return &request{command: "version", version: true}, nil
	}

	if cmd == "" {
		if len(unknown) > 0 {
			return nil, inv("unknown flag " + unknown[0])
		}
		return &request{help: true}, nil
	}

	if len(unknown) > 0 {
		return nil, inv("unknown flag " + unknown[0])
	}
	if raw.workspaceRepeat || raw.capRepeat || raw.instanceRepeat || raw.sampleRepeat || raw.outputRepeat {
		return nil, repeatedFlagError(op, raw)
	}

	req := &request{command: cmd}
	switch cmd {
	case "init":
		if len(operands) != 1 {
			return nil, inv("init requires one new project directory")
		}
		req.pkg = operands[0]
	case "doctor":
		if len(operands) != 0 {
			return nil, inv("extra operand")
		}
	case "compose", "validate", "plan":
		if len(operands) > 1 {
			return nil, inv("extra operand")
		}
		req.pkg = "."
		if len(operands) == 1 {
			req.pkg = operands[0]
		}
	case "pack":
		if len(operands) > 1 {
			return nil, inv("extra operand")
		}
		req.pkg = "."
		if len(operands) == 1 {
			req.pkg = operands[0]
		}
		if !raw.outputSet || raw.output == "" {
			return nil, inv("missing --output")
		}
		req.output = raw.output
	case "run", "resume":
		if len(operands) > 1 {
			return nil, inv("extra operand")
		}
		req.pkg = "."
		if len(operands) == 1 {
			req.pkg = operands[0]
		}
		if !raw.workspaceSet || raw.workspace == "" {
			return nil, inv("missing --workspace")
		}
		req.workspace = raw.workspace
		if raw.capSet {
			n, err := strconv.Atoi(raw.capStr)
			if err != nil {
				return nil, inv("non-integer --cap")
			}
			req.cap = n
		}
	case "inspect":
		if len(operands) == 0 {
			return nil, inv("missing view")
		}
		if len(operands) > 1 {
			return nil, inv("extra operand")
		}
		req.view = operands[0]
		if !raw.workspaceSet || raw.workspace == "" {
			return nil, inv("missing --workspace")
		}
		req.workspace = raw.workspace
		if raw.instanceSet {
			req.instance = raw.instance
		}
	case "release", "stop", "watch":
		if len(operands) > 0 {
			return nil, inv("extra operand")
		}
		if !raw.workspaceSet || raw.workspace == "" {
			return nil, inv("missing --workspace")
		}
		req.workspace = raw.workspace
	default:
		return nil, invalidRequest("cli", "unknown command")
	}
	if cmd == "compose" || cmd == "validate" || cmd == "plan" || cmd == "run" || cmd == "resume" {
		if raw.sampleSet {
			if strings.TrimSpace(raw.sample) == "" {
				return nil, inv("empty --sample")
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
		return invalidRequest(op, "repeated flag --workspace")
	case raw.capRepeat:
		return invalidRequest(op, "repeated flag --cap")
	case raw.instanceRepeat:
		return invalidRequest(op, "repeated flag --instance")
	case raw.sampleRepeat:
		return invalidRequest(op, "repeated flag --sample")
	default:
		return invalidRequest(op, "repeated flag --output")
	}
}

func isOperate(cmd string) bool {
	switch cmd {
	case "compose", "validate", "plan", "run", "inspect", "resume", "release", "stop", "pack", "watch", "init", "doctor":
		return true
	default:
		return false
	}
}

func flagsFor(cmd string) (workspace, cap, instance, sample, output bool) {
	switch cmd {
	case "compose", "validate", "plan":
		return false, false, false, true, false
	case "run", "resume":
		return true, true, false, true, false
	case "inspect":
		return true, false, true, false, false
	case "release", "stop", "watch":
		return true, false, false, false, false
	case "pack":
		return false, false, false, false, true
	default:
		return false, false, false, false, false
	}
}
