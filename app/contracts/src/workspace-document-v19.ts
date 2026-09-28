import { Type } from '@sinclair/typebox';
import { restoreSchema } from './storage/restore-schema';
import v19 from './storage/workspace-v19.json' with { type: 'json' };
export const WorkspaceDocumentV19Schema = Type.Unsafe<{ schemaVersion: 19 }>(restoreSchema(v19));
