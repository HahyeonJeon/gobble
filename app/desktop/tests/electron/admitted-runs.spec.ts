import { test, expect } from '@playwright/test';
import { writeFile } from 'node:fs/promises';
import { join } from 'node:path';
import { continuationFixture } from './continuation-fixture';
import { launch, chooseProject } from './support';

test('Run navigation refreshes when a confirmed admission arrives, not on every status poll', async ({}, info) => {
  test.setTimeout(90000);
  const fixture = await continuationFixture(info.outputPath('fixture'));
  const { application, page } = await launch(fixture.profile, fixture.environment);
  try {
    await chooseProject(application, page, fixture.root);
    const probe = join(fixture.profile, 'admission-probe.cjs');
    // Test-owned preload models delayed admission delivery over the real bridge.
    // It never submits work or changes native records; only list responses are gated.
    await writeFile(
      probe,
      `
      const {contextBridge, ipcRenderer} = require('electron');
      const invoke = ipcRenderer.invoke.bind(ipcRenderer);
      let admitted = false, lists = 0, polls = 0;
      ipcRenderer.invoke = async (channel, ...args) => {
        const result = await invoke(channel, ...args);
        if (channel === 'gobble:runs:list:v1') {
          lists++;
          if (!admitted && result.ok) return {...result, value: {...result.value, runs: []}};
        }
        if (channel === 'gobble:launch:list:v1') {
          polls++;
          if (!admitted && result.ok) return {...result, value: []};
        }
        return result;
      };
      contextBridge.exposeInMainWorld('admissionProbe', {
        release: () => { admitted = true; },
        counts: () => ({lists, polls}),
      });
    `,
    );
    await application.evaluate(({ session }, filePath) => {
      session.defaultSession.registerPreloadScript({ type: 'frame', filePath });
    }, probe);
    await page.reload();
    await expect(page.getByText('No runs attached', { exact: true })).toBeVisible();
    const counts = () =>
      page.evaluate(() =>
        (
          window as unknown as { admissionProbe: { counts(): { lists: number; polls: number } } }
        ).admissionProbe.counts(),
      );
    const initial = await counts();
    await page.evaluate(() =>
      (window as unknown as { admissionProbe: { release(): void } }).admissionProbe.release(),
    );
    const navigation = page.getByRole('complementary', { name: 'Project navigation' });
    await expect(navigation.getByRole('button', { name: 'Analysis 1', exact: true })).toBeVisible({
      timeout: 10000,
    });
    const admitted = await counts();
    expect(admitted.lists).toBe(initial.lists + 1);
    await expect
      .poll(async () => (await counts()).polls, { timeout: 12000 })
      .toBeGreaterThanOrEqual(admitted.polls + 2);
    expect((await counts()).lists).toBe(admitted.lists);
    await page.screenshot({ path: info.outputPath('admitted-run-navigation.png') });
  } finally {
    await application.close();
  }
});
