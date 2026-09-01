package scrnaseqscenario

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path"
	"slices"
	"sort"
	"strings"

	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
)

// These frozen SHA-256 values are independent script oracles. They are copied
// from the accepted command contracts, not derived from a task under test.
const (
	gtfGeneFilterScriptSHA256 = "88b6fa5123522f87beba6f5c6cf5a27139b1cc957ae05a54560af2d7f72c0989"
	gtfToT2GScriptSHA256      = "272483d07441b18b2ea722178dacafc84341837033c6cc654cb3b7ed7d1940b4"
	matrixToH5ADScriptSHA256  = "3d08c095ef77e94447c3192892dd7ffcd8aaf768ee73b617e6dca237f4a6976e"
	annDataRScriptSHA256      = "7420966f974fda822acf694c07e79b5327487d2a063a435a48c3f559c01a3879"
	h5adConcatScriptSHA256    = "61f98bc56db08091affe27f0106b2df61f7fbd2c452d434c6a7845260a3121a1"
)

// OfficialInput identifies one exact staged source byte used by command
// contract evidence. Path is workspace-relative.
type OfficialInput struct {
	Name   string
	Path   string
	SHA256 string
}

// BoundInputEvidence records one selected task input and the exact official
// staged bytes at the roots of its declared bind lineage. Paths are the paths
// visible to the selected command. It makes no claim that the hermetic output
// double executed or substituted for that command.
type BoundInputEvidence struct {
	Name           string
	Paths          []string
	OfficialInputs []OfficialInput
}

// CommandEvidence records one independently validated complete argv and every
// declared input bind's exact official-byte lineage.
type CommandEvidence struct {
	TaskName    string
	Argv        []string
	BoundInputs []BoundInputEvidence
}

type officialPlan struct {
	Tasks []pc.Task `json:"tasks"`
	DAG   struct {
		Edges []struct {
			From string `json:"from"`
			To   string `json:"to"`
		} `json:"edges"`
	} `json:"dag"`
}

