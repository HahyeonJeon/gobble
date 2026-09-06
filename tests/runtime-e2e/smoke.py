"""Actual Docker launcher journey. Missing Docker is a failure, never a skip.

Run with: python3 tests/runtime-e2e/smoke.py /absolute/path/to/launcher
GOBBLE_RUNTIME_IMAGE must name the already-built runtime. No host Go is used.
"""
import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
import time

from host import env
from transport import Transport
transport = Transport(sys.argv[1], env)

if not env.get("GOBBLE_RUNTIME_IMAGE"):
    raise RuntimeError("GOBBLE_RUNTIME_IMAGE is required")


def command(cwd, *args):
    result = subprocess.run(transport.argv(cwd, *args), cwd=cwd, env=env, text=True,
                            stdout=subprocess.PIPE, stderr=subprocess.PIPE, timeout=180)
    if result.returncode:
        raise RuntimeError(f"{args}: {result.returncode}\n{result.stderr}")
    return result.stdout


def wait_for(predicate, process=None):
    deadline = time.monotonic() + 120
    while time.monotonic() < deadline:
        value = predicate()
        if value:
            return value
        if process and process.poll() is not None:
            raise RuntimeError(f"controller exited early: {process.returncode}")
        time.sleep(0.2)
    raise TimeoutError("runtime scenario did not reach its expected boundary")


with tempfile.TemporaryDirectory(prefix="Gobble 한글 space ") as temporary:
    root = Path(temporary)
    command(root, "init", "demo")
    project = root / "demo"
    lock = json.loads((project / ".gobble-runtime.json").read_text())
    assert lock["image"].startswith("sha256:") and lock["daemon"]
    doctor = json.loads(command(project, "doctor"))
    assert "sibling container" in doctor["checks"]["workspace"]
    command(project, "plan", ".")
    command(project, "run", ".", "--workspace", "runs/hello")
    assert (project / "runs/hello/results/sequence-count.txt").read_text().strip() == "2"
    command(project, "resume", ".", "--workspace", "runs/hello")

    source = project / "pipeline.go"
    original = source.read_text()
    # Use the already-prepared runtime as a sibling task image: this gate does
    # not depend on an unrelated registry pull succeeding.
    docker_pipeline = original.replace('Name: "count-sequences",',
        'Name: "count-sequences",\n        Image: ' + json.dumps(lock["image"]) + ',')
    long_pipeline = docker_pipeline.replace("awk '/^>/", "echo gobble-live-log; sleep 60; awk '/^>/")
    project_id = hashlib.sha256(str(project.resolve()).encode()).hexdigest()

    for scenario in ("stop", "controller-death"):
        workspace = project / "runs" / scenario
        (workspace / "inputs").mkdir(parents=True)
        shutil.copy(project / "runs/hello/inputs/sequences.fasta", workspace / "inputs/sequences.fasta")
        source.write_text(long_pipeline)
        with tempfile.TemporaryFile() as output:
            process = subprocess.Popen(transport.argv(project, "run", ".", "--workspace", str(workspace)),
                                       cwd=project, env=env, stdout=output, stderr=output)
            try:
                wait_for(lambda: any("gobble-live-log" in p.read_text()
                                     for p in workspace.glob(".gobble/tasks/**/stdout")), process)
                if scenario == "stop":
                    result = json.loads(command(project, "stop", "--workspace", str(workspace)))
                    assert result["status"] == "settled", result
                    assert json.loads(command(project, "stop", "--workspace", str(workspace)))["status"] == "settled"
                else:
                    ids = subprocess.check_output(["docker", "ps", "--quiet",
                        "--filter", transport.controller_filter(project)], env=env, text=True).split()
                    assert len(ids) == 1, ids
                    subprocess.run(["docker", "kill", "--signal=KILL", ids[0]], env=env, check=True)
                assert process.wait(timeout=60) != 0
                source.write_text(docker_pipeline)
                command(project, "resume", ".", "--workspace", str(workspace))
                assert (workspace / "results/sequence-count.txt").read_text().strip() == "2"
                tasks = [json.loads(line) for line in command(project, "inspect", "instances", "--workspace", str(workspace)).splitlines()]
                assert len(tasks) == 1 and tasks[0]["status"] == "succeeded" and tasks[0]["attempt"] == 2, tasks
            finally:
                if process.poll() is None:
                    command(project, "stop", "--workspace", str(workspace))
                    process.wait(timeout=60)
                output.seek(0)
                print(output.read().decode(errors="replace"))
    if transport.compose:
        # A returned Docker client is not the owner. The detached controller
        # remains live, rejects another owner, and is stopped via shared state.
        workspace = project / "runs/detached"
        (workspace / "inputs").mkdir(parents=True)
        shutil.copy(project / "runs/hello/inputs/sequences.fasta", workspace / "inputs/sequences.fasta")
        source.write_text(long_pipeline)
        container = subprocess.check_output(transport.argv(project, "run", ".", "--workspace", "runs/detached", detached=True),
                                            cwd=project, env=env, text=True, timeout=60).strip()
        try:
            wait_for(lambda: any("gobble-live-log" in p.read_text() for p in workspace.glob(".gobble/tasks/**/stdout")))
            assert subprocess.check_output(["docker", "inspect", "--format", "{{.State.Running}}", container], env=env, text=True).strip() == "true"
            duplicate = subprocess.run(transport.argv(project, "run", ".", "--workspace", "runs/detached"), cwd=project, env=env, capture_output=True, timeout=180)
            assert duplicate.returncode != 0, "A second owner was admitted"
            command(project, "inspect", "monitor", "--workspace", "runs/detached")
            assert json.loads(command(project, "stop", "--workspace", "runs/detached"))["status"] == "settled"
            subprocess.run(["docker", "wait", container], env=env, check=True, timeout=60)
            source.write_text(docker_pipeline)
            command(project, "resume", ".", "--workspace", "runs/detached")
            assert (workspace / "results/sequence-count.txt").read_text().strip() == "2"
        finally:
            subprocess.run(["docker", "rm", "-f", container], env=env, check=True)
        # Nested working directories use the actual mounted subdirectory.
        command(root, "doctor")
        nested = subprocess.run(["docker", "compose", "--project-name", transport.project_name(root),
            "run", "--rm", "-T", "--workdir", "/gobble/project/demo", "gobble", "doctor"],
            cwd=root, env=env, text=True, capture_output=True, timeout=180)
        assert nested.returncode == 0, nested.stderr
        # An existing lock must survive an incorrect image/daemon selection.
        lock_path = project / ".gobble-runtime.json"
        previous = lock_path.read_bytes()
        changed = json.loads(previous)
        changed["daemon"] = "different-daemon"
        lock_path.write_text(json.dumps(changed))
        try:
            bad = subprocess.run(transport.argv(project, "doctor"), cwd=project, env=env, text=True, capture_output=True, timeout=60)
            assert bad.returncode != 0 and "mismatch" in bad.stderr, bad
            assert json.loads(lock_path.read_text())["daemon"] == "different-daemon"
        finally:
            lock_path.write_bytes(previous)
    print("PASS: init, doctor, sibling mounts, live logs, Stop, controller death, Resume; Compose adds detach, duplicate owner, nested paths, lock mismatch")
