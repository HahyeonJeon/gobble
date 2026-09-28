# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: pipeline-proposals.spec.ts >> B comparison: scoped Agent proposal, exact chat reference, adoption and restart
- Location: desktop/tests/electron/pipeline-proposals.spec.ts:8:1

# Error details

```
Error: expect(received).toBe(expected) // Object.is equality

Expected: 4
Received: 5
```

# Test source

```ts
  61  |     await expect(primary.getByText('2 steps · 1 input · 2 connections')).toBeVisible({
  62  |       timeout: 135_000,
  63  |     });
  64  |     await readyView(page);
  65  |     await application.evaluate(({ shell }) => {
  66  |       shell.openExternal = async () => {};
  67  |     });
  68  |     await page.getByRole('button', { name: 'Account', exact: true }).click();
  69  |     await page.getByRole('button', { name: 'Sign in with ChatGPT' }).click();
  70  |     await expect(page.getByText('ChatGPT connected', { exact: true })).toBeVisible();
  71  |     await page.getByRole('button', { name: 'Close Codex account' }).click();
  72  |     await openAgentRoster(page);
  73  |     await page.getByRole('button', { name: 'Add agent', exact: true }).click();
  74  |     const editor = page.getByRole('dialog', { name: 'Add agent', exact: true });
  75  |     await editor.getByLabel('Name', { exact: true }).fill('Researcher');
  76  |     await editor.getByLabel('Project access').selectOption('sharedViews');
  77  |     await editor.getByRole('button', { name: 'Add agent', exact: true }).click();
  78  |     await page
  79  |       .getByRole('combobox', { name: 'Conversation recipient' })
  80  |       .selectOption({ label: 'To Researcher' });
  81  |     await primary.getByRole('button', { name: 'Discuss changes', exact: true }).click();
  82  |     await page
  83  |       .getByRole('checkbox', { name: 'Allow a Pipeline proposal for this message' })
  84  |       .check();
  85  |     await page.getByRole('textbox', { name: 'Message draft' }).fill('pipeline-proposal');
  86  |     await page.getByRole('button', { name: 'Send', exact: true }).click();
  87  |     const home = join(profile, 'codex/home');
  88  |     await expect
  89  |       .poll(async () => readFile(join(home, 'proposal-delivery.json'), 'utf8').catch(() => ''), {
  90  |         timeout: 30_000,
  91  |       })
  92  |       .not.toBe('');
  93  |     await primary.getByRole('button', { name: 'Changes', exact: true }).click();
  94  |     const review = page.getByRole('region', { name: 'Pipeline change review' });
  95  |     await expect(review.getByText('30 Phred', { exact: true })).toBeVisible({ timeout: 260_000 });
  96  |     await expect(review.getByText('25 Phred', { exact: true })).toBeVisible();
  97  |     await expect(review.locator('.pipeline-node[data-change="setting"]')).toHaveCount(1);
  98  |     await expect(review.locator('.pipeline-node[data-change="unchanged"]')).toHaveCount(2);
  99  |     for (const node of await review.locator('.pipeline-node[data-change="unchanged"]').all())
  100 |       await expect(node).toBeDisabled();
  101 |     await expect(review.locator('.pipeline-node[data-change="setting"]')).toHaveCSS(
  102 |       'background-color',
  103 |       'rgb(255, 249, 233)',
  104 |     );
  105 |     const addedEdge = review.locator('g[data-added-edge="true"] [role="button"]');
  106 |     await addedEdge.focus();
  107 |     await page.keyboard.press('Enter');
  108 |     await expect(review.getByText('Quality check added', { exact: true })).toBeVisible();
  109 |     await review.getByRole('button', { name: '1 Quality threshold' }).click();
  110 |     await page.screenshot({ path: info.outputPath('b-setting-review.png') });
  111 |     await review.getByRole('button', { name: '2 Add quality check' }).click();
  112 |     await expect(review.getByText('No quality check here', { exact: true })).toBeVisible();
  113 |     await expect(review.getByText('Quality check added', { exact: true })).toBeVisible();
  114 |     await page.screenshot({ path: info.outputPath('b-added-step-review.png') });
  115 |     await review.getByRole('button', { name: '1 Quality threshold' }).click();
  116 |     await review.getByRole('button', { name: 'Discuss this change' }).click();
  117 |     // Engine checking can finish while another app is foreground. Shared pointing
  118 |     // requires the actual Project window, unlike retained comparison reads.
  119 |     await application.evaluate(({ BrowserWindow }) => {
  120 |       const window = BrowserWindow.getAllWindows()[0]!;
  121 |       window.show();
  122 |       window.focus();
  123 |     });
  124 |     await expect
  125 |       .poll(() =>
  126 |         application.evaluate(({ BrowserWindow }) => BrowserWindow.getAllWindows()[0]!.isFocused()),
  127 |       )
  128 |       .toBe(true);
  129 |     await page.getByRole('textbox', { name: 'Message draft' }).fill('pipeline-review');
  130 |     await page.getByRole('button', { name: 'Send', exact: true }).click();
  131 |     await expect
  132 |       .poll(async () => readFile(join(home, 'review-delivery.json'), 'utf8').catch(() => ''), {
  133 |         timeout: 30_000,
  134 |       })
  135 |       .not.toBe('');
  136 |     await expect(review.getByRole('button', { name: /Agent reference/ })).toBeVisible();
  137 |     await page
  138 |       .getByRole('textbox', { name: 'Message draft' })
  139 |       .fill('Keep this unsent note while adopting.');
  140 |     await review.getByRole('button', { name: 'Adopt proposal', exact: true }).click();
  141 |     await expect(review).toContainText('app-managed source copy');
  142 |     await review.getByRole('button', { name: 'Confirm adoption' }).click();
  143 |     await expect(
  144 |       page.getByText('Adopted as current. No Run was started.', { exact: true }),
  145 |     ).toBeVisible();
  146 |     expect(await readFile(join(project, 'trim-review/pipeline.go'), 'utf8')).toBe(source);
  147 |     await page.screenshot({ path: info.outputPath('b-adopted-review.png') });
  148 |     await application.close();
  149 |     ({ application, page } = await launch(profile, { GOBBLE_CODEX_EXECUTABLE: executable }));
  150 |     primary = await readyView(page);
  151 |     await expect(page.getByRole('textbox', { name: 'Message draft' })).toHaveValue(
  152 |       'Keep this unsent note while adopting.',
  153 |     );
  154 |     await primary.getByRole('button', { name: 'Changes', exact: true }).click();
  155 |     await expect(page.getByRole('region', { name: 'Pipeline change review' })).toContainText(
  156 |       'This is the current version. No Run was started.',
  157 |     );
  158 |     await expect(page.getByText('25 Phred', { exact: true })).toBeVisible();
  159 |     await expect(page.getByText('30 Phred', { exact: true })).toBeVisible();
  160 |     const catalog = JSON.parse(await readFile(join(profile, 'service/catalog.json'), 'utf8'));
> 161 |     expect(catalog.schemaVersion).toBe(4);
      |                                   ^ Error: expect(received).toBe(expected) // Object.is equality
  162 |     expect(catalog.runs).toHaveLength(0);
  163 |     expect(Object.keys(catalog.revisions)).toEqual([catalog.pipelines[0].pipelineId]);
  164 |     expect(catalog.drafts).toEqual({});
  165 |     await page.setViewportSize({ width: 850, height: 900 });
  166 |     await page.screenshot({ path: info.outputPath('b-compact-review.png') });
  167 |   } finally {
  168 |     await application.close();
  169 |   }
  170 | });
  171 | 
```