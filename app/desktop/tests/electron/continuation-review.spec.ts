import { test, expect } from '@playwright/test';
import { readFile, writeFile, unlink } from 'node:fs/promises';
import { join } from 'node:path';
import { continuationFixture } from './continuation-fixture';
import { launch, chooseProject, readyView, openAgentRoster } from './support';

test('continuation Flow, exact capture, compact draft and explicit native confirmation', async ({}, info) => {
  test.setTimeout(150000);
  const fixture = await continuationFixture(info.outputPath('fixture'));
  let { application, page } = await launch(fixture.profile, fixture.environment);
  const commands = async () =>
    (await readFile(join(fixture.base, 'commands.jsonl'), 'utf8'))
      .trim()
      .split('\n')
      .map((s) => JSON.parse(s) as string[]);
  try {
    await application.evaluate(({ BrowserWindow }) =>
      BrowserWindow.getAllWindows()[0]!.setSize(1440, 950),
    );
    await chooseProject(application, page, fixture.root);
    await page.getByRole('button', { name: 'Open Run', exact: true }).click();
    await readyView(page);
    const flow = page.getByRole('region', { name: 'Run flow', exact: true });
    await expect(
      flow.getByRole('button', { name: 'Check continuation', exact: true }),
    ).toBeEnabled();
    await flow.getByRole('button', { name: 'Check continuation', exact: true }).click();
    await expect(flow.locator('[data-continuation="reuse"]')).toHaveCount(1, { timeout: 20000 });
    await expect(flow.locator('[data-continuation="restart"]')).toHaveCount(1);
    const node = flow.getByRole('button', { name: /Inspect Quality check/ });
    await node.focus();
    await page.keyboard.press('Enter');
    await expect(flow.getByRole('button', { name: 'Discuss this step' })).toBeEnabled();
    const draft = page.getByRole('textbox', { name: 'Message draft' });
    await draft.fill('Why does quality checking need to restart?');
    await flow.getByRole('button', { name: 'Discuss this step' }).click();
    await expect(page.locator('.attachment-chip')).toHaveCount(1);
    await expect(draft).toHaveValue('Why does quality checking need to restart?');
    await page.locator('.attachment-chip').getByRole('button').first().click();
    const preview = page.getByRole('dialog', { name: /^Attachment:/ });
    await expect(preview).toContainText('Will restart · Attempt 2');
    await expect(preview).toContainText('starts again from its beginning');
    await page.keyboard.press('Escape');
    await page.screenshot({ path: info.outputPath('continuation-ready-wide.png') });
    expect((await commands()).filter((a) => ['create', 'start'].includes(a[0]!))).toHaveLength(0);

    await application.evaluate(({ BrowserWindow }) =>
      BrowserWindow.getAllWindows()[0]!.setSize(1100, 800),
    );
    await page.getByRole('button', { name: 'Workspace', exact: true }).click();
    await expect(node).toBeVisible();
    await page.screenshot({ path: info.outputPath('continuation-compact-flow.png') });
    await page.getByRole('button', { name: 'Chat', exact: true }).click();
    await expect(draft).toHaveValue('Why does quality checking need to restart?');
    await page.screenshot({ path: info.outputPath('continuation-compact-chat.png') });
    await application.close();
    ({ application, page } = await launch(fixture.profile, fixture.environment));
    await application.evaluate(({ BrowserWindow }) =>
      BrowserWindow.getAllWindows()[0]!.setSize(1440, 950),
    );
    await expect(page.getByRole('textbox', { name: 'Message draft' })).toHaveValue(
      'Why does quality checking need to restart?',
    );
    expect((await commands()).filter((a) => ['create', 'start'].includes(a[0]!))).toHaveLength(0);
    await application.evaluate(({ BrowserWindow }) =>
      BrowserWindow.getAllWindows()[0]!.setSize(1440, 950),
    );
    await page.getByRole('button', { name: 'Resume analysis', exact: true }).click();
    await expect(page.getByRole('article', { name: 'Continue analysis' })).toContainText(
      'Continuation confirmed',
      { timeout: 20000 },
    );
    await page.getByRole('button', { name: 'Check continuation status' }).click();
    await expect(page.getByRole('article', { name: 'Continue analysis' })).toContainText(
      'Last checked execution: running',
    );
    await expect(page.getByRole('button', { name: 'Resume analysis' })).toHaveCount(0);
    await expect(page.getByRole('button', { name: 'Stop analysis', exact: true })).toHaveCount(1);
    expect((await commands()).filter((a) => a[0] === 'create')).toHaveLength(1);
    expect((await commands()).filter((a) => a[0] === 'start')).toHaveLength(1);
    await page.screenshot({ path: info.outputPath('continuation-confirmed.png') });
    await page.getByRole('button', { name: 'Stop analysis', exact: true }).click();
    await expect(page.getByRole('article', { name: 'Continue analysis' })).toContainText(
      'Stop settled',
    );
    const records = await page.evaluate(
      async (projectId) => window.gobble.continuations.list({ projectId }),
      fixture.projectId,
    );
    if (!records.ok) throw new Error(records.error.message);
    expect(records.value.at(-1)?.operation?.stopLease).toBe('b'.repeat(32));
    expect(records.value.at(-1)?.operation?.stopLease).not.toBe('a'.repeat(32));
  } finally {
    await application.close();
  }
});

