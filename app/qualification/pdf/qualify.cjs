const path = require('node:path');
const fs = require('node:fs/promises');
const { strict: assert } = require('node:assert');
const { createHash } = require('node:crypto');
const { performance } = require('node:perf_hooks');
const { _electron: electron } = require('@playwright/test');
const { prepare, evidence } = require('./build.cjs');
const report = {
  date: new Date().toISOString(),
  status: 'running',
  scope: 'Synthetic source-checkout qualification; no production PDF integration or release claim.',
  cases: [],
  renders: [],
  errors: [],
};
let application, build;
async function save() {
  await fs.writeFile(
    path.join(evidence, 'qualification.json'),
    JSON.stringify(report, null, 2) + '\n',
  );
}
async function check(name, work) {
  const start = performance.now();
  let deadline;
  try {
    const details = await Promise.race([
      work(),
      new Promise((_, reject) => {
        deadline = setTimeout(
          () => reject(new Error('Qualification observer exceeded its 30-second deadline.')),
          30000,
        );
      }),
    ]);
    report.cases.push({
      name,
      result: 'passed',
      ms: performance.now() - start,
      ...(details === undefined ? {} : { details }),
    });
  } catch (error) {
    report.cases.push({
      name,
      result: 'product defect',
      message: error.message,
      ms: performance.now() - start,
    });
    throw error;
  } finally {
    clearTimeout(deadline);
    await save();
    console.log(name + ': ' + report.cases.at(-1).result);
  }
}
async function invoke(method, ...args) {
  return application.evaluate(
    async (_electron, { method, args }) => globalThis.pdfQualification[method](...args),
    { method, args },
  );
}
const withoutImage = (result) =>
  result.kind === 'page' ? { ...result, raster: { ...result.raster, png: '[omitted]' } } : result;
