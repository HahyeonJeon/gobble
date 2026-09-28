import { test, expect, type ElectronApplication, type Page, type Locator } from '@playwright/test';
import { chmod, copyFile, mkdtemp, rm, writeFile } from 'node:fs/promises';
import { join, resolve } from 'node:path';
import { tmpdir } from 'node:os';
import {
  captureWindow,
  launch,
  chooseProject,
  projectFixture,
  readyView,
  openAgentRoster,
  attachmentPreview,
  closeAttachmentPreview,
  openPaneActions,
} from './support';

let application: ElectronApplication;
let page: Page;
let base: string;
let profile: string;
let root: string;
let executable: string;
const env = () => ({ GOBBLE_CODEX_EXECUTABLE: executable });

test.beforeEach(async () => {
  base = await mkdtemp(join(tmpdir(), 'gobble-ux-review-'));
  profile = join(base, 'profile');
  executable = join(base, 'codex');
  await copyFile(resolve('desktop/tests/fixtures/codex.cjs'), executable);
  await chmod(executable, 0o700);
  ({ application, page } = await launch(profile, env()));
  root = await projectFixture(
    base,
    'A Project for reviewing results and discussing the next experiment',
  );
  await chooseProject(application, page, root);
  await application.evaluate(({ shell }) => {
    shell.openExternal = async () => {};
  });
  await page.getByRole('button', { name: 'Account', exact: true }).click();
  await page.getByRole('button', { name: 'Sign in with ChatGPT' }).click();
  await expect(page.getByText('ChatGPT connected', { exact: true })).toBeVisible();
  await page.getByRole('button', { name: 'Close Codex account' }).click();
});
test.afterEach(async () => {
  await application?.close();
  if (base) await rm(base, { recursive: true, force: true });
});
async function addAgent(name: string, shared = false) {
  await openAgentRoster(page);
  await page.getByRole('button', { name: 'Add agent', exact: true }).click();
  const dialog = page.getByRole('dialog', { name: 'Add agent', exact: true });
  await dialog.getByLabel('Name', { exact: true }).fill(name);
  if (shared) await dialog.getByLabel('Project access').selectOption('sharedViews');
  await dialog.getByRole('button', { name: 'Add agent', exact: true }).click();
  await expect(dialog).not.toBeVisible();
}
async function viewport(width: number, height: number, zoom = 1) {
  await application.evaluate(
    ({ BrowserWindow }, size) => {
      const window = BrowserWindow.getAllWindows()[0]!;
      window.setContentSize(size.width, size.height);
      window.webContents.setZoomFactor(size.zoom);
    },
    { width, height, zoom },
  );
  await expect.poll(() => page.evaluate(() => innerWidth)).toBe(Math.round(width / zoom));
}
async function withinViewport(locator: Locator) {
  await expect(locator).toBeVisible();
  const bounds = (await locator.boundingBox())!;
  const view = await page.evaluate(() => ({ width: innerWidth, height: innerHeight }));
  expect(bounds.x).toBeGreaterThanOrEqual(0);
  expect(bounds.y).toBeGreaterThanOrEqual(0);
  expect(bounds.x + bounds.width).toBeLessThanOrEqual(view.width + 1);
  expect(bounds.y + bounds.height).toBeLessThanOrEqual(view.height + 1);
}
async function projectBarDoesNotOverlap() {
  const title = (await page
    .getByRole('button', { name: 'Switch Project', exact: true })
    .boundingBox())!;
  const actions = (await page.locator('.project-bar-actions').boundingBox())!;
  expect(title.x + title.width).toBeLessThanOrEqual(actions.x);
}
async function documentSnapshot() {
  return page.evaluate(async () => {
    const projects = await window.gobble.projects.list();
    if (!projects.ok) throw new Error(projects.error.message);
    const result = await window.gobble.workspace.read({ projectId: projects.value[0]!.projectId });
    if (!result.ok) throw new Error(result.error.message);
    return result.value;
  });
}

