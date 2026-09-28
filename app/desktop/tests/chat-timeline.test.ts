import { expect, it } from 'vitest';
import type { Submission } from '@gobble/contracts';
import { emptyWorkspace } from '../src/main/workspace/model';
import { chatItems, groupViewActivity } from '../src/renderer/chat/timeline';

const submission = (id: string, createdAt: number): Submission => ({
  requestId: 'req_' + id,
  agentId: 'agt_' + id,
  text: id,
  model: 'fixture',
  effort: 'low',
  createdAt,
  responseStartedAt: null,
  state: 'running',
  threadId: id,
  turnId: id,
  response: '',
  problem: null,
});
it('projects delayed peer responses in receipt order while retaining request identity and original bodies', () => {
  const doc = emptyWorkspace('prj_one');
  const one = { ...submission('one', 10), responseStartedAt: 40, response: 'Later' };
  const two = { ...submission('two', 20), responseStartedAt: 30, response: 'Earlier' };
  doc.collaboration = { submissions: [one, two] };
  const items = chatItems(doc);
  expect(items.map((item) => item.id)).toEqual([
    'req_one',
    'req_two',
    'req_two:response',
    'req_one:response',
  ]);
  expect(items[2]).toMatchObject({ kind: 'response', submission: two });
  if (items[2]?.kind === 'response') expect(items[2].submission).toBe(two);
  two.response += ' streaming growth';
  expect(chatItems(doc).map((item) => item.id)).toEqual(items.map((item) => item.id));
});
it('retains legacy pairing and does not create a speculative waiting response', () => {
  const doc = emptyWorkspace('prj_one');
  const old = submission('old', 10);
  delete old.responseStartedAt;
  old.response = 'Legacy reply';
  doc.collaboration = { submissions: [old, submission('waiting', 20)] };
  expect(chatItems(doc).map((item) => item.kind)).toEqual(['submission', 'submission']);
});
it('resolves timestamp ties deterministically without changing records', () => {
  const doc = emptyWorkspace('prj_one');
  doc.collaboration = { submissions: [submission('b', 10), submission('a', 10)] };
  expect(chatItems(doc).map((item) => item.id)).toEqual(['req_a', 'req_b']);
  expect(doc.collaboration.submissions.map((item) => item.requestId)).toEqual(['req_b', 'req_a']);
});

it('groups adjacent routine view activity with a stable first anchor, preserving unknown problems and message boundaries', () => {
  const doc = emptyWorkspace('prj_one');
  doc.activity = [
    { id: 'a', at: 1, text: 'Opened a Project view.' },
    { id: 'b', at: 2, text: 'Pinned notes.txt.' },
    { id: 'c', at: 4, text: 'Moved notes.txt to the secondary pane.' },
    { id: 'd', at: 5, text: 'View failed to load.' },
    { id: 'e', at: 6, text: 'Opened a second pane.' },
    { id: 'f', at: 7, text: 'Combined the panes.' },
  ];
  doc.collaboration = { submissions: [submission('middle', 3)] };
  const source = chatItems(doc);
  const before = structuredClone(source);
  const grouped = groupViewActivity(source);
  expect(grouped.map((item) => [item.id, item.kind])).toEqual([
    ['a', 'activityGroup'],
    ['req_middle', 'submission'],
    ['c', 'activity'],
    ['d', 'activity'],
    ['e', 'activityGroup'],
  ]);
  expect(grouped[0]).toMatchObject({ at: 1, items: [doc.activity[0], doc.activity[1]] });
  expect(source).toEqual(before);
  expect(groupViewActivity(source)).toEqual(grouped);
});
