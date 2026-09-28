import { execFile } from 'node:child_process';
import { mkdtemp, readFile, writeFile, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { dirname, resolve } from 'node:path';
import { promisify } from 'node:util';
import ts from 'typescript';

// Generate only the fields Gobble sends, with empty environment arrays as
// deliberate restrictions. Include their actual upstream transitive type definitions.
const common = [
  'model',
  'modelProvider',
  'cwd',
  'approvalPolicy',
  'sandbox',
  'config',
  'developerInstructions',
];
const roots: Record<string, string[] | undefined> = {
  'v2/DynamicToolCallParams.ts': undefined,
  'v2/DynamicToolCallResponse.ts': undefined,
  'v2/ThreadStartParams.ts': [...common, 'environments', 'dynamicTools', 'ephemeral'],
  'v2/ThreadResumeParams.ts': [...common, 'threadId'],
  'v2/TurnStartParams.ts': [
    'threadId',
    'clientUserMessageId',
    'input',
    'model',
    'effort',
    'environments',
    'sandboxPolicy',
    'approvalPolicy',
  ],
};
async function main() {
  const arch = process.arch === 'arm64' ? 'aarch64' : 'x86_64';
  const target = arch + (process.platform === 'darwin' ? '-apple-darwin' : '-unknown-linux-musl');
  const executable =
    process.env.GOBBLE_CODEX_EXECUTABLE ??
    resolve(
      'node_modules/@openai',
      'codex-' + process.platform + '-' + process.arch,
      'vendor',
      target,
      'bin/codex',
    );
  const generated = await mkdtemp(resolve(tmpdir(), 'gobble-protocol-'));
  const output = resolve('desktop/src/main/codex/generated.ts');
  try {
    const env = { PATH: process.env.PATH, CODEX_HOME: generated };
    const run = promisify(execFile);
    if ((await run(executable, ['--version'], { env })).stdout.trim() !== 'codex-cli 0.153.4')
      throw new Error('Codex 0.153.4 is required.');
    await run(executable, ['app-server', 'generate-ts', '--experimental', '--out', generated], {
      env,
      maxBuffer: 1024 * 1024,
    });
    const definitions = new Map<string, string>();
    const visited = new Set<string>();
    const printer = ts.createPrinter({ removeComments: true });
    async function include(path: string, fields?: string[]): Promise<void> {
      if (visited.has(path)) return;
      visited.add(path);
      const source = ts.createSourceFile(
        path,
        await readFile(path, 'utf8'),
        ts.ScriptTarget.Latest,
        true,
      );
      const imports = new Map<string, string>();
      for (const statement of source.statements) {
        if (ts.isImportDeclaration(statement) && ts.isStringLiteral(statement.moduleSpecifier)) {
          const bindings = statement.importClause?.namedBindings;
          if (bindings && ts.isNamedImports(bindings))
            for (const element of bindings.elements)
              imports.set(
                element.name.text,
                resolve(dirname(path), statement.moduleSpecifier.text + '.ts'),
              );
        }
      }
      for (const statement of source.statements) {
        if (!ts.isTypeAliasDeclaration(statement)) continue;
        let type = statement.type;
        if (fields) {
          if (!ts.isTypeLiteralNode(type)) throw new Error('Expected parameter object.');
          const members = type.members
            .filter(
              (member) =>
                ts.isPropertySignature(member) && fields.includes(member.name.getText(source)),
            )
            .map((member) => {
              if (ts.isPropertySignature(member) && member.name.getText(source) === 'environments')
                return ts.factory.updatePropertySignature(
                  member,
                  member.modifiers,
                  member.name,
                  member.questionToken,
                  ts.factory.createTupleTypeNode([]),
                );
              return member;
            });
          if (members.length !== fields.length)
            throw new Error('A selected protocol field is missing.');
          type = ts.factory.updateTypeLiteralNode(type, ts.factory.createNodeArray(members));
        }
        const declaration = ts.factory.updateTypeAliasDeclaration(
          statement,
          statement.modifiers,
          statement.name,
          statement.typeParameters,
          type,
        );
        const text = printer.printNode(ts.EmitHint.Unspecified, declaration, source);
        if (definitions.has(statement.name.text)) throw new Error('Duplicate generated type name.');
        definitions.set(statement.name.text, text);
        for (const [name, dependency] of imports)
          if (new RegExp('\\b' + name + '\\b').test(text)) await include(dependency);
      }
    }
    for (const [path, fields] of Object.entries(roots))
      await include(resolve(generated, path), fields);
    const prettier = await import('prettier');
    const content = await prettier.format(
      '// Generated from official Codex 0.153.4. Do not edit.\n// Regenerate: npm run codex:schema. Empty environment lists are an app restriction.\n\n' +
        [...definitions.entries()]
          .sort(([a], [b]) => a.localeCompare(b))
          .map(([, text]) => text)
          .join('\n\n'),
      { ...(await prettier.resolveConfig(output)), parser: 'typescript' },
    );
    if (process.argv.includes('--check')) {
      if ((await readFile(output, 'utf8')) !== content)
        throw new Error('Codex protocol is out of date.');
    } else await writeFile(output, content);
    console.log('Codex protocol verified: ' + definitions.size + ' focused types.');
  } finally {
    await rm(generated, { recursive: true, force: true });
  }
}
main().catch((error: unknown) => {
  console.error(error instanceof Error ? error.message : 'Protocol generation failed.');
  process.exitCode = 1;
});