test('C1: long names and 0, 1, 5, 12 Agents stay navigable with enlarged text', async ({}, info) => {
  await openAgentRoster(page);
  await expect(page.getByText('Add an agent to this Project.', { exact: true })).toBeVisible();
  await page.keyboard.press('Escape');
  const names = [
    'Researcher for comparing the treatment results with the reference samples',
    'Reviewer',
    'Workspace guide',
    'Evidence reviewer',
    'Question reviewer',
    ...Array.from({ length: 7 }, (_, i) => 'Analysis collaborator ' + (i + 6)),
  ];
  for (let i = 0; i < names.length; i++) {
    await addAgent(names[i]!);
    if (i === 0 || i === 4) {
      await openAgentRoster(page);
      await expect(page.locator('.agent-row')).toHaveCount(i + 1);
      await page.keyboard.press('Escape');
    }
  }
  const draft = page.getByRole('textbox', { name: 'Message draft' });
  await draft.fill('Keep the comparison draft while inspecting the team.');
  const before = await documentSnapshot();
  await viewport(900, 650, 1.5);
  await page.getByRole('button', { name: 'Chat', exact: true }).click();
  await openAgentRoster(page);
  const roster = page.getByRole('dialog', { name: 'Project agents', exact: true });
  await captureWindow(application, info.outputPath('C1-roster-large-text.png'));
  await expect(roster.locator('.agent-row')).toHaveCount(12);
  await withinViewport(roster);
  expect(await roster.evaluate((element) => element.scrollWidth <= element.clientWidth)).toBe(true);
  await roster.getByRole('button', { name: 'Settings for ' + names[0], exact: true }).click();
  await expect(page.getByLabel('Name', { exact: true })).toHaveValue(names[0]!);
  await page.keyboard.press('Escape');
  await openAgentRoster(page);
  await roster.getByRole('button', { name: 'Message ' + names.at(-1), exact: true }).focus();
  await page.keyboard.press('Space');
  await expect(draft).toBeFocused();
  await expect(draft).toHaveValue(before.chat.draft);
  await expect(page.getByRole('combobox', { name: 'Conversation recipient' })).toHaveValue(
    (await documentSnapshot()).workspace.agents.at(-1)!.agentId,
  );
  await withinViewport(page.getByRole('button', { name: 'Account', exact: true }));
  await projectBarDoesNotOverlap();
  expect((await documentSnapshot()).collaboration?.submissions ?? []).toHaveLength(0);
});

test('C2: long resources and multiple attachments retain a usable workspace at 100–150% zoom', async ({}, info) => {
  await addAgent('Reviewer');
  await page
    .getByRole('combobox', { name: 'Conversation recipient' })
    .selectOption({ label: 'To Reviewer' });
  const files = Array.from(
    { length: 4 },
    (_, i) => `Comparison-notes-for-treatment-and-reference-samples-round-${i + 1}.txt`,
  );
  for (const file of files)
    await writeFile(
      join(root, file),
      'Review the treatment and reference samples together.\n'.repeat(35),
    );
  await page.getByRole('button', { name: 'Refresh files', exact: true }).click();
  for (const file of files) {
    await page.getByRole('button', { name: file, exact: true }).click();
    await readyView(page);
    await page.getByRole('button', { name: 'Add view preview of ' + file + ' to message' }).click();
  }
  await page.getByRole('button', { name: 'Open quality.png in the other pane' }).click();
  await readyView(page, 'Secondary pane');
  const draft = page.getByRole('textbox', { name: 'Message draft' });
  const text =
    'Review the selected results, retain the earlier notes and explain the important differences.\n'.repeat(
      12,
    );
  await draft.fill(text);
  await expect(page.locator('.attachment-status')).toContainText(
    '4 attachments · Ready for Reviewer',
  );
  const before = await documentSnapshot();
  for (const [width, height, zoom] of [
    [1280, 808, 1],
    [900, 600, 1.25],
    [900, 650, 1.5],
  ]) {
    await viewport(width!, height!, zoom!);
    if (await page.getByRole('button', { name: 'Chat', exact: true }).isVisible())
      await page.getByRole('button', { name: 'Chat', exact: true }).click();
    await captureWindow(application, info.outputPath(`C2-chat-${zoom}.png`));
    await withinViewport(page.getByRole('button', { name: 'Account', exact: true }));
    await projectBarDoesNotOverlap();
    await withinViewport(page.getByRole('button', { name: 'Send', exact: true }));
    await expect(page.locator('textarea[aria-label="Message draft"]')).toHaveCount(1);
    const attachment = page
      .getByLabel('Message attachments', { exact: true })
      .getByRole('button', { name: new RegExp('^' + files[0]) });
    await attachment.click();
    await expect(attachmentPreview(page)).toContainText('Review the treatment');
    await withinViewport(
      attachmentPreview(page).getByRole('button', { name: /^Close Attachment:/ }),
    );
    await captureWindow(application, info.outputPath(`C2-preview-${zoom}.png`));
    await closeAttachmentPreview(page);
    await expect(attachment).toBeFocused();
    await expect(draft).toHaveValue(text);
    if (await page.getByRole('button', { name: 'Workspace', exact: true }).isVisible())
      await page.getByRole('button', { name: 'Workspace', exact: true }).click();
    await captureWindow(application, info.outputPath(`C2-workspace-${zoom}.png`));
    const pane = await readyView(page, 'Secondary pane');
    await openPaneActions(pane, 'quality.png');
    await withinViewport(pane.getByRole('group', { name: 'Actions for quality.png' }));
    await page.keyboard.press('Escape');
  }
  const after = await documentSnapshot();
  expect(after.chat).toEqual(before.chat);
  expect(after.workspace.surfaces).toEqual(before.workspace.surfaces);
  await application.close();
  ({ application, page } = await launch(profile, env()));
  await page.getByRole('button', { name: 'Chat', exact: true }).click();
  await expect(page.getByRole('textbox', { name: 'Message draft' })).toHaveValue(text);
  expect((await documentSnapshot()).chat).toEqual(before.chat);
});

