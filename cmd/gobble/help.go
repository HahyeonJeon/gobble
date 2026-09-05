package main

import "io"

const rootHelp = `Usage: gobble <command> [arguments]

Gobble is a pre-1.0 trusted-local linux/amd64 preview for an exclusive
caller-owned workspace. Agents use the Go library and generic command. Humans
receive one packed runner for one pipeline. Gobble is licensed under MIT.
Docker is an isolation convenience, not a sandbox.

Commands:
  compose    compose a pipeline from a Go package
  validate   compose then validate a pipeline package
  plan       write plan JSON for a pipeline package
  run        run a pipeline package in a workspace
  inspect    write a workspace view as JSON or JSONL
  watch      monitor pipeline progress in an interactive terminal
  resume     resume a released run
  release    close occupancy on a workspace
  pack       write a standalone linux/amd64 runner
  help       print help
  version    print version JSON

Use gobble help <command> for command help.

Supported Go contract: Compose, Validate, BuildPlan, Run, Inspect, Release,
Resume; Module, Branch, Merge, Scatter, Gather, When; PathSpec and File, Group,
Tree binds; explicit-path samplesheet parsing; structured Error, Defect, and
DefectCode values. Other exports are provisional.

Graph verbs and pack require go on PATH. Consumer internal/ packages are
unsupported. The installed command, selected module, pipeline, platform,
install family, and workspace identity must match before Pipeline runs or a
workspace changes. The published agent install is
github.com/HahyeonJeon/gobble@v0.1.0 and
github.com/HahyeonJeon/gobble/cmd/gobble@v0.1.0. No supported install uses
@latest.

Watch renders on stderr and leaves stdout empty; q exits only the monitor.
Other success stdout is protocol JSON or JSONL only. Exits are 0 success, 1 domain
or operational failure, and 2 invocation or input-shape failure. A spaced
valued flag does not consume a following token that starts with '-'; use
--flag=value.

Recovery is inspect, then release, then resume remaining work. Later-process
release never signals an unproved process PID. Proved-stopped Docker leftovers
do not wedge occupancy; unproved Docker stays unknown-backend and blocks
resume. First-horizon installed-path evidence passed on linux/amd64 for
local-pin agents and packed runners with:
  go test -tags=live ./tests/install-e2e
The published module version is v0.1.0.
`

var commandHelp = map[string]string{
	"compose": `Usage: gobble compose [package] [--sample PATH]

Compose the pipeline exported by package (default ".").
--sample PATH is the samplesheet CSV. When omitted, pipelines that read a
sheet use samplesheet.csv in the process current directory.
`,
	"validate": `Usage: gobble validate [package] [--sample PATH]

Compose then validate the pipeline exported by package (default ".").
--sample PATH is the samplesheet CSV. When omitted, pipelines that read a
sheet use samplesheet.csv in the process current directory.
`,
	"plan": `Usage: gobble plan [package] [--sample PATH]

Compose then write BuildPlan JSON for the pipeline exported by package (default ".").
--sample PATH is the samplesheet CSV. When omitted, pipelines that read a
sheet use samplesheet.csv in the process current directory.
`,
	"run": `Usage: gobble run [package] --workspace DIR [--cap N] [--sample PATH]

Compose then run the pipeline exported by package (default ".") in DIR.
--workspace is required and is not created. Omit --cap to pass 0.
--sample PATH is the samplesheet CSV. When omitted, pipelines that read a
sheet use samplesheet.csv in the process current directory.
`,
	"inspect": `Usage: gobble inspect VIEW --workspace DIR [--instance ID]

Write one workspace view. VIEW is run, instances, errors, logs, timing, dag, lineage, remaining, reuse, identity, or monitor.
--workspace is required and is not created. Omit --instance to read every reserved identity.
`,
	"watch": `Usage: gobble watch --workspace DIR

Monitor the pipeline graph, progress, samples, problems, and task logs.
Requires terminal stdin and stderr. Press / to find a sample, ! for problems,
and q to exit the monitor; the pipeline keeps running in its owning process.
--workspace is required and is not created.
`,
	"resume": `Usage: gobble resume [package] --workspace DIR [--cap N] [--sample PATH]

Compose then resume a released run of the pipeline exported by package (default ".") in DIR.
--workspace is required and is not created. Omit --cap to pass 0.
--sample PATH is the samplesheet CSV. When omitted, pipelines that read a
sheet use samplesheet.csv in the process current directory.
`,
	"release": `Usage: gobble release --workspace DIR

Reconcile and close occupancy on DIR. Documents and artifacts remain.
A later process never signals an unproved process PID. Docker unknown-backend
keeps occupancy active. --workspace is required.
`,
	"pack": `Usage: gobble pack [package] --output PATH

Write one standalone linux/amd64 runner for package (default ".") to PATH.
The runner contains Gobble and one embedded pipeline. Gobble portions are
licensed under MIT. The embedded pipeline may have a different license set by
its author. PATH is replaced atomically when it is an existing regular file.
`,
	"help": `Usage: gobble help [command]

Print root help, or help for command.
`,
	"version": `Usage: gobble version

Print version JSON. Same as gobble --version.
`,
}

