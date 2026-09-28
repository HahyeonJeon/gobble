# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: pipeline-creation.spec.ts >> Creation: chosen data, exact discussion, restart and prepare
- Location: desktop/tests/electron/pipeline-creation.spec.ts:8:3

# Error details

```
Error: expect(locator).toContainText(expected) failed

Locator: locator('.chat-composer .review-context')
Expected substring: "Run review · settings"
Timeout: 5000ms
Error: element(s) not found

Call log:
  - Expect "toContainText" locator('.chat-composer .review-context') with timeout 5000ms
  - waiting for locator('.chat-composer .review-context')

```

```yaml
- link "Skip to workspace":
  - /url: "#workspace-main"
- banner:
  - button "Toggle files" [expanded]
  - heading "Read quality study" [level=1]:
    - button "Switch Project": Read quality study
  - button "Single pane" [pressed]
  - button "Two panes"
  - button "Project selections": Selections 0
  - button "Account"
- complementary "Project navigation":
  - group "Pipelines":
    - text: › Pipelines 1
    - button "New pipeline": ＋ New pipeline
    - list:
      - listitem:
        - button "Open Read quality study pipeline": Read quality study
        - button "Open Read quality study pipeline in the other pane"
    - button "Import pipeline"
  - group:
    - text: › Files
    - button "Project files" [disabled]
    - button "Refresh files"
    - list:
      - listitem:
        - button "sample.fastq.gz"
        - button "Open sample.fastq.gz in the other pane"
  - group:
    - text: › Runs Existing analyses
    - button "Refresh runs"
    - list
    - paragraph: No runs attached
- status: Workspace ready
- main:
  - region "Primary pane":
    - tablist "Primary views":
      - tab "New pipeline"
      - tab "Read quality study" [selected]
    - button "More actions for Read quality study": More
    - button "Close Read quality study"
    - tabpanel "Read quality study":
      - strong: Pipeline flow
      - text: Managed current · Checked Sep 12, 09:42 PM
      - button "Changes"
      - button "Discuss changes"
      - button "Refresh pipeline view": Refresh
      - button "Check flow" [disabled]
      - text: 2 steps · 1 input · 2 connections
      - group "Flow representation":
        - button "Flow" [pressed]
        - button "Step list"
      - group "Flow zoom":
        - button "Fit pipeline flow": Fit
        - button "Zoom out pipeline flow": −
        - status "Flow zoom level": 76%
        - button "Zoom in pipeline flow": +
      - img "Pipeline connections":
        - button "Pipeline input · reads → Trim adapters and low-quality bases · read1"
        - button "Trim adapters and low-quality bases · trimmed_read1 → Inspect trimmed read quality · reads"
      - button "Inspect reads · file":
        - strong: reads
        - text: file
      - button "Inspect Trim adapters and low-quality bases · 1 input · 2 outputs":
        - strong: Trim adapters and low-quality bases
        - text: 1 input · 2 outputs
      - button "Inspect Inspect trimmed read quality · 1 input · 2 outputs":
        - strong: Inspect trimmed read quality
        - text: 1 input · 2 outputs
      - text: Input Step Selected Design view · select to inspect
      - region "Run preparation":
        - button "▾ Run review" [expanded]
        - text: Prepared · No Run started
        - combobox "Preparation engine":
          - option "Local engine · 9145e5f6ecc2" [selected]
        - button "Refresh preparation engines": ↻
        - button "Prepare again"
        - paragraph: Review this flow, its selected data and analysis environment together. Preparing does not start the analysis.
        - text: Review history
        - combobox "Run review history":
          - option "9:42:58 PM · ready"
          - option "9:44:09 PM · ready" [selected]
        - group "Run review section":
          - button "Data"
          - button "Settings" [pressed]
          - button "Environment"
        - article:
          - heading "Settings 2 steps" [level=3]
          - strong: Trim adapters and low-quality bases
          - paragraph: Quality threshold 25 Phred
          - paragraph: Minimum length 40 bp
          - strong: Inspect trimmed read quality
          - button "Discuss settings"
        - group: Expected results
        - paragraph: Plan saved for review · No Run started
- separator "Resize chat"
- region "Project chat":
  - heading "Chat" [level=2]
  - button "Project agents": Agents 1
  - button "Collapse chat" [expanded]
  - text: Opened a Project view.
  - article:
    - strong: You → Researcher
    - time: 09:41 PM
    - text: New pipeline · Selected data · Proposal allowed
    - paragraph: pipeline-creation
  - article:
    - strong: Researcher
    - 'button "In reply to: pipeline-creation"'
    - paragraph: Fixture response from Researcher.
    - text: completed
  - article:
    - strong: You → Researcher
    - time: 09:42 PM
    - text: Proposed addition · Quality threshold · 25 Phred · Discussion only
    - paragraph: pipeline-creation-review
  - article:
    - strong: Researcher
    - 'button "In reply to: pipeline-creation-review"'
    - paragraph: Fixture response from Researcher.
    - text: completed
  - text: Opened a Project view.
  - article:
    - strong: You → Researcher
    - time: 09:44 PM
    - text: Run review · settings · Discussion only
    - button "Discuss this version"
    - paragraph: pipeline-preparation-review
  - article:
    - strong: Researcher
    - 'button "In reply to: pipeline-preparation-review"'
    - paragraph: Fixture response from Researcher.
    - text: completed
  - textbox "Message draft":
    - /placeholder: Write a message for your agent…
    - text: Keep discussing these settings.
  - text: Conversation recipient
  - combobox "Conversation recipient":
    - option "Choose recipient"
    - option "To Researcher" [selected]
  - text: Message model
  - combobox "Message model" [disabled]:
    - option "fixture-model · unavailable" [selected]
  - button "Send" [disabled]
  - text: Draft saved locally Enter to send · Shift+Enter for a new line
```

