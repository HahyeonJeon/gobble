package moduleevidence

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

const (
	validDigest               = "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	validImage  modules.Image = "registry.example/tool/command:1.2.3@" + validDigest
)

type contractOptions struct {
	modules.Options
	Label string
}

type contractPorts struct {
	Output gobble.Handle
}

func TestOneCommandModuleContractComposes(t *testing.T) {
	p, ports := contractModulePipeline(nil)
	if ports.Output.IsZero() {
		t.Fatal("contract module output is zero, want declared output Handle")
	}
	graph, err := gobble.Compose(p)
	if err != nil {
		t.Fatalf("contract module Compose() error = %v, want nil", err)
	}
	if got := graph.TaskIDs(); !reflect.DeepEqual(got, []string{"copy"}) {
		t.Fatalf("contract module TaskIDs() = %#v, want [copy]", got)
	}
}

func TestOneCommandModuleContractRecordsOptionFailure(t *testing.T) {
	p, _ := contractModulePipeline([]string{"--label=override"})
	_, err := gobble.Compose(p)
	var composeErr *gobble.Error
	if !errors.As(err, &composeErr) {
		t.Fatalf("contract module Compose() error = %T %v, want *gobble.Error", err, err)
	}
	if composeErr.Op != "compose" || len(composeErr.Defects) != 1 {
		t.Fatalf("contract module Compose() error = %#v, want one compose defect", composeErr)
	}
	if defect := composeErr.Defects[0]; defect.Code != gobble.DefectInvalidValue || defect.Unit != "copy" {
		t.Fatalf("contract module Compose() defect = %#v, want invalid-value for copy", defect)
	}
}

func TestOptionsCloneOwnsExtraArgs(t *testing.T) {
	original := modules.Options{
		Image:     validImage,
		Resources: gobble.Resources{CPU: 2, Memory: "4g"},
		ExtraArgs: []string{"--advanced", "value"},
	}
	cloned := original.Clone()
	cloned.ExtraArgs[0] = "--changed"
	if original.ExtraArgs[0] != "--advanced" {
		t.Fatalf("Options.Clone() aliased ExtraArgs: got %q", original.ExtraArgs[0])
	}
}

func TestImageTaskReference(t *testing.T) {
	tests := []struct {
		name  string
		image modules.Image
	}{
		{name: "domain registry", image: validImage},
		{name: "localhost port and name separators", image: "localhost:5000/tool_name/command--name:_1.2-3@" + validDigest},
		{name: "IPv6 registry", image: "[2001:db8::1]/tool/command:1@" + validDigest},
		{name: "maximum tag length", image: modules.Image("registry.example/tool/command:" + strings.Repeat("a", 128) + "@" + validDigest)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.image.TaskReference("example")
			if err != nil {
				t.Fatalf("Image.TaskReference() error = %v, want nil", err)
			}
			if got != string(tt.image) {
				t.Fatalf("Image.TaskReference() = %q, want %q", got, tt.image)
			}
		})
	}
}

func TestImageTaskReferenceRejectsInvalidPins(t *testing.T) {
	tests := []struct {
		name  string
		image modules.Image
	}{
		{name: "empty", image: ""},
		{name: "implicit registry", image: "tool/command:1.2.3@" + validDigest},
		{name: "malformed registry", image: "registry..example/tool/command:1.2.3@" + validDigest},
		{name: "malformed registry port", image: "registry.example:port/tool/command:1.2.3@" + validDigest},
		{name: "empty repository", image: "registry.example/:1@" + validDigest},
		{name: "empty repository component", image: "registry.example/tool//command:1.2.3@" + validDigest},
		{name: "malformed repository", image: "registry.example/tool..command:1.2.3@" + validDigest},
		{name: "uppercase repository", image: "registry.example/Tool/command:1.2.3@" + validDigest},
		{name: "repository name too long", image: modules.Image("registry.example/" + strings.Repeat("a", 256) + ":1.2.3@" + validDigest)},
		{name: "missing tag", image: "registry.example/tool/command@" + validDigest},
		{name: "empty tag", image: "registry.example/tool/command:@" + validDigest},
		{name: "malformed tag", image: "registry.example/tool/command:-1.2.3@" + validDigest},
		{name: "tag too long", image: modules.Image("registry.example/tool/command:" + strings.Repeat("a", 129) + "@" + validDigest)},
		{name: "latest", image: "registry.example/tool/command:latest@" + validDigest},
		{name: "missing digest", image: "registry.example/tool/command:1.2.3"},
		{name: "short digest", image: "registry.example/tool/command:1.2.3@sha256:0123"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.image.TaskReference("example")
			if got != "" {
				t.Fatalf("Image.TaskReference() = %q, want empty reference", got)
			}
			var composeErr *gobble.Error
			if !errors.As(err, &composeErr) {
				t.Fatalf("Image.TaskReference() error = %T %v, want *gobble.Error", err, err)
			}
			if composeErr.Op != "compose" || len(composeErr.Defects) != 1 {
				t.Fatalf("Image.TaskReference() error = %#v, want one compose defect", composeErr)
			}
			defect := composeErr.Defects[0]
			if defect.Code != gobble.DefectInvalidValue || defect.Unit != "example" || defect.Message == "" {
				t.Fatalf("Image.TaskReference() defect = %#v, want invalid-value for example", defect)
			}
		})
	}
}

