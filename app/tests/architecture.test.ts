import { readdir, readFile } from 'node:fs/promises';
import { dirname, resolve, relative } from 'node:path';
import ts from 'typescript';
import { describe, expect, it } from 'vitest';

const layers = {
  'contracts/src': ['@sinclair/typebox', '@sinclair/typebox/value'],
  'desktop/src/main': [
    'electron',
    '@gobble/contracts',
    '@gobble/contracts/pdf-decoder',
    '@gobble/contracts/pdf-geometry',
  ],
  'desktop/src/preload': ['electron', '@gobble/contracts'],
  'desktop/src/report-reader': ['@gobble/contracts'],
  'desktop/src/renderer': [
    'react',
    'react-dom',
    'react-dom/client',
    '@gobble/contracts',
    '@gobble/contracts/pdf-decoder',
    '@gobble/contracts/pdf-geometry',
    'plotly.js-basic-dist-min',
  ],
};

function violations(file: string, source: string, layer: string, external: string[]): string[] {
  const result: string[] = [];
  function check(specifier: string): void {
    if (specifier.startsWith('.')) {
      const target = relative(resolve(layer), resolve(dirname(file), specifier));
      if (target.startsWith('..')) result.push(specifier);
    } else if (
      !external.includes(specifier) &&
      !(layer === 'desktop/src/main' && specifier.startsWith('node:'))
    ) {
      result.push(specifier);
    }
  }
  function visit(node: ts.Node): void {
    if (
      (ts.isImportDeclaration(node) || ts.isExportDeclaration(node)) &&
      node.moduleSpecifier &&
      ts.isStringLiteral(node.moduleSpecifier)
    ) {
      check(node.moduleSpecifier.text);
    }
    if (
      ts.isImportTypeNode(node) &&
      ts.isLiteralTypeNode(node.argument) &&
      ts.isStringLiteral(node.argument.literal)
    ) {
      check(node.argument.literal.text);
    }
    if (
      ts.isCallExpression(node) &&
      (node.expression.kind === ts.SyntaxKind.ImportKeyword ||
        (ts.isIdentifier(node.expression) && node.expression.text === 'require'))
    ) {
      const argument = node.arguments[0];
      if (argument && ts.isStringLiteral(argument)) check(argument.text);
      else result.push('computed module loading');
    }
    ts.forEachChild(node, visit);
  }
  visit(ts.createSourceFile(file, source, ts.ScriptTarget.Latest, true));
  return result;
}

describe('process dependency boundaries', () => {
  it('isolates PDF decoder, fixed worker and narrow preload from Workspace and Node', async () => {
    const layer = 'desktop/src/pdf';
    expect((await readdir(layer)).sort()).toEqual(['decoder.ts', 'preload.ts', 'worker.ts']);
    for (const file of ['decoder.ts', 'preload.ts', 'worker.ts']) {
      let source = await readFile(resolve(layer, file), 'utf8');
      if (file === 'worker.ts') {
        // The sole computed import is an immutable packaged vendor URL, reviewed here.
        expect(source).toContain(
          "const workerModule = 'pdf-host://decoder/vendor/pdf.worker.mjs';",
        );
        expect(source.match(/await import\(workerModule\);/g)).toHaveLength(1);
        source = source.replace('await import(workerModule);', '');
      }
      expect(
        violations(
          resolve(layer, file),
          source,
          layer,
          file === 'preload.ts'
            ? ['electron', '@gobble/contracts/pdf-decoder']
            : ['pdfjs-dist', '@gobble/contracts/pdf-decoder', '@gobble/contracts/pdf-geometry'],
        ),
      ).toEqual([]);
    }
  });
  for (const [layer, external] of Object.entries(layers)) {
    it(`${layer} imports only its own files and allowed packages`, async () => {
      const files = (await readdir(layer, { recursive: true })).filter((file) =>
        /\.tsx?$/.test(file),
      );
      expect(files.length).toBeGreaterThan(0);
      for (const file of files) {
        const path = resolve(layer, file);
        expect(
          violations(
            path,
            await readFile(path, 'utf8'),
            layer,
            layer === 'desktop/src/main' &&
              ['notebook/targets.ts', 'notebook/worker-protocol.ts'].includes(file)
              ? [...external, '@sinclair/typebox', '@sinclair/typebox/value']
              : external,
          ),
          `${layer}/${file}`,
        ).toEqual([]);
      }
    });
  }

  it('detects relative escapes, re-exports, import types and computed loading', () => {
    expect(
      violations(
        resolve('desktop/src/renderer/probe.ts'),
        `
      import { readFile } from 'node:fs';
      export * from '../main/app-info';
      type Secret = import('electron').App;
      import(dynamicName);
      require('node:child_process');
    `,
        'desktop/src/renderer',
        layers['desktop/src/renderer'],
      ),
    ).toEqual([
      'node:fs',
      '../main/app-info',
      'electron',
      'computed module loading',
      'node:child_process',
    ]);
  });
});
