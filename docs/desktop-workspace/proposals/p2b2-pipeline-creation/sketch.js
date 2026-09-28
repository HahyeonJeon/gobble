// Review-only sketch. All data and transitions below are illustrative and local.
const $ = (s) => document.querySelector(s);
const params = new URLSearchParams(location.search);
let concept = params.get("concept") === "chat" ? "chat" : "project";
let scene = params.get("scene") || (concept === "chat" ? "welcome" : "setup");
const fileSizes = {
  "S01.fastq.gz": "1.2 GB",
  "S02.fastq.gz": "1.4 GB",
  "S03.fastq.gz": "0.9 GB",
};
let pendingFile = "S01.fastq.gz";
let boundFile = ["review", "changed", "adopted", "checking"].includes(scene)
  ? "S01.fastq.gz"
  : null;
let selected = "trim";
let reference = null;
let confirming = false;
let sent = ["review", "changed", "adopted", "checking"].includes(scene);
let toastTimer;
let mockJob = 0;
const names = {
  input: "Input reads",
  trim: "Trim adapters",
  quality: "Check read quality",
};
function toast(text) {
  $("#toast").textContent = text;
  $("#toast").hidden = false;
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => ($("#toast").hidden = true), 4000);
}
function setArea(area) {
  $(".app").dataset.area = area;
  document.querySelectorAll("[data-area]").forEach((b) => {
    if (b.tagName === "BUTTON")
      b.setAttribute("aria-pressed", String(b.dataset.area === area));
  });
}
function changeScene(next) {
  scene = next;
  confirming = false;
  mockJob++;
  if (["review", "changed", "adopted", "checking"].includes(next)) {
    boundFile ||= "S01.fastq.gz";
    sent = true;
  }
  $("#permission").checked = false;
  render();
}
function diagram() {
  return `<div class="diagram" aria-label="Proposed read-quality flow"><div class="graph"><svg viewBox="0 0 800 145" preserveAspectRatio="none" aria-hidden="true"><defs><marker id="arrow" viewBox="0 0 10 10" refX="8" refY="5" markerWidth="5" markerHeight="5" orient="auto"><path d="M0 0 L10 5 L0 10z" fill="#6fa78b"/></marker></defs><path d="M216 54 H288 M480 54 H524 Q536 54 536 66 V83 Q536 95 548 95 H568" fill="none" stroke="#6fa78b" stroke-width="2" marker-end="url(#arrow)"/></svg><button class="node input" data-node="input" aria-pressed="${selected === "input"}" style="--x:3%;--y:22px"><i>1 ＋</i><strong><span class="glyph">▧</span> ${boundFile || pendingFile}</strong><small>Single-end · Selected file</small></button><button class="node" data-node="trim" aria-pressed="${selected === "trim"}" style="--x:36%;--y:22px"><i>2 ＋</i><strong><span class="glyph">◇</span> Trim adapters</strong><small>Trim Galore · 2 outputs</small></button><button class="node" data-node="quality" aria-pressed="${selected === "quality"}" style="--x:71%;--y:63px"><i>3 ＋</i><strong><span class="glyph">◎</span> Check read quality</strong><small>FastQC · 2 outputs</small></button></div></div>`;
}
function details() {
  const after =
    selected === "input"
      ? `<strong>${boundFile || "S01.fastq.gz"}</strong><p>From this Project · Single-end reads</p><div class="fact"><span>File size</span><b>${fileSizes[boundFile || pendingFile]}</b></div><div class="fact"><span>Contents</span><b>Not processed</b></div>`
      : selected === "trim"
        ? `<strong>Trim adapters</strong><p>Input: selected reads → trimmed reads</p><div class="fact"><span>Quality threshold</span><b>25 Phred</b></div><div class="fact"><span>Minimum length</span><b>40 bp</b></div><p>Declares trimmed reads and a trimming report.</p>`
        : `<strong>Check read quality</strong><p>Input: trimmed reads</p><div class="fact"><span>Tool</span><b>FastQC</b></div><div class="fact"><span>Declared outputs</span><b>HTML · ZIP</b></div><p>Reports will be generated only when a Run starts.</p>`;
  return `<nav class="changes" aria-label="Added elements">${Object.entries(
    names,
  )
    .map(
      ([id, name], i) =>
        `<button data-node="${id}" aria-pressed="${selected === id}"><b>${i + 1}</b>${name}</button>`,
    )
    .join(
      "",
    )}</nav><section class="details" aria-label="Selected addition"><header><div><small>NEW PIPELINE · ${selected === "input" ? "INPUT" : "ADDED STEP"}</small><h2>${names[selected]}</h2></div><button id="discuss">Discuss this addition →</button></header><div class="pair"><section><h3>${scene === "adopted" ? "Before creation" : "Current"}</h3><strong>No current version</strong><p>This is a new Pipeline. Existing Project pipelines stay as they are.</p></section><section><h3>${scene === "adopted" ? "Adopted first version" : "＋ Proposed"}</h3>${after}</section></div><p class="note">Example checked design · File content and analysis results have not been validated.</p></section>`;
}
function setup() {
  return `${scene === "offline" ? '<div class="notice" role="status"><strong>Analysis engine unavailable</strong>Your draft and data selection are saved. Reconnect to check a proposal.</div>' : ""}<div class="intro"><span class="eyebrow">NEW PIPELINE</span><h1>Start with your data</h1><p>Select the reads you want to use. Describe your analysis in Chat; your Agent will propose the steps for you to review.</p></div><section class="data-block" aria-label="Select input reads"><header class="sectionhead"><strong>Project files / reads</strong><span>Select one single-end FASTQ file</span></header><label class="sr-only" for="search">Search Project files</label><input id="search" class="search" placeholder="Search Project files…"><div id="files">${["S01.fastq.gz", "S02.fastq.gz", "S03.fastq.gz"].map((f, i) => `<label class="file-row" data-file="${f}"><input type="radio" name="reads" value="${f}" ${pendingFile === f ? "checked" : ""}><span><strong>${f}</strong><small>Modified today · Project file</small></span><em>${["1.2", "1.4", "0.9"][i]} GB</em></label>`).join("")}</div><p id="no-files" class="data-note" hidden>No matching files in this folder.</p><div class="data-note">Reads remain in your Project. Only the selected file’s name and metadata are shared when you send the message.</div></section><div class="engine"><span>${scene === "offline" ? "○" : "●"}</span><div><strong>Local analysis engine</strong><small>${scene === "offline" ? "Not connected · Your draft is preserved" : "Ready for read-quality design · No processing will start"}</small></div>${scene === "offline" ? '<button id="reconnect">Reconnect</button>' : '<span class="pill">Ready</span>'}</div>`;
}
function render() {
  $("#scene").value =
    scene === "welcome" || scene === "discarded" ? "setup" : scene;
  document
    .querySelectorAll("[data-concept]")
    .forEach((b) =>
      b.setAttribute("aria-pressed", String(b.dataset.concept === concept)),
    );
  const activeDraft = !["welcome", "discarded", "adopted"].includes(scene);
  $("#draft-nav").hidden = !activeDraft;
  $("#pipeline-count").textContent = scene === "adopted" ? "2" : "1";
  $("#adopted-item")?.remove();
  if (scene === "adopted") {
    const item = document.createElement("div");
    item.id = "adopted-item";
    item.className = "navitem active";
    item.textContent = "Read quality";
    $("#new").before(item);
  }
  $("#view-name").textContent = [
    "review",
    "changed",
    "adopted",
    "checking",
  ].includes(scene)
    ? "Read quality"
    : "New pipeline";
  $("#state-badge").textContent =
    scene === "adopted"
      ? "Version 1"
      : scene === "checking"
        ? "Checking"
        : scene === "changed"
          ? "Needs review"
          : "Draft";
  $("#state-badge").hidden = ["welcome", "discarded"].includes(scene);
  $("#discard").hidden = !activeDraft;
  const inReview = ["review", "changed", "adopted"].includes(scene);
  const body = $("#workbody");
  body.classList.toggle("review-body", inReview);
  if (inReview)
    body.innerHTML = `${scene === "changed" ? '<div class="notice" role="status"><strong>The selected file changed</strong>This comparison uses the earlier file selection. Confirm the file and check a new proposal before adopting.</div>' : ""}<div class="review-title"><div><span class="eyebrow">${scene === "adopted" ? "FIRST VERSION ADOPTED" : "FIRST PROPOSAL · NOT YET A PIPELINE"}</span><h1>Read quality</h1><p>${scene === "adopted" ? "The reviewed design is now in this Project." : "Trim adapters, then check the quality of the trimmed reads."}</p></div><span>＋ 1 input · 2 steps</span></div>${diagram()}${details()}`;
  else if (scene === "checking")
    body.innerHTML =
      '<div class="waiting"><span class="orb">◌</span><h1>Checking the proposed flow</h1><p>Your Agent has submitted a first draft. Gobble is checking its steps, connections and selected input.</p><div class="steps"><span>✓ Draft received</span><span>◌ Checking design</span><span>○ Ready to review</span></div><p class="note">No Pipeline has been added yet. Your reads are not being processed.</p></div>';
  else if (scene === "welcome" || scene === "discarded")
    body.innerHTML = `<div class="waiting"><span class="orb">◇</span><h1>${scene === "discarded" ? "Draft discarded" : "Create an analysis together"}</h1><p>${scene === "discarded" ? "No Pipeline was added. Your existing pipelines and unsent Chat text are unchanged." : "Start in Chat with what you want to do. Confirm a new draft when your Agent asks, then choose the Project data."}</p><button class="primary" id="start-draft">${scene === "discarded" ? "Start another draft" : "Start pipeline draft"}</button><p class="note">One shared workspace and one message composer.</p></div>`;
  else body.innerHTML = setup();
  const actions = $("#actions");
  actions.innerHTML =
    scene === "adopted"
      ? '<div><strong>Added to Project · Version 1</strong>No Run was started. This comparison stays in history.</div><button id="show-current">View current flow</button>'
      : scene === "review"
        ? confirming
          ? '<div><strong>Add “Read quality” to this Project?</strong>Save this checked first version. No analysis will start.</div><button id="back">Back</button><button class="primary" id="confirm">Confirm adoption</button>'
          : '<div><strong>Ready for your review</strong>Adds a new Pipeline and its first version together.</div><button id="revise">Ask for changes</button><button class="primary" id="adopt">Adopt as new pipeline</button>'
        : scene === "changed"
          ? '<div><strong>Confirm the input before continuing</strong>Your earlier comparison remains available.</div><button id="reselect">Review selected file</button><button disabled>Adopt as new pipeline</button>'
          : scene === "checking"
            ? '<div><strong>Draft only</strong>Other pipelines and Runs are unaffected.</div><button id="cancel-check">Cancel check</button>'
            : ["welcome", "discarded"].includes(scene)
              ? "<div>A Pipeline is registered only after you review and adopt its first version.</div>"
              : `<div><strong>${boundFile ? "Selected: " + boundFile : "Choose one file to continue"}</strong>${boundFile ? "Next, describe the analysis in Chat." : "The first supported scope is single-end read quality."}</div><button id="use-file" class="primary">${boundFile === pendingFile ? "Continue in Chat →" : "Use selected file"}</button>`;
  const context = $("#context");
  context.replaceChildren();
  if (reference || (activeDraft && boundFile)) {
    const title = document.createElement("strong");
    title.textContent = reference
      ? "First proposal · " + names[reference]
      : "New Pipeline · " + boundFile;
    const sub = document.createElement("small");
    sub.textContent = reference
      ? "This exact addition will be discussed."
      : "Selected Project metadata · Single-end reads";
    const remove = document.createElement("button");
    remove.textContent = "×";
    remove.setAttribute("aria-label", "Remove discussion context");
    remove.onclick = () => {
      reference = null;
      $("#permission").checked = false;
      render();
    };
    if (reference) context.append(remove);
    context.append(title, sub);
  }
  $(".permission").hidden = ["adopted", "welcome", "discarded"].includes(scene);
  updateSend();
  const timeline = $("#timeline");
  const author =
    '<div class="author"><span class="avatar">R</span><strong>Research partner</strong></div>';
  timeline.innerHTML =
    concept === "chat" && scene === "welcome"
      ? `<div class="bubble">I want a read-quality pipeline for this Project.</div>${author}<p>I can help design that. Start a new draft and choose the reads to use. We will review the flow before adding it to the Project.</p><button id="chat-start" class="message-action">＋ Start pipeline draft<small>No source template or configuration files needed.</small></button>`
      : sent
        ? `<div class="bubble">Trim adapters from ${boundFile || pendingFile} at quality 25 and minimum length 40, then check the trimmed reads.</div>${author}<p>${scene === "checking" ? "I submitted a draft for checking. Your existing pipelines are unchanged." : "The proposal has two processing steps. Select either one to review what it does."}</p><button data-chat-node="trim" class="message-action">2 · Trim adapters<small>Quality 25 Phred · Minimum length 40 bp</small></button><button data-chat-node="quality" class="message-action">3 · Check read quality<small>FastQC reports from the trimmed reads</small></button><p class="note">${scene === "adopted" ? "You adopted version 1. No Run has started." : "These are proposed steps, not completed results."}</p>`
        : `${author}<p>Choose the reads you want to work with, then tell me the analysis you want.</p><p>I will prepare a flow that we can review together. You decide whether to add it to this Project.</p><p class="note">Only selected data context is sent. Your other pipelines stay unchanged.</p>`;
  bind();
}
function bind() {
  document.querySelectorAll("[data-node]").forEach(
    (b) =>
      (b.onclick = () => {
        selected = b.dataset.node;
        render();
      }),
  );
  document.querySelectorAll("[data-chat-node]").forEach(
    (b) =>
      (b.onclick = () => {
        selected = b.dataset.chatNode;
        if (!["review", "changed", "adopted"].includes(scene)) {
          toast(
            "This draft is still being checked. Review links are available after checking.",
          );
          return;
        }
        render();
        setArea("work");
      }),
  );
  document.querySelectorAll("input[name=reads]").forEach(
    (input) =>
      (input.onchange = () => {
        pendingFile = input.value;
        render();
      }),
  );
  $("#search")?.addEventListener("input", (e) => {
    let count = 0;
    document.querySelectorAll("[data-file]").forEach((row) => {
      row.hidden = !row.dataset.file
        .toLowerCase()
        .includes(e.target.value.toLowerCase());
      if (!row.hidden) count++;
    });
    $("#no-files").hidden = !!count;
  });
  for (const id of ["start-draft", "chat-start"])
    if ($("#" + id))
      $("#" + id).onclick = () => {
        changeScene("setup");
        setArea("work");
      };
  if ($("#use-file"))
    $("#use-file").onclick = () => {
      boundFile = pendingFile;
      reference = null;
      $("#permission").checked = false;
      render();
      setArea("chat");
      $("#message").focus();
      toast(
        "Selected file added to this message. Your draft text is preserved.",
      );
    };
  if ($("#reconnect"))
    $("#reconnect").onclick = () => {
      scene = "setup";
      render();
      toast("Preview: the same local engine is available again.");
    };
  if ($("#reselect"))
    $("#reselect").onclick = () => {
      reference = null;
      boundFile = null;
      changeScene("setup");
    };
  if ($("#cancel-check"))
    $("#cancel-check").onclick = () => {
      changeScene("setup");
      toast("Check cancelled. Draft retained; no Pipeline registered.");
    };
  if ($("#discuss"))
    $("#discuss").onclick = () => {
      reference = selected;
      $("#permission").checked = false;
      render();
      setArea("chat");
      $("#message").focus();
      toast("Added this exact addition to Chat.");
    };
  if ($("#revise"))
    $("#revise").onclick = () => {
      reference = selected;
      render();
      setArea("chat");
      $("#message").focus();
    };
  if ($("#adopt"))
    $("#adopt").onclick = () => {
      confirming = true;
      render();
    };
  if ($("#back"))
    $("#back").onclick = () => {
      confirming = false;
      render();
    };
  if ($("#confirm"))
    $("#confirm").onclick = () => {
      reference = null;
      changeScene("adopted");
      toast(
        "Preview: Pipeline and first version added together. No Run started.",
      );
    };
  if ($("#show-current"))
    $("#show-current").onclick = () =>
      toast(
        "The adopted flow becomes Current; this first-version comparison remains in history.",
      );
}
$("#scene").onchange = (e) => {
  changeScene(e.target.value);
  setArea("work");
};
document.querySelectorAll("[data-concept]").forEach(
  (b) =>
    (b.onclick = () => {
      concept = b.dataset.concept;
      boundFile = null;
      reference = null;
      sent = false;
      changeScene(concept === "chat" ? "welcome" : "setup");
      setArea("work");
    }),
);
document
  .querySelectorAll("button[data-area]")
  .forEach((b) => (b.onclick = () => setArea(b.dataset.area)));
