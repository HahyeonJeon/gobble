# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: dependencies.spec.ts >> User selects a group, exact attempt and directed dependency; frozen evidence and navigation survive restart and source changes
- Location: desktop/tests/electron/dependencies.spec.ts:16:1

# Error details

```
Error: locator.check: Clicking the checkbox did not change its state
Call log:
  - waiting for getByRole('region', { name: 'Primary pane', exact: true }).getByRole('radio', { name: 'Instance align-template · Attempt 0', exact: true })
    - locator resolved to <input type="radio" name="_r_2_" aria-label="Instance align-template · Attempt 0"/>
  - attempting click action
    - waiting for element to be visible, enabled and stable
    - element is visible, enabled and stable
    - scrolling into view if needed
    - done scrolling
    - performing click action
    - click action done
    - waiting for scheduled navigations to finish
    - navigations have finished

```

# Test source

```ts
  1  | import { expect, test, type Page, type ElectronApplication } from '@playwright/test';
  2  | import { readFile, rm, writeFile } from 'node:fs/promises';
  3  | import { join } from 'node:path';
  4  | import { runtimeFixture } from './runtime-fixture';
  5  | import { launch, chooseProject, readyView, openAgentRoster, attachmentPreview, closeAttachmentPreview, captureWindow } from './support';
  6  | async function addResearcher(application:ElectronApplication,page:Page){
  7  |  await application.evaluate(({shell})=>{shell.openExternal=async()=>{};});
  8  |  await page.getByRole('button',{name:'Account',exact:true}).click();await page.getByRole('button',{name:'Sign in with ChatGPT'}).click();
  9  |  await expect(page.getByText('ChatGPT connected',{exact:true})).toBeVisible();await page.getByRole('button',{name:'Close Codex account'}).click();
  10 |  await openAgentRoster(page);await page.getByRole('button',{name:'Add agent',exact:true}).click();const dialog=page.getByRole('dialog',{name:'Add agent',exact:true});
  11 |  await dialog.getByLabel('Name',{exact:true}).fill('Researcher');await dialog.getByRole('button',{name:'Add agent',exact:true}).click();await expect(dialog).not.toBeVisible();
  12 | }
  13 | async function document(page:Page){return page.evaluate(async()=>{const projects=await window.gobble.projects.list();if(!projects.ok)throw new Error(projects.error.message);const state=await window.gobble.workspace.read({projectId:projects.value[0]!.projectId});if(!state.ok)throw new Error(state.error.message);return state.value;});}
  14 | async function paint(page:Page){await page.evaluate(()=>new Promise<void>(resolve=>requestAnimationFrame(()=>requestAnimationFrame(()=>resolve()))));}
  15 | 
  16 | test('User selects a group, exact attempt and directed dependency; frozen evidence and navigation survive restart and source changes',async({},info)=>{
  17 |  test.setTimeout(90_000);const fixture=await runtimeFixture();let {application,page}=await launch(fixture.profile,fixture.environment);
  18 |  try{
  19 |   await chooseProject(application,page,fixture.root);await addResearcher(application,page);
  20 |   await page.getByRole('button',{name:'Attach',exact:true}).click();await page.getByRole('button',{name:'existing',exact:true}).click();let primary=await readyView(page);
  21 |   await primary.getByRole('searchbox',{name:'Search tasks'}).fill('S03');await expect(primary.getByRole('radio')).toHaveCount(1);
  22 |   await primary.getByRole('button',{name:'Dependencies',exact:true}).click();await readyView(page);
  23 |   const group=primary.getByRole('button',{name:'Task group align',exact:true});await group.focus();await group.press('Enter');
  24 |   await expect(primary.getByRole('button',{name:'Discuss group',exact:true})).toBeEnabled();
> 25 |   await primary.getByRole('radio',{name:'Instance align-template · Attempt 0',exact:true}).check();
     |                                                                                            ^ Error: locator.check: Clicking the checkbox did not change its state
  26 |   await expect(primary.getByRole('button',{name:'Open logs',exact:true})).toBeDisabled();
  27 |   await primary.getByRole('radio',{name:'Instance align:S03 · Attempt 2',exact:true}).check();
  28 |   await primary.getByRole('button',{name:'Open logs',exact:true}).click();const secondary=await readyView(page,'Secondary pane');
  29 |   await expect(secondary.getByRole('textbox',{name:'stderr log text'})).toHaveValue(/ERROR 😀 exact/);
  30 |   await primary.getByRole('button',{name:'Zoom in dependencies'}).click();await expect(primary.getByLabel('Dependency zoom')).toHaveText('125%');await readyView(page);
  31 |   await primary.getByText('Dependency list · 1',{exact:true}).click();const pair=primary.getByRole('button',{name:'prepare → align',exact:true});await pair.focus();await pair.press('Enter');
  32 |   await primary.getByRole('button',{name:'Discuss dependency',exact:true}).click();await expect(page.getByRole('textbox',{name:'Message draft'})).toBeFocused();
  33 |   await page.getByRole('textbox',{name:'Message draft'}).fill('Explain this observed dependency.');
  34 |   await expect(page.getByText('1 attachment · Ready for Researcher',{exact:true})).toBeVisible();
  35 |   const captured=(await document(page)).chat.attachments![0]!;
  36 |   expect(captured.evidence).toMatchObject({schemaVersion:4,selection:{kind:'run-dependency',fromTaskId:'prepare',toTaskId:'align'}});
  37 |   expect((await document(page)).collaboration?.submissions??[]).toHaveLength(0);
  38 |   await paint(page);await captureWindow(application,info.outputPath('dependency-discussion.png'));
  39 |   await primary.getByRole('button',{name:'Tasks',exact:true}).click();await readyView(page);await expect(primary.getByRole('searchbox',{name:'Search tasks'})).toHaveValue('S03');
  40 |   expect((await document(page)).selections.find(s=>s.evidence.schemaVersion===4)?.evidence).toEqual(captured.evidence);
  41 |   await primary.getByRole('button',{name:'Show in Dependencies',exact:true}).click();await readyView(page);
  42 |   await application.close();({application,page}=await launch(fixture.profile,fixture.environment));primary=await readyView(page);
  43 |   await expect(primary.getByRole('button',{name:'Dependencies',exact:true})).toHaveAttribute('aria-pressed','true');await expect(primary.getByLabel('Dependency zoom')).toHaveText('125%');
  44 |   await expect(page.getByRole('textbox',{name:'Message draft'})).toHaveValue('Explain this observed dependency.');
  45 |   await expect(primary.getByRole('button',{name:'Discuss dependency',exact:true})).toBeEnabled();
  46 |   fixture.monitor.snapshot='fixture-revision-5';fixture.monitor.tasks[1]!.attempt=3;fixture.monitor.edges=[];await writeFile(fixture.monitorPath,JSON.stringify(fixture.monitor));
  47 |   await primary.getByRole('button',{name:'Refresh',exact:true}).click();await readyView(page);await expect(primary.locator('.dependency-scope')).toContainText('0 of 0 observed dependencies');
  48 |   await expect(primary.getByRole('button',{name:'Discuss dependency',exact:true})).toHaveCount(0);
  49 |   expect((await document(page)).chat.attachments![0]).toEqual(captured);
  50 |   await page.getByRole('button',{name:'Account',exact:true}).click();await page.getByRole('button',{name:'Refresh connection',exact:true}).click();await page.getByRole('button',{name:'Close Codex account'}).click();
  51 |   await expect(page.getByText('1 attachment · Ready for Researcher',{exact:true})).toBeVisible();
  52 |   await page.getByLabel('Message attachments',{exact:true}).locator('.attachment-toggle').click();await expect(attachmentPreview(page).getByText('prepare → align',{exact:true})).toBeVisible();
  53 |   await expect(attachmentPreview(page).getByRole('row').filter({hasText:'align:S03'})).toContainText('2');await paint(page);await captureWindow(application,info.outputPath('dependency-frozen-preview.png'));await closeAttachmentPreview(page);
  54 |   await page.getByRole('button',{name:'Send',exact:true}).click();await expect(page.locator('.message-state').getByText('completed',{exact:true})).toHaveCount(1);
  55 |   const state=JSON.parse(await readFile(join(fixture.profile,'codex/home/fixture-state.json'),'utf8'));
  56 |   const threads=Object.values(state.threads) as {turns:{items:{content?:unknown[]}[]}[]}[];
  57 |   const sent=JSON.stringify(threads[0]!.turns[0]!.items[0]!.content);expect(sent).toContain('observed-authored-task-pair');expect(sent).toContain('fixture-revision-4');expect(sent).not.toContain('Read sample S03');expect(sent).not.toContain('fixture-revision-5');
  58 |  }finally{await application.close();await rm(fixture.directory,{recursive:true,force:true});}
  59 | });
  60 | 
  61 | test('missing, cyclic and bounded topology stays navigable by list and at compact native zoom',async({},info)=>{
  62 |  test.setTimeout(90_000);const fixture=await runtimeFixture();const {application,page}=await launch(fixture.profile,fixture.environment);
  63 |  try{
  64 |   await chooseProject(application,page,fixture.root);await page.getByRole('button',{name:'Attach',exact:true}).click();await page.getByRole('button',{name:'existing',exact:true}).click();const primary=await readyView(page);
  65 |   await primary.getByRole('button',{name:'Dependencies',exact:true}).click();await readyView(page);
  66 |   const raw:Record<string,unknown>=fixture.monitor;delete raw.edges;await writeFile(fixture.monitorPath,JSON.stringify(raw));await primary.getByRole('button',{name:'Refresh',exact:true}).click();await readyView(page);
  67 |   await expect(primary.getByText('Dependencies unavailable. Known task groups are listed below.',{exact:true})).toBeVisible();
  68 |   fixture.monitor.edges=[{from:'prepare',to:'align'},{from:'align',to:'prepare'}];await writeFile(fixture.monitorPath,JSON.stringify(fixture.monitor));await primary.getByRole('button',{name:'Refresh',exact:true}).click();await readyView(page);
  69 |   await expect(primary.getByLabel('Dependency overview list')).toBeVisible();await primary.getByRole('button',{name:'prepare → align',exact:true}).click();await expect(primary.getByRole('button',{name:'Discuss dependency',exact:true})).toBeEnabled();
  70 |   for(const count of [80,81]){
  71 |    fixture.monitor.tasks=Array.from({length:count},(_,i)=>({identity:'instance-'+i,task_id:'task-'+String(i).padStart(2,'0'),name:'Repeated label',status:i%2?'completed':'failed',attempt:1}));
  72 |    fixture.monitor.edges=Array.from({length:count-1},(_,i)=>({from:'task-'+String(i).padStart(2,'0'),to:'task-'+String(i+1).padStart(2,'0')}));await writeFile(fixture.monitorPath,JSON.stringify(fixture.monitor));
  73 |    await primary.getByRole('button',{name:'Refresh',exact:true}).click();await readyView(page);await expect(primary.locator('.dependency-scope')).toContainText(`80 of ${count} observed groups`);
  74 |    if(count===80){
  75 |     await expect(primary.locator('.dependency-node')).toHaveCount(80);await primary.getByRole('button',{name:'Fit graph',exact:true}).click();await readyView(page);
  76 |     await application.evaluate(({BrowserWindow})=>{const w=BrowserWindow.getAllWindows()[0]!;w.setSize(900,650);w.webContents.setZoomFactor(1.5);});
  77 |     const workspace=page.getByRole('button',{name:'Workspace',exact:true});if(await workspace.isVisible())await workspace.click();await readyView(page);
  78 |     await primary.getByRole('combobox',{name:'Dependency representation'}).selectOption('list');await readyView(page);
  79 |     const first=primary.getByRole('button',{name:'task-00 1 observed',exact:true});await first.focus();await first.press('Enter');await expect(primary.getByRole('button',{name:'Discuss group',exact:true})).toBeEnabled();
  80 |     await primary.getByRole('button',{name:'Discuss group',exact:true}).scrollIntoViewIfNeeded();await expect(primary.getByRole('button',{name:'Discuss group',exact:true})).toBeInViewport();
  81 |     await paint(page);await captureWindow(application,info.outputPath('dependency-compact-150.png'));
  82 |     await application.evaluate(({BrowserWindow})=>{const w=BrowserWindow.getAllWindows()[0]!;w.webContents.setZoomFactor(1);w.setSize(1280,840);});
  83 |     await primary.getByRole('combobox',{name:'Dependency representation'}).selectOption('graph');await readyView(page);
  84 |    }else{await expect(primary.getByLabel('Dependency overview list')).toBeVisible();await expect(primary.locator('.dependency-node')).toHaveCount(0);await paint(page);await captureWindow(application,info.outputPath('dependency-bounded-list.png'));}
  85 |   }
  86 |  }finally{await application.close();await rm(fixture.directory,{recursive:true,force:true});}
  87 | });
  88 | 
```