test('C3: a pending reply and attachment error stay actionable in a small enlarged-text chat', async ({}, info) => {
  await page.getByRole('button', { name: 'quality.png', exact: true }).click();
  await readyView(page);
  await addAgent('Researcher', true);
  await page
    .getByRole('combobox', { name: 'Conversation recipient' })
    .selectOption({ label: 'To Researcher' });
  await writeFile(join(profile, 'codex/home/question-gate'), 'ready');
  await page.getByRole('textbox', { name: 'Message draft' }).fill('ask-question');
  await page.getByRole('button', { name: 'Send', exact: true }).click();
  const question = page.getByRole('article', { name: 'Question from Researcher', exact: true });
  await expect(question).toBeVisible();
  await expect(page.locator('.message-state').getByText('completed', { exact: true })).toHaveCount(
    1,
  );
  await question.getByRole('button', { name: 'Reply', exact: true }).click();
  const draft = page.getByRole('textbox', { name: 'Message draft' });
  await draft.fill('Keep this unsent answer while I inspect the question evidence.');
  await page.getByRole('button', { name: 'Add view preview of quality.png to message' }).click();
  await page.getByRole('combobox', { name: 'Message model' }).selectOption('fixture-text-model');
  await expect(page.getByRole('button', { name: 'Send', exact: true })).toBeDisabled();
  await viewport(900, 650, 1.5);
  await page.getByRole('button', { name: 'Chat', exact: true }).click();
  await captureWindow(application, info.outputPath('C3-reply-errors-large-text.png'));
  await withinViewport(page.getByRole('button', { name: 'Send', exact: true }));
  await expect(
    page.getByText('This model cannot receive the question’s images. Choose a supported model.', {
      exact: true,
    }),
  ).toBeInViewport({ ratio: 1 });
  await expect(page.getByRole('combobox', { name: 'Conversation recipient' })).toBeDisabled();
  const target = page.getByRole('region', { name: 'Reply target', exact: true });
  await target.getByRole('button', { name: /^quality.png/ }).click();
  await expect(attachmentPreview(page).getByRole('img')).toBeVisible();
  await closeAttachmentPreview(page);
  await target.getByRole('button', { name: 'Cancel reply' }).click();
  await expect(
    page
      .getByRole('status')
      .filter({ hasText: 'This model cannot receive images. Choose a supported model.' }),
  ).toBeInViewport({ ratio: 1 });
  await expect(page.getByRole('button', { name: 'Prepare again', exact: true })).toBeInViewport({
    ratio: 1,
  });
  await captureWindow(application, info.outputPath('C3-attachment-error-large-text.png'));
  await page.getByRole('combobox', { name: 'Message model' }).selectOption('fixture-model');
  await expect(page.getByRole('button', { name: 'Send', exact: true })).toBeEnabled();
  await expect(draft).toHaveValue('Keep this unsent answer while I inspect the question evidence.');
  expect((await documentSnapshot()).collaboration?.submissions).toHaveLength(1);
});
