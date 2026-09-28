// Review-only state machine. All data and references are synthetic and local.
const $ = (id) => document.getElementById(id);
const state = {
  concept: "a",
  version: 1,
  changed: false,
  selection: null,
  attachment: null,
  reference: null,
  returnScroll: 0,
  returnFocus: null,
};
const targets = {
  source: {
    label: "Cell 3 · Source · Line 1",
    text: 'keep = qc["mapped_pct"] >= 80',
    cell: 3,
  },
  output: {
    label: "Cell 3 · Output 1 · Plain text",
    text: "  sample  mapped_pct\n0 S01           94.2\n2 S03           91.7",
    cell: 3,
  },
  load: {
    label: "Cell 2 · Source",
    text: 'import pandas as pd\nqc = pd.read_csv("mapping_quality.csv")',
    cell: 2,
  },
  notes: {
    label: "Cell 1 · Source",
    text: "# Sample quality review\nInspect mapping quality before choosing a filter. Compare the saved result with the method below.",
    cell: 1,
  },
};
function select(kind) {
  if (state.reference || state.concept === "b") return;
  state.selection = { kind, ...targets[kind], version: state.version };
  if (kind === "source" && state.version === 2)
    state.selection.text = 'keep = qc["mapped_pct"] >= 85';
  renderSelection();
}
function renderSelection() {
  const s = state.selection;
  $("selection-label").textContent = s
    ? s.label
    : "Select a cell, source line or output";
  $("attach").disabled =
    !s || state.reference !== null || state.concept === "b";
  $("clear-selection").hidden = !s;
  document
    .querySelectorAll(".selected")
    .forEach((el) => el.classList.remove("selected"));
  if (s && !state.reference) {
    const el =
      s.kind === "source"
        ? $("filter-line")
        : s.kind === "output"
          ? $("output-text")
          : $("cell-" + s.cell);
    el.classList.add("selected");
  }
}
function renderAttachment() {
  $("attachment").hidden = !state.attachment;
  if (state.attachment) {
    $("attachment-label").textContent = state.attachment.label;
    $("attachment-version").textContent =
      "Frozen attachment · Saved version " + state.attachment.version;
  }
}
function attach() {
  if (!state.selection || state.reference) return;
  state.attachment = structuredClone(state.selection);
  renderAttachment();
  $("draft").focus();
  $("delivery-status").textContent =
    "Attachment captured. Edit your message, then Send.";
}
function saveOrigin() {
  if (state.reference === null) {
    state.returnScroll = $("reader").scrollTop;
    state.returnFocus = document.activeElement;
  }
}
function showReference(captured = false) {
  if (state.concept === "b") return;
  saveOrigin();
  state.reference =
    captured || state.changed || state.version !== 1 ? "captured" : "live";
  const historic = state.reference === "captured";
  $("reader").hidden = historic;
  $("captured-view").hidden = !historic;
  $("reference-banner").hidden = false;
  $("reference-heading").textContent = historic
    ? "Captured evidence · Researcher"
    : "Reference view · Researcher";
  $("reference-subtitle").textContent = historic
    ? "Saved version 1 · Your view and selection are saved."
    : "Cell 3 · Source · Line 1 · Your view and selection are saved.";
  $("reader-toolbar").hidden = true;
  $("selection-bar").hidden = true;
  document.querySelectorAll("[data-selection]").forEach((button) => {
    button.disabled = true;
  });
  $("changed-banner").hidden = true;
  $("filter-line").classList.toggle("agent-mark", !historic);
  renderSelection();
  if (!historic) $("cell-3").scrollIntoView({ block: "start" });
  $("return-view").focus({ preventScroll: true });
}
function returnView(restoreFocus = true) {
  if (!state.reference) return;
  state.reference = null;
  $("reader").hidden = false;
  $("captured-view").hidden = true;
  $("reference-banner").hidden = true;
  $("reader-toolbar").hidden = false;
  $("selection-bar").hidden = false;
  document.querySelectorAll("[data-selection]").forEach((button) => {
    button.disabled = false;
  });
  $("changed-banner").hidden = !state.changed;
  $("filter-line").classList.remove("agent-mark");
  renderSelection();
  $("reader").scrollTop = state.returnScroll;
  if (restoreFocus && state.returnFocus?.isConnected)
    state.returnFocus.focus({ preventScroll: true });
}
function refresh() {
  if (state.reference) return;
  if (state.changed) {
    state.version = 2;
    state.changed = false;
    state.selection = null;
    $("threshold").textContent = "85";
    renderSelection();
    $("delivery-status").textContent =
      "Loaded saved version 2. Existing attachments keep their captured version.";
  } else
    $("delivery-status").textContent =
      "This is the current saved version in the demo.";
  $("changed-banner").hidden = true;
}
function concept(kind) {
  returnView(false);
  state.concept = kind;
  const isB = kind === "b";
  $("concept-a").setAttribute("aria-pressed", String(!isB));
  $("concept-b").setAttribute("aria-pressed", String(isB));
  $("reader").hidden = isB;
  $("reader-toolbar").hidden = isB;
  $("selection-bar").hidden = isB;
  $("connected-view").hidden = !isB;
  $("changed-banner").hidden = isB || !state.changed;
  $("concept-description").textContent = isB
    ? "B · Connect a Jupyter document and kernel. A separate bridge must preserve exact references and reconcile execution."
    : "A · Read saved cells and outputs in Gobble. Discuss exact parts through the existing Chat.";
  $("show-reference").disabled = isB;
  $("show-captured").disabled = isB;
  $("source-change").disabled = isB;
  renderSelection();
}
function toggleFiles() {
  const hidden = document
    .querySelector(".workspace")
    .classList.toggle("files-hidden");
  $("explorer-toggle").setAttribute("aria-expanded", String(!hidden));
}
document
  .querySelectorAll("[data-selection]")
  .forEach((button) =>
    button.addEventListener("click", () => select(button.dataset.selection)),
  );
