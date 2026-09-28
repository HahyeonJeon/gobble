# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: notebook.spec.ts >> explicit Send delivers captured Notebook text and image through the existing addressed path
- Location: desktop/tests/electron/notebook.spec.ts:446:1

# Error details

```
Error: expect(locator).toBeVisible() failed

Locator: getByText('2 attachments · Ready for Reviewer', { exact: true })
Expected: visible
Timeout: 5000ms
Error: element(s) not found

Call log:
  - Expect "toBeVisible" getByText('2 attachments · Ready for Reviewer', { exact: true }) with timeout 5000ms
  - waiting for getByText('2 attachments · Ready for Reviewer', { exact: true })

```

```yaml
- link "Skip to workspace":
  - /url: "#workspace-main"
- banner:
  - button "Toggle files" [expanded]
  - heading "Notebook research" [level=1]:
    - button "Switch Project": Notebook research
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
        - text: analysis.ipynb · filter · Output 2 · Image region
        - button "Add to message from analysis.ipynb": Add to message
        - button "Remove selection from analysis.ipynb"
        - status: Added to message
      - article "Notebook cell 1":
        - strong: Cell 1
        - text: Markdown
        - button "View source"
        - heading "Sample quality review" [level=3]
        - paragraph: Compare the saved code and output before deciding on the threshold.
        - text: Basic saved Markdown · Use Source for exact text and continuation
      - article "Notebook cell 2":
        - strong: Cell 2
        - text: "Code # Review sample quality keep = qc[\"mapped_pct\"] >= 80 qc.loc[keep]"
        - button "Select displayed text"
        - group: Select a text range
        - text: Saved outputs · 3
        - button "Collapse outputs"
        - region "Cell 2 output 1":
          - text: Output 1 · text/plain
          - paragraph: Static representation; active alternatives are not rendered.
          - text: sample mapped_pct 0 S01 94.2 2 S03 91.7
          - button "Select displayed text"
          - group: Select a text range
        - region "Cell 2 output 2":
          - text: Output 2 · image/png
          - button "Hide image"
          - text: 320 × 160 pixels Zoom
          - combobox "Notebook image zoom":
            - option "50%"
            - option "100%" [selected]
            - option "150%"
          - button "Select image region"
          - group "Notebook image selection area":
            - img "Saved Notebook output"
        - region "Cell 2 output 3":
          - text: Output 3 · Unavailable
          - paragraph: No supported saved image or plain-text alternative.
- separator "Resize chat"
- region "Project chat":
  - heading "Chat" [level=2]
  - button "Project agents": Agents 1
  - button "Collapse chat" [expanded]
  - strong: Work together in this Project
  - paragraph: Choose an agent and start a conversation about your work.
  - text: Opened a Project view.
  - button "analysis.ipynb filter · Source · Text range":
    - strong: analysis.ipynb
    - text: filter · Source · Text range
  - button "Remove attachment analysis.ipynb"
  - button "analysis.ipynb filter · Output 2 · Image region":
    - strong: analysis.ipynb
    - text: filter · Output 2 · Image region
  - button "Remove attachment analysis.ipynb"
  - status:
    - text: Choose a recipient to prepare these attachments.
    - button "Prepare again"
  - textbox "Message draft":
    - /placeholder: Write a message for your agent…
    - text: Review these saved Notebook excerpts.
  - text: Conversation recipient
  - combobox "Conversation recipient":
    - option "Choose recipient" [selected]
    - option "To Reviewer"
  - button "Send" [disabled]
  - text: Draft saved locally Enter to send · Shift+Enter for a new line
```

# Test source

