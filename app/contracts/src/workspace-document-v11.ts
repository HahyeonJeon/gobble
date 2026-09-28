import { Type } from '@sinclair/typebox';
import { restoreSchema } from './storage/restore-schema';
import v11 from './storage/workspace-v11.json' with { type: 'json' };
export const WorkspaceDocumentV11Schema = Type.Unsafe<{ schemaVersion: 11 }>(restoreSchema(v11));
