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

	sampleXR1Name      = "Sample_X_S1_L001_R1_001.fastq.gz"
	sampleXR2Name      = "Sample_X_S1_L001_R2_001.fastq.gz"
	sampleYL1R1Name    = "Sample_Y_S1_L001_R1_001.fastq.gz"
	sampleYL1R2Name    = "Sample_Y_S1_L001_R2_001.fastq.gz"
	sampleYL2R1Name    = "Sample_Y_S1_L002_R1_001.fastq.gz"
	sampleYL2R2Name    = "Sample_Y_S1_L002_R2_001.fastq.gz"
	referenceFASTAName = "GRCm38.p6.genome.chr19.fa"
	referenceGTFName   = "gencode.vM19.annotation.chr19.gtf"
	whitelistName      = "10x_V2_barcode_whitelist.txt.gz"
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

type expectedBoundInput struct {
	Producers     []string
	OfficialNames []string
}

// officialInputOracle deliberately freezes the manifest identities again so a
// plan or caller-supplied identity cannot become its own expected value.
var officialInputOracle = []OfficialInput{
	{Name: sampleXR1Name, Path: "in/reads/Sample_X_S1_L001_R1_001.fastq.gz", SHA256: "a318e85709d690b25c763fb3f156fc98d0ce46360115f3de3d9b1b2ecc55a1f7"},
	{Name: sampleXR2Name, Path: "in/reads/Sample_X_S1_L001_R2_001.fastq.gz", SHA256: "1514b26ce8e972ac7ba1860d591fa72e8c1fb6f716825651c2eb24b59c1a38d2"},
	{Name: sampleYL1R1Name, Path: "in/reads/Sample_Y_S1_L001_R1_001.fastq.gz", SHA256: "a318e85709d690b25c763fb3f156fc98d0ce46360115f3de3d9b1b2ecc55a1f7"},
	{Name: sampleYL1R2Name, Path: "in/reads/Sample_Y_S1_L001_R2_001.fastq.gz", SHA256: "1514b26ce8e972ac7ba1860d591fa72e8c1fb6f716825651c2eb24b59c1a38d2"},
	{Name: sampleYL2R1Name, Path: "in/reads/Sample_Y_S1_L002_R1_001.fastq.gz", SHA256: "f33ae3d31e78020843b78198508b130506b2e7899eb2ecc658d2640b845a1114"},
	{Name: sampleYL2R2Name, Path: "in/reads/Sample_Y_S1_L002_R2_001.fastq.gz", SHA256: "6270bd60af5341463fd425079cfc067990221dfe78437880240b5894fa30477f"},
	{Name: referenceFASTAName, Path: "in/reference/genome.fa", SHA256: "b03ea6d17e5e02cd092dabb58358a30a70bfe639f19c71b296c1109ad0b0b931"},
	{Name: referenceGTFName, Path: "in/reference/genes.gtf", SHA256: "2270e0de93df1e12cdcb91cb9bd29640d318cd676d8e538c52a45bebef6c1247"},
	{Name: whitelistName, Path: "in/reference/10x_V2_barcode_whitelist.txt.gz", SHA256: "4101687b6cbb947b8ace340c38eecf872a1a59f230eab23becacd038a46c6fb5"},
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
// argv and compares every declared task input bind with a frozen producer-port
// and official SHA-256 identity oracle. It reads only the nine staged roots.
// Produced placeholder bytes and the hermetic Docker double are outside this
// evidence boundary.
func ProveOfficialBindings(workspace string, rawPlan []byte, inputs []OfficialInput) (map[string]CommandEvidence, error) {
	if len(inputs) != len(officialInputOracle) {
		return nil, fmt.Errorf("official scRNA staged identities = %d, want frozen set of %d", len(inputs), len(officialInputOracle))
	}
	var plan officialPlan
	if err := json.Unmarshal(rawPlan, &plan); err != nil {
		return nil, fmt.Errorf("decode official scRNA plan: %w", err)
	}
	expectedByName := make(map[string]OfficialInput, len(officialInputOracle))
	for _, input := range officialInputOracle {
		expectedByName[input.Name] = input
	}
	byPath := make(map[string]OfficialInput, len(inputs))
	seenNames := make(map[string]bool, len(inputs))
	for _, input := range inputs {
		decoded, err := hex.DecodeString(input.SHA256)
		if input.Name == "" || input.Path == "" || err != nil || len(decoded) != sha256.Size || hex.EncodeToString(decoded) != input.SHA256 || seenNames[input.Name] || byPath[input.Path].Name != "" {
			return nil, fmt.Errorf("invalid or duplicate official scRNA input: %+v", input)
		}
		if want, ok := expectedByName[input.Name]; !ok || input != want {
			return nil, fmt.Errorf("official scRNA input identity = %+v, want frozen identity %+v", input, want)
		}
		record, err := consumeFile(workspace, input.Path)
		if err != nil {
			return nil, err
		}
		if record.SHA256 != input.SHA256 {
			return nil, fmt.Errorf("official scRNA input %s at %s sha256 = %s, want %s", input.Name, input.Path, record.SHA256, input.SHA256)
		}
		seenNames[input.Name] = true
		byPath[input.Path] = input
	}

	expectedBindings := expectedOfficialBindings()
	if len(plan.Tasks) != len(expectedBindings) {
		return nil, fmt.Errorf("selected official tasks = %d, want frozen bind contracts for %d", len(plan.Tasks), len(expectedBindings))
	}
	tasks := make(map[string]pc.Task, len(plan.Tasks))
	for _, task := range plan.Tasks {
		if task.ID == "" || tasks[task.ID].ID != "" {
			return nil, fmt.Errorf("invalid or duplicate selected task id %q", task.ID)
		}
		if _, ok := expectedBindings[task.ID]; !ok {
			return nil, fmt.Errorf("selected task %s (%s) has no independent official bind contract", task.ID, task.Name)
		}
		tasks[task.ID] = task
	}
	incoming := make(map[string][]string)
	for _, edge := range plan.DAG.Edges {
		fromTask := planTaskID(edge.From, tasks)
		toTask := planTaskID(edge.To, tasks)
		if fromTask == "" || toTask == "" {
			return nil, fmt.Errorf("invalid selected task edge %q -> %q", edge.From, edge.To)
		}
		fromPort := strings.TrimPrefix(edge.From, fromTask+".")
		toPort := strings.TrimPrefix(edge.To, toTask+".")
		if fromPort == "" || !hasIOName(tasks[fromTask].Outputs, fromPort) || toPort == "" || !hasIOName(tasks[toTask].Inputs, toPort) {
			return nil, fmt.Errorf("invalid selected task edge endpoint %q -> %q", edge.From, edge.To)
		}
		key := toTask + "\x00" + toPort
		if slices.Contains(incoming[key], edge.From) {
			return nil, fmt.Errorf("duplicate selected task edge %q -> %q", edge.From, edge.To)
		}
		incoming[key] = append(incoming[key], edge.From)
	}

	directUse := make(map[string]bool, len(inputs))
	outputMemo := make(map[string]map[string]OfficialInput, len(tasks))
	activeOutputs := make(map[string]bool, len(tasks))
	var outputOrigins func(string) (map[string]OfficialInput, error)
	var bindOrigins func(string, pc.IO) (map[string]OfficialInput, error)
	bindOrigins = func(taskID string, input pc.IO) (map[string]OfficialInput, error) {
		origins := make(map[string]OfficialInput)
		for _, source := range ioSourcePaths(input) {
			if official := byPath[source]; official.Name != "" {
				origins[official.Path] = official
				directUse[official.Path] = true
			}
		}
		for _, producer := range incoming[taskID+"\x00"+input.Name] {
			upstream, err := outputOrigins(producer)
			if err != nil {
				return nil, err
			}
			for source, official := range upstream {
				origins[source] = official
			}
		}
		return origins, nil
	}
	outputOrigins = func(producer string) (map[string]OfficialInput, error) {
		if origins := outputMemo[producer]; origins != nil {
			return origins, nil
		}
		if activeOutputs[producer] {
			return nil, fmt.Errorf("selected task bind lineage contains cycle at %s", producer)
		}
		activeOutputs[producer] = true
		defer delete(activeOutputs, producer)
		taskID := planTaskID(producer, tasks)
		task := tasks[taskID]
		origins := make(map[string]OfficialInput)
		for _, input := range task.Inputs {
			bound, err := bindOrigins(taskID, input)
			if err != nil {
				return nil, err
			}
			for source, official := range bound {
				origins[source] = official
			}
		}
		if len(origins) == 0 {
			return nil, fmt.Errorf("selected task output %s has no exact official staged-byte bind origin", producer)
		}
		outputMemo[producer] = origins
		return origins, nil
	}

	evidence := make(map[string]CommandEvidence, len(plan.Tasks))
	for _, task := range plan.Tasks {
		commandPaths, err := commandInputPaths(task)
		if err != nil {
			return nil, fmt.Errorf("prove official argv for %s (%s): %w", task.ID, task.Name, err)
		}
		declaredPaths := allIOPaths(task.Inputs)
		if !equalPathSet(commandPaths, declaredPaths) {
			return nil, fmt.Errorf("%s command-bound paths = %q, want every declared input path %q", task.ID, commandPaths, declaredPaths)
		}
		wantBinds := expectedBindings[task.ID]
		if len(task.Inputs) != len(wantBinds) {
			return nil, fmt.Errorf("selected task %s input binds = %d, want %d independent bind contracts", task.ID, len(task.Inputs), len(wantBinds))
		}
		bound := make([]BoundInputEvidence, 0, len(task.Inputs))
		for _, input := range task.Inputs {
			want, ok := wantBinds[input.Name]
			if !ok {
				return nil, fmt.Errorf("selected task %s input bind %s has no independent bind contract", task.ID, input.Name)
			}
			gotProducers := append([]string(nil), incoming[task.ID+"\x00"+input.Name]...)
			wantProducers := append([]string(nil), want.Producers...)
			sort.Strings(gotProducers)
			sort.Strings(wantProducers)
			if !slices.Equal(gotProducers, wantProducers) {
				return nil, fmt.Errorf("selected task %s input bind %s producers = %q, want independent producer-port set %q", task.ID, input.Name, gotProducers, wantProducers)
			}
			origins, err := bindOrigins(task.ID, input)
			if err != nil {
				return nil, err
			}
			if len(origins) == 0 {
				return nil, fmt.Errorf("selected task %s input bind %s has no exact official staged-byte origin", task.ID, input.Name)
			}
			gotOfficial := sortedOfficialInputs(origins)
			wantOfficial, err := expectedOfficialInputs(want.OfficialNames)
			if err != nil {
				return nil, fmt.Errorf("invalid independent bind contract for %s.%s: %w", task.ID, input.Name, err)
			}
			if !slices.Equal(gotOfficial, wantOfficial) {
				return nil, fmt.Errorf("selected task %s input bind %s official identities = %+v, want independent exact SHA-256 identities %+v", task.ID, input.Name, gotOfficial, wantOfficial)
			}
			bound = append(bound, BoundInputEvidence{
				Name:           input.Name,
				Paths:          ioPaths(input),
				OfficialInputs: gotOfficial,
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

func expectedOfficialBindings() map[string]map[string]expectedBoundInput {
	reference := []string{referenceFASTAName, referenceGTFName}
	sampleX := []string{sampleXR1Name, sampleXR2Name}
	sampleY := []string{sampleYL1R1Name, sampleYL1R2Name, sampleYL2R1Name, sampleYL2R2Name}
	sampleXProduct := append(append([]string(nil), reference...), whitelistName)
	sampleXProduct = append(sampleXProduct, sampleX...)
	sampleYProduct := append(append([]string(nil), reference...), whitelistName)
	sampleYProduct = append(sampleYProduct, sampleY...)

	expected := map[string]map[string]expectedBoundInput{
		"reference.gtf_gene_filter": {
			"fasta": expectedBind("", referenceFASTAName),
			"gtf":   expectedBind("", referenceGTFName),
		},
		"reference.gffread_transcriptome": {
			"gtf":   expectedBind("reference.gtf_gene_filter.filtered_gtf", reference...),
			"fasta": expectedBind("", referenceFASTAName),
		},
		"reference.gtf_to_t2g": {
			"gtf": expectedBind("reference.gtf_gene_filter.filtered_gtf", reference...),
		},
		"reference.simpleaf_index": {
			"transcript_fasta": expectedBind("reference.gffread_transcriptome.transcript_fasta", reference...),
		},
		"Sample_Y.consolidate_r1.cat_fastq": {
			"run_1": expectedBind("", sampleYL1R1Name),
			"run_2": expectedBind("", sampleYL2R1Name),
		},
		"Sample_Y.consolidate_r2.cat_fastq": {
			"run_1": expectedBind("", sampleYL1R2Name),
			"run_2": expectedBind("", sampleYL2R2Name),
		},
		"Sample_X.simpleaf_quant": {
			"index":     expectedBind("reference.simpleaf_index.index", reference...),
			"t2g":       expectedBind("reference.gtf_to_t2g.t2g", reference...),
			"whitelist": expectedBind("", whitelistName),
			"read1":     expectedBind("", sampleXR1Name),
			"read2":     expectedBind("", sampleXR2Name),
		},
		"Sample_Y.simpleaf_quant": {
			"index":     expectedBind("reference.simpleaf_index.index", reference...),
			"t2g":       expectedBind("reference.gtf_to_t2g.t2g", reference...),
			"whitelist": expectedBind("", whitelistName),
			"read1":     expectedBind("Sample_Y.consolidate_r1.cat_fastq.fastq", sampleYL1R1Name, sampleYL2R1Name),
			"read2":     expectedBind("Sample_Y.consolidate_r2.cat_fastq.fastq", sampleYL1R2Name, sampleYL2R2Name),
		},
		"Sample_X.qcatch": {
			"quant": expectedBind("Sample_X.simpleaf_quant.quant", sampleXProduct...),
		},
		"Sample_Y.qcatch": {
			"quant": expectedBind("Sample_Y.simpleaf_quant.quant", sampleYProduct...),
		},
		"Sample_X.matrix_to_h5ad": {
			"quant": expectedBind("Sample_X.simpleaf_quant.quant", sampleXProduct...),
		},
		"Sample_Y.matrix_to_h5ad": {
			"quant": expectedBind("Sample_Y.simpleaf_quant.quant", sampleYProduct...),
		},
		"Sample_X.anndatar_convert": {
			"h5ad": expectedBind("Sample_X.matrix_to_h5ad.h5ad", sampleXProduct...),
		},
		"Sample_Y.anndatar_convert": {
			"h5ad": expectedBind("Sample_Y.matrix_to_h5ad.h5ad", sampleYProduct...),
		},
		"cohort.h5ad_concat": {
			"h5ad_1": expectedBind("Sample_X.matrix_to_h5ad.h5ad", sampleXProduct...),
			"h5ad_2": expectedBind("Sample_Y.matrix_to_h5ad.h5ad", sampleYProduct...),
		},
	}
	fastQC := []struct {
		task string
		name string
	}{
		{task: "Sample_X.run_001.raw_fastqc_r1.fastqc", name: sampleXR1Name},
		{task: "Sample_X.run_001.raw_fastqc_r2.fastqc", name: sampleXR2Name},
		{task: "Sample_Y.run_001.raw_fastqc_r1.fastqc", name: sampleYL1R1Name},
		{task: "Sample_Y.run_001.raw_fastqc_r2.fastqc", name: sampleYL1R2Name},
		{task: "Sample_Y.run_002.raw_fastqc_r1.fastqc", name: sampleYL2R1Name},
		{task: "Sample_Y.run_002.raw_fastqc_r2.fastqc", name: sampleYL2R2Name},
	}
	for _, contract := range fastQC {
		expected[contract.task] = map[string]expectedBoundInput{"reads": expectedBind("", contract.name)}
	}
	expected["multiqc"] = map[string]expectedBoundInput{
		"report_0":  expectedBind("Sample_X.run_001.raw_fastqc_r1.fastqc.html", sampleXR1Name),
		"report_1":  expectedBind("Sample_X.run_001.raw_fastqc_r1.fastqc.zip", sampleXR1Name),
		"report_2":  expectedBind("Sample_X.run_001.raw_fastqc_r2.fastqc.html", sampleXR2Name),
		"report_3":  expectedBind("Sample_X.run_001.raw_fastqc_r2.fastqc.zip", sampleXR2Name),
		"report_4":  expectedBind("Sample_X.simpleaf_quant.quant", sampleXProduct...),
		"report_5":  expectedBind("Sample_X.qcatch.report", sampleXProduct...),
		"report_6":  expectedBind("Sample_X.qcatch.metrics", sampleXProduct...),
		"report_7":  expectedBind("Sample_Y.run_001.raw_fastqc_r1.fastqc.html", sampleYL1R1Name),
		"report_8":  expectedBind("Sample_Y.run_001.raw_fastqc_r1.fastqc.zip", sampleYL1R1Name),
		"report_9":  expectedBind("Sample_Y.run_001.raw_fastqc_r2.fastqc.html", sampleYL1R2Name),
		"report_10": expectedBind("Sample_Y.run_001.raw_fastqc_r2.fastqc.zip", sampleYL1R2Name),
		"report_11": expectedBind("Sample_Y.run_002.raw_fastqc_r1.fastqc.html", sampleYL2R1Name),
		"report_12": expectedBind("Sample_Y.run_002.raw_fastqc_r1.fastqc.zip", sampleYL2R1Name),
		"report_13": expectedBind("Sample_Y.run_002.raw_fastqc_r2.fastqc.html", sampleYL2R2Name),
		"report_14": expectedBind("Sample_Y.run_002.raw_fastqc_r2.fastqc.zip", sampleYL2R2Name),
		"report_15": expectedBind("Sample_Y.simpleaf_quant.quant", sampleYProduct...),
		"report_16": expectedBind("Sample_Y.qcatch.report", sampleYProduct...),
		"report_17": expectedBind("Sample_Y.qcatch.metrics", sampleYProduct...),
	}
	return expected
}

func expectedBind(producer string, officialNames ...string) expectedBoundInput {
	producers := []string(nil)
	if producer != "" {
		producers = []string{producer}
	}
	return expectedBoundInput{Producers: producers, OfficialNames: append([]string(nil), officialNames...)}
}

func expectedOfficialInputs(names []string) ([]OfficialInput, error) {
	byName := make(map[string]OfficialInput, len(officialInputOracle))
	for _, input := range officialInputOracle {
		byName[input.Name] = input
	}
	expected := make(map[string]OfficialInput, len(names))
	for _, name := range names {
		input, ok := byName[name]
		if !ok || expected[input.Path].Name != "" {
			return nil, fmt.Errorf("invalid or duplicate frozen official identity %q", name)
		}
		expected[input.Path] = input
	}
	return sortedOfficialInputs(expected), nil
}

func hasIOName(values []pc.IO, name string) bool {
	for _, value := range values {
		if value.Name == name {
			return true
		}
	}
	return false
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
