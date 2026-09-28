import { Type } from '@sinclair/typebox';
import { restoreSchema } from './storage/restore-schema';
import v18 from './storage/workspace-v18.json' with { type: 'json' };
export const WorkspaceDocumentV18Schema = Type.Unsafe<{ schemaVersion: 18 }>(restoreSchema(v18));
