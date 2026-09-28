const path = require('node:path');
const fs = require('node:fs/promises');
const assert = require('node:assert/strict');
const { createHash } = require('node:crypto');
const { execFileSync } = require('node:child_process');
const { _electron: electron } = require('@playwright/test');
const { prepare, evidence, appRoot } = require('./build.cjs');
const report = {
  request: 'R4a1-2026-09-08',
  started: new Date().toISOString(),
  scope:
    'Isolated source-checkout Notebook qualification; no production/Agent/kernel/package claim',
  cases: [],
  errors: [],
};
let build, application, page;
async function save() {
  await fs.writeFile(
    path.join(evidence, 'qualification.json'),
    JSON.stringify(report, null, 2) + '\n',
  );
}
async function check(name, work) {
  const start = performance.now();
  try {
    const details = await work();
    report.cases.push({ name, result: 'passed', ms: performance.now() - start, details });
  } catch (error) {
    report.cases.push({
      name,
      result: 'product defect',
      ms: performance.now() - start,
      message: error.stack,
    });
    throw error;
  } finally {
    await save();
    console.log(name + ': ' + report.cases.at(-1).result);
  }
}
async function invoke(method, ...args) {
  return application.evaluate(
    async (_, { method, args }) => {
      const q = globalThis.notebookQualification,
        h = [...q.readers.values()][0].host;
      return await h[method](...args);
    },
    { method, args },
  );
}
async function open(name) {
  await page.getByLabel('Notebook fixture', { exact: true }).selectOption(name);
  await page.getByRole('button', { name: 'Open notebook', exact: true }).click();
  await page.waitForFunction(() =>
    document.querySelector('[role=status]')?.textContent?.startsWith('Ready'),
  );
}
async function screenshot(options) {
  const base64 = await application.evaluate(async () => {
    const w = [...globalThis.notebookQualification.readers.values()][0].window;
    return (await w.webContents.capturePage()).toPNG().toString('base64');
  });
  await fs.writeFile(options.path, Buffer.from(base64, 'base64'));
}
async function receipt() {
  for (let i = 0; i < 30; i++) {
    try {
      return await invoke('observe');
    } catch {
      await page.waitForTimeout(50);
    }
  }
  throw Error('No ready receipt');
}
const sourceTarget = (s, start = 0, end = 4) => ({
  revision: s.revision,
  profile: s.profile,
  cell: { kind: 'id', id: 'filter' },
  part: { kind: 'source' },
  selector: { kind: 'text', start, end },
});
(async () => {
  await fs.mkdir(evidence, { recursive: true });
  const old = await fs
    .readFile(path.join(evidence, 'qualification.json'), 'utf8')
    .catch(() => null);
  if (old) await fs.writeFile(path.join(evidence, 'previous-' + Date.now() + '.json'), old);
  try {
    build = await prepare();
    const unit = execFileSync(
      path.join(appRoot, 'node_modules/.bin/vitest'),
      ['run', '--config', 'qualification/notebook/test.config.ts'],
      { cwd: appRoot, encoding: 'utf8' },
    );
    await fs.writeFile(path.join(evidence, 'unit.log'), unit);
    report.construction = 'Strict TypeScript, bundled entries and pure tests passed (unit.log)';
    const fixtures = require(path.join(build.scratch, 'fixtures.cjs')),
      parser = require(path.join(build.scratch, 'parser.cjs')),
      worker = require(path.join(build.scratch, 'worker-client.cjs'));
    const inventory = [];
    for (const name of fixtures.FIXTURES.filter((n) => n !== 'jpeg')) {
      const bytes = fixtures.fixture(name);
      inventory.push({
        name,
        bytes: bytes.length,
        sha256: createHash('sha256').update(bytes).digest('hex'),
      });
    }
    await fs.writeFile(
      path.join(evidence, 'fixtures.json'),
      JSON.stringify(inventory, null, 2) + '\n',
    );
    await check(
      'Worker cancellation, deadline and two-read admission release resources',
      async () => {
        const abort = new AbortController(),
          pending = worker.readNotebook(fixtures.fixture('large'), abort.signal);
        abort.abort();
        await assert.rejects(pending, /cancelled/);
        assert.equal(worker.activeWorkers(), 0);
        await assert.rejects(
          worker.readNotebook(fixtures.fixture('large'), new AbortController().signal, 1),
          /deadline/,
        );
        assert.equal(worker.activeWorkers(), 0);
        const one = worker.readNotebook(fixtures.fixture('large'), new AbortController().signal),
          two = worker.readNotebook(fixtures.fixture('many'), new AbortController().signal);
        await assert.rejects(
          worker.readNotebook(fixtures.fixture('python'), new AbortController().signal),
          /Two/,
        );
        await Promise.all([one, two]);
        assert.equal(worker.activeWorkers(), 0);
      },
    );
    await check(
      'Measured embedded-image source and 1000-cell parsing justify a named bounded transport',
      async () => {
        const measures = [];
        for (const name of ['python', 'large', 'many', 'text-limit']) {
          const bytes =
            name === 'text-limit'
              ? fixtures.encode(fixtures.notebook([fixtures.code('filter', 'x'.repeat(1048576))]))
              : fixtures.fixture(name);
          const start = performance.now();
          const s = await worker.readNotebook(bytes, new AbortController().signal);
          assert.equal(s.revision, parser.parseNotebook(bytes).revision);
          measures.push({
            name,
            bytes: bytes.length,
            cells: s.cells.length,
            textBytes: s.textBytes,
            ms: performance.now() - start,
          });
        }
        return measures;
      },
    );
    application = await electron.launch({ args: [build.main], env: build.env });
    page = await application.firstWindow();
    await page.waitForURL('notebook://review/index.html');
    page.on('pageerror', (e) => report.errors.push(e.message));
    report.environment = await application.evaluate(({ app }) => ({
      platform: process.platform,
      arch: process.arch,
      versions: process.versions,
      metrics: app.getAppMetrics(),
    }));
    await check(
      'Actual saved Notebook through emitted sandbox bridge; passive source/output fallback',
      async () => {
        await open('python');
        const s = await invoke('view');
        assert.equal(s.cells.length, 3);
        assert.equal(s.cells[2].outputs[0].mime, 'text/plain');
        assert.equal(s.cells[2].outputs[2].part.kind, 'unavailable');
        assert.equal(
          await page
            .locator('.markdown img,.markdown script,.output iframe,.output script')
            .count(),
          0,
        );
        assert.equal(await page.evaluate(() => globalThis.notebookAttack), undefined);
        return { cells: s.cells.length, revision: s.revision };
      },
    );
    await check(
      'Native text drag identifies the exact source range and captures it without changing draft',
      async () => {
        const pre = page.locator('[data-part="source-2"]');
        await pre.scrollIntoViewIfNeeded();
        const points = await pre.evaluate((el) => {
          const node = el.querySelector('span').firstChild;
          const a = document.createRange(),
            b = document.createRange();
          a.setStart(node, 0);
          a.setEnd(node, 1);
          b.setStart(node, 4);
          b.setEnd(node, 5);
          const x = a.getBoundingClientRect(),
            y = b.getBoundingClientRect();
          return { x1: x.left + 1, y1: x.top + x.height / 2, x2: y.left, y2: y.top + y.height / 2 };
        });
        await page.mouse.move(points.x1, points.y1);
        await page.mouse.down();
        await page.mouse.move(points.x2, points.y2, { steps: 5 });
        await page.mouse.up();
        await pre
          .locator('..')
          .getByRole('button', { name: 'Use text selection', exact: true })
          .click();
        await page.getByRole('button', { name: 'Discuss selection', exact: true }).click();
        await page.waitForFunction(
          () => document.querySelector('.evidence pre')?.textContent === 'keep',
        );
        assert.equal(
          await page.locator('#draft').inputValue(),
          'Should we revisit this threshold?',
        );
        assert.equal(
          await page.locator('#draft').evaluate((el) => el === document.activeElement),
          true,
        );
        await screenshot({ path: path.join(evidence, 'source-capture.png') });
      },
    );
    await check(
      'Visibility receipt excludes folded, offscreen and clipped text; scrolling expires earlier observations',
      async () => {
        await page.locator('[data-part="source-2"]').scrollIntoViewIfNeeded();
        const r = await receipt();
        const s = await invoke('view');
        const p = r.parts.find(
          (p) => p.cell.kind === 'id' && p.cell.id === 'filter' && p.part.kind === 'source',
        );
        assert.ok(p);
        const t = { ...sourceTarget(s), selector: p.selectors[0] };
        await invoke('capture', t, r.generation);
        await page.locator('.reader').evaluate((el) => {
          const prior = el.scrollTop;
          el.scrollTop = prior < 50 ? 80 : 0;
          if (el.scrollTop === prior) throw new Error('Scroll fixture did not move.');
        });
        await page.waitForTimeout(80);
        await assert.rejects(invoke('capture', t, r.generation), /expired|visible|ready/);
        await page.getByRole('button', { name: 'Collapse outputs', exact: true }).click();
        const folded = await receipt();
        assert.ok(!folded.parts.some((p) => p.part.kind === 'output'));
        await page.getByRole('button', { name: 'Expand outputs', exact: true }).click();
      },
    );
    await check(
      'PNG display and scaled pointer selection preserve natural pixel crop',
      async () => {
        await page.getByRole('button', { name: 'View saved image', exact: true }).click();
        const img = page.getByAltText('Saved Notebook output', { exact: true });
        await img.waitFor();
        await img.scrollIntoViewIfNeeded();
        await page.getByLabel('Image scale', { exact: true }).selectOption('150');
        await img.scrollIntoViewIfNeeded();
        const box = await img.boundingBox();
        await page.mouse.move(box.x + 30, box.y + 15);
        await page.mouse.down();
        await page.mouse.move(box.x + 150, box.y + 75, { steps: 5 });
        await page.mouse.up();
        await page.getByRole('button', { name: 'Discuss selection', exact: true }).click();
        await page.getByAltText('Captured attachment', { exact: true }).waitFor();
        const s = await invoke('view'),
          o = s.cells[2].outputs[1];
        const target = {
          revision: s.revision,
          profile: s.profile,
          cell: { kind: 'id', id: 'filter' },
          part: { kind: 'output', index: 1, mime: o.mime, digest: o.part.digest },
          selector: { kind: 'image', rect: { x: 20, y: 10, width: 80, height: 40 } },
        };
        const captured = await invoke('capture', target);
        const same = await application.evaluate(
          ({ nativeImage }, { captured }) => {
            const q = globalThis.notebookQualification,
              raw = q.source('python'),
              data = JSON.parse(raw.toString()).cells[2].outputs[1].data['image/png'];
            return nativeImage
              .createFromBuffer(Buffer.from(captured.base64, 'base64'))
              .toBitmap()
              .equals(
                nativeImage
                  .createFromBuffer(Buffer.from(data, 'base64'))
                  .crop({ x: 20, y: 10, width: 80, height: 40 })
                  .toBitmap(),
              );
          },
          { captured },
        );
        assert.equal(same, true);
        const dims = await page
          .getByAltText('Captured attachment', { exact: true })
          .evaluate((el) => ({ w: el.naturalWidth, h: el.naturalHeight }));
        assert.deepEqual(dims, { w: 80, h: 40 });
        await screenshot({ path: path.join(evidence, 'image-region.png') });
      },
    );
    await check(
      'Captured Show/Return preserves source reader scroll and draft; new source keeps old evidence',
      async () => {
        const scroll = await page.locator('.reader').evaluate((el) => el.scrollTop);
        await page.getByRole('button', { name: 'Show captured reference', exact: true }).click();
        await page.getByRole('button', { name: 'Return to my view', exact: true }).click();
        assert.equal(await page.locator('.reader').evaluate((el) => el.scrollTop), scroll);
        assert.equal(
          await page.locator('#draft').inputValue(),
          'Should we revisit this threshold?',
        );
        await open('changed');
        await page.getByRole('button', { name: 'Show captured reference', exact: true }).click();
        assert.match(await page.locator('.reference').innerText(), /Earlier version/);
        await screenshot({ path: path.join(evidence, 'earlier-version.png') });
        await page.getByRole('button', { name: 'Return to my view', exact: true }).click();
      },
    );
    await check('JPEG decode, malformed image refusal and Unicode raw quote', async () => {
      await open('jpeg');
      await page.getByRole('button', { name: 'View saved image', exact: true }).click();
      await page.getByAltText('Saved Notebook output', { exact: true }).waitFor();
      const s = await invoke('view');
      const p = s.cells[1].source;
      const target = {
        ...sourceTarget(s),
        cell: { kind: 'id', id: 'load' },
        selector: { kind: 'text', start: 0, end: p.text.indexOf('\n') + 1 },
      };
      const c = await invoke('capture', target);
      assert.ok(c.rawQuote.endsWith('\r\n'));
      assert.ok(c.text.endsWith('\n') && !c.text.includes('\r'));
      const outcomes = await application.evaluate(async () => {
        const q = globalThis.notebookQualification,
          h = [...q.readers.values()][0].host,
          nb = JSON.parse(q.source('python').toString());
        let base = Buffer.from(nb.cells[2].outputs[1].data['image/png'], 'base64');
        base.fill(0, 45, 60);
        nb.cells[2].outputs[1].data['image/png'] = base.toString('base64');
        const s = await h.load(Buffer.from(JSON.stringify(nb))),
          o = s.cells[2].outputs[1],
          t = {
            revision: s.revision,
            profile: s.profile,
            cell: { kind: 'id', id: 'filter' },
            part: { kind: 'output', index: 1, mime: o.mime, digest: o.part.digest },
            selector: { kind: 'image', rect: { x: 0, y: 0, width: 320, height: 160 } },
          };
        try {
          h.image(t);
          return 'unexpected success';
        } catch (e) {
          return e.message;
        }
      });
      assert.match(outcomes, /decoding failed/);
      await open('python');
    });
    await check(
      'Host enforces text/pixel/encoded capture budgets and isolates returned view data',
      async () => {
        const checks = await application.evaluate(async () => {
          const q = globalThis.notebookQualification,
            h = [...q.readers.values()][0].host;
          const nb = JSON.parse(q.source('python').toString());
          nb.cells[2].source = 'x'.repeat(65537);
          let s = await h.load(Buffer.from(JSON.stringify(nb))),
            t = {
              revision: s.revision,
              profile: s.profile,
              cell: { kind: 'id', id: 'filter' },
              part: { kind: 'source' },
              selector: { kind: 'text', start: 0, end: 65536 },
            };
          const exact = h.capture(t).text.length;
          let over = '';
          try {
            h.capture({ ...t, selector: { kind: 'text', start: 0, end: 65537 } });
          } catch (e) {
            over = e.message;
          }
          s.cells[2].address.id = 'mutated';
          const intact = h.view().cells[2].address.id;
          return { exact, over, intact };
        });
        assert.equal(checks.exact, 65536);
        assert.match(checks.over, /64 KiB/);
        assert.equal(checks.intact, 'filter');
        const dimensions = fixtures.png(1600, 100).toString('base64');
        const edge = await application.evaluate(
          async (_, { dimensions }) => {
            const q = globalThis.notebookQualification,
              h = [...q.readers.values()][0].host,
              nb = JSON.parse(q.source('python').toString());
            nb.cells[2].outputs[1].data = { 'image/png': dimensions };
            const s = await h.load(Buffer.from(JSON.stringify(nb))),
              o = s.cells[2].outputs[1],
              t = {
                revision: s.revision,
                profile: s.profile,
                cell: { kind: 'id', id: 'filter' },
                part: { kind: 'output', index: 1, mime: o.mime, digest: o.part.digest },
                selector: { kind: 'image', rect: { x: 0, y: 0, width: 1536, height: 100 } },
              };
            const accepted = h.capture(t).width;
            try {
              h.capture({
                ...t,
                selector: { kind: 'image', rect: { x: 0, y: 0, width: 1537, height: 100 } },
              });
              return { accepted, error: '' };
            } catch (e) {
              return { accepted, error: e.message };
            }
          },
          { dimensions },
        );
        assert.equal(edge.accepted, 1536);
        assert.match(edge.error, /1536 pixels/);
        await open('python');
        return {
          text: checks,
          pixels: edge,
          encodedBudget: 'Separate high-entropy scenario',
        };
      },
    );
    await check(
      'High-entropy image capture refuses excessive bytes; native 4 MP decoding succeeds',
      async () => {
        const noise = fixtures.png(600, 600, true).toString('base64'),
          boundary = fixtures.png(2000, 2000).toString('base64');
        const result = await application.evaluate(
          async (_, { noise, boundary }) => {
            const q = globalThis.notebookQualification,
              h = [...q.readers.values()][0].host;
            const outcomes = [];
            for (const [base64, width, height] of [
              [noise, 600, 600],
              [boundary, 2000, 2000],
            ]) {
              const nb = JSON.parse(q.source('python').toString());
              nb.cells[2].outputs[1].data = { 'image/png': base64 };
              const s = await h.load(Buffer.from(JSON.stringify(nb))),
                o = s.cells[2].outputs[1],
                t = {
                  revision: s.revision,
                  profile: s.profile,
                  cell: { kind: 'id', id: 'filter' },
                  part: { kind: 'output', index: 1, mime: o.mime, digest: o.part.digest },
                  selector: { kind: 'image', rect: { x: 0, y: 0, width, height } },
                };
              const start = performance.now();
              const image = h.image(t);
              let capture;
              try {
                capture = h.capture(t).representation;
              } catch (e) {
                capture = e.message;
              }
              outcomes.push({
                width: image.width,
                height: image.height,
                ms: performance.now() - start,
                capture,
              });
            }
            return outcomes;
          },
          { noise, boundary },
        );
        assert.match(result[0].capture, /768 KiB/);
        assert.equal(result[1].width, 2000);
        assert.match(result[1].capture, /1536 pixels/);
        await open('python');
        return result;
      },
    );
    await check(
      'Native keyboard source and image-region controls create exact captures',
      async () => {
        const source = page.locator('[data-part="source-2"]').locator('..');
        await source.locator('summary').click();
        await source.getByLabel('Start UTF-16').fill('0');
        await source.getByLabel('End UTF-16').fill('4');
        await source.getByRole('button', { name: 'Use range', exact: true }).focus();
        await page.keyboard.press('Enter');
        await page.getByRole('button', { name: 'Discuss selection', exact: true }).click();
        await page.waitForFunction(
          () => document.querySelector('.evidence pre')?.textContent === 'keep',
        );
        await page.getByRole('button', { name: 'View saved image', exact: true }).click();
        await page.getByAltText('Saved Notebook output', { exact: true }).waitFor();
        await page.getByText('Select region by coordinates', { exact: true }).click();
        for (const [field, value] of Object.entries({ x: 10, y: 5, width: 30, height: 20 }))
          await page.getByLabel('Region ' + field, { exact: true }).fill(String(value));
        await page.getByRole('button', { name: 'Use image region', exact: true }).focus();
        await page.keyboard.press('Enter');
        await page.getByRole('button', { name: 'Discuss selection', exact: true }).click();
        await page.getByAltText('Captured attachment', { exact: true }).waitFor();
        assert.deepEqual(
          await page
            .getByAltText('Captured attachment', { exact: true })
            .evaluate((el) => [el.naturalWidth, el.naturalHeight]),
          [30, 20],
        );
      },
    );
    await check(
      'A 1 MiB source renders bounded text windows and advances exact absolute offsets',
      async () => {
        const start = performance.now();
        await open('long-text');
        const pre = page.locator('[data-part="source-0"]');
        assert.equal((await pre.innerText()).length, 2048);
        await page.getByRole('button', { name: 'Next text', exact: true }).click();
        assert.equal(await pre.locator('span').first().getAttribute('data-start'), '2048');
        const scoped = await receipt();
        assert.ok(scoped.parts.length > 0);
        assert.ok(
          scoped.parts.every((p) => p.selectors.every((s) => s.kind !== 'text' || s.start >= 2048)),
        );
        await page.getByRole('button', { name: 'Previous text', exact: true }).click();
        assert.equal(await pre.locator('span').first().getAttribute('data-start'), '0');
        const ms = performance.now() - start;
        assert.ok(ms < 2000, 'Bounded source interaction exceeded 2 seconds: ' + ms);
        await open('python');
        return { ms, sourceUnits: 1048576, renderedUnits: 2048 };
      },
    );
    await check('Native sender, bridge, network, popup and CSP boundaries', async () => {
      const prefs = await application.evaluate(() =>
        [
          ...globalThis.notebookQualification.readers.values(),
        ][0].window.webContents.getLastWebPreferences(),
      );
      assert.equal(prefs.sandbox, true);
      assert.equal(prefs.contextIsolation, true);
      assert.equal(prefs.nodeIntegration, false);
      assert.equal(prefs.webSecurity, true);
      assert.deepEqual(await page.evaluate(() => Object.keys(window.notebook).sort()), [
        'capture',
        'image',
        'invalidate',
        'open',
        'visible',
      ]);
      const bad = await page.evaluate(() => window.notebook.open('../../secret'));
      assert.equal(bad.ok, false);
      const failures = await page.evaluate(async () => {
        const a = [];
        try {
          await fetch('https://example.invalid/');
        } catch {
          a.push('fetch blocked');
        }
        const script = document.createElement('script');
        script.textContent = 'globalThis.scriptAttack=true';
        document.body.append(script);
        window.open('https://example.invalid/');
        return { a, attack: globalThis.scriptAttack, require: typeof require };
      });
      assert.deepEqual(failures.a, ['fetch blocked']);
      assert.equal(failures.attack, undefined);
      assert.equal(failures.require, 'undefined');
      assert.equal(application.windows().length, 1);
      assert.equal(
        await application.evaluate(() => {
          try {
            globalThis.notebookQualification.owner({ sender: { id: -1 }, senderFrame: null });
            return false;
          } catch {
            return true;
          }
        }),
        true,
      );
    });
    await check(
      '900x650 compact reader and 150% native zoom keep the composer reachable',
      async () => {
        await application.evaluate(() =>
          [...globalThis.notebookQualification.readers.values()][0].window.setContentSize(900, 650),
        );
        await page.waitForFunction(() => innerWidth === 900);
        await screenshot({ path: path.join(evidence, 'compact.png') });
        await application.evaluate(() => {
          const w = [...globalThis.notebookQualification.readers.values()][0].window;
          w.setContentSize(1440, 1000);
          w.webContents.setZoomFactor(1.5);
        });
        await page.waitForFunction(() => innerWidth >= 955 && innerWidth <= 965);
        const b = await page.locator('#draft').boundingBox();
        const h = await page.evaluate(() => innerHeight);
        assert.ok(b.y + b.height <= h);
        await screenshot({ path: path.join(evidence, 'zoom-150.png') });
        await application.evaluate(() =>
          [
            ...globalThis.notebookQualification.readers.values(),
          ][0].window.webContents.setZoomFactor(1),
        );
      },
    );
    await check(
      'Two readers remain independent, reject a third, and release workers on close',
      async () => {
        await application.evaluate(() => globalThis.notebookQualification.createReader());
        const second = application.windows().at(-1);
        await second.waitForURL('notebook://review/index.html');
        await second.getByLabel('Notebook fixture', { exact: true }).selectOption('many');
        await second.getByRole('button', { name: 'Open notebook', exact: true }).click();
        await second.waitForFunction(() =>
          document.querySelector('[role=status]')?.textContent?.startsWith('Ready'),
        );
        const metrics = await application.evaluate(({ app }) => ({
          readers: globalThis.notebookQualification.readers.size,
          metrics: app.getAppMetrics(),
        }));
        assert.equal(metrics.readers, 2);
        await assert.rejects(
          application.evaluate(() => globalThis.notebookQualification.createReader()),
          /two readers/,
        );
        await second.close();
        await application.evaluate(() => new Promise((r) => setTimeout(r, 50)));
        assert.equal(
          await application.evaluate(() => globalThis.notebookQualification.activeWorkers()),
          0,
        );
        return metrics;
      },
    );
    assert.deepEqual(report.errors, []);
    report.status = 'passed';
    report.finished = new Date().toISOString();
    await save();
  } catch (error) {
    report.status = 'failed';
    report.failure = String(error);
    await save();
    console.error(error);
    process.exitCode = 1;
  } finally {
    if (application) await application.close().catch(() => {});
    if (build) await build.cleanup();
  }
})();
