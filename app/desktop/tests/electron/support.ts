import {
  _electron as electron,
  expect,
  type ElectronApplication,
  type Page,
  type Locator,
} from '@playwright/test';
import { copyFile, mkdir, writeFile } from 'node:fs/promises';
import { join, resolve } from 'node:path';

export async function launch(profile: string, extraEnv: Record<string, string> = {}) {
  const env = Object.fromEntries(
    Object.entries(process.env).filter(
      (entry): entry is [string, string] =>
        entry[1] !== undefined &&
        !['ELECTRON_RUN_AS_NODE', 'ELECTRON_RENDERER_URL'].includes(entry[0]),
    ),
  );
  const application = await electron.launch({
    args: [resolve('desktop'), '--gobble-profile=' + profile],
    env: { ...env, ...extraEnv },
  });
  try {
    const page = await application.firstWindow();
    await expect(page.locator('.app-status')).toHaveText('Workspace ready');
    await focusWindow(application);
    return { application, page };
  } catch (error) {
    await application.close().catch(() => {});
    throw error;
  }
}
/** Shared observation is gated on native focus; CDP clicks alone do not activate a macOS app. */
export async function focusWindow(application: ElectronApplication): Promise<void> {
  // Shared observation requires an actual foreground window, including on macOS relaunch.
  await application.evaluate(({ app, BrowserWindow }) => {
    app.focus({ steal: true });
    const window = BrowserWindow.getAllWindows().find((w) =>
      w.webContents.getURL().startsWith('app://gobble/'),
    )!;
    window.show();
    window.focus();
  });
  await expect
    .poll(() =>
      application.evaluate(({ BrowserWindow }) =>
        BrowserWindow.getAllWindows()
          .find((w) => w.webContents.getURL().startsWith('app://gobble/'))!
          .isFocused(),
      ),
    )
    .toBe(true);
}
export async function projectFixture(base: string, name = 'Atlas study') {
  const root = join(base, name);
  await mkdir(root, { recursive: true });
  await writeFile(
    join(root, 'samples.csv'),
    'Sample,Group,Read 1,Read 2\nS01,Control,S01_R1.fastq.gz,S01_R2.fastq.gz\nS02,Control,S02_R1.fastq.gz,S02_R2.fastq.gz\nS03,Treatment,S03_R1.fastq.gz,S03_R2.fastq.gz\nS04,Treatment,S04_R1.fastq.gz,S04_R2.fastq.gz\nS05,Treatment,S05_R1.fastq.gz,S05_R2.fastq.gz\nS06,Control,S06_R1.fastq.gz,S06_R2.fastq.gz\n',
  );
  await writeFile(join(root, 'notes.txt'), 'Inspect sample S03.\nThis draft stays local.\n');
  await copyFile(resolve('desktop/tests/electron/fixtures/quality.png'), join(root, 'quality.png'));
  return root;
}
export async function chooseProject(application: ElectronApplication, page: Page, root: string) {
  // Only the native chooser response is simulated. The button, host and Go service are real.
  await application.evaluate(({ dialog }, root) => {
    dialog.showOpenDialog = async () => ({ canceled: false, filePaths: [root] });
  }, root);
  if (!(await page.getByRole('button', { name: 'Open folder', exact: true }).isVisible()))
    await page.getByRole('button', { name: 'Switch Project', exact: true }).click();
  await page.getByRole('button', { name: 'Open folder', exact: true }).click();
  await expect(
    page.getByRole('heading', { name: root.split('/').at(-1) ?? '', exact: true }),
  ).toBeVisible();
}
export async function readyView(page: Page, pane = 'Primary pane') {
  const scope = page.getByRole('region', { name: pane, exact: true });
  await expect(scope.locator('.surface-view')).toHaveAttribute('data-ready', 'true');
  return scope;
}

export async function openAgentRoster(page: Page) {
  await page.getByRole('button', { name: 'Project agents', exact: true }).click();
  await expect(page.getByRole('dialog', { name: 'Project agents', exact: true })).toBeVisible();
}

export async function openPaneActions(scope: Page | Locator, title: string) {
  await scope.getByRole('button', { name: 'More actions for ' + title, exact: true }).click();
  await expect(
    scope.getByRole('group', { name: 'Actions for ' + title, exact: true }),
  ).toBeVisible();
}
export function attachmentPreview(page: Page) {
  return page.getByRole('dialog', { name: /^Attachment: / });
}
export async function closeAttachmentPreview(page: Page) {
  await page.keyboard.press('Escape');
  await expect(attachmentPreview(page)).not.toBeVisible();
}

/** Native capture includes the full physical window at non-default page zoom. */
export async function captureWindow(application: ElectronApplication, path: string) {
  const png = await application.evaluate(async ({ BrowserWindow }) =>
    (
      await BrowserWindow.getAllWindows()
        .find((w) => w.webContents.getURL().startsWith('app://gobble/'))!
        .capturePage()
    )
      .toPNG()
      .toString('base64'),
  );
  await writeFile(path, Buffer.from(png, 'base64'));
}
