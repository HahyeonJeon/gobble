import { Type } from '@sinclair/typebox';
import { restoreSchema } from './storage/restore-schema';
import v12 from './storage/workspace-v12.json' with { type: 'json' };
export const WorkspaceDocumentV12Schema = Type.Unsafe<{ schemaVersion: 12 }>(restoreSchema(v12));
