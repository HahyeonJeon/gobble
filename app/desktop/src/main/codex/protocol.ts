import { AppProblem } from '../problem';
import type { ProviderTurn, ProviderEvent } from '../collaboration/provider';

// Narrow projections of the 0.153.4 generated protocol. Unused provider fields
// never cross into app contracts; required fields are checked at runtime.
export const malformed = () =>
  new AppProblem(
    'incompatible_runtime',
    'Codex returned an unsupported response. Reconnect to the supported runtime.',
  );
export function object(value: unknown): Record<string, unknown> {
  if (!value || typeof value !== 'object' || Array.isArray(value)) throw malformed();
  return value as Record<string, unknown>;
}
export function string(value: unknown, max = 256): string {
  if (typeof value !== 'string' || value.length > max) throw malformed();
  return value;
}
export function id(value: unknown): string {
  const result = string(value);
  if (!result) throw malformed();
  return result;
}
export function array(value: unknown, max: number): unknown[] {
  if (!Array.isArray(value) || value.length > max) throw malformed();
  return value;
}
export function decodeTurn(raw: unknown): ProviderTurn {
  const value = object(raw);
  const status = value.status;
  if (
    status !== 'inProgress' &&
    status !== 'completed' &&
    status !== 'interrupted' &&
    status !== 'failed'
  )
    throw malformed();
  const items = array(value.items, 10000).map(object);
  return {
    id: id(value.id),
    status,
    clientIds: items
      .filter((item) => item.type === 'userMessage' && item.clientId != null)
      .map((item) => id(item.clientId)),
    messages: items
      .filter((item) => item.type === 'agentMessage')
      .map((item) => ({ id: id(item.id), text: string(item.text, 32000) })),
  };
}

export function decodeEvent(method: string, raw: unknown): ProviderEvent | null {
  if (method === 'turn/started' || method === 'turn/completed') {
    const value = object(raw);
    return { kind: 'turn', threadId: id(value.threadId), turn: decodeTurn(value.turn) };
  }
  if (method === 'item/agentMessage/delta') {
    const value = object(raw);
    return {
      kind: 'delta',
      threadId: id(value.threadId),
      turnId: id(value.turnId),
      itemId: id(value.itemId),
      text: string(value.delta, 32000),
    };
  }
  if (method === 'item/completed') {
    const value = object(raw);
    const item = object(value.item);
    if (item.type === 'agentMessage')
      return {
        kind: 'message',
        threadId: id(value.threadId),
        turnId: id(value.turnId),
        itemId: id(item.id),
        text: string(item.text, 32000),
      };
  }
  return null;
}