func TestAppendExtraArgsCopiesArgv(t *testing.T) {
	command := []string{"tool", "--threads", "2", "input"}
	extra := []string{"--advanced", "value"}
	got, err := modules.AppendExtraArgs("example", command, extra, []string{"--threads"})
	if err != nil {
		t.Fatalf("AppendExtraArgs() error = %v, want nil", err)
	}
	want := []string{"tool", "--threads", "2", "input", "--advanced", "value"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("AppendExtraArgs() = %#v, want %#v", got, want)
	}
	got[0] = "changed"
	got[len(command)] = "changed"
	if command[0] != "tool" || extra[0] != "--advanced" {
		t.Fatalf("AppendExtraArgs() aliased input argv: command=%#v extra=%#v", command, extra)
	}
}

func TestAppendExtraArgsRejectsNamedOptionCollision(t *testing.T) {
	for _, test := range []struct {
		extra []string
		flag  string
	}{
		{extra: []string{"--threads", "8"}, flag: "--threads"},
		{extra: []string{"--threads=8"}, flag: "--threads"},
		{extra: []string{"-t8"}, flag: "-t"},
		{extra: []string{"-t=8"}, flag: "-t"},
	} {
		_, err := modules.AppendExtraArgs("example", []string{"tool", "--threads", "2"}, test.extra, []string{test.flag})
		var composeErr *gobble.Error
		if !errors.As(err, &composeErr) {
			t.Fatalf("AppendExtraArgs(%#v) error = %T %v, want *gobble.Error", test.extra, err, err)
		}
		if len(composeErr.Defects) != 1 || composeErr.Defects[0].Code != gobble.DefectInvalidValue {
			t.Fatalf("AppendExtraArgs(%#v) error = %#v, want one invalid-value defect", test.extra, composeErr)
		}
	}
}

func TestRejectExtraArgsRecognizesLongAndAttachedShortOptions(t *testing.T) {
	for _, extra := range [][]string{{"--route=other"}, {"-oother"}, {"-o=other"}} {
		err := modules.RejectExtraArgs("example", extra, []string{"--route", "-o"})
		var composeErr *gobble.Error
		if !errors.As(err, &composeErr) || len(composeErr.Defects) != 1 || composeErr.Defects[0].Code != gobble.DefectInvalidValue {
			t.Fatalf("RejectExtraArgs(%#v) error = %#v, want one invalid-value defect", extra, err)
		}
	}
}

func TestRejectExtraArgPrefixesRecognizesEveryLongPrefix(t *testing.T) {
	const protected = "--protected-option"
	for length := 3; length <= len(protected); length++ {
		prefix := protected[:length]
		for _, arg := range []string{prefix, strings.ToUpper(prefix) + "=value"} {
			if got := modules.MatchProtectedExtraArg([]string{arg}, []string{protected}); got != protected {
				t.Fatalf("MatchProtectedExtraArg(%q) = %q, want %q", arg, got, protected)
			}
			err := modules.RejectExtraArgPrefixes("example", []string{arg}, []string{protected})
			var composeErr *gobble.Error
			if !errors.As(err, &composeErr) || len(composeErr.Defects) != 1 || composeErr.Defects[0].Code != gobble.DefectInvalidValue {
				t.Fatalf("RejectExtraArgPrefixes(%q) error = %#v, want one invalid-value defect", arg, err)
			}
		}
	}
	if err := modules.RejectExtraArgPrefixes("example", []string{"--protected-option-extra"}, []string{protected}); err != nil {
		t.Fatalf("RejectExtraArgPrefixes() rejected longer distinct option: %v", err)
	}
}

func TestShellRedirectQuotesEveryArgvToken(t *testing.T) {
	got := modules.ShellRedirect([]string{"tool", "$(touch pwned)", "a'b"}, "out/x y")
	want := `'tool' '$(touch pwned)' 'a'"'"'b' > 'out/x y'`
	if got != want {
		t.Fatalf("ShellRedirect() = %q, want %q", got, want)
	}
}

func contractModulePipeline(extraArgs []string) (*gobble.Pipeline, contractPorts) {
	p := gobble.NewPipeline("contract-module")
	input := p.AddInput("input", gobble.PathSpec{Dir: gobble.Dir("inputs"), Base: "sample", Ext: ".txt"})
	ports, err := addContractModule(p, input, contractOptions{
		Options: modules.Options{
			Image:     validImage,
			Resources: gobble.Resources{CPU: 1, Memory: "1g"},
			ExtraArgs: extraArgs,
		},
		Label: "sample",
	})
	p.RecordComposeError(err)
	return p, ports
}

func addContractModule(parent modules.Parent, input gobble.Handle, options contractOptions) (contractPorts, error) {
	common := options.Options.Clone()
	image, err := common.Image.TaskReference("copy")
	if err != nil {
		return contractPorts{}, err
	}
	inputPath, err := input.Spec().Render()
	if err != nil {
		return contractPorts{}, err
	}
	output := input.Spec().WithDir(gobble.Dir("outputs"))
	outputPath, err := output.Render()
	if err != nil {
		return contractPorts{}, err
	}
	command, err := modules.AppendExtraArgs(
		"copy",
		[]string{"copy", "--label", options.Label, inputPath, outputPath},
		common.ExtraArgs,
		[]string{"--label"},
	)
	if err != nil {
		return contractPorts{}, err
	}
	task := parent.AddTask(gobble.TaskSpec{
		Name:      "copy",
		Command:   command,
		Image:     image,
		Resources: common.Resources,
		Inputs:    []gobble.Bind{{Name: "input", From: input}},
		Outputs:   []gobble.Bind{{Name: "output", Spec: output}},
	})
	return contractPorts{Output: task.Out("output")}, nil
}
