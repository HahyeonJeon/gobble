import { NotebookViewportSchema } from './notebook-viewport';
import { NotebookTargetSchema, NotebookImageSchema, NotebookNavigationSchema } from './notebook';
import { RectSchema } from './pdf-decoder';
import { PdfNavigationSchema } from './pdf';
import { ObservedReferenceViewSchema } from './observed-reference';
import { Type, type Static } from '@sinclair/typebox';
import {
  DecisionIdSchema,
  AgentIdSchema,
  closed,
  CounterSchema,
  ProjectIdSchema,
  RendererSessionIdSchema,
  RequestIdSchema,
  RevisionSchema,
  RunRefSchema,
  SurfaceIdSchema,
} from './identity';
import { SharedReferenceIdSchema } from './shared-context';
import { EvidenceRefSchema } from './context';
import { AttachmentIdSchema } from './evidence';
import { ResourceRefSchema } from './resource';
import { SurfaceDataSchema } from './surface-data';
export { SurfaceDataSchema, type SurfaceData } from './surface-data';
import { ProjectInfoSchema } from './project';
import { serviceResult } from './result';
import { RenderAcknowledgmentSchema } from './surface';
import { ReferenceViewSchema } from './reference-presentation';
import { PaneIdSchema } from './workspace';
import { WorkspaceDocumentSchema } from './workspace-document';
import { TabularActionSchema } from './tabular-actions';
import { RunNavigationIntentSchema } from './run-navigation';

