import { describe, expect, it } from 'vitest';
import { readFileSync } from 'node:fs';
import {
  parse,
  PipelineFlowSchema,
  validatePipelineFlow,
  type PipelineFlow,
  PipelineInspectionInputSchema,
  CheckPipelineInputSchema,
  EvidenceRefSchema,
  WorkspaceDocumentV14Schema,
  parseWorkspaceDocument,
  readStoredWorkspace,
} from '@gobble/contracts';
import { ProjectService } from '../src/main/service/project-service';
import { emptyWorkspace, reusableSurface, transition } from '../src/main/workspace/model';
import {
  flowLayout,
  stepKey,
  inputKey,
  nodeWidth,
  nodeHeight,
} from '../src/renderer/workspace/views/pipeline/flow-layout';
import { flowRoutes } from '../src/renderer/workspace/views/pipeline/flow-routing';

// Produced by the actual pinned Gobble evaluator in P2A-1 qualification.
const fixture = () =>
  parse(
    PipelineFlowSchema,
    JSON.parse(readFileSync(new URL('./fixtures/pipeline-flow.json', import.meta.url), 'utf8')),
  );
const success = (value: unknown) => ({ schemaVersion: 1, ok: true, value });
const artifact = (flow: PipelineFlow) => ({
  artifactId: 'sha256:' + 'a'.repeat(64),
  sourceRevision: 'sha256:' + 'b'.repeat(64),
  checkedAt: '2026-09-09T00:00:00Z',
  flow,
});

