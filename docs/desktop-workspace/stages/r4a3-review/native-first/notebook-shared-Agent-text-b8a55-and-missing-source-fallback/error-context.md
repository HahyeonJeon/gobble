# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: notebook-shared.spec.ts >> Agent text and immutable question evidence preserve User state through Show/Return and missing-source fallback
- Location: desktop/tests/electron/notebook-shared.spec.ts:35:1

# Error details

```
Error: expect(locator).toBeVisible() failed

Locator: locator('.shared-reference-event').filter({ hasText: 'Review this Notebook selection.' }).getByRole('button', { name: 'Open captured evidence', exact: true })
Expected: visible
Timeout: 5000ms
Error: element(s) not found

Call log:
  - Expect "toBeVisible" locator('.shared-reference-event').filter({ hasText: 'Review this Notebook selection.' }).getByRole('button', { name: 'Open captured evidence', exact: true }) with timeout 5000ms
  - waiting for locator('.shared-reference-event').filter({ hasText: 'Review this Notebook selection.' }).getByRole('button', { name: 'Open captured evidence', exact: true })

```

```yaml
- link "Skip to workspace":
  - /url: "#workspace-main"
- banner:
  - button "Toggle files" [expanded]
  - heading "Notebook shared research" [level=1]:
    - button "Switch Project": Notebook shared research
  - button "Single pane" [pressed]
  - button "Two panes"
  - button "Project selections": Selections 1
  - button "Account"
- complementary "Project navigation":
  - group:
    - text: › Files
    - button "Project files" [disabled]
    - button "Refresh files"
    - list:
      - listitem:
        - button "analysis.ipynb"
        - button "Open analysis.ipynb in the other pane"
      - listitem:
        - button "notes.txt"
        - button "Open notes.txt in the other pane"
      - listitem:
        - button "quality.png"
        - button "Open quality.png in the other pane"
      - listitem:
        - button "samples.csv"
        - button "Open samples.csv in the other pane"
  - group:
    - text: › Runs Existing analyses
    - button "Refresh runs"
    - list
    - paragraph: No runs attached
- status: Workspace ready
- alert:
  - text: This Notebook version is unavailable. No highlight was applied. Open matching captured evidence when available.
  - button "Reload workspace"
  - button "Dismiss error"
- main:
  - region "Primary pane":
    - tablist "Primary views":
      - tab "analysis.ipynb" [selected]
    - button "More actions for analysis.ipynb": More
    - button "Close analysis.ipynb"
    - tabpanel "analysis.ipynb":
      - text: python · 2 cells
      - button "Refresh Notebook"
      - group: Saved Notebook · Read only
      - region "Selection in analysis.ipynb":
        - text: analysis.ipynb · quality · Source · Text range
        - button "Add to message from analysis.ipynb": Add to message
        - button "Share mark from analysis.ipynb": Share mark
        - button "Remove selection from analysis.ipynb"
      - article "Notebook cell 1":
        - strong: Cell 1
        - text: Code
        - mark: "# Review saved sample quality keep = qc[\"mapped_pct\"] >= 80 qc.loc[keep]"
        - button "Select displayed text"
        - group: Select a text range
        - text: Saved outputs · 2
        - button "Show outputs"
      - article "Notebook cell 2":
        - strong: Cell 2
        - text: Code row_0 = xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx row_1 = xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx row_2 = xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx row_3 = xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx row_4 = xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx row_5 = xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx row_6 = xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx row_7 = xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx row_8 = xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
        - button "Select displayed text"
        - button "Previous text" [disabled]
        - text: Showing 1–2048 of 24989 characters
        - button "Next text"
        - group: Select a text range
- separator "Resize chat"
- region "Project chat":
  - heading "Chat" [level=2]
  - button "Project agents": Agents 1
  - button "Collapse chat" [expanded]
  - text: Opened a Project view.
  - article:
    - strong: You → Researcher
    - time: 07:06 AM
    - paragraph: notebook-point
  - article:
    - strong: Researcher
    - 'button "In reply to: notebook-point"'
    - paragraph: Fixture response from Researcher.
    - text: completed
  - article "Researcher shared a reference":
    - strong: Researcher pointed to analysis.ipynb
    - text: quality · Source · Text range
    - button "Show in view"
    - paragraph: Review this Notebook selection.
    - button "View captured evidence"
    - group: Mark options
  - article "Question from Researcher":
    - strong: Researcher
    - paragraph: Should we review this saved Notebook result?
    - paragraph: "Suggestions: Review this result · Choose another part"
    - button "analysis.ipynb quality · Source · Text range":
      - strong: analysis.ipynb
      - text: quality · Source · Text range
    - text: Awaiting your reply
    - button "Reply"
    - button "Dismiss"
  - textbox "Message draft":
    - /placeholder: Write a message for your agent…
    - text: Keep my next instruction local.
  - text: Conversation recipient
  - combobox "Conversation recipient":
    - option "Choose recipient"
    - option "To Researcher" [selected]
  - text: Message model
  - combobox "Message model":
    - option "Fixture model" [selected]
    - option "Fixture text model"
  - button "Send"
  - text: Draft saved locally Enter to send · Shift+Enter for a new line
```

