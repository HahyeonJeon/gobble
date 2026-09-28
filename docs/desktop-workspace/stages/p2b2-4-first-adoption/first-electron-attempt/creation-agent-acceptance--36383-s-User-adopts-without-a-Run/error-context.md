# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: creation-agent-acceptance.spec.ts >> Creation: signed-in Agent proposes, reads and points; User adopts without a Run
- Location: desktop/tests/electron/creation-agent-acceptance.spec.ts:10:1

# Error details

```
TimeoutError: locator.inputValue: Timeout 20000ms exceeded.
Call log:
  - waiting for getByRole('dialog', { name: 'Add agent', exact: true }).getByLabel('Model', { exact: true })

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
> 51  |     const model = await editor.getByLabel('Model', { exact: true }).inputValue();
      |                                                                     ^ TimeoutError: locator.inputValue: Timeout 20000ms exceeded.
  52  |     const effort = await editor.getByLabel('Reasoning effort', { exact: true }).inputValue();
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
  82  |         'Read the exact attached proposal setting, briefly explain what this quality threshold means, and point to it in the shared flow using the creation review and point tools. This is discussion only; keep the proposal unchanged.',
  83  |       );
  84  |     await expect(page.getByRole('button', { name: 'Send', exact: true })).toBeEnabled({
  85  |       timeout: 90000,
  86  |     });
  87  |     await page.getByRole('button', { name: 'Send', exact: true }).click();
  88  |     await expect(view.getByRole('button', { name: /Researcher:/ })).toBeVisible({
  89  |       timeout: 180000,
  90  |     });
  91  |     await page.screenshot({ path: info.outputPath('agent-shared-review.png') });
  92  |     await view.getByRole('button', { name: 'Adopt as new pipeline', exact: true }).click();
  93  |     await view.getByRole('textbox', { name: 'Pipeline name' }).fill('Read quality acceptance');
  94  |     await view.getByRole('button', { name: 'Confirm creation' }).click();
  95  |     await expect(view.getByText('Created · No Run started.', { exact: true })).toBeVisible({
  96  |       timeout: 45000,
  97  |     });
  98  |     await page
  99  |       .getByRole('textbox', { name: 'Message draft' })
  100 |       .fill('Keep this question for execution preparation.');
  101 |     const catalog = JSON.parse(await readFile(join(profile, 'service/catalog.json'), 'utf8'));
  102 |     expect(catalog.pipelines).toHaveLength(1);
  103 |     expect(catalog.runs).toHaveLength(0);
  104 |     await writeFile(
  105 |       info.outputPath('acceptance.json'),
  106 |       JSON.stringify(
  107 |         {
  108 |           kind: 'real-signed-in-agent',
  109 |           model,
  110 |           effort,
  111 |           profile,
  112 |           project,
  113 |           pipelines: catalog.pipelines,
  114 |           birth: Object.values(catalog.drafts),
  115 |           runs: catalog.runs,
  116 |         },
  117 |         null,
  118 |         2,
  119 |       ),
  120 |     );
  121 |     await application.close();
  122 |     app = undefined;
  123 |     app = await launch(profile, { GOBBLE_CODEX_EXECUTABLE: resolve('node_modules/.bin/codex') });
  124 |     await expect(app.page.getByRole('textbox', { name: 'Message draft' })).toHaveValue(
  125 |       'Keep this question for execution preparation.',
  126 |     );
  127 |     await app.page.getByRole('button', { name: 'Open pipeline', exact: true }).click();
  128 |     await expect(app.page.locator('.pipeline-node')).toHaveCount(2);
  129 |     await app.page.screenshot({ path: info.outputPath('agent-created-current.png') });
  130 |     expect(await readFile(join(project, 'sample.fastq.gz'), 'utf8')).toBe(
  131 |       'Metadata-only isolated acceptance data.',
  132 |     );
  133 |   } catch (error) {
  134 |     if (app) {
  135 |       await app.page
  136 |         .getByRole('button', { name: 'Close Codex account' })
  137 |         .click({ timeout: 1000 })
  138 |         .catch(() => {});
  139 |       await app.page.screenshot({ path: info.outputPath('failure.png') }).catch(() => {});
  140 |       await writeFile(
  141 |         info.outputPath('visible.txt'),
  142 |         await app.page.locator('body').innerText(),
  143 |       ).catch(() => {});
  144 |     }
  145 |     throw error;
  146 |   } finally {
  147 |     await app?.application.close().catch(() => {});
  148 |     await rm(credential, { force: true });
  149 |   }
  150 | });
  151 | 
```