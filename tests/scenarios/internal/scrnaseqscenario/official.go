package scrnaseqscenario

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/HahyeonJeon/gobble/assets/modules"
	"github.com/HahyeonJeon/gobble/assets/pipelines/scrnaseq"
	pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"
)

// OfficialInput identifies one exact staged source byte used by command
// contract evidence. Path is workspace-relative.
type OfficialInput struct {
	Name   string
	Path   string
	SHA256 string
}

// OperationEvidence records the actual Docker argv and the exact staged source
// bytes re-opened while proving one selected command contract. It is separate
// from the placeholder outputs used only for engine occupancy evidence.
type OperationEvidence struct {
	TaskName     string
	Argv         []string
	StagedInputs []ConsumedInput
}

type officialEvidence struct {
	workspace  string
	inputs     map[string]OfficialInput
	origins    map[string][]ConsumedInput
	operations map[string]OperationEvidence
}

// NewRuntimeWithOfficialInputs composes the scRNA graph over already staged
// official inputs and enables command-specific argv and source-byte evidence.
func NewRuntimeWithOfficialInputs(t *testing.T, config scrnaseq.Config, workspace string, inputs []OfficialInput) *Runtime {
	t.Helper()
	r := newRuntime(t, config, workspace, false)
	if err := r.docker.enableOfficialEvidence(workspace, inputs); err != nil {
		t.Fatalf("enable official scRNA evidence: %v", err)
	}
	return r
}

// OperationEvidence returns a deep copy keyed by task id.
func (r *Runtime) OperationEvidence() map[string]OperationEvidence {
	return r.docker.operationEvidence()
}

func (f *fakeDocker) enableOfficialEvidence(workspace string, inputs []OfficialInput) error {
	if len(inputs) == 0 {
		return fmt.Errorf("official scRNA evidence requires staged inputs")
	}
	official := &officialEvidence{
		workspace:  workspace,
		inputs:     make(map[string]OfficialInput, len(inputs)),
		origins:    make(map[string][]ConsumedInput),
		operations: make(map[string]OperationEvidence),
	}
	names := make(map[string]bool, len(inputs))
	for _, input := range inputs {
		if input.Name == "" || input.Path == "" || len(input.SHA256) != 64 || names[input.Name] || official.inputs[input.Path].Name != "" {
			return fmt.Errorf("invalid or duplicate official scRNA input: %+v", input)
		}
		record, err := consumeFile(workspace, input.Path)
		if err != nil {
			return err
		}
		if record.SHA256 != input.SHA256 {
			return fmt.Errorf("official scRNA input %s at %s sha256 = %s, want %s", input.Name, input.Path, record.SHA256, input.SHA256)
		}
		names[input.Name] = true
		official.inputs[input.Path] = input
	}
	f.mu.Lock()
	f.official = official
	f.mu.Unlock()
	return nil
}

func (f *fakeDocker) operationEvidence() map[string]OperationEvidence {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.official == nil {
		return nil
	}
	out := make(map[string]OperationEvidence, len(f.official.operations))
	for id, operation := range f.official.operations {
		operation.Argv = append([]string(nil), operation.Argv...)
		operation.StagedInputs = append([]ConsumedInput(nil), operation.StagedInputs...)
		out[id] = operation
	}
	return out
}

func (f *fakeDocker) proveOfficialTask(mount string, task pc.Task, actualArgv []string) ([]ConsumedInput, error) {
	paths, err := commandInputPaths(task, actualArgv)
	if err != nil {
		return nil, fmt.Errorf("prove official argv for %s (%s): %w", task.ID, task.Name, err)
	}
	declared := make(map[string]bool)
	for _, input := range task.Inputs {
		for _, inputPath := range ioPaths(input) {
			declared[inputPath] = true
		}
	}
	origins := make(map[string]ConsumedInput)
	for _, inputPath := range paths {
		if !declared[inputPath] {
			return nil, fmt.Errorf("%s command operand %s is not a declared input", task.ID, inputPath)
		}
		if err := consumeOperationPath(mount, inputPath); err != nil {
			return nil, fmt.Errorf("%s consume command operand %s: %w", task.ID, inputPath, err)
		}
		inputOrigins := f.officialOrigins(inputPath)
		if len(inputOrigins) == 0 {
			return nil, fmt.Errorf("%s command operand %s has no exact official staged-byte origin", task.ID, inputPath)
		}
		for _, origin := range inputOrigins {
			origins[origin.Path] = origin
		}
	}
	verified := make([]ConsumedInput, 0, len(origins))
	for _, origin := range origins {
		record, err := consumeFile(f.official.workspace, origin.Path)
		if err != nil {
			return nil, fmt.Errorf("%s re-open official staged input %s: %w", task.ID, origin.Path, err)
		}
		if record.SHA256 != origin.SHA256 {
			return nil, fmt.Errorf("%s official staged input %s sha256 = %s, want %s", task.ID, origin.Path, record.SHA256, origin.SHA256)
		}
		verified = append(verified, record)
	}
	sort.Slice(verified, func(i, j int) bool { return verified[i].Path < verified[j].Path })
	return verified, nil
}

