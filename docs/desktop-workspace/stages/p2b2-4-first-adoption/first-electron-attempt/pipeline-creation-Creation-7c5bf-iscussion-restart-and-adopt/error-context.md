# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: pipeline-creation.spec.ts >> Creation: chosen data, exact discussion, restart and adopt
- Location: desktop/tests/electron/pipeline-creation.spec.ts:8:3

# Error details

```
Error: expect(locator).toHaveCount(expected) failed

Locator:  locator('.pipeline-node')
Expected: 2
Received: 3
Timeout:  5000ms

Call log:
  - Expect "toHaveCount" locator('.pipeline-node') with timeout 5000ms
  - waiting for locator('.pipeline-node')
    - locator resolved to 3 elements
    - unexpected value "3"
    - locator resolved to 0 elements
    - unexpected value "0"
    12 × locator resolved to 3 elements
       - unexpected value "3"

```

# Test source

```ts
  94  |         })
  95  |         .not.toBe('');
  96  |       await expect(
  97  |         view.getByRole('button', { name: /Researcher: Quality threshold/ }),
  98  |       ).toBeVisible();
  99  |       await expect(
  100 |         view.getByRole('heading', { name: 'Inspect trimmed read quality', exact: true }),
  101 |       ).toBeVisible();
  102 |       await view.getByRole('button', { name: /Researcher: Quality threshold/ }).click();
  103 |       await page.screenshot({ path: info.outputPath('creation-reviewed.png') });
  104 |       await page
  105 |         .getByRole('textbox', { name: 'Message draft' })
  106 |         .fill('Keep this unsent research question.');
  107 |       await application.close();
  108 |       ({ application, page } = await launch(profile, { GOBBLE_CODEX_EXECUTABLE: executable }));
  109 |       await expect(page.getByRole('textbox', { name: 'Message draft' })).toHaveValue(
  110 |         'Keep this unsent research question.',
  111 |       );
  112 |       await expect(page.getByRole('heading', { name: 'No current version' })).toBeVisible();
  113 |       const catalog = JSON.parse(await readFile(join(profile, 'service/catalog.json'), 'utf8'));
  114 |       expect(catalog.pipelines).toHaveLength(0);
  115 |       expect(catalog.runs).toHaveLength(0);
  116 |       expect(catalog.schemaVersion).toBe(5);
  117 |       const draft = Object.values(catalog.drafts)[0] as {
  118 |         generation: number;
  119 |         input: { resourceId: string };
  120 |       };
  121 |       expect(draft.input.resourceId).toBeTruthy();
  122 |       await page
  123 |         .locator('.creation-view')
  124 |         .getByRole('button', { name: 'Discuss this draft' })
  125 |         .click();
  126 |       await page
  127 |         .locator('.composer-context')
  128 |         .getByRole('button', { name: 'Remove', exact: true })
  129 |         .click();
  130 |       await expect(page.locator('.creation-summary')).toContainText('sample.fastq.gz');
  131 |       await page.setViewportSize({ width: 1000, height: 850 });
  132 |       await page.screenshot({ path: info.outputPath('creation-compact.png') });
  133 |       await page.setViewportSize({ width: 1280, height: 808 });
  134 |       if (finish === 'adopt') {
  135 |         const adoption = page.getByRole('region', { name: 'Creation adoption', exact: true });
  136 |         await adoption.getByRole('button', { name: 'Adopt as new pipeline' }).click();
  137 |         await adoption.getByRole('textbox', { name: 'Pipeline name' }).fill('Read quality study');
  138 |         await expect(adoption).toContainText('sample.fastq.gz');
  139 |         await expect(adoption).toContainText('25 Phred');
  140 |         await page.screenshot({ path: info.outputPath('creation-confirm.png') });
  141 |         await adoption.getByRole('button', { name: 'Confirm creation' }).click();
  142 |         await expect(adoption).toContainText('Created · No Run started.', { timeout: 45000 });
  143 |         await expect(
  144 |           page
  145 |             .getByRole('region', { name: 'Creation adoption' })
  146 |             .getByRole('button', { name: 'Open pipeline' }),
  147 |         ).toBeVisible();
  148 |         await page.screenshot({ path: info.outputPath('creation-adopted.png') });
  149 |         const stored = JSON.parse(await readFile(join(profile, 'service/catalog.json'), 'utf8'));
  150 |         expect(stored.pipelines).toHaveLength(1);
  151 |         expect(stored.runs).toHaveLength(0);
  152 |         const birth = Object.values(stored.drafts)[0] as {
  153 |           draftId: string;
  154 |           adoption: {
  155 |             requestId: string;
  156 |             pipelineId: string;
  157 |             candidateId: string;
  158 |             artifactId: string;
  159 |             generation: number;
  160 |             name: string;
  161 |           };
  162 |         };
  163 |         const current = stored.revisions[birth.adoption.pipelineId];
  164 |         expect(current.creation).toEqual({
  165 |           draftId: birth.draftId,
  166 |           candidateId: birth.adoption.candidateId,
  167 |         });
  168 |         expect(current.artifact.artifactId).toBe(birth.adoption.artifactId);
  169 |         const retry = await page.evaluate(
  170 |           async ({ projectId, draftId, adoption }) =>
  171 |             window.gobble.creation.adopt({
  172 |               projectId,
  173 |               draftId,
  174 |               requestId: adoption.requestId,
  175 |               candidateId: adoption.candidateId,
  176 |               artifactId: adoption.artifactId,
  177 |               expectedGeneration: adoption.generation,
  178 |               name: adoption.name,
  179 |             }),
  180 |           {
  181 |             projectId: stored.pipelines[0].projectId,
  182 |             draftId: birth.draftId,
  183 |             adoption: birth.adoption,
  184 |           },
  185 |         );
  186 |         expect(retry.ok).toBe(true);
  187 |         await application.close();
  188 |         ({ application, page } = await launch(profile, { GOBBLE_CODEX_EXECUTABLE: executable }));
  189 |         await expect(page.getByRole('textbox', { name: 'Message draft' })).toHaveValue(
  190 |           'Keep this unsent research question.',
  191 |         );
  192 |         await expect(page.getByRole('heading', { name: 'No current version' })).toBeVisible();
  193 |         await page.getByRole('button', { name: 'Open pipeline', exact: true }).click();
> 194 |         await expect(page.locator('.pipeline-node')).toHaveCount(2);
      |                                                      ^ Error: expect(locator).toHaveCount(expected) failed
  195 |         await expect(page.getByRole('heading', { name: 'No current version' })).not.toBeVisible();
  196 |         await expect(page.getByRole('textbox', { name: 'Message draft' })).toHaveValue(
  197 |           'Keep this unsent research question.',
  198 |         );
  199 |         await page.screenshot({ path: info.outputPath('creation-current.png') });
  200 |         const final = JSON.parse(await readFile(join(profile, 'service/catalog.json'), 'utf8'));
  201 |         expect(final.pipelines).toHaveLength(1);
  202 |         expect(final.runs).toHaveLength(0);
  203 |         return;
  204 |       }
  205 |       await page
  206 |         .locator('.creation-view')
  207 |         .getByRole('button', { name: 'Discard draft', exact: true })
  208 |         .click();
  209 |       await expect(page.locator('.creation-toolbar')).toContainText('Discarded draft');
  210 |       await expect(page.getByRole('textbox', { name: 'Message draft' })).toHaveValue(
  211 |         'Keep this unsent research question.',
  212 |       );
  213 |       expect(await readFile(join(project, 'sample.fastq.gz'), 'utf8')).toBe(
  214 |         'Metadata-only selected research fixture.',
  215 |       );
  216 |     } catch (error) {
  217 |       await page.screenshot({ path: info.outputPath('failure.png') }).catch(() => {});
  218 |       await writeFile(info.outputPath('visible.txt'), await page.locator('body').innerText()).catch(
  219 |         () => {},
  220 |       );
  221 |       throw error;
  222 |     } finally {
  223 |       await application.close();
  224 |     }
  225 |   });
  226 | 
```