describe('pipeline flow semantics and boundaries', () => {
  it('routes exact connections orthogonally outside every card, including skipped columns', () => {
    const flow = fixture();
    const original = JSON.stringify(flow);
    const layout = flowLayout(flow);
    const routes = flowRoutes(flow, layout);
    expect(routes.edges.map((edge) => edge.id)).toEqual(flow.connections.map((edge) => edge.id));
    for (const edge of routes.edges) {
      expect(edge.path).not.toMatch(/NaN|Infinity|C/);
      expect(edge.points[0]).toEqual(edge.start);
      expect(edge.points.at(-1)).toEqual(edge.end);
      for (let index = 1; index < edge.points.length; index++) {
        const a = edge.points[index - 1]!;
        const b = edge.points[index]!;
        expect(a.x === b.x || a.y === b.y).toBe(true);
        for (const node of layout.nodes) {
          const crosses =
            a.x === b.x
              ? a.x > node.x &&
                a.x < node.x + nodeWidth &&
                Math.max(a.y, b.y) > node.y &&
                Math.min(a.y, b.y) < node.y + nodeHeight
              : a.y > node.y &&
                a.y < node.y + nodeHeight &&
                Math.max(a.x, b.x) > node.x &&
                Math.min(a.x, b.x) < node.x + nodeWidth;
          expect(crosses, `Edge ${edge.id} crosses ${node.label}`).toBe(false);
        }
      }
    }
    expect(routes.edges.some((edge) => edge.path.includes('Q'))).toBe(true);
    expect(JSON.stringify(flow)).toBe(original);
  });
  it('keeps multiple port connections between the same steps independently addressable', () => {
    const flow = fixture();
    const quality = flow.steps.find((step) => step.id === 'quality')!;
    quality.inputs.push({ ...quality.inputs[0]!, name: 'comparison' });
    flow.connections.push({
      id: 'comparison',
      fromTask: 'trim',
      fromPort: 'trimmed',
      toTask: 'quality',
      toPort: 'comparison',
      wait: [],
    });
    validatePipelineFlow(flow);
    const routes = flowRoutes(flow, flowLayout(flow));
    const pair = routes.edges.filter((route) =>
      flow.connections.some(
        (edge) => edge.id === route.id && edge.fromTask === 'trim' && edge.toTask === 'quality',
      ),
    );
    expect(pair).toHaveLength(2);
    // Both edges originate at the same declared output; distinct destination ports stay distinct.
    expect(pair[0]!.start).toEqual(pair[1]!.start);
    expect(pair[0]!.end).not.toEqual(pair[1]!.end);
    expect(pair[0]!.path).not.toEqual(pair[1]!.path);
  });
  it('reuses a pipeline within its Pane and opens an independent second view', () => {
    const resource = { kind: 'pipeline' as const, pipelineId: 'pip_one' };
    const open = { kind: 'open' as const, resource, pane: 'primary' as const, duplicate: false };
    const start = emptyWorkspace('prj_one');
    start.chat.draft = 'Discuss these steps';
    const first = transition(start, open, {
      surfaceId: 'srf_first',
      title: 'Analysis',
      view: 'pipeline',
    });
    expect(reusableSurface(first, open)?.surfaceId).toBe('srf_first');
    const other = { ...open, pane: 'secondary' as const };
    expect(reusableSurface(first, other)).toBeUndefined();
    const both = transition(first, other, {
      surfaceId: 'srf_other',
      title: 'Analysis',
      view: 'pipeline',
    });
    expect(reusableSurface(both, other)?.surfaceId).toBe('srf_other');
    expect(transition(both, other).workspace.surfaces).toHaveLength(2);
    const closed = transition(both, { kind: 'close', surfaceId: 'srf_other' });
    expect(closed.workspace.surfaces.map((surface) => surface.surfaceId)).toEqual(['srf_first']);
    expect(closed.chat.draft).toBe('Discuss these steps');
  });
  it('preserves true fan-out, boundary ports and per-port connection identity', () => {
    const flow = fixture();
    validatePipelineFlow(flow);
    expect(flow.inputs.map((input) => input.name)).toEqual(['reads', 'reference']);
    expect(flow.connections).toHaveLength(7);
    const layout = flowLayout(flow);
    const x = (key: string) => layout.nodes.find((node) => node.key === key)!.x;
    expect(x(stepKey('align'))).toBe(x(stepKey('quality')));
    expect(x(inputKey('reads'))).toBeLessThan(x(stepKey('trim')));
    expect(
      flow.connections.some((edge) => edge.fromTask === 'quality' && edge.toTask === 'align'),
    ).toBe(false);
    const before = JSON.stringify(flow);
    flowLayout(flow);
    expect(JSON.stringify(flow)).toBe(before);
  });
  it('rejects unknown fields, duplicate identities, lost ports and cycles before rendering', () => {
    expect(() => parse(PipelineFlowSchema, { ...fixture(), command: 'run anything' })).toThrow();
    for (const corrupt of [
      (flow: PipelineFlow) => flow.steps.push(flow.steps[0]!),
      (flow: PipelineFlow) => {
        flow.connections[0]!.fromPort = 'missing';
      },
      (flow: PipelineFlow) => flow.connections.push({ ...flow.connections[0]!, id: 'another' }),
      (flow: PipelineFlow) => flow.steps[0]!.inputs.push(flow.steps[0]!.inputs[0]!),
      (flow: PipelineFlow) =>
        flow.connections.push({
          id: 'cycle',
          fromTask: 'summary',
          fromPort: 'summary',
          toTask: 'trim',
          toPort: 'reads',
          wait: [],
        }),
    ]) {
      const flow = fixture();
      corrupt(flow);
      expect(() => validatePipelineFlow(flow)).toThrow();
    }
  });
  it('validates Project/Pipeline/job associations and accepts no source or execution arguments', async () => {
    const input = { projectId: 'prj_one', pipelineId: 'pip_one', requestId: 'req_check' };
    expect(() => parse(PipelineInspectionInputSchema, { ...input, path: '/private' })).toThrow();
    expect(() => parse(CheckPipelineInputSchema, { ...input, command: 'run' })).toThrow();
    const ready = {
      projectId: 'prj_one',
      pipelineId: 'pip_one',
      state: 'ready',
      jobId: 'req_check',
      artifact: artifact(fixture()),
    };
    const service = new ProjectService({ request: async () => success(ready) });
    await expect(service.checkPipeline(input)).resolves.toEqual(ready);
    for (const bad of [
      { ...ready, projectId: 'prj_other' },
      { ...ready, pipelineId: 'pip_other' },
      { ...ready, jobId: 'req_other' },
      { ...ready, artifact: undefined },
    ]) {
      const service = new ProjectService({ request: async () => success(bad) });
      await expect(service.checkPipeline(input)).rejects.toThrow();
    }
  });
  it('adds a storage version without extending historical discussion authority', () => {
    const old = { ...emptyWorkspace('prj_one'), schemaVersion: 14 };
    expect(readStoredWorkspace(old).schemaVersion).toBe(21);
    const doc = emptyWorkspace('prj_one');
    doc.workspace.surfaces.push({
      projectId: 'prj_one',
      surfaceId: 'srf_flow',
      resource: { kind: 'pipeline', pipelineId: 'pip_one' },
      view: 'pipeline',
      pinned: false,
      openedBy: { kind: 'user' },
    });
    doc.workspace.layout.primary = { tabs: ['srf_flow'], activeSurfaceId: 'srf_flow' };
    doc.titles.push({ surfaceId: 'srf_flow', title: 'Analysis' });
    expect(parseWorkspaceDocument(doc).workspace.surfaces[0]!.view).toBe('pipeline');
    expect(() => parse(WorkspaceDocumentV14Schema, { ...doc, schemaVersion: 14 })).toThrow();
    expect(() =>
      parse(EvidenceRefSchema, {
        schemaVersion: 2,
        projectId: 'prj_one',
        resource: { kind: 'pipeline', pipelineId: 'pip_one' },
        dataRevision: 'sha256:' + 'a'.repeat(64),
      }),
    ).toThrow();
  });
});
