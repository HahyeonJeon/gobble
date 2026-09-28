from pathlib import Path
import subprocess,json
root=Path.cwd(); stage=root/'docs/desktop-workspace/stages/p3-run-preparation'
base='sha256:25fee86458e0c8df45293cfd6dcaa992256d3da2cfcdbef62e3c992bc2ac71f1'
def run(*a): return subprocess.check_output(a,text=True).strip()
cid=run('docker','create','--platform','linux/amd64','--network','none','--user','root','--workdir','/opt/gobble','--entrypoint','/bin/sh',base,'-c','GOMAXPROCS=2 CGO_ENABLED=0 go build -buildvcs=false -o /usr/local/bin/gobble ./cmd/gobble')
try:
 for src,dst in [('preparation.go','/opt/gobble/preparation.go'),('internal/preparation','/opt/gobble/internal/preparation'),('internal/engine/prepared.go','/opt/gobble/internal/engine/prepared.go'),('cmd/gobble/driver.go','/opt/gobble/cmd/gobble/driver.go'),('cmd/gobble/parse.go','/opt/gobble/cmd/gobble/parse.go'),('cmd/gobble/main.go','/opt/gobble/cmd/gobble/main.go'),('cmd/gobble/help.go','/opt/gobble/cmd/gobble/help.go'),('cmd/gobble/prepared_review.go','/opt/gobble/cmd/gobble/prepared_review.go')]:
  subprocess.run(['docker','cp',str(root/src),cid+':'+dst],check=True)
 subprocess.run(['docker','start','-a',cid],check=True)
 state=json.loads(run('docker','inspect',cid))[0]['State']
 if state['ExitCode']!=0: raise RuntimeError(state)
 image=run('docker','commit','--change','USER 10001:10001','--change','ENTRYPOINT ["/usr/local/bin/gobble"]','--change','LABEL io.gobble.preparation.scope=single-end-trim-fastqc-v1',cid,'gobble-p3-preparation:local')
 (stage/'engine-image.json').write_text(json.dumps({'base':base,'imageId':image,'scope':'single-end-trim-fastqc-v1'},indent=2)+'\n')
 print(image)
finally: subprocess.run(['docker','rm',cid],check=True)
