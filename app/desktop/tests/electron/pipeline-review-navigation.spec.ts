import { test, expect } from '@playwright/test';
import { mkdir, readFile, writeFile, rm } from 'node:fs/promises';
import { join } from 'node:path';
import type { PipelineArtifact, PipelineProposal, WorkspaceAction } from '@gobble/contracts';
import { continuationFixture } from './continuation-fixture';
import { launch, chooseProject, readyView } from './support';

// Stored synthetic comparisons qualify renderer navigation, not engine checking or adoption.
test('comparison location survives presentation changes without retaining authority or confirmation', async ({}, info) => {
  test.setTimeout(120000);
  const fixture = await continuationFixture(info.outputPath('fixture'));
  const catalog = JSON.parse(await readFile(join(fixture.profile, 'service/catalog.json'), 'utf8'));
  const pipelineId: string = fixture.original.value.pipelineId;
  const artifact: PipelineArtifact = catalog.revisions[pipelineId].artifact;
  const proposalRoot = join(fixture.profile, 'service/pipeline-proposals', pipelineId);
  const proposal = (number: number): PipelineProposal => {
    const base = structuredClone(artifact);
    const proposed = structuredClone(artifact);
    if (base.flow.schemaVersion !== 2 || proposed.flow.schemaVersion !== 2)
      throw Error('Expected setting-aware fixture');
    base.flow.steps[0]!.settings = [
      { key: 'quality', label: 'Quality threshold', unit: 'Phred', value: 20 },
      { key: 'length', label: 'Minimum length', unit: 'bases', value: 30 },
    ];
    proposed.flow.steps[0]!.settings = [
      { key: 'quality', label: 'Quality threshold', unit: 'Phred', value: 25 },
      { key: 'length', label: 'Minimum length', unit: 'bases', value: 40 + number },
    ];
    return {
      projectId: fixture.projectId,
      pipelineId,
      proposalId: `req_navigation_${number}`,
      createdAt: `2026-09-28T00:00:0${number}Z`,
      digest: 'sha256:' + String(number).repeat(64),
      state: 'ready',
      summary: `Review option ${number}`,
      issue: '',
      managed: true,
      base,
      proposed: { ...proposed, artifactId: 'sha256:' + String(number).repeat(64) },
      comparison: {
        schemaVersion: 1,
        gaps: [],
        changes: [
          {
            id: 'quality',
            kind: 'setting',
            stepId: 'trim',
            key: 'quality',
            label: 'Quality threshold',
            unit: 'Phred',
            before: 20,
            after: 25,
          },
          {
            id: 'length',
            kind: 'setting',
            stepId: 'trim',
            key: 'length',
            label: 'Minimum length',
            unit: 'bases',
            before: 30,
            after: 40 + number,
          },
        ],
      },
    };
  };
  const save = async (value: PipelineProposal) => {
    const path = join(proposalRoot, value.proposalId);
    await mkdir(path, { recursive: true });
    await writeFile(join(path, 'review.json'), JSON.stringify(value));
  };
  await save(proposal(1));
  await save(proposal(2));
  let { application, page } = await launch(fixture.profile, fixture.environment);
  const command = async (action: WorkspaceAction) => {
    const result = await page.evaluate(
      async ({ projectId, action }) => {
        const current = await window.gobble.workspace.read({ projectId });
        if (!current.ok) throw Error(current.error.message);
        return window.gobble.workspace.command({
          projectId,
          expectedRevision: current.value.workspace.revision,
          requestId: 'req_' + crypto.randomUUID(),
          action,
        });
      },
      { projectId: fixture.projectId as string, action },
    );
    if (!result.ok) throw Error(result.error.message);
    return result.value;
  };
  const open = (pane: 'primary' | 'secondary') =>
    command({ kind: 'open', resource: { kind: 'pipeline', pipelineId }, pane, duplicate: true });
  const review = () =>
    page
      .getByRole('region', { name: 'Primary pane', exact: true })
      .getByRole('region', { name: 'Pipeline change review' });
  const retained = async () => {
    await expect(review().getByLabel('Proposal history')).toHaveValue('req_navigation_1');
    await expect(review().getByRole('button', { name: '2 Minimum length' })).toHaveAttribute(
      'aria-pressed',
      'true',
    );
    await expect(review().getByRole('region', { name: 'Selected change' })).toContainText(
      '41 bases',
    );
    await expect(page.locator('#primary-pane [data-testid="pipeline-view"]')).toHaveCount(0);
  };
  try {
    await application.evaluate(({ BrowserWindow }) => {
      BrowserWindow.getAllWindows()[0]!.setSize(1440, 950);
      BrowserWindow.getAllWindows()[0]!.webContents.setZoomFactor(0.8);
    });
    page.setDefaultTimeout(10000);
    await chooseProject(application, page, fixture.root);
    await open('primary');
    await readyView(page);
    await open('secondary');
    await readyView(page, 'Secondary pane');
    const primary = page.getByRole('region', { name: 'Primary pane', exact: true });
    await primary.getByRole('button', { name: 'Changes', exact: true }).click();
    await review().getByLabel('Proposal history').selectOption('req_navigation_1');
    await review().getByRole('button', { name: '2 Minimum length' }).click();
    await review().getByRole('button', { name: 'Adopt proposal', exact: true }).click();
    await expect(review().getByRole('button', { name: 'Confirm adoption' })).toBeVisible();
    await expect(page.getByRole('separator', { name: 'Resize panes' })).toBeVisible();
    await primary.getByRole('button', { name: 'Maximize primary pane' }).click();
    await expect(page.getByRole('region', { name: 'Secondary pane', exact: true })).toHaveCount(0);
    await retained();
    await expect(review().getByRole('button', { name: 'Confirm adoption' })).toHaveCount(0);
    await primary.getByRole('button', { name: 'Restore panes' }).click();
    await retained();
    await readyView(page, 'Secondary pane');
    await retained();
    await review().getByRole('region', { name: 'Selected change' }).scrollIntoViewIfNeeded();
    await page.screenshot({ path: info.outputPath('retained-wide.png') });
    await application.evaluate(({ BrowserWindow }) => {
      BrowserWindow.getAllWindows()[0]!.webContents.setZoomFactor(1);
      BrowserWindow.getAllWindows()[0]!.setSize(1100, 800);
    });
    await page.getByRole('button', { name: 'Chat', exact: true }).click();
    await expect(page.locator('.surface-view')).toHaveCount(0);
    await save(proposal(3));
    await page.getByRole('button', { name: 'Workspace', exact: true }).click();
    await retained();
    await review().getByRole('region', { name: 'Selected change' }).scrollIntoViewIfNeeded();
    await page.screenshot({ path: info.outputPath('retained-compact.png') });
    await review().getByRole('button', { name: 'Current flow' }).click();
    await readyView(page);
    await primary.getByRole('button', { name: 'Step list', exact: true }).click();
    await primary.getByRole('button', { name: 'Changes', exact: true }).click();
    await retained();
    const missing = proposal(1);
    missing.comparison!.changes = missing.comparison!.changes.slice(0, 1);
    await save(missing);
    await expect(
      review().getByText('The selected change is unavailable. Choose another change.'),
    ).toBeVisible();
    await expect(review().getByRole('region', { name: 'Selected change' })).toHaveCount(0);
    await review().getByRole('button', { name: '1 Quality threshold' }).click();
    await expect(review().getByRole('region', { name: 'Selected change' })).toContainText(
      '25 Phred',
    );
    await rm(join(proposalRoot, 'req_navigation_1'), { recursive: true });
    await expect(review().getByText('The selected proposal is unavailable')).toBeVisible();
    await expect(
      review().getByRole('button', { name: 'Adopt proposal', exact: true }),
    ).toBeDisabled();
    await review().getByLabel('Proposal history').selectOption('req_navigation_2');
    await expect(review().getByRole('region', { name: 'Selected change' })).toContainText(
      '25 Phred',
    );
    await review().getByRole('button', { name: 'Current flow' }).click();
    await readyView(page);
    await expect(primary.getByRole('button', { name: 'Step list', exact: true })).toHaveAttribute(
      'aria-pressed',
      'true',
    );
    await primary.getByRole('button', { name: 'Changes', exact: true }).click();
    await expect(review().getByLabel('Proposal history')).toHaveValue('req_navigation_2');
    const doc = await page.evaluate(
      async (projectId) => window.gobble.workspace.read({ projectId }),
      fixture.projectId as string,
    );
    if (!doc.ok) throw Error(doc.error.message);
    await command({
      kind: 'close',
      surfaceId: doc.value.workspace.layout.primary.activeSurfaceId!,
    });
    await open('primary');
    await readyView(page);
    await primary.getByRole('button', { name: 'Changes', exact: true }).click();
    await expect(review().getByLabel('Proposal history')).toHaveValue('req_navigation_3');
    await application.close();
    ({ application, page } = await launch(fixture.profile, fixture.environment));
    await application.evaluate(({ BrowserWindow }) => {
      BrowserWindow.getAllWindows()[0]!.setSize(1440, 950);
      BrowserWindow.getAllWindows()[0]!.webContents.setZoomFactor(0.8);
    });
    await readyView(page);
    expect(
      JSON.parse(await readFile(join(fixture.profile, 'service/catalog.json'), 'utf8')).revisions,
    ).toEqual(catalog.revisions);
  } finally {
    await application.close();
  }
});
