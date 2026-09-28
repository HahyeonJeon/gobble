"""Synthetic REST/kernel failure probes; reports contain no connection credentials."""
import datetime
import json
from pathlib import Path
import sys
import time
import uuid
import requests
import websocket

cfg = json.loads(Path(sys.argv[1]).read_text())
http = requests.Session()
http.headers['Authorization'] = 'token ' + cfg['token']
http.verify = cfg['cert']
results = {}
session_id = None

def api(method, path, **kwargs):
    r = http.request(method, cfg['origin'] + path, timeout=30, **kwargs)
    r.raise_for_status()
    return r

def connect(kernel):
    return websocket.create_connection(cfg['origin'].replace('https:', 'wss:') + f'/api/kernels/{kernel}/channels?session_id={uuid.uuid4()}', header={'Authorization': 'token ' + cfg['token']}, origin=cfg['origin'], sslopt={'ca_certs': cfg['cert']}, timeout=15)

def submit(ws, code):
    request_id = str(uuid.uuid4())
    ws.send(json.dumps({'header': {'msg_id': request_id, 'username': 'qualification', 'session': 'r4b-protocol', 'date': datetime.datetime.now(datetime.timezone.utc).isoformat(), 'msg_type': 'execute_request', 'version': '5.3'}, 'parent_header': {}, 'metadata': {}, 'channel': 'shell', 'content': {'code': code, 'silent': False, 'store_history': True, 'user_expressions': {}, 'allow_stdin': False, 'stop_on_error': True}, 'buffers': []}))
    return request_id

def collect(ws, request_id):
    messages = []
    deadline = time.monotonic() + 25
    reply = idle = False
    while time.monotonic() < deadline and not (reply and idle):
        msg = json.loads(ws.recv())
        if msg.get('parent_header', {}).get('msg_id') != request_id:
            continue
        messages.append(msg)
        reply |= msg['msg_type'] == 'execute_reply'
        idle |= msg['msg_type'] == 'status' and msg['content']['execution_state'] == 'idle'
    assert reply and idle, 'No correlated shell reply and idle'
    return messages

def streams(messages):
    return ''.join(m['content']['text'] for m in messages if m['msg_type'] == 'stream')

try:
    path = '/api/contents/conflict.ipynb'
    original = api('GET', path + '?hash=1').json()
    newer = json.loads(json.dumps(original['content']))
    newer['cells'][0]['source'] = 'external_version = 2'
    api('PUT', path, json={'type': 'notebook', 'format': 'json', 'content': newer})
    changed = api('GET', path + '?hash=1').json()
    assert changed['hash'] != original['hash']
    stale = api('PUT', path, headers={'If-Match': '"' + original['hash'] + '"'}, json={'type': 'notebook', 'format': 'json', 'content': original['content']})
    final = api('GET', path + '?hash=1').json()
    results['staleSave'] = {'status': stale.status_code, 'staleIfMatchOverwroteNewerContent': final['content']['cells'][0]['source'] == original['content']['cells'][0]['source'], 'preflightHashDetectsChange': changed['hash'] != original['hash']}
    assert results['staleSave']['staleIfMatchOverwroteNewerContent']
    api('DELETE', path)
    missing = http.get(cfg['origin'] + path, timeout=10)
    assert missing.status_code == 404
    recreated = api('PUT', path, json={'type': 'notebook', 'format': 'json', 'content': original['content']})
    results['missingFile'] = {'readStatus': missing.status_code, 'saveRecreatesWithoutExplicitCreateIntent': recreated.status_code == 201}
    assert results['missingFile']['saveRecreatesWithoutExplicitCreateIntent']
    session = api('POST', '/api/sessions', json={'path': 'protocol.ipynb', 'name': 'qualification', 'type': 'notebook', 'kernel': {'name': 'r4b'}}).json()
    session_id, kernel = session['id'], session['kernel']['id']
    ws = connect(kernel)
    first = collect(ws, submit(ws, "counter = 0\nfrom IPython.display import display\nview = display('first', display_id=True)\nview.update('second')"))
    displays = [m for m in first if m['msg_type'] in ('display_data', 'update_display_data')]
    assert [m['msg_type'] for m in displays] == ['display_data', 'update_display_data']
    assert displays[0]['content']['transient']['display_id'] == displays[1]['content']['transient']['display_id']
    epoch1 = first[-1]['header']['session']
    results['liveOutput'] = {'types': [m['msg_type'] for m in displays], 'sameDisplayId': True, 'allRepliesCorrelated': True}
    lost_request = submit(ws, 'import time\nprint("started", flush=True)\ntime.sleep(1.5)\ncounter += 1\nprint(counter)')
    while True:
        msg = json.loads(ws.recv())
        if msg.get('parent_header', {}).get('msg_id') == lost_request and msg['msg_type'] == 'stream':
            break
    ws.close()
    time.sleep(2)
    ws = connect(kernel)
    recovered = collect(ws, submit(ws, 'print(counter)'))
    assert streams(recovered).strip() == '1'
    results['disconnect'] = {'outcomeAtDisconnect': 'unknown', 'counterAfterReconnect': 1, 'replayedOriginalRequest': False, 'kernelStillRunning': api('GET', f'/api/kernels/{kernel}').status_code == 200}
    ws.close()
    restarted = api('POST', f'/api/kernels/{kernel}/restart').json()
    ws = connect(kernel)
    after = collect(ws, submit(ws, 'print("counter" in globals())'))
    epoch2 = after[-1]['header']['session']
    assert streams(after).strip() == 'False' and epoch1 != epoch2
    results['restart'] = {'kernelIdRetained': restarted['id'] == kernel, 'kernelMessageEpochChanged': epoch1 != epoch2, 'oldVariablesGone': True}
    ws.close()
    results['viewLifetime'] = {'sessionSurvivesAllProbeSocketsClosed': api('GET', f'/api/sessions/{session_id}').status_code == 200}
    print(json.dumps(results, indent=2))
finally:
    if session_id:
        api('DELETE', f'/api/sessions/{session_id}')
