const fs = require('node:fs'),
  path = require('node:path');
/** Deterministic provider routing only. Real IPC, rendering, Notebook worker and captures run in the app. */
module.exports = async function notebookTools({ thread, turn, input, tool, resultText }) {
  const file = (name) => path.join(process.env.CODEX_HOME, name);
  const wait = async (name) => {
    while (!fs.existsSync(file(name))) await new Promise((resolve) => setTimeout(resolve, 25));
  };
  const record = (name, value) => fs.writeFileSync(file(name), JSON.stringify(value));
  await wait('notebook-gate');
  const listed = resultText(await tool(thread.id, turn.id, 'workspace_list', {}));
  const surface = listed.surfaces.find((s) => s.view === 'notebook');
  const read = async (args = {}) => {
    const response = await tool(thread.id, turn.id, 'workspace_observe', {
      surfaceId: surface.surfaceId,
      ...args,
    });
    const value = resultText(response);
    if (!value) throw new Error('Notebook observe failed: ' + JSON.stringify(response));
    return { response, value };
  };
  const point = (receipt, evidence, note = 'Review this Notebook selection.') =>
    tool(thread.id, turn.id, 'workspace_point', {
      evidence,
      observedReadId: receipt.observedReadId,
      observation: receipt.observation,
      note,
    });
  const visible = await read();
  record('notebook-observed.json', visible);
  const selection =
    input[0].text === 'notebook-image'
      ? visible.value.content.imagesAvailable[0]?.selection
      : visible.value.content.textParts.find((p) => p.evidence.selection.part.kind === 'source')
          ?.evidence.selection;
  if (!selection) throw new Error('No visible Notebook selection');
  if (input[0].text === 'notebook-scope') {
    const evidence = visible.value.content.textParts[0].evidence;
    await wait('notebook-next');
    const stale = await point(visible.value.receipt, evidence);
    const preview = await read({ selection, scope: 'source-preview' });
    const previewPoint = await point(preview.value.receipt, preview.value.receipt.evidence);
    const missing = await tool(thread.id, turn.id, 'workspace_point', {
      evidence,
      note: 'Missing receipt',
    });
    const current = await read();
    const freshEvidence = current.value.content.textParts[0].evidence;
    const fresh = await point(current.value.receipt, freshEvidence);
    record('notebook-delivery.json', {
      visible,
      current,
      stale,
      preview,
      previewPoint,
      missing,
      fresh,
    });
    return;
  }
  if (selection.selector.kind === 'image') {
    const r = selection.selector.rect;
    selection.selector.rect = {
      x: r.x,
      y: r.y,
      width: Math.min(120, r.width),
      height: Math.min(80, r.height),
    };
  }
  const selected = await read({ selection });
  const published = await point(selected.value.receipt, selected.value.receipt.evidence);
  if (!resultText(published)?.published)
    throw new Error('Notebook point failed: ' + JSON.stringify(published));
  const question = await tool(thread.id, turn.id, 'workspace_question', {
    question: 'Should we review this saved Notebook result?',
    options: ['Review this result', 'Choose another part'],
    evidenceIds: [selected.value.receipt.evidenceId],
  });
  if (!resultText(question))
    throw new Error('Notebook question failed: ' + JSON.stringify(question));
  record('notebook-delivery.json', { visible, selected, published, question });
};
