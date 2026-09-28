import { Type, type Static } from '@sinclair/typebox';
import { closed, PipelineIdSchema, ProjectIdSchema, RequestIdSchema } from './identity';
import { ContractValidationError } from './validation-error';

const text = Type.String({ maxLength: 4096 });
const identity = Type.String({ minLength: 1, maxLength: 256 });
const digest = Type.String({ pattern: '^sha256:[a-f0-9]{64}$' });
const strings = Type.Array(text, { maxItems: 2000 });
export const PipelinePortSchema = Type.Object(
  {
    name: identity,
    kind: Type.Union([Type.Literal('file'), Type.Literal('group'), Type.Literal('tree')]),
    path: text,
    members: Type.Array(Type.Object({ name: identity, path: text }, closed), { maxItems: 2000 }),
  },
  closed,
);
export const PipelineSettingSchema = Type.Object(
  {
    key: Type.String({ minLength: 1, maxLength: 128 }),
    label: Type.String({ minLength: 1, maxLength: 256 }),
    unit: Type.String({ maxLength: 64 }),
    value: Type.Union([
      Type.Integer({ minimum: -Number.MAX_SAFE_INTEGER, maximum: Number.MAX_SAFE_INTEGER }),
      Type.Null(),
    ]),
  },
  closed,
);
const PipelineStepV1Schema = Type.Object(
  {
    id: identity,
    name: identity,
    module: text,
    display: Type.Object(
      { stage: Type.Optional(text), scope: Type.Optional(text), samples: Type.Optional(strings) },
      closed,
    ),
    image: text,
    backend: text,
    cpu: Type.Number({ minimum: 0 }),
    memory: text,
    inputs: Type.Array(PipelinePortSchema, { maxItems: 500 }),
    outputs: Type.Array(PipelinePortSchema, { maxItems: 500 }),
    control: Type.Object(
      {
        branch: text,
        merge: text,
        scatter: text,
        gather: text,
        when: text,
        scatterFromKind: text,
        scatterFromTask: text,
        scatterFromPort: text,
        scatterFromPath: text,
        scatterMembers: strings,
        scatterMemberPaths: strings,
        skipIfMissingTask: text,
        skipIfMissingPort: text,
        skipIfMissingPath: text,
        skipIfFalse: text,
      },
      closed,
    ),
  },
  closed,
);
export const PipelineConnectionSchema = Type.Object(
  {
    id: identity,
    fromTask: text,
    fromPort: identity,
    toTask: identity,
    toPort: identity,
    wait: strings,
  },
  closed,
);
export const PipelineStepSchema = Type.Object(
  {
    ...PipelineStepV1Schema.properties,
    settings: Type.Optional(Type.Array(PipelineSettingSchema, { maxItems: 32 })),
  },
  closed,
);
/** Gobble-owned authored flow. No command inference, execution or scientific-validity claim. */
export const PipelineFlowV1Schema = Type.Object(
  {
    schemaVersion: Type.Literal(1),
    name: identity,
    inputs: Type.Array(PipelinePortSchema, { maxItems: 500 }),
    steps: Type.Array(PipelineStepV1Schema, { maxItems: 500 }),
    connections: Type.Array(PipelineConnectionSchema, { maxItems: 2000 }),
  },
  closed,
);
export const PipelineFlowSchema = Type.Union([
  PipelineFlowV1Schema,
  Type.Object(
    {
      ...PipelineFlowV1Schema.properties,
      schemaVersion: Type.Literal(2),
      steps: Type.Array(
        Type.Object(
          {
            ...PipelineStepV1Schema.properties,
            settings: Type.Array(PipelineSettingSchema, { maxItems: 32 }),
          },
          closed,
        ),
        { maxItems: 500 },
      ),
    },
    closed,
  ),
]);
export const PipelineArtifactSchema = Type.Object(
  {
    artifactId: digest,
    sourceRevision: digest,
    checkedAt: Type.String({ minLength: 1, maxLength: 100 }),
    flow: PipelineFlowSchema,
  },
  closed,
);
export const PipelineInspectionSchema = Type.Object(
  {
    projectId: ProjectIdSchema,
    pipelineId: PipelineIdSchema,
    managed: Type.Optional(Type.Boolean()),
    state: Type.Union(
      (['idle', 'checking', 'ready', 'failed', 'cancelled'] as const).map((value) =>
        Type.Literal(value),
      ),
    ),
    jobId: Type.Optional(RequestIdSchema),
    issue: Type.Optional(Type.String({ maxLength: 1000 })),
    artifact: Type.Optional(PipelineArtifactSchema),
  },
  closed,
);
export const PipelineInspectionInputSchema = Type.Object(
  { projectId: ProjectIdSchema, pipelineId: PipelineIdSchema },
  closed,
);
export const CheckPipelineInputSchema = Type.Object(
  { ...PipelineInspectionInputSchema.properties, requestId: RequestIdSchema },
  closed,
);
export const CancelPipelineCheckInputSchema = Type.Object(
  { ...PipelineInspectionInputSchema.properties, jobId: RequestIdSchema },
  closed,
);
export type PipelineFlow = Omit<Static<typeof PipelineFlowSchema>, 'steps'> & {
  steps: PipelineStep[];
};
export type PipelineStep = Static<typeof PipelineStepSchema>;
export type PipelinePort = Static<typeof PipelinePortSchema>;
export type PipelineConnection = Static<typeof PipelineConnectionSchema>;
export type PipelineArtifact = Static<typeof PipelineArtifactSchema>;
export type PipelineInspection = Static<typeof PipelineInspectionSchema>;
export type PipelineInspectionInput = Static<typeof PipelineInspectionInputSchema>;
export type CheckPipelineInput = Static<typeof CheckPipelineInputSchema>;
export type CancelPipelineCheckInput = Static<typeof CancelPipelineCheckInputSchema>;