```ts
  372 |       }),
  373 |     )
  374 |     .toMatchObject({
  375 |       crashed: false,
  376 |       loading: false,
  377 |       log: expect.arrayContaining(['dialog', 'loaded']),
  378 |     });
  379 |   await expect
  380 |     .poll(async () =>
  381 |       application.evaluate(async ({ BrowserWindow }) => {
  382 |         const w = BrowserWindow.getAllWindows().find((w) =>
  383 |           w.webContents.getURL().startsWith('app://gobble/'),
  384 |         )!;
  385 |         return w.webContents.executeJavaScript(
  386 |           `(async()=>{const projects=await window.gobble.projects.list();const doc=await window.gobble.workspace.read({projectId:projects.value[0].projectId});return {draft:document.querySelector('[aria-label="Message draft"]')?.value,ready:document.querySelector('.surface-view')?.dataset.ready,chat:doc.value.chat};})()`,
  387 |         );
  388 |       }),
  389 |     )
  390 |     .toMatchObject({
  391 |       draft: 'Keep this Notebook review.',
  392 |       ready: 'true',
  393 |       chat: { attachments: saved.attachments },
  394 |     });
  395 | });
  396 | 
  397 | test('one MiB text stays bounded, continuation keeps absolute offsets and oversized capture leaves the draft intact', async () => {
  398 |   await writeFile(join(root, 'analysis.ipynb'), fixture('long-text'));
  399 |   const primary = await open();
  400 |   const text = primary.getByLabel('Cell 1 source', { exact: true });
  401 |   expect((await text.textContent())!.length).toBeLessThanOrEqual(2048);
  402 |   await primary.getByRole('button', { name: 'Next text', exact: true }).click();
  403 |   await primary.getByRole('button', { name: 'Select displayed text', exact: true }).click();
  404 |   expect((await readDocument()).selections[0]!.evidence.selection).toMatchObject({
  405 |     selector: { start: 2048, end: 4096 },
  406 |   });
  407 |   await primary.getByRole('button', { name: 'Add to message from analysis.ipynb' }).click();
  408 |   const saved = (await readDocument()).chat.attachments;
  409 |   await primary.getByText('Select a text range', { exact: true }).click();
  410 |   await primary.getByLabel('Cell 1 source start offset', { exact: true }).fill('0');
  411 |   await primary.getByLabel('Cell 1 source end offset', { exact: true }).fill('100000');
  412 |   await primary.getByRole('button', { name: 'Use range', exact: true }).click();
  413 |   await primary.getByRole('button', { name: 'Add to message from analysis.ipynb' }).click();
  414 |   await expect(page.locator('.app-status')).toContainText('64 KiB');
  415 |   expect((await readDocument()).chat.attachments).toEqual(saved);
  416 | });
  417 | 
  418 | test('embedded image above generic text limits opens on demand and bounded exact capture succeeds', async () => {
  419 |   const bytes = fixture('large');
  420 |   expect(bytes.length).toBeGreaterThan(3 * 1024 * 1024);
  421 |   await writeFile(join(root, 'analysis.ipynb'), bytes);
  422 |   const primary = await open(),
  423 |     output = primary.getByRole('region', { name: 'Cell 3 output 2', exact: true });
  424 |   const surface = (await readDocument()).workspace.surfaces[0]!;
  425 |   if (surface.resource.kind !== 'file') throw new Error('Missing file');
  426 |   const raw = await page.evaluate(async (input) => window.gobble.files.read(input), {
  427 |     projectId: surface.projectId,
  428 |     resourceId: surface.resource.resourceId,
  429 |   });
  430 |   expect(raw).toMatchObject({ ok: false, error: { code: 'unsupported' } });
  431 |   await expect(output.getByRole('img')).toHaveCount(0);
  432 |   await output.getByRole('button', { name: 'View saved image' }).click();
  433 |   await expect(output.getByRole('img')).toBeVisible();
  434 |   await output.getByRole('button', { name: 'Select image region' }).click();
  435 |   await output.getByRole('button', { name: 'Use image region' }).click();
  436 |   await primary.getByRole('button', { name: 'Add to message from analysis.ipynb' }).click();
  437 |   expect((await readDocument()).chat.attachments![0]!.capture).toMatchObject({
  438 |     kind: 'notebook',
  439 |     representation: { width: 120, height: 80, originalWidth: 860, originalHeight: 860 },
  440 |   });
  441 |   await output.getByRole('button', { name: 'Hide image' }).click();
  442 |   await expect(output.getByRole('img')).toHaveCount(0);
  443 |   expect(await page.evaluate(() => Reflect.has(globalThis, 'notebookAttack'))).toBe(false);
  444 | });
  445 | 
  446 | test('explicit Send delivers captured Notebook text and image through the existing addressed path', async () => {
  447 |   const primary = await open();
  448 |   await primary.getByRole('button', { name: 'Select displayed text', exact: true }).first().click();
  449 |   await primary.getByRole('button', { name: 'Add to message from analysis.ipynb' }).click();
  450 |   const output = primary.getByRole('region', { name: 'Cell 2 output 2', exact: true });
  451 |   await output.getByRole('button', { name: 'View saved image' }).click();
  452 |   await expect(output.getByRole('img')).toBeVisible();
  453 |   await output.getByRole('button', { name: 'Select image region' }).click();
  454 |   await output.getByRole('button', { name: 'Use image region' }).click();
  455 |   await primary.getByRole('button', { name: 'Add to message from analysis.ipynb' }).click();
  456 |   await application.evaluate(({ shell }) => {
  457 |     shell.openExternal = async () => {};
  458 |   });
  459 |   await page.getByRole('button', { name: 'Account', exact: true }).click();
  460 |   await page.getByRole('button', { name: 'Sign in with ChatGPT' }).click();
  461 |   await expect(page.getByText('ChatGPT connected', { exact: true })).toBeVisible();
  462 |   await page.getByRole('button', { name: 'Close Codex account' }).click();
  463 |   await openAgentRoster(page);
  464 |   await page.getByRole('button', { name: 'Add agent', exact: true }).click();
  465 |   const dialog = page.getByRole('dialog', { name: 'Add agent', exact: true });
  466 |   await dialog.getByLabel('Name', { exact: true }).fill('Reviewer');
  467 |   await dialog.getByRole('button', { name: 'Add agent', exact: true }).click();
  468 |   await expect(dialog).not.toBeVisible();
  469 |   await page
  470 |     .getByRole('textbox', { name: 'Message draft' })
  471 |     .fill('Review these saved Notebook excerpts.');
> 472 |   await expect(page.getByText('2 attachments · Ready for Reviewer', { exact: true })).toBeVisible();
      |                                                                                       ^ Error: expect(locator).toBeVisible() failed
  473 |   await rm(join(root, 'analysis.ipynb'));
  474 |   await page.getByRole('button', { name: 'Send', exact: true }).click();
  475 |   await expect(page.locator('.message-state').getByText('completed', { exact: true })).toHaveCount(
  476 |     1,
  477 |   );
  478 |   const sent = JSON.parse(await readFile(join(profile, 'codex/home/fixture-state.json'), 'utf8'));
  479 |   expect(JSON.stringify(sent.threads)).toContain('notebook-display-utf16');
  480 |   expect(JSON.stringify(sent.threads)).toContain('natural-image-pixels');
  481 |   expect(JSON.stringify(sent.threads)).toContain('data:image/png;base64,');
  482 |   expect((await readDocument()).chat.attachments).toEqual([]);
  483 | });
  484 | 
```