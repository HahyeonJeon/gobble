import { isQuestion, type Question } from '@gobble/contracts';
import type { SharedReference, WorkspaceDocument } from '@gobble/contracts';

type Submission = NonNullable<WorkspaceDocument['collaboration']>['submissions'][number];
type ChatItem =
  | { kind: 'submission'; id: string; at: number; submission: Submission }
  | { kind: 'response'; id: string; at: number; submission: Submission }
  | { kind: 'question'; id: string; at: number; question: Question }
  | { kind: 'reference'; id: string; at: number; reference: SharedReference }
  | { kind: 'activity'; id: string; at: number; text: string };

/** Read projection only: split new replies by first receipt, preserve legacy pairing; no second writable message store. */
export function chatItems(document: WorkspaceDocument): ChatItem[] {
  return [
    ...document.workspace.decisions.filter(isQuestion).map((question): ChatItem => ({
      kind: 'question',
      id: question.decisionId,
      at: question.createdAt,
      question,
    })),
    ...(document.collaboration?.submissions ?? []).flatMap((submission): ChatItem[] => [
      { kind: 'submission', id: submission.requestId, at: submission.createdAt, submission },
      ...(typeof submission.responseStartedAt === 'number'
        ? [
            {
              kind: 'response' as const,
              id: submission.requestId + ':response',
              at: submission.responseStartedAt,
              submission,
            },
          ]
        : []),
    ]),
    ...(document.sharedReferences ?? []).map((reference): ChatItem => ({
      kind: 'reference',
      id: reference.referenceId,
      at: reference.createdAt,
      reference,
    })),
    ...document.activity.map((item): ChatItem => ({ kind: 'activity', ...item })),
  ].sort((a, b) => a.at - b.at || a.id.localeCompare(b.id));
}

type Activity = Extract<ChatItem, { kind: 'activity' }>;
export type TimelineEntry =
  ChatItem | { kind: 'activityGroup'; id: string; at: number; items: Activity[] };

// Recognize only the existing controller's routine view messages. Future/unknown
// activity stays explicit until its meaning has been reviewed here.
function routine(item: ChatItem): item is Activity {
  return (
    item.kind === 'activity' &&
    ([
      'Opened a Project view.',
      'Duplicated a view into another pane.',
      'Closed a view. Its source is unchanged.',
      'Opened a second pane.',
      'Combined the panes.',
    ].includes(item.text) ||
      /^(?:Pinned|Unpinned) .+\.$/s.test(item.text) ||
      /^Moved .+ to the (?:primary|secondary) pane\.$/s.test(item.text))
  );
}
/** Adjacent routine events only; never collapse across a message, question or mark. */
export function groupViewActivity(items: ChatItem[]): TimelineEntry[] {
  const result: TimelineEntry[] = [];
  for (const item of items) {
    const previous = result.at(-1);
    if (routine(item) && previous?.kind === 'activityGroup') previous.items.push(item);
    else if (routine(item) && previous && previous.kind !== 'activityGroup' && routine(previous))
      result[result.length - 1] = {
        kind: 'activityGroup',
        id: previous.id,
        at: previous.at,
        items: [previous, item],
      };
    else result.push(item);
  }
  return result;
}
