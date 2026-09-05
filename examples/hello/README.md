# Hello Gobble

Count two sequences in a small FASTA file. This example uses local `sh` and
`awk` on Linux/amd64. It needs no Docker, samplesheet, reference genome, or data
download. It demonstrates the engine lifecycle, not an analysis pipeline.

From the repository root, with Go 1.26+ and Git:

```sh
go build -o ./bin/gobble ./cmd/gobble
mkdir -p ./runs/hello/inputs
cp examples/hello/sequences.fasta ./runs/hello/inputs/sequences.fasta

./bin/gobble validate ./examples/hello
./bin/gobble plan ./examples/hello
./bin/gobble run ./examples/hello --workspace ./runs/hello
cat ./runs/hello/results/sequence-count.txt
./bin/gobble inspect run --workspace ./runs/hello
./bin/gobble release --workspace ./runs/hello
```

The output is `2`. Run this in a new workspace. To repeat with another directory,
stage the same input under its `inputs/` directory and change `--workspace`.

For a concrete recovery example, remove the result after Release and resume:

```sh
rm ./runs/hello/results/sequence-count.txt
./bin/gobble resume ./examples/hello --workspace ./runs/hello
cat ./runs/hello/results/sequence-count.txt
./bin/gobble inspect instances --workspace ./runs/hello
./bin/gobble release --workspace ./runs/hello
```

The missing output is recreated in a new attempt. Completed compatible work can
be reused; this command does not resume execution from inside an interrupted
tool. Keep the same CLI binary and checkout for the exercise.

To distribute the example to an operator without Go:

Use a clean, committed checkout when packing; the current identity contract
rejects dirty Gobble source.

```sh
./bin/gobble pack ./examples/hello --output ./bin/hello-runner
```

Stage the input in a new workspace. Run `./bin/hello-runner run --workspace DIR`
without a package operand. The runner still needs the local commands used by
the pipeline. [Operations](../../docs/operations.md) covers the full lifecycle.
