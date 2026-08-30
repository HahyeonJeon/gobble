// Package fastp owns the graph-stable fastp command module.
package fastp

import (
	"strconv"

	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/assets/modules"
)

// fastpImage is the biocontainers tag for the nf-core fastp 1.3.6 pin.
const fastpImage = "quay.io/biocontainers/fastp:1.3.6--h43da1c4_0"

const fastpTaskName = "fastp"

// FastpOptions are typed fastp settings. ExtraArgs are argv tokens
// appended after named flags.
//
// --thread copies Resources.CPU when CPU is at least 1, as an integer.
// This asset is paired-end and always passes --detect_adapter_for_pe.
type FastpOptions struct {
	ExtraArgs []string
	Resources gobble.Resources
	OutDir    gobble.Directory
}

// FastpPorts are the cleaned reads and the report files MultiQC consumes.
type FastpPorts struct {
	CleanR1 gobble.Handle
	CleanR2 gobble.Handle
	JSON    gobble.Handle
	HTML    gobble.Handle
}

// AddFastp records one paired-end fastp task on parent. The shared
// builder does not call AddInput.
func AddFastp(parent modules.Parent, r1, r2 gobble.Handle, opts FastpOptions) FastpPorts {
	return addFastp(parent, r1, r2, opts)
}

// FastpPipeline returns a standalone fastp pipeline. It AddInputs r1
// and r2, then calls the same builder as AddFastp.
func FastpPipeline(r1, r2 gobble.PathSpec, opts FastpOptions) *gobble.Pipeline {
	return modules.Standalone("fastp", []modules.Input{
		{Name: "r1", Spec: r1},
		{Name: "r2", Spec: r2},
	}, func(parent modules.Parent, hs []gobble.Handle) {
		addFastp(parent, hs[0], hs[1], opts)
	})
}

func addFastp(parent modules.Parent, r1, r2 gobble.Handle, opts FastpOptions) FastpPorts {
	outDir := opts.OutDir
	if outDir.IsZero() {
		outDir = gobble.Dir("work/fastp")
	}
	cleanR1 := r1.Spec().AppendSuffix("clean").WithDir(outDir)
	cleanR2 := r2.Spec().AppendSuffix("clean").WithDir(outDir)
	prefix := r1.Spec().Base
	if prefix == "" {
		prefix = "reads"
	}
	jsonSpec := gobble.PathSpec{Dir: outDir, Base: prefix, Ext: ".fastp.json"}
	htmlSpec := gobble.PathSpec{Dir: outDir, Base: prefix, Ext: ".fastp.html"}

	cmd := []string{
		"fastp",
		"--in1", modules.MustCommandPath(r1.Spec()),
		"--in2", modules.MustCommandPath(r2.Spec()),
		"--out1", modules.MustCommandPath(cleanR1),
		"--out2", modules.MustCommandPath(cleanR2),
		"--json", modules.MustCommandPath(jsonSpec),
		"--html", modules.MustCommandPath(htmlSpec),
		"--detect_adapter_for_pe",
	}
	if n := modules.ThreadCount(opts.Resources.CPU); n > 0 {
		cmd = append(cmd, "--thread", strconv.Itoa(n))
	}
	cmd = modules.AppendLegacyExtraArgs(cmd, opts.ExtraArgs)

	task := parent.AddTask(gobble.TaskSpec{
		Name:    fastpTaskName,
		Command: cmd,
		Image:   fastpImage,
		Inputs: []gobble.Bind{
			{Name: "r1", From: r1},
			{Name: "r2", From: r2},
		},
		Outputs: []gobble.Bind{
			{Name: "clean_r1", Spec: cleanR1},
			{Name: "clean_r2", Spec: cleanR2},
			{Name: "json", Spec: jsonSpec},
			{Name: "html", Spec: htmlSpec},
		},
		Resources: opts.Resources,
	})
	return FastpPorts{
		CleanR1: task.Out("clean_r1"),
		CleanR2: task.Out("clean_r2"),
		JSON:    task.Out("json"),
		HTML:    task.Out("html"),
	}
}
