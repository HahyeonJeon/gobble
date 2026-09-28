// Local design simulation. No filesystem, service, provider or execution integration.
const $ = (id) => document.getElementById(id);
const state = {
  scenario: 'ready',
  concept: 'single',
  side: 'proposed',
  selected: 'quality',
  list: false,
  adopted: false,
  proposal: 3,
  attachments: [],
};
const html = (s) =>
  String(s).replace(
    /[&<>"']/g,
    (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' })[c],
  );
const changes = {
  quality: {
    title: 'Quality threshold',
    step: 'Trim adapters',
    before: '25 Phred',
    after: () => (state.proposal === 4 ? '28 Phred' : '30 Phred'),
  },
  report: {
    title: 'Quality report',
    step: 'Check read quality',
    before: 'Not present',
    after: () => 'Added after trimming',
  },
};
const values = () => ({ number: state.proposal, value: state.proposal === 4 ? 28 : 30 });
let toastTimer;
function toast(text) {
  $('toast').textContent = text;
  $('toast').hidden = false;
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => ($('toast').hidden = true), 4500);
}
function focusChat() {
  document.querySelector('.shell').dataset.mobile = 'chat';
  document
    .querySelectorAll('button[data-mobile]')
    .forEach((b) => b.setAttribute('aria-pressed', String(b.dataset.mobile === 'chat')));
  $('draft').focus();
}
function switchWorkspace() {
  document.querySelector('.shell').dataset.mobile = 'work';
  document
    .querySelectorAll('button[data-mobile]')
    .forEach((b) => b.setAttribute('aria-pressed', String(b.dataset.mobile === 'work')));
}
function diagram(side) {
  const proposed = side === 'proposed';
  const nodes = [
    {
      id: 'input',
      name: 'Input reads',
      sub: 'sample reads · FASTQ',
      x: 5,
      y: 16,
      kind: 'input',
      icon: '▤',
    },
    {
      id: 'quality',
      name: 'Trim adapters',
      sub: `Quality · ${proposed ? values().value : 25} Phred`,
      x: 38,
      y: 16,
      kind: proposed ? 'changed' : '',
      icon: '⚙',
    },
    {
      id: 'align',
      name: 'Align reads',
      sub: 'Existing input preserved',
      x: 71,
      y: 16,
      kind: '',
      icon: '◈',
    },
  ];
  if (proposed)
    nodes.push({
      id: 'report',
      name: 'Check read quality',
      sub: 'FastQC · quality report',
      x: 71,
      y: 63,
      kind: 'added',
      icon: '▥',
    });
  return `<div class="diagram" aria-label="${proposed ? 'Proposed' : 'Current'} flow"><span class="version-label">${proposed ? (state.adopted ? 'ACCEPTED · VERSION ' + (state.proposal === 4 ? 4 : 3) : 'PROPOSED · ' + state.proposal) : state.adopted ? 'PREVIOUS · VERSION 2' : 'BASE · VERSION 2'}</span><svg viewBox="0 0 1000 345" preserveAspectRatio="none" aria-hidden="true"><defs><marker id="arrow-${side}" viewBox="0 0 10 10" refX="8" refY="5" markerWidth="5" markerHeight="5" orient="auto-start-reverse"><path d="M0 0L10 5L0 10Z" fill="#80968a"/></marker><marker id="new-${side}" viewBox="0 0 10 10" refX="8" refY="5" markerWidth="5" markerHeight="5" orient="auto-start-reverse"><path d="M0 0L10 5L0 10Z" fill="#598d64"/></marker></defs><path d="M270 100H380" stroke="#80968a" stroke-width="2" fill="none" marker-end="url(#arrow-${side})"/><path d="M600 100H710" stroke="#80968a" stroke-width="2" fill="none" marker-end="url(#arrow-${side})"/>${proposed ? `<path d="M600 100H646Q656 100 656 110V253Q656 263 666 263H710" stroke="#598d64" stroke-width="2.4" fill="none" marker-end="url(#new-${side})"/><text x="582" y="320" fill="#65816e" font-size="12">Trimmed reads → quality report</text>` : ''}</svg>${nodes.map((n) => `<button class="node ${n.kind} ${state.selected === n.id ? 'is-selected' : ''}" data-subject="${n.id}" data-side="${side}" style="left:${n.x}%;top:${n.y}%" aria-label="Inspect ${html(n.name)} in ${proposed ? 'Proposed' : 'Current'} flow" aria-pressed="${state.selected === n.id}">${n.kind === 'changed' ? '<em>Changed</em>' : n.kind === 'added' ? '<em>＋ Added</em>' : ''}<strong><b aria-hidden="true">${n.icon}</b>${n.name}</strong><small>${n.sub}</small></button>`).join('')}</div>`;
}
function details() {
  if (state.selected === 'quality')
    return `<span class="eyebrow">CHANGED SETTING · TRIM ADAPTERS</span><h3>Quality threshold</h3><div class="value-comparison"><div class="value-box"><small>${state.adopted ? 'Previous' : 'Base'}</small><strong>25 <span>Phred</span></strong></div><span aria-hidden="true">→</span><div class="value-box proposed"><small>${state.adopted ? 'Accepted' : 'Proposed'}</small><strong>${values().value} <span>Phred</span></strong></div></div><p>Minimum length stays at 40 bp. Alignment keeps the same trimmed-read input.</p>`;
  if (state.selected === 'report')
    return `<span class="eyebrow">ADDED STEP · PROPOSED FLOW</span><h3>Check read quality</h3><dl><dt>Tool</dt><dd>FastQC</dd><dt>Input</dt><dd>Trim adapters → trimmed reads</dd><dt>Output</dt><dd>Quality report (declared)</dd></dl><p>A parallel branch. Alignment does not wait for this report.</p>`;
  if (state.selected === 'unknown')
    return '<span class="eyebrow">UNEXPLAINED PROCESSING CHANGE</span><h3>More review support is needed</h3><p>An additional processing change cannot yet be described by this comparison. Ask the Agent to refine the proposal. Acceptance is unavailable.</p>';
  return `<span class="eyebrow">UNCHANGED · ${state.side.toUpperCase()} FLOW</span><h3>${state.selected === 'align' ? 'Align reads' : 'Input reads'}</h3><p>${state.selected === 'align' ? 'Alignment receives the same output from Trim adapters. Its tool and settings are unchanged in this illustrative comparison.' : 'The declared FASTQ input is unchanged. Selecting an input does not select or copy the dataset contents.'}</p>`;
}
function render() {
  const unavailable = ['checking', 'invalid', 'new'].includes(state.scenario),
    blocked = !['ready', 'revised', 'imported'].includes(state.scenario);
  $('revision-line').textContent = state.adopted
    ? `ACCEPTED PROPOSAL ${state.proposal} · REVIEW HISTORY`
    : `PROPOSAL ${state.proposal} · BASED ON VERSION 2`;
  document.querySelector('.surfacebar>span:first-child').textContent =
    state.scenario === 'new' ? '＋ New pipeline' : '◈ RNA quality';
  document.querySelector('.review-tab').textContent =
    state.scenario === 'new'
      ? 'Plan analysis'
      : state.adopted
        ? 'Review history'
        : 'Review proposal';
  $('proposal-title').textContent =
    state.scenario === 'new'
      ? 'Start with your analysis goal'
      : 'Refine trimming and add a quality report';
  $('status').textContent = state.adopted
    ? 'Now current'
    : {
        ready: 'Ready to review',
        imported: 'First Project version',
        revised: 'Ready to review',
        incomplete: 'Needs review support',
        conflict: 'Current version changed',
        checking: 'Checking…',
        invalid: 'Needs attention',
        interrupted: 'Checking adoption',
        new: 'No pipeline yet',
      }[state.scenario];
  $('status').classList.toggle('warning', blocked);
  const messages = {
    imported:
      'Save a Project version in Gobble. Your imported files remain separate. This proposal becomes the version Gobble uses; your data stays in its Project location.',
    incomplete:
      'One processing change is not explained by the comparison. Review support is needed before this proposal can become current.',
    conflict:
      'This proposal was based on version 2. The pipeline is now version 3. Ask for an updated proposal; the earlier comparison stays available.',
    checking: 'Gobble is checking proposal 3. Your current flow and conversation remain available.',
    invalid:
      'The proposed quality check is missing a required input. Ask the Agent to revise it. Your current pipeline is unchanged.',
    interrupted:
      'The App closed during adoption. Resolve the recorded outcome before trying again. The proposal and the previous version are retained.',
  };
  $('notice').textContent = state.adopted
    ? 'Proposal ' + state.proposal + ' is now current in this preview. Analysis has not started.'
    : messages[state.scenario] || '';
  $('notice').hidden = !$('notice').textContent;
  $('notice').classList.toggle('success', state.adopted);
  const side = unavailable ? 'current' : state.side;
  $('current').textContent = state.adopted
    ? 'Previous'
    : state.scenario === 'conflict'
      ? 'Base v2'
      : 'Current';
  $('proposed').textContent = state.adopted ? 'Accepted' : 'Proposed';
  $('current').setAttribute('aria-pressed', String(side === 'current'));
  $('proposed').setAttribute('aria-pressed', String(side === 'proposed'));
  $('proposed').disabled = unavailable;
  document.querySelector('.flowbar').hidden = state.scenario === 'new';
  document.querySelector('.lower').hidden = state.scenario === 'new';
  document.querySelector('.legend').hidden = state.scenario === 'new';
  document.querySelector('.decision').hidden = state.scenario === 'new';
  $('revision-line').hidden = state.scenario === 'new';
  $('flow-context').textContent =
    side === 'proposed'
      ? '3 steps · 1 input · 1 added · 1 setting changed'
      : '2 steps · 1 input · original flow';
  $('list-toggle').setAttribute('aria-pressed', String(state.list));
  if (state.scenario === 'new')
    $('canvas').innerHTML =
      '<section class="onboarding"><span class="eyebrow">NEW PIPELINE</span><h2>What would you like to analyze?</h2><p>Describe the outcome in Chat and choose the data to discuss. Your Agent will prepare a pipeline for visual review.</p><button id="choose-data">▤ Choose example reads</button><button id="describe-goal" class="primary">Describe the analysis in Chat →</button><p style="margin-top:15px;font-size:11px">Preview uses illustrative data. No files are accessed.</p></section>';
  else if (state.list)
    $('canvas').innerHTML =
      '<div class="step-list">' +
      [
        ['input', 'Input reads', 'Unchanged'],
        [
          'quality',
          'Trim adapters',
          side === 'proposed'
            ? `Quality threshold · 25 → ${values().value} Phred`
            : 'Quality threshold · 25 Phred',
        ],
        ['align', 'Align reads', 'From Trim adapters · unchanged'],
        ...(side === 'proposed'
          ? [['report', 'Check read quality', 'Added · from Trim adapters']]
          : []),
      ]
        .map(
          ([id, name, note]) =>
            `<button class="step-row" data-subject="${id}" data-side="${side}"><strong>${name}</strong><small>${note}</small></button>`,
        )
        .join('') +
      '</div>';
  else
    $('canvas').innerHTML =
      state.concept === 'paired' && !unavailable
        ? `<div class="paired">${diagram('current')}${diagram('proposed')}</div>`
        : diagram(side);
  $('changes-title').innerHTML =
    `Changes <span>${unavailable ? '—' : state.scenario === 'incomplete' ? 3 : 2}</span>`;
  $('change-list').innerHTML = unavailable
    ? '<p class="muted" style="padding:8px;font-size:12px;line-height:1.6">A checked proposal is needed to compare changes.</p>'
    : [
        ['quality', '~', 'Quality threshold', `25 → ${values().value} Phred`, ''],
        ['report', '+', 'Quality report', 'New step and connection', 'add'],
        ...(state.scenario === 'incomplete'
          ? [['unknown', '!', 'Processing behavior', 'Cannot compare yet', '']]
          : []),
      ]
        .map(
          ([id, icon, title, note, cls]) =>
            `<button class="change ${cls} ${state.selected === id ? 'active' : ''}" data-change="${id}" aria-pressed="${state.selected === id}"><span aria-hidden="true">${icon}</span><span><strong>${title}</strong><small>${note}</small></span></button>`,
        )
        .join('');
  $('details').innerHTML = unavailable
    ? '<span class="eyebrow">CURRENT PIPELINE PRESERVED</span><h3>Keep discussing the analysis</h3><p>The new comparison will appear here when the proposal is checked.</p>'
    : details();
  $('attach').disabled = unavailable;
  $('accept').disabled = blocked || state.adopted;
  $('accept').textContent = state.adopted ? 'Current version' : `Use proposal ${state.proposal}`;
  $('decision-summary').textContent =
    state.scenario === 'incomplete'
      ? 'One change still needs an explanation'
      : state.scenario === 'conflict'
        ? 'An updated comparison is required'
        : state.scenario === 'interrupted'
          ? 'Resolve the recorded outcome'
          : state.adopted
            ? 'Version saved'
            : 'Use this exact proposal';
  $('decision-note').textContent =
    state.scenario === 'incomplete'
      ? 'An Agent explanation alone cannot make the comparison complete.'
      : state.scenario === 'conflict'
        ? 'No automatic merge or overwrite. Earlier discussion stays readable.'
        : state.adopted
          ? 'The accepted version is retained. Analysis starts separately.'
          : 'Save it as the current pipeline version. Analysis starts separately.';
  $('revise').textContent =
    state.scenario === 'conflict'
      ? 'Ask for updated proposal'
      : state.scenario === 'interrupted'
        ? 'Check outcome'
        : 'Ask for revision';
  document.querySelectorAll('[data-subject]').forEach(
    (b) =>
      (b.onclick = () => {
        state.selected = b.dataset.subject;
        state.side = b.dataset.side;
        render();
        document
          .querySelector(`[data-subject="${state.selected}"][data-side="${state.side}"]`)
          ?.focus();
      }),
  );
  document.querySelectorAll('[data-change]').forEach(
    (b) =>
      (b.onclick = () => {
        state.selected = b.dataset.change;
        if (state.selected === 'report') state.side = 'proposed';
        render();
        document.querySelector(`[data-change="${state.selected}"]`)?.focus();
      }),
  );
  if ($('choose-data'))
    $('choose-data').onclick = () => {
      capture('input', 'Example input reads · FASTQ');
      toast('Example input reference attached. No file has been opened.');
    };
  if ($('describe-goal'))
    $('describe-goal').onclick = () => {
      focusChat();
      toast('Describe the analysis in the existing composer.');
    };
  renderAttachments();
}
function capture(kind = state.selected, label) {
  if (state.scenario === 'new') {
    state.attachments.push(
      Object.freeze({
        proposal: null,
        label: label || 'Example input reads · FASTQ',
        subject: 'input',
        kind: 'example-data',
      }),
    );
    renderAttachments();
    return;
  }
  const change = changes[kind];
  const c = Object.freeze({
    project: 'demo-atlas',
    pipeline: 'demo-rna-quality',
    comparison: `demo-comparison-${state.proposal}`,
    baseArtifact: 'demo-artifact-v2',
    candidateArtifact: `demo-artifact-proposal-${state.proposal}`,
    proposal: state.proposal,
    side: state.side,
    subject: kind,
    label:
      label ||
      (change
        ? `${change.step} · ${change.title}`
        : kind === 'align'
          ? 'Align reads'
          : 'Input reads'),
    before: change?.before,
    after: change?.after(),
  });
  state.attachments.push(c);
  renderAttachments();
  $('attachment-feedback').textContent = 'Added to message';
}
function renderAttachments() {
  $('attachments').innerHTML = state.attachments
    .map(
      (a, i) =>
        `<div class="attachment"><span>◈ ${html(a.label)}<br>${a.proposal === null ? 'Example data' : 'Proposal ' + a.proposal}${a.before ? ' · ' + html(a.before) + ' → ' + html(a.after) : ''}${a.proposal !== null && a.proposal !== state.proposal ? ' · Earlier proposal' : ''}</span><button aria-label="Remove attachment ${i + 1}" data-remove="${i}">×</button></div>`,
    )
    .join('');
  document.querySelectorAll('[data-remove]').forEach(
    (b) =>
      (b.onclick = () => {
        state.attachments.splice(Number(b.dataset.remove), 1);
        renderAttachments();
      }),
  );
}
$('current').onclick = () => {
  state.side = 'current';
  render();
  $('current').focus();
};
$('proposed').onclick = () => {
  state.side = 'proposed';
  render();
  $('proposed').focus();
};
$('list-toggle').onclick = () => {
  state.list = !state.list;
  render();
  $('list-toggle').focus();
};
$('attach').onclick = () => capture();
$('concept').onchange = (e) => {
  state.concept = e.target.value;
  render();
};
$('scenario').insertAdjacentHTML(
  'beforeend',
  '<option value="revised">Revised proposal 4</option>',
);
$('scenario').onchange = (e) => {
  state.scenario = e.target.value;
  state.adopted = false;
  state.proposal = state.scenario === 'revised' ? 4 : 3;
  if (state.scenario === 'incomplete') state.selected = 'unknown';
  else state.selected = 'quality';
  render();
};
$('accept').onclick = () => {
  state.adopted = true;
  render();
  toast('Preview: the exact reviewed proposal is now current. No analysis started.');
};
$('revise').onclick = () => {
  if (state.scenario === 'interrupted') {
    state.adopted = true;
    state.scenario = 'ready';
    $('scenario').value = 'ready';
    render();
    toast('Preview: the recorded adoption completed once. No repeated application.');
    return;
  }
  capture(state.selected, `Proposal ${state.proposal} · Request a revision`);
  focusChat();
  toast('Proposal reference attached. Your existing draft was preserved.');
};
$('review-card').onclick = () => {
  switchWorkspace();
  document.querySelector('[data-change="quality"]')?.focus();
};
$('new-pipeline').onclick = () => {
  state.scenario = 'new';
  $('scenario').value = 'new';
  render();
};
$('send').onclick = () => {
  const text = $('draft').value.trim();
  if (!text && !state.attachments.length) return;
  const message = document.createElement('div');
  message.className = 'user-message';
  message.textContent = text || 'Discuss these references.';
  const refs = document.createElement('p');
  refs.className = 'muted';
  refs.style.fontSize = '10px';
  refs.textContent = state.attachments
    .map((a) => `${a.proposal === null ? 'Example data' : 'Proposal ' + a.proposal} · ${a.label}`)
    .join(' / ');
  message.append(refs);
  $('conversation').append(message);
  $('draft').value = '';
  state.attachments = [];
  renderAttachments();
  $('conversation').scrollTop = $('conversation').scrollHeight;
  $('draft').focus();
  toast('Preview message only. No Agent is connected.');
};
$('more').onclick = () =>
  toast('Pipeline history and optional implementation details belong here.');
$('agents').onclick = () =>
  toast('Existing Project Agents remain in Chat. This preview has one illustrative Agent.');
$('reset').onclick = () => location.reload();
document.querySelectorAll('button[data-mobile]').forEach(
  (b) =>
    (b.onclick = () => {
      document.querySelector('.shell').dataset.mobile = b.dataset.mobile;
      document
        .querySelectorAll('button[data-mobile]')
        .forEach((x) => x.setAttribute('aria-pressed', String(x === b)));
    }),
);
render();
