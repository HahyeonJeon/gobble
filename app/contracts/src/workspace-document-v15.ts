import { Type } from '@sinclair/typebox';
import { restoreSchema } from './storage/restore-schema';
import v15 from './storage/workspace-v15.json' with { type: 'json' };
export const WorkspaceDocumentV15Schema = Type.Unsafe<{ schemaVersion: 15 }>(restoreSchema(v15));