async function bitmapCheck(page, capture) {
  return application.evaluate(
    ({ nativeImage }, { page, capture }) => {
      const full = nativeImage.createFromBuffer(Buffer.from(page.raster.png, 'base64'));
      const crop = nativeImage.createFromBuffer(Buffer.from(capture.raster.png, 'base64'));
      const expected = full.crop(capture.pixelRect);
      const pixels = crop.toBitmap();
      const middle =
        (Math.floor(capture.raster.height / 2) * capture.raster.width +
          Math.floor(capture.raster.width / 2)) *
        4;
      return {
        samePixels: expected.toBitmap().equals(pixels),
        bgra: [...pixels.subarray(middle, middle + 4)],
        size: crop.getSize(),
      };
    },
    { page, capture },
  );
}
(async () => {
  await fs.mkdir(evidence, { recursive: true });
  const previous = await fs
    .readFile(path.join(evidence, 'qualification.json'), 'utf8')
    .catch(() => undefined);
  if (previous)
    await fs.writeFile(path.join(evidence, 'previous-' + Date.now() + '.json'), previous);
  try {
    build = await prepare();
    report.construction = 'passed: TypeScript + bundled Main/preloads/decoder/React entries';
    const geometry = require(path.join(build.scratch, 'geometry.cjs'));
    const protocol = require(path.join(build.scratch, 'protocol.cjs'));
    const fixtures = require(path.join(build.scratch, 'fixtures.cjs'));
    const raster = require(path.join(build.scratch, 'raster.cjs'));
    await check('pure geometry and closed selector validation', () => {
      const box = [20, 40, 580, 800],
        region = [100, 120, 220, 180];
      assert.throws(() => geometry.assertRegion([0, 0, 0, 1], box));
      assert.throws(() => geometry.assertRegion([19, 40, 30, 50], box));
      assert.throws(() => geometry.assertRegion([NaN, 50, 60, 70], box));
      assert.throws(() => geometry.invert([1, 1, 1, 1, 0, 0]));
      assert.throws(() => protocol.parseRegion([1, 2, 3, 4, 5]));
      assert.throws(() => protocol.parseRegion([1, 2, 3, Infinity]));
      assert.throws(() =>
        protocol.parseReply({
          jobId: 'x',
          ok: true,
          value: {
            kind: 'opened',
            revision: 'sha256:' + 'a'.repeat(64),
            profile: protocol.PROFILE,
            pageCount: 201,
          },
        }),
      );
      assert.throws(() => raster.assertPngHeader(Buffer.alloc(40).toString('base64'), 1, 1));
      geometry.assertRegion(region, box);
      return { invalidCases: 8 };
    });
    const fixtureManifest = [];
    for (const name of fixtures.FIXTURE_NAMES) {
      const bytes =
        name === 'encrypted'
          ? await fs.readFile(path.join(__dirname, 'fixtures/encrypted.pdf'))
          : fixtures.fixture(name);
      fixtureManifest.push({
        name,
        bytes: bytes.length,
        sha256: createHash('sha256').update(bytes).digest('hex'),
      });
    }
    await fs.writeFile(
      path.join(evidence, 'fixtures.json'),
      JSON.stringify(fixtureManifest, null, 2) + '\n',
    );
    application = await electron.launch({ args: [build.main], env: build.env });
    const preview = await application.firstWindow();
    await preview.waitForURL('app://gobble/index.html');
    preview.on('pageerror', (error) => report.errors.push(error.message));
    report.environment = await application.evaluate(({ app }) => ({
      platform: process.platform,
      arch: process.arch,
      versions: process.versions,
      metrics: app.getAppMetrics().map((m) => ({ type: m.type, pid: m.pid, memory: m.memory })),
    }));
    await check('real fixture through sandboxed preview bridge', async () => {
      await preview.getByRole('button', { name: 'Open PDF', exact: true }).click();
      await preview
        .getByRole('img', { name: 'PDF page rendered by the isolated decoder', exact: true })
        .waitFor();
      assert.match(await preview.getByRole('status').innerText(), /Ready/);
      const current = await invoke('current');
      assert.equal(current.model.pageIndex, 0);
      await preview.getByRole('button', { name: 'Capture known region', exact: true }).click();
      await preview.getByRole('img', { name: 'Captured PDF region', exact: true }).waitFor();
      await preview.screenshot({ path: path.join(evidence, 'native-page-and-capture.png') });
      return { page: withoutImage(current) };
    });
    await check('native source identity, pointer capture and compact controls', async () => {
      await preview.getByLabel('PDF fixture', { exact: true }).selectOption('scanned');
      assert.equal(await preview.locator('.reader .pane-title strong').innerText(), 'report.pdf');
      await preview.getByLabel('PDF fixture', { exact: true }).selectOption('report');
      const image = preview.getByRole('img', {
        name: 'PDF page rendered by the isolated decoder',
        exact: true,
      });
      const box = await image.boundingBox();
      assert.ok(box);
      await preview.mouse.move(box.x + 20, box.y + 20);
      await preview.mouse.down();
      await preview.mouse.move(box.x + 100, box.y + 70, { steps: 4 });
      await preview.mouse.up();
      await preview.waitForFunction(() =>
        document.querySelector('.capture-label')?.textContent?.includes('80 × 50'),
      );
      await application.evaluate(() =>
        globalThis.pdfQualification.preview().setContentSize(900, 650),
      );
      const button = preview.getByRole('button', { name: 'Capture known region', exact: true });
      await button.focus();
      await button.press('Enter');
      await preview.waitForFunction(() =>
        document.querySelector('.capture-label')?.textContent?.includes('120 × 60'),
      );
      const layout = await preview.evaluate(() => ({
        width: innerWidth,
        height: innerHeight,
        overflow: document.documentElement.scrollWidth > innerWidth,
        controls: [...document.querySelectorAll('button')].map((el) => {
          const b = el.getBoundingClientRect();
          return {
            name: el.textContent,
            visible: b.left >= 0 && b.right <= innerWidth && b.top >= 0 && b.bottom <= innerHeight,
          };
        }),
      }));
      assert.equal(layout.overflow, false);
      assert.ok(layout.controls.every((control) => control.visible));
      await preview.screenshot({ path: path.join(evidence, 'native-compact.png') });
      await application.evaluate(() =>
        globalThis.pdfQualification.preview().setContentSize(1280, 808),
      );
      return layout;
    });
    await check(
      'actual source geometry and retained-raster crop across transformations',
      async () => {
        let maxError = 0,
          limited = 0;
        for (const name of ['report', 'rotated', 'user-unit']) {
          await invoke('replace', name);
          let firstHash;
          for (const scale of [0.75, 1, 1.25])
            for (const dpr of [1, 2])
              for (const rotation of [0, 90, 180, 270]) {
                const pixels = 560 * 760 * (name === 'user-unit' ? 2 : 1) ** 2 * (scale * dpr) ** 2;
                if (pixels > protocol.LIMITS.rasterPixels) {
                  await assert.rejects(invoke('page', 0, scale, rotation, dpr), /4-megapixel/);
                  limited++;
                  continue;
                }
                const page = await invoke('page', 0, scale, rotation, dpr);
                if (!firstHash) firstHash = page.modelHash;
                else assert.equal(page.modelHash, firstHash);
                const matrix = page.viewport,
                  inv = geometry.invert(matrix),
                  rect = [100, 120, 220, 180];
                for (const [x, y] of [
                  [20, 40],
                  [580, 800],
                  [100, 120],
                  [220, 180],
                  [201.23, 455.89],
                ]) {
                  const [px, py] = geometry.transformPoint(matrix, x, y),
                    [backX, backY] = geometry.transformPoint(inv, px, py);
                  maxError = Math.max(maxError, Math.abs(backX - x), Math.abs(backY - y));
                }
                const capture = await invoke('capture', page.renditionId, page.modelHash, rect);
                const result = await bitmapCheck(page, capture);
                assert.equal(result.samePixels, true);
                assert.ok(
                  Math.abs(result.bgra[2] - 217) <= 3 &&
                    Math.abs(result.bgra[1] - 51) <= 3 &&
                    Math.abs(result.bgra[0] - 38) <= 3,
                );
                report.renders.push({
                  fixture: name,
                  scale,
                  dpr,
                  rotation,
                  width: page.raster.width,
                  height: page.raster.height,
                  ...page.timings,
                  pngBytes: Buffer.from(page.raster.png, 'base64').length,
                  cropPixelsMatch: true,
                });
              }
        }
        assert.ok(maxError < 1e-9);
        return {
          rendered: report.renders.length,
          pixelLimited: limited,
          maxUserSpaceRoundTripError: maxError,
          modelHashStableAcrossPresentations: true,
        };
      },
    );
    await check('stale rendition and changed source cannot capture', async () => {
      await invoke('replace', 'report');
      const old = await invoke('page', 0);
      await invoke('page', 1);
      await assert.rejects(
        invoke('capture', old.renditionId, old.modelHash, [100, 120, 220, 180]),
        /no longer current/,
      );
      await invoke('replace', 'rotated');
      await invoke('page', 0);
      await assert.rejects(
        invoke('capture', old.renditionId, old.modelHash, [100, 120, 220, 180]),
        /no longer current/,
      );
      return { oldTargetNeverSubstituted: true };
    });
    await check('empty scan and ambiguous text remain diagnostic-only', async () => {
      await invoke('replace', 'scanned');
      const scan = await invoke('page', 0);
      assert.equal(scan.model.text.items.length, 0);
      await invoke('replace', 'unicode');
      const unicode = await invoke('page', 0);
      const strings = unicode.model.text.items.map((item) => item.text);
      assert.equal(unicode.model.text.capability, 'diagnostic-only');
      assert.equal(strings.filter((s) => s === 'Same filtered cohort.').length, 2);
      await fs.writeFile(
        path.join(evidence, 'text-diagnostics.json'),
        JSON.stringify(unicode.model, null, 2) + '\n',
      );
      return {
        scanItems: 0,
        duplicates: 2,
        unicodeStrings: strings,
        textTargeting:
          'unsupported target or claim: per-character glyph mapping is not established',
      };
    });
    await check('file and page count limits', async () => {
      for (const name of ['limit-minus', 'limit-exact']) {
        const result = await invoke('replace', name);
        assert.equal(result.kind, 'opened');
        await invoke('page', 0);
      }
      await assert.rejects(invoke('replace', 'limit-plus'), /8 MiB/);
      await invoke('replace', 'many-pages');
      const last = await invoke('page', 199);
      assert.equal(last.model.pageIndex, 199);
      await assert.rejects(invoke('page', 200), /outside/);
      await assert.rejects(invoke('replace', 'too-many-pages'), /200-page/);
      return {
        admittedBytes: [protocol.LIMITS.fileBytes - 1, protocol.LIMITS.fileBytes],
        rejectedBytes: protocol.LIMITS.fileBytes + 1,
        maxPages: 200,
      };
    });
    await check('malformed and password-protected files fail locally', async () => {
      await assert.rejects(invoke('replace', 'malformed'), /PDF|Invalid|structure/i);
      await assert.rejects(invoke('replace', 'encrypted'), /password/i);
      await invoke('replace', 'report');
      await invoke('page', 0);
    });
    await check('noisy evidence enforces byte limits', async () => {
      await invoke('replace', 'noise');
      const page = await invoke('page', 0, 1.25, 0, 2);
      const size = Buffer.from(page.raster.png, 'base64').length;
      await assert.rejects(
        invoke('capture', page.renditionId, page.modelHash, page.model.viewBox),
        /evidence byte limit/,
      );
      const crop = await invoke('capture', page.renditionId, page.modelHash, [100, 120, 220, 180]);
      assert.ok(crop.raster.png.length <= protocol.LIMITS.capturePngBase64);
      return { pagePngBytes: size, smallerRegionAccepted: true };
    });
    await check('actual worker, separate process and offline asset policy', async () => {
      await invoke('replace', 'active');
      await invoke('page', 0);
      const result = await application.evaluate(async () => {
        const q = globalThis.pdfQualification,
          h = q.host(),
          w = h.testWindow();
        const preferences = w.webContents.getLastWebPreferences();
        const capabilities = await w.webContents.executeJavaScript(
          '({node:typeof window.require,workspace:typeof window.gobble,preview:typeof window.pdfPreview})',
        );
        const blocked = await w.webContents.executeJavaScript(
          'Promise.all([fetch("https://example.invalid/pdf-fixture").then(()=>false,()=>true),fetch("pdf-host://decoder/private.txt").then(r=>r.status===404)]).then(results=>({results,child:window.open("https://example.invalid/")===null}))',
        );
        return {
          preferences,
          capabilities,
          blocked,
          diagnostics: h.diagnostics(),
          previewPid: q.preview().webContents.getOSProcessId(),
        };
      });
      assert.equal(result.preferences.sandbox, true);
      assert.equal(result.preferences.nodeIntegration, false);
      assert.equal(result.preferences.contextIsolation, true);
      assert.equal(result.capabilities.node, 'undefined');
      assert.equal(result.capabilities.workspace, 'undefined');
      assert.equal(result.capabilities.preview, 'undefined');
      assert.ok(result.blocked.results.every(Boolean));
      assert.equal(result.blocked.child, true);
      assert.notEqual(result.diagnostics.pid, result.previewPid);
      assert.ok(result.diagnostics.requests.includes('/vendor/pdf.worker.mjs'));
      assert.ok(result.diagnostics.requests.some((s) => s.startsWith('/vendor/standard_fonts/')));
      return result;
    });
    await check('oversized and corrupt source images cannot silently disappear', async () => {
      const failures = [];
      for (const fixture of ['oversized-image', 'corrupt-image']) {
        await invoke('replace', fixture);
        await assert.rejects(invoke('page', 0), (error) => {
          assert.match(error.message, /Incomplete PDF/);
          failures.push({ fixture, message: error.message });
          return true;
        });
        assert.equal(await invoke('current'), undefined);
      }
      await invoke('replace', 'report');
      await invoke('page', 0);
      return { failures, recovered: true };
    });
    await check('preview bridge rejects injected capabilities', async () => {
      const result = await preview.evaluate(async () =>
        window.pdfPreview
          .request({ kind: 'open', fixture: 'report', path: '/not-authorized' })
          .then(
            () => false,
            () => true,
          ),
      );
      assert.equal(result, true);
      return { closedInputRejected: true };
    });
    await check('foreign sender cannot complete a pending decode job', async () => {
      const result = await application.evaluate(async ({ ipcMain }) => {
        const q = globalThis.pdfQualification;
        await q.replace('dense');
        const pending = q.page(0);
        const h = q.host(),
          preview = q.preview().webContents;
        ipcMain.emit(
          'pdf-qualification:reply',
          { sender: preview, senderFrame: preview.mainFrame },
          { jobId: 'forged', ok: false, message: 'foreign reply' },
        );
        const page = await pending;
        return { kind: page.kind, diagnostics: h.diagnostics() };
      });
      assert.equal(result.kind, 'page');
    });
    await check('cancellation destroys host and prevents late publication', async () => {
      await invoke('replace', 'dense');
      const result = await application.evaluate(async () => {
        const q = globalThis.pdfQualification,
          controller = new AbortController(),
          h = q.host();
        const started = Date.now(),
          pending = q.page(0, 1, 0, 1, controller.signal).then(
            () => ({ published: true }),
            (e) => ({ published: false, message: e.message }),
          );
        setTimeout(() => controller.abort(), 10);
        const result = await pending;
        return {
          ...result,
          ms: Date.now() - started,
          status: h.diagnostics(),
          current: q.current(),
        };
      });
      assert.equal(result.published, false);
      assert.match(result.message, /cancel/);
      assert.equal(result.status.closed, true);
      assert.equal(result.status.pending, false);
      assert.equal(result.current, undefined);
      await invoke('replace', 'report');
      await invoke('page', 0);
      return result;
    });
    await check('timeout ends a blocked decoder process', async () => {
      await invoke('replace', 'report');
      const result = await application.evaluate(async () => {
        const q = globalThis.pdfQualification,
          h = q.host();
        const blocked = h
          .testWindow()
          .webContents.executeJavaScript('{const end=Date.now()+8000;while(Date.now()<end){};true}')
          .catch(() => null);
        const start = Date.now(),
          outcome = await q.page(0).then(
            () => ({ published: true }),
            (e) => ({ published: false, message: e.message }),
          );
        void blocked;
        return { ...outcome, ms: Date.now() - start, status: h.diagnostics() };
      });
      assert.equal(result.published, false);
      assert.match(result.message, /timed out/);
      assert.ok(result.ms >= 4500 && result.ms < 7500);
      assert.equal(result.status.closed, true);
      await invoke('replace', 'report');
      await invoke('page', 0);
      return result;
    });
    await check('decoder crash is local and recoverable', async () => {
      await invoke('replace', 'dense');
      const result = await application.evaluate(async () => {
        const q = globalThis.pdfQualification,
          h = q.host(),
          pending = q.page(0).then(
            () => false,
            (e) => /exited/.test(e.message),
          );
        h.testWindow().webContents.forcefullyCrashRenderer();
        return {
          rejected: await pending,
          closed: h.diagnostics().closed,
          previewAlive: !q.preview().isDestroyed(),
        };
      });
      assert.equal(result.rejected, true);
      assert.equal(result.closed, true);
      assert.equal(result.previewAlive, true);
      await invoke('replace', 'report');
      await invoke('page', 0);
      return result;
    });
    await check('cold/warm latency and cleanup observations', async () => {
      const measurements = [];
      for (let i = 0; i < 3; i++) {
        const start = performance.now();
        await invoke('replace', 'report');
        const opened = performance.now();
        const page = await invoke('page', 0);
        const cold = performance.now();
        await invoke('page', 0);
        measurements.push({
          openMs: opened - start,
          coldPageMs: cold - opened,
          warmPageMs: performance.now() - cold,
          rasterPixels: page.raster.width * page.raster.height,
        });
      }
      const live = await application.evaluate(({ app }) =>
        app.getAppMetrics().map((m) => ({ type: m.type, pid: m.pid, memory: m.memory })),
      );
      await invoke('stop');
      const closed = await application.evaluate(({ app, BrowserWindow }) => ({
        windows: BrowserWindow.getAllWindows().length,
        metrics: app.getAppMetrics().map((m) => ({ type: m.type, pid: m.pid, memory: m.memory })),
        status: globalThis.pdfQualification.status(),
      }));
      assert.equal(closed.windows, 1);
      assert.ok(closed.status.retired.every((h) => h.closed && !h.pending));
      return { measurements, live, afterStop: closed };
    });
    assert.deepEqual(report.errors, []);
    report.status = 'passed';
    report.textDecision =
      'R3c2 should expose page/region only. Text extraction is diagnostic; exact glyph/range targeting is not qualified.';
    await save();
    console.log(
      'PDF qualification passed: ' +
        report.cases.length +
        ' cases, ' +
        report.renders.length +
        ' rendered transformations.',
    );
  } catch (error) {
    report.status = 'failed';
    report.failure = error.stack || error.message;
    await save();
    console.error(error.stdout?.toString() || error.stack);
    process.exitCode = 1;
  } finally {
    if (application) await application.close().catch(() => {});
    if (build) await fs.rm(build.scratch, { recursive: true, force: true });
  }
})();
