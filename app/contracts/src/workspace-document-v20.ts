import { Type } from '@sinclair/typebox';
import { restoreSchema } from './storage/restore-schema';
import v20 from './storage/workspace-v20.json' with { type: 'json' };
export const WorkspaceDocumentV20Schema = Type.Unsafe<{ schemaVersion: 20 }>(restoreSchema(v20));
