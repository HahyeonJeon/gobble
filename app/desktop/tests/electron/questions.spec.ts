import { focusWindow } from './support';
import {
  attachmentPreview,
  closeAttachmentPreview,
  openAgentRoster,
  chooseProject,
  launch,
  projectFixture,
  readyView,
} from './support';
import { test, expect, type ElectronApplication, type Page } from '@playwright/test';
import { chmod, copyFile, mkdtemp, readFile, readdir, rm, writeFile } from 'node:fs/promises';
import { join, resolve } from 'node:path';
import { tmpdir } from 'node:os';
import { isQuestion, parseWorkspaceDocument } from '@gobble/contracts';

let application: ElectronApplication;
let page: Page;
let base: string;
let profile: string;
let root: string;
let executable: string;
let gate: string;
const env = () => ({ GOBBLE_CODEX_EXECUTABLE: executable, GOBBLE_FIXTURE_QUESTION_GATE: gate });
test.beforeEach(async () => {
  base = await mkdtemp(join(tmpdir(), 'gobble-question-electron-'));
  profile = join(base, 'profile');
  gate = join(profile, 'codex/home/question-gate');
  executable = join(base, 'codex');
  await copyFile(resolve('desktop/tests/fixtures/codex.cjs'), executable);
  await chmod(executable, 0o700);
  ({ application, page } = await launch(profile, env()));
  root = await projectFixture(base);
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
async function addAgent(name: string) {
  await openAgentRoster(page);
  await page.getByRole('button', { name: 'Add agent', exact: true }).click();
  const dialog = page.getByRole('dialog', { name: 'Add agent', exact: true });
  await dialog.getByLabel('Name', { exact: true }).fill(name);
  await dialog.getByLabel('Project access').selectOption('sharedViews');
  await dialog.getByRole('button', { name: 'Add agent', exact: true }).click();
  await expect(dialog).not.toBeVisible();
}
async function send(text: string) {
  await focusWindow(application);
  await page.getByRole('textbox', { name: 'Message draft', exact: true }).fill(text);
  await page.getByRole('button', { name: 'Send', exact: true }).click();
  await expect(page.getByRole('textbox', { name: 'Message draft', exact: true })).toHaveValue('');
}
async function readDoc() {
  const directory = join(profile, 'workspace/projects');
  const names = await readdir(directory);
  const file = join(
    directory,
    names.find((name) => name.endsWith('.json'))!,
  );
  return { file, doc: parseWorkspaceDocument(JSON.parse(await readFile(file, 'utf8'))) };
}
async function reconnect() {
  await page.getByRole('button', { name: 'Account', exact: true }).click();
  await page.getByRole('button', { name: 'Refresh connection' }).click();
  await expect(page.getByText('ChatGPT connected', { exact: true })).toBeVisible();
  await page.getByRole('button', { name: 'Close Codex account' }).click();
}
const completed = () => page.locator('.message-state').getByText('completed', { exact: true });
const question = () => page.getByRole('article', { name: 'Question from Researcher', exact: true });
test('question arrival preserves a peer draft; Reply/Cancel and Enter use one composer with immutable evidence and one addressed answer', async ({}, info) => {
  await page.getByRole('button', { name: 'notes.txt', exact: true }).click();
  await readyView(page);
  await addAgent('Researcher');
  await addAgent('Reviewer');
  await page
    .getByRole('combobox', { name: 'Conversation recipient' })
    .selectOption({ label: 'To Researcher' });
  await send('ask-question');
  await page
    .getByRole('combobox', { name: 'Conversation recipient' })
    .selectOption({ label: 'To Reviewer' });
  await page
    .getByRole('textbox', { name: 'Message draft', exact: true })
    .fill('Keep S03; note the selected context.');
  await page.getByRole('button', { name: 'Add view preview of notes.txt to message' }).click();
  await expect(page.locator('.attachment-status')).toContainText('Ready for Reviewer');
  await expect(question()).toHaveCount(0);
  await focusWindow(application);
  await writeFile(gate, 'ready');
  await expect(question()).toBeVisible();
  await expect(completed()).toHaveCount(1);
  await expect(page.getByRole('textbox', { name: 'Message draft', exact: true })).toHaveValue(
    'Keep S03; note the selected context.',
  );
  await expect(page.getByRole('combobox', { name: 'Conversation recipient' })).toHaveValue(
    (await readDoc()).doc.workspace.agents.find((item) => item.name === 'Reviewer')!.agentId,
  );
  await expect(question().getByRole('radio')).toHaveCount(0);
  await expect(question()).toContainText('Suggestions: Keep S03 · Review S03 first');
  await question().getByRole('button', { name: 'Reply', exact: true }).focus();
  await page.keyboard.press('Space');
  await expect(page.getByRole('region', { name: 'Reply target', exact: true })).toContainText(
    'Replying to Researcher',
  );
  await expect(page.getByRole('combobox', { name: 'Conversation recipient' })).toBeDisabled();
  await openAgentRoster(page);
  await expect(page.getByRole('button', { name: 'Message Reviewer', exact: true })).toBeDisabled();
  await page.keyboard.press('Escape');
  await page.getByRole('button', { name: 'Cancel reply' }).click();
  await expect(page.getByRole('region', { name: 'Reply target', exact: true })).toHaveCount(0);
  await expect(page.getByRole('textbox', { name: 'Message draft', exact: true })).toHaveValue(
    'Keep S03; note the selected context.',
  );
  await question().getByRole('button', { name: 'Reply', exact: true }).click();
  await expect(page.locator('.attachment-status')).toContainText('Ready for Researcher');
  const target = page.getByRole('region', { name: 'Reply target', exact: true });
  await target.getByRole('button', { name: /^notes.txt/ }).click();
  await expect(attachmentPreview(page)).toContainText('Inspect sample S03.');
  await closeAttachmentPreview(page);
  await expect(page.locator('.chat-composer textarea')).toHaveCount(1);
  await expect(page.getByRole('button', { name: 'Send', exact: true })).toHaveCount(1);
  const sendBounds = await page.getByRole('button', { name: 'Send', exact: true }).boundingBox();
  expect(sendBounds!.y + sendBounds!.height).toBeLessThanOrEqual(
    await page.evaluate(() => innerHeight),
  );
  const avatarBounds = await question().locator('.avatar').boundingBox();
  expect(avatarBounds!.width).toBe(28);
  await page.screenshot({ path: info.outputPath('inline-question-reply.png') });
  await page.getByRole('textbox', { name: 'Message draft', exact: true }).press('Enter');
  await expect(completed()).toHaveCount(2);
  await expect(question()).toContainText('Answer recorded');
  await expect(target).toHaveCount(0);
  const { doc } = await readDoc();
  const q = doc.workspace.decisions.filter(isQuestion)[0]!;
  expect(q.state.kind).toBe('answered');
  expect(doc.collaboration!.submissions[1]!.replyToQuestionId).toBe(q.decisionId);
  expect(doc.collaboration!.submissions[1]!.agentId).toBe(q.requestedBy);
  const fixture = JSON.parse(
    await readFile(join(profile, 'codex/home/fixture-state.json'), 'utf8'),
  );
  const threads = Object.values(fixture.threads) as {
    name: string;
    turns: { items: { content?: unknown }[] }[];
  }[];
  expect(threads).toHaveLength(1);
  expect(threads[0]!.name).toBe('Researcher');
  expect(threads[0]!.turns).toHaveLength(2);
  expect(JSON.stringify(threads[0]!.turns[1])).toContain('Explicit user reply to this question');
  expect(JSON.stringify(threads[0]!.turns[1])).toContain('Inspect sample S03.');
});
test('pending reply restores without replay; changed source invalidates Send while historical question preview and draft remain', async () => {
  await page.getByRole('button', { name: 'notes.txt', exact: true }).click();
  await readyView(page);
  await addAgent('Researcher');
  await page
    .getByRole('combobox', { name: 'Conversation recipient' })
    .selectOption({ label: 'To Researcher' });
  await send('ask-question');
  await focusWindow(application);
  await writeFile(gate, 'ready');
  await expect(question()).toBeVisible();
  await expect(completed()).toHaveCount(1);
  await question().getByRole('button', { name: 'Reply', exact: true }).click();
  await page
    .getByRole('textbox', { name: 'Message draft', exact: true })
    .fill('Keep my unsent answer');
  await expect(page.locator('.composer-footer')).toContainText('Draft saved locally');
  await application.close();
  ({ application, page } = await launch(profile, env()));
  await expect(question()).toContainText('Awaiting your reply');
  await expect(page.getByRole('textbox', { name: 'Message draft', exact: true })).toHaveValue(
    'Keep my unsent answer',
  );
  await expect(page.getByRole('region', { name: 'Reply target', exact: true })).toBeVisible();
  expect((await readDoc()).doc.collaboration?.submissions).toHaveLength(1);
  await reconnect();
  await writeFile(join(root, 'notes.txt'), 'New content after the question.\n');
  await page.getByRole('button', { name: 'Send', exact: true }).click();
  await expect(question()).toContainText('Evidence changed');
  await expect(page.getByRole('textbox', { name: 'Message draft', exact: true })).toHaveValue(
    'Keep my unsent answer',
  );
  const target = page.getByRole('region', { name: 'Reply target', exact: true });
  await target.getByRole('button', { name: /^notes.txt/ }).click();
  await expect(attachmentPreview(page)).toContainText('Inspect sample S03.');
  await expect(attachmentPreview(page)).not.toContainText('New content after the question.');
  await closeAttachmentPreview(page);
  await page.getByRole('button', { name: 'Cancel reply' }).click();
  await expect(page.getByRole('textbox', { name: 'Message draft', exact: true })).toHaveValue(
    'Keep my unsent answer',
  );
  expect((await readDoc()).doc.collaboration?.submissions).toHaveLength(1);
});
test('image questions block text-only reply and dismissal sends nothing; older shared tool bindings require explicit new conversation', async () => {
  await page.getByRole('button', { name: 'quality.png', exact: true }).click();
  await readyView(page);
  await addAgent('Researcher');
  await page
    .getByRole('combobox', { name: 'Conversation recipient' })
    .selectOption({ label: 'To Researcher' });
  await send('ask-question');
  await focusWindow(application);
  await writeFile(gate, 'ready');
  await expect(question()).toBeVisible();
  await expect(completed()).toHaveCount(1);
  await question().getByRole('button', { name: 'Reply', exact: true }).click();
  await page
    .getByRole('textbox', { name: 'Message draft', exact: true })
    .fill('Keep this image reply');
  await page.getByRole('combobox', { name: 'Message model' }).selectOption('fixture-text-model');
  await expect(page.getByRole('button', { name: 'Send', exact: true })).toBeDisabled();
  await expect(
    page.getByText('This model cannot receive the question’s images. Choose a supported model.', {
      exact: true,
    }),
  ).toBeVisible();
  await question().getByRole('button', { name: 'Dismiss', exact: true }).click();
  await expect(question()).toContainText('Dismissed');
  await expect(page.getByRole('textbox', { name: 'Message draft', exact: true })).toHaveValue(
    'Keep this image reply',
  );
  await page.getByRole('button', { name: 'Cancel reply' }).click();
  await page.getByRole('combobox', { name: 'Message model' }).selectOption('fixture-model');
  await application.close();
  const saved = await readDoc();
  saved.doc.workspace.agents[0]!.provider.toolsetVersion = 'shared-views-v1';
  await writeFile(saved.file, JSON.stringify(saved.doc));
  const fixturePath = join(profile, 'codex/home/fixture-state.json');
  const fixture = JSON.parse(await readFile(fixturePath, 'utf8'));
  const priorThread = fixture.threads[saved.doc.workspace.agents[0]!.provider.threadId!];
  priorThread.tools = priorThread.tools.filter(
    (item: { name: string }) => item.name !== 'workspace_question',
  );
  await writeFile(fixturePath, JSON.stringify(fixture));
  ({ application, page } = await launch(profile, env()));
  await reconnect();
  await expect(page.getByRole('button', { name: 'Send', exact: true })).toBeDisabled();
  await expect(page.getByText(/Shared-view tools have changed/)).toBeVisible();
  await openAgentRoster(page);
  await page.getByRole('button', { name: 'Settings for Researcher' }).click();
  await expect(page.getByRole('dialog')).toContainText(
    'Questions need the updated shared-view tools',
  );
  await page.getByRole('button', { name: 'Start new conversation', exact: true }).click();
  await expect(page.getByRole('dialog')).not.toBeVisible();
  await expect(page.getByRole('textbox', { name: 'Message draft', exact: true })).toHaveValue(
    'Keep this image reply',
  );
  await page.getByRole('button', { name: 'Send', exact: true }).click();
  await expect(completed()).toHaveCount(2);
  const final = (await readDoc()).doc;
  expect(final.workspace.agents[0]!.provider.toolsetVersion).toBe('shared-views-v15');
  expect(final.collaboration!.submissions[0]!.threadId).not.toBe(
    final.collaboration!.submissions[1]!.threadId,
  );
  expect(final.workspace.decisions.filter(isQuestion)[0]!.state.kind).toBe('dismissed');
});
