from pathlib import Path
import subprocess,json,hashlib,tempfile
root=Path.cwd(); stage=root/'docs/desktop-workspace/stages/p4-run-control'
base='sha256:9145e5f6ecc26e8c8d61b7df6855d15a54371fc5f517a38029fdee4b5af5b19f'
def run(*args): return subprocess.check_output(args,text=True).strip()
before=json.loads((stage/'before.json').read_text()); paths=set(root.glob('*.go'))
for folder in ['internal/engine','internal/preparation','cmd/gobble']:
 paths.update((root/folder).rglob('*.go'))
changed=[p for p in sorted(paths) if not p.name.endswith('_test.go') and before.get(str(p.relative_to(root)))!=hashlib.sha256(p.read_bytes()).hexdigest()]
cid=run('docker','create','--platform','linux/amd64','--network','none','--user','root','--workdir','/opt/gobble','--entrypoint','/bin/sh',base,'-c','GOMAXPROCS=2 CGO_ENABLED=0 go build -buildvcs=false -o /usr/local/bin/gobble ./cmd/gobble')
try:
 # Reuse the already qualified local runtime's Linux Docker client. No download
 # or dependency installation occurs when a User confirms Start.
 client_source=run('docker','image','inspect','--format','{{.Id}}','sha256:5f3d800dd82d')
 client=run('docker','create','--entrypoint','/bin/true',client_source)
 try:
  with tempfile.TemporaryDirectory(prefix='gobble-p4-client-') as tmp:
   subprocess.run(['docker','cp',client+':/usr/local/bin/docker',tmp+'/docker'],check=True)
   subprocess.run(['docker','cp',tmp+'/docker',cid+':/usr/local/bin/docker'],check=True)
 finally: subprocess.run(['docker','rm',client],check=True)
 for p in changed: subprocess.run(['docker','cp',str(p),cid+':/opt/gobble/'+str(p.relative_to(root))],check=True)
 subprocess.run(['docker','start','-a',cid],check=True)
 state=json.loads(run('docker','inspect',cid))[0]['State']
 if state['ExitCode']!=0:raise RuntimeError(state)
 image=run('docker','commit','--change','USER 10001:10001','--change','ENTRYPOINT ["/usr/local/bin/gobble"]','--change','LABEL io.gobble.launch.scope=single-end-trim-fastqc-v1',cid,'gobble-p4-launch:local')
 (stage/'engine-image.json').write_text(json.dumps({'base':base,'dockerClientSource':client_source,'imageId':image,'sourceFiles':[str(p.relative_to(root)) for p in changed]},indent=2)+'\n');print(image)
finally:subprocess.run(['docker','rm',cid],check=True)