$("#new").onclick = () => {
  sent = false;
  boundFile = null;
  reference = null;
  changeScene("setup");
  setArea("work");
};
$("#draft-item").onclick = () => {
  setArea("work");
  toast(
    "This draft is separate from registered pipelines. Closing its view keeps the draft.",
  );
};
$("#discard").onclick = () => {
  boundFile = null;
  reference = null;
  sent = false;
  changeScene("discarded");
  toast(
    "Preview: draft discarded. Existing pipelines and unsent text preserved.",
  );
};
$("#agent").onclick = () =>
  toast(
    "Uses the existing Project Agent roster; no separate creation Agent is required.",
  );
function updateSend() {
  const needsPermission = ["setup", "offline"].includes(scene);
  $("#send").disabled =
    !$("#message").value.trim() ||
    ["checking", "changed", "discarded"].includes(scene) ||
    (needsPermission && (!boundFile || !$("#permission").checked));
}
$("#permission").onchange = updateSend;
$("#message").oninput = updateSend;
$("#send").onclick = () => {
  if ((reference || scene === "review") && !$("#permission").checked) {
    toast(
      "Preview: this exact comparison reference is sent in the existing conversation.",
    );
    return;
  }
  if (scene === "welcome") {
    changeScene("setup");
    setArea("work");
    return;
  }
  if (scene === "offline") {
    toast("Draft retained. Reconnect the engine before checking.");
    return;
  }
  if (scene === "adopted") {
    toast("Preview: continue discussing the adopted flow in Chat.");
    return;
  }
  sent = true;
  changeScene("checking");
  setArea("work");
  const job = mockJob;
  setTimeout(() => {
    if (scene === "checking" && mockJob === job) changeScene("review");
  }, 1000);
};
render();
