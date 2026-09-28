# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: continuation-restored.spec.ts >> completed Run keeps its Flow and exact task actions reachable in stacked compact Panes
- Location: desktop/tests/electron/continuation-restored.spec.ts:68:1

# Error details

```
Error: expect(locator).toBeInViewport() failed

Locator:  getByRole('region', { name: 'Run flow', exact: true }).locator('.pipeline-node[data-run-status="succeeded"]').last()
Expected: in viewport
Received: viewport ratio 0
Timeout:  5000ms

Call log:
  - Expect "toBeInViewport" getByRole('region', { name: 'Run flow', exact: true }).locator('.pipeline-node[data-run-status="succeeded"]').last() with timeout 5000ms
  - waiting for getByRole('region', { name: 'Run flow', exact: true }).locator('.pipeline-node[data-run-status="succeeded"]').last()
    14 × locator resolved to <button aria-pressed="true" class="pipeline-node " data-agent-mark="false" data-run-status="succeeded" title="Inspect trimmed read quality · 1 input · 2 outputs · Selected" aria-label="Inspect Inspect trimmed read quality · 1 input · 2 outputs">…</button>
       - unexpected value "viewport ratio 0"

```

```yaml
- button "Inspect Inspect trimmed read quality · 1 input · 2 outputs" [pressed]:
  - strong: Inspect trimmed read quality
  - text: 1 input · 2 outputs succeeded
```

# Test source

```ts
  1   | import { test, expect } from '@playwright/test';
  2   | import { readFile, cp, mkdtemp } from 'node:fs/promises';
  3   | import { tmpdir } from 'node:os';
  4   | import { join } from 'node:path';
  5   | import { launch, readyView } from './support';
  6   | 
  7   | test('completed actual continuation restores its latest attempt and original review without executing', async ({}, info) => {
  8   |   test.skip(
  9   |     !process.env.GOBBLE_CONTINUATION_RESTORE_FIXTURE,
  10  |     'Requires the owned completed actual-tool fixture',
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
  26  |     await expect(flow).toContainText('This analysis is complete.');
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
  44  |     await expect(flow).toContainText('This analysis is complete.');
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
  67  | 
  68  | test('completed Run keeps its Flow and exact task actions reachable in stacked compact Panes', async ({}, info) => {
  69  |   test.skip(!process.env.GOBBLE_CONTINUATION_RESTORE_FIXTURE, 'Requires the owned completed actual-tool fixture');
  70  |   test.setTimeout(90000);
  71  |   const fixture = JSON.parse(await readFile(process.env.GOBBLE_CONTINUATION_RESTORE_FIXTURE!, 'utf8'));
  72  |   const profile = join(await mkdtemp(join(tmpdir(), 'gobble-completed-layout-')), 'profile');
  73  |   await cp(fixture.profile, profile, { recursive: true, filter: (source) => !source.split('/').at(-1)!.startsWith('Singleton') });
  74  |   const before = JSON.parse(await readFile(join(profile, 'service/catalog.json'), 'utf8'));
  75  |   const { application, page } = await launch(profile, { GOBBLE_CODEX_EXECUTABLE: join(fixture.base, 'codex') });
  76  |   try {
  77  |     await application.evaluate(({ BrowserWindow }) => BrowserWindow.getAllWindows()[0]!.setSize(1440, 950));
  78  |     await readyView(page);
  79  |     const flow = page.getByRole('region', { name: 'Run flow', exact: true });
  80  |     const node = flow.locator('.pipeline-node[data-run-status="succeeded"]').last();
  81  |     await expect(node).toBeEnabled();
  82  |     await node.focus();
  83  |     await page.keyboard.press('Enter');
  84  |     await expect(page.locator('.observation-actions')).toContainText('fastqc · Attempt 3');
  85  |     await page.getByRole('button', { name: 'Open logs', exact: true }).click();
  86  |     await expect(page.getByRole('textbox', { name: 'stderr log text', exact: true })).toContainText('Started analysis');
  87  |     await application.evaluate(({ BrowserWindow }) => BrowserWindow.getAllWindows()[0]!.setSize(1100, 800));
  88  |     await page.getByRole('button', { name: 'Workspace', exact: true }).click();
  89  |     await page.screenshot({ path: info.outputPath('completed-stacked-compact.png') });
  90  |     // Visibility alone accepts a node hidden below an ancestor's scroll edge.
> 91  |     await expect(node).toBeInViewport({ ratio: 1 });
      |                        ^ Error: expect(locator).toBeInViewport() failed
  92  |     await expect(flow).toContainText('Execution complete');
  93  |     await expect(flow.getByText('Continue this analysis', { exact: true })).toHaveCount(0);
  94  |     await page.getByRole('button', { name: 'Discuss task', exact: true }).click();
  95  |     await page.getByRole('button', { name: 'Chat', exact: true }).click();
  96  |     await expect(page.locator('.attachment-chip').last()).toContainText('Task · fastqc · Attempt 3');
  97  |     await expect(page.getByRole('textbox', { name: 'Message draft' })).toHaveValue('Keep the trimmed reads. Why does the quality check restart?');
  98  |     await page.screenshot({ path: info.outputPath('completed-stacked-chat.png') });
  99  |     expect(JSON.parse(await readFile(join(profile, 'service/catalog.json'), 'utf8')).runs).toEqual(before.runs);
  100 |   } finally { await application.close(); }
  101 | });
  102 | 
```