/** Validate the actual graph, not its drawing. Parallel port edges remain distinct. */
export function validatePipelineFlow(flow: PipelineFlow): void {
  const fail = () => {
    throw new ContractValidationError('The pipeline returned inconsistent flow identities.');
  };
  const ports = (values: PipelinePort[]) => {
    const keys = new Set(values.map((value) => value.name));
    if (keys.size !== values.length) fail();
    for (const value of values)
      if (new Set(value.members.map((member) => member.name)).size !== value.members.length) fail();
    return keys;
  };
  for (const step of flow.steps) {
    if (step.settings && new Set(step.settings.map((s) => s.key)).size !== step.settings.length)
      fail();
  }
  const inputs = ports(flow.inputs);
  const steps = new Map(
    flow.steps.map((step) => [step.id, { input: ports(step.inputs), output: ports(step.outputs) }]),
  );
  if (
    steps.size !== flow.steps.length ||
    new Set(flow.connections.map((edge) => edge.id)).size !== flow.connections.length
  )
    fail();
  const endpoints = new Set<string>();
  const incoming = new Map(flow.steps.map((step) => [step.id, 0]));
  const outgoing = new Map<string, string[]>();
  for (const edge of flow.connections) {
    if (
      !(edge.fromTask === '' ? inputs : steps.get(edge.fromTask)?.output)?.has(edge.fromPort) ||
      !steps.get(edge.toTask)?.input.has(edge.toPort)
    )
      fail();
    const key = JSON.stringify([edge.fromTask, edge.fromPort, edge.toTask, edge.toPort]);
    if (endpoints.has(key)) fail();
    endpoints.add(key);
    if (edge.fromTask !== '') {
      incoming.set(edge.toTask, incoming.get(edge.toTask)! + 1);
      outgoing.set(edge.fromTask, [...(outgoing.get(edge.fromTask) ?? []), edge.toTask]);
    }
  }
  const queue = [...incoming].filter(([, degree]) => degree === 0).map(([id]) => id);
  for (let index = 0; index < queue.length; index++)
    for (const next of outgoing.get(queue[index]!) ?? []) {
      const degree = incoming.get(next)! - 1;
      incoming.set(next, degree);
      if (degree === 0) queue.push(next);
    }
  if (queue.length !== steps.size) fail();
}