# Test source

```ts
  1  | import {test,expect,type ElectronApplication,type Page} from '@playwright/test';
  2  | import {mkdtemp,rm,copyFile,chmod,writeFile,readFile} from 'node:fs/promises';
  3  | import {join,resolve} from 'node:path';
  4  | import {tmpdir} from 'node:os';
  5  | import {code,encode,notebook,png} from '../../../qualification/notebook/fixtures';
  6  | import {launch,chooseProject,projectFixture,readyView,openAgentRoster,captureWindow} from './support';
  7  | let application:ElectronApplication,page:Page,base:string,profile:string,root:string;
  8  | const file=(name:string)=>join(profile,'codex/home',name);
  9  | async function document(){return page.evaluate(async()=>{
  10 |   const projects=await window.gobble.projects.list();if(!projects.ok)throw new Error(projects.error.message);
  11 |   const value=await window.gobble.workspace.read({projectId:projects.value[0]!.projectId});if(!value.ok)throw new Error(value.error.message);return value.value;
  12 | });}
  13 | async function userState(){const d=await document();return {surfaces:d.workspace.surfaces,selections:d.selections,chat:d.chat};}
  14 | test.beforeEach(async()=>{
  15 |   base=await mkdtemp(join(tmpdir(),'gobble-r4a3-'));profile=join(base,'profile');root=await projectFixture(base,'Notebook shared research');
  16 |   const cell=code('quality','# Review saved sample quality\r\nkeep = qc["mapped_pct"] >= 80\r\nqc.loc[keep]');
  17 |   cell.outputs=[{output_type:'execute_result',execution_count:7,metadata:{},data:{'text/plain':'Sample S01: 94.2%\nSample S03: 91.7%'}},{output_type:'display_data',metadata:{},data:{'image/png':png().toString('base64')}}];
  18 |   await writeFile(join(root,'analysis.ipynb'),encode(notebook([cell,code('long',Array.from({length:100},(_,i)=>'row_'+i+' = '+('x'.repeat(240))).join('\n'))])));
  19 |   const executable=join(base,'codex');await copyFile(resolve('desktop/tests/fixtures/codex.cjs'),executable);await chmod(executable,0o700);await copyFile(resolve('desktop/tests/fixtures/notebook-tools.cjs'),join(base,'notebook-tools.cjs'));
  20 |   ({application,page}=await launch(profile,{GOBBLE_CODEX_EXECUTABLE:executable}));await chooseProject(application,page,root);
  21 |   await application.evaluate(({shell})=>{shell.openExternal=async()=>{};});
  22 |   await page.getByRole('button',{name:'Account',exact:true}).click();await page.getByRole('button',{name:'Sign in with ChatGPT'}).click();await expect(page.getByText('ChatGPT connected',{exact:true})).toBeVisible();await page.getByRole('button',{name:'Close Codex account'}).click();
  23 |   await openAgentRoster(page);await page.getByRole('button',{name:'Add agent',exact:true}).click();
  24 |   const editor=page.getByRole('dialog',{name:'Add agent',exact:true});await editor.getByLabel('Name',{exact:true}).fill('Researcher');await editor.getByLabel('Project access').selectOption('sharedViews');await editor.getByRole('button',{name:'Add agent',exact:true}).click();await expect(editor).not.toBeVisible();
  25 |   await page.getByRole('combobox',{name:'Conversation recipient'}).selectOption({label:'To Researcher'});
  26 |   await page.getByRole('button',{name:'analysis.ipynb',exact:true}).click();await readyView(page);
  27 | });
  28 | test.afterEach(async()=>{await application?.close();await rm(base,{recursive:true,force:true});});
  29 | async function begin(message:string){
  30 |   const draft=page.getByRole('textbox',{name:'Message draft'});await draft.fill(message);await page.getByRole('button',{name:'Send',exact:true}).click();await expect(draft).toHaveValue('');await draft.fill('Keep my next instruction local.');
  31 |   const before=await userState();await writeFile(file('notebook-gate'),'ready');return before;
  32 | }
  33 | async function delivered(){await expect(page.locator('.message-state').getByText('completed',{exact:true})).toHaveCount(1);return JSON.parse(await readFile(file('notebook-delivery.json'),'utf8'));}
  34 | 
  35 | test('Agent text and immutable question evidence preserve User state through Show/Return and missing-source fallback',async({},info)=>{
  36 |   test.setTimeout(90_000);const primary=await readyView(page);
  37 |   await primary.getByRole('button',{name:'Select displayed text',exact:true}).first().click();
  38 |   const before=await begin('notebook-point'),result=await delivered();expect(await userState()).toEqual(before);
  39 |   await expect(page.getByRole('textbox',{name:'Message draft'})).toBeFocused();
  40 |   expect(result.visible.value.receipt.evidenceId).toBeUndefined();expect(result.selected.value.receipt).toMatchObject({pointable:true,evidence:{schemaVersion:6}});
  41 |   expect(result.selected.value.receipt.evidenceId).toBeTruthy();await expect(primary.locator('.notebook-authored-text').first()).toBeVisible();
  42 |   await captureWindow(application,info.outputPath('notebook-agent-text.png'));
  43 |   const event=page.locator('.shared-reference-event').filter({hasText:'Review this Notebook selection.'});
  44 |   await primary.getByRole('button',{name:'Collapse outputs',exact:true}).click();
  45 |   await primary.locator('.notebook-scroll').evaluate(el=>{el.scrollTop=160;});const scroll=await primary.locator('.notebook-scroll').evaluate(el=>el.scrollTop),state=await userState();
  46 |   await event.getByRole('button',{name:'Show in view',exact:true}).click();await readyView(page);
  47 |   await expect(primary.getByLabel('Notebook reference content')).toContainText('keep = qc');expect(await userState()).toEqual(state);
  48 |   await captureWindow(application,info.outputPath('notebook-agent-show.png'));
  49 |   await primary.getByRole('button',{name:'Return to my view'}).click();await readyView(page);expect(await userState()).toEqual(state);
  50 |   expect(await primary.locator('.notebook-scroll').evaluate(el=>el.scrollTop)).toBe(scroll);await expect(primary.getByRole('button',{name:'Show outputs',exact:true})).toBeVisible();
  51 |   await rm(join(root,'analysis.ipynb'));
  52 |   await event.getByRole('button',{name:'Show in view',exact:true}).click();await expect(page.getByText(/This Notebook version is unavailable/)).toBeVisible();
> 53 |   await expect(event.getByRole('button',{name:'Open captured evidence',exact:true})).toBeVisible();
     |                                                                                      ^ Error: expect(locator).toBeVisible() failed
  54 |   await event.getByRole('button',{name:'Open captured evidence',exact:true}).click();await expect(page.locator('.evidence-preview-body')).toContainText('keep = qc');
  55 |   await captureWindow(application,info.outputPath('notebook-agent-captured.png'));
  56 | });
  57 | 
  58 | test('only explicit painted image observation receives pixels and preserves an unfinished User region through Show/Return',async({},info)=>{
  59 |   test.setTimeout(90_000);const primary=await readyView(page);
  60 |   await primary.getByRole('button',{name:'View saved image'}).click();await expect(primary.getByRole('img',{name:'Saved Notebook output'})).toBeVisible();
  61 |   await primary.getByRole('button',{name:'Select image region'}).click();
  62 |   const before=await begin('notebook-image'),result=await delivered();expect(await userState()).toEqual(before);
  63 |   expect(result.visible.response.contentItems.some((i:{type:string})=>i.type==='inputImage')).toBe(false);
  64 |   expect(result.visible.value.content.imagesAvailable.length).toBe(1);expect(result.selected.response.contentItems.some((i:{type:string})=>i.type==='inputImage')).toBe(true);
  65 |   await expect(primary.locator('.notebook-authored-image')).toContainText('Researcher');
  66 |   const region=await primary.locator('.notebook-image-region').getAttribute('style');
  67 |   const event=page.locator('.shared-reference-event');await event.getByRole('button',{name:'Show in view',exact:true}).click();await readyView(page);
  68 |   await expect(primary.getByRole('img',{name:'Exact Notebook reference region'})).toBeVisible();await captureWindow(application,info.outputPath('notebook-agent-image.png'));
  69 |   await primary.getByRole('button',{name:'Return to my view'}).click();await readyView(page);
  70 |   await expect(primary.getByRole('button',{name:'Use image region'})).toBeVisible();expect(await primary.locator('.notebook-image-region').getAttribute('style')).toBe(region);expect(await userState()).toEqual(before);
  71 | });
  72 | 
  73 | test('actual scrolling revokes old Notebook receipts and clipped source-preview cannot authorize a mark',async()=>{
  74 |   test.setTimeout(90_000);const primary=await readyView(page);await begin('notebook-scope');
  75 |   await expect.poll(async()=>{try{return JSON.parse(await readFile(file('notebook-observed.json'),'utf8')).value.content.textParts.length;}catch{return 0;}}).toBeGreaterThan(0);
  76 |   await primary.locator('.notebook-scroll').evaluate(el=>{el.scrollTop=el.scrollHeight;});
  77 |   // An actual browser frame lets the visibility protocol revoke the previous receipt.
  78 |   await page.evaluate(()=>new Promise(resolve=>requestAnimationFrame(()=>requestAnimationFrame(resolve))));
  79 |   await writeFile(file('notebook-next'),'ready');const result=await delivered();
  80 |   for(const key of ['stale','previewPoint','missing'])expect(result[key].success).toBe(false);
  81 |   expect(result.fresh.success).toBe(true);expect(result.preview.value.receipt.pointable).toBe(false);
  82 |   const parts=result.current.value.content.textParts;expect(parts.every((part:{evidence:{selection:{cell:{id:string}}}})=>part.evidence.selection.cell.id==='long')).toBe(true);
  83 |   expect(parts.reduce((n:number,part:{text:string})=>n+part.text.length,0)).toBeLessThanOrEqual(4096);
  84 |   expect(parts[0].text.length).toBeLessThan(247);
  85 | });
  86 | 
```