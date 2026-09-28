import { mkdir, readFile, writeFile } from 'node:fs/promises';
import * as contracts from '../src/index';

const definitions = Object.fromEntries(
  Object.entries(contracts)
    .filter(([name]) => name.endsWith('Schema'))
    .sort(([a], [b]) => a.localeCompare(b)),
);
const output = `${JSON.stringify(
  {
    $schema: 'http://json-schema.org/draft-07/schema#',
    $id: 'urn:gobble:app:contracts:v30',
    title: 'Gobble App contracts v30 — select a definition; no untyped root payload',
    not: {},
    definitions,
  },
  null,
  2,
)}\n`;
const directory = new URL('../schema/', import.meta.url);
const target = new URL('v30.json', directory);
if (process.argv.includes('--check')) {
  if ((await readFile(target, 'utf8')) !== output)
    throw new Error('Contract schema is stale. Run npm run schema:generate.');
} else {
  await mkdir(directory, { recursive: true });
  await writeFile(target, output);
}
