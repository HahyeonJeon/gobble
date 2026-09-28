// Deterministic test peer: actual Main capability, source service, Gobble check,
// immutable comparison, and User adoption stay production paths. No LLM claim.
const fs = require('node:fs');
const path = require('node:path');
module.exports = async ({ thread, turn, input, tool, resultText }) => {
  const home = process.env.CODEX_HOME;
  if (input[0].text === 'pipeline-proposal') {
    const source = resultText(await tool(thread.id, turn.id, 'gobble_pipeline_source', {}));
    if (!source) throw new Error('Scoped source unavailable');
    const file = source.files.find((f) => f.path.endsWith('pipeline.go'));
    const content = file.content
      .replace('Quality: 25', 'Quality: 30')
      .replace('import (', 'import (\n "github.com/HahyeonJeon/gobble/assets/modules/fastqc"')
      .replace(
        'return p',
        'if _,err:=fastqc.Add(modules.WithDisplay(p,gobble.TaskDisplay{Stage:"Inspect trimmed read quality"}),trimmed.Read1,fastqc.Options{});err!=nil{panic(err)}\n return p',
      );
    const result = await tool(thread.id, turn.id, 'gobble_pipeline_propose', {
      summary:
        'Raise the quality threshold from 25 to 30 Phred and add a quality check for the trimmed reads.',
      files: [{ path: file.path, content }],
    });
    if (!result.success) throw new Error('Proposal failed: ' + JSON.stringify(result));
    fs.writeFileSync(
      path.join(home, 'proposal-delivery.json'),
      JSON.stringify({ input, source, proposal: resultText(result) }),
    );
  } else {
    const delivery = input.find((i) => i.type === 'text' && i.text.includes('pipeline-discussion'));
    const context = JSON.parse(delivery.text).reference;
    const sourceAttempt = await tool(thread.id, turn.id, 'gobble_pipeline_source', {});
    if (sourceAttempt.success) throw new Error('Discussion unexpectedly authorized source');
    const facts = resultText(await tool(thread.id, turn.id, 'gobble_pipeline_review', { context }));
    if (!facts?.change) throw new Error('Exact change unavailable');
    const mark = await tool(thread.id, turn.id, 'gobble_pipeline_point', {
      context,
      note: 'This changes the quality threshold from 25 to 30 Phred.',
    });
    if (!mark.success) throw new Error('Comparison reference refused: ' + JSON.stringify(mark));
    fs.writeFileSync(
      path.join(home, 'review-delivery.json'),
      JSON.stringify({ input, facts, sourceAttempt, mark }),
    );
  }
};