test('older engine, blocked recheck, historical capture and unknown confirmation never silently resubmit', async ({}, info) => {
  test.setTimeout(150000);
  const fixture = await continuationFixture(info.outputPath('fixture'));
  await writeFile(join(fixture.base, 'old-engine'), 'true');
  let { application, page } = await launch(fixture.profile, fixture.environment);
  const creates = async () =>
    (await readFile(join(fixture.base, 'commands.jsonl'), 'utf8'))
      .trim()
      .split('\n')
      .map((s) => JSON.parse(s) as string[])
      .filter((a) => ['create', 'start'].includes(a[0]!));
  try {
    await application.evaluate(({ BrowserWindow }) =>
      BrowserWindow.getAllWindows()[0]!.setSize(1440, 950),
    );
    await chooseProject(application, page, fixture.root);
    await page.getByRole('button', { name: 'Open Run', exact: true }).click();
    const flow = page.getByRole('region', { name: 'Run flow', exact: true });
    await expect(flow).toContainText('original engine does not support continuation');
    await expect(
      flow.getByRole('button', { name: 'Check continuation', exact: true }),
    ).toBeDisabled();
    await unlink(join(fixture.base, 'old-engine'));
    await page.getByRole('button', { name: 'Close Analysis 1', exact: true }).click();
    await page.getByRole('button', { name: 'Open Run', exact: true }).click();
    await expect(
      flow.getByRole('button', { name: 'Check continuation', exact: true }),
    ).toBeEnabled();
    await flow.getByRole('button', { name: 'Check continuation', exact: true }).click();
    await expect(flow.locator('[data-continuation="restart"]')).toHaveCount(1, { timeout: 20000 });
    await flow.getByRole('button', { name: /Inspect Quality check/ }).click();
    await flow.getByRole('button', { name: 'Discuss this step' }).click();
    await writeFile(join(fixture.base, 'blocked'), 'true');
    await flow.getByRole('button', { name: 'Recheck continuation', exact: true }).click();
    await expect(page.getByRole('article', { name: 'Continue analysis' })).toContainText(
      'Continuation needs review',
      { timeout: 20000 },
    );
    await expect(page.getByRole('button', { name: 'Resume analysis' })).toHaveCount(0);
    await page.locator('.attachment-chip').getByRole('button').first().click();
    const preview = page.getByRole('dialog', { name: /^Attachment:/ });
    await expect(preview).toContainText('Will restart · Attempt 2');
    await expect(preview.getByRole('button', { name: 'Resume analysis' })).toHaveCount(0);
    await page.screenshot({ path: info.outputPath('historical-review-after-recheck.png') });
    await page.keyboard.press('Escape');
    await unlink(join(fixture.base, 'blocked'));
    await flow.getByRole('button', { name: 'Recheck continuation', exact: true }).click();
    await expect(page.getByRole('button', { name: 'Resume analysis' })).toBeEnabled({
      timeout: 20000,
    });
    await writeFile(join(fixture.base, 'lose-ack'), 'true');
    await page.getByRole('button', { name: 'Resume analysis' }).click();
    await expect(page.getByRole('article', { name: 'Continue analysis' })).toContainText(
      'acknowledgement is unavailable',
      { timeout: 20000 },
    );
    await expect(
      flow.getByRole('button', { name: 'Recheck continuation', exact: true }),
    ).toBeDisabled();
    await expect(page.getByRole('button', { name: 'Resume analysis' })).toHaveCount(0);
    await page.getByRole('button', { name: 'Check continuation status' }).click();
    await expect.poll(creates).toHaveLength(1);
    await expect(page.getByRole('button', { name: 'Check continuation status' })).toBeEnabled();
    await readyView(page);
    await page.screenshot({ path: info.outputPath('continuation-unknown.png') });
    await application.close();
    ({ application, page } = await launch(fixture.profile, fixture.environment));
    await application.evaluate(({ BrowserWindow }) =>
      BrowserWindow.getAllWindows()[0]!.setSize(1440, 950),
    );
    await expect(page.getByRole('article', { name: 'Continue analysis' })).toContainText(
      'acknowledgement is unavailable',
    );
    await page.getByRole('button', { name: 'Check continuation status' }).click();
    await expect(page.getByRole('button', { name: 'Resume analysis' })).toHaveCount(0);
    expect(await creates()).toHaveLength(1);
  } finally {
    await application.close();
  }
});

