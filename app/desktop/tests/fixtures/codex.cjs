#!/usr/bin/env node
// Test-local protocol peer. Never imported, packaged, or enabled by product code.
const fs = require('node:fs');
const path = require('node:path');
const readline = require('node:readline');
if (process.argv.includes('--version')) {
  process.stdout.write('codex-cli 0.153.4\n');
  process.exit(0);
}
const statePath = path.join(process.env.CODEX_HOME, 'fixture-state.json');
let state = fs.existsSync(statePath)
  ? JSON.parse(fs.readFileSync(statePath, 'utf8'))
  : { signedIn: false, threads: {}, count: 0 };
const save = () => fs.writeFileSync(statePath, JSON.stringify(state));
const send = (value) => process.stdout.write(JSON.stringify(value) + '\n');
const notify = (method, params) => send({ method, params });
const timers = new Map();
const callbacks = new Map();
let callbackId = 0;
const tool = (threadId, turnId, name, args) =>
  new Promise((resolve) => {
    const id = 'callback_' + ++callbackId;
    callbacks.set(id, resolve);
    send({
      id,
      method: 'item/tool/call',
      params: { threadId, turnId, callId: id, namespace: null, tool: name, arguments: args },
    });
  });
function resultText(result) {
  return result?.success
    ? JSON.parse(result.contentItems.find((item) => item.type === 'inputText').text)
    : null;
}
async function reportRead(thread, turn, input) {
  const attached = input
    .filter((i) => i.type === 'text' && i.text.startsWith('User-attached evidence'))
    .map((i) => JSON.parse(i.text.slice(i.text.indexOf('\n') + 1)))
    .find((i) => i.representation.kind === 'report');
  if (!attached) throw Error('No report in this message');
  if (JSON.stringify(attached).includes('base64'))
    throw Error('Initial report contains image bytes');
  const reading = resultText(
    await tool(thread.id, turn.id, 'read_report', { attachmentId: attached.attachmentId }),
  );
  if (!reading) throw Error('Report text unavailable');
  const charts = [];
  for (const module of reading.report.content.modules)
    for (const image of module.blocks.filter((b) => b.kind === 'image')) {
      const result = await tool(thread.id, turn.id, 'read_report', {
        attachmentId: attached.attachmentId,
        imageId: image.id,
      });
      if (!result?.success || !result.contentItems.some((i) => i.type === 'inputImage'))
        throw Error('Chart image unavailable');
      charts.push({ id: image.id, result });
    }
  const listed = resultText(await tool(thread.id, turn.id, 'workspace_list', {}));
  const surface = listed.surfaces.find((s) => s.view === 'report');
  let observed, pointed;
  if (surface) {
    observed = resultText(
      await tool(thread.id, turn.id, 'workspace_observe', { surfaceId: surface.surfaceId }),
    );
    if (!observed?.receipt?.observedReadId) throw Error('Report observation missing');
    pointed = resultText(
      await tool(thread.id, turn.id, 'workspace_point', {
        evidence: observed.receipt.evidence,
        observation: observed.receipt.observation,
        observedReadId: observed.receipt.observedReadId,
        note: 'Let us discuss this saved quality report.',
      }),
    );
    if (!pointed?.published) throw Error('Report pointer missing');
  }
  fs.writeFileSync(
    path.join(process.env.CODEX_HOME, 'report-delivery.json'),
    JSON.stringify({ attached, reading, charts, observed, pointed }),
  );
}
async function sharedPoint(thread, turn) {
  const listed = resultText(await tool(thread.id, turn.id, 'workspace_list', {}));
  if (!listed) throw new Error('List failed');
  const outputs = [];
  for (const surface of listed.surfaces) {
    const observationResult = await tool(thread.id, turn.id, 'workspace_observe', {
      surfaceId: surface.surfaceId,
    });
    const observed = resultText(observationResult);
    if (!observed) throw new Error('Observation failed: ' + JSON.stringify(observationResult));
    const evidence = observed.receipt.evidence;
    if (observed.image)
      evidence.selection = {
        coordinateSpace: 'normalized-original-image',
        kind: 'image',
        x: 0.2,
        y: 0.2,
        width: 0.5,
        height: 0.5,
        originalWidth: observed.image.originalWidth,
        originalHeight: observed.image.originalHeight,
        contentHash: observed.image.contentHash,
      };
    else if (observed.content.kind === 'table')
      evidence.selection = {
        coordinateSpace: 'revision-row-column-keys',
        kind: 'table',
        rowKeys: [observed.content.rows[2].key],
        columns: [observed.content.columns[0].id, observed.content.columns[1].id],
      };
    else
      evidence.selection = {
        coordinateSpace: 'utf16-line-column',
        kind: 'text',
        start: { line: 1, column: 0 },
        end: { line: 1, column: 7 },
      };
    const pointed = resultText(
      await tool(thread.id, turn.id, 'workspace_point', {
        evidence,
        note: 'Fixture pointer from ' + thread.name,
      }),
    );
    if (!pointed) throw new Error('Point failed');
    outputs.push(pointed.referenceId);
  }
  return outputs;
}
async function plotPoint(thread, turn, input) {
  const listed = resultText(await tool(thread.id, turn.id, 'workspace_list', {}));
  const plot = listed.surfaces.find((surface) => surface.view === 'scatter');
  const observed = resultText(
    await tool(thread.id, turn.id, 'workspace_observe', {
      surfaceId: plot.surfaceId,
      scope: 'source-preview',
    }),
  );
  if (!observed) throw new Error('Plot observation failed');
  const row = observed.content.rows[9];
  const evidence = {
    ...observed.receipt.evidence,
    selection: {
      kind: 'table',
      coordinateSpace: 'revision-row-column-keys',
      rowKeys: [row.key],
      columns: observed.content.columns.map((column) => column.id),
    },
  };
  const result = resultText(
    await tool(thread.id, turn.id, 'workspace_point', {
      evidence,
      observation: observed.receipt.observation,
      note: 'Inspect sample S10 at X = 9, Y = 11. This pointer leaves your selection unchanged.',
    }),
  );
  if (!result?.published) throw new Error('Plot pointer failed');
  fs.writeFileSync(
    path.join(process.env.CODEX_HOME, 'plot-delivery.json'),
    JSON.stringify({ input, observed, result }),
  );
}
async function continuationPoint(thread, turn) {
  const listed = resultText(await tool(thread.id, turn.id, 'workspace_list', {}));
  const surface = listed.surfaces.find((item) => item.view === 'run');
  const observed = resultText(
    await tool(thread.id, turn.id, 'workspace_observe', {
      surfaceId: surface.surfaceId,
      scope: 'view',
      selection: {
        kind: 'run-task',
        coordinateSpace: 'observed-instance-attempt',
        instanceId: 'qc',
        attempt: 1,
      },
    }),
  );
  if (!observed?.content.continuation) throw new Error('Shared review context missing');
  const result = resultText(
    await tool(thread.id, turn.id, 'workspace_point', {
      evidence: observed.receipt.evidence,
      observedReadId: observed.receipt.observedReadId,
      observation: observed.receipt.observation,
      note: 'This checked step restarts from its beginning.',
    }),
  );
  if (!result?.published) throw new Error('Continuation pointer was refused');
  fs.writeFileSync(
    path.join(process.env.CODEX_HOME, 'continuation-delivery.json'),
    JSON.stringify(observed),
  );
}
async function observedPoint(thread, turn, input) {
  const gate = path.join(process.env.CODEX_HOME, 'observed-gate');
  while (!fs.existsSync(gate)) await new Promise((resolve) => setTimeout(resolve, 25));
  const listed = resultText(await tool(thread.id, turn.id, 'workspace_list', {}));
  const receipts = [];
  for (const kind of ['run', 'log']) {
    const surface = listed.surfaces.find((item) => item.view === kind);
    const selection =
      kind === 'run'
        ? {
            kind: 'run-task',
            coordinateSpace: 'observed-instance-attempt',
            instanceId: 'prepare',
            attempt: 1,
          }
        : {
            kind: 'log-text',
            coordinateSpace: 'decoded-preview-utf16-line-column',
            stream: 'stderr',
            start: { line: 1, column: 0 },
            end: { line: 1, column: 14 },
          };
    const response = await tool(thread.id, turn.id, 'workspace_observe', {
      surfaceId: surface.surfaceId,
      scope: 'view',
      selection,
    });
    const observed = resultText(response);
    if (!observed) throw new Error('Observed read failed: ' + JSON.stringify(response));
    const responsePoint = await tool(thread.id, turn.id, 'workspace_point', {
      evidence: observed.receipt.evidence,
      observedReadId: observed.receipt.observedReadId,
      observation: observed.receipt.observation,
      note:
        kind === 'run'
          ? 'Check the preparation attempt.'
          : 'The output says ERROR 😀 exact. Your stderr selection is unchanged.',
    });
    const result = resultText(responsePoint);
    if (!result?.published)
      throw new Error('Observed point failed: ' + JSON.stringify(responsePoint));
    receipts.push({ observed, result });
  }
  fs.writeFileSync(
    path.join(process.env.CODEX_HOME, 'observed-delivery.json'),
    JSON.stringify({ input, receipts }),
  );
}
async function dependencyPoint(thread, turn, input) {
  const gate = path.join(process.env.CODEX_HOME, 'dependency-gate');
  while (!fs.existsSync(gate)) await new Promise((resolve) => setTimeout(resolve, 25));
  const listed = resultText(await tool(thread.id, turn.id, 'workspace_list', {}));
  const surface = listed.surfaces.find((item) => item.view === 'run');
  const observed = resultText(
    await tool(thread.id, turn.id, 'workspace_observe', {
      surfaceId: surface.surfaceId,
      representation: 'dependencies',
    }),
  );
  if (!observed?.receipt.pointable) throw new Error('Dependencies were not pointable');
  const group = observed.content.groups.find((item) => item.taskId === 'align').target;
  const edge = observed.content.edges[0].target;
  const selected = resultText(
    await tool(thread.id, turn.id, 'workspace_observe', {
      surfaceId: surface.surfaceId,
      selection: group.selection,
    }),
  );
  if (!selected?.receipt.evidenceId)
    throw new Error('Selected dependency has no question evidence');
  const points = [];
  for (const evidence of [group, edge]) {
    const result = resultText(
      await tool(thread.id, turn.id, 'workspace_point', {
        evidence,
        observedReadId: observed.receipt.observedReadId,
        observation: observed.receipt.observation,
        note:
          evidence.selection.kind === 'run-group'
            ? 'Review the align group.'
            : 'Review prepare to align.',
      }),
    );
    if (!result?.published) throw new Error('Dependency pointer failed: ' + JSON.stringify(result));
    points.push(result);
  }
  fs.writeFileSync(
    path.join(process.env.CODEX_HOME, 'dependency-delivery.json'),
    JSON.stringify({ input, observed, selected, points }),
  );
}
async function pdfOpen(thread, turn) {
  const listing = resultText(await tool(thread.id, turn.id, 'workspace_resources', {}));
  const entry = listing.entries.find((e) => e.name === 'report.pdf');
  const result = await tool(thread.id, turn.id, 'workspace_open', {
    resources: [{ kind: 'file', resourceId: entry.resourceId }],
  });
  const opened = resultText(result);
  if (!opened) throw new Error('Open PDF failed: ' + JSON.stringify(result));
  fs.writeFileSync(path.join(process.env.CODEX_HOME, 'pdf-open.json'), JSON.stringify(opened));
}
async function pdfPoint(thread, turn, input) {
  const wait = async (name) => {
    while (!fs.existsSync(path.join(process.env.CODEX_HOME, name)))
      await new Promise((resolve) => setTimeout(resolve, 25));
  };
  const record = (name, data) =>
    fs.writeFileSync(path.join(process.env.CODEX_HOME, name), JSON.stringify(data));
  await wait('pdf-gate');
  const listed = resultText(await tool(thread.id, turn.id, 'workspace_list', {}));
  const surface =
    listed.surfaces.find((s) => s.view === 'pdf' && s.presentation === 'ready') ||
    listed.surfaces.find((s) => s.view === 'pdf');
  const observe = async (args = {}) => {
    const response = await tool(thread.id, turn.id, 'workspace_observe', {
      surfaceId: surface.surfaceId,
      ...args,
    });
    const value = resultText(response);
    if (!value) throw new Error('PDF observation failed: ' + JSON.stringify(response));
    return { value, response };
  };
  const point = (receipt, evidence = receipt.evidence, note = 'Review this PDF region.') =>
    tool(thread.id, turn.id, 'workspace_point', {
      evidence,
      observation: receipt.observation,
      observedReadId: receipt.observedReadId,
      note,
    });
  const visible = await observe();
  record('pdf-observed.json', visible);
  if (input[0].text === 'pdf-scope') {
    const original = visible.value.receipt;
    await wait('pdf-next');
    const stale = await point(original);
    const current = await observe();
    const source = await observe({ scope: 'source-preview' });
    const sourcePoint = await point(source.value.receipt);
    const offPage = await tool(thread.id, turn.id, 'workspace_observe', {
      surfaceId: surface.surfaceId,
      selection: { ...current.value.receipt.evidence.selection, pageIndex: 3 },
    });
    const outside = await point(current.value.receipt, {
      ...current.value.receipt.evidence,
      selection: {
        ...current.value.receipt.evidence.selection,
        region: current.value.content.viewBox,
        scope: 'page',
      },
    });
    const missingReceipt = await tool(thread.id, turn.id, 'workspace_point', {
      evidence: current.value.receipt.evidence,
      note: 'Missing read.',
    });
    const fresh = await point(current.value.receipt);
    record('pdf-delivery.json', {
      visible,
      current,
      source,
      stale,
      sourcePoint,
      offPage,
      outside,
      missingReceipt,
      fresh,
    });
    if (!resultText(fresh)?.published)
      throw new Error('Fresh PDF point failed: ' + JSON.stringify(fresh));
    return;
  }
  const [x, y, r, t] = visible.value.receipt.evidence.selection.region;
  const selection = {
    ...visible.value.receipt.evidence.selection,
    scope: 'region',
    region: [x + (r - x) * 0.2, y + (t - y) * 0.2, x + (r - x) * 0.8, y + (t - y) * 0.8],
  };
  const selected = await observe({ selection });
  const published = await point(selected.value.receipt);
  if (!resultText(published)?.published)
    throw new Error('PDF point failed: ' + JSON.stringify(published));
  const question = await tool(thread.id, turn.id, 'workspace_question', {
    question: 'Is this the PDF region to review?',
    options: ['Review this region', 'Choose another region'],
    evidenceIds: [selected.value.receipt.evidenceId],
  });
  if (!resultText(question)) throw new Error('PDF question failed: ' + JSON.stringify(question));
  record('pdf-delivery.json', { visible, selected, published, question });
}
async function askQuestion(thread, turn) {
  const gate = path.join(process.env.CODEX_HOME, 'question-gate');
  while (gate && !fs.existsSync(gate)) await new Promise((resolve) => setTimeout(resolve, 25));
  const listed = resultText(await tool(thread.id, turn.id, 'workspace_list', {}));
  if (!listed) throw new Error('List failed');
  const evidenceIds = [];
  for (const surface of listed.surfaces) {
    const result = await tool(thread.id, turn.id, 'workspace_observe', {
      surfaceId: surface.surfaceId,
    });
    const observed = resultText(result);
    if (!observed?.receipt.evidenceId)
      throw new Error('Question observation failed: ' + JSON.stringify(result));
    evidenceIds.push(observed.receipt.evidenceId);
  }
  const question = resultText(
    await tool(thread.id, turn.id, 'workspace_question', {
      question: 'Should we keep sample S03 for this review?',
      options: ['Keep S03', 'Review S03 first'],
      evidenceIds,
    }),
  );
  if (!question) throw new Error('Question creation failed');
  thread.questions = [...(thread.questions ?? []), question];
  save();
}
const threadResult = (thread) => ({
  thread,
  model: thread.model,
  modelProvider: 'openai',
  approvalPolicy: 'never',
  sandbox: { type: 'readOnly', networkAccess: false },
});
const finish = (threadId, turn, status) => {
  clearTimeout(timers.get(turn.id));
  timers.delete(turn.id);
  if (status === 'completed') {
    const item = {
      type: 'agentMessage',
      id: 'answer_' + turn.id,
      text: 'Fixture response from ' + state.threads[threadId].name + '.',
      phase: 'final_answer',
    };
    turn.items.push(item);
    notify('item/completed', { threadId, turnId: turn.id, item });
  }
  turn.status = status;
  save();
  notify('turn/completed', { threadId, turn });
};
readline.createInterface({ input: process.stdin }).on('line', (line) => {
  const message = JSON.parse(line);
  const p = message.params ?? {};
  if (!message.method) {
    const callback = callbacks.get(message.id);
    if (callback) {
      callbacks.delete(message.id);
      callback(message.result ?? { success: false, error: message.error });
    }
    return;
  }
  if (!('id' in message)) return;
  const reply = (result) => send({ id: message.id, result });
  switch (message.method) {
    case 'initialize':
      reply({ userAgent: 'gobble-test-fixture' });
      break;
    case 'account/read':
      reply({
        account: state.signedIn
          ? { type: 'chatgpt', email: 'fixture@example.test', planType: 'test' }
          : null,
        requiresOpenaiAuth: true,
      });
      break;
    case 'account/login/start':
      reply({
        type: 'chatgpt',
        loginId: 'fixture-login',
        authUrl: 'https://auth.openai.com/authorize?fixture=true',
      });
      timers.set(
        'login',
        setTimeout(() => {
          timers.delete('login');
          state.signedIn = true;
          save();
          notify('account/login/completed', {
            loginId: 'fixture-login',
            success: true,
            error: null,
          });
          notify('account/updated', { authMode: 'chatgpt', planType: 'test' });
        }, 150),
      );
      break;
    case 'account/login/cancel':
      clearTimeout(timers.get('login'));
      timers.delete('login');
      reply({});
      break;
    case 'account/logout':
      state.signedIn = false;
      save();
      reply({});
      notify('account/updated', { authMode: null, planType: null });
      break;
    case 'model/list':
      setTimeout(
        () =>
          reply({
            data: [
              {
                model: 'fixture-model',
                displayName: 'Fixture model',
                hidden: false,
                isDefault: true,
                inputModalities: ['text', 'image'],
                supportedReasoningEfforts: [
                  { reasoningEffort: 'low' },
                  { reasoningEffort: 'high' },
                ],
                defaultReasoningEffort: 'low',
              },
              {
                model: 'fixture-text-model',
                displayName: 'Fixture text model',
                hidden: false,
                isDefault: false,
                inputModalities: ['text'],
                supportedReasoningEfforts: [{ reasoningEffort: 'low' }],
                defaultReasoningEffort: 'low',
              },
            ],
            nextCursor: null,
          }),
        state.modelDelay ?? 0,
      );
      break;
    case 'thread/start': {
      if (
        p.sandbox !== 'read-only' ||
        p.approvalPolicy !== 'never' ||
        p.environments?.length !== 0 ||
        !Array.isArray(p.dynamicTools) ||
        p.dynamicTools.some(
          (tool) =>
            !tool.name.startsWith('workspace_') &&
            !tool.name.startsWith('gobble_') &&
            tool.name !== 'read_report',
        )
      ) {
        send({ id: message.id, error: { code: -32600, message: 'Policy mismatch' } });
        break;
      }
      const thread = {
        id: 'thread_' + ++state.count,
        model: p.model,
        tools: p.dynamicTools,
        name: p.developerInstructions.match(/Your name: (.*)/)?.[1] ?? 'Agent',
        turns: [],
      };
      state.threads[thread.id] = thread;
      save();
      reply(threadResult(thread));
      break;
    }
    case 'thread/resume':
      reply(threadResult(state.threads[p.threadId]));
      break;
    case 'thread/read':
      reply({ thread: state.threads[p.threadId] });
      break;
    case 'turn/start': {
      const thread = state.threads[p.threadId];
      const turn = {
        id: 'turn_' + ++state.count,
        status: 'inProgress',
        items: [
          {
            type: 'userMessage',
            id: 'user_' + state.count,
            clientId: p.clientUserMessageId,
            content: p.input,
          },
        ],
      };
      thread.turns.push(turn);
      save();
      if (p.input[0].text === 'disconnect-after-accept') {
        turn.status = 'completed';
        turn.items.push({
          type: 'agentMessage',
          id: 'recovered',
          text: 'Recovered fixture response.',
        });
        save();
        process.exit(0);
      }
      notify('turn/started', { threadId: thread.id, turn });
      reply({ turn });
      if (p.input[0].text.startsWith('controlled:')) {
        // Deterministic external peer timing; no test-only API enters the app.
        let consumed = 0;
        const item = { type: 'agentMessage', id: 'answer_' + turn.id, text: '' };
        turn.items.push(item);
        timers.set(
          turn.id,
          setInterval(() => {
            const control = path.join(process.env.CODEX_HOME, 'stream-control.json');
            let chunks;
            try {
              chunks = JSON.parse(fs.readFileSync(control, 'utf8'))[thread.name] ?? [];
            } catch {
              return;
            }
            for (; consumed < chunks.length; consumed++) {
              item.text += chunks[consumed];
              notify('item/agentMessage/delta', {
                threadId: thread.id,
                turnId: turn.id,
                itemId: item.id,
                delta: chunks[consumed],
              });
            }
            save();
          }, 25),
        );
        break;
      }
      notify('item/agentMessage/delta', {
        threadId: thread.id,
        turnId: turn.id,
        itemId: 'answer_' + turn.id,
        delta: 'Fixture response from ' + thread.name,
      });
      if (p.input[0].text === 'ask-question') {
        void askQuestion(thread, turn)
          .then(() => finish(thread.id, turn, 'completed'))
          .catch((error) => {
            const item = {
              type: 'agentMessage',
              id: 'question_failure_' + turn.id,
              text: 'Fixture question failed: ' + error.message,
            };
            turn.items.push(item);
            notify('item/completed', { threadId: thread.id, turnId: turn.id, item });
            finish(thread.id, turn, 'failed');
          });
        break;
      }
      if (
        [
          'shared-point',
          'report-read',
          'pipeline-creation',
          'pipeline-creation-review',
          'pipeline-preparation-review',
          'pipeline-feedback',
          'pipeline-proposal',
          'pipeline-review',
          'pipeline-point',
          'pipeline-stale',
          'plot-point',
          'observed-point',
          'continuation-point',
          'dependency-point',
          'notebook-point',
          'notebook-image',
          'notebook-scope',
          'pdf-open',
          'pdf-point',
          'pdf-scope',
        ].includes(p.input[0].text)
      ) {
        void (
          p.input[0].text === 'report-read'
            ? reportRead(thread, turn, p.input)
            : p.input[0].text === 'continuation-point'
              ? continuationPoint(thread, turn)
              : p.input[0].text.startsWith('pipeline-')
                ? require('./pipeline-tools.cjs')({
                    thread,
                    turn,
                    input: p.input,
                    tool,
                    resultText,
                  })
                : p.input[0].text.startsWith('notebook-')
                  ? require('./notebook-tools.cjs')({
                      thread,
                      turn,
                      input: p.input,
                      tool,
                      resultText,
                    })
                  : p.input[0].text === 'pdf-open'
                    ? pdfOpen(thread, turn)
                    : p.input[0].text.startsWith('pdf-')
                      ? pdfPoint(thread, turn, p.input)
                      : p.input[0].text === 'dependency-point'
                        ? dependencyPoint(thread, turn, p.input)
                        : p.input[0].text === 'observed-point'
                          ? observedPoint(thread, turn, p.input)
                          : p.input[0].text === 'plot-point'
                            ? plotPoint(thread, turn, p.input)
                            : sharedPoint(thread, turn)
        )
          .then(() => finish(thread.id, turn, 'completed'))
          .catch((error) => {
            const item = {
              type: 'agentMessage',
              id: 'failure_' + turn.id,
              text: 'Fixture shared tool failed: ' + error.message,
            };
            turn.items.push(item);
            notify('item/completed', { threadId: thread.id, turnId: turn.id, item });
            finish(thread.id, turn, 'failed');
          });
        break;
      }
      if (!p.input[0].text.includes('wait'))
        timers.set(
          turn.id,
          setTimeout(() => finish(thread.id, turn, 'completed'), 200),
        );
      break;
    }
    case 'turn/interrupt': {
      const turn = state.threads[p.threadId].turns.find((turn) => turn.id === p.turnId);
      reply({});
      if (turn.status === 'inProgress') finish(p.threadId, turn, 'interrupted');
      break;
    }
    case 'fixture/model-delay':
      state.modelDelay = p.ms;
      reply({});
      break;
    case 'fixture/tool': {
      const id = 'fixture_callback_' + ++callbackId;
      callbacks.set(id, (result) => reply(result));
      send({ id, method: p.method ?? 'item/tool/call', params: p.params });
      break;
    }
    case 'fixture/no-response':
      break;
    case 'fixture/malformed':
      process.stdout.write('{bad\n');
      break;
    case 'fixture/request':
      send({ id: 'server-request', method: 'command/exec', params: { command: 'forbidden' } });
      reply({});
      break;
    default:
      send({ id: message.id, error: { code: -32601, message: 'Unknown fixture method' } });
  }
});
process.stdin.on('end', () => {
  for (const timer of timers.values()) clearTimeout(timer);
  process.exit(0);
});
