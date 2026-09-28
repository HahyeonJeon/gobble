// Test-only dynamic tool peer; no LLM claims or production hook.
const fs = require('node:fs');
const path = require('node:path');
module.exports = async ({ thread, turn, input, tool, resultText }) => {
  const home = process.env.CODEX_HOME;
  const wait = async (name) => {
    for (let i = 0; i < 1200; i++) {
      if (fs.existsSync(path.join(home, name))) return;
      await new Promise((resolve) => setTimeout(resolve, 100));
    }
    throw new Error('Pipeline test gate timed out');
  };
  await wait('pipeline-observe');
  const listed = resultText(await tool(thread.id, turn.id, 'workspace_list', {}));
  const surface = listed.surfaces.find((s) => s.view === 'pipeline');
  const observed = resultText(
    await tool(thread.id, turn.id, 'workspace_observe', { surfaceId: surface.surfaceId }),
  );
  if (!observed?.receipt.pointable) throw new Error('Pipeline observation unavailable');
  fs.writeFileSync(path.join(home, 'pipeline-observed.json'), JSON.stringify(observed));
  await wait('pipeline-point');
  const targets = observed.content.subjects.filter(
    (s) => s.target.selection.subject.kind === 'setting',
  );
  const results = [];
  for (const item of targets.slice(0, 1)) {
    const result = await tool(thread.id, turn.id, 'workspace_point', {
      evidence: item.target,
      observedReadId: observed.receipt.observedReadId,
      observation: observed.receipt.observation,
      note: 'Quality threshold is 25 Phred in this checked version. Your local selection stays independent.',
    });
    results.push(result);
    if (input[0].text === 'pipeline-stale' ? result.success : !result.success)
      throw new Error('Unexpected pipeline pointer authority: ' + JSON.stringify(result));
  }
  if (!results.length) throw new Error('No module settings returned');
  fs.writeFileSync(
    path.join(home, 'pipeline-delivery.json'),
    JSON.stringify({ input, observed, results }),
  );
};
