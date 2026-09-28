# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: notebook.spec.ts >> one MiB text stays bounded, continuation keeps absolute offsets and oversized capture leaves the draft intact
- Location: desktop/tests/electron/notebook.spec.ts:397:1

# Error details

```
Error: expect(locator).toContainText(expected) failed

Locator: locator('.app-status')
Expected substring: "64 KiB"
Received string:    "Workspace ready"
Timeout: 5000ms

Call log:
  - Expect "toContainText" locator('.app-status') with timeout 5000ms
  - waiting for locator('.app-status')
    14 × locator resolved to <div role="status" class="app-status sr-only">Workspace ready</div>
       - unexpected value "Workspace ready"

```

```yaml
- status: Workspace ready
```

# Test source

```ts
  314 |   ).toBeVisible();
  315 |   await page.getByRole('button', { name: 'Chat', exact: true }).click();
  316 |   await application.evaluate(({ BrowserWindow }) => {
  317 |     const w = BrowserWindow.getAllWindows().find((w) =>
  318 |       w.webContents.getURL().startsWith('app://gobble/'),
  319 |     )!;
  320 |     w.setBounds({ width: 1440, height: 900 });
  321 |     w.webContents.setZoomFactor(1.5);
  322 |   });
  323 |   await expect(page.getByRole('textbox', { name: 'Message draft' })).toBeInViewport();
  324 |   await captureWindow(application, info.outputPath('notebook-zoom-150.png'));
  325 | });
  326 | 
  327 | test('malformed sources recover explicitly and renderer replacement preserves draft and captured attachments', async () => {
  328 |   test.setTimeout(60_000);
  329 |   const primary = await open();
  330 |   await primary.getByRole('button', { name: 'Select displayed text', exact: true }).first().click();
  331 |   await primary.getByRole('button', { name: 'Add to message from analysis.ipynb' }).click();
  332 |   await page.getByRole('textbox', { name: 'Message draft' }).fill('Keep this Notebook review.');
  333 |   const saved = (await readDocument()).chat;
  334 |   await writeFile(join(root, 'analysis.ipynb'), Buffer.from('{broken'));
  335 |   await primary.getByRole('button', { name: 'Refresh Notebook' }).click();
  336 |   await expect(primary.getByRole('alert')).toContainText('malformed');
  337 |   await writeFile(join(root, 'analysis.ipynb'), research());
  338 |   await primary.getByRole('button', { name: 'Retry view' }).click();
  339 |   await readyView(page);
  340 |   const crashed = page.waitForEvent('crash');
  341 |   await application.evaluate(({ dialog, BrowserWindow }) => {
  342 |     const log: string[] = [];
  343 |     (process as typeof process & { recoveryLog: string[] }).recoveryLog = log;
  344 |     dialog.showMessageBox = async () => {
  345 |       log.push('dialog');
  346 |       return { response: 0, checkboxChecked: false };
  347 |     };
  348 |     dialog.showErrorBox = (a, b) => {
  349 |       log.push(a + ':' + b);
  350 |     };
  351 |     const w = BrowserWindow.getAllWindows().find((w) =>
  352 |       w.webContents.getURL().startsWith('app://gobble/'),
  353 |     )!;
  354 |     w.webContents.on('render-process-gone', (_, d) => log.push('gone:' + d.reason));
  355 |     w.webContents.on('did-start-loading', () => log.push('loading'));
  356 |     w.webContents.on('did-finish-load', () => log.push('loaded'));
  357 |     w.webContents.on('did-fail-load', (_, c, d) => log.push('failed:' + c + ':' + d));
  358 |     setTimeout(() => w.webContents.forcefullyCrashRenderer(), 100);
  359 |   });
  360 |   await crashed;
  361 |   await expect
  362 |     .poll(async () =>
  363 |       application.evaluate(async ({ BrowserWindow }) => {
  364 |         const w = BrowserWindow.getAllWindows().find((w) =>
  365 |           w.webContents.getURL().startsWith('app://gobble/'),
  366 |         )!;
  367 |         return {
  368 |           crashed: w.webContents.isCrashed(),
  369 |           loading: w.webContents.isLoading(),
  370 |           log: (process as typeof process & { recoveryLog: string[] }).recoveryLog,
  371 |         };
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
> 414 |   await expect(page.locator('.app-status')).toContainText('64 KiB');
      |                                             ^ Error: expect(locator).toContainText(expected) failed
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
  472 |   await expect(page.getByText('2 attachments · Ready for Reviewer', { exact: true })).toBeVisible();
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