/** Visual walkthrough only. Run from app/ after npm run build. Synthetic, temporary Project. */
import { rm } from 'node:fs/promises';
import { resolve } from 'node:path';
import { runtimeFixture } from '../../../../app/desktop/tests/electron/runtime-fixture';
import { launch, chooseProject, readyView, openAgentRoster, captureWindow } from '../../../../app/desktop/tests/electron/support';
async function main() {
  const fixture=await runtimeFixture();const {application,page}=await launch(fixture.profile,fixture.environment);
  try {
    await chooseProject(application,page,fixture.root);
    await application.evaluate(({shell})=>{shell.openExternal=async()=>{};});
    await page.getByRole('button',{name:'Account',exact:true}).click();
    await page.getByRole('button',{name:'Sign in with ChatGPT'}).click();
    await page.getByText('ChatGPT connected',{exact:true}).waitFor();
    await page.getByRole('button',{name:'Close Codex account'}).click();
    await openAgentRoster(page);await page.getByRole('button',{name:'Add agent',exact:true}).click();
    const dialog=page.getByRole('dialog',{name:'Add agent',exact:true});await dialog.getByLabel('Name',{exact:true}).fill('Researcher');
    await dialog.getByRole('button',{name:'Add agent',exact:true}).click();await dialog.waitFor({state:'hidden'});
    await page.getByRole('button',{name:'Attach',exact:true}).click();await page.getByRole('button',{name:'existing',exact:true}).click();
    const primary=await readyView(page);await primary.getByRole('button',{name:'Dependencies',exact:true}).click();await readyView(page);
    await primary.getByRole('button',{name:'Task group align',exact:true}).click();
    await primary.getByRole('button',{name:'Discuss group',exact:true}).click();
    await page.getByRole('combobox',{name:'Conversation recipient'}).selectOption({label:'To Researcher'});
    await page.getByText('1 attachment · Ready for Researcher',{exact:true}).waitFor();
    await page.getByRole('textbox',{name:'Message draft'}).fill('Review this task group and its observed attempts.');
    await primary.locator('.dependency-viewport').hover();await page.mouse.wheel(0,-800);
    await primary.locator('.dependency-view').hover();await page.mouse.wheel(0,-800);
    await page.getByRole('textbox',{name:'Message draft'}).focus();
    await page.evaluate(()=>new Promise<void>(done=>requestAnimationFrame(()=>requestAnimationFrame(()=>done()))));
    await captureWindow(application,resolve('../docs/desktop-workspace/stages/r3b2-review/dependency-overview.png'));
  } finally {await application.close();await rm(fixture.directory,{recursive:true,force:true});}
}
main().catch(error=>{console.error(error);process.exitCode=1;});
