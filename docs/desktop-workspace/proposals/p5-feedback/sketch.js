// Design fixture only. No network requests, filesystem operations or execution API.
const scenes = new Set([
  "failure",
  "feedback",
  "review",
  "ready",
  "stale",
  "resume",
  "concepts",
  "running",
  "current",
]);
let scene = new URLSearchParams(location.search).get("scene") || "failure";
if (!scenes.has(scene)) scene = "failure";
let selected = scene === "failure" ? "qc" : "trim";
let attached = null;
let sent = false;
let logOpen = false;
let adopted = false;
const draft = document.querySelector("#composer");
const main = document.querySelector("main");
const messages = document.querySelector("#messages");
const esc = (s) =>
  s.replace(
    /[&<>"']/g,
    (c) =>
      ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[
        c
      ],
  );
const badge = (text, kind = "") => `<span class="badge ${kind}">${text}</span>`;
const button = (label, action, cls = "") =>
  `<button data-action="${action}" class="${cls}">${label}</button>`;
const ref = (text, to = "feedback") =>
  `<button class="origin" data-scene="${to}">↗ ${text}</button>`;
function announce(text) {
  document.querySelector("#announcement").textContent = text;
}
function area(value) {
  document.querySelector(".layout").dataset.area = value;
  document.querySelectorAll("[data-area]").forEach((b) => {
    if (b.tagName === "BUTTON")
      b.setAttribute("aria-pressed", String(b.dataset.area === value));
  });
}
function navigate(next) {
  announce("");
  scene = next;
  selected = next === "failure" ? "qc" : "trim";
  logOpen = false;
  sent = false;
  history.replaceState(null, "", `?scene=${next}`);
  render();
  area("workspace");
  main.focus();
}
function connector(down) {
  return `<svg class="connector" viewBox="0 0 30 150" aria-hidden="true"><path d="${down ? "M0 58 H7 Q11 58 11 62 V90 Q11 94 15 94 H30" : "M0 94 H30"}"/></svg>`;
}
function node(id, label, sub, status, kind, icon) {
  return `<button class="node ${id === "input" ? "input" : ""} ${kind} ${selected === id ? "selected" : ""}" data-node="${id}" aria-pressed="${selected === id}"><div class="node-body"><span class="node-icon" aria-hidden="true">${icon}</span><b>${label}</b><small>${sub}</small></div><div class="node-footer">${status}</div></button>`;
}
function flow(mode) {
  let trimStatus = "✓ Succeeded",
    qcStatus = "✓ Succeeded",
    trimKind = "succeeded",
    qcKind = "succeeded";
  if (mode === "failure") {
    qcStatus = "! Failed";
    qcKind = "failed";
  }
  if (mode === "review") {
    trimStatus = "Δ 1 setting changed";
    trimKind = "changed";
    qcStatus = "Unchanged · downstream";
    qcKind = "";
  }
  if (mode === "ready") {
    trimStatus = qcStatus = "↻ Runs again";
    trimKind = qcKind = "";
  }
  if (mode === "current") {
    trimStatus = qcStatus = "Declared step";
    trimKind = qcKind = "";
  }
  if (mode === "running") {
    trimStatus = "◷ Running";
    qcStatus = "○ Not started";
    trimKind = "running";
    qcKind = "";
  }
  if (mode === "resume") {
    trimStatus = "Can reuse · after checks";
    qcStatus = "Will run · was stopped";
    trimKind = "succeeded";
    qcKind = "";
  }
  return `<div class="flow"><div class="flow-track">${node("input", "Read data", "sample.fastq.gz", "Original input", "", "≋")}${connector(true)}${node("trim", "Trim reads", "Trim Galore", trimStatus, trimKind, "✂")}${connector(false)}${node("qc", "Check read quality", "FastQC", qcStatus, qcKind, "▥")}</div><div class="legend">${mode === "review" ? '<span><i class="dot amber"></i>Changed setting</span><span>Downstream ≠ changed</span>' : mode === "ready" ? "<span>New results location</span><span>All steps run again</span>" : mode === "current" ? "<span>Current design · no Run status</span>" : mode === "running" ? "<span>Observed execution state</span>" : mode === "resume" ? "<span>Future review · illustrative decisions</span>" : '<span><i class="dot"></i>Completed work</span><span><i class="dot red"></i>Failed work</span>'}<span>Click a step to discuss it</span></div></div>`;
}
function head(title, subtitle, status) {
  return `<header class="surface-head"><div><h1>${title}</h1><div class="sub">${subtitle}</div></div>${status}</header>`;
}
function message(who, body, user = false) {
  return `<div class="message ${user ? "user" : ""}"><div class="who"><span class="avatar" aria-hidden="true">${user ? "Y" : "g"}</span>${who}</div>${body}</div>`;
}
function detail() {
  if (selected === "input")
    return `<div class="detail"><h3>Read data</h3><p>sample.fastq.gz · Single-end reads</p><div class="note">The analysis uses a captured input copy. A new analysis checks its selected data again.</div></div>`;
  if (scene === "review")
    return `<div class="detail"><div class="detail-head"><div><h3>${selected === "trim" ? "Minimum length" : "Check read quality"}</h3><div class="sub">${selected === "trim" ? "Trim reads · scientific setting" : "Unchanged step · downstream of Trim reads"}</div></div>${badge(selected === "trim" ? "Changed" : "Unchanged", selected === "trim" ? "change" : "")}</div>${selected === "trim" ? '<div class="value-row"><div class="value"><small>CURRENT</small><strong>30 <em>bp</em></strong></div><div class="arrow">→</div><div class="value proposed"><small>PROPOSED</small><strong>20 <em>bp</em></strong></div></div><p>Reads shorter than 20 bp will be discarded, instead of reads shorter than 30 bp. Other declared settings stay the same.</p>' : "<p>The quality check itself is unchanged. It will receive the output of the revised trimming step.</p>"}<div class="note">This is a design comparison. Adopting it does not run the analysis.</div><div class="detail-actions">${button("Discuss this change", "discuss")}</div></div>`;
  if (scene === "ready")
    return `<div class="detail"><h3>What will run</h3><div class="facts"><div class="fact"><small>DATA COPY</small><b>sample.fastq.gz</b><p>Checked · same content as Analysis 12</p></div><div class="fact new"><small>RESULTS LOCATION</small><b>New, separate folder</b><p>runs/analysis-followup-demo</p></div></div><p>Both steps run from the checked input. Analysis 12 and its results stay available.</p><div class="note">Example ready state. The real App checks data, tools and destination before it offers Start.</div></div>`;
  if (scene === "resume")
    return `<div class="detail"><h3>${selected === "trim" ? "Earlier output may be reusable" : "Continue unfinished work"}</h3><p>${selected === "trim" ? "Gobble must verify this step’s input, settings, tools and recorded output content. A previous success status alone is insufficient." : "The design stays the same. Gobble decides which work runs and records each new attempt."}</p><div class="note">Later P5B concept: only a settled, stopped Run with the same admitted design. Live or unknown owners, changed evidence and missing files require another review.</div></div>`;
  if (scene === "running")
    return `<div class="detail"><h3>${selected === "trim" ? "Trim reads" : "Check read quality"}</h3><p>${selected === "trim" ? "Running with the new minimum length of 20 bp." : "Waiting for trimmed reads. No successful result has been reported."}</p><div class="detail-actions">${button("Discuss this step", "discuss")}${ref("Following Analysis 12")}</div><div class="note">Simulated admission and progress. No analysis was started by this sketch.</div></div>`;
  const failed = scene === "failure" && selected === "qc";
  return `<div class="detail"><div class="detail-head"><div><h3>${selected === "qc" ? "Check read quality" : "Trim reads"}</h3><div class="sub">Analysis 12 · ${selected === "qc" ? "FastQC" : "Trim Galore"} · attempt 1</div></div>${badge(failed ? "! Failed" : "✓ Succeeded", failed ? "fail" : "ok")}</div><p>${failed ? "Observed: the process exited with code 1. The cause has not been confirmed." : "The step completed. You can discuss its result or ask the Agent to propose a supported setting change."}</p>${logOpen ? '<div class="log">Observed excerpt · stderr · 1 line\nProcess exited with code 1.</div><p class="muted">Illustrative excerpt only; not a real tool diagnostic. No full log is attached automatically.</p>' : ""}<div class="detail-actions">${button("Discuss this step", "discuss", "primary")}${failed ? button(logOpen ? "Attach this excerpt" : "View log excerpt", logOpen ? "attach-log" : "log") : ""}</div>${failed ? '<div class="note">Share the evidence first. A setting change is not automatically a fix for this failure.</div>' : '<div class="note">Current example: Minimum length is explicitly set to 30 bp. A tool default would be labeled unknown.</div>'}</div>`;
}
function render() {
  document
    .querySelectorAll("[data-scene]")
    .forEach((b) =>
      b.setAttribute("aria-pressed", String(b.dataset.scene === scene)),
    );
  let content = "",
    chat = "";
  if (scene === "concepts") {
    content =
      head(
        "Two ways to follow up",
        "Same Project · same Chat · different navigation model",
        badge("Design alternatives"),
      ) +
      `<div class="concepts"><article class="concept recommended"><h3>In context</h3>${badge("Recommended", "ok")}<div class="mini-flow"><span>Run Flow ↔ exact evidence in Chat</span><span>Change spotlight ↔ Agent explanation</span><span>Preparation ↔ explicit Start</span></div><p>Move between existing views. A small origin link keeps the earlier analysis close.</p><p>Requires clear version labels when Current changes.</p>${button("Explore this approach", "feedback", "primary")}</article><article class="concept"><h3>Guided recovery path</h3>${badge("Alternative")}<div class="mini-flow"><span>1. Evidence</span><span>2. Diagnosis → 3. Change</span><span>4. Execution · next / back</span></div><p>A dedicated sequence in the center holds progress while Chat stays at the right.</p><p>Adds another navigation layer and may imply every failure has a known fix.</p></article></div>`;
    chat = message(
      "Design note",
      "<p>The recommendation keeps the existing workbench structure. Review the exact point where evidence becomes a proposal, and where adoption becomes an execution request.</p><p>This comparison is a design study, not user testing.</p>",
    );
  } else if (scene === "current") {
    content =
      head(
        "Current design 4",
        "Read quality workflow · checked design",
        badge("Current"),
      ) +
      `<div class="banner"><p>Selected as the base for a new discussion.</p>${ref("Earlier Analysis 12", "stale")}</div>` +
      flow("current") +
      `<div class="detail"><h3>${selected === "qc" ? "Check read quality" : "Minimum length"}</h3><div class="facts"><div class="fact"><small>CURRENT DESIGN 4</small><b>${selected === "qc" ? "Declared quality check" : "25 bp"}</b></div><div class="fact"><small>EARLIER ANALYSIS 12</small><b>Design 3 · Minimum length 30 bp</b></div></div><p>A new proposal must be checked against this Current version. The earlier 30 → 20 example cannot be adopted here.</p><div class="detail-actions">${button("Discuss current design", "discuss", "primary")}</div></div>`;
    chat = message(
      "Research agent",
      "<p>We are now looking at Current design 4. Minimum length is 25 bp. You can share this exact design alongside the earlier Run evidence.</p><p>No proposal has been submitted or rebased.</p>",
    );
  } else if (scene === "stale") {
    content =
      head(
        "Analysis 12",
        "Retained design 3 · Current is now design 4",
        badge("Earlier design", "change"),
      ) +
      `<div class="banner warning"><p><b>This analysis used an earlier design.</b><br>Its evidence remains valid for that earlier analysis.</p></div>` +
      flow("feedback") +
      `<div class="detail"><h3>Choose the design you want to discuss</h3><p>The old Run is not an editable copy of Current. Open Current before asking the Agent to change it.</p><div class="facts"><div class="fact"><small>THIS ANALYSIS</small><b>Design 3 · Minimum length 30 bp</b></div><div class="fact"><small>CURRENT</small><b>Design 4 · Minimum length 25 bp</b></div></div><div class="detail-actions">${button("Open current design", "current", "primary")}${button("Discuss earlier analysis", "discuss")}</div><div class="note" id="stale-note">No automatic step remapping or proposal rebasing.</div></div>`;
    chat = message(
      "Research agent",
      "<p>The selected result belongs to the earlier design. I can discuss that evidence, but I need the checked Current design as the base for a new change.</p>",
    );
  } else if (scene === "resume") {
    content =
      head(
        "Continue Analysis 12",
        "Stopped · same design · same results location",
        badge("P5B · future concept"),
      ) +
      `<div class="banner warning"><p><b>Separate from a new analysis</b><br>Resume requires its own engine review and User approval.</p></div>` +
      flow("resume") +
      detail();
    chat = message(
      "Research agent",
      '<p>If the design stays the same, Gobble may be able to continue unfinished work and reuse verified results.</p><p>If you change the design, the first P5 milestone starts a new analysis instead.</p><div class="actioncard"><h3>Continue stopped work</h3><p>Future example: 1 step can be reused, 1 will run. This is not an actual engine review.</p><button disabled>Resume · not available yet</button></div>',
    );
  } else if (scene === "review") {
    content =
      head(
        "Review proposed change",
        "Read quality workflow · Current → Proposed",
        badge("1 changed setting", "change"),
      ) +
      `<div class="banner"><p>Allow shorter reads through trimming</p>${ref("From Analysis 12")}</div>` +
      flow("review") +
      detail();
    chat =
      message(
        "You",
        "<p>Shorter reads are acceptable for this analysis. Please lower the minimum length from 30 to 20 bp.</p>" +
          ref("Analysis 12 · Trim reads"),
        true,
      ) +
      message(
        "Research agent",
        '<p>I propose changing the minimum length to 20 bp. Select the amber step to compare the exact values.</p><div class="actioncard"><h3>Checked proposal</h3><p>Only Minimum length changes. Using this version updates the design; it does not start work.</p>' +
          button("Use this version", "adopt", "primary") +
          "</div>",
      );
  } else if (scene === "ready") {
    content =
      head(
        "New analysis",
        "Read quality workflow · reviewed design",
        badge("Ready for review", "ok"),
      ) +
      `<div class="banner"><p>${adopted ? "Version adopted. Example data checks are now shown complete." : "Fresh-run example · all steps run again"}</p>${ref("Following Analysis 12")}</div>` +
      flow("ready") +
      detail();
    chat = message(
      "Research agent",
      '<p>The reviewed design is ready for a new analysis. Both steps will run, using a checked input copy and a new results location.</p><div class="actioncard"><h3>Start new analysis</h3><p>sample.fastq.gz · same content as Analysis 12<br>Minimum length: 20 bp<br>2 steps run · previous results kept</p>' +
        button("Start new analysis", "start", "primary") +
        "<p>Prototype action · no real execution</p></div>",
    );
  } else if (scene === "running") {
    content =
      head(
        "Analysis 13",
        "New Run · reviewed design · fresh results",
        badge("◷ Running"),
      ) +
      `<div class="banner"><p>Minimum length 20 bp · sample.fastq.gz</p>${ref("Following Analysis 12")}</div>` +
      flow("running") +
      detail();
    chat = message(
      "Gobble",
      "<p>Simulation: Analysis 13 has started. Trim reads is running; Check read quality is waiting.</p><p>The earlier analysis and its evidence remain available through the origin link.</p>",
    );
  } else {
    const failed = scene === "failure";
    content =
      head(
        "Analysis 12",
        "Read quality workflow · retained run design",
        badge(failed ? "! Failed" : "✓ Completed", failed ? "fail" : "ok"),
      ) +
      `<div class="banner"><p>${failed ? "Select the failed step to inspect and discuss what happened." : "Discuss a result, then review the Agent’s proposed refinement."}</p>${ref("View current design", "current")}</div>` +
      flow(failed ? "failure" : "feedback") +
      detail();
    chat =
      message(
        "Gobble",
        failed
          ? "<p><b>Check read quality failed.</b> Trim reads completed. Select the failed step to inspect its observed state and available log excerpt.</p>"
          : "<p><b>Analysis 12 completed.</b> Its results and design are retained together.</p>",
      ) +
      message(
        "Research agent",
        failed
          ? "<p>I can help investigate the selected step. Share its observation or a specific log excerpt so we can discuss the same evidence.</p><p>An exit code alone does not establish why the tool failed.</p>"
          : '<p>Point to a step or result and tell me what you want to improve. I can propose a supported change for you to review.</p><div class="actioncard"><h3>Explore the refinement example</h3><p>Separate scenario: you ask to retain shorter reads by changing Minimum length from 30 to 20 bp.</p>' +
              button("Open example proposal", "review") +
              "</div>",
      );
  }
  if (sent)
    chat +=
      message(
        "You",
        `<p>${esc(sent.text)}</p>${sent.ref ? `<span class="ref">↗ ${esc(sent.ref)}</span>` : ""}`,
        true,
      ) +
      message(
        "Research agent",
        "<p>Simulation: I can now refer to that exact shared observation. This sketch does not send a message to a real Agent.</p>",
      );
  main.innerHTML = `<section class="surface">${content}</section>`;
  messages.innerHTML = chat;
  renderAttachment();
}
function renderAttachment() {
  document.querySelector("#attachment").innerHTML = attached
    ? `<div class="attachment"><span>↗ ${esc(attached)}</span><button data-action="remove" aria-label="Remove attached reference">×</button></div>`
    : "";
}
function attach(log = false) {
  attached =
    scene === "current"
      ? "Current design 4 · checked artifact · " +
        (selected === "qc" ? "Check read quality" : "Minimum length")
      : log
        ? "Analysis 12 · FastQC · attempt 1 · stderr line 1"
        : scene === "review"
          ? "Proposal · Trim reads · Minimum length"
          : `Analysis ${scene === "running" ? "13" : "12"} · ${selected === "qc" ? "FastQC" : "Trim Galore"} · attempt 1`;
  if (!draft.value)
    draft.value = log
      ? "Please investigate this observed error."
      : scene === "review"
        ? "Can you explain the impact of this setting change?"
        : "Please help me understand this step and what we should consider next.";
  renderAttachment();
  area("chat");
  draft.focus();
  announce(
    "Exact reference added to the draft. Review your message before sending.",
  );
}
document.addEventListener("click", (event) => {
  const el = event.target.closest("button");
  if (!el) return;
  if (el.dataset.scene) {
    navigate(el.dataset.scene);
    return;
  }
  if (el.dataset.area) {
    area(el.dataset.area);
    return;
  }
  if (el.dataset.node) {
    selected = el.dataset.node;
    render();
    main.querySelector(`[data-node="${selected}"]`)?.focus();
    announce("Step selected. Its details are shown below.");
    return;
  }
  switch (el.dataset.action) {
    case "discuss":
      attach();
      break;
    case "attach-log":
      attach(true);
      break;
    case "remove":
      attached = null;
      renderAttachment();
      draft.focus();
      break;
    case "log":
      logOpen = true;
      render();
      main.querySelector('[data-action="attach-log"]')?.focus();
      break;
    case "review":
      navigate("review");
      break;
    case "feedback":
      navigate("feedback");
      break;
    case "adopt":
      adopted = true;
      navigate("ready");
      announce(
        "Simulation: design adopted. Review the separate new-analysis action in Chat.",
      );
      break;
    case "start":
      navigate("running");
      announce("Simulation only. No real analysis was started.");
      break;
    case "current":
      navigate("current");
      announce("Current design 4 opened. No proposal was submitted.");
      break;
  }
});
document.querySelector("#send").addEventListener("click", () => {
  if (!draft.value.trim()) {
    draft.focus();
    return;
  }
  sent = { text: draft.value.trim(), ref: attached };
  draft.value = "";
  attached = null;
  render();
  draft.focus();
  announce("Simulated message added. No real Agent was contacted.");
});
render();
