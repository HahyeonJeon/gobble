import { test, expect } from '@playwright/test';
import { mkdir } from 'node:fs/promises';
import { join } from 'node:path';
import { launch, chooseProject } from './support';

test('continuation preload and native catalog survive restart without execution', async ({}, info) => {
  const profile = info.outputPath('profile'),
    root = info.outputPath('Continuation study');
  await mkdir(root, { recursive: true });
  let session = await launch(profile, {
    GOBBLE_CODEX_EXECUTABLE: join(profile, 'unavailable-codex'),
  });
  try {
    await chooseProject(session.application, session.page, root);
    const projectId = await session.page.evaluate(async () => {
      const projects = await window.gobble.projects.list();
      if (!projects.ok || !projects.value[0]) throw new Error('Project unavailable');
      return projects.value[0].projectId;
    });
    const initial = await session.page.evaluate(async (projectId) => {
      return {
        frozen: Object.isFrozen(window.gobble.continuations),
        list: await window.gobble.continuations.list({ projectId }),
        invalid: await window.gobble.continuations.confirm({
          projectId,
          reviewId: 'req_missing',
          requestId: 'req_continue',
          reviewDigest: 'invalid',
        }),
      };
    }, projectId);
    expect(initial.frozen).toBe(true);
    expect(initial.list).toMatchObject({ ok: true, value: [] });
    expect(initial.invalid).toMatchObject({ ok: false, error: { code: 'invalid_request' } });
    await session.application.close();
    session = await launch(profile, {
      GOBBLE_CODEX_EXECUTABLE: join(profile, 'unavailable-codex'),
    });
    const restored = await session.page.evaluate(
      async (projectId) => ({
        list: await window.gobble.continuations.list({ projectId }),
        runs: await window.gobble.runs.list({ projectId }),
      }),
      projectId,
    );
    expect(restored.list).toMatchObject({ ok: true, value: [] });
    expect(restored.runs).toMatchObject({
      ok: true,
      value: { projectId, runs: [], candidates: [], truncated: false },
    });
  } finally {
    await session.application.close();
  }
});