// ProveOfficialBindings independently validates the plan's command-specific
// argv and traces every declared task input bind to exact staged official
// SHA-256 bytes. It reads only the nine staged roots. Produced placeholder
// bytes and the hermetic Docker double are outside this evidence boundary.
func ProveOfficialBindings(workspace string, rawPlan []byte, inputs []OfficialInput) (map[string]CommandEvidence, error) {
	if len(inputs) == 0 {
		return nil, fmt.Errorf("official scRNA evidence requires staged inputs")
	}
	var plan officialPlan
	if err := json.Unmarshal(rawPlan, &plan); err != nil {
		return nil, fmt.Errorf("decode official scRNA plan: %w", err)
	}
	byPath := make(map[string]OfficialInput, len(inputs))
	names := make(map[string]bool, len(inputs))
	for _, input := range inputs {
		decoded, err := hex.DecodeString(input.SHA256)
		if input.Name == "" || input.Path == "" || err != nil || len(decoded) != sha256.Size || hex.EncodeToString(decoded) != input.SHA256 || names[input.Name] || byPath[input.Path].Name != "" {
			return nil, fmt.Errorf("invalid or duplicate official scRNA input: %+v", input)
		}
		record, err := consumeFile(workspace, input.Path)
		if err != nil {
			return nil, err
		}
		if record.SHA256 != input.SHA256 {
			return nil, fmt.Errorf("official scRNA input %s at %s sha256 = %s, want %s", input.Name, input.Path, record.SHA256, input.SHA256)
		}
		names[input.Name] = true
		byPath[input.Path] = input
	}

	tasks := make(map[string]pc.Task, len(plan.Tasks))
	for _, task := range plan.Tasks {
		if task.ID == "" || tasks[task.ID].ID != "" {
			return nil, fmt.Errorf("invalid or duplicate selected task id %q", task.ID)
		}
		tasks[task.ID] = task
	}
	incoming := make(map[string][]string)
	for _, edge := range plan.DAG.Edges {
		from := planTaskID(edge.From, tasks)
		to := planTaskID(edge.To, tasks)
		if from == "" || to == "" {
			return nil, fmt.Errorf("invalid selected task edge %q -> %q", edge.From, edge.To)
		}
		port := strings.TrimPrefix(edge.To, to+".")
		incoming[to+"\x00"+port] = append(incoming[to+"\x00"+port], from)
	}

	directUse := make(map[string]bool, len(inputs))
	memo := make(map[string]map[string]OfficialInput, len(tasks))
	active := make(map[string]bool, len(tasks))
	var taskOrigins func(string) (map[string]OfficialInput, error)
	taskOrigins = func(taskID string) (map[string]OfficialInput, error) {
		if origins := memo[taskID]; origins != nil {
			return origins, nil
		}
		if active[taskID] {
			return nil, fmt.Errorf("selected task bind lineage contains cycle at %s", taskID)
		}
		active[taskID] = true
		defer delete(active, taskID)
		task := tasks[taskID]
		origins := make(map[string]OfficialInput)
		for _, input := range task.Inputs {
			for _, source := range ioSourcePaths(input) {
				if official := byPath[source]; official.Name != "" {
					origins[official.Path] = official
					directUse[official.Path] = true
				}
			}
			for _, producer := range incoming[taskID+"\x00"+input.Name] {
				upstream, err := taskOrigins(producer)
				if err != nil {
					return nil, err
				}
				for source, official := range upstream {
					origins[source] = official
				}
			}
		}
		if len(origins) == 0 {
			return nil, fmt.Errorf("selected task %s (%s) has no exact official staged-byte bind origin", task.ID, task.Name)
		}
		memo[taskID] = origins
		return origins, nil
	}

	evidence := make(map[string]CommandEvidence, len(plan.Tasks))
	for _, task := range plan.Tasks {
		if _, err := taskOrigins(task.ID); err != nil {
			return nil, err
		}
		commandPaths, err := commandInputPaths(task)
		if err != nil {
			return nil, fmt.Errorf("prove official argv for %s (%s): %w", task.ID, task.Name, err)
		}
		declaredPaths := allIOPaths(task.Inputs)
		if !equalPathSet(commandPaths, declaredPaths) {
			return nil, fmt.Errorf("%s command-bound paths = %q, want every declared input path %q", task.ID, commandPaths, declaredPaths)
		}
		bound := make([]BoundInputEvidence, 0, len(task.Inputs))
		for _, input := range task.Inputs {
			origins := make(map[string]OfficialInput)
			for _, source := range ioSourcePaths(input) {
				if official := byPath[source]; official.Name != "" {
					origins[official.Path] = official
				}
			}
			for _, producer := range incoming[task.ID+"\x00"+input.Name] {
				upstream, err := taskOrigins(producer)
				if err != nil {
					return nil, err
				}
				for source, official := range upstream {
					origins[source] = official
				}
			}
			if len(origins) == 0 {
				return nil, fmt.Errorf("selected task %s input bind %s has no exact official staged-byte origin", task.ID, input.Name)
			}
			bound = append(bound, BoundInputEvidence{
				Name:           input.Name,
				Paths:          ioPaths(input),
				OfficialInputs: sortedOfficialInputs(origins),
			})
		}
		evidence[task.ID] = CommandEvidence{TaskName: task.Name, Argv: taskArgv(task), BoundInputs: bound}
	}
	for _, input := range inputs {
		if !directUse[input.Path] {
			return nil, fmt.Errorf("official staged input %s at %s has no direct selected-task bind", input.Name, input.Path)
		}
	}
	return evidence, nil
}

