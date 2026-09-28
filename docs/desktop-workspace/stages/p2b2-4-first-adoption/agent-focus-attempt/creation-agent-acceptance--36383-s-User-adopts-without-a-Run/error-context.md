# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: creation-agent-acceptance.spec.ts >> Creation: signed-in Agent proposes, reads and points; User adopts without a Run
- Location: desktop/tests/electron/creation-agent-acceptance.spec.ts:10:1

# Error details

```
Error: expect(locator).toBeVisible() failed

Locator: locator('.creation-view').getByRole('button', { name: /Researcher: Quality threshold reviewed:/ })
Expected: visible
Timeout: 180000ms
Error: element(s) not found

Call log:
  - Expect "toBeVisible" locator('.creation-view').getByRole('button', { name: /Researcher: Quality threshold reviewed:/ }) with timeout 180000ms
  - waiting for locator('.creation-view').getByRole('button', { name: /Researcher: Quality threshold reviewed:/ })

```

```yaml
- link "Skip to workspace":
  - /url: "#workspace-main"
- banner:
  - button "Toggle files" [expanded]
  - heading "Read quality acceptance" [level=1]:
    - button "Switch Project": Read quality acceptance
  - button "Single pane" [pressed]
  - button "Two panes"
  - button "Project selections": Selections 0
  - button "Account"
- complementary "Project navigation":
  - group "Pipelines":
    - text: › Pipelines
    - button "New pipeline": ＋ New pipeline
    - text: Drafts
    - list:
      - listitem:
        - button "◇ sample.fastq.gz"
    - paragraph: Import an existing analysis to explore its flow.
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
      - tab "New pipeline" [selected]
    - button "More actions for New pipeline": More
    - button "Close New pipeline"
    - tabpanel "New pipeline":
      - strong: New pipeline
      - text: Draft · Not yet a Pipeline
      - button "Refresh"
      - button "Discard draft"
      - text: sample.fastq.gz · Single-end reads
      - button "Change data"
      - button "Discuss this draft"
      - text: ● Analysis engine connected
      - region "Creation adoption":
        - button "Adopt as new pipeline"
      - text: Proposal
      - combobox "Creation proposal":
        - option "1 · Trim adapters and low-quality bases from the selected single-end reads with Trim Galore at Phred 25 and minimum length 40 bp, then inspect the trimmed reads with FastQC. · ready" [selected]
      - button "Show list"
      - strong: Proposed
      - text: + 2 steps · Checked design
      - group "Flow zoom":
        - button "Fit pipeline flow": Fit
        - button "Zoom out pipeline flow": −
        - status "Flow zoom level": 74%
        - button "Zoom in pipeline flow": +
      - img "Pipeline connections":
        - button "Pipeline input · reads → Trim adapters and low-quality bases · read1"
        - button "Trim adapters and low-quality bases · trimmed_read1 → Inspect trimmed read quality · reads"
      - button "Inspect reads · file":
        - strong: reads
        - text: file
      - button "Inspect Trim adapters and low-quality bases · 1 input · 2 outputs" [pressed]:
        - strong: Trim adapters and low-quality bases
        - text: 1 + 1 input · 2 outputs
      - button "Inspect Inspect trimmed read quality · 1 input · 2 outputs":
        - strong: Inspect trimmed read quality
        - text: 2 + 1 input · 2 outputs
      - text: Current
      - heading "No current version" [level=3]
      - paragraph: This is a new analysis. Every step and connection is a proposed addition.
      - text: Checking a design does not create a Pipeline or start a Run. Proposed · Added
      - region "Details for Quality threshold":
        - text: Setting
        - heading "Quality threshold" [level=3]
        - button "Close step details": Close
        - text: 25 Phred
        - button "Add to message"
        - status: Added to your next message.
        - heading "Inputs" [level=4]
        - list:
          - listitem:
            - button "Select inputs port read1": read1
            - text: file inputs/reads.fastq.gz
        - heading "Declared outputs" [level=4]
        - list:
          - listitem:
            - button "Select declared outputs port trimmed_read1": trimmed_read1
            - text: file work/trim-galore/sample_trimmed.fq.gz
          - listitem:
            - button "Select declared outputs port report1": report1
            - text: file work/trim-galore/reads.fastq.gz_trimming_report.txt
        - heading "Analysis settings" [level=4]
        - button "Quality threshold 25 Phred" [pressed]:
          - text: Quality threshold
          - strong: 25 Phred
        - button "Minimum length 40 bp":
          - text: Minimum length
          - strong: 40 bp
        - term: Tool
        - definition: community.wave.seqera.io/library/trim-galore:2.1.0--27e6376b8f6c1872@sha256:9d747504e44dbf5dfa8a2d66cbbd3bd80f897cc2e17ebe406821b4809b34a3a4
        - term: Requested compute
        - definition: 4 CPU · 2g memory
        - paragraph: This view shows the step’s inputs, expected outputs and requested computing resources.
        - heading "Connections" [level=4]
        - list:
          - listitem:
            - button "Pipeline input · reads → Trim adapters and low-quality bases · read1"
          - listitem:
            - button "Trim adapters and low-quality bases · trimmed_read1 → Inspect trimmed read quality · reads"
- separator "Resize chat"
- region "Project chat":
  - heading "Chat" [level=2]
  - button "Project agents": Agents 1
  - button "Collapse chat" [expanded]
  - text: Opened a Project view.
  - article:
    - strong: You → Researcher
    - time: 08:35 PM
    - text: New pipeline · Selected data · Proposal allowed
    - paragraph: "Please design a new analysis for the selected single-end reads: trim adapters and low-quality bases with a quality threshold of 25 Phred and minimum length of 40 bp, then inspect the trimmed reads with FastQC. Prepare the proposal through the provided creation tools so I can review the flow. Do not start an analysis."
  - article:
    - strong: Researcher
    - 'button "In reply to: Please design a new analysis for the selected single-end reads: trim adapters an…"'
    - paragraph: "I’ll prepare and check a flow that trims the selected single-end reads at Phred 25, keeps reads at least 40 bp long, and sends the trimmed reads to FastQC for review. The proposal has been submitted with Phred 25 and a 40 bp minimum length. Gobble is still checking it. The Project window is unavailable, so I couldn’t reveal the flow there. Proposal submitted: selected single-end reads → Trim Galore (adapter trimming, Phred 25, minimum length 40 bp) → FastQC on trimmed reads. Gobble’s check is still pending. I couldn’t reveal the flow because the Project window is unavailable. No analysis was started."
    - text: completed
  - article:
    - strong: You → Researcher
    - time: 08:37 PM
    - text: Proposed addition · Quality threshold · 25 Phred · Discussion only
    - paragraph: Read the exact attached proposal setting, briefly explain what this quality threshold means, and point to this exact setting using the creation review and point tools. Begin your pointer note with "Quality threshold reviewed:". This is discussion only; keep the proposal unchanged.
  - article:
    - strong: Researcher
    - 'button "In reply to: Read the exact attached proposal setting, briefly explain what this quality thre…"'
    - paragraph: "I’ll review the attached quality setting and point to it with a brief explanation. Reviewed: Trim Galore’s quality threshold is **25 Phred**, used to trim low-quality read ends. Q25 corresponds to an estimated base-call error probability of **0.32%**; it doesn’t guarantee every retained base meets Q25. The exact-setting pointer could not be published because the Project window is unavailable. Reveal it and close dialogs to enable pointing. The proposal remains unchanged."
    - text: completed
  - textbox "Message draft":
    - /placeholder: Write a message for your agent…
  - text: Conversation recipient
  - combobox "Conversation recipient":
    - option "Choose recipient"
    - option "To Researcher" [selected]
  - text: Message model
  - combobox "Message model":
    - option "GPT-6-Astra" [selected]
    - option "GPT-5.6-Sol"
    - option "GPT-5.6-Terra"
    - option "GPT-5.6-Luna"
    - option "GPT-5.5"
    - option "GPT-5.3-Codex-Spark"
  - button "Send" [disabled]
  - text: Draft saved locally Enter to send · Shift+Enter for a new line
```

