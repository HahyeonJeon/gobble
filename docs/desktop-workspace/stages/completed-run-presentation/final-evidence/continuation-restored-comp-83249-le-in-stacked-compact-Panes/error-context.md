# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: continuation-restored.spec.ts >> completed Run keeps its Flow and exact task actions reachable in stacked compact Panes
- Location: desktop/tests/electron/continuation-restored.spec.ts:67:1

# Error details

```
Error: expect(locator).toContainText(expected) failed

Locator: locator('.attachment-chip').last()
Expected substring: "Task · fastqc · Attempt 3"
Received string:    "Analysis 1Task · fastqc · Attempt 1Preview"
Timeout: 5000ms

Call log:
  - Expect "toContainText" locator('.attachment-chip').last() with timeout 5000ms
  - waiting for locator('.attachment-chip').last()
    14 × locator resolved to <div class="attachment-chip">…</div>
       - unexpected value "Analysis 1Task · fastqc · Attempt 1Preview"

```

```yaml
- button "Analysis 1 Task · fastqc · Attempt 1":
  - strong: Analysis 1
  - text: Task · fastqc · Attempt 1
- button "Remove attachment Analysis 1"
```

# Test source

```ts
  11  |   );
  12  |   test.setTimeout(90000);
  13  |   const fixture = JSON.parse(
  14  |     await readFile(process.env.GOBBLE_CONTINUATION_RESTORE_FIXTURE!, 'utf8'),
  15  |   );
  16  |   const before = JSON.parse(await readFile(join(fixture.profile, 'service/catalog.json'), 'utf8'));
  17  |   const { application, page } = await launch(fixture.profile, {
  18  |     GOBBLE_CODEX_EXECUTABLE: join(fixture.base, 'codex'),
  19  |   });
  20  |   try {
  21  |     await application.evaluate(({ BrowserWindow }) =>
  22  |       BrowserWindow.getAllWindows()[0]!.setSize(1440, 950),
  23  |     );
  24  |     await readyView(page);
  25  |     const flow = page.getByRole('region', { name: 'Run flow', exact: true });
  26  |     await expect(flow).toContainText('Execution complete');
  27  |     await expect(flow.locator('.pipeline-node[data-run-status="succeeded"]')).toHaveCount(2);
  28  |     await expect(page.getByRole('row').filter({ hasText: 'fastqc' })).toContainText('3');
  29  |     await expect(page.getByRole('article', { name: 'Continue analysis' })).toContainText(
  30  |       'Saved confirmation plan',
  31  |     );
  32  |     await expect(page.getByRole('button', { name: 'Resume analysis', exact: true })).toHaveCount(0);
  33  |     await page.screenshot({ path: info.outputPath('completed-wide.png') });
  34  |     await page.locator('.attachment-chip').getByRole('button').first().click();
  35  |     const preview = page.getByRole('dialog', { name: /^Attachment:/ });
  36  |     await expect(preview).toContainText('Will restart · Attempt 2');
  37  |     await expect(preview).toContainText('Attempt 1 remains in history.');
  38  |     await page.screenshot({ path: info.outputPath('completed-original-reference.png') });
  39  |     await page.keyboard.press('Escape');
  40  |     await application.evaluate(({ BrowserWindow }) =>
  41  |       BrowserWindow.getAllWindows()[0]!.setSize(1100, 800),
  42  |     );
  43  |     await page.getByRole('button', { name: 'Workspace', exact: true }).click();
  44  |     await expect(flow).toContainText('Execution complete');
  45  |     await page.screenshot({ path: info.outputPath('completed-compact-flow.png') });
  46  |     await page.getByRole('button', { name: 'Chat', exact: true }).click();
  47  |     await expect(page.getByRole('textbox', { name: 'Message draft' })).toHaveValue(
  48  |       'Keep the trimmed reads. Why does the quality check restart?',
  49  |     );
  50  |     await page.screenshot({ path: info.outputPath('completed-compact-chat.png') });
  51  |     const records = await page.evaluate(
  52  |       async (projectId) => window.gobble.continuations.list({ projectId }),
  53  |       before.runs[0].projectId,
  54  |     );
  55  |     if (!records.ok) throw Error(records.error.message);
  56  |     const receipts = records.value.filter((v) => v.operation?.receipt);
  57  |     expect(receipts).toHaveLength(2);
  58  |     expect(receipts.at(-1)?.operation?.observation?.status).toBe('succeeded');
  59  |     expect(
  60  |       JSON.parse(await readFile(join(fixture.profile, 'service/catalog.json'), 'utf8')).runs,
  61  |     ).toEqual(before.runs);
  62  |   } finally {
  63  |     await application.close();
  64  |   }
  65  | });
  66  | 
  67  | test('completed Run keeps its Flow and exact task actions reachable in stacked compact Panes', async ({}, info) => {
  68  |   test.skip(
  69  |     !process.env.GOBBLE_CONTINUATION_RESTORE_FIXTURE,
  70  |     'Requires the owned completed actual-tool fixture',
  71  |   );
  72  |   test.setTimeout(90000);
  73  |   const fixture = JSON.parse(
  74  |     await readFile(process.env.GOBBLE_CONTINUATION_RESTORE_FIXTURE!, 'utf8'),
  75  |   );
  76  |   const profile = join(await mkdtemp(join(tmpdir(), 'gobble-completed-layout-')), 'profile');
  77  |   await cp(fixture.profile, profile, {
  78  |     recursive: true,
  79  |     filter: (source) => !source.split('/').at(-1)!.startsWith('Singleton'),
  80  |   });
  81  |   const before = JSON.parse(await readFile(join(profile, 'service/catalog.json'), 'utf8'));
  82  |   const { application, page } = await launch(profile, {
  83  |     GOBBLE_CODEX_EXECUTABLE: join(fixture.base, 'codex'),
  84  |   });
  85  |   try {
  86  |     await application.evaluate(({ BrowserWindow }) =>
  87  |       BrowserWindow.getAllWindows()[0]!.setSize(1440, 950),
  88  |     );
  89  |     await readyView(page);
  90  |     const flow = page.getByRole('region', { name: 'Run flow', exact: true });
  91  |     const node = flow.locator('.pipeline-node[data-run-status="succeeded"]').last();
  92  |     await expect(node).toBeEnabled();
  93  |     await node.focus();
  94  |     await page.keyboard.press('Enter');
  95  |     await expect(page.locator('.observation-actions')).toContainText('fastqc · Attempt 3');
  96  |     await page.getByRole('button', { name: 'Open logs', exact: true }).click();
  97  |     await expect(page.getByRole('textbox', { name: 'stderr log text', exact: true })).toContainText(
  98  |       'Started analysis',
  99  |     );
  100 |     await application.evaluate(({ BrowserWindow }) =>
  101 |       BrowserWindow.getAllWindows()[0]!.setSize(1100, 800),
  102 |     );
  103 |     await page.getByRole('button', { name: 'Workspace', exact: true }).click();
  104 |     await page.screenshot({ path: info.outputPath('completed-stacked-compact.png') });
  105 |     // Check clipping, allowing only subpixel rounding in Chromium's intersection ratio.
  106 |     await expect(node).toBeInViewport({ ratio: 0.999 });
  107 |     await expect(flow).toContainText('Execution complete');
  108 |     await expect(flow.getByText('Continue this analysis', { exact: true })).toHaveCount(0);
  109 |     await page.getByRole('button', { name: 'Discuss task', exact: true }).click();
  110 |     await page.getByRole('button', { name: 'Chat', exact: true }).click();
> 111 |     await expect(page.locator('.attachment-chip').last()).toContainText(
      |                                                           ^ Error: expect(locator).toContainText(expected) failed
  112 |       'Task · fastqc · Attempt 3',
  113 |     );
  114 |     await expect(page.getByRole('textbox', { name: 'Message draft' })).toHaveValue(
  115 |       'Keep the trimmed reads. Why does the quality check restart?',
  116 |     );
  117 |     await page.screenshot({ path: info.outputPath('completed-stacked-chat.png') });
  118 |     expect(JSON.parse(await readFile(join(profile, 'service/catalog.json'), 'utf8')).runs).toEqual(
  119 |       before.runs,
  120 |     );
  121 |   } finally {
  122 |     await application.close();
  123 |   }
  124 | });
  125 | 
```