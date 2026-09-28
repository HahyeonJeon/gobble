# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: shared-context.spec.ts >> text pointing overlays a revision-bound range without replacing native local selection
- Location: desktop/tests/electron/shared-context.spec.ts:105:1

# Error details

```
Error: expect(locator).toHaveCount(expected) failed

Locator:  locator('.shared-reference-event')
Expected: 2
Received: 1
Timeout:  5000ms

Call log:
  - Expect "toHaveCount" locator('.shared-reference-event') with timeout 5000ms
  - waiting for locator('.shared-reference-event')
    14 × locator resolved to 1 element
       - unexpected value "1"

```

# Page snapshot

```yaml
- generic [ref=e3]:
  - link "Skip to workspace" [ref=e4] [cursor=pointer]:
    - /url: "#workspace-main"
  - banner [ref=e5]:
    - button "Toggle files" [expanded] [ref=e6] [cursor=pointer]
    - heading "Atlas study" [level=1] [ref=e9]:
      - button "Switch Project" [ref=e10] [cursor=pointer]:
        - generic [ref=e11]: Atlas study
    - generic [ref=e14]:
      - generic "Workspace layout" [ref=e15]:
        - button "Single pane" [pressed] [ref=e16] [cursor=pointer]
        - button "Two panes" [ref=e19] [cursor=pointer]
      - button "Project selections" [ref=e22] [cursor=pointer]:
        - text: Selections
        - generic [ref=e23]: "1"
      - button "Account" [ref=e25] [cursor=pointer]
  - generic [ref=e27]:
    - complementary "Project navigation" [ref=e28]:
      - generic [ref=e29]:
        - group "Pipelines" [ref=e30]:
          - generic "› Pipelines" [ref=e31] [cursor=pointer]
          - button "New pipeline" [ref=e34] [cursor=pointer]: ＋ New pipeline
          - paragraph [ref=e35]: Import an existing analysis to explore its flow.
          - button "Import pipeline" [ref=e36] [cursor=pointer]
        - group [ref=e39]:
          - generic "› Files" [ref=e40] [cursor=pointer]
          - generic [ref=e43]:
            - button "Project files" [disabled] [ref=e44]
            - button "Refresh files" [ref=e45] [cursor=pointer]
          - list [ref=e48]:
            - listitem [ref=e49]:
              - button "notes.txt" [ref=e50] [cursor=pointer]
              - button "Open notes.txt in the other pane" [ref=e54] [cursor=pointer]
            - listitem [ref=e57]:
              - button "quality.png" [ref=e58] [cursor=pointer]
              - button "Open quality.png in the other pane" [ref=e62] [cursor=pointer]
            - listitem [ref=e65]:
              - button "samples.csv" [ref=e66] [cursor=pointer]
              - button "Open samples.csv in the other pane" [ref=e70] [cursor=pointer]
        - group [ref=e73]:
          - generic "› Runs" [ref=e74] [cursor=pointer]
          - generic [ref=e77]:
            - generic [ref=e78]: Existing analyses
            - button "Refresh runs" [ref=e79] [cursor=pointer]
          - list [ref=e82]
          - paragraph [ref=e83]: No runs attached
    - generic [ref=e84]:
      - status [ref=e85]: Workspace ready
      - generic [ref=e86]:
        - main [ref=e87]:
          - region "Primary pane" [ref=e90]:
            - generic [ref=e91]:
              - tablist "Primary views" [ref=e92]:
                - tab "notes.txt" [selected] [ref=e93] [cursor=pointer]
              - generic "View actions" [ref=e95]:
                - button "More actions for notes.txt" [ref=e96] [cursor=pointer]: More
                - button "Close notes.txt" [ref=e99] [cursor=pointer]
            - tabpanel "notes.txt" [ref=e102]:
              - generic "notes.txt preview" [ref=e103]:
                - region "Selection in notes.txt" [ref=e104]:
                  - generic "notes.txt · Text · L1:0–L1:7" [ref=e105]
                  - generic [ref=e106]:
                    - button "Add to message from notes.txt" [ref=e107] [cursor=pointer]: Add to message
                    - button "Share mark from notes.txt" [ref=e108] [cursor=pointer]: Share mark
                    - button "Remove selection from notes.txt" [ref=e109] [cursor=pointer]
                  - status [ref=e112]: Shared in Project chat
                - generic "Shared references in this view" [ref=e113]:
                  - button "You · Text · L1:0–L1:7" [ref=e114] [cursor=pointer]
                - generic [ref=e116]:
                  - generic [ref=e117]:
                    - generic [aria-hidden]:
                      - mark: Inspect
                      - text: sample S03. This draft stays local.
                    - textbox "notes.txt text" [ref=e118]: Inspect sample S03. This draft stays local.
                  - generic [ref=e119]:
                    - generic [ref=e120]: Text preview · Read only
                    - button "Use text selection" [ref=e121] [cursor=pointer]
        - separator "Resize chat" [ref=e122]
        - region "Project chat" [ref=e124]:
          - generic [ref=e125]:
            - heading "Chat" [level=2] [ref=e126]
            - button "Project agents" [ref=e127] [cursor=pointer]:
              - text: Agents
              - generic [ref=e130]: "1"
            - button "Collapse chat" [expanded] [ref=e131] [cursor=pointer]
          - generic "Project messages" [ref=e135]:
            - generic [ref=e136]:
              - generic [ref=e137]: Opened a Project view.
              - article "You shared a reference" [ref=e143]:
                - generic [ref=e144]:
                  - generic [ref=e145]:
                    - strong [ref=e146]:
                      - text: You
                      - generic [ref=e147]: pointed to notes.txt
                    - generic [ref=e148]: Text · L1:0–L1:7
                  - button "Reveal" [ref=e149] [cursor=pointer]
                - group [ref=e150]:
                  - generic "Mark options" [ref=e151] [cursor=pointer]
              - article [ref=e153]:
                - generic [ref=e154]:
                  - generic [ref=e155]:
                    - strong [ref=e156]: You → Researcher
                    - time [ref=e157]: 04:17 PM
                  - paragraph [ref=e158]: shared-point
              - article [ref=e160]:
                - generic [ref=e161]:
                  - generic [ref=e162]:
                    - generic [aria-hidden] [ref=e163]: R
                    - strong [ref=e164]: Researcher
                  - 'button "In reply to: shared-point" [ref=e165] [cursor=pointer]'
                  - paragraph [ref=e166]: "Fixture response from Researcher Fixture shared tool failed: Observation failed: {\"success\":false,\"contentItems\":[{\"type\":\"inputText\",\"text\":\"{\\\"error\\\":{\\\"code\\\":\\\"runtime_unavailable\\\",\\\"message\\\":\\\"The Project window is unavailable. Reveal it and close dialogs before sharing views.\\\"}}\"}]}"
                - generic [ref=e167]: failed
                - paragraph [ref=e169]: Codex could not finish this message. Check your account before trying a new message.
          - generic [ref=e170]:
            - textbox "Message draft" [ref=e171]:
              - /placeholder: Write a message for your agent…
            - generic [ref=e172]:
              - generic [ref=e173]:
                - generic [ref=e174]: Conversation recipient
                - combobox "Conversation recipient" [ref=e177]:
                  - option "Choose recipient"
                  - option "To Researcher" [selected]
              - generic [ref=e178]:
                - generic [ref=e179]: Message model
                - combobox "Message model" [ref=e180]:
                  - option "Fixture model" [selected]
                  - option "Fixture text model"
              - button "Send" [disabled] [ref=e181]
            - generic [ref=e184]:
              - generic [ref=e185]: Draft saved locally
              - generic [ref=e186]: Enter to send · Shift+Enter for a new line
```