# Test source

```ts
  220 |           await expect(review).toContainText('25 Phred');
  221 |           await review.getByRole('button', { name: 'Data', exact: true }).click();
  222 |           await expect(review.locator('.preparation-file')).toBeVisible();
  223 |           await expect(review).toContainText('sample.fastq.gz');
  224 |           await expect(review).toContainText(
  225 |             'Data contents and FASTQ validity have not been checked or copied.',
  226 |           );
  227 |           await review.getByRole('button', { name: 'Environment', exact: true }).click();
  228 |           await expect(
  229 |             review.getByText('One processing task at a time.', { exact: true }),
  230 |           ).toBeVisible();
  231 |           await review.getByRole('button', { name: 'Settings', exact: true }).click();
  232 |           await review.getByRole('button', { name: 'Discuss settings', exact: true }).click();
  233 |           await expect(page.locator('.chat-composer .review-context')).toContainText(
  234 |             'Run review · settings',
  235 |           );
  236 |           const message = page.getByRole('textbox', { name: 'Message draft' });
  237 |           await page.getByRole('button', { name: 'Account', exact: true }).click();
  238 |           await page.getByRole('button', { name: 'Refresh connection', exact: true }).click();
  239 |           await expect(page.getByText('ChatGPT connected', { exact: true })).toBeVisible();
  240 |           await page.getByRole('button', { name: 'Close Codex account', exact: true }).click();
  241 |           await message.fill('pipeline-preparation-review');
  242 |           await application.evaluate(({ app, BrowserWindow }) => {
  243 |             app.focus({ steal: true });
  244 |             BrowserWindow.getAllWindows()[0]?.focus();
  245 |           });
  246 |           await page.getByRole('button', { name: 'Send', exact: true }).click();
  247 |           await expect(review.locator('.preparation-mark')).toContainText(
  248 |             'This saved plan uses the reviewed quality threshold.',
  249 |             { timeout: 30000 },
  250 |           );
  251 |           await message.fill('Keep discussing these settings.');
  252 |           await page.screenshot({ path: info.outputPath('preparation-review.png') });
  253 |           const saved = JSON.parse(
  254 |             await readFile(
  255 |               join(profile, 'workspace/projects', stored.pipelines[0].projectId + '.json'),
  256 |               'utf8',
  257 |             ).catch(
  258 |               async () =>
  259 |                 await readFile(
  260 |                   join(profile, 'workspaces/projects', stored.pipelines[0].projectId + '.json'),
  261 |                   'utf8',
  262 |                 ),
  263 |             ),
  264 |           );
  265 |           await writeFile(
  266 |             info.outputPath('preparation-workspace.json'),
  267 |             JSON.stringify(saved, null, 2),
  268 |           );
  269 |           const listed = await page.evaluate(
  270 |             async (input) => window.gobble.preparations.list(input),
  271 |             { projectId: stored.pipelines[0].projectId, pipelineId: birth.adoption.pipelineId },
  272 |           );
  273 |           expect(listed.ok).toBe(true);
  274 |           if (!listed.ok) throw Error('Missing preparation');
  275 |           expect(listed.value).toHaveLength(1);
  276 |           expect(JSON.stringify(listed)).not.toContain('"payload"');
  277 |           const exact = listed.value[0]!;
  278 |           const retry = await page.evaluate(async (v) => window.gobble.preparations.prepare(v), {
  279 |             projectId: exact.projectId,
  280 |             pipelineId: exact.pipelineId,
  281 |             requestId: exact.requestId,
  282 |             artifactId: exact.artifactId,
  283 |             engineId: exact.engineId,
  284 |           });
  285 |           expect(retry.ok).toBe(true);
  286 |           await writeFile(
  287 |             info.outputPath('preparation-result.json'),
  288 |             JSON.stringify(listed, null, 2),
  289 |           );
  290 |           await application.close();
  291 |           ({ application, page } = await launch(profile, { GOBBLE_CODEX_EXECUTABLE: executable }));
  292 |           await expect(page.getByRole('textbox', { name: 'Message draft' })).toHaveValue(
  293 |             'Keep discussing these settings.',
  294 |           );
  295 |           await page.getByRole('button', { name: '▸ Run review', exact: true }).click();
  296 |           await expect(page.locator('.preparation-mark')).toContainText(
  297 |             'This saved plan uses the reviewed quality threshold.',
  298 |           );
  299 |           await writeFile(
  300 |             join(project, 'sample.fastq.gz'),
  301 |             'Changed metadata fixture after review.',
  302 |           );
  303 |           await expect(page.getByRole('region', { name: 'Run preparation' })).toContainText(
  304 |             'This review belongs to an earlier observation.',
  305 |             { timeout: 15000 },
  306 |           );
  307 |           await page.screenshot({ path: info.outputPath('preparation-stale.png') });
  308 |           await page
  309 |             .getByRole('region', { name: 'Run preparation' })
  310 |             .getByRole('button', { name: 'Prepare again', exact: true })
  311 |             .click();
  312 |           await expect(
  313 |             page
  314 |               .getByRole('region', { name: 'Run preparation' })
  315 |               .getByText('Prepared · No Run started', { exact: true }),
  316 |           ).toBeVisible({ timeout: 150000 });
  317 |           await expect(
  318 |             page.getByRole('combobox', { name: 'Run review history' }).locator('option'),
  319 |           ).toHaveCount(2);
> 320 |           await expect(page.locator('.chat-composer .review-context')).toContainText(
      |                                                                        ^ Error: expect(locator).toContainText(expected) failed
  321 |             'Run review · settings',
  322 |           );
  323 |           const finalReviews = await page.evaluate(
  324 |             async (v) => window.gobble.preparations.list(v),
  325 |             { projectId: exact.projectId, pipelineId: exact.pipelineId },
  326 |           );
  327 |           expect(finalReviews.ok).toBe(true);
  328 |           if (!finalReviews.ok) throw Error('Missing history');
  329 |           expect(finalReviews.value.filter((v) => v.state === 'ready' && v.fresh)).toHaveLength(1);
  330 |           await writeFile(
  331 |             info.outputPath('preparation-history.json'),
  332 |             JSON.stringify(finalReviews, null, 2),
  333 |           );
  334 |           await page.screenshot({ path: info.outputPath('preparation-new-review.png') });
  335 |         }
  336 | 
  337 |         const final = JSON.parse(await readFile(join(profile, 'service/catalog.json'), 'utf8'));
  338 |         expect(final.pipelines).toHaveLength(1);
  339 |         expect(final.runs).toHaveLength(0);
  340 |         return;
  341 |       }
  342 |       await page
  343 |         .locator('.creation-view')
  344 |         .getByRole('button', { name: 'Discard draft', exact: true })
  345 |         .click();
  346 |       await expect(page.locator('.creation-toolbar')).toContainText('Discarded draft');
  347 |       await expect(page.getByRole('textbox', { name: 'Message draft' })).toHaveValue(
  348 |         'Keep this unsent research question.',
  349 |       );
  350 |       expect(await readFile(join(project, 'sample.fastq.gz'), 'utf8')).toBe(
  351 |         'Metadata-only selected research fixture.',
  352 |       );
  353 |     } catch (error) {
  354 |       await page.screenshot({ path: info.outputPath('failure.png') }).catch(() => {});
  355 |       await writeFile(info.outputPath('visible.txt'), await page.locator('body').innerText()).catch(
  356 |         () => {},
  357 |       );
  358 |       throw error;
  359 |     } finally {
  360 |       await application.close();
  361 |     }
  362 |   });
  363 | 
```