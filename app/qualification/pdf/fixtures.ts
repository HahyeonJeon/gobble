import { deflateSync } from 'node:zlib';
import { LIMITS } from './protocol';

class PdfWriter {
  private objects: Buffer[] = [];
  add(value: string | Buffer): number {
    this.objects.push(typeof value === 'string' ? Buffer.from(value, 'binary') : value);
    return this.objects.length;
  }
  replace(id: number, value: string) {
    this.objects[id - 1] = Buffer.from(value, 'binary');
  }
  stream(data: Buffer, dictionary = ''): number {
    const compressed = deflateSync(data);
    return this.add(
      Buffer.concat([
        Buffer.from(
          '<< ' +
            dictionary +
            ' /Length ' +
            compressed.length +
            ' /Filter /FlateDecode >>\nstream\n',
        ),
        compressed,
        Buffer.from('\nendstream'),
      ]),
    );
  }
  finish(root: number): Buffer {
    const chunks = [Buffer.from('%PDF-1.7\n%\xe2\xe3\xcf\xd3\n', 'binary')];
    const offsets = [0];
    let size = chunks[0]!.length;
    this.objects.forEach((object, index) => {
      offsets.push(size);
      const chunk = Buffer.concat([
        Buffer.from(index + 1 + ' 0 obj\n'),
        object,
        Buffer.from('\nendobj\n'),
      ]);
      chunks.push(chunk);
      size += chunk.length;
    });
    const xref = [
      'xref',
      '0 ' + (this.objects.length + 1),
      '0000000000 65535 f ',
      ...offsets.slice(1).map((n) => String(n).padStart(10, '0') + ' 00000 n '),
      'trailer',
      '<< /Size ' + (this.objects.length + 1) + ' /Root ' + root + ' 0 R >>',
      'startxref',
      String(size),
      '%%EOF',
    ];
    chunks.push(Buffer.from(xref.join('\n') + '\n'));
    return Buffer.concat(chunks);
  }
}
const text = (value: string, x: number, y: number, size = 14) =>
  'BT /F1 ' +
  size +
  ' Tf 1 0 0 1 ' +
  x +
  ' ' +
  y +
  ' Tm (' +
  value.replace(/[()\\]/g, '\\$&') +
  ') Tj ET\n';
