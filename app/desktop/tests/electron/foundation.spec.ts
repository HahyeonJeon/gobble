import {
  _electron as electron,
  expect,
  test,
  type ElectronApplication,
  type Page,
} from '@playwright/test';
import { mkdtemp, rm, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { resolve, join } from 'node:path';
import { execFile } from 'node:child_process';
import { promisify } from 'node:util';

let application: ElectronApplication;
let page: Page;
let profile: string;

declare global {
  interface Window {
    testProbe: {
      runtime: () => { sandboxed: boolean; contextIsolated: boolean };
      invoke: (input: unknown) => Promise<unknown>;
    };
  }
}

async function installTestProbe(): Promise<void> {
  // Test-owned session preload; never built into or enabled by product code.
  const path = join(profile, 'probe.cjs');
  await writeFile(
    path,
    `
    const { contextBridge, ipcRenderer } = require('electron');
    contextBridge.exposeInMainWorld('testProbe', {
      runtime: () => ({ sandboxed: process.sandboxed, contextIsolated: process.contextIsolated }),
      invoke: input => ipcRenderer.invoke('gobble:app:get-info:v1', input),
    });
  `,
  );
  await application.evaluate(({ session }, filePath) => {
    session.defaultSession.registerPreloadScript({ type: 'frame', filePath });
  }, path);
  await page.reload();
  await expect(page.getByRole('status')).toHaveText('Workspace ready');
}

test.beforeEach(async () => {
  profile = await mkdtemp(join(tmpdir(), 'gobble-electron-'));
  const env = Object.fromEntries(
    Object.entries(process.env).filter(
      (entry): entry is [string, string] =>
        entry[1] !== undefined &&
        !['ELECTRON_RUN_AS_NODE', 'ELECTRON_RENDERER_URL'].includes(entry[0]),
    ),
  );
  application = await electron.launch({
    args: [resolve('desktop'), `--gobble-profile=${profile}`],
    env,
  });
  page = await application.firstWindow();
  await expect(page.getByRole('status')).toHaveText('Workspace ready');
});

test.afterEach(async () => {
  await application?.close();
  if (profile) await rm(profile, { recursive: true, force: true });
});

test('built app renders in English through a sandboxed named bridge', async ({}, testInfo) => {
  expect(page.url()).toBe('app://gobble/index.html');
  await expect(page.locator('html')).toHaveAttribute('lang', 'en');
  await expect(page.getByRole('heading', { name: 'Start with a Project' })).toBeVisible();
  const globals = await page.evaluate(() => ({
    node: ['require', 'process', 'Buffer', 'ipcRenderer'].filter((name) => name in globalThis),
    methods: Object.keys(window.gobble).sort(),
  }));
  expect(globals).toEqual({
    node: [],
    methods: [
      'collaboration',
      'continuations',
      'creation',
      'evidence',
      'files',
      'getAppInfo',
      'launches',
      'pipelineReviews',
      'pipelines',
      'preparations',
      'projects',
      'runs',
      'workspace',
    ],
  });
  expect(await page.evaluate(() => Object.keys(window.gobble.pipelineReviews).sort())).toEqual([
    'adopt',
    'list',
    'outcome',
    'select',
  ]);
  expect(await page.evaluate(() => window.gobble.getAppInfo())).toMatchObject({
    schemaVersion: 1,
    ok: true,
    value: { name: 'Gobble', electronVersion: '44.2.0', stage: 'workspace' },
  });
  await installTestProbe();
  expect(await page.evaluate(() => window.testProbe.runtime())).toEqual({
    sandboxed: true,
    contextIsolated: true,
  });
  await page.screenshot({ path: testInfo.outputPath('foundation.png') });
});

test('native folder selection registers a real Project and serves its files', async () => {
  const root = await mkdtemp(join(tmpdir(), 'gobble-e2e-project-'));
  try {
    await writeFile(join(root, 'samples.csv'), 'sample,condition\nS03,control\n');
    // Simulate only the user's native folder choice; host, service and filesystem remain real.
    await application.evaluate(({ dialog }, root) => {
      dialog.showOpenDialog = async () => ({ canceled: false, filePaths: [root] });
    }, root);
    const selected = await page.evaluate(() =>
      window.gobble.projects.chooseFolder({ requestId: 'req_electron' }),
    );
    if (!selected.ok || selected.value.kind !== 'selected')
      throw new Error('Project selection failed');
    const projectId = selected.value.project.projectId;
    const files = await page.evaluate(
      (projectId) => window.gobble.files.list({ projectId }),
      projectId,
    );
    if (!files.ok) throw new Error(files.error.message);
    const entry = files.value.entries.find((entry) => entry.name === 'samples.csv');
    if (!entry) throw new Error('CSV missing');
    const result = await page.evaluate((input) => window.gobble.files.read(input), {
      projectId,
      resourceId: entry.resourceId,
    });
    expect(result).toMatchObject({
      ok: true,
      value: { content: { kind: 'table', rows: [{ cells: ['S03', 'control'] }] } },
    });
    expect(JSON.stringify(selected)).not.toContain(root);
    expect(await page.evaluate(() => window.gobble.projects.list())).toMatchObject({
      ok: true,
      value: [{ projectId }],
    });
  } finally {
    await rm(root, { recursive: true, force: true });
  }
});

test('real IPC rejects bad payloads and an unregistered window', async () => {
  await installTestProbe();
  expect(
    await page.evaluate(() => window.testProbe.invoke({ schemaVersion: 2, path: '/private' })),
  ).toMatchObject({ ok: false, error: { code: 'invalid_request' } });

  const extraPage = application.waitForEvent('window');
  await application.evaluate(async ({ BrowserWindow }, preloadPath) => {
    const extra = new BrowserWindow({
      show: false,
      webPreferences: {
        preload: preloadPath,
        sandbox: true,
        contextIsolation: true,
        nodeIntegration: false,
      },
    });
    await extra.loadURL('app://gobble/index.html');
  }, resolve('desktop/out/preload/index.cjs'));
  const unregistered = await extraPage;
  expect(await unregistered.evaluate(() => window.gobble.getAppInfo())).toMatchObject({
    ok: false,
    error: { code: 'forbidden' },
  });
});

test('blocks renderer code injection, frames, external navigation and new windows', async () => {
  const violations = await page.evaluate(async () => {
    const blocked: string[] = [];
    document.addEventListener('securitypolicyviolation', (event) =>
      blocked.push(event.violatedDirective),
    );
    const script = document.createElement('script');
    script.textContent = 'document.documentElement.dataset.injected = "yes"';
    document.body.append(script);
    const frame = document.createElement('iframe');
    frame.src = 'app://gobble/index.html';
    document.body.append(frame);
    window.open('https://example.invalid');
    window.location.assign('https://example.invalid');
    await new Promise((resolve) => setTimeout(resolve, 100));
    return { blocked, injected: document.documentElement.dataset.injected ?? null };
  });
  expect(violations.injected).toBeNull();
  expect(violations.blocked).toEqual(expect.arrayContaining(['script-src-elem', 'frame-src']));
  expect(page.url()).toBe('app://gobble/index.html');
  expect(application.windows()).toHaveLength(1);
  expect(await page.evaluate(() => Notification.requestPermission())).toBe('denied');
});

test('macOS window close preserves the host and activation recreates an isolated window', async () => {
  test.skip(process.platform !== 'darwin', 'macOS lifecycle contract');
  const pid = application.process().pid;
  await application.evaluate(({ BrowserWindow }) => BrowserWindow.getAllWindows()[0]?.close());
  await expect
    .poll(() => application.evaluate(({ BrowserWindow }) => BrowserWindow.getAllWindows().length))
    .toBe(0);
  const nextWindow = application.waitForEvent('window');
  await application.evaluate(({ app }) => app.emit('activate'));
  const restored = await nextWindow;
  await expect(restored.getByRole('status')).toHaveText('Workspace ready');
  expect(application.process().pid).toBe(pid);
});

test('a second launch reuses the current profile owner', async () => {
  const child = application.process();
  const env = Object.fromEntries(
    Object.entries(process.env).filter(
      (entry): entry is [string, string] =>
        entry[1] !== undefined &&
        !['ELECTRON_RUN_AS_NODE', 'ELECTRON_RENDERER_URL'].includes(entry[0]),
    ),
  );
  await promisify(execFile)(child.spawnfile, [resolve('desktop'), `--gobble-profile=${profile}`], {
    env,
    timeout: 10_000,
  });
  expect(application.process().pid).toBe(child.pid);
  expect(application.windows()).toHaveLength(1);
  await expect(page.getByRole('status')).toHaveText('Workspace ready');
});
