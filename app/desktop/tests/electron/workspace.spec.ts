import { openPaneActions, chooseProject, launch, projectFixture, readyView } from './support';
import { expect, test, type ElectronApplication, type Page } from '@playwright/test';
import { mkdtemp, readFile, readdir, rm, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';

let application: ElectronApplication;
let page: Page;
let directory: string;
let profile: string;
let project: string;
test.beforeEach(async () => {
  directory = await mkdtemp(join(tmpdir(), 'gobble-workspace-e2e-'));
  profile = join(directory, 'profile');
  project = await projectFixture(directory);
  ({ application, page } = await launch(profile));
  await chooseProject(application, page, project);
});
test.afterEach(async () => {
  await application?.evaluate(({ dialog }) => {
    dialog.showMessageBox = async () => ({ response: 1, checkboxChecked: false });
  });
  await application?.close();
  await rm(directory, { recursive: true, force: true });
});

test('real files, split, selection and draft survive a normal app relaunch', async ({}, testInfo) => {
  await page.getByRole('button', { name: 'samples.csv', exact: true }).click();
  const primary = await readyView(page);
  await primary.getByRole('checkbox', { name: 'Select row 3: S03' }).check();
  await openPaneActions(primary, 'samples.csv');
  await primary.getByRole('button', { name: 'Pin samples.csv', exact: true }).click();
  await page.getByRole('button', { name: 'Open quality.png in the other pane' }).click();
  await readyView(page, 'Secondary pane');
  await page.getByRole('separator', { name: 'Resize panes' }).focus();
  await page.keyboard.press('ArrowDown');
  await expect(page.getByRole('separator', { name: 'Resize panes' })).toHaveAttribute(
    'aria-valuenow',
    '55',
  );
  await page
    .getByRole('textbox', { name: 'Message draft' })
    .fill('Review sample S03 alongside this image.');
  await expect(page.getByText('Draft saved locally', { exact: true })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Send', exact: true })).toBeDisabled();
  await page.screenshot({ path: testInfo.outputPath('03-workspace.png') });
  await application.close();
  ({ application, page } = await launch(profile));
  const restored = await readyView(page);
  await readyView(page, 'Secondary pane');
  await expect(restored.getByRole('checkbox', { name: 'Select row 3: S03' })).toBeChecked();
  await openPaneActions(restored, 'samples.csv');
  await expect(
    restored.getByRole('button', { name: 'Unpin samples.csv', exact: true }),
  ).toHaveAttribute('aria-pressed', 'true');
  await page.keyboard.press('Escape');
  await expect(page.getByRole('textbox', { name: 'Message draft' })).toHaveValue(
    'Review sample S03 alongside this image.',
  );
  await expect(page.getByRole('separator', { name: 'Resize panes' })).toHaveAttribute(
    'aria-valuenow',
    '55',
  );
});

test('keyboard tabs, duplicate, move and close preserve the remaining views', async () => {
  await page.getByRole('button', { name: 'samples.csv', exact: true }).click();
  await readyView(page);
  await page.getByRole('button', { name: 'notes.txt', exact: true }).click();
  let primary = await readyView(page);
  await primary.getByRole('tab', { name: 'notes.txt', exact: true }).focus();
  await page.keyboard.press('ArrowLeft');
  await expect(primary.getByRole('tab', { name: 'samples.csv', exact: true })).toHaveAttribute(
    'aria-selected',
    'true',
  );
  await readyView(page);
  await openPaneActions(primary, 'samples.csv');
  await primary.getByRole('button', { name: 'Duplicate samples.csv in other pane' }).click();
  const secondary = await readyView(page, 'Secondary pane');
  await expect(page.getByRole('tab', { name: 'samples.csv', exact: true })).toHaveCount(2);
  await secondary.getByRole('button', { name: 'Close samples.csv', exact: true }).click();
  primary = await readyView(page);
  await primary.getByRole('button', { name: 'Close samples.csv', exact: true }).click();
  await expect(primary.getByRole('tab', { name: 'notes.txt', exact: true })).toBeFocused();
  await readyView(page);
  await openPaneActions(primary, 'notes.txt');
  await primary.getByRole('button', { name: 'Move notes.txt to other pane' }).click();
  await readyView(page, 'Secondary pane');
  await expect(primary.getByRole('tab')).toHaveCount(0);
  await page.getByRole('button', { name: 'Single pane', exact: true }).click();
  await readyView(page);
  await expect(page.getByRole('tab', { name: 'notes.txt', exact: true })).toHaveCount(1);
});

test('narrow windows retain both panes and keyboard image and text context', async () => {
  await page.getByRole('button', { name: 'notes.txt', exact: true }).click();
  const primary = await readyView(page);
  const text = primary.getByRole('textbox', { name: 'notes.txt text' });
  await text.focus();
  await page.keyboard.press(process.platform === 'darwin' ? 'Meta+A' : 'Control+A');
  await primary.getByRole('button', { name: 'Use text selection' }).click();
  await expect(
    primary.getByRole('region', { name: 'Selection in notes.txt', exact: true }),
  ).toBeVisible();
  await page.getByRole('button', { name: 'Open quality.png in the other pane' }).click();
  const secondary = await readyView(page, 'Secondary pane');
  await secondary.getByRole('button', { name: 'Select region' }).click();
  await secondary.getByLabel('Left', { exact: true }).fill('20');
  await secondary.getByLabel('Top', { exact: true }).fill('10');
  await secondary.getByLabel('Width', { exact: true }).fill('30');
  await secondary.getByLabel('Height', { exact: true }).fill('40');
  await secondary.getByRole('button', { name: 'Use region' }).click();
  await expect(page.getByText('quality.png · Image region', { exact: true })).toBeVisible();
  await application.evaluate(({ BrowserWindow }) =>
    BrowserWindow.getAllWindows()[0]?.setBounds({ width: 850, height: 610 }),
  );
  await expect(page.getByRole('separator', { name: 'Resize panes' })).toHaveCount(0);
  await page.getByRole('button', { name: 'Primary pane', exact: true }).click();
  await readyView(page);
  await expect(page.getByRole('tab', { name: 'notes.txt', exact: true })).toBeVisible();
  await page.getByRole('button', { name: 'Secondary pane', exact: true }).click();
  await readyView(page, 'Secondary pane');
  await expect(page.getByRole('img', { name: 'quality.png', exact: true })).toBeVisible();
  await application.evaluate(({ BrowserWindow }) =>
    BrowserWindow.getAllWindows()[0]?.setBounds({ width: 1280, height: 840 }),
  );
  await readyView(page);
  await readyView(page, 'Secondary pane');
  await expect(page.getByRole('separator', { name: 'Resize panes' })).toBeVisible();
});

test('changed and missing files remain explicit while other Projects stay separate', async () => {
  await page.getByRole('button', { name: 'samples.csv', exact: true }).click();
  let primary = await readyView(page);
  await primary.getByRole('checkbox', { name: 'Select row 3: S03' }).check();
  await writeFile(join(project, 'samples.csv'), 'Sample,Group\nS09,Treatment\n');
  await openPaneActions(primary, 'samples.csv');
  await primary.getByRole('button', { name: 'Refresh samples.csv', exact: true }).click();
  primary = await readyView(page);
  await expect(primary.getByText('Selection changed.', { exact: false })).toBeVisible();
  await expect(primary.getByRole('checkbox', { name: 'Select row 1: S09' })).not.toBeChecked();
  await primary.getByRole('button', { name: 'Clear old selection' }).click();
  await rm(join(project, 'samples.csv'));
  await openPaneActions(primary, 'samples.csv');
  await primary.getByRole('button', { name: 'Refresh samples.csv', exact: true }).click();
  await expect(primary.getByRole('heading', { name: 'This view is unavailable' })).toBeVisible();
  await writeFile(join(project, 'samples.csv'), 'Sample,Group\nS09,Treatment\n');
  await primary.getByRole('button', { name: 'Retry view' }).click();
  await readyView(page);
  await page.getByRole('textbox', { name: 'Message draft' }).fill('Atlas-only draft');
  await expect(page.getByText('Draft saved locally', { exact: true })).toBeVisible();
  const other = await projectFixture(directory, 'WGS pilot');
  await chooseProject(application, page, other);
  await expect(page.getByRole('tablist', { name: 'Primary views' }).getByRole('tab')).toHaveCount(
    0,
  );
  await expect(page.getByRole('textbox', { name: 'Message draft' })).toHaveValue('');
  await page.getByRole('button', { name: 'Switch Project', exact: true }).click();
  await page
    .getByRole('navigation', { name: 'Projects', exact: true })
    .getByRole('button', { name: 'Atlas study', exact: true })
    .click();
  await readyView(page);
  await expect(page.getByRole('textbox', { name: 'Message draft' })).toHaveValue(
    'Atlas-only draft',
  );
  expect(await readFile(join(project, 'samples.csv'), 'utf8')).toBe(
    'Sample,Group\nS09,Treatment\n',
  );
});

test('a failed draft save preserves the Project and honors Keep Open on window close', async () => {
  const projects = join(profile, 'workspace', 'projects');
  const saved = (await readdir(projects)).find((name) => name.endsWith('.json'));
  expect(saved).toBeTruthy();
  const path = join(projects, saved!);
  await writeFile(path, (await readFile(path, 'utf8')) + ' ');
  await page.getByRole('textbox', { name: 'Message draft' }).fill('Preserve this unsaved draft');
  await expect(page.getByText('Draft not saved', { exact: true })).toBeVisible();
  await page.getByRole('button', { name: 'Switch Project', exact: true }).click();
  await page.getByRole('button', { name: 'All projects', exact: true }).click();
  await expect(
    page.getByText(
      'Your draft is not saved. Keep this Project open and copy the draft before closing the app.',
      { exact: true },
    ),
  ).toBeVisible();
  await expect(page.getByRole('textbox', { name: 'Message draft' })).toHaveValue(
    'Preserve this unsaved draft',
  );
  const action = await application.evaluate(
    ({ dialog, BrowserWindow }) =>
      new Promise<string>((resolve) => {
        dialog.showMessageBox = async (options) => {
          resolve('buttons' in options ? (options.buttons?.[1] ?? '') : '');
          return { response: 0, checkboxChecked: false };
        };
        BrowserWindow.getAllWindows()[0]?.close();
      }),
  );
  expect(action).toBe('Close Without Saving');
  await expect
    .poll(() =>
      application.evaluate(
        ({ BrowserWindow }) => BrowserWindow.getAllWindows()[0]?.isDestroyed() ?? true,
      ),
    )
    .toBe(false);
  await expect(page.getByRole('textbox', { name: 'Message draft' })).toHaveValue(
    'Preserve this unsaved draft',
  );
  await application.evaluate(({ dialog }) => {
    dialog.showMessageBox = async () => ({ response: 1, checkboxChecked: false });
  });
});

test('stacked panes, fitted image and one composer retain their identity across resizing and compact chat', async ({}, testInfo) => {
  await page.getByRole('button', { name: 'samples.csv', exact: true }).click();
  await readyView(page);
  await page.getByRole('button', { name: 'Open quality.png in the other pane' }).click();
  const secondary = await readyView(page, 'Secondary pane');
  const geometry = await page.evaluate(() => {
    const upper = document.getElementById('primary-pane')!.getBoundingClientRect();
    const lower = document.getElementById('secondary-pane')!.getBoundingClientRect();
    const chat = document.getElementById('project-chat')!.getBoundingClientRect();
    return {
      upper: { x: upper.x, bottom: upper.bottom, right: upper.right },
      lower: { x: lower.x, y: lower.y, right: lower.right },
      chat: { x: chat.x, width: chat.width },
    };
  });
  expect(geometry.lower.x).toBeCloseTo(geometry.upper.x, 0);
  expect(geometry.lower.y).toBeGreaterThan(geometry.upper.bottom);
  expect(geometry.chat.x).toBeGreaterThan(geometry.lower.right);
  expect(geometry.chat.width).toBeCloseTo(420, 0);
  const image = secondary.getByRole('img', { name: 'quality.png' });
  await expect
    .poll(() =>
      image.evaluate((element) => {
        const bounds = element.getBoundingClientRect();
        const viewport = element.closest('.image-scroll')!.getBoundingClientRect();
        return (
          bounds.top >= viewport.top &&
          bounds.bottom <= viewport.bottom &&
          bounds.left >= viewport.left &&
          bounds.right <= viewport.right
        );
      }),
    )
    .toBe(true);
  const bounds = (await image.boundingBox())!;
  await page.mouse.move(bounds.x + bounds.width * 0.25, bounds.y + bounds.height * 0.2);
  await page.mouse.down();
  await page.mouse.move(bounds.x + bounds.width * 0.75, bounds.y + bounds.height * 0.6, {
    steps: 8,
  });
  await page.mouse.up();
  await expect(page.getByText('quality.png · Image region', { exact: true })).toBeVisible();
  const draft = page.getByRole('textbox', { name: 'Message draft' });
  await draft.fill('Keep this exact draft while changing views.');
  await expect(page.getByText('Draft saved locally', { exact: true })).toBeVisible();
  const originalInput = await draft.elementHandle();
  const paneDivider = page.getByRole('separator', { name: 'Resize panes' });
  const dividerBounds = (await paneDivider.boundingBox())!;
  await page.mouse.move(
    dividerBounds.x + dividerBounds.width / 2,
    dividerBounds.y + dividerBounds.height / 2,
  );
  await page.mouse.down();
  await page.mouse.move(dividerBounds.x + dividerBounds.width / 2, dividerBounds.y + 55, {
    steps: 6,
  });
  await page.mouse.up();
  await expect
    .poll(async () => Number(await paneDivider.getAttribute('aria-valuenow')))
    .toBeGreaterThan(54);
  const chatDivider = page.getByRole('separator', { name: 'Resize chat' });
  const chatBounds = (await chatDivider.boundingBox())!;
  await page.mouse.move(chatBounds.x + chatBounds.width / 2, chatBounds.y + 120);
  await page.mouse.down();
  await page.mouse.move(chatBounds.x - 30, chatBounds.y + 120, { steps: 6 });
  await page.mouse.up();
  await expect
    .poll(async () => Number(await chatDivider.getAttribute('aria-valuenow')))
    .toBeGreaterThan(445);
  // Restore a known width, then exercise its opposite-edge keyboard direction.
  await chatDivider.focus();
  await page.keyboard.press('Home');
  await page.keyboard.press('ArrowLeft');
  await page.keyboard.press('ArrowLeft');
  await page.keyboard.press('ArrowLeft');

  await page.getByRole('separator', { name: 'Resize chat' }).focus();
  await page.keyboard.press('ArrowLeft');
  await expect(page.getByRole('separator', { name: 'Resize chat' })).toHaveAttribute(
    'aria-valuenow',
    '440',
  );
  await page.getByRole('button', { name: 'Collapse chat' }).click();
  await expect(draft).not.toBeVisible();
  await expect(page.getByRole('button', { name: 'Show chat' })).toBeFocused();
  await page.getByRole('button', { name: 'Show chat' }).click();
  await expect(draft).toHaveValue('Keep this exact draft while changing views.');
  await application.evaluate(({ BrowserWindow }) =>
    BrowserWindow.getAllWindows()[0]!.setBounds({ width: 850, height: 840 }),
  );
  await page
    .getByRole('navigation', { name: 'Project regions' })
    .getByRole('button', { name: 'Chat', exact: true })
    .click();
  await expect(draft).toBeVisible();
  await expect(page.locator('.surface-view')).toHaveCount(0);
  expect(
    await page.evaluate(
      (input) => input === document.querySelector('[aria-label="Message draft"]'),
      originalInput,
    ),
  ).toBe(true);
  await expect(page.locator('textarea[aria-label="Message draft"]')).toHaveCount(1);
  await page.getByRole('button', { name: 'Workspace', exact: true }).click();
  await readyView(page);
  await readyView(page, 'Secondary pane');
  await application.evaluate(({ BrowserWindow }) =>
    BrowserWindow.getAllWindows()[0]!.setBounds({ width: 1280, height: 840 }),
  );
  await expect(draft).toBeVisible();
  await expect(page.getByRole('separator', { name: 'Resize chat' })).toHaveAttribute(
    'aria-valuenow',
    '440',
  );
  await secondary.getByRole('button', { name: 'Select region', exact: true }).click();
  await expect(secondary.getByLabel('Left', { exact: true })).toHaveValue('25');
  await expect(secondary.getByLabel('Top', { exact: true })).toHaveValue('20');
  await expect(secondary.getByLabel('Width', { exact: true })).toHaveValue('50');
  await expect(secondary.getByLabel('Height', { exact: true })).toHaveValue('40');
  await secondary.getByRole('button', { name: 'Hide region controls' }).click();
  await page.screenshot({ path: testInfo.outputPath('05-1-chat-workspace.png') });
});

test('the real app migrates saved v1 geometry on the first change and retains the original archive', async () => {
  await page.getByRole('button', { name: 'samples.csv', exact: true }).click();
  const primary = await readyView(page);
  await primary.getByRole('checkbox', { name: 'Select row 3: S03' }).check();
  await openPaneActions(primary, 'samples.csv');
  await primary.getByRole('button', { name: 'Pin samples.csv', exact: true }).click();
  await page.getByRole('button', { name: 'Open quality.png in the other pane' }).click();
  await readyView(page, 'Secondary pane');
  await page.getByRole('textbox', { name: 'Message draft' }).fill('Draft from the earlier app');
  await expect(page.getByText('Draft saved locally', { exact: true })).toBeVisible();
  await application.close();
  const projects = join(profile, 'workspace', 'projects');
  const name = (await readdir(projects)).find((name) => name.endsWith('.json'))!;
  const path = join(projects, name);
  const doc = JSON.parse(await readFile(path, 'utf8'));
  doc.schemaVersion = 1;
  delete doc.viewLinks;
  doc.selections = doc.selections.map(
    ({ surfaceId, evidence }: import('@gobble/contracts').LocalSelection) => {
      if (
        evidence.schemaVersion === 4 ||
        evidence.schemaVersion === 5 ||
        evidence.schemaVersion === 6
      )
        throw new Error('Legacy fixture required');
      const { schemaVersion: _version, origin: _origin, selection, ...target } = evidence;
      const { coordinateSpace: _space, ...legacySelection } = selection!;
      return { ...target, surfaceId, selection: legacySelection };
    },
  );
  doc.discussion = {
    collapsed: true,
    height: 380,
    draft: doc.chat.draft,
    recipientAgentId: doc.chat.recipientAgentId,
  };
  delete doc.chat;
  delete doc.paneOrientation;
  doc.workspace.layout.primaryFraction = 0.73;
  const legacy = JSON.stringify(doc, null, 2) + '\n';
  await writeFile(path, legacy); // isolated test profile, not the user's workspace
  ({ application, page } = await launch(profile));
  await readyView(page);
  await expect(page.getByRole('button', { name: 'Show chat' })).toBeVisible();
  expect(await readFile(path, 'utf8')).toBe(legacy);
  await page.getByRole('button', { name: 'Show chat' }).click();
  await expect(page.getByRole('textbox', { name: 'Message draft' })).toHaveValue(
    'Draft from the earlier app',
  );
  await expect(page.getByRole('separator', { name: 'Resize panes' })).toHaveAttribute(
    'aria-valuenow',
    '50',
  );
  await expect(
    (await readyView(page)).getByRole('checkbox', { name: 'Select row 3: S03' }),
  ).toBeChecked();
  await openPaneActions(page, 'samples.csv');
  await expect(
    page.getByRole('button', { name: 'Unpin samples.csv', exact: true }),
  ).toHaveAttribute('aria-pressed', 'true');
  await page.keyboard.press('Escape');
  expect(await readFile(path + '.v1.backup', 'utf8')).toBe(legacy);
  await page.getByRole('textbox', { name: 'Message draft' }).fill('Another saved draft');
  await expect(page.getByText('Draft saved locally', { exact: true })).toBeVisible();
  expect(await readFile(path + '.v1.backup', 'utf8')).toBe(legacy);
});

test('selection overview finds hidden sources, More returns keyboard focus, and empty panes target navigation', async ({}, info) => {
  await page.getByRole('button', { name: 'samples.csv', exact: true }).click();
  const primary = await readyView(page);
  await primary.getByRole('checkbox', { name: 'Select row 3: S03' }).check();
  await expect(primary.getByRole('region', { name: 'Selection in samples.csv' })).toBeVisible();
  await page.getByRole('button', { name: 'notes.txt', exact: true }).click();
  await readyView(page);
  await expect(page.getByRole('button', { name: 'Add to message from samples.csv' })).toHaveCount(
    0,
  );
  await page
    .getByRole('textbox', { name: 'Message draft' })
    .fill('Keep this draft while finding my selection.');
  await application.evaluate(({ BrowserWindow }) =>
    BrowserWindow.getAllWindows()[0]!.setBounds({ width: 900, height: 650 }),
  );
  await page
    .getByRole('navigation', { name: 'Project regions' })
    .getByRole('button', { name: 'Chat', exact: true })
    .click();
  await page.getByRole('button', { name: 'Project selections' }).click();
  await page.getByRole('button', { name: 'Return to selection in samples.csv' }).click();
  await readyView(page);
  await expect(primary.getByRole('checkbox', { name: 'Select row 3: S03' })).toBeChecked();
  await expect(page.getByRole('tab', { name: 'samples.csv' })).toHaveAttribute(
    'aria-selected',
    'true',
  );
  await expect(page.getByRole('tab', { name: 'samples.csv' })).toBeFocused();
  const more = primary.getByRole('button', { name: 'More actions for samples.csv' });
  await more.focus();
  await page.keyboard.press('Enter');
  await expect(primary.getByRole('button', { name: 'Pin samples.csv', exact: true })).toBeFocused();
  await page.keyboard.press('Escape');
  await expect(more).toBeFocused();
  await page.screenshot({ path: info.outputPath('05-6b-compact-selection.png') });
  await application.evaluate(({ BrowserWindow }) =>
    BrowserWindow.getAllWindows()[0]!.setBounds({ width: 1280, height: 840 }),
  );
  await page.getByRole('button', { name: 'Two panes', exact: true }).click();
  const secondary = page.getByRole('region', { name: 'Secondary pane', exact: true });
  // Do not let a rendered activePane update hide the fast Browse → file ordering race.
  await secondary.getByRole('button', { name: 'Browse files' }).evaluate((browse) => {
    if (!(browse instanceof HTMLButtonElement)) throw new Error('Expected the Browse button');
    browse.click();
    document
      .querySelector<HTMLButtonElement>('#project-explorer button[title="quality.png"]')!
      .click();
  });
  await readyView(page, 'Secondary pane');
  await expect(secondary.getByRole('tab', { name: 'quality.png' })).toBeVisible();
  await secondary.getByRole('button', { name: 'Close quality.png' }).click();
  await secondary.getByText('Move a view here', { exact: true }).click();
  await secondary.getByRole('button', { name: 'notes.txt', exact: true }).click();
  await readyView(page, 'Secondary pane');
  await expect(page.getByRole('tab', { name: 'notes.txt', exact: true })).toHaveCount(1);
  await expect(page.getByRole('textbox', { name: 'Message draft' })).toHaveValue(
    'Keep this draft while finding my selection.',
  );
  await expect(page.getByLabel('Message attachments', { exact: true })).toHaveCount(0);
  await page.screenshot({ path: info.outputPath('05-6b-source-actions.png') });
});
