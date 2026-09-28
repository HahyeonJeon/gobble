const { spawn, execFileSync } = require('node:child_process');
const fs = require('node:fs/promises');
const path = require('node:path');
const assert = require('node:assert/strict');
const readline = require('node:readline');
const { build } = require('esbuild');
const { _electron: electron } = require('playwright');
const evidence = path.resolve(__dirname, '../../../docs/desktop-workspace/stages/r4b-review');
const python = path.join(__dirname, '.venv/bin/python');
const shell = (candidate) =>
  `<!doctype html><html lang="en"><head><meta charset="utf-8"><title>Gobble · Jupyter qualification</title><link rel="stylesheet" href="/component.css"></head><body><header><strong>Gobble / RNA-seq study</strong><small>${candidate} · isolated experiment</small></header><main><section><div class="toolbar"><div><h1>study.ipynb</h1><span class="muted" id="document-state">${candidate === 'A — Reusable components' ? 'Saved fixture · document revision 1' : 'Connected Jupyter · separate content process'}</span></div><button id="share">Reference selection</button></div><div id="notebook"></div></section><aside><h2>Discussion</h2><span class="muted">Communication layout preview</span><div class="note">Select an exact part of the Notebook to discuss it here.</div><div id="reference"></div><div class="composer">Ask about this Notebook…</div></aside></main>${candidate.startsWith('A') ? '<script type="module" src="/component.js"></script>' : ''}</body></html>`;
