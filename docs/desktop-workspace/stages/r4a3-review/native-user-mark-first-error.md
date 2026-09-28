# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: notebook-shared.spec.ts >> User output marks use the same exact Show contract and remain usable in a compact enlarged view
- Location: desktop/tests/electron/notebook-shared.spec.ts:221:1

# Error details

```
TimeoutError: locator.click: Timeout 30000ms exceeded.
Call log:
  - waiting for getByRole('region', { name: 'Primary pane', exact: true }).getByRole('button', { name: 'Share mark', exact: true })

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
              - generic "analysis.ipynb Notebook reader" [ref=e104]:
                - generic [ref=e105]:
                  - generic [ref=e106]: python · 2 cells
                  - button "Refresh Notebook" [ref=e107] [cursor=pointer]
                - group [ref=e108]:
                  - generic "Saved Notebook · Read only" [ref=e109] [cursor=pointer]
                - region "Selection in analysis.ipynb" [ref=e110]:
                  - generic "analysis.ipynb · quality · Output 1 · Text range" [ref=e111]
                  - generic [ref=e112]:
                    - button "Add to message from analysis.ipynb" [ref=e113] [cursor=pointer]: Add to message
                    - button "Share mark from analysis.ipynb" [ref=e114] [cursor=pointer]: Share mark
                    - button "Remove selection from analysis.ipynb" [ref=e115] [cursor=pointer]
                - generic [ref=e118]:
                  - article "Notebook cell 1" [ref=e119]:
                    - generic [ref=e120]:
                      - strong [ref=e121]: Cell 1
                      - generic [ref=e122]: Code
                    - generic [ref=e123]:
                      - generic "Cell 1 source" [ref=e124]: "# Review saved sample quality keep = qc[\"mapped_pct\"] >= 80 qc.loc[keep]"
                      - generic [ref=e125]:
                        - button "Select displayed text" [ref=e126] [cursor=pointer]
                        - group [ref=e127]:
                          - generic "Select a text range" [ref=e128] [cursor=pointer]
                    - generic [ref=e129]:
                      - generic [ref=e130]: Saved outputs · 2
                      - button "Collapse outputs" [ref=e131] [cursor=pointer]
                    - region "Cell 1 output 1" [ref=e132]:
                      - generic [ref=e133]: Output 1 · text/plain
                      - generic [ref=e134]:
                        - generic "Cell 1 output 1 text" [ref=e135]:
                          - mark [ref=e136]: "Sample S01: 94.2% Sample S03: 91.7%"
                        - generic [ref=e137]:
                          - button "Select displayed text" [ref=e138] [cursor=pointer]
                          - group [ref=e139]:
                            - generic "Select a text range" [ref=e140] [cursor=pointer]
                    - region "Cell 1 output 2" [ref=e141]:
                      - generic [ref=e142]: Output 2 · image/png
                      - generic [ref=e144]:
                        - button "View saved image" [ref=e145] [cursor=pointer]
                        - generic [ref=e146]: 320 × 160 pixels
                  - article "Notebook cell 2" [ref=e147]:
                    - generic [ref=e148]:
                      - strong [ref=e149]: Cell 2
                      - generic [ref=e150]: Code
                    - generic [ref=e151]:
                      - generic "Cell 2 source" [ref=e152]: row_0 = xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx row_1 = xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx row_2 = xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx row_3 = xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx row_4 = xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx row_5 = xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx row_6 = xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx row_7 = xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx row_8 = xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
                      - generic [ref=e153]:
                        - button "Select displayed text" [ref=e154] [cursor=pointer]
                        - button "Previous text" [disabled] [ref=e155]
                        - generic [ref=e156]: Showing 1–2048 of 24989 characters
                        - button "Next text" [ref=e157] [cursor=pointer]
                        - group [ref=e158]:
                          - generic "Select a text range" [ref=e159] [cursor=pointer]
        - separator "Resize chat" [ref=e160]
        - region "Project chat" [ref=e162]:
          - generic [ref=e163]:
            - heading "Chat" [level=2] [ref=e164]
            - button "Project agents" [ref=e165] [cursor=pointer]:
              - text: Agents
              - generic [ref=e168]: "1"
            - button "Collapse chat" [expanded] [ref=e169] [cursor=pointer]
          - generic "Project messages" [ref=e173]:
            - generic [ref=e174]:
              - generic [ref=e175]:
                - strong [ref=e178]: Work together in this Project
                - paragraph [ref=e179]: Choose an agent and start a conversation about your work.
              - generic [ref=e180]: Opened a Project view.
          - generic [ref=e185]:
            - textbox "Message draft" [active] [ref=e186]:
              - /placeholder: Write a message for your agent…
              - text: Discuss the saved output.
            - generic [ref=e187]:
              - generic [ref=e188]:
                - generic [ref=e189]: Conversation recipient
                - combobox "Conversation recipient" [ref=e192]:
                  - option "Choose recipient"
                  - option "To Researcher" [selected]
              - generic [ref=e193]:
                - generic [ref=e194]: Message model
                - combobox "Message model" [ref=e195]:
                  - option "Fixture model" [selected]
                  - option "Fixture text model"
              - button "Send" [ref=e196] [cursor=pointer]
            - generic [ref=e199]:
              - generic [ref=e200]: Draft saved locally
              - generic [ref=e201]: Enter to send · Shift+Enter for a new line
```

