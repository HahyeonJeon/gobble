# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: workspace.spec.ts >> selection overview finds hidden sources, More returns keyboard focus, and empty panes target navigation
- Location: desktop/tests/electron/workspace.spec.ts:394:1

# Error details

```
Error: expect(locator).toHaveAttribute(expected) failed

Locator: getByRole('region', { name: 'Secondary pane', exact: true }).locator('.surface-view')
Expected: "true"
Timeout: 5000ms
Error: element(s) not found

Call log:
  - Expect "toHaveAttribute" getByRole('region', { name: 'Secondary pane', exact: true }).locator('.surface-view') with timeout 5000ms
  - waiting for getByRole('region', { name: 'Secondary pane', exact: true }).locator('.surface-view')

```

```yaml
- link "Skip to workspace":
  - /url: "#workspace-main"
- banner:
  - button "Toggle files" [expanded]
  - heading "Atlas study" [level=1]:
    - button "Switch Project": Atlas study
  - button "Single pane"
  - button "Two panes" [pressed]
  - button "Project selections": Selections 1
  - button "Account"
- complementary "Project navigation":
  - group:
    - text: › Files
    - button "Project files" [disabled]
    - button "Refresh files"
    - list:
      - listitem:
        - button "notes.txt"
        - button "Open notes.txt in the other pane"
      - listitem:
        - button "quality.png"
        - button "Open quality.png in the other pane"
      - listitem:
        - button "samples.csv"
        - button "Open samples.csv in the other pane"
  - group:
    - text: › Runs Existing analyses
    - button "Refresh runs"
    - list
    - paragraph: No runs attached
- status: Workspace ready
- main:
  - region "Primary pane":
    - tablist "Primary views":
      - tab "samples.csv"
      - tab "notes.txt"
      - tab "quality.png" [selected]
    - button "More actions for quality.png": More
    - button "Maximize primary pane"
    - button "Close quality.png"
    - tabpanel "quality.png":
      - button "Add view preview of quality.png to message": Add view preview
      - img "quality.png"
      - text: Image preview · 1260 × 840
      - button "Select region"
  - separator "Resize panes"
  - region "Secondary pane":
    - tablist "Secondary views": Second pane
    - button "Maximize secondary pane"
    - heading "What would you like to open?" [level=2]
    - paragraph: Choose a file or Run from this Project.
    - button "Browse files"
    - group: Move a view here
- separator "Resize chat"
- region "Project chat":
  - heading "Chat" [level=2]
  - button "Project agents": Agents 0
  - button "Collapse chat" [expanded]
  - strong: Work together in this Project
  - paragraph: Choose an agent and start a conversation about your work.
  - group: 4 view updates
  - textbox "Message draft":
    - /placeholder: Write a message for your agent…
    - text: Keep this draft while finding my selection.
  - text: Conversation recipient
  - combobox "Conversation recipient" [disabled]:
    - option "No agent connected" [selected]
  - button "Send" [disabled]
  - text: Draft saved locally Enter to send · Shift+Enter for a new line
```

# Test source

