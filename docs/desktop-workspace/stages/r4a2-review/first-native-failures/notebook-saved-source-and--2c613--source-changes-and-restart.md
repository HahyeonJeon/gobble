# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: notebook.spec.ts >> saved source and output attach through the existing composer, preview offline and survive source changes and restart
- Location: desktop/tests/electron/notebook.spec.ts:93:1

# Error details

```
Error: expect(received).toMatchObject(expected)

- Expected  - 3
+ Received  + 1

  Object {
    "part": Object {
-     "index": 0,
-     "kind": "output",
-     "mime": "text/plain",
+     "kind": "source",
    },
  }
```

# Page snapshot

```yaml
- generic [ref=e3]:
  - link "Skip to workspace" [ref=e4] [cursor=pointer]:
    - /url: "#workspace-main"
  - banner [ref=e5]:
    - button "Toggle files" [expanded] [ref=e6] [cursor=pointer]
    - heading "Notebook research" [level=1] [ref=e9]:
      - button "Switch Project" [ref=e10] [cursor=pointer]:
        - generic [ref=e11]: Notebook research
    - generic [ref=e14]:
      - generic "Workspace layout" [ref=e15]:
        - button "Single pane" [pressed] [ref=e16] [cursor=pointer]
        - button "Two panes" [ref=e19] [cursor=pointer]
      - button "Project selections" [ref=e22] [cursor=pointer]:
        - text: Selections
        - generic [ref=e23]: "1"
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
              - generic "analysis.ipynb Notebook reader" [ref=e102]:
                - generic [ref=e103]:
                  - generic [ref=e104]: python · 2 cells
                  - button "Refresh Notebook" [ref=e105] [cursor=pointer]
                - group [ref=e106]:
                  - generic "Saved Notebook · Read only" [ref=e107] [cursor=pointer]
                - region "Selection in analysis.ipynb" [ref=e108]:
                  - generic "analysis.ipynb · filter · Output 1 · Text range" [ref=e109]
                  - generic [ref=e110]:
                    - button "Add to message from analysis.ipynb" [ref=e111] [cursor=pointer]: Add to message
                    - button "Remove selection from analysis.ipynb" [ref=e112] [cursor=pointer]
                  - status [ref=e115]: Added to message
                - generic [ref=e116]:
                  - article "Notebook cell 1" [ref=e117]:
                    - generic [ref=e118]:
                      - strong [ref=e119]: Cell 1
                      - generic [ref=e120]: Markdown
                      - button "View source" [ref=e121] [cursor=pointer]
                    - generic [ref=e122]:
                      - heading "Sample quality review" [level=3] [ref=e123]
                      - paragraph [ref=e124]: Compare the saved code and output before deciding on the threshold.
                      - generic [ref=e125]: Basic saved Markdown · Use Source for exact text and continuation
                  - article "Notebook cell 2" [ref=e126]:
                    - generic [ref=e127]:
                      - strong [ref=e128]: Cell 2
                      - generic [ref=e129]: Code
                    - generic [ref=e130]:
                      - generic "Cell 2 source" [ref=e131]: "# Review sample quality keep = qc[\"mapped_pct\"] >= 80 qc.loc[keep]"
                      - generic [ref=e132]:
                        - button "Select displayed text" [ref=e133] [cursor=pointer]
                        - group [ref=e134]:
                          - generic "Select a text range" [ref=e135] [cursor=pointer]
                    - generic [ref=e136]:
                      - generic [ref=e137]: Saved outputs · 3
                      - button "Collapse outputs" [ref=e138] [cursor=pointer]
                    - region "Cell 2 output 1" [ref=e139]:
                      - generic [ref=e140]: Output 1 · text/plain
                      - paragraph [ref=e141]: Static representation; active alternatives are not rendered.
                      - generic [ref=e142]:
                        - generic "Cell 2 output 1 text" [ref=e143]:
                          - mark [ref=e144]: sample mapped_pct 0 S01 94.2 2 S03 91.7
                        - generic [ref=e145]:
                          - button "Select displayed text" [ref=e146] [cursor=pointer]
                          - group [ref=e147]:
                            - generic "Select a text range" [ref=e148] [cursor=pointer]
                    - region "Cell 2 output 2" [ref=e149]:
                      - generic [ref=e150]: Output 2 · image/png
                      - generic [ref=e152]:
                        - button "View saved image" [ref=e153] [cursor=pointer]
                        - generic [ref=e154]: 320 × 160 pixels
                    - region "Cell 2 output 3" [ref=e155]:
                      - generic [ref=e156]: Output 3 · Unavailable
                      - paragraph [ref=e157]: No supported saved image or plain-text alternative.
        - separator "Resize chat" [ref=e158]
        - region "Project chat" [ref=e160]:
          - generic [ref=e161]:
            - heading "Chat" [level=2] [ref=e162]
            - button "Project agents" [ref=e163] [cursor=pointer]:
              - text: Agents
              - generic [ref=e166]: "0"
            - button "Collapse chat" [expanded] [ref=e167] [cursor=pointer]
          - generic "Project messages" [ref=e171]:
            - generic [ref=e172]:
              - generic [ref=e173]:
                - strong [ref=e176]: Work together in this Project
                - paragraph [ref=e177]: Choose an agent and start a conversation about your work.
              - generic [ref=e178]: Opened a Project view.
          - generic [ref=e183]:
            - generic "Message attachments" [ref=e185]:
              - generic [ref=e186]:
                - generic [ref=e187]:
                  - button "analysis.ipynb filter · Source · Text range" [ref=e188] [cursor=pointer]:
                    - generic [ref=e191]:
                      - strong [ref=e192]: analysis.ipynb
                      - generic [ref=e193]: filter · Source · Text range
                    - generic [aria-hidden] [ref=e194]: Preview
                  - button "Remove attachment analysis.ipynb" [ref=e195] [cursor=pointer]
                - generic [ref=e198]:
                  - button "analysis.ipynb filter · Source · Text range" [ref=e199] [cursor=pointer]:
                    - generic [ref=e202]:
                      - strong [ref=e203]: analysis.ipynb
                      - generic [ref=e204]: filter · Source · Text range
                    - generic [aria-hidden] [ref=e205]: Preview
                  - button "Remove attachment analysis.ipynb" [ref=e206] [cursor=pointer]
            - status [ref=e209]:
              - text: Choose a recipient to prepare these attachments.
              - button "Prepare again" [ref=e210] [cursor=pointer]
            - textbox "Message draft" [active] [ref=e211]:
              - /placeholder: Write a message for your agent…
              - text: Should we revisit this threshold?
            - generic [ref=e212]:
              - generic [ref=e213]:
                - generic [ref=e214]: Conversation recipient
                - combobox "Conversation recipient" [disabled] [ref=e217]:
                  - option "No agent connected" [selected]
              - button "Send" [disabled] [ref=e218]
            - generic [ref=e221]:
              - generic [ref=e222]: Draft saved locally
              - generic [ref=e223]: Enter to send · Shift+Enter for a new line
```

