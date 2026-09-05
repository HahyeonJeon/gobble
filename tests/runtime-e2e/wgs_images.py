"""Verify corrected WGS manifest pins and declared tool versions with Docker."""
import json
from pathlib import Path
import subprocess
from host import env

# Assertions are independent of a tag merely existing. Keep the existing
# benchmark tool versions when repairing manifest identities.
CHECKS = (
    ("bwa-index", "bwa", (), "0.7.19", (0, 1)),
    ("fastp", "fastp", ("--version",), "1.1.0", (0,)),
    ("samtools-sort", "samtools", ("--version",), "1.24", (0,)),
    ("gatk4-markduplicates", "gatk", ("--version",), "4.6.2.0", (0,)),
    ("gatk4-haplotypecaller", "gatk", ("--version",), "4.6.2.0", (0,)),
    ("mosdepth", "mosdepth", ("--version",), "0.3.14", (0,)),
    ("bcftools-sort", "bcftools", ("--version",), "1.23.1", (0,)),
    ("multiqc", "multiqc", ("--version",), "1.35", (0,)),
)


def verify(project, artifacts=None):
    entries = json.loads((Path(project) / "fixture-manifest.json").read_text())["images"]
    images = {entry["module"]: entry["reference"] + "@" + entry["digest"] for entry in entries}
    evidence = []
    try:
        for module, tool, args, expected, exits in CHECKS:
            image = images[module]
            print(f"Verify {module}: {image}", flush=True)
            subprocess.run(["docker", "pull", "--platform", "linux/amd64", image], env=env, check=True, timeout=600)
            platform = subprocess.check_output(["docker", "image", "inspect", "--format", "{{.Os}}/{{.Architecture}}", image], env=env, text=True).strip()
            assert platform == "linux/amd64", (image, platform)
            result = subprocess.run(["docker", "run", "--rm", "--platform", "linux/amd64", "--network=none",
                                     "--entrypoint", tool, image, *args], env=env, text=True, capture_output=True, timeout=120)
            output = result.stdout + result.stderr
            evidence.append(f"{image}\n{platform}\n{output}\n")
            if result.returncode not in exits or expected not in output:
                # Public image metadata helps distinguish packaging from a
                # tool failure without guessing executable locations.
                config = subprocess.check_output(["docker", "image", "inspect", "--format", "{{json .Config}}", image], env=env, text=True)
                evidence.append(config)
                raise AssertionError((image, result.returncode, output, config))
    finally:
        if artifacts:
            (artifacts / "image-versions.txt").write_text("\n".join(evidence))