document.querySelectorAll("[data-jump]").forEach((button) =>
  button.addEventListener("click", () => {
    $("cell-jump").open = false;
    const cell = $("cell-" + button.dataset.jump);
    cell.scrollIntoView({ block: "start" });
    cell.querySelector("button").focus({ preventScroll: true });
  }),
);
$("attach").addEventListener("click", attach);
$("clear-selection").addEventListener("click", () => {
  state.selection = null;
  renderSelection();
});
$("remove-attachment").addEventListener("click", () => {
  state.attachment = null;
  renderAttachment();
  $("draft").focus();
});
$("show-reference").addEventListener("click", () => showReference());
$("show-captured").addEventListener("click", () => showReference(true));
$("return-view").addEventListener("click", () => returnView());
$("source-change").addEventListener("click", () => {
  state.changed = true;
  $("changed-banner").hidden = state.reference !== null;
  $("delivery-status").textContent =
    "External edit simulated. Saved references remain at version 1.";
});
$("refresh").addEventListener("click", refresh);
$("refresh-banner").addEventListener("click", refresh);
$("concept-a").addEventListener("click", () => concept("a"));
$("concept-b").addEventListener("click", () => concept("b"));
$("connection-preview").addEventListener("click", () => {
  $("disconnected").hidden = false;
});
$("reset").addEventListener("click", () => location.reload());
$("explorer-toggle").addEventListener("click", toggleFiles);
$("focus-pane").addEventListener("click", () => {
  const focused = document.querySelector(".panes").classList.toggle("focused");
  $("focus-pane").textContent = focused ? "Restore split" : "Focus";
});
document
  .querySelector('[data-file="notebook"]')
  .addEventListener("click", () => {
    concept("a");
    $("reader").focus();
  });
document.querySelector('[data-file="method"]').addEventListener("click", () => {
  document.querySelector(".panes").classList.remove("focused");
  $("focus-pane").textContent = "Focus";
  document.querySelector(".method-content").setAttribute("tabindex", "-1");
  document.querySelector(".method-content").focus();
});
$("send").addEventListener("click", () => {
  if (!$("draft").value.trim() && !state.attachment) {
    $("delivery-status").textContent = "Write a message or attach a selection.";
    return;
  }
  const sent = document.createElement("article");
  sent.className = "sent-message";
  const author = document.createElement("strong");
  author.textContent = "You · Demo message";
  sent.append(author);
  const message = document.createElement("p");
  message.textContent = $("draft").value;
  sent.append(message);
  if (state.attachment) {
    const info = document.createElement("small");
    info.textContent =
      state.attachment.label +
      " · Captured version " +
      state.attachment.version;
    sent.append(info);
    const quote = document.createElement("p");
    quote.textContent = state.attachment.text;
    sent.append(quote);
  }
  $("sent-messages").append(sent);
  state.attachment = null;
  renderAttachment();
  $("draft").value = "";
  $("delivery-status").textContent =
    "Demo only. The example Researcher reference above can be opened with Show.";
  $("timeline").scrollTop = $("timeline").scrollHeight;
});
document.addEventListener("keydown", (event) => {
  if (event.key === "Escape") {
    if (state.reference) {
      event.preventDefault();
      returnView();
    } else if ($("cell-jump").open) {
      $("cell-jump").open = false;
      $("cell-jump").querySelector("summary").focus();
    }
  }
});
if (matchMedia("(max-width: 1100px)").matches) toggleFiles();
