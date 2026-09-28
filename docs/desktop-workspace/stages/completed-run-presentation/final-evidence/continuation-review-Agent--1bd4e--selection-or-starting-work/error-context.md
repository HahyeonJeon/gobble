# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: continuation-review.spec.ts >> Agent points to the same continuation plan without changing User selection or starting work
- Location: desktop/tests/electron/continuation-review.spec.ts:179:1

# Error details

```
Error: expect(locator).toBeVisible() failed

Locator: locator('.shared-reference-event').filter({ hasText: 'This checked step restarts from its beginning.' })
Expected: visible
Timeout: 5000ms
Error: element(s) not found

Call log:
  - Expect "toBeVisible" locator('.shared-reference-event').filter({ hasText: 'This checked step restarts from its beginning.' }) with timeout 5000ms
  - waiting for locator('.shared-reference-event').filter({ hasText: 'This checked step restarts from its beginning.' })

```

```yaml
- link "Skip to workspace":
  - /url: "#workspace-main"
- banner:
  - button "Toggle files" [expanded]
  - heading "Continuation study" [level=1]:
    - button "Switch Project": Continuation study
  - button "Single pane" [pressed]
  - button "Two panes"
  - button "Project selections": Selections 1
  - button "Account"
- complementary "Project navigation":
  - group "Pipelines":
    - text: › Pipelines 1
    - button "New pipeline": ＋ New pipeline
    - list:
      - listitem:
        - button "Open Read quality analysis pipeline": Read quality analysis
        - button "Open Read quality analysis pipeline in the other pane"
    - button "Import pipeline"
  - group:
    - text: › Files
    - button "Project files" [disabled]
    - button "Refresh files"
    - list:
      - listitem:
        - button "reads.fastq.gz"
        - button "Open reads.fastq.gz in the other pane"
      - listitem:
        - button "runs"
  - group:
    - text: › Runs Existing analyses
    - button "Refresh runs"
    - list:
      - listitem:
        - button "Analysis 1"
- status: Workspace ready
- main:
  - region "Primary pane":
    - tablist "Primary views":
      - tab "Analysis 1" [selected]
    - button "More actions for Analysis 1": More
    - button "Close Analysis 1"
    - tabpanel "Analysis 1":
      - region "Run flow":
        - strong: Run flow
        - text: stopped · Saved design
        - group "Flow zoom":
          - button "Fit pipeline flow": Fit
          - button "Zoom out pipeline flow": −
          - status "Flow zoom level": 64%
          - button "Zoom in pipeline flow": +
        - img "Pipeline connections"
        - button "Inspect reads · file" [disabled]:
          - strong: reads
          - text: file
        - button "Inspect Trim reads · 1 input · 1 output · Can reuse · Attempt 1" [pressed]:
          - strong: Trim reads
          - text: Can reuse · Attempt 1
        - button "Inspect Quality check · 1 input · 1 output · Will restart · Attempt 2":
          - strong: Quality check
          - text: Will restart · Attempt 2
        - paragraph: This analysis used an earlier design. Its saved Flow and evidence remain unchanged.
        - button "Open current design"
        - strong: Continue this analysis
        - status: Review the kept and restarted steps. Confirm in Chat when ready.
        - button "Recheck continuation"
        - paragraph:
          - strong: Trim reads · Can reuse · Attempt 1
          - text: The checked result is kept.
        - text: Saved design and checked data · Review bd102ec9 · Captured plan, not live status
        - button "Discuss this step"
      - group "Run view":
        - button "Tasks" [pressed]
        - button "Dependencies"
      - status: Observed · 5:42:06 PM
      - strong: Tasks
      - searchbox "Search tasks"
      - combobox "Task state":
        - option "All states" [selected]
        - option "succeeded"
        - option "canceled"
      - button "Refresh"
      - table:
        - rowgroup:
          - row "Task instance State Attempt":
            - columnheader "Task instance"
            - columnheader "State"
            - columnheader "Attempt"
        - rowgroup:
          - row "Trim reads · trim · Attempt 1 Trim reads trim succeeded 1":
            - cell "Trim reads · trim · Attempt 1 Trim reads trim":
              - radio "Trim reads · trim · Attempt 1" [checked]
              - strong: Trim reads
              - text: trim
            - cell "succeeded"
            - cell "1"
          - row "Quality check · qc · Attempt 1 Quality check qc canceled 1":
            - cell "Quality check · qc · Attempt 1 Quality check qc":
              - radio "Quality check · qc · Attempt 1"
              - strong: Quality check
              - text: qc
            - cell "canceled"
            - cell "1"
      - strong: trim · Attempt 1
      - text: Selected task instance
      - button "Open logs"
      - button "Discuss task"
      - group: 2 shown · 2 loaded · Run details
- separator "Resize chat"
- region "Project chat":
  - heading "Chat" [level=2]
  - button "Project agents": Agents 1
  - button "Collapse chat" [expanded]
  - text: Opened a Project view.
  - article:
    - strong: You → Researcher
    - time: 05:42 PM
    - paragraph: continuation-point
  - article:
    - strong: Researcher
    - 'button "In reply to: continuation-point"'
    - paragraph: "Fixture response from Researcher Fixture shared tool failed: Shared review context missing"
    - text: failed
    - paragraph: Codex could not finish this message. Check your account before trying a new message.
  - article "Continue analysis":
    - strong: Resume this analysis?
    - paragraph:
      - strong: Keep 1 result · Restart 1 step
    - list:
      - listitem: Trim reads · Can reuse · Attempt 1
      - listitem: Quality check · Will restart · Attempt 2
    - paragraph: Uses this Run’s saved design and checked data. Restarted steps run from their beginning.
    - text: Gobble checks this plan again before starting.
    - paragraph: reads.fastq.gz
    - button "Resume analysis"
    - button "Open Run"
  - textbox "Message draft":
    - /placeholder: Write a message for your agent…
    - text: Keep my next question here.
  - text: Conversation recipient
  - combobox "Conversation recipient":
    - option "Choose recipient"
    - option "To Researcher" [selected]
  - text: Message model
  - combobox "Message model":
    - option "Fixture model" [selected]
    - option "Fixture text model"
  - button "Send"
  - text: Draft saved locally Enter to send · Shift+Enter for a new line
```