export const WorkspaceActionSchema = Type.Union([
  Type.Object(
    {
      kind: Type.Literal('reportNavigate'),
      surfaceId: SurfaceIdSchema,
      moduleId: Type.String({ pattern: '^M[0-9]+$', maxLength: 16 }),
      acknowledgment: RenderAcknowledgmentSchema,
    },
    closed,
  ),
  Type.Object(
    {
      kind: Type.Literal('openReport'),
      runRef: RunRefSchema,
      instance: Type.String({ minLength: 1, maxLength: 256 }),
      attempt: Type.Integer({ minimum: 1, maximum: Number.MAX_SAFE_INTEGER }),
    },
    closed,
  ),
  Type.Object(
    {
      kind: Type.Literal('notebookNavigate'),
      surfaceId: SurfaceIdSchema,
      navigation: NotebookNavigationSchema,
      acknowledgment: RenderAcknowledgmentSchema,
    },
    closed,
  ),
  Type.Object(
    {
      kind: Type.Literal('pdfNavigate'),
      surfaceId: SurfaceIdSchema,
      navigation: PdfNavigationSchema,
      acknowledgment: RenderAcknowledgmentSchema,
    },
    closed,
  ),
  RunNavigationIntentSchema,
  Type.Object(
    { kind: Type.Literal('returnReferenceView'), referenceRequestId: RequestIdSchema },
    closed,
  ),
  TabularActionSchema,
  Type.Object(
    { kind: Type.Literal('duplicateView'), surfaceId: SurfaceIdSchema, pane: PaneIdSchema },
    closed,
  ),
  Type.Object({ kind: Type.Literal('reply'), questionId: DecisionIdSchema }, closed),
  Type.Object({ kind: Type.Literal('cancelReply') }, closed),
  Type.Object({ kind: Type.Literal('dismissQuestion'), questionId: DecisionIdSchema }, closed),
  Type.Object(
    {
      kind: Type.Literal('attach'),
      surfaceId: SurfaceIdSchema,
      evidence: EvidenceRefSchema,
      acknowledgment: Type.Optional(RenderAcknowledgmentSchema),
    },
    closed,
  ),
  Type.Object({ kind: Type.Literal('detach'), attachmentId: AttachmentIdSchema }, closed),
  Type.Object(
    {
      kind: Type.Literal('share'),
      surfaceId: SurfaceIdSchema,
      evidence: EvidenceRefSchema,
      note: Type.String({ maxLength: 500 }),
    },
    closed,
  ),
  Type.Object({ kind: Type.Literal('retract'), referenceId: SharedReferenceIdSchema }, closed),
  Type.Object({ kind: Type.Literal('reveal'), referenceId: SharedReferenceIdSchema }, closed),
  Type.Object(
    { kind: Type.Literal('currentDependency'), referenceId: SharedReferenceIdSchema },
    closed,
  ),
  Type.Object({ kind: Type.Literal('currentTask'), referenceId: SharedReferenceIdSchema }, closed),
  Type.Object(
    {
      kind: Type.Literal('open'),
      resource: ResourceRefSchema,
      pane: PaneIdSchema,
      duplicate: Type.Boolean(),
    },
    closed,
  ),
  Type.Object({ kind: Type.Literal('activate'), surfaceId: SurfaceIdSchema }, closed),
  Type.Object({ kind: Type.Literal('focusPane'), pane: PaneIdSchema }, closed),
  Type.Object({ kind: Type.Literal('close'), surfaceId: SurfaceIdSchema }, closed),
  Type.Object(
    { kind: Type.Literal('pin'), surfaceId: SurfaceIdSchema, pinned: Type.Boolean() },
    closed,
  ),
  Type.Object(
    { kind: Type.Literal('move'), surfaceId: SurfaceIdSchema, pane: PaneIdSchema },
    closed,
  ),
  Type.Object(
    {
      kind: Type.Literal('arrange'),
      layout: Type.Union([Type.Literal('single'), Type.Literal('split')]),
    },
    closed,
  ),
  Type.Object(
    { kind: Type.Literal('resize'), primaryFraction: Type.Number({ minimum: 0.2, maximum: 0.8 }) },
    closed,
  ),
  Type.Object(
    { kind: Type.Literal('maximize'), pane: Type.Union([PaneIdSchema, Type.Null()]) },
    closed,
  ),
  Type.Object(
    {
      kind: Type.Literal('chat'),
      collapsed: Type.Boolean(),
    },
    closed,
  ),
  Type.Object(
    { kind: Type.Literal('resizeChat'), width: Type.Integer({ minimum: 360, maximum: 560 }) },
    closed,
  ),
  Type.Object(
    { kind: Type.Literal('recipient'), agentId: Type.Union([AgentIdSchema, Type.Null()]) },
    closed,
  ),
  Type.Object(
    {
      kind: Type.Literal('select'),
      acknowledgment: Type.Optional(RenderAcknowledgmentSchema),
      surfaceId: SurfaceIdSchema,
      evidence: Type.Union([EvidenceRefSchema, Type.Null()]),
    },
    closed,
  ),
]);
export const WorkspaceCommandSchema = Type.Object(
  {
    projectId: ProjectIdSchema,
    expectedRevision: CounterSchema,
    requestId: RequestIdSchema,
    action: WorkspaceActionSchema,
  },
  closed,
);
export const WorkspaceProjectInputSchema = Type.Object(
  { projectId: Type.Union([ProjectIdSchema, Type.Null()]) },
  closed,
);
export const WorkspaceDraftInputSchema = Type.Object(
  {
    projectId: ProjectIdSchema,
    draft: Type.String({ maxLength: 16000 }),
  },
  closed,
);
export const WorkspaceReadInputSchema = Type.Object({ projectId: ProjectIdSchema }, closed);
/** Trusted renderer reports its actual mounted regions before any resource load/acknowledgment. */
export const WorkspacePresentationInputSchema = Type.Object(
  {
    projectId: ProjectIdSchema,
    rendererSessionId: RendererSessionIdSchema,
    visiblePanes: Type.Array(PaneIdSchema, { maxItems: 2, uniqueItems: true }),
  },
  closed,
);
export type WorkspacePresentationInput = Static<typeof WorkspacePresentationInputSchema>;
export const NotebookImageInputSchema = Type.Object(
  { acknowledgment: RenderAcknowledgmentSchema, target: NotebookTargetSchema },
  closed,
);
export type NotebookImageInput = Static<typeof NotebookImageInputSchema>;
export const NotebookImageResultSchema = serviceResult(NotebookImageSchema);
export const NotebookViewportInputSchema = Type.Object(
  { acknowledgment: RenderAcknowledgmentSchema, viewport: NotebookViewportSchema },
  closed,
);
export type NotebookViewportInput = Static<typeof NotebookViewportInputSchema>;
export const PdfViewportInputSchema = Type.Object(
  { acknowledgment: RenderAcknowledgmentSchema, region: Type.Union([RectSchema, Type.Null()]) },
  closed,
);
export type PdfViewportInput = Static<typeof PdfViewportInputSchema>;
export const WorkspaceInteractionInputSchema = Type.Object(
  { token: RequestIdSchema, blocked: Type.Boolean() },
  closed,
);
export const WorkspaceLoadInputSchema = Type.Object(
  {
    projectId: ProjectIdSchema,
    surfaceId: SurfaceIdSchema,
    rendererSessionId: RendererSessionIdSchema,
    refresh: Type.Optional(Type.Boolean()),
  },
  closed,
);
export const SurfaceLoadSchema = Type.Object(
  {
    acknowledgment: RenderAcknowledgmentSchema,
    data: SurfaceDataSchema,
    observedRevision: Type.Optional(RevisionSchema),
    dependencyRevision: Type.Optional(RevisionSchema),
    refreshProblem: Type.Optional(Type.String({ maxLength: 1000 })),
    referenceView: Type.Optional(ReferenceViewSchema),
    observedReferenceView: Type.Optional(ObservedReferenceViewSchema),
  },
  closed,
);
export const WorkspaceBootstrapSchema = Type.Object(
  {
    rendererSessionId: RendererSessionIdSchema,
    projects: Type.Array(ProjectInfoSchema, { maxItems: 100 }),
    document: Type.Union([WorkspaceDocumentSchema, Type.Null()]),
    notice: Type.Union([Type.String({ maxLength: 500 }), Type.Null()]),
  },
  closed,
);
export const WorkspaceResultSchema = serviceResult(WorkspaceDocumentSchema);
export const WorkspaceBootstrapResultSchema = serviceResult(WorkspaceBootstrapSchema);
export const SurfaceLoadResultSchema = serviceResult(SurfaceLoadSchema);
export const WorkspaceAcknowledgedResultSchema = serviceResult(
  Type.Object({ accepted: Type.Literal(true) }, closed),
);
export const WorkspaceShortcutSchema = Type.Union([
  Type.Literal('open-folder'),
  Type.Literal('close-view'),
  Type.Literal('split'),
  Type.Literal('pin'),
]);
export type WorkspaceShortcut = Static<typeof WorkspaceShortcutSchema>;