# Test source

```ts
  24  |   await copyFile(resolve('desktop/tests/fixtures/codex.cjs'), executable);
  25  |   await chmod(executable, 0o700);
  26  |   ({ application, page } = await launch(profile, { GOBBLE_CODEX_EXECUTABLE: executable }));
  27  |   root = await projectFixture(base);
  28  |   await chooseProject(application, page, root);
  29  |   await application.evaluate(({ shell }) => {
  30  |     shell.openExternal = async () => {};
  31  |   });
  32  |   await page.getByRole('button', { name: 'Account', exact: true }).click();
  33  |   await page.getByRole('button', { name: 'Sign in with ChatGPT' }).click();
  34  |   await expect(page.getByText('ChatGPT connected', { exact: true })).toBeVisible();
  35  |   await page.getByRole('button', { name: 'Close Codex account' }).click();
  36  | });
  37  | test.afterEach(async () => {
  38  |   await application?.close();
  39  |   if (base) await rm(base, { recursive: true, force: true });
  40  | });
  41  | async function agent(shared = true) {
  42  |   await openAgentRoster(page);
  43  |   await page.getByRole('button', { name: 'Add agent', exact: true }).click();
  44  |   const dialog = page.getByRole('dialog', { name: 'Add agent', exact: true });
  45  |   await dialog.getByLabel('Name', { exact: true }).fill('Researcher');
  46  |   if (shared) await dialog.getByLabel('Project access').selectOption('sharedViews');
  47  |   await dialog.getByRole('button', { name: 'Add agent', exact: true }).click();
  48  |   await expect(dialog).not.toBeVisible();
  49  |   await page
  50  |     .getByRole('combobox', { name: 'Conversation recipient' })
  51  |     .selectOption({ label: 'To Researcher' });
  52  | }
  53  | async function send(text: string) {
  54  |   await page.getByRole('textbox', { name: 'Message draft' }).fill(text);
  55  |   await page.getByRole('button', { name: 'Send', exact: true }).click();
  56  |   await expect(page.getByRole('textbox', { name: 'Message draft' })).toHaveValue('');
  57  | }
  58  | test('User and Agent share table/image pointers while local selection and composer focus stay independent', async ({}, info) => {
  59  |   await page.getByRole('button', { name: 'samples.csv', exact: true }).click();
  60  |   await readyView(page);
  61  |   await page.getByRole('checkbox', { name: /Select row 3:/ }).check();
  62  |   await page.getByRole('button', { name: 'Share mark from samples.csv' }).click();
  63  |   await expect(page.locator('.shared-reference-event')).toHaveCount(1);
  64  |   await expect(page.locator('.shared-cell')).toHaveCount(4);
  65  |   await page.getByRole('button', { name: 'quality.png', exact: true }).click();
  66  |   await readyView(page);
  67  |   await openPaneActions(page, 'quality.png');
  68  |   await page.getByRole('button', { name: 'Move quality.png to other pane' }).click();
  69  |   await readyView(page);
  70  |   await readyView(page, 'Secondary pane');
  71  |   await agent();
  72  |   await send('shared-point');
  73  |   await page.getByRole('textbox', { name: 'Message draft' }).fill('Keep this draft local.');
  74  |   await expect(page.locator('.shared-reference-event')).toHaveCount(3);
  75  |   await expect(page.locator('.message-state').getByText('completed', { exact: true })).toHaveCount(
  76  |     1,
  77  |   );
  78  |   await expect(page.locator('.shared-image-mark')).toHaveCount(1);
  79  |   await expect(page.getByRole('checkbox', { name: /Select row 3:/ })).toBeChecked();
  80  |   await expect(page.getByRole('textbox', { name: 'Message draft' })).toHaveValue(
  81  |     'Keep this draft local.',
  82  |   );
  83  |   await expect(page.getByRole('textbox', { name: 'Message draft' })).toBeFocused();
  84  |   const imageEvent = page
  85  |     .getByRole('article', { name: 'Researcher shared a reference' })
  86  |     .filter({ hasText: 'quality.png' });
  87  |   await page.getByRole('button', { name: 'Close quality.png', exact: true }).click();
  88  |   await expect(imageEvent).toContainText('quality.png');
  89  |   await imageEvent.getByRole('button', { name: 'Reveal', exact: true }).click();
  90  |   await readyView(page, 'Secondary pane');
  91  |   await expect(page.locator('.shared-image-mark')).toHaveCount(1);
  92  |   await page.screenshot({ path: info.outputPath('shared-pointing.png') });
  93  |   await application.close();
  94  |   ({ application, page } = await launch(profile, { GOBBLE_CODEX_EXECUTABLE: executable }));
  95  |   await expect(page.locator('.shared-reference-event')).toHaveCount(3);
  96  |   await expect(page.getByRole('textbox', { name: 'Message draft' })).toHaveValue(
  97  |     'Keep this draft local.',
  98  |   );
  99  |   const ownMark = page.getByRole('article', { name: 'You shared a reference', exact: true });
  100 |   await ownMark.getByText('Mark options', { exact: true }).click();
  101 |   await ownMark.getByRole('button', { name: 'Retract mark', exact: true }).click();
  102 |   await expect(ownMark).toContainText('Reference retracted · history preserved');
  103 |   await expect(page.locator('.shared-reference-event')).toHaveCount(3);
  104 | });
  105 | test('text pointing overlays a revision-bound range without replacing native local selection', async () => {
  106 |   await page.getByRole('button', { name: 'notes.txt', exact: true }).click();
  107 |   await readyView(page);
  108 |   const text = page.getByRole('textbox', { name: 'notes.txt text' });
  109 |   await text.focus();
  110 |   await text.press('ControlOrMeta+a');
  111 |   await text.press('ArrowLeft');
  112 |   for (let index = 0; index < 7; index++) await text.press('Shift+ArrowRight');
  113 |   expect(
  114 |     await text.evaluate((element: HTMLTextAreaElement) => [
  115 |       element.selectionStart,
  116 |       element.selectionEnd,
  117 |     ]),
  118 |   ).toEqual([0, 7]);
  119 |   await page.getByRole('button', { name: 'Use text selection' }).click();
  120 |   await page.getByRole('button', { name: 'Share mark from notes.txt' }).click();
  121 |   await expect(page.locator('.text-highlights mark')).toHaveText('Inspect');
  122 |   await agent();
  123 |   await send('shared-point');
> 124 |   await expect(page.locator('.shared-reference-event')).toHaveCount(2);
      |                                                         ^ Error: expect(locator).toHaveCount(expected) failed
  125 |   expect(
  126 |     await text.evaluate((element: HTMLTextAreaElement) => [
  127 |       element.selectionStart,
  128 |       element.selectionEnd,
  129 |     ]),
  130 |   ).toEqual([0, 7]);
  131 |   await page
  132 |     .locator('.shared-reference-event')
  133 |     .first()
  134 |     .getByRole('button', { name: 'Reveal', exact: true })
  135 |     .click();
  136 |   await writeFile(join(root, 'notes.txt'), 'A changed file.\nOriginal selection is stale.\n');
  137 |   await openPaneActions(page, 'notes.txt');
  138 |   await page.getByRole('button', { name: 'Refresh notes.txt' }).click();
  139 |   await readyView(page);
  140 |   await expect(page.locator('.text-highlights mark')).toHaveCount(0);
  141 |   await expect(
  142 |     page.getByText('This shared reference targets an older version.', { exact: false }),
  143 |   ).toBeVisible();
  144 | });
  145 | test('explicit shared-view opt-in starts a new conversation and retains earlier messages', async () => {
  146 |   await agent(false);
  147 |   await send('Earlier message');
  148 |   await expect(page.locator('.message-state').getByText('completed', { exact: true })).toHaveCount(
  149 |     1,
  150 |   );
  151 |   await openAgentRoster(page);
  152 |   await page.getByRole('button', { name: 'Settings for Researcher' }).click();
  153 |   await expect(page.getByText('Messages only', { exact: true })).toBeVisible();
  154 |   await page
  155 |     .getByRole('button', { name: 'Enable shared views and start new conversation', exact: true })
  156 |     .click();
  157 |   await expect(page.getByRole('dialog')).not.toBeVisible();
  158 |   await expect(page.getByText('Earlier message', { exact: true })).toBeVisible();
  159 |   await openAgentRoster(page);
  160 |   await page.getByRole('button', { name: 'Settings for Researcher' }).click();
  161 |   await expect(page.getByText('Shared views enabled', { exact: true })).toBeVisible();
  162 |   await page.getByRole('button', { name: 'Disable shared views and stop' }).click();
  163 |   await expect(page.getByRole('dialog')).not.toBeVisible();
  164 | });
  165 | 
```