# Test source

```ts
  118 |       flow.getByRole('button', { name: 'Check continuation', exact: true }),
  119 |     ).toBeDisabled();
  120 |     await unlink(join(fixture.base, 'old-engine'));
  121 |     await page.getByRole('button', { name: 'Close Analysis 1', exact: true }).click();
  122 |     await page.getByRole('button', { name: 'Open Run', exact: true }).click();
  123 |     await expect(
  124 |       flow.getByRole('button', { name: 'Check continuation', exact: true }),
  125 |     ).toBeEnabled();
  126 |     await flow.getByRole('button', { name: 'Check continuation', exact: true }).click();
  127 |     await expect(flow.locator('[data-continuation="restart"]')).toHaveCount(1, { timeout: 20000 });
  128 |     await flow.getByRole('button', { name: /Inspect Quality check/ }).click();
  129 |     await flow.getByRole('button', { name: 'Discuss this step' }).click();
  130 |     await writeFile(join(fixture.base, 'blocked'), 'true');
  131 |     await flow.getByRole('button', { name: 'Recheck continuation', exact: true }).click();
  132 |     await expect(page.getByRole('article', { name: 'Continue analysis' })).toContainText(
  133 |       'Continuation needs review',
  134 |       { timeout: 20000 },
  135 |     );
  136 |     await expect(page.getByRole('button', { name: 'Resume analysis' })).toHaveCount(0);
  137 |     await page.locator('.attachment-chip').getByRole('button').first().click();
  138 |     const preview = page.getByRole('dialog', { name: /^Attachment:/ });
  139 |     await expect(preview).toContainText('Will restart · Attempt 2');
  140 |     await expect(preview.getByRole('button', { name: 'Resume analysis' })).toHaveCount(0);
  141 |     await page.screenshot({ path: info.outputPath('historical-review-after-recheck.png') });
  142 |     await page.keyboard.press('Escape');
  143 |     await unlink(join(fixture.base, 'blocked'));
  144 |     await flow.getByRole('button', { name: 'Recheck continuation', exact: true }).click();
  145 |     await expect(page.getByRole('button', { name: 'Resume analysis' })).toBeEnabled({
  146 |       timeout: 20000,
  147 |     });
  148 |     await writeFile(join(fixture.base, 'lose-ack'), 'true');
  149 |     await page.getByRole('button', { name: 'Resume analysis' }).click();
  150 |     await expect(page.getByRole('article', { name: 'Continue analysis' })).toContainText(
  151 |       'acknowledgement is unavailable',
  152 |       { timeout: 20000 },
  153 |     );
  154 |     await expect(
  155 |       flow.getByRole('button', { name: 'Recheck continuation', exact: true }),
  156 |     ).toBeDisabled();
  157 |     await expect(page.getByRole('button', { name: 'Resume analysis' })).toHaveCount(0);
  158 |     await page.getByRole('button', { name: 'Check continuation status' }).click();
  159 |     await expect.poll(creates).toHaveLength(1);
  160 |     await expect(page.getByRole('button', { name: 'Check continuation status' })).toBeEnabled();
  161 |     await readyView(page);
  162 |     await page.screenshot({ path: info.outputPath('continuation-unknown.png') });
  163 |     await application.close();
  164 |     ({ application, page } = await launch(fixture.profile, fixture.environment));
  165 |     await application.evaluate(({ BrowserWindow }) =>
  166 |       BrowserWindow.getAllWindows()[0]!.setSize(1440, 950),
  167 |     );
  168 |     await expect(page.getByRole('article', { name: 'Continue analysis' })).toContainText(
  169 |       'acknowledgement is unavailable',
  170 |     );
  171 |     await page.getByRole('button', { name: 'Check continuation status' }).click();
  172 |     await expect(page.getByRole('button', { name: 'Resume analysis' })).toHaveCount(0);
  173 |     expect(await creates()).toHaveLength(1);
  174 |   } finally {
  175 |     await application.close();
  176 |   }
  177 | });
  178 | 
  179 | test('Agent points to the same continuation plan without changing User selection or starting work', async ({}, info) => {
  180 |   test.setTimeout(90000);
  181 |   const fixture = await continuationFixture(info.outputPath('fixture'));
  182 |   const { application, page } = await launch(fixture.profile, fixture.environment);
  183 |   try {
  184 |     await application.evaluate(({ BrowserWindow, shell }) => {
  185 |       BrowserWindow.getAllWindows()[0]!.setSize(1440, 950);
  186 |       shell.openExternal = async () => {};
  187 |     });
  188 |     await chooseProject(application, page, fixture.root);
  189 |     await page.getByRole('button', { name: 'Account', exact: true }).click();
  190 |     await page.getByRole('button', { name: 'Sign in with ChatGPT' }).click();
  191 |     await expect(page.getByText('ChatGPT connected', { exact: true })).toBeVisible();
  192 |     await page.getByRole('button', { name: 'Close Codex account' }).click();
  193 |     await openAgentRoster(page);
  194 |     await page.getByRole('button', { name: 'Add agent', exact: true }).click();
  195 |     const editor = page.getByRole('dialog', { name: 'Add agent', exact: true });
  196 |     await editor.getByLabel('Name', { exact: true }).fill('Researcher');
  197 |     await editor.getByLabel('Project access').selectOption('sharedViews');
  198 |     await editor.getByRole('button', { name: 'Add agent', exact: true }).click();
  199 |     await expect(editor).not.toBeVisible();
  200 |     await page
  201 |       .getByRole('combobox', { name: 'Conversation recipient' })
  202 |       .selectOption({ label: 'To Researcher' });
  203 |     await page.getByRole('button', { name: 'Open Run', exact: true }).click();
  204 |     const flow = page.getByRole('region', { name: 'Run flow', exact: true });
  205 |     await expect(
  206 |       flow.getByRole('button', { name: 'Check continuation', exact: true }),
  207 |     ).toBeEnabled();
  208 |     await flow.getByRole('button', { name: 'Check continuation', exact: true }).click();
  209 |     await expect(flow.locator('[data-continuation="restart"]')).toHaveCount(1, { timeout: 20000 });
  210 |     const selected = flow.getByRole('button', { name: /Inspect Trim reads/ });
  211 |     await selected.click();
  212 |     await page.getByRole('textbox', { name: 'Message draft' }).fill('continuation-point');
  213 |     await page.getByRole('button', { name: 'Send', exact: true }).click();
  214 |     await page.getByRole('textbox', { name: 'Message draft' }).fill('Keep my next question here.');
  215 |     const event = page
  216 |       .locator('.shared-reference-event')
  217 |       .filter({ hasText: 'This checked step restarts from its beginning.' });
> 218 |     await expect(event).toBeVisible();
      |                         ^ Error: expect(locator).toBeVisible() failed
  219 |     await expect(selected).toHaveAttribute('aria-pressed', 'true');
  220 |     await expect(flow.locator('.pipeline-node[data-agent-mark="true"]')).toHaveCount(1);
  221 |     await event.getByRole('button', { name: 'Show in view', exact: true }).click();
  222 |     const reference = page.locator('.observed-reference-panel');
  223 |     await expect(reference).toContainText('Will restart · Attempt 2');
  224 |     await expect(reference.getByRole('button', { name: 'Resume analysis' })).toHaveCount(0);
  225 |     await page.screenshot({ path: info.outputPath('agent-shared-continuation.png') });
  226 |     await reference.getByRole('button', { name: 'Return to my view' }).click();
  227 |     await expect(selected).toHaveAttribute('aria-pressed', 'true');
  228 |     await expect(page.getByRole('textbox', { name: 'Message draft' })).toHaveValue(
  229 |       'Keep my next question here.',
  230 |     );
  231 |     const observed = JSON.parse(
  232 |       await readFile(join(fixture.profile, 'codex/home/continuation-delivery.json'), 'utf8'),
  233 |     );
  234 |     expect(observed.content.continuation.review.steps[1]).toMatchObject({
  235 |       action: 'restart',
  236 |       plannedAttempt: 2,
  237 |     });
  238 |     const commands = (await readFile(join(fixture.base, 'commands.jsonl'), 'utf8'))
  239 |       .trim()
  240 |       .split('\n')
  241 |       .map((s) => JSON.parse(s));
  242 |     expect(commands.filter((a) => ['create', 'start'].includes(a[0]))).toHaveLength(0);
  243 |   } finally {
  244 |     await application.close();
  245 |   }
  246 | });
  247 | 
```