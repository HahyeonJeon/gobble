import { Type } from '@sinclair/typebox';

// Prefixes prevent accidental interchange; possession of an ID grants no access.
const identifier = (prefix: string) =>
  Type.String({ pattern: `^${prefix}_[A-Za-z0-9_-]{1,80}$`, maxLength: 90 });

export const ProjectIdSchema = identifier('prj');
export const AgentIdSchema = identifier('agt');
export const SurfaceIdSchema = identifier('srf');
export const ResourceIdSchema = identifier('res');
export const RunRefSchema = identifier('run');
export const PipelineIdSchema = identifier('pip');
export const DecisionIdSchema = identifier('dec');
export const RequestIdSchema = identifier('req');
export const RendererSessionIdSchema = identifier('rnd');
export const RevisionSchema = Type.String({ minLength: 1, maxLength: 256 });
export const LabelSchema = Type.String({ minLength: 1, maxLength: 200 });
export const TimestampSchema = Type.Integer({ minimum: 0, maximum: Number.MAX_SAFE_INTEGER });
export const CounterSchema = Type.Integer({ minimum: 0, maximum: Number.MAX_SAFE_INTEGER });
export const closed = { additionalProperties: false };
