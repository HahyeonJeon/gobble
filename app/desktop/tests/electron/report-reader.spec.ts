import { _electron as electron, test, expect, type ElectronApplication } from '@playwright/test';
import { build } from 'esbuild';
import { mkdtemp, writeFile, readFile, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join, resolve } from 'node:path';
import type { FastqcContent } from '@gobble/contracts';

let application: ElectronApplication, base: string, host: string, source: string;
test.beforeAll(async () => {
  base = await mkdtemp(join(tmpdir(), 'gobble-report-reader-'));
  host = join(base, 'reader.cjs');
  await build({
    entryPoints: [resolve('desktop/src/main/fastqc.ts')],
    outfile: host,
    bundle: true,
    platform: 'node',
    format: 'cjs',
    external: ['electron'],
    logLevel: 'silent',
  });
  const entry = join(base, 'entry.cjs');
  await writeFile(
    entry,
    "const {app}=require('electron');globalThis.FastqcReader=require(" +
      JSON.stringify(host) +
      ").FastqcReader;app.on('window-all-closed',()=>{});app.whenReady();",
  );
  const env = Object.fromEntries(
    Object.entries(process.env).filter(
      (entry): entry is [string, string] =>
        entry[1] !== undefined && entry[0] !== 'ELECTRON_RUN_AS_NODE',
    ),
  );
  application = await electron.launch({ args: [entry], env });
  source = await readFile(resolve('desktop/tests/fixtures/fastqc-0.12.1.html'), 'utf8');
});
test.afterAll(async () => {
  await application?.close();
  await rm(base, { recursive: true, force: true });
});
async function decode(html: string): Promise<FastqcContent> {
  return application.evaluate(
    async (_electron, { script, html }) => {
      const { FastqcReader } = globalThis as unknown as {
        FastqcReader: typeof import('../../src/main/fastqc').FastqcReader;
      };
      const reader = new FastqcReader(script);
      try {
        return await reader.read(Buffer.from(html));
      } finally {
        reader.close();
      }
    },
    { host, script: resolve('desktop/out/report-reader/reader.js'), html },
  );
}
test('qualified FastQC source preserves ordered text, tables, status variants and all original charts', async () => {
  const content = await decode(source);
  expect(content.modules).toHaveLength(10);
  expect(content.summary).toHaveLength(10);
  expect(content.modules.map((m) => m.id)).toEqual([
    'M0',
    'M1',
    'M3',
    'M4',
    'M5',
    'M6',
    'M7',
    'M8',
    'M9',
    'M10',
  ]);
  expect(content.summary[4]!.status).toBe('[WARNING]');
  expect(content.modules[4]!.status).toBe('[WARN]');
  expect(content.summary[0]!.status).toBe('[PASS]');
  expect(content.modules[0]!.status).toBe('[OK]');
  const images = content.modules.flatMap((m) => m.blocks).filter((b) => b.kind === 'image');
  const expected = [
    ...source.matchAll(
      /<img class="indented" src="data:image\/png;base64,([A-Za-z0-9+/=]+)" alt="([^"]*)"\/>/g,
    ),
  ];
  expect(images).toHaveLength(8);
  expect(images.map((i) => [i.base64, i.alt])).toEqual(expected.map((m) => [m[1], m[2]]));
  expect(images.map((i) => [i.width, i.height])).toEqual([
    ...Array.from({ length: 7 }, () => [800, 600]),
    [1035, 600],
  ]);
  expect(content.modules[0]!.blocks[0]).toEqual({
    kind: 'table',
    headers: ['Measure', 'Value'],
    rows: [
      ['Filename', 'sample_trimmed.fq.gz'],
      ['File type', 'Conventional base calls'],
      ['Encoding', 'Illumina 1.5'],
      ['Total Sequences', '100'],
      ['Total Bases', '8 kbp'],
      ['Sequences flagged as poor quality', '0'],
      ['Sequence length', '80'],
      ['%GC', '50'],
    ],
  });
  expect(content.modules[8]!.blocks[0]).toEqual({
    kind: 'table',
    headers: ['Sequence', 'Count', 'Percentage', 'Possible Source'],
    rows: [['ACGTACGTACGTACGTACGTACGTACGTACGTACGTACGTACGTACGTAC', '100', '100.0', 'No Hit']],
  });
  expect(content.headerFilename).toBe('Mon 28 Sep 2026\nsample_trimmed.fq.gz');
  expect(content.footer).toBe('Produced by FastQC  (version 0.12.1)');
  const paragraph = source.replace(
    /(<h2 id="M9">[\s\S]*?<\/h2>)<table>[\s\S]*?<\/table>/,
    '$1<p>No overrepresented sequences</p>',
  );
  expect((await decode(paragraph)).modules[8]!.blocks).toEqual([
    { kind: 'text', text: 'No overrepresented sequences' },
  ]);
  expect(JSON.stringify(content).length).toBeLessThan(1024 * 1024);
  expect(
    await application.evaluate(({ BrowserWindow }) => BrowserWindow.getAllWindows().length),
  ).toBe(0);
  await writeFile(
    resolve('../docs/desktop-workspace/stages/saved-fastqc-report/qualified-content.json'),
    JSON.stringify(content),
  );
});
test('rejects active, external, malformed, unknown and oversized content without a partial rendition', async () => {
  test.setTimeout(60_000);
  const invalid = [
    source.replace('</head>', '<script>globalThis.reportExecuted=true</script></head>'),
    source.replace('<body>', '<body onload="alert(1)">'),
    source.replace('data:image/png;base64,', 'https://example.invalid/image?data='),
    source.replace('<div class="main">', '<div class="main"><p>Unknown additional content</p>'),
    source.replace('</table>', '</div>'),
    source.replace('version 0.12.1', 'version 0.12.2'),
    source.replace('</style>', 'body {color:red}</style>'),
    source.replace('<th>Measure</th>', '<th><b>Measure</b></th>'),
    source.replace('<td>100</td>', '<td>' + 'x'.repeat(65536) + '</td>'),
    source.replace('<body>', '<body>' + ' '.repeat(1024 * 1024)),
    source.replace(
      '<!DOCTYPE html>',
      '<!DOCTYPE html SYSTEM "https://example.invalid/external.dtd">',
    ),
    source.replace('iVBORw0KGgoAAAANSUhEUgAAAyAAAAJY', 'iVBORw0KGgoAAAANSUhEUgAAAyBAAAJY'),
  ];
  for (const [index, html] of invalid.entries()) {
    expect(html, 'mutation ' + index).not.toBe(source);
    await expect(decode(html), 'mutation ' + index).rejects.toThrow();
  }
  expect(
    await application.evaluate(({ BrowserWindow }) => BrowserWindow.getAllWindows().length),
  ).toBe(0);
});
test('uses a hidden sandbox without a preload or privileged bridge, and a closed reader refuses more jobs', async () => {
  const result = await application.evaluate(
    async ({ app, BrowserWindow }, { script, source }) => {
      const { FastqcReader } = globalThis as unknown as {
        FastqcReader: typeof import('../../src/main/fastqc').FastqcReader;
      };
      const reader = new FastqcReader(script);
      let preferences: unknown,
        visible = true;
      const handler = (_event: unknown, window: Electron.BrowserWindow) => {
        preferences = (
          window.webContents as Electron.WebContents & { getLastWebPreferences: () => unknown }
        ).getLastWebPreferences();
        visible = window.isVisible();
      };
      app.once('browser-window-created', handler);
      await reader.read(Buffer.from(source));
      reader.close();
      let stopped = false;
      try {
        await reader.read(Buffer.from(source));
      } catch {
        stopped = true;
      }
      return { preferences, visible, stopped, windows: BrowserWindow.getAllWindows().length };
    },
    { host, script: resolve('desktop/out/report-reader/reader.js'), source },
  );
  expect(result.visible).toBe(false);
  expect(result.stopped).toBe(true);
  expect(result.windows).toBe(0);
  expect(result.preferences).toMatchObject({
    sandbox: true,
    contextIsolation: true,
    nodeIntegration: false,
    webSecurity: true,
  });
  expect(result.preferences).not.toHaveProperty('preload');
});
test('close and the finite deadline dispose a stalled parser job', async () => {
  test.setTimeout(30_000);
  const stalled = join(base, 'stalled.js');
  await writeFile(stalled, 'var FastqcDecoder = {read:()=>new Promise(()=>{})};');
  for (const interrupt of [true, false]) {
    const result = await application.evaluate(
      async ({ app, BrowserWindow }, { script, interrupt }) => {
        const { FastqcReader } = globalThis as unknown as {
          FastqcReader: typeof import('../../src/main/fastqc').FastqcReader;
        };
        const reader = new FastqcReader(script);
        if (interrupt)
          app.once('browser-window-created', () => setTimeout(() => reader.close(), 50));
        const started = Date.now();
        let failed = false;
        try {
          await reader.read(Buffer.from('<unused>'));
        } catch {
          failed = true;
        } finally {
          reader.close();
        }
        return {
          failed,
          elapsed: Date.now() - started,
          windows: BrowserWindow.getAllWindows().length,
        };
      },
      { script: stalled, interrupt },
    );
    expect(result.failed).toBe(true);
    expect(result.windows).toBe(0);
    expect(result.elapsed).toBeLessThan(15_000);
    if (!interrupt) expect(result.elapsed).toBeGreaterThanOrEqual(9_500);
  }
});