export type WorkspaceAction = Static<typeof WorkspaceActionSchema>;
export type WorkspaceCommand = Static<typeof WorkspaceCommandSchema>;
export type WorkspaceBootstrap = Static<typeof WorkspaceBootstrapSchema>;
export type SurfaceLoad = Static<typeof SurfaceLoadSchema>;
export type WorkspaceBridge = Readonly<{
  workspace: Readonly<{
    onChanged: (listener: (document: Static<typeof WorkspaceDocumentSchema>) => void) => () => void;
    onShortcut: (listener: (shortcut: WorkspaceShortcut) => void) => () => void;
    connect: () => Promise<Static<typeof WorkspaceBootstrapResultSchema>>;
    read: (
      input: Static<typeof WorkspaceReadInputSchema>,
    ) => Promise<Static<typeof WorkspaceResultSchema>>;
    openProject: (
      input: Static<typeof WorkspaceProjectInputSchema>,
    ) => Promise<Static<typeof WorkspaceBootstrapResultSchema>>;
    command: (input: WorkspaceCommand) => Promise<Static<typeof WorkspaceResultSchema>>;
    updateDraft: (
      input: Static<typeof WorkspaceDraftInputSchema>,
    ) => Promise<Static<typeof WorkspaceResultSchema>>;
    present: (
      input: WorkspacePresentationInput,
    ) => Promise<Static<typeof WorkspaceAcknowledgedResultSchema>>;
    notebookImage: (input: NotebookImageInput) => Promise<Static<typeof NotebookImageResultSchema>>;
    notebookViewport: (
      input: NotebookViewportInput,
    ) => Promise<Static<typeof WorkspaceAcknowledgedResultSchema>>;
    pdfViewport: (
      input: PdfViewportInput,
    ) => Promise<Static<typeof WorkspaceAcknowledgedResultSchema>>;
    interaction: (
      input: Static<typeof WorkspaceInteractionInputSchema>,
    ) => Promise<Static<typeof WorkspaceAcknowledgedResultSchema>>;
    loadSurface: (
      input: Static<typeof WorkspaceLoadInputSchema>,
    ) => Promise<Static<typeof SurfaceLoadResultSchema>>;
    acknowledge: (
      input: Static<typeof RenderAcknowledgmentSchema>,
    ) => Promise<Static<typeof WorkspaceAcknowledgedResultSchema>>;
    invalidate: (
      input: Static<typeof RenderAcknowledgmentSchema>,
    ) => Promise<Static<typeof WorkspaceAcknowledgedResultSchema>>;
  }>;
}>;
export const WORKSPACE_CHANNELS = {
  notebookImage: 'gobble:workspace:notebook-image:v1',
  notebookViewport: 'gobble:workspace:notebook-viewport:v1',
  pdfViewport: 'gobble:workspace:pdf-viewport:v1',
  read: 'gobble:workspace:read:v1',
  connect: 'gobble:workspace:connect:v1',
  openProject: 'gobble:workspace:open-project:v1',
  command: 'gobble:workspace:command:v1',
  updateDraft: 'gobble:workspace:draft:v1',
  present: 'gobble:workspace:present:v2',
  interaction: 'gobble:workspace:interaction:v1',
  loadSurface: 'gobble:workspace:load:v3',
  acknowledge: 'gobble:workspace:acknowledge:v3',
  invalidate: 'gobble:workspace:invalidate:v3',
  shortcut: 'gobble:workspace:shortcut:v1',
} as const;
