"""Build a distinct local qualification engine; never replace a prior stage image."""
from pathlib import Path
import hashlib
import json
import subprocess
import tempfile

root = Path.cwd()
stage = root / 'docs/desktop-workspace/stages/report-output-evidence'
base = 'sha256:5d7b7247b407844149bc80173e8da2bf656077b827dd3f2ae9149ee490258e76'

def run(*args):
    return subprocess.check_output(args, text=True).strip()

paths = set(root.glob('*.go')) | {root / 'go.mod', root / 'go.sum'}
for directory in ('cmd', 'internal', 'assets', 'distribution', 'monitor'):
    paths.update(p for p in (root / directory).rglob('*') if p.is_file())
inventory = {str(p.relative_to(root)): hashlib.sha256(p.read_bytes()).hexdigest() for p in sorted(paths)}
with tempfile.TemporaryDirectory(prefix='gobble-report-build-') as directory:
    import tarfile
    archive = Path(directory) / 'source.tar'
    with tarfile.open(archive, 'w') as tar:
        for p in sorted(paths):
            tar.add(p, arcname=str(p.relative_to(root)))
    command = 'rm -rf /opt/gobble && mkdir /opt/gobble && tar -xf /tmp/source.tar -C /opt/gobble && cd /opt/gobble && GOMAXPROCS=2 CGO_ENABLED=0 GOPROXY=off GOTOOLCHAIN=local go build -buildvcs=false -o /usr/local/bin/gobble ./cmd/gobble && rm /tmp/source.tar'
    container = run('docker', 'create', '--platform', 'linux/amd64', '--network', 'none', '--user', 'root', '--entrypoint', '/bin/sh', base, '-c', command)
    try:
        subprocess.run(['docker', 'cp', str(archive), container + ':/tmp/source.tar'], check=True)
        subprocess.run(['docker', 'start', '-a', container], check=True)
        state = json.loads(run('docker', 'inspect', container))[0]['State']
        if state['ExitCode'] != 0:
            raise RuntimeError(state)
        image = run('docker', 'commit', '--change', 'USER 10001:10001', '--change', 'ENTRYPOINT ["/usr/local/bin/gobble"]', '--change', 'LABEL io.gobble.continuation.scope=single-end-trim-fastqc-v1', '--change', 'LABEL io.gobble.output-evidence.version=1', container, 'gobble-report-output:local')
        (stage / 'engine-image.json').write_text(json.dumps({'base': base, 'imageId': image, 'sourceSHA256': inventory}, indent=2) + '\n')
        print(image)
    finally:
        subprocess.run(['docker', 'rm', container], check=True)