# Test source

```ts
  25  | function research() {
  26  |   const cell = code(
  27  |     'filter',
  28  |     '# Review sample quality\r\nkeep = qc["mapped_pct"] >= 80\r\nqc.loc[keep]',
  29  |   );
  30  |   cell.outputs = [
  31  |     {
  32  |       output_type: 'execute_result',
  33  |       execution_count: 7,
  34  |       metadata: {},
  35  |       data: {
  36  |         'text/plain': '    sample  mapped_pct\n0   S01          94.2\n2   S03          91.7',
  37  |         'text/html': '<script>window.notebookAttack=true</script>',
  38  |       },
  39  |     },
  40  |     {
  41  |       output_type: 'display_data',
  42  |       metadata: {},
  43  |       data: { 'image/png': savedPng.toString('base64') },
  44  |     },
  45  |     {
  46  |       output_type: 'display_data',
  47  |       metadata: {},
  48  |       data: { 'application/vnd.jupyter.widget-view+json': { model_id: 'example' } },
  49  |     },
  50  |   ];
  51  |   return encode(
  52  |     notebook([
  53  |       {
  54  |         cell_type: 'markdown',
  55  |         id: 'intro',
  56  |         metadata: {},
  57  |         source:
  58  |           '# Sample quality review\nCompare the saved code and output before deciding on the threshold.',
  59  |       },
  60  |       cell,
  61  |     ]),
  62  |   );
  63  | }
  64  | async function readDocument() {
  65  |   return page.evaluate(async () => {
  66  |     const projects = await window.gobble.projects.list();
  67  |     if (!projects.ok) throw new Error(projects.error.message);
  68  |     const result = await window.gobble.workspace.read({ projectId: projects.value[0]!.projectId });
  69  |     if (!result.ok) throw new Error(result.error.message);
  70  |     return result.value;
  71  |   });
  72  | }
  73  | async function open() {
  74  |   await page.getByRole('button', { name: 'analysis.ipynb', exact: true }).click();
  75  |   return readyView(page);
  76  | }
  77  | test.beforeEach(async () => {
  78  |   base = await mkdtemp(join(tmpdir(), 'gobble-r4a2-'));
  79  |   profile = join(base, 'profile');
  80  |   executable = join(base, 'codex');
  81  |   await copyFile(resolve('desktop/tests/fixtures/codex.cjs'), executable);
  82  |   await chmod(executable, 0o700);
  83  |   root = await projectFixture(base, 'Notebook research');
  84  |   await writeFile(join(root, 'analysis.ipynb'), research());
  85  |   ({ application, page } = await launch(profile, { GOBBLE_CODEX_EXECUTABLE: executable }));
  86  |   await chooseProject(application, page, root);
  87  | });
  88  | test.afterEach(async () => {
  89  |   await application?.close();
  90  |   await rm(base, { recursive: true, force: true });
  91  | });
  92  | 
  93  | test('saved source and output attach through the existing composer, preview offline and survive source changes and restart', async ({}, info) => {
  94  |   test.setTimeout(60_000);
  95  |   let primary = await open();
  96  |   const cell = primary.getByRole('article', { name: 'Notebook cell 2' });
  97  |   await expect(cell.getByText('No supported saved image or plain-text alternative.')).toBeVisible();
  98  |   await page
  99  |     .getByRole('textbox', { name: 'Message draft' })
  100 |     .fill('Should we revisit this threshold?');
  101 |   await cell.getByRole('button', { name: 'Select displayed text', exact: true }).first().click();
  102 |   await primary.getByRole('button', { name: 'Add to message from analysis.ipynb' }).click();
  103 |   let doc = await readDocument();
  104 |   const first = doc.chat.attachments![0]!;
  105 |   expect(first.evidence).toMatchObject({
  106 |     schemaVersion: 6,
  107 |     selection: { cell: { kind: 'id', id: 'filter' }, part: { kind: 'source' } },
  108 |   });
  109 |   expect(first.capture?.kind).toBe('notebook');
  110 |   await page
  111 |     .getByLabel('Message attachments', { exact: true })
  112 |     .locator('.attachment-toggle')
  113 |     .first()
  114 |     .click();
  115 |   await expect(attachmentPreview(page).locator('.evidence-preview-body pre')).toContainText(
  116 |     'keep = qc',
  117 |   );
  118 |   await captureWindow(application, info.outputPath('notebook-offline-preview.png'));
  119 |   await closeAttachmentPreview(page);
  120 |   const output = cell.getByRole('region', { name: 'Cell 2 output 1', exact: true });
  121 |   await output.getByRole('button', { name: 'Select displayed text', exact: true }).click();
  122 |   await primary.getByRole('button', { name: 'Add to message from analysis.ipynb' }).click();
  123 |   doc = await readDocument();
  124 |   expect(doc.chat.attachments).toHaveLength(2);
> 125 |   expect(doc.chat.attachments![1]!.evidence.selection).toMatchObject({
      |                                                        ^ Error: expect(received).toMatchObject(expected)
  126 |     part: { kind: 'output', index: 0, mime: 'text/plain' },
  127 |   });
  128 |   const captured = doc.chat.attachments;
  129 |   await captureWindow(application, info.outputPath('notebook-discussion.png'));
  130 |   await writeFile(join(root, 'analysis.ipynb'), fixture('changed'));
  131 |   await primary.getByRole('button', { name: 'Refresh Notebook', exact: true }).click();
  132 |   await expect.poll(async () => (await readDocument()).selections.length).toBe(0);
  133 |   await readyView(page);
  134 |   expect((await readDocument()).selections).toHaveLength(0);
  135 |   expect((await readDocument()).chat.attachments).toEqual(captured);
  136 |   await application.close();
  137 |   ({ application, page } = await launch(profile, { GOBBLE_CODEX_EXECUTABLE: executable }));
  138 |   primary = await readyView(page);
  139 |   expect((await readDocument()).chat.attachments).toEqual(captured);
  140 |   await expect(page.getByRole('textbox', { name: 'Message draft' })).toHaveValue(
  141 |     'Should we revisit this threshold?',
  142 |   );
  143 |   await rm(join(root, 'analysis.ipynb'));
  144 |   await primary.getByRole('button', { name: 'Refresh Notebook', exact: true }).click();
  145 |   await expect(primary.getByRole('button', { name: 'Retry view', exact: true })).toBeVisible();
  146 |   await page
  147 |     .getByLabel('Message attachments', { exact: true })
  148 |     .locator('.attachment-toggle')
  149 |     .first()
  150 |     .click();
  151 |   await expect(attachmentPreview(page).locator('.evidence-preview-body pre')).toContainText(
  152 |     'keep = qc',
  153 |   );
  154 |   await closeAttachmentPreview(page);
  155 |   await writeFile(join(root, 'analysis.ipynb'), research());
  156 |   await primary.getByRole('button', { name: 'Retry view', exact: true }).click();
  157 |   await readyView(page);
  158 |   expect((await readDocument()).collaboration?.submissions ?? []).toHaveLength(0);
  159 | });
  160 | 
  161 | test('native text drag uses exact display offsets and rejects stale or foreign image requests', async () => {
  162 |   const primary = await open(),
  163 |     text = primary.getByLabel('Cell 2 source', { exact: true });
  164 |   const points = await text.evaluate((pre) => {
  165 |     const node = pre.firstChild!;
  166 |     const a = document.createRange(),
  167 |       b = document.createRange();
  168 |     a.setStart(node, 0);
  169 |     a.setEnd(node, 1);
  170 |     b.setStart(node, 8);
  171 |     b.setEnd(node, 9);
  172 |     const start = a.getBoundingClientRect(),
  173 |       end = b.getBoundingClientRect();
  174 |     return { x1: start.left, y: start.top + start.height / 2, x2: end.left };
  175 |   });
  176 |   await page.mouse.move(points.x1, points.y);
  177 |   await page.mouse.down();
  178 |   await page.mouse.move(points.x2, points.y, { steps: 8 });
  179 |   await page.mouse.up();
  180 |   await expect
  181 |     .poll(async () => (await readDocument()).selections[0]?.evidence.selection)
  182 |     .toMatchObject({ kind: 'notebook', selector: { kind: 'text', start: 0, end: 8 } });
  183 |   const result = await page.evaluate(async () => {
  184 |     const projects = await window.gobble.projects.list();
  185 |     if (!projects.ok) throw new Error('Projects unavailable');
  186 |     const projectId = projects.value[0]!.projectId;
  187 |     const doc = await window.gobble.workspace.read({ projectId });
  188 |     const connected = await window.gobble.workspace.connect();
  189 |     if (!doc.ok || !connected.ok) throw new Error('Workspace unavailable');
  190 |     // Reconnection retires every old ready lease before a stale image request.
  191 |     const s = doc.value.workspace.surfaces[0]!;
  192 |     const selected = doc.value.selections[0]!.evidence;
  193 |     if (selected.schemaVersion !== 6) throw new Error('Missing Notebook selection');
  194 |     const ack = {
  195 |       schemaVersion: 3 as const,
  196 |       projectId,
  197 |       surfaceId: s.surfaceId,
  198 |       rendererSessionId: connected.value.rendererSessionId,
  199 |       requestId: 'req_forged',
  200 |       generation: 1,
  201 |       dataRevision: selected.dataRevision,
  202 |       presentation: { spec: 0, view: 0, filter: 0 },
  203 |     };
  204 |     const stale = await window.gobble.workspace.notebookImage({
  205 |       acknowledgment: ack,
  206 |       target: selected,
  207 |     });
  208 |     const foreign = await window.gobble.workspace.notebookImage({
  209 |       acknowledgment: ack,
  210 |       target: { ...selected, projectId: 'prj_foreign' },
  211 |     });
  212 |     return { stale, foreign };
  213 |   });
  214 |   expect(result.stale).toMatchObject({ ok: false, error: { code: 'stale_revision' } });
  215 |   expect(result.foreign).toMatchObject({ ok: false, error: { code: 'forbidden' } });
  216 | });
  217 | 
  218 | test('native pointer and keyboard image selection at 150 percent captures the exact original pixels', async ({}, info) => {
  219 |   const primary = await open(),
  220 |     output = primary.getByRole('region', { name: 'Cell 2 output 2', exact: true });
  221 |   await output.getByRole('button', { name: 'View saved image' }).click();
  222 |   await expect(output.getByRole('img')).toBeVisible();
  223 |   await output.getByRole('combobox', { name: 'Notebook image zoom' }).selectOption('1.5');
  224 |   await output.getByRole('button', { name: 'Select image region' }).click();
  225 |   const area = output.getByRole('group', { name: 'Notebook image selection area' });
```