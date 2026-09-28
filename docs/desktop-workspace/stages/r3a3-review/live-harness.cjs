const {_electron:electron} = require('/Users/hahyeon/.codex/.chatgpt-projects/g-p-6a9bae3a3ef481919aec253b9710060a/gobble/app/node_modules/playwright-core');
const {readFileSync}=require('node:fs');
const readline=require('node:readline');
const root='/Users/hahyeon/.codex/.chatgpt-projects/g-p-6a9bae3a3ef481919aec253b9710060a/gobble';
const profile='/Users/hahyeon/Library/Application Support/Gobble-stage4-review';
(async()=>{
 const env=Object.fromEntries(Object.entries(process.env).filter(([k,v])=>v!==undefined&&!['ELECTRON_RUN_AS_NODE','ELECTRON_RENDERER_URL','GOBBLE_CODEX_EXECUTABLE'].includes(k)));
 const application=await electron.launch({args:[root+'/app/desktop','--gobble-profile='+profile],env: {...env, ...JSON.parse(readFileSync(root+'/docs/desktop-workspace/stages/r3a3-review/live-fixture.json','utf8')).environment}});
 const page=await application.firstWindow();
 page.setDefaultTimeout(15000);
 await page.locator('.app-status').filter({hasText:'Workspace ready'}).waitFor();
 await page.getByRole('button',{name:'Account',exact:true}).click();
 await page.getByRole('button',{name:'Refresh connection'}).click();
 await page.getByText('ChatGPT connected',{exact:true}).waitFor();
 await page.getByRole('button',{name:'Close Codex account'}).click();
 await application.evaluate(({app, BrowserWindow}) => { app.focus({steal:true}); const window=BrowserWindow.getAllWindows()[0]; window.show(); window.focus(); });
 console.log('REVIEW_RESTORED_AND_CONNECTED');
 let queue=Promise.resolve();
 readline.createInterface({input:process.stdin}).on('line',line=>{queue=queue.then(async()=>{
  try{
   const cmd=JSON.parse(line);
   if(cmd.action==='close'){await application.close();process.exit(0);}
   const code=cmd.path?readFileSync(cmd.path,'utf8'):cmd.code;
   const fn=new Function('page','application','return (async()=>{'+code+'})()');
   console.log(JSON.stringify({result:await fn(page,application)}));
  }catch(error){console.log(JSON.stringify({error:error.message}));}
 });});
})().catch(error=>{console.error(error.message);process.exitCode=1;});