```ts
  1  | import {
  2  |   _electron as electron,
  3  |   expect,
  4  |   type ElectronApplication,
  5  |   type Page,
  6  |   type Locator,
  7  | } from '@playwright/test';
  8  | import { copyFile, mkdir, writeFile } from 'node:fs/promises';
  9  | import { join, resolve } from 'node:path';
  10 | 
  11 | export async function launch(profile: string, extraEnv: Record<string, string> = {}) {
  12 |   const env = Object.fromEntries(
  13 |     Object.entries(process.env).filter(
  14 |       (entry): entry is [string, string] =>
  15 |         entry[1] !== undefined &&
  16 |         !['ELECTRON_RUN_AS_NODE', 'ELECTRON_RENDERER_URL'].includes(entry[0]),
  17 |     ),
  18 |   );
  19 |   const application = await electron.launch({
  20 |     args: [resolve('desktop'), '--gobble-profile=' + profile],
  21 |     env: { ...env, ...extraEnv },
  22 |   });
  23 |   const page = await application.firstWindow();
  24 |   await expect(page.locator('.app-status')).toHaveText('Workspace ready');
  25 |   // Shared observation requires an actual foreground window, including on macOS relaunch.
  26 |   await application.evaluate(({ app, BrowserWindow }) => {
  27 |     app.focus({ steal: true });
  28 |     const window = BrowserWindow.getAllWindows()[0]!;
  29 |     window.show();
  30 |     window.focus();
  31 |   });
  32 |   await expect
  33 |     .poll(() =>
  34 |       application.evaluate(({ BrowserWindow }) => BrowserWindow.getAllWindows()[0]!.isFocused()),
  35 |     )
  36 |     .toBe(true);
  37 |   return { application, page };
  38 | }
  39 | export async function projectFixture(base: string, name = 'Atlas study') {
  40 |   const root = join(base, name);
  41 |   await mkdir(root, { recursive: true });
  42 |   await writeFile(
  43 |     join(root, 'samples.csv'),
  44 |     'Sample,Group,Read 1,Read 2\nS01,Control,S01_R1.fastq.gz,S01_R2.fastq.gz\nS02,Control,S02_R1.fastq.gz,S02_R2.fastq.gz\nS03,Treatment,S03_R1.fastq.gz,S03_R2.fastq.gz\nS04,Treatment,S04_R1.fastq.gz,S04_R2.fastq.gz\nS05,Treatment,S05_R1.fastq.gz,S05_R2.fastq.gz\nS06,Control,S06_R1.fastq.gz,S06_R2.fastq.gz\n',
  45 |   );
  46 |   await writeFile(join(root, 'notes.txt'), 'Inspect sample S03.\nThis draft stays local.\n');
  47 |   await copyFile(resolve('desktop/tests/electron/fixtures/quality.png'), join(root, 'quality.png'));
  48 |   return root;
  49 | }
  50 | export async function chooseProject(application: ElectronApplication, page: Page, root: string) {
  51 |   // Only the native chooser response is simulated. The button, host and Go service are real.
  52 |   await application.evaluate(({ dialog }, root) => {
  53 |     dialog.showOpenDialog = async () => ({ canceled: false, filePaths: [root] });
  54 |   }, root);
  55 |   if (!(await page.getByRole('button', { name: 'Open folder', exact: true }).isVisible()))
  56 |     await page.getByRole('button', { name: 'Switch Project', exact: true }).click();
  57 |   await page.getByRole('button', { name: 'Open folder', exact: true }).click();
  58 |   await expect(
  59 |     page.getByRole('heading', { name: root.split('/').at(-1) ?? '', exact: true }),
  60 |   ).toBeVisible();
  61 | }
  62 | export async function readyView(page: Page, pane = 'Primary pane') {
  63 |   const scope = page.getByRole('region', { name: pane, exact: true });
> 64 |   await expect(scope.locator('.surface-view')).toHaveAttribute('data-ready', 'true');
     |                                                ^ Error: expect(locator).toHaveAttribute(expected) failed
  65 |   return scope;
  66 | }
  67 | 
  68 | export async function openAgentRoster(page: Page) {
  69 |   await page.getByRole('button', { name: 'Project agents', exact: true }).click();
  70 |   await expect(page.getByRole('dialog', { name: 'Project agents', exact: true })).toBeVisible();
  71 | }
  72 | 
  73 | export async function openPaneActions(scope: Page | Locator, title: string) {
  74 |   await scope.getByRole('button', { name: 'More actions for ' + title, exact: true }).click();
  75 |   await expect(
  76 |     scope.getByRole('group', { name: 'Actions for ' + title, exact: true }),
  77 |   ).toBeVisible();
  78 | }
  79 | export function attachmentPreview(page: Page) {
  80 |   return page.getByRole('dialog', { name: /^Attachment: / });
  81 | }
  82 | export async function closeAttachmentPreview(page: Page) {
  83 |   await page.keyboard.press('Escape');
  84 |   await expect(attachmentPreview(page)).not.toBeVisible();
  85 | }
  86 | 
  87 | /** Native capture includes the full physical window at non-default page zoom. */
  88 | export async function captureWindow(application: ElectronApplication, path: string) {
  89 |   const png = await application.evaluate(async ({ BrowserWindow }) =>
  90 |     (await BrowserWindow.getAllWindows()[0]!.capturePage()).toPNG().toString('base64'),
  91 |   );
  92 |   await writeFile(path, Buffer.from(png, 'base64'));
  93 | }
  94 | 
```