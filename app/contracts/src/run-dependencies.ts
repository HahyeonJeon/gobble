import { Type, type Static } from '@sinclair/typebox';
import {
  closed,
  CounterSchema,
  ProjectIdSchema,
  RunRefSchema,
  RevisionSchema,
  TimestampSchema,
} from './identity';
import { RunPresentationSchema } from './run-presentation';

export const DEPENDENCY_LIMITS = {
  inspectedEdges: 1000,
  groups: 80,
  edges: 200,
  captureMembers: 100,
  captureBytes: 65536,
} as const;
const id = Type.String({ minLength: 1, maxLength: 256 });
export const TaskDependencySchema = Type.Object({ fromTaskId: id, toTaskId: id }, closed);
export const DependencyTopologySchema = Type.Union([
  Type.Object(
    {
      kind: Type.Literal('unavailable'),
      reason: Type.Union([Type.Literal('missing'), Type.Literal('invalid')]),
    },
    closed,
  ),
  Type.Object(
    {
      kind: Type.Literal('reported'),
      availableEntries: CounterSchema,
      inspectedEntries: CounterSchema,
      invalidEntries: CounterSchema,
      duplicateEntries: CounterSchema,
      truncated: Type.Boolean(),
      edges: Type.Array(TaskDependencySchema, { maxItems: DEPENDENCY_LIMITS.inspectedEdges }),
    },
    closed,
  ),
]);
/** Independent from the frozen RunPresentation used by v2/v3 references. */
export const DependencyReadSchema = Type.Object(
  {
    schemaVersion: Type.Literal(1),
    run: Type.Omit(RunPresentationSchema, ['dependencies']),
    topology: DependencyTopologySchema,
  },
  closed,
);
export const DependencyMemberSchema = Type.Object(
  {
    instanceId: id,
    attempt: CounterSchema,
    status: Type.Union([Type.String(), Type.Null()]),
    template: Type.Union([Type.Boolean(), Type.Null()]),
    expanded: Type.Union([Type.Boolean(), Type.Null()]),
  },
  closed,
);
export const DependencyGroupSchema = Type.Object(
  {
    taskId: id,
    members: Type.Array(DependencyMemberSchema, { maxItems: 1000 }),
    membership: Type.Union([Type.Literal('complete-preview'), Type.Literal('partial-preview')]),
    counts: Type.Object(
      {
        templates: CounterSchema,
        unstarted: CounterSchema,
        attempted: CounterSchema,
        unknownTemplate: CounterSchema,
        states: Type.Array(
          Type.Object(
            { status: Type.Union([Type.String(), Type.Null()]), count: CounterSchema },
            closed,
          ),
          { maxItems: 1000 },
        ),
      },
      closed,
    ),
  },
  closed,
);
export const DependencyObservationSchema = Type.Object(
  {
    schemaVersion: Type.Literal(1),
    source: Type.Object(
      {
        projectId: ProjectIdSchema,
        runRef: RunRefSchema,
        engineRevision: RevisionSchema,
        observedAt: TimestampSchema,
        imageId: Type.String(),
        runId: Type.String(),
      },
      closed,
    ),
    topology: Type.Union([Type.Literal('unavailable'), Type.Literal('reported')]),
    display: Type.Union([Type.Literal('graph'), Type.Literal('list'), Type.Literal('unavailable')]),
    diagnostics: Type.Array(
      Type.Union([
        Type.Literal('missing-topology'),
        Type.Literal('invalid-topology'),
        Type.Literal('duplicate-edges'),
        Type.Literal('partial-topology'),
        Type.Literal('cyclic-topology'),
        Type.Literal('group-limit'),
        Type.Literal('edge-limit'),
        Type.Literal('partial-membership'),
        Type.Literal('unmapped-members'),
      ]),
      { maxItems: 9, uniqueItems: true },
    ),
    scope: Type.Object(
      {
        availableTasks: Type.Union([CounterSchema, Type.Null()]),
        returnedTasks: CounterSchema,
        membershipComplete: Type.Boolean(),
        observedGroups: CounterSchema,
        observedEdges: CounterSchema,
        availableEdgeEntries: Type.Union([CounterSchema, Type.Null()]),
        inspectedEdgeEntries: CounterSchema,
        invalidEdgeEntries: CounterSchema,
        duplicateEdgeEntries: CounterSchema,
      },
      closed,
    ),
    groups: Type.Array(DependencyGroupSchema, { maxItems: DEPENDENCY_LIMITS.groups }),
    edges: Type.Array(TaskDependencySchema, { maxItems: DEPENDENCY_LIMITS.edges }),
    unmappedInstances: Type.Array(id, { maxItems: 1000, uniqueItems: true }),
  },
  closed,
);
export const DependencySelectionSchema = Type.Union([
  Type.Object(
    {
      kind: Type.Literal('run-group'),
      coordinateSpace: Type.Literal('observed-authored-task-group'),
      taskId: id,
    },
    closed,
  ),
  Type.Object(
    {
      kind: Type.Literal('run-dependency'),
      coordinateSpace: Type.Literal('observed-authored-task-pair'),
      fromTaskId: id,
      toTaskId: id,
    },
    closed,
  ),
]);
/** Exact dependency address used by current selections, captures and shared references. */
export const DependencyTargetSchema = Type.Object(
  {
    schemaVersion: Type.Literal(4),
    projectId: ProjectIdSchema,
    resource: Type.Object({ kind: Type.Literal('run'), runRef: RunRefSchema }, closed),
    dataRevision: RevisionSchema,
    selection: DependencySelectionSchema,
  },
  closed,
);
export const DependencyCaptureSchema = Type.Object(
  {
    schemaVersion: Type.Literal(1),
    capturedAt: TimestampSchema,
    target: DependencyTargetSchema,
    source: DependencyObservationSchema.properties.source,
    scope: DependencyObservationSchema.properties.scope,
    diagnostics: DependencyObservationSchema.properties.diagnostics,
    groups: Type.Array(
      Type.Object(
        {
          taskId: id,
          membership: DependencyGroupSchema.properties.membership,
          counts: DependencyGroupSchema.properties.counts,
          observedMembers: CounterSchema,
          members: Type.Array(DependencyMemberSchema, {
            maxItems: DEPENDENCY_LIMITS.captureMembers,
          }),
          truncated: Type.Boolean(),
        },
        closed,
      ),
      { maxItems: 2 },
    ),
    edge: Type.Union([TaskDependencySchema, Type.Null()]),
  },
  closed,
);
export type DependencyRead = Static<typeof DependencyReadSchema>;
export type DependencyTopology = Static<typeof DependencyTopologySchema>;
export type DependencyObservation = Static<typeof DependencyObservationSchema>;
export type DependencyGroup = Static<typeof DependencyGroupSchema>;
export type DependencyTarget = Static<typeof DependencyTargetSchema>;
export type DependencyCapture = Static<typeof DependencyCaptureSchema>;
export type TaskDependency = Static<typeof TaskDependencySchema>;
