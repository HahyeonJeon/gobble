import { Type } from '@sinclair/typebox';
import { restoreSchema } from './storage/restore-schema';
import v13 from './storage/workspace-v13.json' with { type: 'json' };
export const WorkspaceDocumentV13Schema = Type.Unsafe<{ schemaVersion: 13 }>(restoreSchema(v13));
