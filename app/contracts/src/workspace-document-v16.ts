import { Type } from '@sinclair/typebox';
import { restoreSchema } from './storage/restore-schema';
import v16 from './storage/workspace-v16.json' with { type: 'json' };
export const WorkspaceDocumentV16Schema = Type.Unsafe<{ schemaVersion: 16 }>(restoreSchema(v16));
