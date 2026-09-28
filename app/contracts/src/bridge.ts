import { Type, type Static } from '@sinclair/typebox';
import { closed } from './identity';
import type { ServiceBridge } from './service';
import type { WorkspaceBridge } from './workspace-bridge';
import type { CollaborationBridge } from './collaboration';
import type { EvidenceBridge } from './evidence-bridge';

import { ContractErrorSchema } from './result';

export const AppInfoRequestSchema = Type.Object({ schemaVersion: Type.Literal(1) }, closed);
export const AppInfoResultSchema = Type.Union([
  Type.Object(
    {
      schemaVersion: Type.Literal(1),
      ok: Type.Literal(true),
      value: Type.Object(
        {
          name: Type.Literal('Gobble'),
          appVersion: Type.String({ minLength: 1, maxLength: 64 }),
          electronVersion: Type.String({ minLength: 1, maxLength: 64 }),
          stage: Type.Literal('workspace'),
        },
        closed,
      ),
    },
    closed,
  ),
  Type.Object(
    { schemaVersion: Type.Literal(1), ok: Type.Literal(false), error: ContractErrorSchema },
    closed,
  ),
]);

export const APP_INFO_CHANNEL = 'gobble:app:get-info:v1';
export type AppInfoResult = Static<typeof AppInfoResultSchema>;
export type DesktopBridge = Readonly<{ getAppInfo: () => Promise<AppInfoResult> }> &
  ServiceBridge &
  CollaborationBridge &
  EvidenceBridge &
  WorkspaceBridge;
