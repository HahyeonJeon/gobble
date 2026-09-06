"""Exercise either the common Compose entry point or the compatibility launcher."""
import hashlib
from pathlib import Path

class Transport:
    def __init__(self, entry, env):
        self.compose = entry == "compose"
        self.launcher = str(Path(entry).resolve())
        self.env = env

    def project_name(self, cwd):
        return "gobble-test-" + hashlib.sha256(str(Path(cwd).resolve()).encode()).hexdigest()[:16]

    def argv(self, cwd, *args, detached=False):
        if not self.compose:
            assert not detached
            return [self.launcher, *args]
        config = Path(cwd) / "compose.yaml"
        if not config.exists():
            template = Path(__file__).resolve().parents[2] / "distribution/runtime/compose.yaml"
            config.write_text(template.read_text().replace("ghcr.io/hahyeonjeon/gobble:develop", self.env["GOBBLE_RUNTIME_IMAGE"]))
        prefix = ["docker", "compose", "--project-name", self.project_name(cwd), "run"]
        prefix += ["-d"] if detached else ["--rm", "-T"]
        return [*prefix, "gobble", *args]

    def controller_filter(self, cwd):
        if self.compose:
            return "label=com.docker.compose.project=" + self.project_name(cwd)
        return "label=io.gobble.project=" + hashlib.sha256(str(Path(cwd).resolve()).encode()).hexdigest()
