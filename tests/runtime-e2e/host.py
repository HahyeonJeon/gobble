"""A host command path containing Docker, with no host Go."""
import os
from pathlib import Path
import shutil
import tempfile

env = os.environ.copy()
# Keep Docker Desktop's discovered CLI, but exclude host Go on every platform.
# A symlink preserves Docker's CLI/plugin behavior on Mac; Windows needs a copy.
docker = shutil.which("docker")
if not docker:
    for directory in (Path.home() / ".docker/bin", Path("/usr/local/bin"),
                      Path("/Applications/Docker.app/Contents/Resources/bin")):
        candidate = directory / "docker"
        if candidate.is_file():
            docker = str(candidate)
            break
if not docker:
    raise RuntimeError("Install and start Docker before running the acceptance test")
host_bin = tempfile.TemporaryDirectory(prefix="gobble-host-bin-")
docker_target = Path(host_bin.name) / ("docker.exe" if os.name == "nt" else "docker")
if os.name == "nt":
    shutil.copy2(docker, docker_target)
    env["PATH"] = host_bin.name + os.pathsep + os.path.join(os.environ["SystemRoot"], "System32")
else:
    docker_target.symlink_to(Path(docker).resolve())
    env["PATH"] = host_bin.name
assert shutil.which("go", path=env["PATH"]) is None
