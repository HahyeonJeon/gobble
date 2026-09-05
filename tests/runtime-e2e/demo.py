"""Run an installed, unmodified assay on its official fixture using real tools.

Usage: python3 tests/runtime-e2e/demo.py /absolute/launcher rnaseq [ARTIFACT_DIR]
Requires a built GOBBLE_RUNTIME_IMAGE and a local Docker engine. No host Go.
Unlike the hermetic suite, this downloads upstream data and pinned tool images.
"""
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import time
from host import env

launcher = str(Path(sys.argv[1]).resolve())
assay = sys.argv[2]
artifacts = Path(sys.argv[3]).resolve() if len(sys.argv) > 3 else None
if artifacts:
    artifacts.mkdir(parents=True, exist_ok=True)
root = Path(tempfile.mkdtemp(prefix="Gobble assay 한글 "))
project = root / assay
started = time.monotonic()
print(f"Assay: {assay}; project: {project}", flush=True)


def command(cwd, *args, timeout=300):
    result = subprocess.run([launcher, *args], cwd=cwd, env=env, text=True,
                            stdout=subprocess.PIPE, stderr=subprocess.PIPE, timeout=timeout)
    print(result.stderr, end="", flush=True)
    if result.returncode:
        raise RuntimeError(f"{args}: exit {result.returncode}\n{result.stdout}\n{result.stderr}")
    return result.stdout


try:
    prepared = json.loads(command(root, "demo", assay, assay, timeout=900))
    command(project, "doctor")
    command(project, "validate", ".")
    plan = command(project, "plan", ".")
    if artifacts:
        (artifacts / "plan.json").write_text(plan)
    command(project, "run", ".", "--workspace", "runs/demo", "--cap", "1", timeout=2400)
    for relative in prepared["expected_outputs"]:
        output = project / "runs/demo" / relative
        assert output.is_file() and output.stat().st_size > 0, f"Missing/empty output: {relative}"
    before = [json.loads(line) for line in command(project, "inspect", "instances", "--workspace", "runs/demo").splitlines()]
    assert before and all(row["status"] in ("succeeded", "skipped") for row in before), before
    command(project, "resume", ".", "--workspace", "runs/demo", "--cap", "1")
    after = [json.loads(line) for line in command(project, "inspect", "instances", "--workspace", "runs/demo").splitlines()]
    assert [(r["identity"], r["attempt"]) for r in before] == [(r["identity"], r["attempt"]) for r in after], "Unchanged completed work was rerun"
    print(f"PASS {assay}: {len(after)} tasks; outputs verified; Resume reused work; {time.monotonic()-started:.1f}s", flush=True)
finally:
    if project.is_dir():
        # A timeout must stop our run before leaving its workspace for diagnosis.
        subprocess.run([launcher, "stop", "--workspace", "runs/demo"], cwd=project, env=env, timeout=90)
        if artifacts:
            for view in ("run", "errors", "instances"):
                result = subprocess.run([launcher, "inspect", view, "--workspace", "runs/demo"], cwd=project, env=env,
                                        text=True, capture_output=True, timeout=90)
                (artifacts / (view + ".jsonl")).write_text(result.stdout + result.stderr)
            logs = artifacts / "task-logs"
            for source in (project / "runs/demo/.gobble/tasks").rglob("*"):
                if source.is_file() and source.name in ("stdout", "stderr"):
                    target = logs / source.relative_to(project / "runs/demo/.gobble/tasks")
                    target.parent.mkdir(parents=True, exist_ok=True)
                    target.write_bytes(source.read_bytes())
    print(f"Workspace retained for diagnosis: {project}", flush=True)