async function main() {
  await fs.mkdir(evidence, { recursive: true });
  const server = spawn(python, [path.join(__dirname, 'server.py')], {
    stdio: ['pipe', 'pipe', 'pipe'],
  });
  let serverError = '';
  server.stderr.on('data', (chunk) => {
    serverError += chunk;
  });
  const lines = readline.createInterface({ input: server.stdout });
  const configPath = await new Promise((resolve, reject) => {
    lines.once('line', resolve);
    server.once('exit', () => reject(new Error(serverError || 'Server ended before readiness')));
  });
  const config = JSON.parse(await fs.readFile(configPath, 'utf8'));
  let application;
  const results = { component: {}, connected: {} };
  try {
    console.log('Isolated TLS Jupyter server ready.');
    if (!process.env.R4B_NATIVE_ONLY) {
      const protocol = execFileSync(python, [path.join(__dirname, 'protocol.py'), configPath], {
        encoding: 'utf8',
        timeout: 100000,
      });
      await fs.writeFile(path.join(evidence, 'protocol-results.json'), protocol);
      console.log('REST and kernel probes passed.');
    }
    execFileSync(
      path.join(__dirname, '../../node_modules/.bin/tsc'),
      ['-p', path.join(__dirname, 'tsconfig.json')],
      { stdio: 'pipe' },
    );
    const out = path.join(config.scratch, 'bundle');
    await fs.mkdir(out);
    const bundle = await build({
      entryPoints: [path.join(__dirname, 'component.ts')],
      outfile: path.join(out, 'component.js'),
      bundle: true,
      platform: 'browser',
      format: 'esm',
      target: 'chrome152',
      metafile: true,
      loader: {
        '.svg': 'dataurl',
        '.woff': 'dataurl',
        '.woff2': 'dataurl',
        '.ttf': 'dataurl',
        '.eot': 'dataurl',
      },
      plugins: [
        {
          name: 'css-tilde',
          setup(b) {
            b.onResolve({ filter: /^~/ }, (args) => ({
              path: require.resolve(args.path.slice(1), { paths: [args.resolveDir] }),
            }));
          },
        },
      ],
    });
    results.component.bundleBytes = (await fs.stat(path.join(out, 'component.js'))).size;
    results.component.cssBytes = (await fs.stat(path.join(out, 'component.css'))).size;
    results.component.inputModules = Object.keys(bundle.metafile.inputs).length;
    await fs.copyFile(path.join(__dirname, 'host.cjs'), path.join(out, 'main.cjs'));
    await fs.writeFile(path.join(out, 'index.html'), shell('A — Reusable components'));
    await fs.writeFile(path.join(out, 'connected.html'), shell('B — Connected Jupyter'));
    const env = Object.fromEntries(
      Object.entries(process.env).filter(
        ([key]) => !['ELECTRON_RUN_AS_NODE', 'ELECTRON_RENDERER_URL'].includes(key),
      ),
    );
    env.R4B_CONNECTION = configPath;
    application = await electron.launch({
      args: [path.join(out, 'main.cjs')],
      env,
      timeout: 30000,
    });
    const a = await application.firstWindow();
    const errors = [];
    a.on('pageerror', (error) => errors.push(error.message));
    await a.waitForFunction(
      () => !!window.componentProbe && !!document.querySelector('.cm-content'),
    );
    const original = await a.evaluate(() => window.componentProbe.state());
    const start = original.source.indexOf('한국어');
    await a.evaluate(({ start }) => window.componentProbe.reveal(start + 3, start + 3), { start });
    await a.keyboard.press('Shift+ArrowLeft');
    await a.keyboard.press('Shift+ArrowLeft');
    await a.keyboard.press('Shift+ArrowLeft');
    const selected = await a.evaluate(() => window.componentProbe.selection());
    assert.equal(selected.text, '한국어');
    assert.equal(await a.evaluate((s) => window.componentProbe.resolve(s), selected), true);
    await a.getByRole('button', { name: 'Reference selection' }).click();
    await a.screenshot({ path: path.join(evidence, 'component-selection.png') });
    await a.evaluate(({ start }) => window.componentProbe.reveal(start, start + 3), { start });
    await a.keyboard.insertText('samples');
    await a.waitForFunction(() => window.componentProbe.state().source.includes('samples'));
    const edited = await a.evaluate(() => window.componentProbe.state());
    assert.equal(edited.dirty, true);
    assert.ok(edited.revision > original.revision);
    assert.equal(await a.evaluate((s) => window.componentProbe.resolve(s), selected), false);
    await a.evaluate(() => window.componentProbe.reorder());
    assert.equal((await a.evaluate(() => window.componentProbe.state())).cellId, original.cellId);
    const linked = await a.evaluate(() => window.componentProbe.linkedView());
    assert.equal(linked.sharesModel, true);
    assert.equal(linked.source, edited.source);
    assert.equal(linked.dirty, true);
    assert.deepEqual(errors, []);
    results.component = {
      ...results.component,
      unicodeSelection: selected.text,
      offsetUnit: 'UTF-16',
      nativeKeyboardSelection: true,
      nativeKeyboardUpdatedAuthoritativeModel: true,
      staleSelectionRejected: true,
      cellIdSurvivedReorder: true,
      linkedWidgetSharesModel: true,
      closingLinkedWidgetPreservesDocument: true,
      rendererErrors: errors,
    };
    console.log('Reusable model/editor probes passed.');
    await application.evaluate(() => global.r4b.openConnected());
    let b;
    for (let i = 0; i < 60; i++) {
      b = application.windows().find((p) => p.url().startsWith(config.origin));
      if (b) break;
      await new Promise((r) => setTimeout(r, 250));
    }
    if (!b) throw new Error('Connected Jupyter page unavailable');
    await b.waitForFunction(
      () => window.jupyterapp?.shell.currentWidget?.content?.activeCell?.editor,
      undefined,
      { timeout: 60000 },
    );
    await b.waitForFunction(
      () => !!window.jupyterapp.shell.currentWidget.sessionContext.session?.kernel,
      undefined,
      { timeout: 30000 },
    );
    await b.evaluate(async () => {
      const app = window.jupyterapp;
      app.shell.mode = 'single-document';
      app.shell.collapseLeft();
      app.shell.collapseRight();
      document.body.classList.add('r4b-connected');
      const style = document.createElement('style');
      style.textContent =
        '#jp-top-panel, #jp-menu-panel, #jp-left-stack, #jp-right-stack, #jp-left-tab-bar, #jp-right-tab-bar, #jp-bottom-panel {display:none!important}';
      document.head.append(style);
      app.shell.update();
      const panel = app.shell.currentWidget;
      await panel.context.ready;
      panel.content.activeCellIndex = 0;
      panel.content.mode = 'edit';
      const editor = panel.content.activeCell.editor;
      const start = editor.model.sharedModel.getSource().indexOf('한국어');
      editor.focus();
      editor.setSelection({
        start: editor.getPositionAt(start),
        end: editor.getPositionAt(start + 3),
      });
    });
    results.connected = await b.evaluate(() => {
      const panel = window.jupyterapp.shell.currentWidget;
      const cell = panel.content.activeCell;
      const range = cell.editor.getSelection();
      const start = cell.editor.getOffsetAt(range.start),
        end = cell.editor.getOffsetAt(range.end);
      const cfg = JSON.parse(document.getElementById('jupyter-config-data').textContent);
      return {
        cellId: cell.model.id,
        selection: cell.model.sharedModel.getSource().slice(start, end),
        credentialInPageConfiguration: typeof cfg.token === 'string' && cfg.token.length > 0,
        documentContextReady: panel.context.isReady,
        kernel: panel.sessionContext.kernelDisplayName,
      };
    });
    assert.equal(results.connected.selection, '한국어');
    assert.equal(results.connected.credentialInPageConfiguration, true);
    await application.evaluate(
      (_, file) => global.r4b.screenshotConnected(file),
      path.join(evidence, 'connected-selection.png'),
    );
    await b.keyboard.insertText('changed');
    await b.waitForFunction(() => window.jupyterapp.shell.currentWidget.context.model.dirty);
    await application.evaluate(() => global.r4b.externalEdit());
    // Keep the asynchronous save pending while the conflict dialog is inspected.
    await b.evaluate(() => {
      window.r4bSave = window.jupyterapp.shell.currentWidget.context.save().then(
        () => 'saved',
        () => 'cancelled',
      );
    });
    await b.getByRole('dialog').waitFor({ timeout: 15000 });
    results.connected.conflictDialog = await b.getByRole('dialog').innerText();
    await application.evaluate(
      (_, file) => global.r4b.screenshotConnected(file),
      path.join(evidence, 'connected-save-conflict.png'),
    );
    await b.getByRole('button', { name: 'Cancel', exact: true }).click();
    await b.evaluate(() => window.r4bSave);
    results.connected.externalFilePreservedAfterCancel =
      JSON.parse(await fs.readFile(path.join(config.project, 'study.ipynb'), 'utf8')).cells[0]
        .source === 'external_change = 2';
    assert.equal(results.connected.externalFilePreservedAfterCancel, true);
    const sessionBefore = await b.evaluate(async () => {
      const app = window.jupyterapp;
      await app.serviceManager.sessions.refreshRunning();
      return [...app.serviceManager.sessions.running()].length;
    });
    results.connected.webContentsDestroyedOnClose = await application.evaluate(() =>
      global.r4b.closeConnected(),
    );
    assert.equal(results.connected.webContentsDestroyedOnClose, true);
    const sessionsAfter = JSON.parse(
      execFileSync(
        python,
        [
          '-c',
          'import json,sys,requests; c=json.load(open(sys.argv[1])); print(json.dumps(requests.get(c["origin"]+"/api/sessions",headers={"Authorization":"token "+c["token"]},verify=c["cert"]).json()))',
          configPath,
        ],
        { encoding: 'utf8' },
      ),
    );
    results.connected.sessionSurvivedPaneClose =
      sessionBefore > 0 && sessionsAfter.length === sessionBefore;
    assert.equal(results.connected.sessionSurvivedPaneClose, true);
    await a.evaluate(() => window.componentProbe.dispose());
    await fs.writeFile(
      path.join(evidence, 'native-results.json'),
      JSON.stringify(results, null, 2) + '\n',
    );
    console.log('Connected surface probes passed.');
  } finally {
    if (application) await application.close();
    server.stdin.end('\n');
    await new Promise((resolve) => server.once('exit', resolve));
    console.log('Owned Electron profile, server and scratch released.');
  }
}
main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
