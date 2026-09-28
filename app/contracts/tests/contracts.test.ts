import { describe, expect, it } from 'vitest';
import {
  AppInfoRequestSchema,
  ContractValidationError,
  DecisionSchema,
  parse,
  parseAddressedContext,
  parseWorkspace,
  ProjectIdSchema,
  RenderAcknowledgmentSchema,
  SelectionSchema,
  validateSelection,
} from '../src';
import { contextFixture, workspaceFixture } from './fixtures';

describe('versioned wire contracts', () => {
  it('accepts two agents and shared split views in a JSON round trip', () => {
    const workspace = workspaceFixture();
    expect(parseWorkspace(JSON.parse(JSON.stringify(workspace)))).toEqual(workspace);
    expect(parseAddressedContext(contextFixture()).recipientAgentId).toBe('agt_methods');
  });

  it.each([null, {}, { schemaVersion: 2 }, { schemaVersion: 1, path: '/private' }])(
    'rejects unsupported or expanded requests: %j',
    (input) => {
      expect(() => parse(AppInfoRequestSchema, input)).toThrow(ContractValidationError);
    },
  );

  it.each(['agt_atlas', '../prj_atlas', 'prj_', 'prj_a/b', 'prj_a\u0000', 'prj_a\n'])(
    'rejects invalid Project identity: %j',
    (id) => {
      expect(() => parse(ProjectIdSchema, id)).toThrow(ContractValidationError);
    },
  );

  it('requires renderer session, request, generation and data revision for an acknowledgment', () => {
    const acknowledgment = {
      schemaVersion: 3,
      presentation: { spec: 0, view: 0, filter: 0 },
      projectId: 'prj_atlas',
      surfaceId: 'srf_plot',
      rendererSessionId: 'rnd_current',
      requestId: 'req_open',
      generation: 2,
      dataRevision: 'hash-v1',
    };
    expect(parse(RenderAcknowledgmentSchema, acknowledgment)).toEqual(acknowledgment);
    for (const key of [
      'rendererSessionId',
      'requestId',
      'generation',
      'dataRevision',
      'presentation',
    ]) {
      const missing = Object.fromEntries(
        Object.entries(acknowledgment).filter(([name]) => name !== key),
      );
      expect(() => parse(RenderAcknowledgmentSchema, missing)).toThrow(ContractValidationError);
    }
  });

  it('rejects newer persisted versions without converting or erasing the input', () => {
    const input = { ...workspaceFixture(), schemaVersion: 2 };
    expect(() => parseWorkspace(input)).toThrow(ContractValidationError);
    expect(input.schemaVersion).toBe(2);
  });
});

describe('workspace association and layout', () => {
  it.each(['agents', 'surfaces', 'decisions'] as const)('rejects cross-Project %s', (key) => {
    const workspace = workspaceFixture();
    const first = workspace[key][0];
    if (!first) throw new Error('Missing fixture');
    first.projectId = 'prj_other';
    expect(() => parseWorkspace(workspace)).toThrow('another Project');
  });

  it('rejects duplicated identifiers, dangling tabs and active views in another pane', () => {
    const workspace = workspaceFixture();
    const agent = workspace.agents[0];
    if (!agent) throw new Error('Missing fixture');
    workspace.agents.push(agent);
    expect(() => parseWorkspace(workspace)).toThrow('unique');
    const dangling = workspaceFixture();
    dangling.layout.primary.tabs = ['srf_missing'];
    expect(() => parseWorkspace(dangling)).toThrow('exactly one pane');
    const active = workspaceFixture();
    active.layout.primary.activeSurfaceId = 'srf_plot';
    expect(() => parseWorkspace(active)).toThrow('Active surface');
  });

  it('rejects duplicate tab placement and invalid view/resource pairings', () => {
    const workspace = workspaceFixture();
    workspace.layout.primary.tabs.push('srf_plot');
    expect(() => parseWorkspace(workspace)).toThrow('exactly one pane');
    const invalid = workspaceFixture();
    const surface = invalid.surfaces[0];
    if (!surface) throw new Error('Missing fixture');
    surface.resource = { kind: 'run', runRef: 'run_existing' };
    expect(() => parseWorkspace(invalid)).toThrow('incompatible');
  });

  it('retains evidence after its view is closed', () => {
    const workspace = workspaceFixture();
    workspace.surfaces = workspace.surfaces.filter((surface) => surface.surfaceId !== 'srf_table');
    workspace.layout.primary = { tabs: [], activeSurfaceId: null };
    expect(parseWorkspace(workspace).decisions).toHaveLength(1);
  });
});

describe('explicit context and decisions', () => {
  it('rejects cross-Project evidence and missing recipients', () => {
    const context = contextFixture();
    const evidence = context.evidence[0];
    if (!evidence) throw new Error('Missing fixture');
    evidence.projectId = 'prj_other';
    expect(() => parseAddressedContext(context)).toThrow('another Project');
    expect(() =>
      parseAddressedContext({ ...contextFixture(), recipientAgentId: undefined }),
    ).toThrow(ContractValidationError);
  });

  it('rejects empty and duplicate table selections', () => {
    for (const rowKeys of [[], ['S03', 'S03']]) {
      expect(() =>
        parse(SelectionSchema, {
          coordinateSpace: 'revision-row-column-keys',
          kind: 'table',
          rowKeys,
          columns: ['sample'],
        }),
      ).toThrow(ContractValidationError);
    }
  });

  it('validates normalized image bounds and nonempty ordered text ranges', () => {
    const image = parse(SelectionSchema, {
      coordinateSpace: 'normalized-original-image',
      kind: 'image',
      x: 0.75,
      y: 0,
      width: 0.25,
      height: 1,
      originalWidth: 1000,
      originalHeight: 500,
      contentHash: `sha256:${'a'.repeat(64)}`,
    });
    expect(() => validateSelection(image)).not.toThrow();
    if (image.kind !== 'image') throw new Error('Missing image fixture');
    expect(() => validateSelection({ ...image, kind: 'image', x: 0.9 })).toThrow('bounds');
    expect(() =>
      validateSelection({
        coordinateSpace: 'utf16-line-column',
        kind: 'text',
        start: { line: 2, column: 0 },
        end: { line: 1, column: 5 },
      }),
    ).toThrow('ordered');
    expect(() =>
      validateSelection({
        coordinateSpace: 'utf16-line-column',
        kind: 'text',
        start: { line: 1, column: 5 },
        end: { line: 1, column: 5 },
      }),
    ).toThrow('nonempty');
  });

  it('requires an explicit answer and distinguishes dismissal and invalidation', () => {
    const decision = workspaceFixture().decisions[0];
    if (!decision) throw new Error('Missing fixture');
    for (const state of [
      { kind: 'answered' },
      { kind: 'answered', answer: '', answeredAt: 5 },
      { kind: 'pending', answer: 'yes' },
      { kind: 'dismissed', dismissedAt: 5, answer: 'yes' },
    ]) {
      expect(() => parse(DecisionSchema, { ...decision, state })).toThrow(ContractValidationError);
    }
    for (const state of [
      { kind: 'answered', answer: 'S03', answeredAt: 5 },
      { kind: 'dismissed', dismissedAt: 5 },
      { kind: 'invalidated', reason: 'evidence_changed', invalidatedAt: 5 },
    ]) {
      expect(parse(DecisionSchema, { ...decision, state }).state.kind).toBe(state.kind);
    }
  });
});