test('Agent points to the same continuation plan without changing User selection or starting work', async ({}, info) => {
  test.setTimeout(90000);
  const fixture = await continuationFixture(info.outputPath('fixture'));
  const { application, page } = await launch(fixture.profile, fixture.environment);
  try {
    await application.evaluate(({ BrowserWindow, shell }) => {
      BrowserWindow.getAllWindows()[0]!.setSize(1440, 950);
      shell.openExternal = async () => {};
    });
    await chooseProject(application, page, fixture.root);
    await page.getByRole('button', { name: 'Account', exact: true }).click();
    await page.getByRole('button', { name: 'Sign in with ChatGPT' }).click();
    await expect(page.getByText('ChatGPT connected', { exact: true })).toBeVisible();
    await page.getByRole('button', { name: 'Close Codex account' }).click();
    await openAgentRoster(page);
    await page.getByRole('button', { name: 'Add agent', exact: true }).click();
    const editor = page.getByRole('dialog', { name: 'Add agent', exact: true });
    await editor.getByLabel('Name', { exact: true }).fill('Researcher');
    await editor.getByLabel('Project access').selectOption('sharedViews');
    await editor.getByRole('button', { name: 'Add agent', exact: true }).click();
    await expect(editor).not.toBeVisible();
    await page
      .getByRole('combobox', { name: 'Conversation recipient' })
      .selectOption({ label: 'To Researcher' });
    await page.getByRole('button', { name: 'Open Run', exact: true }).click();
    const flow = page.getByRole('region', { name: 'Run flow', exact: true });
    await expect(
      flow.getByRole('button', { name: 'Check continuation', exact: true }),
    ).toBeEnabled();
    await flow.getByRole('button', { name: 'Check continuation', exact: true }).click();
    await expect(flow.locator('[data-continuation="restart"]')).toHaveCount(1, { timeout: 20000 });
    const selected = flow.getByRole('button', { name: /Inspect Trim reads/ });
    await selected.click();
    await page.getByRole('textbox', { name: 'Message draft' }).fill('continuation-point');
    await page.getByRole('button', { name: 'Send', exact: true }).click();
    await page.getByRole('textbox', { name: 'Message draft' }).fill('Keep my next question here.');
    const event = page
      .locator('.shared-reference-event')
      .filter({ hasText: 'This checked step restarts from its beginning.' });
    await expect(event).toBeVisible();
    await expect(selected).toHaveAttribute('aria-pressed', 'true');
    await expect(flow.locator('.pipeline-node[data-agent-mark="true"]')).toHaveCount(1);
    await event.getByRole('button', { name: 'Show in view', exact: true }).click();
    const reference = page.locator('.observed-reference-panel');
    await expect(reference).toContainText('Will restart · Attempt 2');
    await expect(reference.getByRole('button', { name: 'Resume analysis' })).toHaveCount(0);
    await page.screenshot({ path: info.outputPath('agent-shared-continuation.png') });
    await reference.getByRole('button', { name: 'Return to my view' }).click();
    await expect(selected).toHaveAttribute('aria-pressed', 'true');
    await expect(page.getByRole('textbox', { name: 'Message draft' })).toHaveValue(
      'Keep my next question here.',
    );
    const observed = JSON.parse(
      await readFile(join(fixture.profile, 'codex/home/continuation-delivery.json'), 'utf8'),
    );
    expect(observed.content.continuation.review.steps[1]).toMatchObject({
      action: 'restart',
      plannedAttempt: 2,
    });
    const commands = (await readFile(join(fixture.base, 'commands.jsonl'), 'utf8'))
      .trim()
      .split('\n')
      .map((s) => JSON.parse(s));
    expect(commands.filter((a) => ['create', 'start'].includes(a[0]))).toHaveLength(0);
  } finally {
    await application.close();
  }
});
