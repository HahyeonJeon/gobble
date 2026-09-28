import { verifyLiveContinuation } from './continuation-live-scenario';
import { verifyRunFollowUp } from './run-feedback-scenario';
import { gzipSync } from 'node:zlib';
import { test, expect } from '@playwright/test';
import { mkdtemp, mkdir, copyFile, chmod, readFile, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join, resolve } from 'node:path';
import { launch, chooseProject, openAgentRoster } from './support';

for (const finish of ['discard', 'adopt', 'prepare', 'launch'] as const)
  test(`Creation: chosen data, exact discussion, restart and ${finish}`, async ({}, info) => {
    test.skip(
      process.env.GOBBLE_CREATION_LIVE !== '1',
      'Requires installed labeled creation engine',
    );
    test.skip(
      (finish === 'prepare' || finish === 'launch') && !process.env.GOBBLE_PREPARATION_IMAGE,
      'Requires qualified preparation image',
    );
    test.setTimeout(
      process.env.GOBBLE_FEEDBACK_LIVE === '1' || process.env.GOBBLE_CONTINUATION_LIVE === '1'
        ? 1_200_000
        : 600_000,
    );
    const base = await mkdtemp(join(tmpdir(), 'gobble-creation-ui-')),
      project = join(base, 'Read quality study'),
      profile = join(base, 'profile'),
      executable = join(base, 'codex');
    await mkdir(project);
    await writeFile(
      join(project, 'sample.fastq.gz'),
      finish === 'launch'
        ? gzipSync(
            Array.from(
              { length: process.env.GOBBLE_CONTINUATION_LIVE === '1' ? 1_000_000 : 100 },
              (_, i) => `@synthetic-${i}\n${'ACGT'.repeat(20)}\n+\n${'I'.repeat(80)}\n`,
            ).join(''),
          )
        : 'Metadata-only selected research fixture.',
    );
    await copyFile(resolve('desktop/tests/fixtures/codex.cjs'), executable);
    await chmod(executable, 0o700);
    await copyFile(
      resolve('desktop/tests/fixtures/pipeline-creation-tools.cjs'),
      join(base, 'pipeline-tools.cjs'),
    );
    await writeFile(
      info.outputPath('fixture.json'),
      JSON.stringify({ base, project, profile }, null, 2),
    );
    let { application, page } = await launch(profile, { GOBBLE_CODEX_EXECUTABLE: executable });
    try {
      page.setDefaultTimeout(15000);
      await chooseProject(application, page, project);
      await page.getByRole('button', { name: 'New pipeline', exact: true }).click();
      const view = page.locator('.creation-view');
      await expect(view.getByRole('heading', { name: 'Choose your data' })).toBeVisible();
      await view.getByRole('button', { name: '◇ sample.fastq.gz', exact: true }).click();
      await view.getByRole('button', { name: 'Use selected file' }).click();
      await expect(page.locator('.composer-context')).toContainText('New pipeline · Selected data');
      if (finish === 'prepare' || (finish === 'launch' && process.env.GOBBLE_PREPARATION_IMAGE))
        await page
          .getByRole('combobox', { name: 'Analysis engine', exact: true })
          .selectOption(
            (process.env.GOBBLE_PREPARATION_SOURCE_IMAGE ?? process.env.GOBBLE_PREPARATION_IMAGE)!,
          );
      await view.getByRole('button', { name: 'Connect engine', exact: true }).click();
      await expect(view.getByText('● Analysis engine connected', { exact: true })).toBeVisible({
        timeout: 45000,
      });
      await page.screenshot({ path: info.outputPath('creation-ready.png') });
      await application.evaluate(({ shell }) => {
        shell.openExternal = async () => {};
      });
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
      await page
        .getByRole('combobox', { name: 'Conversation recipient' })
        .selectOption({ label: 'To Researcher' });
      await page
        .getByRole('checkbox', { name: 'Allow a new Pipeline proposal for this message' })
        .check();
      await page.getByRole('textbox', { name: 'Message draft' }).fill('pipeline-creation');
      await page.getByRole('button', { name: 'Send', exact: true }).click();
      const home = join(profile, 'codex/home');
      await expect
        .poll(() => readFile(join(home, 'creation-delivery.json'), 'utf8').catch(() => ''), {
          timeout: 35000,
        })
        .not.toBe('');
      await expect(view.getByRole('heading', { name: 'No current version' })).toBeVisible({
        timeout: 180000,
      });
      await expect(view.locator('.pipeline-node[data-change="added-step"]')).toHaveCount(2);
      await view.getByRole('button', { name: /Inspect Trim adapters/ }).click();
      await view.getByRole('button', { name: /Quality threshold.*25/ }).click();
      await expect(
        view.getByRole('heading', { name: 'Quality threshold', exact: true }),
      ).toBeVisible();
      await view.getByRole('button', { name: 'Add to message', exact: true }).click();
      await expect(page.locator('.composer-context')).toContainText('Quality threshold · 25 Phred');
      await view.getByRole('button', { name: /Inspect Inspect trimmed read quality/ }).click();
      await expect(page.locator('.composer-context')).toContainText('Quality threshold · 25 Phred');
      await application.evaluate(({ app, BrowserWindow }) => {
        app.focus({ steal: true });
        BrowserWindow.getAllWindows()[0]!.focus();
      });
      await page.getByRole('textbox', { name: 'Message draft' }).fill('pipeline-creation-review');
      await page.getByRole('button', { name: 'Send', exact: true }).click();
      await expect
        .poll(() => readFile(join(home, 'creation-review.json'), 'utf8').catch(() => ''), {
          timeout: 35000,
        })
        .not.toBe('');
      await expect(
        view.getByRole('button', { name: /Researcher: Quality threshold/ }),
      ).toBeVisible();
      await expect(
        view.getByRole('heading', { name: 'Inspect trimmed read quality', exact: true }),
      ).toBeVisible();
      await view.getByRole('button', { name: /Researcher: Quality threshold/ }).click();
      await page.screenshot({ path: info.outputPath('creation-reviewed.png') });
      await page
        .getByRole('textbox', { name: 'Message draft' })
        .fill('Keep this unsent research question.');
      await application.close();
      ({ application, page } = await launch(profile, { GOBBLE_CODEX_EXECUTABLE: executable }));
      await expect(page.getByRole('textbox', { name: 'Message draft' })).toHaveValue(
        'Keep this unsent research question.',
      );
      await expect(page.getByRole('heading', { name: 'No current version' })).toBeVisible();
      const catalog = JSON.parse(await readFile(join(profile, 'service/catalog.json'), 'utf8'));
      expect(catalog.pipelines).toHaveLength(0);
      expect(catalog.runs).toHaveLength(0);
      expect(catalog.schemaVersion).toBe(5);
      const draft = Object.values(catalog.drafts)[0] as {
        generation: number;
        input: { resourceId: string };
      };
      expect(draft.input.resourceId).toBeTruthy();
      await page
        .locator('.creation-view')
        .getByRole('button', { name: 'Discuss this draft' })
        .click();
      await page
        .locator('.composer-context')
        .getByRole('button', { name: 'Remove', exact: true })
        .click();
      await expect(page.locator('.creation-summary')).toContainText('sample.fastq.gz');
      await page.setViewportSize({ width: 1000, height: 850 });
      await page.screenshot({ path: info.outputPath('creation-compact.png') });
      await page.setViewportSize({ width: 1280, height: 808 });
      if (finish !== 'discard') {
        const adoption = page.getByRole('region', { name: 'Creation adoption', exact: true });
        await adoption.getByRole('button', { name: 'Adopt as new pipeline' }).click();
        await adoption.getByRole('textbox', { name: 'Pipeline name' }).fill('Read quality study');
        await expect(adoption).toContainText('sample.fastq.gz');
        await expect(adoption).toContainText('25 Phred');
        await page.screenshot({ path: info.outputPath('creation-confirm.png') });
        await adoption.getByRole('button', { name: 'Confirm creation' }).click();
        await expect(adoption).toContainText('Created · No Run started.', { timeout: 45000 });
        await expect(
          page
            .getByRole('region', { name: 'Creation adoption' })
            .getByRole('button', { name: 'Open pipeline' }),
        ).toBeVisible();
        await page.screenshot({ path: info.outputPath('creation-adopted.png') });
        const stored = JSON.parse(await readFile(join(profile, 'service/catalog.json'), 'utf8'));
        expect(stored.pipelines).toHaveLength(1);
        expect(stored.runs).toHaveLength(0);
        const birth = Object.values(stored.drafts)[0] as {
          draftId: string;
          adoption: {
            requestId: string;
            pipelineId: string;
            candidateId: string;
            artifactId: string;
            generation: number;
            name: string;
          };
        };
        const current = stored.revisions[birth.adoption.pipelineId];
        expect(current.creation).toEqual({
          draftId: birth.draftId,
          candidateId: birth.adoption.candidateId,
        });
        expect(current.artifact.artifactId).toBe(birth.adoption.artifactId);
        const retry = await page.evaluate(
          async ({ projectId, draftId, adoption }) =>
            window.gobble.creation.adopt({
              projectId,
              draftId,
              requestId: adoption.requestId,
              candidateId: adoption.candidateId,
              artifactId: adoption.artifactId,
              expectedGeneration: adoption.generation,
              name: adoption.name,
            }),
          {
            projectId: stored.pipelines[0].projectId,
            draftId: birth.draftId,
            adoption: birth.adoption,
          },
        );
        expect(retry.ok).toBe(true);
        await application.close();
        ({ application, page } = await launch(profile, { GOBBLE_CODEX_EXECUTABLE: executable }));
        await expect(page.getByRole('textbox', { name: 'Message draft' })).toHaveValue(
          'Keep this unsent research question.',
        );
        await expect(page.getByRole('heading', { name: 'No current version' })).toBeVisible();
        await page.getByRole('button', { name: 'Open pipeline', exact: true }).click();
        await expect(page.locator('.pipeline-node')).toHaveCount(3);
        await expect(page.getByRole('heading', { name: 'No current version' })).not.toBeVisible();
        await expect(page.getByRole('textbox', { name: 'Message draft' })).toHaveValue(
          'Keep this unsent research question.',
        );
        await page.screenshot({ path: info.outputPath('creation-current.png') });
        if (finish === 'prepare' || finish === 'launch') {
          await expect(page.locator('.creation-view')).toHaveCount(0);
          const review = page.getByRole('region', { name: 'Run preparation' });
          await page
            .getByRole('combobox', { name: 'Preparation engine', exact: true })
            .selectOption(process.env.GOBBLE_PREPARATION_IMAGE!);
          await review.getByRole('button', { name: 'Prepare run', exact: true }).click();
          await expect(
            review.getByRole('button', { name: 'Check data and tools', exact: true }),
          ).toBeVisible({ timeout: 150000 });
          await expect(review).toContainText('25 Phred');
          if (finish === 'launch') {
            await review.getByRole('button', { name: 'Check data and tools', exact: true }).click();
            const card = page.getByRole('article', { name: 'Confirm analysis' });
            await expect(
              card.getByRole('button', { name: 'Start analysis', exact: true }),
            ).toBeEnabled({ timeout: 90000 });
            await expect(card).toContainText(
              process.env.GOBBLE_CONTINUATION_LIVE === '1'
                ? '1,000,000 records checked'
                : '100 records checked',
            );
            await expect(review).toContainText('Checked · Confirm in Chat');
            await page.screenshot({ path: info.outputPath('launch-ready.png') });
            await card.getByRole('button', { name: 'Start analysis', exact: true }).click();
            const run = page.getByRole('article', { name: 'Analysis run' });
            await expect(run.getByRole('button', { name: 'Open Run', exact: true })).toBeVisible({
              timeout: 90000,
            });
            await run.getByRole('button', { name: 'Open Run', exact: true }).click();
            await expect(page.getByRole('region', { name: 'Run flow', exact: true })).toBeVisible();
            if (process.env.GOBBLE_CONTINUATION_LIVE === '1') {
              await verifyLiveContinuation({ application, page, profile, base, project, info });
              return;
            }
            await expect(run).toContainText('succeeded', { timeout: 180000 });
            await expect(
              page.locator('.retained-run-flow .pipeline-node[data-run-status="succeeded"]'),
            ).toHaveCount(2, { timeout: 20000 });
            await page.screenshot({ path: info.outputPath('launch-finished.png') });
            const node = page
              .locator('.retained-run-flow .pipeline-node[data-run-status="succeeded"]')
              .first();
            await expect(node).toBeEnabled({ timeout: 20000 });
            await node.click();
            await expect(
              page.getByRole('button', { name: 'Discuss task', exact: true }),
            ).toBeEnabled();
            await page.getByRole('button', { name: 'Discuss task', exact: true }).click();
            await expect(page.locator('.chat-composer')).toContainText('Attempt');
            await writeFile(
              info.outputPath('launch-reviews.json'),
              JSON.stringify(
                await page.evaluate(async () => {
                  const projects = await window.gobble.projects.list();
                  if (!projects.ok || !projects.value[0]) throw Error('Project missing');
                  return window.gobble.launches.list({ projectId: projects.value[0].projectId });
                }),
                null,
                2,
              ),
            );
            if (process.env.GOBBLE_FEEDBACK_LIVE === '1')
              await verifyRunFollowUp({ application, page, profile, base, info });
            return;
          }

          await review.getByRole('button', { name: 'Data', exact: true }).click();
          await expect(review.locator('.preparation-file')).toBeVisible();
          await expect(review).toContainText('sample.fastq.gz');
          await expect(review).toContainText(
            'Data contents and FASTQ validity have not been checked or copied.',
          );
          await review.getByRole('button', { name: 'Environment', exact: true }).click();
          await expect(
            review.getByText('One processing task at a time.', { exact: true }),
          ).toBeVisible();
          await review.getByRole('button', { name: 'Settings', exact: true }).click();
          await review.getByRole('button', { name: 'Discuss settings', exact: true }).click();
          await expect(page.locator('.chat-composer .review-context')).toContainText(
            'Run review · settings',
          );
          const message = page.getByRole('textbox', { name: 'Message draft' });
          await page.getByRole('button', { name: 'Account', exact: true }).click();
          await page.getByRole('button', { name: 'Refresh connection', exact: true }).click();
          await expect(page.getByText('ChatGPT connected', { exact: true })).toBeVisible();
          await page.getByRole('button', { name: 'Close Codex account', exact: true }).click();
          await message.fill('pipeline-preparation-review');
          await application.evaluate(({ app, BrowserWindow }) => {
            app.focus({ steal: true });
            BrowserWindow.getAllWindows()[0]?.focus();
          });
          await page.getByRole('button', { name: 'Send', exact: true }).click();
          await expect(review.locator('.preparation-mark')).toContainText(
            'This saved plan uses the reviewed quality threshold.',
            { timeout: 30000 },
          );
          await message.fill('Keep discussing these settings.');
          await page.screenshot({ path: info.outputPath('preparation-review.png') });
          const saved = JSON.parse(
            await readFile(
              join(profile, 'workspace/projects', stored.pipelines[0].projectId + '.json'),
              'utf8',
            ).catch(
              async () =>
                await readFile(
                  join(profile, 'workspaces/projects', stored.pipelines[0].projectId + '.json'),
                  'utf8',
                ),
            ),
          );
          await writeFile(
            info.outputPath('preparation-workspace.json'),
            JSON.stringify(saved, null, 2),
          );
          const listed = await page.evaluate(
            async (input) => window.gobble.preparations.list(input),
            { projectId: stored.pipelines[0].projectId, pipelineId: birth.adoption.pipelineId },
          );
          expect(listed.ok).toBe(true);
          if (!listed.ok) throw Error('Missing preparation');
          expect(listed.value).toHaveLength(1);
          expect(JSON.stringify(listed)).not.toContain('"payload"');
          const exact = listed.value[0]!;
          const retry = await page.evaluate(async (v) => window.gobble.preparations.prepare(v), {
            projectId: exact.projectId,
            pipelineId: exact.pipelineId,
            requestId: exact.requestId,
            artifactId: exact.artifactId,
            engineId: exact.engineId,
          });
          expect(retry.ok).toBe(true);
          await writeFile(
            info.outputPath('preparation-result.json'),
            JSON.stringify(listed, null, 2),
          );
          await application.close();
          ({ application, page } = await launch(profile, { GOBBLE_CODEX_EXECUTABLE: executable }));
          await expect(page.getByRole('textbox', { name: 'Message draft' })).toHaveValue(
            'Keep discussing these settings.',
          );
          await page.getByRole('button', { name: '▸ Run review', exact: true }).click();
          await expect(page.locator('.preparation-mark')).toContainText(
            'This saved plan uses the reviewed quality threshold.',
          );
          await writeFile(
            join(project, 'sample.fastq.gz'),
            'Changed metadata fixture after review.',
          );
          await expect(page.getByRole('region', { name: 'Run preparation' })).toContainText(
            'This review belongs to an earlier observation.',
            { timeout: 15000 },
          );
          await page.screenshot({ path: info.outputPath('preparation-stale.png') });
          await page
            .getByRole('region', { name: 'Run preparation' })
            .getByRole('button', { name: 'Prepare again', exact: true })
            .click();
          await expect(
            page
              .getByRole('region', { name: 'Run preparation' })
              .getByText('Prepared review', { exact: true }),
          ).toBeVisible({ timeout: 150000 });
          await expect(
            page.getByRole('combobox', { name: 'Run review history' }).locator('option'),
          ).toHaveCount(2);
          await expect(page.locator('.chat-composer .review-context')).toHaveCount(0);
          await page.getByRole('button', { name: 'Discuss this version', exact: true }).click();
          await expect(page.locator('.chat-composer .review-context')).toContainText(
            'Run review · settings',
          );
          const finalStored = JSON.parse(
            await readFile(join(profile, 'workspace/projects', exact.projectId + '.json'), 'utf8'),
          );
          expect(finalStored.chat.pipelineReview.preparationId).toBe(exact.requestId);
          expect(
            finalStored.collaboration.submissions.find(
              (s: { text: string }) => s.text === 'pipeline-preparation-review',
            ).pipelineReview.preparationId,
          ).toBe(exact.requestId);
          const finalReviews = await page.evaluate(
            async (v) => window.gobble.preparations.list(v),
            { projectId: exact.projectId, pipelineId: exact.pipelineId },
          );
          expect(finalReviews.ok).toBe(true);
          if (!finalReviews.ok) throw Error('Missing history');
          expect(finalReviews.value.filter((v) => v.state === 'ready' && v.fresh)).toHaveLength(1);
          await writeFile(
            info.outputPath('preparation-history.json'),
            JSON.stringify(finalReviews, null, 2),
          );
          await page.screenshot({ path: info.outputPath('preparation-new-review.png') });
        }

        const final = JSON.parse(await readFile(join(profile, 'service/catalog.json'), 'utf8'));
        expect(final.pipelines).toHaveLength(1);
        expect(final.runs).toHaveLength(0);
        return;
      }
      await page
        .locator('.creation-view')
        .getByRole('button', { name: 'Discard draft', exact: true })
        .click();
      await expect(page.locator('.creation-toolbar')).toContainText('Discarded draft');
      await expect(page.getByRole('textbox', { name: 'Message draft' })).toHaveValue(
        'Keep this unsent research question.',
      );
      expect(await readFile(join(project, 'sample.fastq.gz'), 'utf8')).toBe(
        'Metadata-only selected research fixture.',
      );
    } catch (error) {
      await page.screenshot({ path: info.outputPath('failure.png') }).catch(() => {});
      await page
        .locator('body')
        .innerText()
        .then((text) => writeFile(info.outputPath('visible.txt'), text))
        .catch(() => {});
      throw error;
    } finally {
      await application.close().catch(() => {});
    }
  });
