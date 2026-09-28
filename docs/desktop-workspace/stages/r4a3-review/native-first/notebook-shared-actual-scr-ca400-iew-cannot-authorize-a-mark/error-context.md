# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: notebook-shared.spec.ts >> actual scrolling revokes old Notebook receipts and clipped source-preview cannot authorize a mark
- Location: desktop/tests/electron/notebook-shared.spec.ts:73:1

# Error details

```
Error: expect(received).toBe(expected) // Object.is equality

Expected: true
Received: false
```

# Page snapshot

```yaml
- generic [ref=e3]:
  - link "Skip to workspace" [ref=e4] [cursor=pointer]:
    - /url: "#workspace-main"
  - banner [ref=e5]:
    - button "Toggle files" [expanded] [ref=e6] [cursor=pointer]
    - heading "Notebook shared research" [level=1] [ref=e9]:
      - button "Switch Project" [ref=e10] [cursor=pointer]:
        - generic [ref=e11]: Notebook shared research
    - generic [ref=e14]:
      - generic "Workspace layout" [ref=e15]:
        - button "Single pane" [pressed] [ref=e16] [cursor=pointer]
        - button "Two panes" [ref=e19] [cursor=pointer]
      - button "Project selections" [ref=e22] [cursor=pointer]:
        - text: Selections
        - generic [ref=e23]: "0"
      - button "Account" [ref=e25] [cursor=pointer]
  - generic [ref=e27]:
    - complementary "Project navigation" [ref=e28]:
      - generic [ref=e29]:
        - group [ref=e30]:
          - generic "› Files" [ref=e31] [cursor=pointer]
          - generic [ref=e34]:
            - button "Project files" [disabled] [ref=e35]
            - button "Refresh files" [ref=e36] [cursor=pointer]
          - list [ref=e39]:
            - listitem [ref=e40]:
              - button "analysis.ipynb" [ref=e41] [cursor=pointer]
              - button "Open analysis.ipynb in the other pane" [ref=e45] [cursor=pointer]
            - listitem [ref=e48]:
              - button "notes.txt" [ref=e49] [cursor=pointer]
              - button "Open notes.txt in the other pane" [ref=e53] [cursor=pointer]
            - listitem [ref=e56]:
              - button "quality.png" [ref=e57] [cursor=pointer]
              - button "Open quality.png in the other pane" [ref=e61] [cursor=pointer]
            - listitem [ref=e64]:
              - button "samples.csv" [ref=e65] [cursor=pointer]
              - button "Open samples.csv in the other pane" [ref=e69] [cursor=pointer]
        - group [ref=e72]:
          - generic "› Runs" [ref=e73] [cursor=pointer]
          - generic [ref=e76]:
            - generic [ref=e77]: Existing analyses
            - button "Refresh runs" [ref=e78] [cursor=pointer]
          - list [ref=e81]
          - paragraph [ref=e82]: No runs attached
    - generic [ref=e83]:
      - status [ref=e84]: Workspace ready
      - generic [ref=e85]:
        - main [ref=e86]:
          - region "Primary pane" [ref=e89]:
            - generic [ref=e90]:
              - tablist "Primary views" [ref=e91]:
                - tab "analysis.ipynb" [selected] [ref=e92] [cursor=pointer]
              - generic "View actions" [ref=e94]:
                - button "More actions for analysis.ipynb" [ref=e95] [cursor=pointer]: More
                - button "Close analysis.ipynb" [ref=e98] [cursor=pointer]
            - tabpanel "analysis.ipynb" [ref=e101]:
              - generic "analysis.ipynb Notebook reader" [ref=e104]:
                - generic [ref=e105]:
                  - generic [ref=e106]: python · 2 cells
                  - button "Refresh Notebook" [ref=e107] [cursor=pointer]
                - group [ref=e108]:
                  - generic "Saved Notebook · Read only" [ref=e109] [cursor=pointer]
                - generic [ref=e110]:
                  - article "Notebook cell 1" [ref=e111]:
                    - generic [ref=e112]:
                      - strong [ref=e113]: Cell 1
                      - generic [ref=e114]: Code
                    - generic [ref=e115]:
                      - generic "Cell 1 source" [ref=e116]:
                        - text: "# Review saved sample quality"
                        - mark [ref=e117]: keep = qc["mapped_pct"] >= 80 qc.loc[keep]
                      - generic [ref=e118]:
                        - button "Select displayed text" [ref=e119] [cursor=pointer]
                        - group [ref=e120]:
                          - generic "Select a text range" [ref=e121] [cursor=pointer]
                    - generic [ref=e122]:
                      - generic [ref=e123]: Saved outputs · 2
                      - button "Collapse outputs" [ref=e124] [cursor=pointer]
                    - region "Cell 1 output 1" [ref=e125]:
                      - generic [ref=e126]: Output 1 · text/plain
                      - generic [ref=e127]:
                        - generic "Cell 1 output 1 text" [ref=e128]: "Sample S01: 94.2% Sample S03: 91.7%"
                        - generic [ref=e129]:
                          - button "Select displayed text" [ref=e130] [cursor=pointer]
                          - group [ref=e131]:
                            - generic "Select a text range" [ref=e132] [cursor=pointer]
                    - region "Cell 1 output 2" [ref=e133]:
                      - generic [ref=e134]: Output 2 · image/png
                      - generic [ref=e136]:
                        - button "View saved image" [ref=e137] [cursor=pointer]
                        - generic [ref=e138]: 320 × 160 pixels
                  - article "Notebook cell 2" [ref=e139]:
                    - generic [ref=e140]:
                      - strong [ref=e141]: Cell 2
                      - generic [ref=e142]: Code
                    - generic [ref=e143]:
                      - generic "Cell 2 source" [ref=e144]: row_0 = xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx row_1 = xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx row_2 = xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx row_3 = xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx row_4 = xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx row_5 = xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx row_6 = xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx row_7 = xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx row_8 = xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
                      - generic [ref=e145]:
                        - button "Select displayed text" [ref=e146] [cursor=pointer]
                        - button "Previous text" [disabled] [ref=e147]
                        - generic [ref=e148]: Showing 1–2048 of 24989 characters
                        - button "Next text" [ref=e149] [cursor=pointer]
                        - group [ref=e150]:
                          - generic "Select a text range" [ref=e151] [cursor=pointer]
        - separator "Resize chat" [ref=e152]
        - region "Project chat" [ref=e154]:
          - generic [ref=e155]:
            - heading "Chat" [level=2] [ref=e156]
            - button "Project agents" [ref=e157] [cursor=pointer]:
              - text: Agents
              - generic [ref=e160]: "1"
            - button "Collapse chat" [expanded] [ref=e161] [cursor=pointer]
          - generic "Project messages" [ref=e165]:
            - generic [ref=e166]:
              - generic [ref=e167]: Opened a Project view.
              - article [ref=e173]:
                - generic [ref=e174]:
                  - generic [ref=e175]:
                    - strong [ref=e176]: You → Researcher
                    - time [ref=e177]: 07:06 AM
                  - paragraph [ref=e178]: notebook-scope
              - article [ref=e180]:
                - generic [ref=e181]:
                  - generic [ref=e182]:
                    - generic [aria-hidden] [ref=e183]: R
                    - strong [ref=e184]: Researcher
                  - 'button "In reply to: notebook-scope" [ref=e185] [cursor=pointer]'
                  - paragraph [ref=e186]: Fixture response from Researcher
                - generic [ref=e187]: completed
              - article "Researcher shared a reference" [ref=e190]:
                - generic [ref=e191]:
                  - generic [ref=e192]:
                    - strong [ref=e193]:
                      - text: Researcher
                      - generic [ref=e194]: pointed to analysis.ipynb
                    - generic [ref=e195]: quality · Source · Text range
                  - button "Show in view" [ref=e196] [cursor=pointer]
                - paragraph [ref=e197]: Review this Notebook selection.
                - group [ref=e198]:
                  - generic "Mark options" [ref=e199] [cursor=pointer]
          - generic [ref=e200]:
            - textbox "Message draft" [active] [ref=e201]:
              - /placeholder: Write a message for your agent…
              - text: Keep my next instruction local.
            - generic [ref=e202]:
              - generic [ref=e203]:
                - generic [ref=e204]: Conversation recipient
                - combobox "Conversation recipient" [ref=e207]:
                  - option "Choose recipient"
                  - option "To Researcher" [selected]
              - generic [ref=e208]:
                - generic [ref=e209]: Message model
                - combobox "Message model" [ref=e210]:
                  - option "Fixture model" [selected]
                  - option "Fixture text model"
              - button "Send" [ref=e211] [cursor=pointer]
            - generic [ref=e214]:
              - generic [ref=e215]: Draft saved locally
              - generic [ref=e216]: Enter to send · Shift+Enter for a new line
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
  53 |   await expect(event.getByRole('button',{name:'Open captured evidence',exact:true})).toBeVisible();
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
> 82 |   const parts=result.current.value.content.textParts;expect(parts.every((part:{evidence:{selection:{cell:{id:string}}}})=>part.evidence.selection.cell.id==='long')).toBe(true);
     |                                                                                                                                                                      ^ Error: expect(received).toBe(expected) // Object.is equality
  83 |   expect(parts.reduce((n:number,part:{text:string})=>n+part.text.length,0)).toBeLessThanOrEqual(4096);
  84 |   expect(parts[0].text.length).toBeLessThan(247);
  85 | });
  86 | 
```