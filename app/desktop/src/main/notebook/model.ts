/** Main-owned decoded bytes and private worker/capture values. Public View and target contracts live in contracts/notebook.ts. */
export const PROFILE = 'notebook-passive-1';
export const LIMITS = {
  bytes: 8 * 1024 * 1024,
  text: 1024 * 1024,
  cells: 1000,
  outputsPerCell: 100,
  outputs: 1000,
  depth: 32,
  nodes: 100_000,
  imagePixels: 4_000_000,
  imageBytes: 4 * 1024 * 1024,
  captureText: 64 * 1024,
  captureBytes: 768 * 1024,
  captureEdge: 1536,
  jobMs: 5000,
} as const;
export class NotebookProblem extends Error {
  constructor(
    readonly code: 'invalid' | 'unsupported' | 'limit' | 'stale' | 'cancelled' | 'timeout' | 'busy',
    message: string,
  ) {
    super(message);
  }
}
export type CellAddress = { kind: 'id'; id: string } | { kind: 'ordinal'; index: number };
export type TextPart = { kind: 'text'; text: string; digest: string; raw: string };
export type ImagePart = {
  kind: 'image';
  mime: 'image/png' | 'image/jpeg';
  base64: string;
  width: number;
  height: number;
  digest: string;
};
export type Output = {
  index: number;
  type: string;
  mime: string;
  alternatives: string[];
  notice: string;
  part: TextPart | ImagePart | { kind: 'unavailable'; reason: string };
};
export type Cell = {
  index: number;
  address: CellAddress;
  type: 'code' | 'markdown' | 'raw' | 'unknown';
  source: TextPart;
  outputs: Output[];
  notice: string;
};
export type Snapshot = {
  profile: typeof PROFILE;
  revision: string;
  bytes: number;
  minor: number;
  language: string;
  cells: Cell[];
  textBytes: number;
};
export type PartAddress = {
  cell: CellAddress;
  part: { kind: 'source' } | { kind: 'output'; index: number; mime: string; digest: string };
};
export type TextSelector = { kind: 'text'; start: number; end: number };
export type Rect = { x: number; y: number; width: number; height: number };
export type Selector = TextSelector | { kind: 'image'; rect: Rect };
export type Target = PartAddress & {
  revision: string;
  profile: typeof PROFILE;
  selector: Selector;
};
export type Capture =
  | {
      target: Target;
      label: string;
      representation: 'text/plain';
      text: string;
      rawQuote: string;
      digest: string;
    }
  | {
      target: Target;
      label: string;
      representation: 'image/png';
      base64: string;
      width: number;
      height: number;
      digest: string;
    };
export type ViewSnapshot = Omit<Snapshot, 'cells'> & {
  cells: Array<
    Omit<Cell, 'source' | 'outputs'> & {
      source: Omit<TextPart, 'raw'>;
      outputs: Array<
        Omit<Output, 'part'> & {
          part:
            | Omit<TextPart, 'raw'>
            | Omit<ImagePart, 'base64'>
            | { kind: 'unavailable'; reason: string };
        }
      >;
    }
  >;
};
