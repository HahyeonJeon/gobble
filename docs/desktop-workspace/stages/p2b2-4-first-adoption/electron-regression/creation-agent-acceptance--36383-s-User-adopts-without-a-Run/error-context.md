# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: creation-agent-acceptance.spec.ts >> Creation: signed-in Agent proposes, reads and points; User adopts without a Run
- Location: desktop/tests/electron/creation-agent-acceptance.spec.ts:10:1

# Error details

```
Error: expect(locator).toHaveCount(expected) failed

Locator:  getByText('Completed', { exact: true })
Expected: 2
Received: 0
Timeout:  90000ms

Call log:
  - Expect "toHaveCount" getByText('Completed', { exact: true }) with timeout 90000ms
  - waiting for getByText('Completed', { exact: true })
    182 × locator resolved to 0 elements
        - unexpected value "0"

```

# Test source

```ts
  1   | import { test, expect } from '@playwright/test';
  2   | import { mkdtemp, mkdir, copyFile, chmod, readFile, writeFile, rm } from 'node:fs/promises';
  3   | import { tmpdir } from 'node:os';
  4   | import { join, resolve } from 'node:path';
  5   | import { launch, chooseProject, openAgentRoster } from './support';
  6   | 
  7   | // Opt-in account qualification, separate from the deterministic protocol peer.
  8   | // Only the explicitly selected local auth file enters this disposable profile;
  9   | // it is never logged, copied into evidence, or retained after the test.
  10  | test('Creation: signed-in Agent proposes, reads and points; User adopts without a Run', async ({}, info) => {
  11  |   test.skip(
  12  |     process.env.GOBBLE_REAL_CREATION !== '1' || !process.env.GOBBLE_CREATION_AUTH_FILE,
  13  |     'requires explicit local signed-in account qualification',
  14  |   );
  15  |   test.setTimeout(600_000);
  16  |   const base = await mkdtemp(join(tmpdir(), 'gobble-creation-agent-')),
  17  |     profile = join(base, 'profile'),
  18  |     project = join(base, 'Read quality acceptance'),
  19  |     home = join(profile, 'codex/home');
  20  |   await mkdir(project);
  21  |   await mkdir(home, { recursive: true, mode: 0o700 });
  22  |   await writeFile(join(project, 'sample.fastq.gz'), 'Metadata-only isolated acceptance data.');
  23  |   const credential = join(home, 'auth.json');
  24  |   await copyFile(process.env.GOBBLE_CREATION_AUTH_FILE!, credential);
  25  |   await chmod(credential, 0o600);
  26  |   let app: Awaited<ReturnType<typeof launch>> | undefined;
  27  |   try {
  28  |     app = await launch(profile, { GOBBLE_CODEX_EXECUTABLE: resolve('node_modules/.bin/codex') });
  29  |     const { application, page } = app;
  30  |     page.setDefaultTimeout(20000);
  31  |     await chooseProject(application, page, project);
  32  |     await page.getByRole('button', { name: 'New pipeline', exact: true }).click();
  33  |     const view = page.locator('.creation-view');
  34  |     await view.getByRole('button', { name: '◇ sample.fastq.gz', exact: true }).click();
  35  |     await view.getByRole('button', { name: 'Use selected file' }).click();
  36  |     await view.getByRole('button', { name: 'Connect engine', exact: true }).click();
  37  |     await expect(view.getByText('● Analysis engine connected', { exact: true })).toBeVisible({
  38  |       timeout: 45000,
  39  |     });
  40  |     await page.getByRole('button', { name: 'Account', exact: true }).click();
  41  |     await page.getByRole('button', { name: 'Refresh connection' }).click();
  42  |     await expect(page.getByText('ChatGPT connected', { exact: true })).toBeVisible({
  43  |       timeout: 45000,
  44  |     });
  45  |     await page.getByRole('button', { name: 'Close Codex account' }).click();
  46  |     await openAgentRoster(page);
  47  |     await page.getByRole('button', { name: 'Add agent', exact: true }).click();
  48  |     const editor = page.getByRole('dialog', { name: 'Add agent', exact: true });
  49  |     await editor.getByLabel('Name', { exact: true }).fill('Researcher');
  50  |     await editor.getByLabel('Project access').selectOption('sharedViews');
  51  |     const model = await editor.getByLabel('Model').inputValue();
  52  |     const effort = await editor.getByLabel('Reasoning effort').inputValue();
  53  |     await editor
  54  |       .getByLabel('Role instructions')
  55  |       .fill(
  56  |         'Discuss this analysis in concise English using the shared flow and its exact facts. Keep source code behind the UI.',
  57  |       );
  58  |     await editor.getByRole('button', { name: 'Add agent', exact: true }).click();
  59  |     await page
  60  |       .getByRole('combobox', { name: 'Conversation recipient' })
  61  |       .selectOption({ label: 'To Researcher' });
  62  |     await page
  63  |       .getByRole('checkbox', { name: 'Allow a new Pipeline proposal for this message' })
  64  |       .check();
  65  |     await page
  66  |       .getByRole('textbox', { name: 'Message draft' })
  67  |       .fill(
  68  |         'Please design a new analysis for the selected single-end reads: trim adapters and low-quality bases with a quality threshold of 25 Phred and minimum length of 40 bp, then inspect the trimmed reads with FastQC. Prepare the proposal through the provided creation tools so I can review the flow. Do not start an analysis.',
  69  |       );
  70  |     await page.getByRole('button', { name: 'Send', exact: true }).click();
  71  |     await expect(view.getByRole('heading', { name: 'No current version' })).toBeVisible({
  72  |       timeout: 240000,
  73  |     });
  74  |     await expect(view.locator('.pipeline-node[data-change="added-step"]')).toHaveCount(2);
  75  |     await view.getByRole('button', { name: /Inspect Trim adapters/ }).click();
  76  |     await view.getByRole('button', { name: /Quality threshold.*25/ }).click();
  77  |     await view.getByRole('button', { name: 'Add to message', exact: true }).click();
  78  |     await expect(page.locator('.composer-context')).toContainText('Quality threshold · 25 Phred');
  79  |     await page
  80  |       .getByRole('textbox', { name: 'Message draft' })
  81  |       .fill(
  82  |         'Read the exact attached proposal setting, briefly explain what this quality threshold means, and point to this exact setting using the creation review and point tools. Begin your pointer note with "Quality threshold reviewed:". This is discussion only; keep the proposal unchanged.',
  83  |       );
  84  |     await expect(page.getByRole('button', { name: 'Send', exact: true })).toBeEnabled({
  85  |       timeout: 90000,
  86  |     });
  87  |     await page.getByRole('button', { name: 'Send', exact: true }).click();
  88  |     await expect(
  89  |       view.getByRole('button', { name: /Researcher: Quality threshold reviewed:/ }),
  90  |     ).toBeVisible({
  91  |       timeout: 180000,
  92  |     });
> 93  |     await expect(page.getByText('Completed', { exact: true })).toHaveCount(2, { timeout: 90000 });
      |                                                                ^ Error: expect(locator).toHaveCount(expected) failed
  94  |     await page.screenshot({ path: info.outputPath('agent-shared-review.png') });
  95  |     await view.getByRole('button', { name: 'Adopt as new pipeline', exact: true }).click();
  96  |     await view.getByRole('textbox', { name: 'Pipeline name' }).fill('Read quality acceptance');
  97  |     await view.getByRole('button', { name: 'Confirm creation' }).click();
  98  |     await expect(view.getByText('Created · No Run started.', { exact: true })).toBeVisible({
  99  |       timeout: 45000,
  100 |     });
  101 |     await page
  102 |       .getByRole('textbox', { name: 'Message draft' })
  103 |       .fill('Keep this question for execution preparation.');
  104 |     const catalog = JSON.parse(await readFile(join(profile, 'service/catalog.json'), 'utf8'));
  105 |     expect(catalog.pipelines).toHaveLength(1);
  106 |     expect(catalog.runs).toHaveLength(0);
  107 |     await writeFile(
  108 |       info.outputPath('acceptance.json'),
  109 |       JSON.stringify(
  110 |         {
  111 |           kind: 'real-signed-in-agent',
  112 |           model,
  113 |           effort,
  114 |           profile,
  115 |           project,
  116 |           pipelines: catalog.pipelines,
  117 |           birth: Object.values(catalog.drafts),
  118 |           runs: catalog.runs,
  119 |         },
  120 |         null,
  121 |         2,
  122 |       ),
  123 |     );
  124 |     await application.close();
  125 |     app = undefined;
  126 |     app = await launch(profile, { GOBBLE_CODEX_EXECUTABLE: resolve('node_modules/.bin/codex') });
  127 |     await expect(app.page.getByRole('textbox', { name: 'Message draft' })).toHaveValue(
  128 |       'Keep this question for execution preparation.',
  129 |     );
  130 |     await app.page.getByRole('button', { name: 'Open pipeline', exact: true }).click();
  131 |     await expect(app.page.locator('.creation-view')).toHaveCount(0);
  132 |     await expect(app.page.locator('.pipeline-node')).toHaveCount(3);
  133 |     await app.page.screenshot({ path: info.outputPath('agent-created-current.png') });
  134 |     expect(await readFile(join(project, 'sample.fastq.gz'), 'utf8')).toBe(
  135 |       'Metadata-only isolated acceptance data.',
  136 |     );
  137 |   } catch (error) {
  138 |     if (app) {
  139 |       await app.page
  140 |         .getByRole('button', { name: 'Close Codex account' })
  141 |         .click({ timeout: 1000 })
  142 |         .catch(() => {});
  143 |       await app.page.screenshot({ path: info.outputPath('failure.png') }).catch(() => {});
  144 |       await writeFile(
  145 |         info.outputPath('visible.txt'),
  146 |         await app.page.locator('body').innerText(),
  147 |       ).catch(() => {});
  148 |     }
  149 |     throw error;
  150 |   } finally {
  151 |     await app?.application.close().catch(() => {});
  152 |     await rm(credential, { force: true });
  153 |   }
  154 | });
  155 | 
```