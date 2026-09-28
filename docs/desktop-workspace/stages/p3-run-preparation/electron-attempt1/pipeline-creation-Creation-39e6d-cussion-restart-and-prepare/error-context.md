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

Locator: locator('.review-context')
Expected substring: "Run review · settings"
Error: strict mode violation: locator('.review-context') resolved to 2 elements:
    1) <div class="review-context">…</div> aka locator('div').filter({ hasText: /^New pipeline · Selected data · Proposal allowed$/ })
    2) <div class="review-context">…</div> aka locator('div').filter({ hasText: /^Proposed addition · Quality threshold · 25 Phred · Discussion only$/ })

Call log:
  - Expect "toContainText" locator('.review-context') with timeout 5000ms
  - waiting for locator('.review-context')

```

# Test source

```ts
  126 |       expect(catalog.schemaVersion).toBe(5);
  127 |       const draft = Object.values(catalog.drafts)[0] as {
  128 |         generation: number;
  129 |         input: { resourceId: string };
  130 |       };
  131 |       expect(draft.input.resourceId).toBeTruthy();
  132 |       await page
  133 |         .locator('.creation-view')
  134 |         .getByRole('button', { name: 'Discuss this draft' })
  135 |         .click();
  136 |       await page
  137 |         .locator('.composer-context')
  138 |         .getByRole('button', { name: 'Remove', exact: true })
  139 |         .click();
  140 |       await expect(page.locator('.creation-summary')).toContainText('sample.fastq.gz');
  141 |       await page.setViewportSize({ width: 1000, height: 850 });
  142 |       await page.screenshot({ path: info.outputPath('creation-compact.png') });
  143 |       await page.setViewportSize({ width: 1280, height: 808 });
  144 |       if (finish !== 'discard') {
  145 |         const adoption = page.getByRole('region', { name: 'Creation adoption', exact: true });
  146 |         await adoption.getByRole('button', { name: 'Adopt as new pipeline' }).click();
  147 |         await adoption.getByRole('textbox', { name: 'Pipeline name' }).fill('Read quality study');
  148 |         await expect(adoption).toContainText('sample.fastq.gz');
  149 |         await expect(adoption).toContainText('25 Phred');
  150 |         await page.screenshot({ path: info.outputPath('creation-confirm.png') });
  151 |         await adoption.getByRole('button', { name: 'Confirm creation' }).click();
  152 |         await expect(adoption).toContainText('Created · No Run started.', { timeout: 45000 });
  153 |         await expect(
  154 |           page
  155 |             .getByRole('region', { name: 'Creation adoption' })
  156 |             .getByRole('button', { name: 'Open pipeline' }),
  157 |         ).toBeVisible();
  158 |         await page.screenshot({ path: info.outputPath('creation-adopted.png') });
  159 |         const stored = JSON.parse(await readFile(join(profile, 'service/catalog.json'), 'utf8'));
  160 |         expect(stored.pipelines).toHaveLength(1);
  161 |         expect(stored.runs).toHaveLength(0);
  162 |         const birth = Object.values(stored.drafts)[0] as {
  163 |           draftId: string;
  164 |           adoption: {
  165 |             requestId: string;
  166 |             pipelineId: string;
  167 |             candidateId: string;
  168 |             artifactId: string;
  169 |             generation: number;
  170 |             name: string;
  171 |           };
  172 |         };
  173 |         const current = stored.revisions[birth.adoption.pipelineId];
  174 |         expect(current.creation).toEqual({
  175 |           draftId: birth.draftId,
  176 |           candidateId: birth.adoption.candidateId,
  177 |         });
  178 |         expect(current.artifact.artifactId).toBe(birth.adoption.artifactId);
  179 |         const retry = await page.evaluate(
  180 |           async ({ projectId, draftId, adoption }) =>
  181 |             window.gobble.creation.adopt({
  182 |               projectId,
  183 |               draftId,
  184 |               requestId: adoption.requestId,
  185 |               candidateId: adoption.candidateId,
  186 |               artifactId: adoption.artifactId,
  187 |               expectedGeneration: adoption.generation,
  188 |               name: adoption.name,
  189 |             }),
  190 |           {
  191 |             projectId: stored.pipelines[0].projectId,
  192 |             draftId: birth.draftId,
  193 |             adoption: birth.adoption,
  194 |           },
  195 |         );
  196 |         expect(retry.ok).toBe(true);
  197 |         await application.close();
  198 |         ({ application, page } = await launch(profile, { GOBBLE_CODEX_EXECUTABLE: executable }));
  199 |         await expect(page.getByRole('textbox', { name: 'Message draft' })).toHaveValue(
  200 |           'Keep this unsent research question.',
  201 |         );
  202 |         await expect(page.getByRole('heading', { name: 'No current version' })).toBeVisible();
  203 |         await page.getByRole('button', { name: 'Open pipeline', exact: true }).click();
  204 |         await expect(page.locator('.pipeline-node')).toHaveCount(3);
  205 |         await expect(page.getByRole('heading', { name: 'No current version' })).not.toBeVisible();
  206 |         await expect(page.getByRole('textbox', { name: 'Message draft' })).toHaveValue(
  207 |           'Keep this unsent research question.',
  208 |         );
  209 |         await page.screenshot({ path: info.outputPath('creation-current.png') });
  210 |         if (finish === 'prepare') {
  211 |           await expect(page.locator('.creation-view')).toHaveCount(0);
  212 |           const review = page.getByRole('region', { name: 'Run preparation' });
  213 |           await page
  214 |             .getByRole('combobox', { name: 'Preparation engine', exact: true })
  215 |             .selectOption(process.env.GOBBLE_PREPARATION_IMAGE!);
  216 |           await review.getByRole('button', { name: 'Prepare run', exact: true }).click();
  217 |           await expect(
  218 |             review.getByText('Plan saved for review · No Run started', { exact: true }),
  219 |           ).toBeVisible({ timeout: 150000 });
  220 |           await expect(review).toContainText('25 Phred');
  221 |           await expect(review).toContainText('sample.fastq.gz');
  222 |           await expect(review).toContainText(
  223 |             'Data contents and FASTQ validity have not been checked or copied.',
  224 |           );
  225 |           await review.getByRole('button', { name: 'Discuss settings', exact: true }).click();
> 226 |           await expect(page.locator('.review-context')).toContainText('Run review · settings');
      |                                                         ^ Error: expect(locator).toContainText(expected) failed
  227 |           const message = page.getByRole('textbox', { name: 'Message draft' });
  228 |           await message.fill('pipeline-preparation-review');
  229 |           await application.evaluate(({ app, BrowserWindow }) => {
  230 |             app.focus({ steal: true });
  231 |             BrowserWindow.getAllWindows()[0]?.focus();
  232 |           });
  233 |           await page.getByRole('button', { name: 'Send', exact: true }).click();
  234 |           await expect(review.locator('.preparation-mark')).toContainText(
  235 |             'This saved plan uses the reviewed quality threshold.',
  236 |             { timeout: 30000 },
  237 |           );
  238 |           await message.fill('Keep discussing these settings.');
  239 |           await page.screenshot({ path: info.outputPath('preparation-review.png') });
  240 |           const saved = JSON.parse(
  241 |             await readFile(
  242 |               join(profile, 'workspace/projects', stored.pipelines[0].projectId + '.json'),
  243 |               'utf8',
  244 |             ).catch(
  245 |               async () =>
  246 |                 await readFile(
  247 |                   join(profile, 'workspaces/projects', stored.pipelines[0].projectId + '.json'),
  248 |                   'utf8',
  249 |                 ),
  250 |             ),
  251 |           );
  252 |           await writeFile(
  253 |             info.outputPath('preparation-workspace.json'),
  254 |             JSON.stringify(saved, null, 2),
  255 |           );
  256 |           const listed = await page.evaluate(
  257 |             async (input) => window.gobble.preparations.list(input),
  258 |             { projectId: stored.pipelines[0].projectId, pipelineId: birth.adoption.pipelineId },
  259 |           );
  260 |           expect(listed.ok).toBe(true);
  261 |           if (!listed.ok) throw Error('Missing preparation');
  262 |           expect(listed.value).toHaveLength(1);
  263 |           expect(JSON.stringify(listed)).not.toContain('"payload"');
  264 |           const exact = listed.value[0]!;
  265 |           const retry = await page.evaluate(async (v) => window.gobble.preparations.prepare(v), {
  266 |             projectId: exact.projectId,
  267 |             pipelineId: exact.pipelineId,
  268 |             requestId: exact.requestId,
  269 |             artifactId: exact.artifactId,
  270 |             engineId: exact.engineId,
  271 |           });
  272 |           expect(retry.ok).toBe(true);
  273 |           await writeFile(
  274 |             info.outputPath('preparation-result.json'),
  275 |             JSON.stringify(listed, null, 2),
  276 |           );
  277 |           await application.close();
  278 |           ({ application, page } = await launch(profile, { GOBBLE_CODEX_EXECUTABLE: executable }));
  279 |           await expect(page.getByRole('textbox', { name: 'Message draft' })).toHaveValue(
  280 |             'Keep discussing these settings.',
  281 |           );
  282 |           await page.getByRole('button', { name: '▸ Run review', exact: true }).click();
  283 |           await expect(page.locator('.preparation-mark')).toContainText(
  284 |             'This saved plan uses the reviewed quality threshold.',
  285 |           );
  286 |           await writeFile(
  287 |             join(project, 'sample.fastq.gz'),
  288 |             'Changed metadata fixture after review.',
  289 |           );
  290 |           await expect(page.getByRole('region', { name: 'Run preparation' })).toContainText(
  291 |             'This review belongs to an earlier observation.',
  292 |             { timeout: 15000 },
  293 |           );
  294 |           await page.screenshot({ path: info.outputPath('preparation-stale.png') });
  295 |           await page
  296 |             .getByRole('region', { name: 'Run preparation' })
  297 |             .getByRole('button', { name: 'Prepare again', exact: true })
  298 |             .click();
  299 |           await expect(
  300 |             page
  301 |               .getByRole('region', { name: 'Run preparation' })
  302 |               .getByText('Prepared · No Run started', { exact: true }),
  303 |           ).toBeVisible({ timeout: 150000 });
  304 |           await expect(
  305 |             page.getByRole('combobox', { name: 'Run review history' }).locator('option'),
  306 |           ).toHaveCount(2);
  307 |           await expect(page.locator('.review-context')).toContainText('Run review · settings');
  308 |           const finalReviews = await page.evaluate(
  309 |             async (v) => window.gobble.preparations.list(v),
  310 |             { projectId: exact.projectId, pipelineId: exact.pipelineId },
  311 |           );
  312 |           expect(finalReviews.ok).toBe(true);
  313 |           if (!finalReviews.ok) throw Error('Missing history');
  314 |           expect(finalReviews.value.filter((v) => v.state === 'ready' && v.fresh)).toHaveLength(1);
  315 |           await writeFile(
  316 |             info.outputPath('preparation-history.json'),
  317 |             JSON.stringify(finalReviews, null, 2),
  318 |           );
  319 |           await page.screenshot({ path: info.outputPath('preparation-new-review.png') });
  320 |         }
  321 | 
  322 |         const final = JSON.parse(await readFile(join(profile, 'service/catalog.json'), 'utf8'));
  323 |         expect(final.pipelines).toHaveLength(1);
  324 |         expect(final.runs).toHaveLength(0);
  325 |         return;
  326 |       }
```