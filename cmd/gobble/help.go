package main

import "io"

const rootHelp = `Usage: gobble <command> [arguments]

Commands:
  compose    compose a pipeline from a Go package
  validate   compose then validate a pipeline package
  plan       write plan JSON for a pipeline package
  run        run a pipeline package in a workspace
  inspect    write a workspace view as JSON or JSONL
  resume     resume a released run
  release    close occupancy on a workspace
  help       print help
  version    print version JSON

Use gobble help <command> for command help.

Recovery after a contained failure or interrupt is inspect, then release, then resume.
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

Write one workspace view. VIEW is run, instances, errors, logs, timing, dag, lineage, remaining, or reuse.
--workspace is required and is not created. Omit --instance to read every reserved identity.
`,
	"resume": `Usage: gobble resume [package] --workspace DIR [--cap N] [--sample PATH]

Compose then resume a released run of the pipeline exported by package (default ".") in DIR.
--workspace is required and is not created. Omit --cap to pass 0.
--sample PATH is the samplesheet CSV. When omitted, pipelines that read a
sheet use samplesheet.csv in the process current directory.
`,
	"release": `Usage: gobble release --workspace DIR

Close occupancy on DIR. Documents and artifacts remain. --workspace is required.
`,
	"help": `Usage: gobble help [command]

Print root help, or help for command.
`,
	"version": `Usage: gobble version

Print version JSON. Same as gobble --version.
`,
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
