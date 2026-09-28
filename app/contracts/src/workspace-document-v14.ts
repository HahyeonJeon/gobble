import { Type } from '@sinclair/typebox';
import { restoreSchema } from './storage/restore-schema';
import v14 from './storage/workspace-v14.json' with { type: 'json' };
export const WorkspaceDocumentV14Schema = Type.Unsafe<{ schemaVersion: 14 }>(restoreSchema(v14));