func (f *fakeDocker) officialOrigins(inputPath string) []ConsumedInput {
	if input := f.official.inputs[inputPath]; input.Name != "" {
		return []ConsumedInput{{Path: input.Path, SHA256: input.SHA256}}
	}
	return f.official.origins[inputPath]
}

func (f *fakeDocker) recordOfficialTask(task pc.Task, actualArgv []string, origins []ConsumedInput) {
	record := OperationEvidence{
		TaskName:     task.Name,
		Argv:         append([]string(nil), actualArgv...),
		StagedInputs: append([]ConsumedInput(nil), origins...),
	}
	f.official.operations[task.ID] = record
	for _, output := range task.Outputs {
		for _, outputPath := range ioPaths(output) {
			f.official.origins[outputPath] = append([]ConsumedInput(nil), origins...)
		}
	}
}

func consumeOperationPath(mount, rel string) error {
	fullPath := filepath.Join(mount, filepath.FromSlash(rel))
	info, err := os.Stat(fullPath)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		_, err = consumeFile(mount, rel)
		return err
	}
	files := 0
	err = filepath.WalkDir(fullPath, func(filePath string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !entry.Type().IsRegular() {
			return nil
		}
		relPath, err := filepath.Rel(mount, filePath)
		if err != nil {
			return err
		}
		if _, err := consumeFile(mount, filepath.ToSlash(relPath)); err != nil {
			return err
		}
		files++
		return nil
	})
	if err != nil {
		return err
	}
	if files == 0 {
		return fmt.Errorf("directory contains no regular files")
	}
	return nil
}

func commandInputPaths(task pc.Task, actualArgv []string) ([]string, error) {
	if !slices.Equal(actualArgv, taskArgv(task)) {
		return nil, fmt.Errorf("Docker argv = %q, want planned argv %q", actualArgv, taskArgv(task))
	}
	input := func(name string) (string, error) { return namedIOPath(task.Inputs, name) }
	output := func(name string) (string, error) { return namedIOPath(task.Outputs, name) }
	switch task.Name {
	case "cat_fastq":
		inputs := allIOPaths(task.Inputs)
		out, err := output("fastq")
		if err != nil {
			return nil, err
		}
		if err := exactScript(task, modules.ShellRedirect(append([]string{"cat"}, inputs...), out)); err != nil {
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
		if err := embeddedCommand(task, "python", "-c", []string{"FASTA contains no sequence names", "GTF has no annotations on reference sequences"}, fasta, gtf, filtered); err != nil {
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
		if err := embeddedCommand(task, "python", "-c", []string{"transcript_id", "GTF contains no complete transcript-to-gene relations"}, gtf, t2g); err != nil {
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
		if err := exactScript(task, "'simpleaf' 'set-paths'\n"+modules.ShellCommand(command)); err != nil {
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
		if err := exactScript(task, "'simpleaf' 'set-paths'\n"+modules.ShellCommand(command)); err != nil {
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
		if err := embeddedCommand(task, "python", "-c", []string{"source + \"/alevin/quants.h5ad\"", "adata.write_h5ad"}, quant, h5ad, taskParam(task, "sample"), taskParam(task, "expected_cells"), taskParam(task, "seq_center")); err != nil {
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
		if err := embeddedCommand(task, "Rscript", "-e", []string{"read_h5ad", "as_Seurat", "as_SingleCellExperiment"}, h5ad, seurat, sce); err != nil {
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
		if err := embeddedCommand(task, "python", "-c", []string{"ad.concat", "combined.write_h5ad"}, operands...); err != nil {
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
	if task.Script != want {
		return fmt.Errorf("command = %q script = %q, want exact script %q", task.Command, task.Script, want)
	}
	return nil
}

func embeddedCommand(task pc.Task, executable, flag string, markers []string, operands ...string) error {
	if len(task.Command) != len(operands)+3 || task.Command[0] != executable || task.Command[1] != flag || task.Script != "" {
		return fmt.Errorf("embedded command = %q script = %q", task.Command, task.Script)
	}
	for _, marker := range markers {
		if !strings.Contains(task.Command[2], marker) {
			return fmt.Errorf("embedded %s command omits contract marker %q", executable, marker)
		}
	}
	want := append([]string{executable, flag, task.Command[2]}, operands...)
	if !slices.Equal(task.Command, want) {
		return fmt.Errorf("embedded command = %q, want %q", task.Command, want)
	}
	return nil
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

func taskParam(task pc.Task, name string) string {
	for _, param := range task.Params {
		if param.Name == name {
			return param.Value
		}
	}
	return ""
}
