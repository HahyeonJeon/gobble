# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: saved-reports.spec.ts >> Flow opens a complete saved report; compact/focus/close/restart preserve its original result
- Location: desktop/tests/electron/saved-reports.spec.ts:16:1

# Error details

```
Error: expect(locator).toHaveCount(expected) failed

Locator:  locator('.message-state').getByText('completed', { exact: true })
Expected: 1
Received: 0
Timeout:  5000ms

Call log:
  - Expect "toHaveCount" locator('.message-state').getByText('completed', { exact: true }) with timeout 5000ms
  - waiting for locator('.message-state').getByText('completed', { exact: true })
    14 × locator resolved to 0 elements
       - unexpected value "0"

```

# Test source

```ts
  84  |     expect(await report.locator('a,script,iframe').count()).toBe(0);
  85  |     await page.screenshot({ path: info.outputPath('report-stacked.png') });
  86  |     await report.getByRole('combobox', { name: 'Report section' }).selectOption('M1');
  87  |     await expect(
  88  |       report.getByRole('img', { name: 'Per base quality graph', exact: true }),
  89  |     ).toBeInViewport();
  90  |     await page.screenshot({ path: info.outputPath('report-chart.png') });
  91  |     const pane = page.locator('.work-pane').filter({ has: report });
  92  |     await pane.getByRole('button', { name: 'Maximize secondary pane', exact: true }).click();
  93  |     await expect(page.getByRole('button', { name: 'Restore panes', exact: true })).toBeVisible();
  94  |     await expect(page.locator('#primary-pane')).toHaveCount(0);
  95  |     await expect(report.getByRole('combobox', { name: 'Report section' })).toHaveValue('M1');
  96  |     await expect(
  97  |       report.getByRole('img', { name: 'Per base quality graph', exact: true }),
  98  |     ).toBeInViewport();
  99  |     await page.screenshot({ path: info.outputPath('report-focused.png') });
  100 |     await application.evaluate(({ BrowserWindow }) =>
  101 |       BrowserWindow.getAllWindows()[0]!.setSize(1000, 760),
  102 |     );
  103 |     await expect(report.getByRole('combobox', { name: 'Report section' })).toBeVisible();
  104 |     await page.screenshot({ path: info.outputPath('report-compact.png') });
  105 |     await expect
  106 |       .poll(() => report.evaluate((el) => el.scrollWidth <= el.clientWidth + 1))
  107 |       .toBe(true);
  108 |     const state = await page.evaluate(async (projectId) => {
  109 |       const s = await window.gobble.workspace.read({ projectId });
  110 |       if (!s.ok) throw Error(s.error.message);
  111 |       return s.value;
  112 |     }, fixture.projectId);
  113 |     const saved = state.savedReports![0]!;
  114 |     const savedBytes = await readFile(
  115 |       join(
  116 |         fixture.profile,
  117 |         'workspace/evidence',
  118 |         fixture.projectId,
  119 |         saved.asset.hash.slice(7) + '.blob',
  120 |       ),
  121 |     );
  122 |     expect(createHash('sha256').update(savedBytes).digest('hex')).toBe(saved.asset.hash.slice(7));
  123 |     expect(JSON.parse(savedBytes.toString()).content.modules).toHaveLength(10);
  124 |     await pane
  125 |       .getByRole('button', { name: 'Close sample_trimmed.fq.gz FastQC Report', exact: true })
  126 |       .click();
  127 |     await unlink(source);
  128 |     await writeFile(join(fixture.base, 'new-attempt'), 'yes');
  129 |     await application.close();
  130 |     application = undefined;
  131 |     launched = await launch(fixture.profile, fixture.environment);
  132 |     application = launched.application;
  133 |     page = launched.page;
  134 |     await application.evaluate(({ BrowserWindow }) =>
  135 |       BrowserWindow.getAllWindows()[0]!.setSize(1600, 1000),
  136 |     );
  137 |     await page.getByRole('button', { name: 'Analysis 1', exact: true }).click();
  138 |     const restored = page.getByRole('region', { name: 'Run flow', exact: true });
  139 |     await restored.getByText('Saved reports (1)', { exact: true }).click();
  140 |     await restored
  141 |       .getByRole('button', { name: 'sample_trimmed.fq.gz FastQC Report · attempt 1', exact: true })
  142 |       .click();
  143 |     await expect(page.locator('.report-view img')).toHaveCount(8);
  144 |     const doc = await page.evaluate(async (projectId) => {
  145 |       const s = await window.gobble.workspace.read({ projectId });
  146 |       if (!s.ok) throw Error(s.error.message);
  147 |       return s.value;
  148 |     }, fixture.projectId);
  149 |     expect(doc.savedReports).toEqual([saved]);
  150 |     await page.screenshot({ path: info.outputPath('report-restored.png') });
  151 |     await application.evaluate(({ shell }) => {
  152 |       shell.openExternal = async () => {};
  153 |     });
  154 |     await page.getByRole('button', { name: 'Account', exact: true }).click();
  155 |     await page.getByRole('button', { name: 'Sign in with ChatGPT' }).click();
  156 |     await expect(page.getByText('ChatGPT connected', { exact: true })).toBeVisible();
  157 |     await page.getByRole('button', { name: 'Close Codex account' }).click();
  158 |     await openAgentRoster(page);
  159 |     await page.getByRole('button', { name: 'Add agent', exact: true }).click();
  160 |     const dialog = page.getByRole('dialog', { name: 'Add agent', exact: true });
  161 |     await dialog.getByLabel('Name', { exact: true }).fill('Researcher');
  162 |     await dialog.getByLabel('Project access').selectOption('sharedViews');
  163 |     await dialog.getByRole('button', { name: 'Add agent', exact: true }).click();
  164 |     await page
  165 |       .getByRole('combobox', { name: 'Conversation recipient' })
  166 |       .selectOption({ label: 'To Researcher' });
  167 |     await page.getByRole('button', { name: 'Attach report', exact: true }).click();
  168 |     await expect(
  169 |       page.getByText('1 attachment · Ready for Researcher', { exact: true }),
  170 |     ).toBeVisible();
  171 |     await page
  172 |       .getByLabel('Message attachments', { exact: true })
  173 |       .getByRole('button', { name: /^sample_trimmed/ })
  174 |       .click();
  175 |     await expect(attachmentPreview(page).locator('.report-view img')).toHaveCount(8);
  176 |     await page.screenshot({ path: info.outputPath('report-attachment.png') });
  177 |     await closeAttachmentPreview(page);
  178 |     await page.getByRole('button', { name: 'Point to report', exact: true }).click();
  179 |     await expect(page.getByLabel('Report pointers')).toContainText('You · Whole report');
  180 |     await page.getByRole('textbox', { name: 'Message draft' }).fill('report-read');
  181 |     await page.getByRole('button', { name: 'Send', exact: true }).click();
  182 |     await expect(
  183 |       page.locator('.message-state').getByText('completed', { exact: true }),
> 184 |     ).toHaveCount(1);
      |       ^ Error: expect(locator).toHaveCount(expected) failed
  185 |     await expect(page.getByLabel('Report pointers')).toContainText('Researcher · Whole report');
  186 |     const delivery = JSON.parse(
  187 |       await readFile(join(fixture.profile, 'codex/home/report-delivery.json'), 'utf8'),
  188 |     );
  189 |     const retained = JSON.parse(savedBytes.toString('utf8'));
  190 |     const originals = retained.content.modules
  191 |       .flatMap((m: { blocks: { kind: string; id?: string; base64?: string }[] }) => m.blocks)
  192 |       .filter((b: { kind: string }) => b.kind === 'image');
  193 |     expect(delivery.charts).toHaveLength(8);
  194 |     expect(JSON.stringify(delivery.attached)).not.toContain('base64');
  195 |     expect(delivery.reading.report.content.modules).toHaveLength(10);
  196 |     for (const chart of delivery.charts) {
  197 |       const original = originals.find((b: { id: string }) => b.id === chart.id);
  198 |       expect(
  199 |         chart.result.contentItems.find((i: { type: string }) => i.type === 'inputImage').imageUrl,
  200 |       ).toBe('data:image/png;base64,' + original.base64);
  201 |     }
  202 |     expect(delivery.reading.pointable).toBe(false);
  203 |     expect(delivery.observed.receipt.pointable).toBe(true);
  204 |     expect(delivery.observed.content.imagesReturned).toEqual([]);
  205 |     await page.screenshot({ path: info.outputPath('report-discussion.png') });
  206 | 
  207 |     await page.getByRole('button', { name: 'Producer logs · attempt 1', exact: true }).click();
  208 |     await expect(
  209 |       page
  210 |         .getByText(/attempt.*changed|attempt.*unavailable|no longer.*attempt|requested attempt/i)
  211 |         .first(),
  212 |     ).toBeVisible();
  213 |     expect(
  214 |       (
  215 |         await readFile(
  216 |           join(
  217 |             fixture.profile,
  218 |             'workspace/evidence',
  219 |             fixture.projectId,
  220 |             saved.asset.hash.slice(7) + '.blob',
  221 |           ),
  222 |         )
  223 |       ).equals(savedBytes),
  224 |     ).toBe(true);
  225 |   } catch (error) {
  226 |     const page = application && (await application.windows())[0];
  227 |     await page?.screenshot({ path: info.outputPath('failure.png') }).catch(() => {});
  228 |     throw error;
  229 |   } finally {
  230 |     await application?.close();
  231 |   }
  232 | });
  233 | 
```