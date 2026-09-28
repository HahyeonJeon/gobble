// Design prototype only. No service, Agent, file access, persistence or execution.
const $ = (id) => document.getElementById(id);
const concepts = {
  stacked: [
    'See both versions together',
    'Current above, Proposed below. Matching steps keep their positions.',
  ],
  spotlight: [
    'Find the change. See the difference.',
    'Select a numbered change in the flow to compare its before and after below.',
  ],
  diff: [
    'Read the changes, side by side',
    'Current on the left, Proposed on the right. Unchanged details stay folded.',
  ],
};
const requestedConcept = new URLSearchParams(location.search).get('concept');
const state = {
  concept: concepts[requestedConcept] ? requestedConcept : 'stacked',
  selected: 'quality',
  dim: true,
  context: false,
  scenario: 'ready',
  adopted: false,
  attachments: [],
};
const changes = {
  quality: {
    number: 1,
    title: 'Quality threshold',
    step: 'Trim adapters',
    before: '25 Phred',
    after: '30 Phred',
  },
  report: {
    number: 2,
    title: 'New quality check',
    step: 'FastQC',
    before: 'No quality check',
    after: 'FastQC branch from trimmed reads',
  },
};
const escape = (text) =>
  String(text).replace(
    /[&<>"']/g,
    (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' })[c],
  );
const labels = () =>
  state.adopted
    ? { before: 'Previous', after: 'Accepted' }
    : { before: state.scenario === 'conflict' ? 'Base version' : 'Current', after: 'Proposed' };
const number = (key) =>
  `<span class="change-number ${key === 'quality' ? 'amber' : 'green'}">${changes[key].number}</span>`;
const icons = {
  reads: '<path d="M5 3h12v18H5zM8 7h6M8 11h6M8 15h4"/>',
  trim: '<circle cx="6" cy="6" r="3"/><circle cx="6" cy="18" r="3"/><path d="m8 8 12 12M8 16 20 4"/>',
  align: '<path d="m12 2 10 10-10 10L2 12zM7 12h10M12 7v10"/>',
  report: '<path d="M4 3h16v18H4zM8 16v-4M12 16V7M16 16v-7"/>',
};
const icon = (name) =>
  `<svg class="node-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">${icons[name]}</svg>`;
let toastTimer;
function toast(message) {
  $('toast').textContent = message;
  $('toast').hidden = false;
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => ($('toast').hidden = true), 3800);
}
function mobile(side) {
  document.querySelector('.app').dataset.mobile = side;
  document
    .querySelectorAll('button[data-mobile]')
    .forEach((b) => b.setAttribute('aria-pressed', String(b.dataset.mobile === side)));
}
function flow(side, mode) {
  const next = side === 'proposed',
    unique = `${mode}-${side}`;
  const nodes = [
    {
      key: 'input',
      title: 'Input reads',
      sub: 'FASTQ reads',
      icon: 'reads',
      x: 4,
      y: 15,
      cls: 'input unchanged',
    },
    {
      key: 'quality',
      title: 'Trim adapters',
      icon: 'trim',
      x: 38,
      y: 15,
      cls: next ? 'changed' : 'current-change',
    },
    {
      key: 'align',
      title: 'Align reads',
      sub: 'Same trimmed-read input',
      icon: 'align',
      x: 73,
      y: 15,
      cls: 'unchanged',
    },
  ];
  if (next)
    nodes.push({
      key: 'report',
      title: 'Quality check',
      sub: 'FastQC · HTML + ZIP',
      icon: 'report',
      x: 73,
      y: 65,
      cls: 'added',
    });
  return `<div class="diagram" aria-label="${labels()[next ? 'after' : 'before']} pipeline flow"><svg class="connections" viewBox="0 0 1000 300" preserveAspectRatio="none" aria-hidden="true"><defs><marker id="arrow-${unique}" viewBox="0 0 10 10" refX="8" refY="5" markerWidth="5" markerHeight="5" orient="auto"><path d="M0 0L10 5L0 10Z" fill="#a1b19a"/></marker><marker id="add-${unique}" viewBox="0 0 10 10" refX="8" refY="5" markerWidth="5" markerHeight="5" orient="auto"><path d="M0 0L10 5L0 10Z" fill="#468f4d"/></marker></defs><path d="M270 85.5H380" stroke="#a1b19a" stroke-width="2" fill="none" marker-end="url(#arrow-${unique})"/><path d="M610 85.5H730" stroke="#a1b19a" stroke-width="2" fill="none" marker-end="url(#arrow-${unique})"/>${next ? `<path d="M610 85.5H650Q660 85.5 660 95.5V225.5Q660 235.5 670 235.5H730" stroke="#468f4d" stroke-width="3" fill="none" marker-end="url(#add-${unique})"/>` : ''}</svg>${nodes
    .map((n) => {
      const isChange = !!changes[n.key];
      const sub =
        n.key === 'quality'
          ? `<span class="node-diff">Quality ${mode === 'spotlight' ? '<span class="removed-value">− 25</span><span>→</span><span class="added-value">+ 30</span>' : next ? '<span class="added-value">+ 30</span><span>Phred</span>' : '<span class="removed-value">− 25</span><span>Phred</span>'}</span>`
          : `<span class="node-subtitle">${n.sub}</span>`;
      const tag = isChange ? 'button' : 'div';
      return `<${tag} class="flow-node ${n.cls} ${isChange && state.selected === n.key ? 'is-selected' : ''}" style="--x:${n.x}%;--y:${n.y}%" ${isChange ? `data-node="${n.key}" data-side="${side}" aria-label="Change ${changes[n.key].number}: ${n.title}, ${labels()[next ? 'after' : 'before']} flow" aria-pressed="${state.selected === n.key}"` : ''}>${next && isChange ? `<span class="node-badge"><b>${changes[n.key].number}</b> ${n.key === 'report' ? '+ Added' : 'Changed'}</span>` : ''}<span class="node-title">${icon(n.icon)}${n.title}</span>${sub}</${tag}>`;
    })
    .join(
      '',
    )}${!next ? '<div class="absence-note"><strong>No quality check</strong>Only alignment follows trimming</div>' : '<span class="graph-caption">+ New branch from trimmed reads</span>'}</div>`;
}
function flowPanel(side, mode) {
  const next = side === 'proposed';
  return `<section class="flow-panel ${side} ${mode === 'spotlight' && state.dim ? 'dim' : ''}" aria-label="${labels()[next ? 'after' : 'before']} version"><header><span class="panel-label">${labels()[next ? 'after' : 'before']}</span><span class="panel-version">${next ? 'Proposal 3' : 'Version 2'}</span>${mode === 'spotlight' ? `<button class="graph-option" id="dim-toggle" aria-pressed="${state.dim}">${state.dim ? '✓ ' : ''}Dim unchanged</button>` : `<span class="panel-meta">${next ? '3 steps · changes 1 and 2' : '2 steps · before the changes'}</span>`}</header><div class="diagram-wrap">${flow(side, mode)}</div></section>`;
}
function discussionButton(key = state.selected) {
  return `<button data-discuss="${key}" aria-label="Discuss change ${changes[key].number}">↗ Discuss change</button>`;
}
function strip() {
  const c = changes[state.selected];
  return `<section class="selection-strip" aria-label="Selected change"><div class="strip-content"><div class="strip-title">${number(state.selected)} ${c.title} <span style="font-weight:400;color:#819179">· ${c.step}</span></div><div class="strip-detail"><span class="${state.selected === 'quality' ? 'removed-value' : 'absence-value'}">${state.selected === 'quality' ? '− ' : ''}${c.before}</span><span>→</span><span class="added-value">+ ${c.after}</span></div></div>${discussionButton()}</section>`;
}
function detail() {
  const c = changes[state.selected];
  const body =
    state.selected === 'quality'
      ? `<section><div class="column-label">${labels().before} · Version 2</div><p class="field-label">Quality threshold</p><div class="large-value"><span class="sign">−</span> 25 <small>Phred</small></div><p class="same-value">Minimum length &nbsp; 40 bp · unchanged</p></section><section><div class="column-label">${labels().after} · Proposal 3</div><p class="field-label">Quality threshold</p><div class="large-value new"><span class="sign">+</span> 30 <small>Phred</small></div><p class="same-value">Minimum length &nbsp; 40 bp · unchanged</p></section>`
      : `<section><div class="column-label">${labels().before} · Version 2</div><div class="mini-path"><span class="mini-node">Trim adapters</span> → <span class="mini-node">Align reads</span></div><p class="missing-step">No quality-check step</p></section><section><div class="column-label">${labels().after} · Proposal 3</div><div class="added-step-title">+ FastQC</div><div class="detail-property"><span>Input</span> Trimmed reads</div><div class="detail-property"><span>Output</span> HTML report + ZIP</div><div class="detail-property"><span>Route</span> A parallel branch</div></section>`;
  return `<section class="detail-card" aria-label="Before and after details"><header class="detail-header">${number(state.selected)}<h2>${c.title}</h2><span class="step-name">${state.selected === 'quality' ? 'Trim adapters' : ''}</span>${discussionButton()}</header><div class="before-after">${body}</div><p class="detail-note">${state.selected === 'quality' ? 'Only the quality threshold changes here; minimum length stays at 40 bp.' : 'Alignment keeps its existing input and does not wait for the new quality report.'}</p></section>`;
}
function contextRows() {
  return `<div class="diff-row unchanged-row"><div class="diff-cell"><span class="line-sign"> </span><div class="diff-property">Minimum length <strong>40 bp</strong></div></div><div class="diff-cell"><span class="line-sign"> </span><div class="diff-property">Minimum length <strong>40 bp</strong></div></div></div><div class="diff-row unchanged-row"><div class="diff-cell"><span class="line-sign"> </span><div class="diff-property">Input <strong>FASTQ reads</strong></div></div><div class="diff-cell"><span class="line-sign"> </span><div class="diff-property">Input <strong>FASTQ reads</strong></div></div></div>`;
}
function miniTopology(next) {
  return `<div class="mini-branch"><span class="mini-node origin">Trim adapters</span><svg viewBox="0 0 32 100" preserveAspectRatio="none" fill="none" aria-hidden="true"><path d="M0 20H32" stroke="#adc0a2" stroke-width="1.5"/>${next ? '<path d="M0 20H6Q12 20 12 26V74Q12 80 18 80H32" stroke="#519047" stroke-width="2"/>' : ''}</svg><span class="mini-node normal">Align reads</span>${next ? '<span class="mini-node new">+ FastQC</span>' : ''}</div>`;
}
function diffDocument() {
  return `<section class="diff-document" aria-label="Visual change diff"><header><strong>RNA quality</strong><span style="color:#849276;font-size:11px">· 2 changed sections</span><button id="context-all" aria-pressed="${state.context}">${state.context ? 'Hide' : 'Show'} unchanged details</button></header><div class="diff-columns"><span>${labels().before} · Version 2</span><span>${labels().after} · Proposal 3</span></div><section class="diff-block ${state.selected === 'quality' ? 'is-selected' : ''}" data-block="quality"><header class="diff-block-title">${number('quality')}<h2>Trim adapters</h2><small>Setting changed</small>${discussionButton('quality')}</header>${state.context ? contextRows() : ''}<div class="diff-row"><div class="diff-cell removed"><span class="line-sign">−</span><div class="diff-property">Quality threshold <strong>25 <small>Phred</small></strong></div></div><div class="diff-cell added"><span class="line-sign">+</span><div class="diff-property">Quality threshold <strong>30 <small>Phred</small></strong></div></div></div><button id="context-inline" class="context-toggle">${state.context ? '− Hide' : '··· Show'} 2 unchanged details</button></section><section class="diff-block ${state.selected === 'report' ? 'is-selected' : ''}" data-block="report"><header class="diff-block-title">${number('report')}<h2>Quality check</h2><small>Step added</small>${discussionButton('report')}</header><div class="topology-diff"><section><div class="topology-title">After trimming</div>${miniTopology(false)}<p class="missing-step">No quality-check step</p></section><section><div class="topology-title">After trimming <span class="added-word">· + New branch</span></div>${miniTopology(true)}<p class="topology-footnote">FastQC receives the trimmed reads.<br>Produces an HTML report and ZIP.</p></section></div></section><p class="diff-endnote">Alignment and its input connection are unchanged.</p></section>`;
}
function render() {
  const scroll = $('review-body').scrollTop;
  $('concept-title').textContent = concepts[state.concept][0];
  $('concept-description').textContent = concepts[state.concept][1];
  document
    .querySelectorAll('[data-concept]')
    .forEach((b) => b.setAttribute('aria-pressed', String(b.dataset.concept === state.concept)));
  document
    .querySelectorAll('[data-change]')
    .forEach((b) => b.setAttribute('aria-pressed', String(b.dataset.change === state.selected)));
  $('review-body').innerHTML =
    state.concept === 'stacked'
      ? `<div class="stacked">${flowPanel('current', 'stacked')}${flowPanel('proposed', 'stacked')}${strip()}</div>`
      : state.concept === 'spotlight'
        ? `<div class="spotlight">${flowPanel('proposed', 'spotlight')}${detail()}</div>`
        : diffDocument();
  $('review-body').scrollTop = scroll;
  const notice = state.adopted
    ? 'Preview: proposal 3 is now current. The two versions remain here as review history. No analysis has started.'
    : state.scenario === 'incomplete'
      ? 'One more processing change cannot yet be explained. The two visible changes are not the whole comparison. Ask the Agent to revise the proposal before adopting it.'
      : state.scenario === 'conflict'
        ? 'This comparison is based on version 2, but the current pipeline is now version 3. Keep this comparison for discussion and ask for an updated proposal.'
        : '';
  $('notice').textContent = notice;
  $('notice').hidden = !notice;
  $('notice').classList.toggle('success', state.adopted);
  $('adopt').disabled = state.adopted || state.scenario !== 'ready';
  $('adopt').textContent = state.adopted ? 'Version saved' : 'Use proposal 3';
  $('decision-label').textContent = state.adopted
    ? 'Proposal 3 is now current'
    : state.scenario === 'incomplete'
      ? 'The comparison is incomplete'
      : state.scenario === 'conflict'
        ? 'A newer current version needs review'
        : 'Use this reviewed version';
  $('decision-note').textContent =
    state.scenario === 'ready'
      ? 'Checked with example reads. Analysis starts separately.'
      : 'Your current pipeline and conversation remain available.';
  $('review-version').textContent = state.adopted
    ? 'REVIEW HISTORY · ACCEPTED PROPOSAL 3'
    : 'PROPOSAL 3 · BASED ON VERSION 2';
  document.querySelector('.change-count').innerHTML =
    state.scenario === 'incomplete'
      ? '2 shown · 1 unexplained'
      : '<span class="diff-legend"><span>− Before</span><span>+ After</span></span>';
}
function select(key, focusSelector) {
  state.selected = key;
  render();
  if (focusSelector) document.querySelector(focusSelector)?.focus();
  if (state.concept === 'diff')
    document.querySelector(`[data-block="${key}"]`)?.scrollIntoView({ block: 'nearest' });
}
function attach(key) {
  const c = changes[key];
  state.attachments.push(
    Object.freeze({
      project: 'demo-atlas',
      pipeline: 'demo-rna-quality',
      comparison: 'comparison-3',
      base: 'artifact-v2',
      candidate: 'artifact-proposal-3',
      subject: key,
      ...c,
    }),
  );
  renderAttachments();
  mobile('chat');
  $('draft').focus();
  toast(`Change ${c.number} attached. Your draft is preserved.`);
}
function renderAttachments() {
  $('attachments').innerHTML = state.attachments
    .map(
      (a, i) =>
        `<div class="attachment">${number(a.subject)}<div><strong>${escape(a.title)}</strong><br>${escape(a.before)} → ${escape(a.after)}<br><small>Proposal 3 · Current v2 → Proposed</small></div><button data-remove="${i}" aria-label="Remove attachment ${i + 1}">×</button></div>`,
    )
    .join('');
}
document.querySelector('.concepts').addEventListener('click', (e) => {
  const b = e.target.closest('[data-concept]');
  if (!b) return;
  state.concept = b.dataset.concept;
  $('review-body').scrollTop = 0;
  render();
  b.focus();
  history.replaceState(null, '', `?concept=${state.concept}`);
});
document.querySelector('.change-tabs').addEventListener('click', (e) => {
  const b = e.target.closest('[data-change]');
  if (b) select(b.dataset.change);
});
$('review-body').addEventListener('click', (e) => {
  const discuss = e.target.closest('[data-discuss]');
  if (discuss) {
    state.selected = discuss.dataset.discuss;
    render();
    attach(state.selected);
    return;
  }
  const node = e.target.closest('[data-node]');
  if (node) {
    select(
      node.dataset.node,
      `[data-node="${node.dataset.node}"][data-side="${node.dataset.side}"]`,
    );
    return;
  }
  if (e.target.closest('#dim-toggle')) {
    state.dim = !state.dim;
    render();
    $('dim-toggle').focus();
    return;
  }
  const context = e.target.closest('#context-all, #context-inline');
  if (context) {
    const id = context.id;
    state.context = !state.context;
    render();
    $(id).focus();
  }
});
document.querySelectorAll('[data-chat-change]').forEach((b) =>
  b.addEventListener('click', () => {
    mobile('work');
    select(b.dataset.chatChange);
    document.querySelector(`[data-change="${b.dataset.chatChange}"]`).focus();
  }),
);
$('attachments').addEventListener('click', (e) => {
  const b = e.target.closest('[data-remove]');
  if (b) {
    state.attachments.splice(Number(b.dataset.remove), 1);
    renderAttachments();
    $('draft').focus();
  }
});
$('revise').onclick = () => attach(state.selected);
$('scenario').onchange = (e) => {
  state.scenario = e.target.value;
  state.adopted = false;
  render();
};
$('adopt').onclick = () => {
  state.adopted = true;
  render();
  toast('Local simulation only. No pipeline or source file was changed.');
};
$('send').onclick = () => {
  const text = $('draft').value.trim();
  if (!text && !state.attachments.length) return;
  const item = document.createElement('div');
  item.className = 'user-message';
  item.textContent = text || 'Please review these changes.';
  const refs = document.createElement('p');
  refs.style.cssText = 'font-size:10px;margin-top:8px;color:#839779';
  refs.textContent = state.attachments
    .map((a) => `Change ${a.number} · ${a.before} → ${a.after} · Proposal 3`)
    .join(' / ');
  item.append(refs);
  $('conversation').append(item);
  state.attachments = [];
  $('draft').value = '';
  renderAttachments();
  $('conversation').scrollTop = $('conversation').scrollHeight;
  toast('Preview message only. No Agent is connected.');
  $('draft').focus();
};
document
  .querySelectorAll('button[data-mobile]')
  .forEach((b) => (b.onclick = () => mobile(b.dataset.mobile)));
document
  .querySelectorAll('.rail button')
  .forEach(
    (b) => (b.onclick = () => toast('Project navigation is outside this comparison sketch.')),
  );
render();
