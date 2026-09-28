"""Own one disposable, token-authenticated TLS Jupyter server for qualification."""
import json
import os
from pathlib import Path
import secrets
import signal
import socket
import subprocess
import sys
import tempfile
import time
import requests

scratch = Path(tempfile.mkdtemp(prefix='gobble-r4b-'))
server = None
try:
    os.chmod(scratch, 0o700)
    for name in ('project', 'config', 'data', 'runtime', 'ipython'):
        (scratch / name).mkdir(mode=0o700)
    cert, key = scratch / 'cert.pem', scratch / 'key.pem'
    subprocess.run(['openssl', 'req', '-x509', '-newkey', 'rsa:2048', '-nodes', '-keyout', str(key), '-out', str(cert), '-days', '1', '-subj', '/CN=127.0.0.1', '-addext', 'subjectAltName=IP:127.0.0.1'], check=True, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    os.chmod(key, 0o600)
    with socket.socket() as sock:
        sock.bind(('127.0.0.1', 0))
        port = sock.getsockname()[1]
    token = secrets.token_hex(32)
    kernelspec = scratch / 'data/kernels/r4b'
    kernelspec.mkdir(parents=True)
    (kernelspec / 'kernel.json').write_text(json.dumps({'argv': [sys.executable, '-m', 'ipykernel_launcher', '-f', '{connection_file}'], 'display_name': 'R4b synthetic Python', 'language': 'python'}))
    notebook = {'nbformat': 4, 'nbformat_minor': 5, 'metadata': {'kernelspec': {'name': 'r4b', 'display_name': 'R4b synthetic Python', 'language': 'python'}}, 'cells': [
        {'id': 'cell-counts', 'cell_type': 'code', 'metadata': {}, 'source': 'label = "🧬 한국어"\ncounts = [12, 24, 18]\nprint(sum(counts))', 'outputs': [], 'execution_count': None},
        {'id': 'cell-filter', 'cell_type': 'code', 'metadata': {}, 'source': 'threshold = 20\nselected = [value for value in counts if value > threshold]', 'outputs': [], 'execution_count': None}]}
    for name in ('study.ipynb', 'protocol.ipynb', 'conflict.ipynb'):
        (scratch / 'project' / name).write_text(json.dumps(notebook))
    config = scratch / 'config/jupyter_server_config.py'
    config.write_text('\n'.join([
        'c = get_config()', f'c.ServerApp.ip = "127.0.0.1"', f'c.ServerApp.port = {port}', 'c.ServerApp.port_retries = 0',
        f'c.ServerApp.root_dir = {str(scratch / "project")!r}', f'c.ServerApp.certfile = {str(cert)!r}', f'c.ServerApp.keyfile = {str(key)!r}',
        f'c.IdentityProvider.token = {token!r}', 'c.ServerApp.open_browser = False', 'c.ServerApp.terminals_enabled = False',
        'c.KernelSpecManager.allowed_kernelspecs = {"r4b"}', 'c.LabApp.expose_app_in_browser = True', 'c.LabApp.core_mode = True'
    ]))
    os.chmod(config, 0o600)
    env = dict(os.environ)
    for variable, name in [('JUPYTER_CONFIG_DIR','config'), ('JUPYTER_DATA_DIR','data'), ('JUPYTER_RUNTIME_DIR','runtime'), ('IPYTHONDIR','ipython')]:
        env[variable] = str(scratch / name)
    env['JUPYTER_NO_CONFIG'] = '0'
    log = open(scratch / 'server-private.log', 'w')
    server = subprocess.Popen([sys.executable, '-m', 'jupyterlab', '--config', str(config)], cwd=scratch / 'project', env=env, stdout=log, stderr=log)
    origin = f'https://127.0.0.1:{port}'
    for _ in range(120):
        if server.poll() is not None:
            raise RuntimeError('Isolated server failed; inspect private scratch log locally')
        try:
            if requests.get(origin + '/api/status', headers={'Authorization': 'token ' + token}, verify=str(cert), timeout=1).status_code == 200:
                break
        except requests.RequestException:
            pass
        time.sleep(0.25)
    else:
        raise RuntimeError('Isolated server readiness timed out')
    private = scratch / 'connection.json'
    private.write_text(json.dumps({'origin': origin, 'token': token, 'cert': str(cert), 'project': str(scratch / 'project'), 'scratch': str(scratch)}))
    os.chmod(private, 0o600)
    print(str(private), flush=True)
    # The orchestrator owns this pipe. EOF or a line releases the server.
    sys.stdin.readline()
finally:
    if server is not None and server.poll() is None:
        server.terminate()
        try:
            server.wait(timeout=15)
        except subprocess.TimeoutExpired:
            server.kill()
            server.wait(timeout=5)
    import shutil
    shutil.rmtree(scratch)
