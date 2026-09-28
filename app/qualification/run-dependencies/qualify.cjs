const path = require('node:path');
const fs = require('node:fs/promises');
const { tmpdir } = require('node:os');
const { execFileSync } = require('node:child_process');
const { strict: assert } = require('node:assert');
const { createHash } = require('node:crypto');
const { build } = require('esbuild');
const { _electron: electron } = require('@playwright/test');
const appRoot = path.resolve(__dirname, '../..');
const evidence = path.resolve(appRoot, '../docs/desktop-workspace/stages/r3b1-review');
const report = {
  date: new Date().toISOString(),
  status: 'running',
  candidates: {},
  scenarios: [],
  errors: [],
  limits:
    'Synthetic source-checkout Electron qualification on one macOS host. No Agent, Project integration, assistive-user study, installed lifecycle or release claim.',
};
(async () => {
  const scratch = await fs.mkdtemp(path.join(tmpdir(), 'gobble-r3b-qualification-'));
  let application;
  try {
    await fs.mkdir(evidence, { recursive: true });
    const candidatePackages = { '@xyflow/react': '12.11.6', '@dagrejs/dagre': '3.1.1' };
    await fs.writeFile(
      path.join(scratch, 'package.json'),
      JSON.stringify({
        private: true,
        dependencies: { ...candidatePackages, react: '19.2.8', 'react-dom': '19.2.8' },
      }),
    );
    execFileSync('npm', ['install', '--no-audit', '--no-fund', '--ignore-scripts'], {
      cwd: scratch,
      stdio: 'pipe',
    });
    await fs.copyFile(
      path.join(scratch, 'package-lock.json'),
      path.join(evidence, 'renderer-package-lock.json'),
    );
    const packages = path.join(scratch, 'node_modules');
    const resolveFrom = (name) => require.resolve(name, { paths: [scratch] });
    const assets = path.join(scratch, 'assets');
    await fs.mkdir(path.join(assets, 'assets'), { recursive: true });
    const rendererBuild = await build({
      entryPoints: [path.join(__dirname, 'renderer.tsx')],
      outfile: path.join(assets, 'assets', 'renderer.js'),
      bundle: true,
      format: 'esm',
      platform: 'browser',
      target: 'chrome152',
      jsx: 'automatic',
      minify: true,
      metafile: true,
      alias: {
        '@xyflow/react': path.join(packages, '@xyflow/react'),
        '@dagrejs/dagre': resolveFrom('@dagrejs/dagre'),
        react: path.join(appRoot, 'node_modules/react'),
        'react-dom': path.join(appRoot, 'node_modules/react-dom'),
      },
    });
    await fs.writeFile(
      path.join(assets, 'index.html'),
      '<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>R3b1 dependency qualification</title><link rel="stylesheet" href="/assets/renderer.css"></head><body><div id="root"></div><script type="module" src="/assets/renderer.js"></script></body></html>',
    );
    const main = path.join(scratch, 'main.cjs');
    await build({
      entryPoints: [path.join(__dirname, 'main.ts')],
      outfile: main,
      bundle: true,
      format: 'cjs',
      platform: 'node',
      external: ['electron'],
    });
    const config = {
      extends: path.join(appRoot, 'tsconfig.base.json'),
      compilerOptions: {
        lib: ['ES2022', 'DOM'],
        types: ['node'],
        typeRoots: [path.join(appRoot, 'node_modules/@types')],
        jsx: 'react-jsx',
        paths: {
          '@xyflow/react': [path.join(packages, '@xyflow/react')],
          '@dagrejs/dagre': [path.join(packages, '@dagrejs/dagre')],
        },
      },
      include: [path.join(__dirname, '*.ts'), path.join(__dirname, '*.tsx')],
    };
    await fs.writeFile(path.join(scratch, 'tsconfig.json'), JSON.stringify(config));
    try {
      execFileSync(
        path.join(appRoot, 'node_modules/.bin/tsc'),
        ['-p', path.join(scratch, 'tsconfig.json')],
        { cwd: appRoot, stdio: 'pipe' },
      );
      report.typecheck = 'passed';
    } catch (error) {
      await fs.writeFile(path.join(evidence, 'renderer-types.log'), error.stdout ?? String(error));
      throw error;
    }
    const env = Object.fromEntries(
      Object.entries(process.env).filter(
        ([key, value]) =>
          value !== undefined && !['ELECTRON_RUN_AS_NODE', 'ELECTRON_RENDERER_URL'].includes(key),
      ),
    );
    application = await electron.launch({
      args: [main],
      env: {
        ...env,
        R3B_QUALIFICATION_PROFILE: path.join(scratch, 'profile'),
        R3B_QUALIFICATION_ASSETS: assets,
      },
    });
    const page = await application.firstWindow();
    page.on('pageerror', (error) => report.errors.push(error.message));
    page.on('console', (message) => {
      if (message.type() === 'error') report.errors.push(message.text());
    });
    await page.waitForURL('app://gobble/index.html', { waitUntil: 'load' });
    await page.reload();
    await application.evaluate(({ app, BrowserWindow }) => {
      app.focus({ steal: true });
      BrowserWindow.getAllWindows()[0].focus();
    });
    await page.waitForSelector('body[data-ready="simple-8-true"]');
    const cdp = await page.context().newCDPSession(page);
    await cdp.send('Performance.enable');
    async function heap() {
      await cdp.send('HeapProfiler.collectGarbage');
      return (await cdp.send('Performance.getMetrics')).metrics.find(
        (m) => m.name === 'JSHeapUsedSize',
      ).value;
    }
    report.environment = await application.evaluate(({ app, BrowserWindow }) => ({
      platform: process.platform,
      arch: process.arch,
      versions: process.versions,
      preferences: BrowserWindow.getAllWindows()[0].webContents.getLastWebPreferences(),
      appVersion: app.getVersion(),
    }));
    for (const candidate of ['simple', 'flow']) {
      const measurements = [];
      await page.getByLabel('Renderer', { exact: true }).selectOption(candidate);
      for (const size of [8, 80]) {
        const start = performance.now();
        await page.getByLabel('Groups', { exact: true }).selectOption(String(size));
        await page.waitForSelector(`body[data-ready="${candidate}-${size}-true"]`);
        const node =
          candidate === 'simple' ? page.locator('.task-node') : page.locator('.react-flow__node');
        await node.first().waitFor({ state: 'visible' });
        assert.equal(await node.count(), size);
        measurements.push({ groups: size, mountMs: Math.round(performance.now() - start) });
        await node.first().focus();
        await page.keyboard.press('Enter');
        await page.waitForFunction(() =>
          document.querySelector('#selection').textContent.includes('run-group'),
        );
        const selection = await page.locator('#selection').innerText();
        assert.deepEqual(JSON.parse(selection), {
          kind: 'run-group',
          coordinateSpace: 'observed-authored-task-group',
          taskId: 'task-0',
        });
        await page.keyboard.press('Delete');
        assert.equal(await node.count(), size);
        assert.equal(await page.locator('#selection').innerText(), selection);
        await page.locator('aside button').first().focus();
        await page.keyboard.press('Enter');
        assert.deepEqual(JSON.parse(await page.locator('#selection').innerText()), {
          kind: 'run-dependency',
          coordinateSpace: 'observed-authored-task-pair',
          fromTaskId: 'task-0',
          toTaskId: size === 8 ? 'task-2' : 'task-14',
        });
        await page.evaluate(
          () =>
            new Promise((resolve) => requestAnimationFrame(() => requestAnimationFrame(resolve))),
        );
        const png = await application.evaluate(async ({ BrowserWindow }) =>
          (await BrowserWindow.getAllWindows()[0].capturePage()).toPNG().toString('base64'),
        );
        await fs.writeFile(
          path.join(evidence, `renderer-${candidate}-${size}.png`),
          Buffer.from(png, 'base64'),
        );
        report.scenarios.push(
          `${candidate}/${size}: exact group and directed-pair keyboard selection; Delete leaves read-only topology intact.`,
        );
      }
      await page.getByLabel('Groups', { exact: true }).selectOption('81');
      await page.waitForSelector('.graph-host [role="status"]');
      assert.match(await page.locator('.graph-host').innerText(), /limit reached/);
      await page.getByLabel('Groups', { exact: true }).selectOption('8');
      await application.evaluate(({ BrowserWindow }) => {
        const window = BrowserWindow.getAllWindows()[0];
        window.setSize(900, 650);
        window.webContents.setZoomFactor(1.5);
      });
      await page.locator('aside button').first().click();
      const footer = await page.locator('footer').boundingBox(),
        height = await page.evaluate(() => innerHeight);
      assert(footer.y + footer.height <= height + 1);
      await page.evaluate(
        () => new Promise((resolve) => requestAnimationFrame(() => requestAnimationFrame(resolve))),
      );
      const png = await application.evaluate(async ({ BrowserWindow }) =>
        (await BrowserWindow.getAllWindows()[0].capturePage()).toPNG().toString('base64'),
      );
      await fs.writeFile(
        path.join(evidence, `renderer-${candidate}-150.png`),
        Buffer.from(png, 'base64'),
      );
      await application.evaluate(({ BrowserWindow }) => {
        const window = BrowserWindow.getAllWindows()[0];
        window.webContents.setZoomFactor(1);
        window.setSize(1280, 840);
      });
      const before = await heap();
      for (let i = 0; i < 15; i++) {
        await page.getByRole('button', { name: 'Unmount', exact: true }).click();
        await page.getByRole('button', { name: 'Mount', exact: true }).click();
      }
      const after = await heap();
      assert(
        after - before < 15 * 1024 * 1024,
        'Retained heap growth exceeded the qualification bound',
      );
      report.candidates[candidate] = {
        measurements,
        heapBefore: before,
        heapAfter: after,
        heapDelta: after - before,
        cycles: 15,
      };
      report.scenarios.push(
        `${candidate}: 81-group fallback; 900×650 at native 150% keeps selected target reachable; 15 unmount/remount cycles remain within 15 MiB retained-heap growth.`,
      );
    }
    report.cspViolations = await page.evaluate(() => window.qualificationViolations);
    assert.deepEqual(report.cspViolations, []);
    assert.deepEqual(report.errors, []);
    report.status = 'passed';
    report.packages = candidatePackages;
    report.bundleBytes = Object.fromEntries(
      Object.entries(rendererBuild.metafile.outputs).map(([file, info]) => [
        path.basename(file),
        info.bytes,
      ]),
    );
    report.subjectHashes = Object.fromEntries(
      await Promise.all(
        ['main.ts', 'renderer.tsx', 'renderers.tsx', 'fixture.ts', 'style.css', 'qualify.cjs'].map(
          async (name) => [
            name,
            createHash('sha256')
              .update(await fs.readFile(path.join(__dirname, name)))
              .digest('hex'),
          ],
        ),
      ),
    );
  } catch (error) {
    report.status = 'failed';
    report.failure = String(error.stack);
    throw error;
  } finally {
    if (application) await application.close();
    await fs.writeFile(
      path.join(evidence, 'renderer-qualification.json'),
      JSON.stringify(report, null, 2) + '\n',
    );
    await fs.rm(scratch, { recursive: true, force: true });
  }
})()
  .then(() => process.stdout.write(JSON.stringify(report, null, 2) + '\n'))
  .catch((error) => {
    process.stderr.write(String(error.stack) + '\n');
    process.exitCode = 1;
  });
