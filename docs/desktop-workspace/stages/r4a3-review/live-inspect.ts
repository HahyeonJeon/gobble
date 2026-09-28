// Resume inspection of the already running isolated live trial after its input pipe closed.
import {chromium} from '@playwright/test';
import {readFile,writeFile} from 'node:fs/promises';
import {resolve,join} from 'node:path';
async function main(){
const base='/var/folders/xr/75hrwk3x4bxcbr7f3lrb5d480000gn/T/gobble-r4a3-live-5oWujR';
const port=(await readFile(join(base,'profile/DevToolsActivePort'),'utf8')).split('\n')[0];
const browser=await chromium.connectOverCDP('http://127.0.0.1:'+port);
const page=browser.contexts()[0]!.pages().find(p=>p.url().startsWith('app://gobble/'))!;
const output=resolve('../docs/desktop-workspace/stages/r4a3-review');
const doc=()=>page.evaluate(async()=>{const p=await window.gobble.projects.list();if(!p.ok)throw new Error(p.error.message);const d=await window.gobble.workspace.read({projectId:p.value[0]!.projectId});if(!d.ok)throw new Error(d.error.message);return d.value;});
const d=await doc();
if(process.argv[2]==='show'){
  await page.locator('.shared-reference-event').getByRole('button',{name:'Show in view',exact:true}).click();
  await page.getByRole('img',{name:'Exact Notebook reference region'}).waitFor();
  await page.screenshot({path:join(output,'live-agent-show.png')});
  await page.getByRole('button',{name:'Return to my view'}).click();
  await page.getByRole('img',{name:'Saved Notebook output'}).waitFor();
  await page.screenshot({path:join(output,'live-agent-return.png')});
  const after=await doc();
  const before=JSON.parse(await readFile(join(output,'live-request.json'),'utf8')).before;
  const state={surfaces:after.workspace.surfaces,selections:after.selections,chat:after.chat};
  await writeFile(join(output,'live-review.json'),JSON.stringify({showImageVisible:true,returnedReader:true,userPreserved:JSON.stringify(before)===JSON.stringify(state)},null,2));
}
await writeFile(join(output,'live-delivery.json'),JSON.stringify(d,null,2));
await page.screenshot({path:join(output,'live-agent-final.png')});
console.log(JSON.stringify({state:d.collaboration?.submissions.at(-1)?.state,model:d.collaboration?.submissions.at(-1)?.model,references:d.sharedReferences?.length,questions:d.workspace.decisions.filter(x=>x.kind==='question').length,messages:await page.locator('.agent-message').allTextContents()}));
if(process.argv[2]==='close')await page.evaluate(()=>window.close());
process.exit(0);

}
main().catch(error=>{console.error(String(error));process.exit(1);});
