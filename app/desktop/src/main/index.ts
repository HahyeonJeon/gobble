import { FastqcReader } from './fastqc';
import { ReportEvidence } from './evidence/report';
import { PipelineReviewHost } from './pipeline-review/host';
import { NotebookViews } from './workspace/notebook-views';
import { NotebookHost } from './notebook/host';
import { materializeNotebook } from './evidence/notebook-evidence';
import { installRendererRecovery } from './workspace/recovery';
import { PdfViews } from './workspace/pdf-views';
import { PdfHost } from './pdf/host';
import { materializePdf, capturePdfRaster } from './evidence/pdf-evidence';
import { app, BrowserWindow, dialog, protocol, screen, session, shell } from 'electron';
import { dirname, isAbsolute, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { registerAppInfo } from './app-info';
import { readShellAsset } from './security/content';
import { rendererURL } from './security/origin';
import { restrictContents, restrictSession } from './security/permissions';
import { ProjectServiceClient } from './service/client';
import { ProjectService } from './service/project-service';
import { registerProjectService } from './service/ipc';
import { WorkspaceController } from './workspace/controller';
import { registerWorkspace } from './workspace/ipc';
import { ServiceResources } from './workspace/service';
import { WorkspaceStorage } from './workspace/storage';
import { clampBounds } from './workspace/bounds';
import { installWorkspaceMenu } from './workspace/window';
import type { WindowState } from '@gobble/contracts';
import { SharedContextHost } from './shared-context/host';
import { renderObservationImage } from './shared-context/image';
import { CollaborationHost } from './collaboration/host';
import { registerCollaboration } from './collaboration/ipc';
import { QuestionService } from './questions/service';
import { EvidenceService } from './evidence/service';
import { EvidenceStorage } from './evidence/storage';
import { EvidenceCapture } from './evidence/capture';

const directory = dirname(fileURLToPath(import.meta.url));
app.setName('Gobble');
app.commandLine.appendSwitch('lang', 'en-US');
// Explicit profiles keep local development/test state separate from user projects.
const profile = app.commandLine.getSwitchValue('gobble-profile');
if (profile) {
  if (!isAbsolute(profile)) throw new Error('Gobble profile must be an absolute path.');
  app.setPath('userData', profile);
}
protocol.registerSchemesAsPrivileged([
  {
    scheme: 'pdf-host',
    privileges: { standard: true, secure: true, supportFetchAPI: true, corsEnabled: true },
  },
  { scheme: 'app', privileges: { standard: true, secure: true, supportFetchAPI: true } },
]);

let window: BrowserWindow | undefined;
let quitting = false;
let serviceStopped = false;
let projectService: ProjectServiceClient | undefined;
let workspaceController: WorkspaceController | undefined;
let collaborationHost: CollaborationHost | undefined;
let unregisterCollaboration: (() => void) | undefined;
let savedBounds: WindowState['bounds'] = null;
let quitInProgress = false;
const target = rendererURL(app.isPackaged, process.env.ELECTRON_RENDERER_URL);

async function discardUnsaved(action: 'close' | 'quit'): Promise<boolean> {
  const answer = await dialog.showMessageBox({
    type: 'warning',
    title: 'Changes could not be saved',
    message: 'Some workspace changes could not be saved.',
    detail: `Keep the app open to preserve your draft, or ${action} without those changes.`,
    buttons: ['Keep Open', action === 'close' ? 'Close Without Saving' : 'Quit Without Saving'],
    defaultId: 0,
    cancelId: 0,
  });
  return answer.response === 1;
}

function openWorkspace(): void {
  if (quitting) return;
  if (window) {
    if (window.isMinimized()) window.restore();
    window.show();
    window.focus();
    return;
  }
  const created = new BrowserWindow({
    title: 'Gobble',
    ...clampBounds(
      savedBounds,
      savedBounds
        ? screen.getDisplayMatching(savedBounds).workArea
        : screen.getPrimaryDisplay().workArea,
    ),
    minWidth: 720,
    minHeight: 520,
    backgroundColor: '#f5f6f2',
    show: false,
    webPreferences: {
      preload: resolve(directory, '../preload/index.cjs'),
      nodeIntegration: false,
      nodeIntegrationInWorker: false,
      nodeIntegrationInSubFrames: false,
      contextIsolation: true,
      sandbox: true,
      webSecurity: true,
      allowRunningInsecureContent: false,
      webviewTag: false,
      navigateOnDragDrop: false,
    },
  });
  window = created;
  installRendererRecovery(
    created,
    target,
    () => workspaceController?.disconnect(),
    () => quitting,
  );
  const persistBounds = () => {
    if (created.isDestroyed() || quitting) return;
    savedBounds = created.getNormalBounds();
    void workspaceController?.saveBounds(savedBounds).catch(() => {});
  };
  const invalidate = () => workspaceController?.invalidatePresentation();
  created.on('blur', invalidate);
  created.on('hide', invalidate);
  created.on('minimize', invalidate);
  created.on('focus', invalidate);
  created.on('show', invalidate);
  created.on('restore', invalidate);
  created.on('resized', persistBounds);
  created.on('moved', persistBounds);
  created.once('ready-to-show', () => {
    if (!created.isDestroyed()) created.show();
  });
  created.on('closed', () => {
    workspaceController?.disconnect();
    if (window === created) window = undefined;
  });
  let closeReady = false;
  let closeInProgress = false;
  created.on('close', (event) => {
    if (quitting || closeReady || !workspaceController) return;
    event.preventDefault();
    if (closeInProgress) return;
    closeInProgress = true;
    persistBounds();
    void workspaceController
      .flush()
      .then(async () => {
        if (workspaceController?.unsaved && !(await discardUnsaved('close'))) return;
        closeReady = true;
        if (!created.isDestroyed()) created.close();
      })
      .finally(() => {
        closeInProgress = false;
      });
  });
  void created.loadURL(target).catch(() => {
    if (!quitting) {
      dialog.showErrorBox(
        'Gobble could not open',
        'The app interface could not be loaded. Rebuild the desktop app and try again.',
      );
      app.quit();
    }
  });
}

if (!app.requestSingleInstanceLock()) {
  app.quit();
} else {
  app.on('web-contents-created', (_event, contents) => restrictContents(contents));
  app.on('second-instance', () => {
    if (app.isReady()) openWorkspace();
  });
  app.on('activate', () => {
    if (app.isReady()) openWorkspace();
  });
  app.on('before-quit', (event) => {
    if (projectService && !serviceStopped) {
      event.preventDefault();
      if (quitInProgress) return;
      quitInProgress = true;
      void (async () => {
        await workspaceController?.flush();
        if (window && !window.isDestroyed()) {
          await workspaceController?.saveBounds(window.getNormalBounds());
        }
        if (workspaceController?.unsaved && !(await discardUnsaved('quit'))) {
          quitInProgress = false;
          return;
        }
        const alreadyUnsaved = workspaceController?.unsaved;
        await collaborationHost?.stop().catch(() => {
          if (workspaceController) workspaceController.unsaved = true;
        });
        await workspaceController?.flush();
        if (!alreadyUnsaved && workspaceController?.unsaved && !(await discardUnsaved('quit'))) {
          quitInProgress = false;
          return;
        }
        quitting = true;
        unregisterCollaboration?.();
        await workspaceController?.stop();
        await projectService?.stop();
        serviceStopped = true;
        app.quit();
      })().catch(() => {
        quitInProgress = false;
      });
    } else {
      quitting = true;
    }
  });
  app.on('window-all-closed', () => {
    if (process.platform !== 'darwin') app.quit();
  });
  void app
    .whenReady()
    .then(async () => {
      restrictSession(session.defaultSession);
      protocol.handle('app', (request) =>
        request.method === 'GET'
          ? readShellAsset(resolve(directory, '../renderer'), request.url)
          : new Response(null, { status: 405 }),
      );
      registerAppInfo(() => window?.webContents, target);
      projectService = new ProjectServiceClient(
        app.isPackaged
          ? resolve(process.resourcesPath, 'service/gobble-service')
          : resolve(directory, '../service/gobble-service'),
        app.getPath('userData'),
      );
      const service = new ProjectService(projectService);
      let pipelineReview: PipelineReviewHost;
      registerProjectService(
        service,
        () => window,
        target,
        () => pipelineReview,
      );
      const evidenceStorage = new EvidenceStorage(
        resolve(app.getPath('userData'), 'workspace/evidence'),
      );
      const pdfViews = new PdfViews(
        () => new PdfHost(resolve(directory, '../pdf'), resolve(directory, '../pdf/preload.cjs')),
      );
      const notebookViews = new NotebookViews(
        () => new NotebookHost(resolve(directory, '../notebook/worker.cjs')),
      );
      const reportReader = new FastqcReader(resolve(directory, '../report-reader/reader.js'));
      app.once('before-quit', () => reportReader.close());
      const reports = new ReportEvidence(evidenceStorage, (source) => reportReader.read(source));
      workspaceController = new WorkspaceController(
        new WorkspaceStorage(resolve(app.getPath('userData'), 'workspace')),
        new ServiceResources(service, pdfViews, notebookViews, reports),
        () =>
          !!window &&
          !window.isDestroyed() &&
          window.isVisible() &&
          !window.isMinimized() &&
          window.isFocused(),
        new EvidenceCapture(
          evidenceStorage,
          (attachment, data) => materializePdf(pdfViews, attachment, data),
          (attachment, data) => materializeNotebook(notebookViews, attachment, data),
        ),
      );
      savedBounds = (await workspaceController.initialize()).bounds;
      registerWorkspace(workspaceController, () => window, target);
      const nativeTarget =
        process.platform === 'darwin'
          ? (process.arch === 'arm64' ? 'aarch64' : 'x86_64') + '-apple-darwin'
          : (process.arch === 'arm64' ? 'aarch64' : 'x86_64') + '-unknown-linux-musl';
      pipelineReview = new PipelineReviewHost(
        workspaceController,
        service,
        async (projectId, manifest) => {
          await evidenceStorage.read(projectId, manifest);
        },
      );
      collaborationHost = new CollaborationHost(
        workspaceController,
        app.isPackaged
          ? resolve(process.resourcesPath, 'codex/bin/codex')
          : (process.env.GOBBLE_CODEX_EXECUTABLE ??
              resolve(
                directory,
                '../../../node_modules/@openai',
                'codex-' + process.platform + '-' + process.arch,
                'vendor',
                nativeTarget,
                'bin/codex',
              )),
        app.getPath('userData'),
        (url) => shell.openExternal(url),
        (supportsImages, questions) =>
          new SharedContextHost(
            workspaceController!,
            new ServiceResources(service, pdfViews, notebookViews, reports),
            service,
            renderObservationImage,
            supportsImages,
            questions,
            capturePdfRaster,
            (attachment, data) => materializeNotebook(notebookViews, attachment, data),
            pipelineReview,
          ),
        (account) =>
          new EvidenceService(
            workspaceController!,
            new ServiceResources(service, pdfViews, notebookViews, reports),
            evidenceStorage,
            renderObservationImage,
            account,
          ),
        (account) =>
          new QuestionService(
            workspaceController!,
            new ServiceResources(service, pdfViews, notebookViews),
            evidenceStorage,
            account,
            capturePdfRaster,
            (attachment, data) => materializeNotebook(notebookViews, attachment, data),
          ),
        pipelineReview,
      );
      unregisterCollaboration = registerCollaboration(
        collaborationHost,
        workspaceController,
        () => window,
        target,
      );
      installWorkspaceMenu(() => window);
      openWorkspace();
    })
    .catch(() => {
      dialog.showErrorBox('Gobble could not start', 'The desktop host could not initialize.');
      app.quit();
    });
}
