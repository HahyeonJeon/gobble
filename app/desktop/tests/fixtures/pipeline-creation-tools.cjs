// Deterministic peer. Main authorization and native Gobble checks are production paths.
const fs = require('node:fs'),
  path = require('node:path');
module.exports = async ({ thread, turn, input, tool, resultText }) => {
  const home = process.env.CODEX_HOME;
  if (input[0].text === 'pipeline-feedback') {
    const source = resultText(await tool(thread.id, turn.id, 'gobble_pipeline_source', {}));
    if (!source?.files) throw Error('Scoped Current source unavailable');
    const files = source.files.map((f) => ({
      ...f,
      content: f.content.replace('Length: 40', 'Length: 20'),
    }));
    if (JSON.stringify(files) === JSON.stringify(source.files))
      throw Error('Expected explicit length setting missing');
    const result = await tool(thread.id, turn.id, 'gobble_pipeline_propose', {
      summary: 'Allow shorter reads: change minimum length from 40 to 20 bp.',
      files,
    });
    if (!result.success) throw Error(JSON.stringify(result));
    const proposed = resultText(result);
    if (!proposed.followUp?.evidence?.length) throw Error('Run evidence provenance missing');
    fs.writeFileSync(
      path.join(home, 'feedback-delivery.json'),
      JSON.stringify({ input, proposed }),
    );
  } else if (input[0].text === 'pipeline-preparation-review') {
    const delivery = JSON.parse(
      input.find((i) => i.type === 'text' && i.text.includes('pipeline-discussion')).text,
    );
    const context = delivery.reference;
    if (!context.preparationId || delivery.facts.kind !== 'preparation')
      throw Error('Exact preparation missing');
    const sourceAttempt = await tool(thread.id, turn.id, 'gobble_pipeline_source', {});
    if (sourceAttempt.success) throw Error('Discussion authorized source editing');
    const premature = await tool(thread.id, turn.id, 'gobble_pipeline_point', {
      context,
      note: 'Premature',
    });
    if (premature.success) throw Error('Point without read');
    const facts = resultText(await tool(thread.id, turn.id, 'gobble_pipeline_review', { context }));
    if (!facts?.preparation || JSON.stringify(facts).includes('"payload"'))
      throw Error('Unsafe or missing preparation');
    const mark = await tool(thread.id, turn.id, 'gobble_pipeline_point', {
      context,
      note: 'This saved plan uses the reviewed quality threshold. Data contents will be checked before Start.',
    });
    if (!mark.success) throw Error(JSON.stringify(mark));
    fs.writeFileSync(
      path.join(home, 'preparation-review.json'),
      JSON.stringify({ delivery, facts, sourceAttempt, premature, mark }),
    );
  } else if (input[0].text === 'pipeline-creation') {
    const source = resultText(await tool(thread.id, turn.id, 'gobble_creation_source', {}));
    if (!source?.files) throw Error('Creation source unavailable');
    const result = await tool(thread.id, turn.id, 'gobble_creation_propose', {
      summary: 'Trim low-quality bases and inspect read quality',
      files: source.files.map((f) => ({
        ...f,
        content: f.content.replace('Quality: 0, Length: 0', 'Quality: 25, Length: 40'),
      })),
    });
    if (!result.success) throw Error(JSON.stringify(result));
    fs.writeFileSync(
      path.join(home, 'creation-delivery.json'),
      JSON.stringify({ input, source, result }),
    );
  } else {
    const delivery = input.find(
      (i) => i.type === 'text' && i.text.includes('pipeline-creation-discussion'),
    );
    const context = JSON.parse(delivery.text).reference;
    const sourceAttempt = await tool(thread.id, turn.id, 'gobble_creation_source', {});
    if (sourceAttempt.success) throw Error('Discussion authorized authoring');
    const beforeRead = await tool(thread.id, turn.id, 'gobble_creation_point', {
      context,
      note: 'Premature',
    });
    if (beforeRead.success) throw Error('Point without read');
    const facts = resultText(await tool(thread.id, turn.id, 'gobble_creation_review', { context }));
    if (!facts?.facts) throw Error('Creation facts unavailable');
    const mark = await tool(thread.id, turn.id, 'gobble_creation_point', {
      context,
      note: 'Quality threshold: 25 Phred for this proposed trimming step.',
    });
    if (!mark.success) throw Error(JSON.stringify(mark));
    fs.writeFileSync(
      path.join(home, 'creation-review.json'),
      JSON.stringify({ input, facts, sourceAttempt, beforeRead, mark }),
    );
  }
};