func commandInputPaths(task pc.Task) ([]string, error) {
	input := func(name string) (string, error) { return namedIOPath(task.Inputs, name) }
	output := func(name string) (string, error) { return namedIOPath(task.Outputs, name) }
	switch task.Name {
	case "cat_fastq":
		inputs := allIOPaths(task.Inputs)
		out, err := output("fastq")
		if err != nil {
			return nil, err
		}
		want := frozenShellCommand(append([]string{"cat"}, inputs...)) + " > " + frozenShellQuote(out)
		if err := exactScript(task, want); err != nil {
			return nil, err
		}
		return inputs, nil
	case "fastqc":
		reads, err := input("reads")
		if err != nil {
			return nil, err
		}
		html, err := output("html")
		if err != nil {
			return nil, err
		}
		zip, err := output("zip")
		if err != nil {
			return nil, err
		}
		if err := exactPath("FastQC zip", zip, strings.TrimSuffix(html, ".html")+".zip"); err != nil {
			return nil, err
		}
		return []string{reads}, exactCommand(task, []string{"fastqc", "--outdir", path.Dir(html), "--noextract", "--threads", "2", reads})
	case "gtf_gene_filter":
		fasta, err := input("fasta")
		if err != nil {
			return nil, err
		}
		gtf, err := input("gtf")
		if err != nil {
			return nil, err
		}
		filtered, err := output("filtered_gtf")
		if err != nil {
			return nil, err
		}
		if err := embeddedCommand(task, "python", "-c", gtfGeneFilterScriptSHA256, fasta, gtf, filtered); err != nil {
			return nil, err
		}
		return []string{fasta, gtf}, nil
	case "gffread_transcriptome":
		gtf, err := input("gtf")
		if err != nil {
			return nil, err
		}
		fasta, err := input("fasta")
		if err != nil {
			return nil, err
		}
		transcripts, err := output("transcript_fasta")
		if err != nil {
			return nil, err
		}
		return []string{gtf, fasta}, exactCommand(task, []string{"gffread", "-F", gtf, "-w", transcripts, "-g", fasta})
	case "gtf_to_t2g":
		gtf, err := input("gtf")
		if err != nil {
			return nil, err
		}
		t2g, err := output("t2g")
		if err != nil {
			return nil, err
		}
		if err := embeddedCommand(task, "python", "-c", gtfToT2GScriptSHA256, gtf, t2g); err != nil {
			return nil, err
		}
		return []string{gtf}, nil
	case "simpleaf_index":
		transcripts, err := input("transcript_fasta")
		if err != nil {
			return nil, err
		}
		index, err := output("index")
		if err != nil {
			return nil, err
		}
		if path.Base(index) != "index" {
			return nil, fmt.Errorf("Simpleaf index Tree = %q, want index member root", index)
		}
		command := []string{"simpleaf", "index", "--threads", "4", "--ref-seq", transcripts, "--output", path.Dir(index)}
		if err := exactScript(task, "'simpleaf' 'set-paths'\n"+frozenShellCommand(command)); err != nil {
			return nil, err
		}
		return []string{transcripts}, nil
	case "simpleaf_quant":
		index, err := input("index")
		if err != nil {
			return nil, err
		}
		t2g, err := input("t2g")
		if err != nil {
			return nil, err
		}
		whitelist, err := input("whitelist")
		if err != nil {
			return nil, err
		}
		read1, err := input("read1")
		if err != nil {
			return nil, err
		}
		read2, err := input("read2")
		if err != nil {
			return nil, err
		}
		mapDir, err := output("map")
		if err != nil {
			return nil, err
		}
		quantDir, err := output("quant")
		if err != nil {
			return nil, err
		}
		outDir := path.Dir(mapDir)
		if err := exactPath("Simpleaf map Tree", mapDir, path.Join(outDir, "af_map")); err != nil {
			return nil, err
		}
		if err := exactPath("Simpleaf quantification Tree", quantDir, path.Join(outDir, "af_quant")); err != nil {
			return nil, err
		}
		command := []string{"simpleaf", "quant", "--index", index, "--t2g-map", t2g, "--chemistry", "10xv2", "--reads1", read1, "--reads2", read2, "--resolution", "cr-like", "--output", outDir, "--threads", "4", "--anndata-out", "--unfiltered-pl", whitelist}
		if err := exactScript(task, "'simpleaf' 'set-paths'\n"+frozenShellCommand(command)); err != nil {
			return nil, err
		}
		return []string{index, t2g, whitelist, read1, read2}, nil
	case "qcatch":
		quant, err := input("quant")
		if err != nil {
			return nil, err
		}
		report, err := output("report")
		if err != nil {
			return nil, err
		}
		metrics, err := output("metrics")
		if err != nil {
			return nil, err
		}
		filtered, err := output("filtered_h5ad")
		if err != nil {
			return nil, err
		}
		outDir := path.Dir(report)
		if err := exactPath("QCatch report", report, path.Join(outDir, "QCatch_report.html")); err != nil {
			return nil, err
		}
		if err := exactPath("QCatch metrics", metrics, path.Join(outDir, "summary_table.csv")); err != nil {
			return nil, err
		}
		if err := exactPath("QCatch filtered h5ad", filtered, path.Join(outDir, "filtered_quants.h5ad")); err != nil {
			return nil, err
		}
		command := []string{"qcatch", "--input", quant, "--output", outDir, "--chemistry", "10X_3p_v2", "--save_filtered_h5ad", "--export_summary_table"}
		return []string{quant}, exactCommand(task, command)
	case "matrix_to_h5ad":
		quant, err := input("quant")
		if err != nil {
			return nil, err
		}
		h5ad, err := output("h5ad")
		if err != nil {
			return nil, err
		}
		if err := embeddedCommand(task, "python", "-c", matrixToH5ADScriptSHA256, quant, h5ad, taskParam(task, "sample"), taskParam(task, "expected_cells"), taskParam(task, "seq_center")); err != nil {
			return nil, err
		}
		return []string{quant}, nil
	case "anndatar_convert":
		h5ad, err := input("h5ad")
		if err != nil {
			return nil, err
		}
		seurat, err := output("seurat_rds")
		if err != nil {
			return nil, err
		}
		sce, err := output("sce_rds")
		if err != nil {
			return nil, err
		}
		if err := embeddedCommand(task, "Rscript", "-e", annDataRScriptSHA256, h5ad, seurat, sce); err != nil {
			return nil, err
		}
		return []string{h5ad}, nil
	case "h5ad_concat":
		inputs := allIOPaths(task.Inputs)
		combined, err := output("h5ad")
		if err != nil {
			return nil, err
		}
		operands := []string{combined}
		for _, inputPath := range inputs {
			operands = append(operands, path.Base(path.Dir(inputPath)), inputPath)
		}
		if err := embeddedCommand(task, "python", "-c", h5adConcatScriptSHA256, operands...); err != nil {
			return nil, err
		}
		return inputs, nil
	case "multiqc":
		html, err := output("html")
		if err != nil {
			return nil, err
		}
		data, err := output("data")
		if err != nil {
			return nil, err
		}
		if err := exactPath("MultiQC data Tree", data, path.Join(path.Dir(html), "multiqc_data")); err != nil {
			return nil, err
		}
		return allIOPaths(task.Inputs), exactCommand(task, []string{"multiqc", "--force", "--outdir", path.Dir(html), "."})
	default:
		return nil, fmt.Errorf("no official command contract for selected task %q", task.Name)
	}
}

