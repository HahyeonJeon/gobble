import { Notebook, NotebookModel, StaticNotebook } from '@jupyterlab/notebook';
import {
  CodeMirrorEditorFactory,
  CodeMirrorMimeTypeService,
  EditorExtensionRegistry,
  EditorLanguageRegistry,
  EditorThemeRegistry,
  ybinding,
} from '@jupyterlab/codemirror';
import { RenderMimeRegistry, standardRendererFactories } from '@jupyterlab/rendermime';
import { Widget } from '@lumino/widgets';
import type { IYText } from '@jupyter/ydoc';
import type { ComponentProbe } from './model';
import './style.css';

declare global {
  interface Window {
    componentProbe: ComponentProbe;
  }
}
export function mountComponent(): void {
  const host = document.getElementById('notebook');
  if (!host) throw new Error('Missing notebook host');
  const languages = new EditorLanguageRegistry();
  EditorLanguageRegistry.getDefaultLanguages()
    .filter((l) => l.name.toLowerCase() === 'python')
    .forEach((l) => languages.addLanguage(l));
  const themes = new EditorThemeRegistry();
  EditorThemeRegistry.getDefaultThemes().forEach((t) => themes.addTheme(t));
  const extensions = new EditorExtensionRegistry();
  EditorExtensionRegistry.getDefaultExtensions({ themes }).forEach((e) =>
    extensions.addExtension(e),
  );
  extensions.addExtension({
    name: 'shared-model-binding',
    factory: (options) => {
      const shared = options.model.sharedModel as IYText;
      return EditorExtensionRegistry.createImmutableExtension(
        ybinding({
          ytext: shared.ysource,
          ...(shared.undoManager ? { undoManager: shared.undoManager } : {}),
        }),
      );
    },
  });
  const factory = new CodeMirrorEditorFactory({ languages, extensions });
  const model = new NotebookModel();
  model.fromJSON({
    nbformat: 4,
    nbformat_minor: 5,
    metadata: { language_info: { name: 'python' } },
    cells: [
      {
        id: 'cell-counts',
        cell_type: 'code',
        metadata: {},
        source: 'label = "🧬 한국어"\ncounts = [12, 24, 18]\nprint(sum(counts))',
        outputs: [{ output_type: 'stream', name: 'stdout', text: '54\n' }],
        execution_count: 1,
      },
      {
        id: 'cell-filter',
        cell_type: 'code',
        metadata: {},
        source: 'threshold = 20\nselected = [value for value in counts if value > threshold]',
        outputs: [],
        execution_count: null,
      },
    ],
  });
  model.dirty = false;
  let revision = 1;
  const documentSessionId = crypto.randomUUID();
  const notebook = new Notebook({
    rendermime: new RenderMimeRegistry({ initialFactories: standardRendererFactories }),
    contentFactory: new Notebook.ContentFactory({ editorFactory: factory.newInlineEditor }),
    mimeTypeService: new CodeMirrorMimeTypeService(languages),
    notebookConfig: { ...StaticNotebook.defaultNotebookConfig, windowingMode: 'none' },
  });
  notebook.model = model;
  notebook.addClass('qualification-notebook');
  Widget.attach(notebook, host);
  const refresh = () => {
    revision += 1;
    document.getElementById('document-state')!.textContent =
      `Unsaved · document revision ${revision}`;
  };
  model.contentChanged.connect(refresh);
  const resize = () => notebook.update();
  window.addEventListener('resize', resize);
  window.componentProbe = {
    state() {
      const cell = notebook.activeCell?.model;
      return {
        revision,
        dirty: model.dirty,
        source: cell?.sharedModel.getSource() ?? '',
        cellId: cell?.id ?? '',
        cells: model.cells.length,
      };
    },
    selection() {
      const cell = notebook.activeCell;
      if (!cell?.editor) return null;
      const range = cell.editor.getSelection();
      const offsets = [
        cell.editor.getOffsetAt(range.start),
        cell.editor.getOffsetAt(range.end),
      ].sort((a, b) => a - b);
      const [start, end] = offsets;
      if (start === undefined || end === undefined || start === end) return null;
      return {
        profile: 'jupyter-model-1',
        documentSessionId,
        modelRevision: revision,
        cellId: cell.model.id,
        start,
        end,
        text: cell.model.sharedModel.getSource().slice(start, end),
      };
    },
    resolve(selection) {
      if (
        selection.profile !== 'jupyter-model-1' ||
        selection.documentSessionId !== documentSessionId ||
        selection.modelRevision !== revision
      )
        return false;
      const cell = [...model.cells].find((c) => c.id === selection.cellId);
      return (
        !!cell &&
        selection.start >= 0 &&
        selection.end > selection.start &&
        selection.end <= cell.sharedModel.getSource().length &&
        cell.sharedModel.getSource().slice(selection.start, selection.end) === selection.text
      );
    },
    reveal(start, end) {
      const editor = notebook.activeCell?.editor;
      if (!editor) throw new Error('Editor unavailable');
      const a = editor.getPositionAt(start),
        b = editor.getPositionAt(end);
      if (!a || !b) throw new Error('Invalid offsets');
      notebook.mode = 'edit';
      editor.focus();
      editor.setSelection({ start: a, end: b });
    },
    edit(source) {
      notebook.activeCell?.model.sharedModel.setSource(source);
    },
    reorder() {
      model.sharedModel.moveCell(0, 1);
      notebook.activeCellIndex = 1;
    },
    linkedView() {
      const mirror = new Notebook({
        rendermime: new RenderMimeRegistry({ initialFactories: standardRendererFactories }),
        contentFactory: new Notebook.ContentFactory({ editorFactory: factory.newInlineEditor }),
        mimeTypeService: new CodeMirrorMimeTypeService(languages),
        notebookConfig: { ...StaticNotebook.defaultNotebookConfig, windowingMode: 'none' },
      });
      mirror.model = model;
      const result = {
        sharesModel: mirror.model === notebook.model,
        source: mirror.model.cells.get(1)?.sharedModel.getSource() ?? '',
        dirty: mirror.model.dirty,
      };
      mirror.dispose();
      if (model.isDisposed) throw new Error('Closing a linked view disposed the shared document');
      return result;
    },
    dispose() {
      window.removeEventListener('resize', resize);
      notebook.dispose();
      model.dispose();
    },
  };
  document.getElementById('share')!.addEventListener('click', () => {
    const selection = window.componentProbe.selection();
    if (!selection) return;
    document.getElementById('reference')!.textContent =
      `${selection.cellId} · revision ${selection.modelRevision}\n“${selection.text}”\nUTF-16 ${selection.start}–${selection.end}`;
  });
  notebook.activeCellIndex = 0;
  resize();
}
mountComponent();
