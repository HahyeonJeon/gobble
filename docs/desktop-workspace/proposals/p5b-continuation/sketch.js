// Local design fixture. All checks and execution states are simulated; no execution API or network.
const allowed = new Set([
  "ready",
  "stopping",
  "stopped",
  "stale",
  "blocked",
  "newer",
  "older",
  "uncertain",
  "running",
  "complete",
  "current",
]);
const params = new URLSearchParams(location.search);
let scene = allowed.has(params.get("scene")) ? params.get("scene") : "ready";
let concept = params.get("concept") === "guided" ? "guided" : "context";
let selected = scene === "blocked" ? "trim" : "qc",
  wizardStep = 1,
  attachment = null,
  sent = [],
  marked = false,
  reviewVersion = 4;
const draft = document.querySelector("#draft");
const $ = (s) => document.querySelector(s);
const esc = (s) =>
  String(s).replace(
    /[&<>"']/g,
    (c) =>
      ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[
        c
      ],
  );
const btn = (label, action, primary = false, disabled = false) =>
  `<button data-action="${action}" ${primary ? 'class="primary"' : ""} ${disabled ? "disabled" : ""}>${label}</button>`;
function area(value) {
  $(".layout").dataset.area = value;
  document
    .querySelectorAll("[data-area]")
    .forEach((b) =>
      b.setAttribute("aria-pressed", String(b.dataset.area === value)),
    );
}
function announce(text) {
  $("#status").textContent = text;
}
const checked = () => ["ready", "newer"].includes(scene);
const hasReview = () =>
  checked() ||
  ["stale", "blocked", "running", "uncertain", "complete"].includes(scene);
function reference() {
  const review = hasReview() && !["running", "complete"].includes(scene);
  const attempt =
    ["running", "complete"].includes(scene) && selected === "qc" ? 2 : 1;
  return {
    kind: review ? "continuation-review-step" : "run-task",
    run: "Analysis 12",
    runRef: "run_analysis_12",
    snapshot: ["running", "complete"].includes(scene)
      ? `snap_${scene}_12`
      : "snap_stopped_12",
    ...(review
      ? {
          reviewId: `review_${reviewVersion}`,
          reviewDigest: `review-content-${reviewVersion}`,
        }
      : {}),
    step: selected,
    stepLabel: selected === "trim" ? "Trim reads" : "Inspect read quality",
    priorAttempt: attempt,
    ...(checked() ? { plannedAttempt: selected === "trim" ? 1 : 2 } : {}),
    decision: statusFor(selected)[1],
  };
}
function wire() {
  return '<svg class="wire" viewBox="0 0 28 26" aria-hidden="true"><path d="M0 8 H8 Q12 8 12 12 V14 Q12 18 16 18 H27 M23 15 L27 18 L23 21"/></svg>';
}
function statusFor(id) {
  if (scene === "current") return ["work", "Current design"];
  if (scene === "stale") return ["stopped", "Earlier review · Check again"];
  if (scene === "uncertain") return ["stopped", "Confirmation pending"];
  if (scene === "blocked" && id === "qc")
    return ["stopped", "Blocked upstream"];
  if (id === "trim")
    return scene === "blocked"
      ? ["error", "Cannot verify result"]
      : [
          "reuse",
          ["running", "complete"].includes(scene)
            ? "Reused · Attempt 1"
            : hasReview()
              ? "Can reuse · Attempt 1"
              : "Completed · Attempt 1",
        ];
  return ["running", "complete"].includes(scene)
    ? [
        scene === "complete" ? "reuse" : "work",
        scene === "complete" ? "Completed · Attempt 2" : "Running · Attempt 2",
      ]
    : ["stopped", "stopping", "older"].includes(scene)
      ? [
          "stopped",
          scene === "stopping" ? "Stopping · Attempt 1" : "Stopped · Attempt 1",
        ]
      : ["work", "Will restart · Attempt 2"];
}
function flow() {
  const node = (id) => {
    const [kind, label] = statusFor(id);
    return `<button class="node ${kind}" data-select="${id}" aria-pressed="${selected === id}" aria-label="${id === "trim" ? "Trim reads" : "Inspect read quality"}: ${label}">${marked && id === "trim" ? '<span class="agent-mark">Agent reference</span>' : ""}<b><span class="icon" aria-hidden="true">${id === "trim" ? "✂" : "▥"}</span>${id === "trim" ? "Trim adapters and low-quality bases" : "Inspect trimmed read quality"}</b><small>${label}</small></button>`;
  };
  return `<div class="flow" aria-label="Analysis flow"><div class="node"><b><span class="icon" aria-hidden="true">▤</span>Read data</b><small>Saved input copy</small></div>${wire()}${node("trim")}${wire()}${node("qc")}</div><div class="legend"><span>✓ Verified result</span><span>↻ Work to perform</span><span>Selection adds no permission</span></div>`;
}
function detail() {
  const trim = selected === "trim";
  const accepted = ["running", "complete"].includes(scene);
  const invalid = ["stale", "blocked", "uncertain"].includes(scene);
  const reviewed = checked() || accepted;
  const title = accepted
    ? trim
      ? "Trim result kept · Attempt 1"
      : scene === "complete"
        ? "Quality check completed · Attempt 2"
        : "Quality check running · Attempt 2"
    : invalid
      ? scene === "blocked"
        ? "No continuation can start"
        : "This plan needs confirmation"
      : reviewed
        ? trim
          ? "Keep the verified Trim result"
          : "Restart the stopped quality check"
        : trim
          ? "Trim result · Attempt 1"
          : "Quality check · Attempt 1";
  const description = invalid
    ? scene === "blocked"
      ? "A completed Trim result is unavailable. Gobble will not silently rerun the completed step."
      : "The earlier plan is not an executable permission. Resolve the current status before continuing."
    : reviewed
      ? trim
        ? "The saved data, settings, installed tool and recorded outputs match this analysis. No new Trim attempt is planned."
        : "Attempt 1 stopped before a complete result was published. The whole quality check will run again; it does not continue from inside the process."
      : "This is the recorded task state. Completed status alone does not verify that its output can be reused. A continuation check is required.";
  return `<section class="detail" aria-label="Selected step"><header><div><span class="eyebrow">${hasReview() ? "CONTINUATION REVIEW" : "OBSERVED RUN"} · ${trim ? "STEP 1" : "STEP 2"}</span><h2>${title}</h2></div>${btn("Discuss this step", "attach", false, scene === "current")}</header><p>${description}</p><dl><div><dt>Earlier attempt</dt><dd>${trim ? "Completed · Attempt 1" : scene === "stopping" ? "Stopping · Attempt 1" : "Stopped · Attempt 1"}</dd></div><div><dt>${accepted ? "Accepted continuation" : "Continuation decision"}</dt><dd>${reviewed || invalid ? statusFor(selected)[1] : "Not checked"}</dd></div></dl><p class="muted">Earlier attempt records and captured references remain available.</p></section>`;
}
function notice() {
  const map = {
    stopping: [
      "",
      "Stop is still settling",
      "No continuation check is ready yet. Wait for Gobble to confirm that execution has stopped.",
    ],
    stale: [
      "error",
      "This review has changed",
      "The saved results no longer match the reviewed snapshot. Check again; the app will not silently replace “Can reuse” with “Run again”.",
    ],
    blocked: [
      "error",
      "Continuation is blocked",
      "A completed Trim result is missing. Nothing will run. Discuss the issue or open Current to prepare a separate new analysis.",
    ],
    newer: [
      "info",
      "Current is newer than this Run",
      "Continuation uses this analysis’s saved design and data. Changes in Current will not be applied.",
    ],
    older: [
      "",
      "This analysis uses an earlier engine",
      "This record supports inspection, but not the new continuation contract. No in-place conversion is proposed.",
    ],
    uncertain: [
      "info",
      "Checking the same request",
      "The connection was lost after confirmation. The outcome is unknown. Check status; do not send another Resume or start a replacement Run.",
    ],
    current: [
      "info",
      "Current design",
      "Editing and a new analysis use the existing visual proposal and Start flow. This sketch does not create a new analysis.",
    ],
  };
  const n = map[scene];
  return n
    ? `<aside class="notice ${n[0]}" role="status"><strong>${n[1]}</strong><p>${n[2]}</p></aside>`
    : "";
}
function render() {
  document
    .querySelectorAll("[data-scene]")
    .forEach((b) =>
      b.setAttribute("aria-pressed", String(b.dataset.scene === scene)),
    );
  document
    .querySelectorAll("[data-concept]")
    .forEach((b) =>
      b.setAttribute("aria-pressed", String(b.dataset.concept === concept)),
    );
  $("#concept-note").textContent =
    concept === "context"
      ? "Flow, step details and one Chat confirmation."
      : "A sequential review in the center; the same Chat composer.";
  const stateText =
    scene === "current"
      ? "Current"
      : scene === "uncertain"
        ? "Checking status"
        : scene === "stopping"
          ? "Stopping"
          : scene === "running"
            ? "Running"
            : scene === "complete"
              ? "Completed"
              : "Stopped";
  let body = `<header class="view-head"><div><h1>${scene === "current" ? "Read quality workflow" : "Analysis 12"}</h1><small>${scene === "current" ? "Current design" : "Saved design · Same data · Same results location"}</small></div><span class="pill ${["running", "complete"].includes(scene) ? "green" : ""}">${stateText}</span></header>${notice()}`;
  if (concept === "guided" && checked())
    body += `<div class="stepper"><span class="${wizardStep === 1 ? "active" : ""}">1 · Eligibility</span><span class="${wizardStep === 2 ? "active" : ""}">2 · Planned work</span><span class="${wizardStep === 3 ? "active" : ""}">3 · Confirm in Chat</span></div>`;
  if (concept === "guided" && checked() && wizardStep === 1)
    body += `<section class="guided"><h2>Continue this analysis</h2><p>Stopped cleanly. The saved design, input copy and installed tools match. Results were checked without starting any task.</p><p>One completed step can be reused. One stopped step will restart from its beginning.</p><footer>${btn("Review planned work →", "next", true)}</footer></section>`;
  else
    body += `<div class="section-head"><div><strong>${scene === "current" ? "Design flow" : ["running", "complete"].includes(scene) ? "Continued analysis" : hasReview() ? "Continuation plan" : "Saved Run flow"}</strong><small>${["running", "complete"].includes(scene) ? "1 result kept · 1 new attempt" : checked() ? "1 result to keep · 1 step to restart" : hasReview() ? "No executable plan · Resolve the notice above" : "Inspect the observed steps before deciding what to do."}</small></div>${["stopping", "uncertain"].includes(scene) ? btn("Check status", "status") : ["stopped", "stale", "blocked"].includes(scene) ? btn(scene === "stopped" ? "Check continuation" : "Check again", "check", true) : btn("Open current design", "current")}</div>${flow()}${detail()}${concept === "guided" && checked() && wizardStep === 2 ? `<div class="guided">${btn("Send review to Chat →", "next", true)}</div>` : ""}<details class="checks"><summary>What Gobble verifies</summary><ul><li>Stop has settled and no execution owner is active.</li><li>This Run’s saved design, data copy and installed tools match.</li><li>Completed output contents match their recorded fingerprints.</li><li>The stopped attempt has no ambiguous published outputs.</li><li>The same facts are checked again when you confirm.</li></ul></details>`;
  body +=
    '<p class="bottom-note">Design fixture only. No real analysis runs here. A continuation keeps the Run identity and its history; a changed design needs a separate analysis.</p>';
  $("#workspace").innerHTML = body;
  $("#messages").innerHTML =
    `<div class="speaker">◉ Research agent</div><p>The Trim step completed before you stopped the quality check. We can inspect what is safe to keep before continuing.</p>${btn("Show the Trim result", "point")}<p style="margin-top:12px">${scene === "newer" ? "Current contains newer changes. Those changes are outside this Run." : "A stopped step starts a new attempt. Completed steps are kept only after Gobble verifies them."}</p>${sent.map((x) => `<div class="user">${esc(x.text)}${x.ref ? `<small><br>↗ ${esc(x.ref.stepLabel)} · ${esc(x.ref.reviewId || x.ref.snapshot)}</small>` : ""}</div><div class="speaker">Research agent</div><p>I can discuss this saved reference. Continuing the analysis still requires your explicit confirmation.</p>`).join("")}`;
  let card = "";
  if (checked() && (concept === "context" || wizardStep === 3))
    card = `<h2>Continue Analysis 12?</h2><p><strong>Keep 1 result · Restart 1 step</strong></p><p>Same design and input copy.<br>Same results location. Earlier attempts stay recorded.</p><small>Gobble will verify this review again before starting. The stopped quality check restarts from its beginning.</small><footer>${btn("Resume analysis", "resume", true)}${btn("Not now", "dismiss")}</footer>`;
  else if (scene === "running" || scene === "complete")
    card = `<h2>${scene === "running" ? "Analysis 12 continued" : "Analysis 12 completed"}</h2><p>Trim: Reused · Attempt 1<br>Quality check: ${scene === "running" ? "Running" : "Completed"} · Attempt 2</p><small>One accepted continuation. Previous attempts preserved.</small>${scene === "running" ? btn("Show completed state", "finish") : ""}`;
  else if (scene === "uncertain")
    card = `<h2>Confirmation pending</h2><p>Reconnect to this exact request. Do not duplicate it.</p>${btn("Check status", "status", true)}`;
  else if (["stale", "blocked", "older", "stopping"].includes(scene))
    card = `<h2>${scene === "stale" ? "Review needs another check" : scene === "blocked" ? "Cannot continue yet" : scene === "older" ? "Continuation unavailable" : "Waiting for Stop"}</h2><p>${scene === "older" ? "Existing evidence stays available. A new analysis can use Current." : "Nothing new has started."}</p>${btn("Resume analysis", "resume", true, true)}`;
  $("#confirmation").innerHTML = card
    ? `<article class="confirm-card" aria-label="Continuation confirmation">${card}</article>`
    : "";
  $("#attachment").innerHTML = attachment
    ? `<div class="attachment"><button data-action="preview">↗ ${esc(attachment.stepLabel)}<small>${esc(attachment.reviewId || attachment.snapshot)} · Attempt ${attachment.priorAttempt}${attachment.reviewId && attachment.reviewId !== `review_${reviewVersion}` ? " · Earlier review" : ""}</small></button><button data-action="remove" aria-label="Remove reference">×</button></div>`
    : "";
  $("#send").disabled = !draft.value.trim() && !attachment;
  history.replaceState(null, "", `?scene=${scene}&concept=${concept}`);
}
function change(next) {
  scene = next;
  wizardStep = 1;
  selected = scene === "blocked" ? "trim" : "qc";
  render();
  announce(`${next} design scenario. No real work started.`);
}
document.addEventListener("click", (e) => {
  const b = e.target.closest("button");
  if (!b || b.disabled) return;
  if (b.dataset.scene) {
    change(b.dataset.scene);
    return;
  }
  if (b.dataset.concept) {
    concept = b.dataset.concept;
    wizardStep = 1;
    render();
    return;
  }
  if (b.dataset.area) {
    area(b.dataset.area);
    return;
  }
  if (b.dataset.select) {
    selected = b.dataset.select;
    render();
    announce(
      `Selected ${selected === "trim" ? "Trim reads" : "Inspect read quality"}.`,
    );
    return;
  }
  switch (b.dataset.action) {
    case "point":
      marked = true;
      selected = "trim";
      render();
      area("workspace");
      announce("Showing the Agent reference.");
      break;
    case "attach":
      attachment = reference();
      render();
      area("chat");
      draft.focus();
      announce("Exact reference added to draft. Nothing sent.");
      break;
    case "preview":
      $("#capture-body").innerHTML =
        `<h2>${esc(attachment.stepLabel)}</h2><p>${esc(attachment.decision)}. This is the saved reference; it does not follow a newer review.</p><dl><dt>Analysis</dt><dd>${esc(attachment.run)}</dd><dt>Captured view</dt><dd>${attachment.reviewId ? "Continuation review " + esc(attachment.reviewId.split("_")[1]) : "Recorded task"}</dd><dt>Earlier attempt</dt><dd>Attempt ${attachment.priorAttempt}</dd>${attachment.plannedAttempt ? `<dt>Reviewed attempt</dt><dd>Attempt ${attachment.plannedAttempt}</dd>` : ""}</dl><p>Exact Run, snapshot, step and review identifiers travel with this reference behind the view.</p>`;
      $("#capture").showModal();
      break;
    case "remove":
      attachment = null;
      render();
      break;
    case "check":
      reviewVersion++;
      change(scene === "stale" || scene === "blocked" ? "blocked" : "ready");
      break;
    case "resume":
      if (checked() && (concept === "context" || wizardStep === 3)) {
        change("running");
        announce("Simulated Resume accepted once.");
      }
      break;
    case "status":
      change(scene === "stopping" ? "stopped" : "running");
      break;
    case "next":
      wizardStep++;
      render();
      if (wizardStep === 3) {
        area("chat");
        document.querySelector('[data-action="resume"]')?.focus();
      }
      break;
    case "dismiss":
      change("stopped");
      break;
    case "current":
      change("current");
      area("workspace");
      break;
    case "origin":
      change("stopped");
      area("workspace");
      break;
    case "finish":
      change("complete");
      break;
  }
});
$("#close-capture").onclick = () => $("#capture").close();
$("#send").onclick = () => {
  if (!draft.value.trim() && !attachment) return;
  sent.push({
    text: draft.value || "Please review this step.",
    ref: attachment ? { ...attachment } : null,
  });
  draft.value = "";
  attachment = null;
  render();
  announce("Message sent inside this design fixture only.");
};
draft.addEventListener("input", () => {
  $("#send").disabled = !draft.value.trim() && !attachment;
});
area("workspace");
render();