const packedMITLicense = `MIT License

Copyright (c) 2026 HahyeonJeon

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
`

const packedLicenseSummary = `
License: Gobble portions are licensed under MIT. The embedded pipeline may have
a different license set by its author. See root help for Gobble's MIT notice.
`

const packedRootHelp = `Usage: gobble <command> [arguments]

This standalone linux/amd64 runner contains Gobble and one embedded pipeline.
Gobble portions are licensed under MIT. The embedded pipeline may have a
different license set by its author. It needs no Go, package operand, or pack
command. Docker is required only for pipeline tasks that use Docker.

Commands:
  compose    compose the embedded pipeline
  validate   compose then validate the embedded pipeline
  plan       write plan JSON for the embedded pipeline
  run        run the embedded pipeline in a workspace
  inspect    write a workspace view as JSON or JSONL
  watch      monitor pipeline progress in an interactive terminal
  resume     resume a released run
  release    close occupancy on a workspace
  help       print help
  version    print version JSON

Use gobble help <command> for command help.

The embedded identity must match the workspace identity. Recovery is inspect,
then release, then resume remaining work. Proved-stopped Docker leftovers do
not wedge occupancy; unproved Docker stays unknown-backend and blocks resume.

Watch renders on stderr and leaves stdout empty; q exits only the monitor.
Other success stdout is protocol JSON or JSONL only. Exits are 0 success, 1 domain
or operational failure, and 2 invocation or input-shape failure. First-horizon
installed-path evidence passed on linux/amd64 for local-pin source and packed
runners with go test -tags=live ./tests/install-e2e. The published Gobble
module version is v0.1.0.

The MIT notice below applies to Gobble portions of this runner. It does not
license the embedded pipeline unless its author says so.

` + packedMITLicense

var packedCommandHelp = map[string]string{
	"compose": `Usage: gobble compose [--sample PATH]

Compose the embedded pipeline. --sample PATH is the samplesheet CSV. When
omitted, pipelines that read a sheet use samplesheet.csv in the process current
directory.
` + packedLicenseSummary,
	"validate": `Usage: gobble validate [--sample PATH]

Compose then validate the embedded pipeline. --sample PATH is the samplesheet
CSV. When omitted, pipelines that read a sheet use samplesheet.csv in the
process current directory.
` + packedLicenseSummary,
	"plan": `Usage: gobble plan [--sample PATH]

Compose then write BuildPlan JSON for the embedded pipeline. --sample PATH is
the samplesheet CSV. When omitted, pipelines that read a sheet use
samplesheet.csv in the process current directory.
` + packedLicenseSummary,
	"run": `Usage: gobble run --workspace DIR [--cap N] [--sample PATH]

Compose then run the embedded pipeline in DIR. --workspace is required and is
not created. Omit --cap to pass 0. --sample PATH is the samplesheet CSV.
` + packedLicenseSummary,
	"inspect": `Usage: gobble inspect VIEW --workspace DIR [--instance ID]

Write one workspace view. VIEW is run, instances, errors, logs, timing, dag,
lineage, remaining, reuse, identity, or monitor. --workspace is required and is not
created. Omit --instance to read every reserved identity.
` + packedLicenseSummary,
	"watch": `Usage: gobble watch --workspace DIR

Monitor the pipeline graph, progress, samples, problems, and task logs.
Requires terminal stdin and stderr. Press / to find a sample, ! for problems,
and q to exit the monitor; the pipeline keeps running in its owning process.
--workspace is required and is not created.
` + packedLicenseSummary,
	"resume": `Usage: gobble resume --workspace DIR [--cap N] [--sample PATH]

Compose then resume a released run in DIR. --workspace is required and is not
created. Omit --cap to pass 0. --sample PATH is the samplesheet CSV.
` + packedLicenseSummary,
	"release": `Usage: gobble release --workspace DIR

Reconcile and close occupancy on DIR. Documents and artifacts remain.
Proved-stopped Docker leftovers do not wedge occupancy; unproved Docker stays
unknown-backend and blocks close. --workspace is required.
` + packedLicenseSummary,
	"help": `Usage: gobble help [command]

Print root help, or help for command.
` + packedLicenseSummary,
	"version": `Usage: gobble version

Print version JSON. Same as gobble --version.
` + packedLicenseSummary,
}

func writeHelp(stdout, stderr io.Writer, command string) int {
	text := rootHelp
	if command != "" {
		if cmd, ok := commandHelp[command]; ok {
			text = cmd
		}
	}
	if _, err := io.WriteString(stdout, text); err != nil {
		return writeErr(stderr, invalidRequest("cli", "stdout write failed"), 1)
	}
	return 0
}