# Test source

```ts
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
  31  |     const focus = async () => {
  32  |       await application.evaluate(({ app, BrowserWindow }) => {
  33  |         app.focus({ steal: true });
  34  |         const window = BrowserWindow.getAllWindows().find((w) =>
  35  |           w.webContents.getURL().startsWith('app://gobble/'),
  36  |         )!;
  37  |         window.show();
  38  |         window.focus();
  39  |       });
  40  |       await expect.poll(() => page.evaluate(() => document.hasFocus())).toBe(true);
  41  |     };
  42  |     await chooseProject(application, page, project);
  43  |     await page.getByRole('button', { name: 'New pipeline', exact: true }).click();
  44  |     const view = page.locator('.creation-view');
  45  |     await view.getByRole('button', { name: '◇ sample.fastq.gz', exact: true }).click();
  46  |     await view.getByRole('button', { name: 'Use selected file' }).click();
  47  |     await view.getByRole('button', { name: 'Connect engine', exact: true }).click();
  48  |     await expect(view.getByText('● Analysis engine connected', { exact: true })).toBeVisible({
  49  |       timeout: 45000,
  50  |     });
  51  |     await page.getByRole('button', { name: 'Account', exact: true }).click();
  52  |     await page.getByRole('button', { name: 'Refresh connection' }).click();
  53  |     await expect(page.getByText('ChatGPT connected', { exact: true })).toBeVisible({
  54  |       timeout: 45000,
  55  |     });
  56  |     await page.getByRole('button', { name: 'Close Codex account' }).click();
  57  |     await openAgentRoster(page);
  58  |     await page.getByRole('button', { name: 'Add agent', exact: true }).click();
  59  |     const editor = page.getByRole('dialog', { name: 'Add agent', exact: true });
  60  |     await editor.getByLabel('Name', { exact: true }).fill('Researcher');
  61  |     await editor.getByLabel('Project access').selectOption('sharedViews');
  62  |     const model = await editor.getByLabel('Model').inputValue();
  63  |     const effort = await editor.getByLabel('Reasoning effort').inputValue();
  64  |     await editor
  65  |       .getByLabel('Role instructions')
  66  |       .fill(
  67  |         'Discuss this analysis in concise English using the shared flow and its exact facts. Keep source code behind the UI.',
  68  |       );
  69  |     await editor.getByRole('button', { name: 'Add agent', exact: true }).click();
  70  |     await page
  71  |       .getByRole('combobox', { name: 'Conversation recipient' })
  72  |       .selectOption({ label: 'To Researcher' });
  73  |     await page
  74  |       .getByRole('checkbox', { name: 'Allow a new Pipeline proposal for this message' })
  75  |       .check();
  76  |     await page
  77  |       .getByRole('textbox', { name: 'Message draft' })
  78  |       .fill(
  79  |         'Please design a new analysis for the selected single-end reads: trim adapters and low-quality bases with a quality threshold of 25 Phred and minimum length of 40 bp, then inspect the trimmed reads with FastQC. Prepare the proposal through the provided creation tools so I can review the flow. Do not start an analysis.',
  80  |       );
  81  |     await focus();
  82  |     await page.getByRole('button', { name: 'Send', exact: true }).click();
  83  |     await expect(view.getByRole('heading', { name: 'No current version' })).toBeVisible({
  84  |       timeout: 240000,
  85  |     });
  86  |     await expect(view.locator('.pipeline-node[data-change="added-step"]')).toHaveCount(2);
  87  |     await view.getByRole('button', { name: /Inspect Trim adapters/ }).click();
  88  |     await view.getByRole('button', { name: /Quality threshold.*25/ }).click();
  89  |     await view.getByRole('button', { name: 'Add to message', exact: true }).click();
  90  |     await expect(page.locator('.composer-context')).toContainText('Quality threshold · 25 Phred');
  91  |     await page
  92  |       .getByRole('textbox', { name: 'Message draft' })
  93  |       .fill(
  94  |         'Read the exact attached proposal setting, briefly explain what this quality threshold means, and point to this exact setting using the creation review and point tools. Begin your pointer note with "Quality threshold reviewed:". This is discussion only; keep the proposal unchanged.',
  95  |       );
  96  |     await expect(page.getByRole('button', { name: 'Send', exact: true })).toBeEnabled({
  97  |       timeout: 90000,
  98  |     });
  99  |     await focus();
  100 |     await page.getByRole('button', { name: 'Send', exact: true }).click();
  101 |     await expect(
  102 |       view.getByRole('button', { name: /Researcher: Quality threshold reviewed:/ }),
> 103 |     ).toBeVisible({
      |       ^ Error: expect(locator).toBeVisible() failed
  104 |       timeout: 180000,
  105 |     });
  106 |     await expect(
  107 |       page.locator('.message-state small').filter({ hasText: /^completed$/ }),
  108 |     ).toHaveCount(2, { timeout: 90000 });
  109 |     const draftCatalog = JSON.parse(await readFile(join(profile, 'service/catalog.json'), 'utf8'));
  110 |     const selectedDraft = Object.values(draftCatalog.drafts)[0] as {
  111 |       projectId: string;
  112 |       draftId: string;
  113 |     };
  114 |     const shared = await page.evaluate((scope) => window.gobble.creation.state(scope), {
  115 |       projectId: selectedDraft.projectId,
  116 |       draftId: selectedDraft.draftId,
  117 |     });
  118 |     if (!shared.ok) throw new Error(shared.error.message);
  119 |     const exactMark = shared.value.marks.find((mark) =>
  120 |       mark.note.startsWith('Quality threshold reviewed:'),
  121 |     );
  122 |     expect(exactMark?.context.kind).toBe('candidate');
  123 |     if (exactMark?.context.kind !== 'candidate')
  124 |       throw new Error('Missing retained candidate reference');
  125 |     expect(exactMark.context.target?.kind).toBe('setting');
  126 |     if (exactMark.context.target?.kind !== 'setting')
  127 |       throw new Error('Agent did not point to the setting');
  128 |     const reference = exactMark.context;
  129 |     const target = reference.target;
  130 |     if (target?.kind !== 'setting') throw new Error('Missing setting target');
  131 |     const checked = shared.value.candidates.find(
  132 |       (candidate) => candidate.candidateId === reference.candidateId,
  133 |     )!;
  134 |     const selectedStep = checked.artifact!.check.review.flow.steps.find(
  135 |       (step) => step.id === target.id,
  136 |     )!;
  137 |     expect(selectedStep.settings.find((setting) => setting.key === target.name)).toMatchObject({
  138 |       label: 'Quality threshold',
  139 |       value: 25,
  140 |       unit: 'Phred',
  141 |     });
  142 |     await writeFile(
  143 |       info.outputPath('agent-exact-reference.json'),
  144 |       JSON.stringify(exactMark, null, 2),
  145 |     );
  146 |     await page.screenshot({ path: info.outputPath('agent-shared-review.png') });
  147 |     await view.getByRole('button', { name: 'Adopt as new pipeline', exact: true }).click();
  148 |     await view.getByRole('textbox', { name: 'Pipeline name' }).fill('Read quality acceptance');
  149 |     await view.getByRole('button', { name: 'Confirm creation' }).click();
  150 |     await expect(view.getByText('Created · No Run started.', { exact: true })).toBeVisible({
  151 |       timeout: 45000,
  152 |     });
  153 |     await page
  154 |       .getByRole('textbox', { name: 'Message draft' })
  155 |       .fill('Keep this question for execution preparation.');
  156 |     const catalog = JSON.parse(await readFile(join(profile, 'service/catalog.json'), 'utf8'));
  157 |     expect(catalog.pipelines).toHaveLength(1);
  158 |     expect(catalog.runs).toHaveLength(0);
  159 |     await writeFile(
  160 |       info.outputPath('acceptance.json'),
  161 |       JSON.stringify(
  162 |         {
  163 |           kind: 'real-signed-in-agent',
  164 |           model,
  165 |           effort,
  166 |           profile,
  167 |           project,
  168 |           pipelines: catalog.pipelines,
  169 |           birth: Object.values(catalog.drafts),
  170 |           runs: catalog.runs,
  171 |         },
  172 |         null,
  173 |         2,
  174 |       ),
  175 |     );
  176 |     await application.close();
  177 |     app = undefined;
  178 |     app = await launch(profile, { GOBBLE_CODEX_EXECUTABLE: resolve('node_modules/.bin/codex') });
  179 |     await expect(app.page.getByRole('textbox', { name: 'Message draft' })).toHaveValue(
  180 |       'Keep this question for execution preparation.',
  181 |     );
  182 |     await app.page.getByRole('button', { name: 'Open pipeline', exact: true }).click();
  183 |     await expect(app.page.locator('.creation-view')).toHaveCount(0);
  184 |     await expect(app.page.locator('.pipeline-node')).toHaveCount(3);
  185 |     await app.page.screenshot({ path: info.outputPath('agent-created-current.png') });
  186 |     expect(await readFile(join(project, 'sample.fastq.gz'), 'utf8')).toBe(
  187 |       'Metadata-only isolated acceptance data.',
  188 |     );
  189 |   } catch (error) {
  190 |     if (app) {
  191 |       await app.page
  192 |         .getByRole('button', { name: 'Close Codex account' })
  193 |         .click({ timeout: 1000 })
  194 |         .catch(() => {});
  195 |       await app.page.screenshot({ path: info.outputPath('failure.png') }).catch(() => {});
  196 |       await writeFile(
  197 |         info.outputPath('visible.txt'),
  198 |         await app.page.locator('body').innerText(),
  199 |       ).catch(() => {});
  200 |     }
  201 |     throw error;
  202 |   } finally {
  203 |     await app?.application.close().catch(() => {});
```