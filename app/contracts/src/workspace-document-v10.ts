import { restoreSchema } from './storage/restore-schema';
import { Type } from '@sinclair/typebox';
import v9 from './storage/workspace-v9.json' with { type: 'json' };
import v10 from './storage/workspace-v10.json' with { type: 'json' };
/** Immutable storage-only boundary; migrations never infer fields from the new live schema. */
export const FrozenWorkspaceV9Schema = Type.Unsafe<{ schemaVersion: 9 }>(restoreSchema(v9));
export const WorkspaceDocumentV10Schema = Type.Unsafe<{ schemaVersion: 10 }>(restoreSchema(v10));