func exactPath(label, got, want string) error {
	if got != want {
		return fmt.Errorf("%s path = %q, want %q", label, got, want)
	}
	return nil
}

func exactCommand(task pc.Task, want []string) error {
	if task.Script != "" || !slices.Equal(task.Command, want) {
		return fmt.Errorf("command = %q script = %q, want command %q", task.Command, task.Script, want)
	}
	return nil
}

func exactScript(task pc.Task, want string) error {
	wantCommand := []string{"sh", "-c", "set -eu\n" + want}
	if !slices.Equal(task.Command, wantCommand) || task.Script != want {
		return fmt.Errorf("command = %q script = %q, want command %q and exact script %q", task.Command, task.Script, wantCommand, want)
	}
	return nil
}

func frozenShellCommand(command []string) string {
	quoted := make([]string, len(command))
	for i, token := range command {
		quoted[i] = frozenShellQuote(token)
	}
	return strings.Join(quoted, " ")
}

func frozenShellQuote(token string) string {
	return "'" + strings.ReplaceAll(token, "'", "'\"'\"'") + "'"
}

func embeddedCommand(task pc.Task, executable, flag, wantScriptSHA256 string, operands ...string) error {
	if len(task.Command) != len(operands)+3 || task.Command[0] != executable || task.Command[1] != flag || task.Script != "" {
		return fmt.Errorf("embedded command = %q script = %q", task.Command, task.Script)
	}
	gotScriptSHA256 := fmt.Sprintf("%x", sha256.Sum256([]byte(task.Command[2])))
	if gotScriptSHA256 != wantScriptSHA256 {
		return fmt.Errorf("embedded %s script sha256 = %s, want frozen oracle %s", executable, gotScriptSHA256, wantScriptSHA256)
	}
	if !slices.Equal(task.Command[3:], operands) {
		return fmt.Errorf("embedded %s operands = %q, want %q", executable, task.Command[3:], operands)
	}
	return nil
}

func planTaskID(endpoint string, tasks map[string]pc.Task) string {
	matched := ""
	for id := range tasks {
		if len(id) > len(matched) && strings.HasPrefix(endpoint, id+".") {
			matched = id
		}
	}
	return matched
}

func equalPathSet(left, right []string) bool {
	left = append([]string(nil), left...)
	right = append([]string(nil), right...)
	sort.Strings(left)
	sort.Strings(right)
	return slices.Equal(left, right)
}

func sortedOfficialInputs(inputs map[string]OfficialInput) []OfficialInput {
	out := make([]OfficialInput, 0, len(inputs))
	for _, input := range inputs {
		out = append(out, input)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

func namedIOPath(values []pc.IO, name string) (string, error) {
	for _, value := range values {
		if value.Name != name {
			continue
		}
		paths := ioPaths(value)
		if len(paths) != 1 {
			return "", fmt.Errorf("%s has %d paths, want one", name, len(paths))
		}
		return paths[0], nil
	}
	return "", fmt.Errorf("missing IO %q", name)
}

func allIOPaths(values []pc.IO) []string {
	var paths []string
	for _, value := range values {
		paths = append(paths, ioPaths(value)...)
	}
	return paths
}

func ioPaths(value pc.IO) []string {
	if value.Path != "" {
		return []string{value.Path}
	}
	if value.Source != "" {
		return []string{value.Source}
	}
	paths := make([]string, 0, len(value.Members))
	for _, member := range value.Members {
		paths = append(paths, member.Path)
	}
	return paths
}

func ioSourcePaths(value pc.IO) []string {
	if value.Source != "" {
		return []string{value.Source}
	}
	if value.Path != "" {
		return []string{value.Path}
	}
	paths := make([]string, 0, len(value.Members))
	for _, member := range value.Members {
		paths = append(paths, member.Path)
	}
	return paths
}

func taskParam(task pc.Task, name string) string {
	for _, param := range task.Params {
		if param.Name == name {
			return param.Value
		}
	}
	return ""
}
