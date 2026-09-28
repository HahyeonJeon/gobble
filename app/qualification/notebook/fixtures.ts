import { deflateSync } from 'node:zlib';
export function png(width = 320, height = 160, noise = false): Buffer {
  function chunk(type: string, data: Buffer) {
    const b = Buffer.concat([Buffer.from(type), data]);
    let crc = 0xffffffff;
    for (const n of b) {
      crc ^= n;
      for (let i = 0; i < 8; i++) crc = (crc >>> 1) ^ (crc & 1 ? 0xedb88320 : 0);
    }
    const length = Buffer.alloc(4),
      check = Buffer.alloc(4);
    length.writeUInt32BE(data.length);
    check.writeUInt32BE((crc ^ 0xffffffff) >>> 0);
    return Buffer.concat([length, b, check]);
  }
  const header = Buffer.alloc(13);
  header.writeUInt32BE(width);
  header.writeUInt32BE(height, 4);
  header[8] = 8;
  header[9] = 6;
  let seed = 123456789;
  const random = () => {
    seed ^= seed << 13;
    seed ^= seed >>> 17;
    seed ^= seed << 5;
    return seed & 255;
  };
  const pixels = Buffer.alloc((width * 4 + 1) * height);
  for (let y = 0; y < height; y++)
    for (let x = 0; x < width; x++) {
      const at = y * (width * 4 + 1) + 1 + x * 4;
      pixels[at] = noise ? random() : x < width / 2 ? 32 : 198;
      pixels[at + 1] = noise ? random() : x < width / 2 ? 126 : 71;
      pixels[at + 2] = noise ? random() : x < width / 2 ? 132 : 58;
      pixels[at + 3] = 255;
    }
  return Buffer.concat([
    Buffer.from('89504e470d0a1a0a', 'hex'),
    chunk('IHDR', header),
    chunk('IDAT', deflateSync(pixels)),
    chunk('IEND', Buffer.alloc(0)),
  ]);
}
export function code(
  id = 'filter',
  source: string | string[] = 'keep = qc["mapped_pct"] >= 80\nqc.loc[keep]',
) {
  return {
    cell_type: 'code',
    id,
    metadata: {},
    source,
    execution_count: 7,
    outputs: [] as unknown[],
  };
}
export function notebook(cells: unknown[] = [code()], minor = 5): Record<string, unknown> {
  return {
    nbformat: 4,
    nbformat_minor: minor,
    metadata: { language_info: { name: 'python' } },
    cells,
  };
}
export function encode(value: unknown): Buffer {
  return Buffer.from(JSON.stringify(value));
}
export const FIXTURES = [
  'python',
  'r',
  'legacy',
  'reordered',
  'changed',
  'unsafe',
  'invalid',
  'many',
  'large',
  'jpeg',
  'long-text',
] as const;
export type FixtureName = (typeof FIXTURES)[number];
export function fixture(name: FixtureName, jpeg?: string): Buffer {
  if (name === 'long-text') return encode(notebook([code('filter', 'x'.repeat(1048576))]));
  if (name === 'invalid') return Buffer.from('{"cells":');
  if (name === 'many')
    return encode(
      notebook(
        Array.from({ length: 1000 }, (_, i) => code('cell-' + i, '# Cell ' + i + '\nvalue = ' + i)),
      ),
    );
  const filter = code();
  filter.outputs = [
    {
      output_type: 'execute_result',
      execution_count: 7,
      metadata: {},
      data: {
        'text/html':
          '<script>globalThis.notebookAttack=true</script><table><tr><td>94.2</td></tr></table>',
        'text/plain': '  sample  mapped_pct\n0 S01           94.2\n2 S03           91.7',
      },
    },
    {
      output_type: 'display_data',
      metadata: {},
      data: {
        [name === 'jpeg' ? 'image/jpeg' : 'image/png']:
          name === 'jpeg' ? (jpeg ?? '') : png().toString('base64'),
      },
    },
    {
      output_type: 'display_data',
      metadata: {},
      data: { 'application/vnd.jupyter.widget-view+json': { model_id: 'untrusted-widget' } },
    },
  ];
  const cells: unknown[] = [
    {
      cell_type: 'markdown',
      id: 'intro',
      metadata: {},
      source:
        '# Sample quality review\nDiscuss the saved code and results.\n![remote](https://example.invalid/track.png)\n<script>globalThis.notebookAttack=true</script>',
    },
    code('load', [
      '# Unicode 🧬 and CRLF\r\n',
      'import pandas as pd\r\n',
      'qc = pd.read_csv("quality.csv")',
    ]),
    filter,
  ];
  if (name === 'r') {
    filter.source = 'keep <- qc$mapped_pct >= 80\nqc[keep, ]';
    const nb = notebook(cells);
    nb.metadata = { language_info: { name: 'R' } };
    return encode(nb);
  }
  if (name === 'reordered') cells.reverse();
  if (name === 'changed') filter.source = 'keep = qc["mapped_pct"] >= 85\nqc.loc[keep]';
  if (name === 'legacy') {
    return encode(
      notebook(
        cells.map((c) => {
          const item = { ...(c as Record<string, unknown>) };
          delete item.id;
          return item;
        }),
        4,
      ),
    );
  }
  if (name === 'unsafe')
    filter.outputs.push({
      output_type: 'display_data',
      metadata: {},
      data: {
        'image/svg+xml': '<svg onload="globalThis.notebookAttack=true"></svg>',
        'application/javascript': 'globalThis.notebookAttack=true',
      },
    });
  const nb = notebook(cells);
  if (name === 'large')
    filter.outputs[1] = {
      output_type: 'display_data',
      metadata: {},
      data: { 'image/png': png(860, 860, true).toString('base64') },
    };
  return encode(nb);
}