type Options = {
  pages?: number;
  rotate?: number;
  unit?: number;
  scan?: boolean;
  dense?: boolean;
  active?: boolean;
  unicode?: boolean;
  noise?: boolean;
  imageSize?: number;
  corruptImage?: boolean;
};
export function makePdf(options: Options = {}): Buffer {
  const w = new PdfWriter();
  const catalog = w.add(''),
    pages = w.add('');
  const font = w.add('<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>');
  let extraFont = 0;
  if (options.unicode) {
    const cmap = w.stream(
      Buffer.from(
        '/CIDInit /ProcSet findresource begin 12 dict begin begincmap /CIDSystemInfo << /Registry (Adobe) /Ordering (UCS) /Supplement 0 >> def /CMapName /FixtureUnicode def /CMapType 2 def 1 begincodespacerange <00> <FF> endcodespacerange 5 beginbfchar <41> <00660069> <42> <D83DDE00> <43> <00650301> <44> <05D005D1> <45> <0041> endbfchar endcmap CMapName currentdict /CMap defineresource pop end end',
      ),
    );
    extraFont = w.add(
      '<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica /ToUnicode ' + cmap + ' 0 R >>',
    );
  }
  let image = 0;
  if (options.corruptImage) {
    image = w.add(
      '<< /Type /XObject /Subtype /Image /Width 128 /Height 128 /ColorSpace /DeviceRGB /BitsPerComponent 8 /Filter /DCTDecode /Length 8 >>\nstream\nnot-jpeg\nendstream',
    );
  } else if (options.scan || options.noise) {
    const side = options.imageSize ?? (options.noise ? 1200 : 128);
    const pixels = Buffer.alloc(side * side * 3);
    let seed = 12345;
    for (let i = 0; i < pixels.length; i++) {
      seed = (Math.imul(seed, 1664525) + 1013904223) >>> 0;
      pixels[i] = options.noise ? seed >>> 24 : Math.floor(i / 3) % side < side / 2 ? 90 : 210;
    }
    image = w.stream(
      pixels,
      '/Type /XObject /Subtype /Image /Width ' +
        side +
        ' /Height ' +
        side +
        ' /ColorSpace /DeviceRGB /BitsPerComponent 8',
    );
  }
  const ids: number[] = [];
  for (let i = 0; i < (options.pages ?? 1); i++) {
    let commands =
      options.scan || options.noise
        ? 'q 420 0 0 360 70 210 cm /Im1 Do Q\n'
        : '0.12 0.24 0.20 rg\n' +
          text('PDF shared reading', 70, 720, 24) +
          text('Synthetic qualification report - page ' + (i + 1), 70, 687, 11) +
          text('Same filtered cohort.', 70, 620) +
          text('Same filtered cohort.', 325, 620) +
          text('Left column: samples and filters.', 70, 581, 11) +
          text('Right column: comparison.', 325, 581, 11) +
          text('A marked region refers to the source page.', 70, 520, 13) +
          'q 0 1 -1 0 480 270 cm ' +
          text('Rotated text', 0, 0, 12) +
          ' Q\n' +
          '0.85 0.2 0.15 rg 100 120 120 60 re f\n' +
          '0.12 0.24 0.20 rg ' +
          text('Known red rectangle: [100, 120, 220, 180]', 70, 85, 10);
    if (options.unicode) commands += 'BT /F2 20 Tf 1 0 0 1 70 450 Tm <4142434445> Tj ET\n';
    if (options.dense)
      commands += Array.from(
        { length: 120_000 },
        (_, n) => (n % 500) + 30 + ' ' + ((n % 700) + 50) + ' 1 1 re S\n',
      ).join('');
    const content = w.stream(Buffer.from(commands));
    let annotations = '';
    if (options.active) {
      const annotation = w.add(
        '<< /Type /Annot /Subtype /Link /Rect [70 510 400 540] /A << /S /URI /URI (https://example.invalid/pdf-fixture) >> >>',
      );
      annotations = ' /Annots [' + annotation + ' 0 R]';
    }
    ids.push(
      w.add(
        '<< /Type /Page /Parent ' +
          pages +
          ' 0 R /MediaBox [0 0 612 842] /CropBox [20 40 580 800] /Rotate ' +
          (options.rotate ?? 0) +
          ' /UserUnit ' +
          (options.unit ?? 1) +
          ' /Resources << /Font << /F1 ' +
          font +
          ' 0 R ' +
          (extraFont ? '/F2 ' + extraFont + ' 0 R' : '') +
          ' >> ' +
          (image ? '/XObject << /Im1 ' + image + ' 0 R >>' : '') +
          ' >> /Contents ' +
          content +
          ' 0 R' +
          annotations +
          ' >>',
      ),
    );
  }
  w.replace(
    pages,
    '<< /Type /Pages /Kids [' +
      ids.map((id) => id + ' 0 R').join(' ') +
      '] /Count ' +
      ids.length +
      ' >>',
  );
  w.replace(
    catalog,
    '<< /Type /Catalog /Pages ' +
      pages +
      ' 0 R' +
      (options.active
        ? ' /OpenAction << /S /JavaScript /JS (fetch\\(\\"https://example.invalid/blocked\\"\\)) >>'
        : '') +
      ' >>',
  );
  return w.finish(catalog);
}
export const FIXTURE_NAMES = [
  'report',
  'rotated',
  'user-unit',
  'scanned',
  'unicode',
  'dense',
  'active',
  'many-pages',
  'too-many-pages',
  'noise',
  'oversized-image',
  'corrupt-image',
  'malformed',
  'encrypted',
  'limit-minus',
  'limit-exact',
  'limit-plus',
] as const;
export type FixtureName = (typeof FIXTURE_NAMES)[number];
export function fixture(name: FixtureName): Buffer {
  switch (name) {
    case 'report':
      return makePdf({ pages: 4 });
    case 'rotated':
      return makePdf({ rotate: 90 });
    case 'user-unit':
      return makePdf({ unit: 2 });
    case 'scanned':
      return makePdf({ scan: true });
    case 'unicode':
      return makePdf({ unicode: true });
    case 'dense':
      return makePdf({ dense: true });
    case 'active':
      return makePdf({ active: true });
    case 'many-pages':
      return makePdf({ pages: 200 });
    case 'too-many-pages':
      return makePdf({ pages: 201 });
    case 'noise':
      return makePdf({ noise: true });
    case 'oversized-image':
      return makePdf({ scan: true, imageSize: 4600 });
    case 'corrupt-image':
      return makePdf({ scan: true, corruptImage: true });
    case 'malformed':
      return Buffer.from('%PDF-1.7\n1 0 obj << broken');
    case 'encrypted':
      throw new Error('Encrypted fixture is supplied by the runner.');
    default: {
      const bytes = makePdf();
      const length =
        LIMITS.fileBytes + (name === 'limit-minus' ? -1 : name === 'limit-plus' ? 1 : 0);
      return Buffer.concat([bytes, Buffer.alloc(length - bytes.length, 32)]);
    }
  }
}