# Test source

```ts
  129 |     el.scrollTop = 160;
  130 |   });
  131 |   const scroll = await primary.locator('.notebook-scroll').evaluate((el) => el.scrollTop),
  132 |     state = await userState();
  133 |   await event.getByRole('button', { name: 'Show in view', exact: true }).click();
  134 |   await readyView(page);
  135 |   await expect(primary.getByLabel('Notebook reference content')).toContainText('keep = qc');
  136 |   expect(await userState()).toEqual(state);
  137 |   await captureWindow(application, info.outputPath('notebook-agent-show.png'));
  138 |   await primary.getByRole('button', { name: 'Return to my view' }).click();
  139 |   await readyView(page);
  140 |   expect(await userState()).toEqual(state);
  141 |   expect(await primary.locator('.notebook-scroll').evaluate((el) => el.scrollTop)).toBe(scroll);
  142 |   await expect(primary.getByRole('button', { name: 'Show outputs', exact: true })).toBeVisible();
  143 |   await rm(join(root, 'analysis.ipynb'));
  144 |   await event.getByRole('button', { name: 'Show in view', exact: true }).click();
  145 |   await expect(page.getByText(/This Notebook version is unavailable/)).toBeVisible();
  146 |   await expect(
  147 |     event.getByRole('button', { name: 'View captured evidence', exact: true }),
  148 |   ).toBeVisible();
  149 |   await event.getByRole('button', { name: 'View captured evidence', exact: true }).click();
  150 |   await expect(page.locator('.evidence-preview-body')).toContainText('keep = qc');
  151 |   await captureWindow(application, info.outputPath('notebook-agent-captured.png'));
  152 | });
  153 | 
  154 | test('only explicit painted image observation receives pixels and preserves an unfinished User region through Show/Return', async ({}, info) => {
  155 |   test.setTimeout(90_000);
  156 |   const primary = await readyView(page);
  157 |   await primary.getByRole('button', { name: 'View saved image' }).click();
  158 |   await expect(primary.getByRole('img', { name: 'Saved Notebook output' })).toBeVisible();
  159 |   await primary.getByRole('button', { name: 'Select image region' }).click();
  160 |   const before = await begin('notebook-image'),
  161 |     result = await delivered();
  162 |   expect(await userState()).toEqual(before);
  163 |   expect(
  164 |     result.visible.response.contentItems.some((i: { type: string }) => i.type === 'inputImage'),
  165 |   ).toBe(false);
  166 |   expect(result.visible.value.content.imagesAvailable.length).toBe(1);
  167 |   expect(
  168 |     result.selected.response.contentItems.some((i: { type: string }) => i.type === 'inputImage'),
  169 |   ).toBe(true);
  170 |   await expect(primary.locator('.notebook-authored-image')).toContainText('Researcher');
  171 |   const region = await primary.locator('.notebook-image-region').getAttribute('style');
  172 |   const event = page.locator('.shared-reference-event');
  173 |   await event.getByRole('button', { name: 'Show in view', exact: true }).click();
  174 |   await readyView(page);
  175 |   await expect(primary.getByRole('img', { name: 'Exact Notebook reference region' })).toBeVisible();
  176 |   await captureWindow(application, info.outputPath('notebook-agent-image.png'));
  177 |   await primary.getByRole('button', { name: 'Return to my view' }).click();
  178 |   await readyView(page);
  179 |   await expect(primary.getByRole('button', { name: 'Use image region' })).toBeVisible();
  180 |   expect(await primary.locator('.notebook-image-region').getAttribute('style')).toBe(region);
  181 |   expect(await userState()).toEqual(before);
  182 | });
  183 | 
  184 | test('actual scrolling revokes old Notebook receipts and clipped source-preview cannot authorize a mark', async () => {
  185 |   test.setTimeout(90_000);
  186 |   const primary = await readyView(page);
  187 |   await begin('notebook-scope');
  188 |   await expect
  189 |     .poll(async () => {
  190 |       try {
  191 |         return JSON.parse(await readFile(file('notebook-observed.json'), 'utf8')).value.content
  192 |           .textParts.length;
  193 |       } catch {
  194 |         return 0;
  195 |       }
  196 |     })
  197 |     .toBeGreaterThan(0);
  198 |   await primary.locator('.notebook-scroll').evaluate((el) => {
  199 |     el.scrollTop = el.scrollHeight;
  200 |   });
  201 |   // An actual browser frame lets the visibility protocol revoke the previous receipt.
  202 |   await page.evaluate(
  203 |     () => new Promise((resolve) => requestAnimationFrame(() => requestAnimationFrame(resolve))),
  204 |   );
  205 |   await writeFile(file('notebook-next'), 'ready');
  206 |   const result = await delivered();
  207 |   for (const key of ['stale', 'previewPoint', 'missing']) expect(result[key].success).toBe(false);
  208 |   expect(result.fresh.success).toBe(true);
  209 |   expect(result.preview.value.receipt.pointable).toBe(false);
  210 |   const parts = result.current.value.content.textParts.filter(
  211 |     (part: { evidence: { selection: { cell: { id: string } } } }) =>
  212 |       part.evidence.selection.cell.id === 'long',
  213 |   );
  214 |   expect(parts.length).toBeGreaterThan(0);
  215 |   expect(
  216 |     parts.reduce((n: number, part: { text: string }) => n + part.text.length, 0),
  217 |   ).toBeLessThanOrEqual(4096);
  218 |   expect(parts[0].text.length).toBeLessThan(247);
  219 | });
  220 | 
  221 | test('User output marks use the same exact Show contract and remain usable in a compact enlarged view', async ({}, info) => {
  222 |   test.setTimeout(60_000);
  223 |   const primary = await readyView(page);
  224 |   await primary
  225 |     .getByRole('region', { name: 'Cell 1 output 1', exact: true })
  226 |     .getByRole('button', { name: 'Select displayed text' })
  227 |     .click();
  228 |   await page.getByRole('textbox', { name: 'Message draft' }).fill('Discuss the saved output.');
> 229 |   await primary.getByRole('button', { name: 'Share mark', exact: true }).click();
      |                                                                          ^ TimeoutError: locator.click: Timeout 30000ms exceeded.
  230 |   const event = page.locator('.shared-reference-event');
  231 |   await expect(event).toContainText('You pointed');
  232 |   const state = await userState();
  233 |   await event.getByRole('button', { name: 'Show in view', exact: true }).click();
  234 |   await readyView(page);
  235 |   await expect(primary.getByLabel('Notebook reference content')).toContainText('Sample S03: 91.7%');
  236 |   await application.evaluate(({ BrowserWindow }) => {
  237 |     const w = BrowserWindow.getAllWindows().find((w) =>
  238 |       w.webContents.getURL().startsWith('app://gobble/'),
  239 |     )!;
  240 |     w.setContentSize(1100, 800);
  241 |     w.webContents.setZoomFactor(1.25);
  242 |   });
  243 |   await expect.poll(() => page.evaluate(() => innerWidth)).toBe(880);
  244 |   if (await page.getByRole('button', { name: 'Workspace', exact: true }).isVisible())
  245 |     await page.getByRole('button', { name: 'Workspace', exact: true }).click();
  246 |   await expect(primary.getByRole('button', { name: 'Return to my view' })).toBeInViewport();
  247 |   await captureWindow(application, info.outputPath('notebook-user-compact.png'));
  248 |   await primary.getByRole('button', { name: 'Return to my view' }).click();
  249 |   await readyView(page);
  250 |   expect((await userState()).selections).toEqual(state.selections);
  251 |   expect((await userState()).chat.draft).toBe(state.chat.draft);
  252 | });
  253 | 
```