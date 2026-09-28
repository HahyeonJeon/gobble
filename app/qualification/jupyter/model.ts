/** Qualification-only live identity. Never serialized as the saved-file NotebookTarget v6. */
export type LiveSelection = {
  profile: 'jupyter-model-1';
  documentSessionId: string;
  modelRevision: number;
  cellId: string;
  start: number;
  end: number;
  text: string;
};
export type ComponentProbe = {
  state(): { revision: number; dirty: boolean; source: string; cellId: string; cells: number };
  selection(): LiveSelection | null;
  resolve(selection: LiveSelection): boolean;
  reveal(start: number, end: number): void;
  edit(source: string): void;
  reorder(): void;
  linkedView(): { sharesModel: boolean; source: string; dirty: boolean };
  dispose(): void;
};
