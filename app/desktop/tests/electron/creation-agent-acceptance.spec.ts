import { test, expect } from '@playwright/test';
import { mkdtemp, mkdir, copyFile, chmod, readFile, writeFile, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join, resolve } from 'node:path';
import { launch, chooseProject, openAgentRoster } from './support';

// Opt-in account qualification, separate from the deterministic protocol peer.
// Only the explicitly selected local auth file enters this disposable profile;
// it is never logged, copied into evidence, or retained after the test.
test('Creation: signed-in Agent proposes, reads and points; User adopts without a Run', async ({}, info) => {
  test.skip(
    process.env.GOBBLE_REAL_CREATION !== '1' || !process.env.GOBBLE_CREATION_AUTH_FILE,
    'requires explicit local signed-in account qualification',
  );
  test.setTimeout(600_000);
  const base = await mkdtemp(join(tmpdir(), 'gobble-creation-agent-')),
    profile = join(base, 'profile'),
    project = join(base, 'Read quality acceptance'),
    home = join(profile, 'codex/home');
  await mkdir(project);
  await mkdir(home, { recursive: true, mode: 0o700 });
  await writeFile(join(project, 'sample.fastq.gz'), 'Metadata-only isolated acceptance data.');
  const credential = join(home, 'auth.json');
  await copyFile(process.env.GOBBLE_CREATION_AUTH_FILE!, credential);
  await chmod(credential, 0o600);
  let app: Awaited<ReturnType<typeof launch>> | undefined;
  let foreground: ReturnType<typeof setInterval> | undefined;
  try {
    app = await launch(profile, { GOBBLE_CODEX_EXECUTABLE: resolve('node_modules/.bin/codex') });
    const { application, page } = app;
    page.setDefaultTimeout(20000);
    const focus = async () => {
      await application.evaluate(({ app, BrowserWindow }) => {
        app.focus({ steal: true });
        const window = BrowserWindow.getAllWindows().find((w) =>
          w.webContents.getURL().startsWith('app://gobble/'),
        )!;
        window.show();
        window.focus();
      });
      await expect.poll(() => page.evaluate(() => document.hasFocus())).toBe(true);
    };
    // Live network turns outlast individual clicks. Keep the owned test window
    // physically foreground; never override Main's availability or evidence policy.
    foreground = setInterval(() => {
      void application
        .evaluate(({ app, BrowserWindow }) => {
          const window = BrowserWindow.getAllWindows().find((w) =>
            w.webContents.getURL().startsWith('app://gobble/'),
          );
          if (window && !window.isFocused()) {
            app.focus({ steal: true });
            window.show();
            window.focus();
          }
        })
        .catch(() => {});
    }, 250);
    await chooseProject(application, page, project);
    await page.getByRole('button', { name: 'New pipeline', exact: true }).click();
    const view = page.locator('.creation-view');
    await view.getByRole('button', { name: '◇ sample.fastq.gz', exact: true }).click();
    await view.getByRole('button', { name: 'Use selected file' }).click();
    await view.getByRole('button', { name: 'Connect engine', exact: true }).click();
    await expect(view.getByText('● Analysis engine connected', { exact: true })).toBeVisible({
      timeout: 45000,
    });
    await page.getByRole('button', { name: 'Account', exact: true }).click();
    await page.getByRole('button', { name: 'Refresh connection' }).click();
    await expect(page.getByText('ChatGPT connected', { exact: true })).toBeVisible({
      timeout: 45000,
    });
    await page.getByRole('button', { name: 'Close Codex account' }).click();
    await openAgentRoster(page);
    await page.getByRole('button', { name: 'Add agent', exact: true }).click();
    const editor = page.getByRole('dialog', { name: 'Add agent', exact: true });
    await editor.getByLabel('Name', { exact: true }).fill('Researcher');
    await editor.getByLabel('Project access').selectOption('sharedViews');
    const model = await editor.getByLabel('Model').inputValue();
    const effort = await editor.getByLabel('Reasoning effort').inputValue();
    await editor
      .getByLabel('Role instructions')
      .fill(
        'Discuss this analysis in concise English using the shared flow and its exact facts. Keep source code behind the UI.',
      );
    await editor.getByRole('button', { name: 'Add agent', exact: true }).click();
    await page
      .getByRole('combobox', { name: 'Conversation recipient' })
      .selectOption({ label: 'To Researcher' });
    await page
      .getByRole('checkbox', { name: 'Allow a new Pipeline proposal for this message' })
      .check();
    await page
      .getByRole('textbox', { name: 'Message draft' })
      .fill(
        'Please design a new analysis for the selected single-end reads: trim adapters and low-quality bases with a quality threshold of 25 Phred and minimum length of 40 bp, then inspect the trimmed reads with FastQC. Prepare the proposal through the provided creation tools so I can review the flow. Do not start an analysis.',
      );
    await focus();
    await page.getByRole('button', { name: 'Send', exact: true }).click();
    await expect(view.getByRole('heading', { name: 'No current version' })).toBeVisible({
      timeout: 240000,
    });
    await expect(view.locator('.pipeline-node[data-change="added-step"]')).toHaveCount(2);
    await view.getByRole('button', { name: /Inspect Trim adapters/ }).click();
    await view.getByRole('button', { name: /Quality threshold.*25/ }).click();
    await view.getByRole('button', { name: 'Add to message', exact: true }).click();
    await expect(page.locator('.composer-context')).toContainText('Quality threshold · 25 Phred');
    await page
      .getByRole('textbox', { name: 'Message draft' })
      .fill(
        'Read the exact attached proposal setting, briefly explain what this quality threshold means, and point to this exact setting using the creation review and point tools. Begin your pointer note with "Quality threshold reviewed:". This is discussion only; keep the proposal unchanged.',
      );
    await expect(page.getByRole('button', { name: 'Send', exact: true })).toBeEnabled({
      timeout: 90000,
    });
    await focus();
    await page.getByRole('button', { name: 'Send', exact: true }).click();
    await expect(
      view.getByRole('button', { name: /Researcher: Quality threshold reviewed:/ }),
    ).toBeVisible({
      timeout: 180000,
    });
    await expect(
      page.locator('.message-state small').filter({ hasText: /^completed$/ }),
    ).toHaveCount(2, { timeout: 90000 });
    const draftCatalog = JSON.parse(await readFile(join(profile, 'service/catalog.json'), 'utf8'));
    const selectedDraft = Object.values(draftCatalog.drafts)[0] as {
      projectId: string;
      draftId: string;
    };
    const shared = await page.evaluate((scope) => window.gobble.creation.state(scope), {
      projectId: selectedDraft.projectId,
      draftId: selectedDraft.draftId,
    });
    if (!shared.ok) throw new Error(shared.error.message);
    const exactMark = shared.value.marks.find((mark) =>
      mark.note.startsWith('Quality threshold reviewed:'),
    );
    expect(exactMark?.context.kind).toBe('candidate');
    if (exactMark?.context.kind !== 'candidate')
      throw new Error('Missing retained candidate reference');
    expect(exactMark.context.target?.kind).toBe('setting');
    if (exactMark.context.target?.kind !== 'setting')
      throw new Error('Agent did not point to the setting');
    const reference = exactMark.context;
    const target = reference.target;
    if (target?.kind !== 'setting') throw new Error('Missing setting target');
    const checked = shared.value.candidates.find(
      (candidate) => candidate.candidateId === reference.candidateId,
    )!;
    const selectedStep = checked.artifact!.check.review.flow.steps.find(
      (step) => step.id === target.id,
    )!;
    expect(selectedStep.settings.find((setting) => setting.key === target.name)).toMatchObject({
      label: 'Quality threshold',
      value: 25,
      unit: 'Phred',
    });
    await writeFile(
      info.outputPath('agent-exact-reference.json'),
      JSON.stringify(exactMark, null, 2),
    );
    await page.screenshot({ path: info.outputPath('agent-shared-review.png') });
    await view.getByRole('button', { name: 'Adopt as new pipeline', exact: true }).click();
    await view.getByRole('textbox', { name: 'Pipeline name' }).fill('Read quality acceptance');
    await view.getByRole('button', { name: 'Confirm creation' }).click();
    await expect(view.getByText('Created · No Run started.', { exact: true })).toBeVisible({
      timeout: 45000,
    });
    await page
      .getByRole('textbox', { name: 'Message draft' })
      .fill('Keep this question for execution preparation.');
    const catalog = JSON.parse(await readFile(join(profile, 'service/catalog.json'), 'utf8'));
    expect(catalog.pipelines).toHaveLength(1);
    expect(catalog.runs).toHaveLength(0);
    await writeFile(
      info.outputPath('acceptance.json'),
      JSON.stringify(
        {
          kind: 'real-signed-in-agent',
          model,
          effort,
          profile,
          project,
          pipelines: catalog.pipelines,
          birth: Object.values(catalog.drafts),
          runs: catalog.runs,
        },
        null,
        2,
      ),
    );
    clearInterval(foreground);
    foreground = undefined;
    await application.close();
    app = undefined;
    app = await launch(profile, { GOBBLE_CODEX_EXECUTABLE: resolve('node_modules/.bin/codex') });
    await expect(app.page.getByRole('textbox', { name: 'Message draft' })).toHaveValue(
      'Keep this question for execution preparation.',
    );
    await app.page.getByRole('button', { name: 'Open pipeline', exact: true }).click();
    await expect(app.page.locator('.creation-view')).toHaveCount(0);
    await expect(app.page.locator('.pipeline-node')).toHaveCount(3);
    await app.page.screenshot({ path: info.outputPath('agent-created-current.png') });
    expect(await readFile(join(project, 'sample.fastq.gz'), 'utf8')).toBe(
      'Metadata-only isolated acceptance data.',
    );
  } catch (error) {
    if (app) {
      await app.page
        .getByRole('button', { name: 'Close Codex account' })
        .click({ timeout: 1000 })
        .catch(() => {});
      await app.page.screenshot({ path: info.outputPath('failure.png') }).catch(() => {});
      await writeFile(
        info.outputPath('visible.txt'),
        await app.page.locator('body').innerText(),
      ).catch(() => {});
    }
    throw error;
  } finally {
    clearInterval(foreground);
    await app?.application.close().catch(() => {});
    await rm(credential, { force: true });
  }
});
