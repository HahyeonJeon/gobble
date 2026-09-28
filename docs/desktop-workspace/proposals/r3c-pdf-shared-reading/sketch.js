const $ = (id) => document.getElementById(id);
const escapeHtml = (value) =>
  String(value).replace(
    /[&<>"']/g,
    (c) =>
      ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[
        c
      ],
  );
const state = {
  concept: "a",
  page: 3,
  zoom: "fit",
  pages: false,
  selection: null,
  attachment: null,
  region: false,
  reference: false,
  saved: null,
  scanned: false,
  changed: false,
  capture: true,
};
const titles = [
  "Study overview",
  "Methods and cohort",
  "Results and quality control",
  "Limitations and next steps",
];
const sampleQuote =
  "All comparisons use the same filtered cohort, with library size normalization and adjustment for age, sex and batch.";
let noticeTimer;
let drag = null;
const figure =
  '<div class="figure"><svg viewBox="0 0 490 100" role="img" aria-label="Synthetic printed figure showing samples, filtering and analysis"><rect x="5" y="20" width="122" height="58" rx="5" fill="#e7f0e9" stroke="#b6cabc"/><text x="66" y="45" text-anchor="middle" font-size="12" fill="#3d6150">48 samples</text><text x="66" y="63" text-anchor="middle" font-size="10" fill="#75857a">case + control</text><path d="M136 49 H176 M171 44 L176 49 171 54" stroke="#809d8c" fill="none"/><rect x="187" y="20" width="121" height="58" rx="5" fill="#f5f7f2" stroke="#bdcbbf"/><text x="248" y="45" text-anchor="middle" font-size="12" fill="#3d6150">Quality filters</text><text x="248" y="63" text-anchor="middle" font-size="10" fill="#75857a">one shared cohort</text><path d="M317 49 H357 M352 44 L357 49 352 54" stroke="#809d8c" fill="none"/><rect x="367" y="20" width="117" height="58" rx="5" fill="#e7f0e9" stroke="#b6cabc"/><text x="425" y="45" text-anchor="middle" font-size="12" fill="#3d6150">Comparison</text><text x="425" y="63" text-anchor="middle" font-size="10" fill="#75857a">report results</text></svg></div>';
function notice(text) {
  clearTimeout(noticeTimer);
  $("notice").textContent = text;
  $("notice").classList.add("visible");
  noticeTimer = setTimeout(() => $("notice").classList.remove("visible"), 2600);
}
function switchSection(section) {
  document.querySelector(".app").dataset.section = section;
  $("workspace-tab").setAttribute(
    "aria-pressed",
    String(section === "workspace"),
  );
  $("chat-tab").setAttribute("aria-pressed", String(section === "chat"));
}
function label(selection) {
  return selection
    ? "PDF · p. " +
        selection.page +
        " · " +
        { text: "Selected text", region: "Region", page: "Whole page" }[
          selection.kind
        ]
    : "";
}
function paperContent() {
  const page = state.reference ? 2 : state.page;
  const scanned = state.scanned && !state.reference;
  if (state.reference && state.changed && state.capture)
    return (
      '<div class="paper-label">CAPTURED REGION / SYNTHETIC EVIDENCE</div><h1>Saved excerpt</h1><div class="agent-target"><p>' +
      sampleQuote +
      '</p></div><p class="caption">Page 2 · Captured region only. The rest of the original page is not retained in this sample.</p><div class="page-footer"><span>analysis-report.pdf · Earlier version</span><span>Region</span></div>'
    );
  return (
    '<div class="paper-label">SYNTHETIC RESEARCH REPORT / DESIGN FIXTURE</div><h1>' +
    titles[page - 1] +
    "</h1>" +
    (scanned
      ? '<div class="scan"><h2>Scanned report page</h2>' +
        figure +
        '<p class="caption">Image-only fixture. Select a region or the whole page.</p></div>'
      : "<p>This illustrative report reviews 48 RNA-seq samples from 24 cases and 24 controls. The text and figure are synthetic and are used only to evaluate the reading workflow.</p>" +
        "<h2>" +
        (page === 2 ? "Cohort and comparison" : "A consistent comparison") +
        '</h2><p class="selectable ' +
        (state.reference ? "agent-target" : "") +
        '">' +
        sampleQuote +
        "</p>" +
        "<p>Read the report alongside your project notes. Select an excerpt to discuss the precise wording, or select a region to refer to a figure without interpreting it as underlying data.</p>" +
        figure +
        '<p class="caption">Figure 1. A printed figure inside the synthetic PDF page. This View does not create charts.</p>') +
    '<div class="page-footer"><span>RNA-seq study · Illustrative content</span><span>' +
    page +
    "</span></div>"
  );
}
function drawRegion() {
  const box = $("region-box"),
    paper = $("paper"),
    scroller = $("page-scroll");
  if (state.reference || state.selection?.kind !== "region") {
    box.hidden = true;
    return;
  }
  const a = paper.getBoundingClientRect(),
    b = scroller.getBoundingClientRect(),
    rect = state.selection.rect;
  box.hidden = false;
  Object.assign(box.style, {
    left: a.left - b.left + scroller.scrollLeft + rect.x * a.width + "px",
    top: a.top - b.top + scroller.scrollTop + rect.y * a.height + "px",
    width: rect.w * a.width + "px",
    height: rect.h * a.height + "px",
  });
}
function renderSelection() {
  $("selection-bar").hidden = !state.selection || state.reference;
  $("selection-label").textContent = label(state.selection);
  $("paper").classList.toggle(
    "selected-text",
    state.selection?.kind === "text" && !state.reference,
  );
  $("gesture-hint").textContent = state.reference
    ? "Read-only reference"
    : state.region
      ? "Arrows move · Shift + arrows resize · Esc cancels"
      : state.scanned
        ? "Region or page selection"
        : "Drag text to select";
  drawRegion();
}
function renderDraft() {
  $("attachment").hidden = !state.attachment;
  $("attachment-label").textContent = label(state.attachment);
  $("mobile-count").textContent = state.attachment ? "1" : "";
  $("send").disabled = !$("composer").value.trim() && !state.attachment;
  $("draft-status").textContent = state.attachment
    ? "Captured sample · not sent"
    : "One conversation. Shared context.";
}
function renderRail() {
  const popover =
    state.concept === "a" || matchMedia("(max-width:780px)").matches;
  const visible = (!popover || state.pages) && !state.reference;
  $("page-rail").hidden = !visible;
  $("page-rail").classList.toggle("popover", popover);
  $("pages-toggle").setAttribute("aria-expanded", String(visible));
}
function render() {
  const unavailable = state.reference && state.changed && !state.capture;
  $("concept-a").setAttribute("aria-pressed", String(state.concept === "a"));
  $("concept-b").setAttribute("aria-pressed", String(state.concept === "b"));
  renderRail();
  $("pages-toggle").disabled = state.reference;
  $("page-rail").innerHTML = titles
    .map(
      (title, i) =>
        '<button data-page="' +
        (i + 1) +
        '" aria-label="Page ' +
        (i + 1) +
        ": " +
        title +
        '" aria-current="' +
        (state.page === i + 1 ? "page" : "false") +
        '"><span class="mini-page" aria-hidden="true"></span><span>Page ' +
        (i + 1) +
        "</span></button>",
    )
    .join("");
  $("page-number").value = state.reference ? 2 : state.page;
  $("page-number").disabled = state.reference;
  $("previous").disabled = state.reference || state.page === 1;
  $("next").disabled = state.reference || state.page === 4;
  $("region-mode").disabled = state.reference;
  $("select-page").disabled = state.reference;
  $("region-mode").setAttribute(
    "aria-pressed",
    String(state.region && !state.reference),
  );
  $("sample").disabled = state.reference || state.scanned;
  $("zoom").disabled = state.reference;
  $("zoom").value = state.zoom;
  $("reference-banner").hidden = !state.reference;
  $("reference-title").textContent = state.changed
    ? "Captured reference · Researcher"
    : "Reference view · Researcher";
  if (unavailable) $("reference-title").textContent = "Reference unavailable";
  $("reference-detail").textContent = unavailable
    ? "Your view and draft are preserved."
    : state.changed
      ? "Historical sample · source has changed."
      : "Page 2 · Your page, selection and draft are preserved.";
  $("unavailable").hidden = !unavailable;
  $("paper").hidden = unavailable;
  $("paper").innerHTML = paperContent();
  $("paper").classList.toggle("reference", state.reference);
  $("paper").classList.toggle(
    "captured",
    state.reference && state.changed && state.capture,
  );
  $("paper").classList.toggle("region-mode", state.region && !state.reference);
  $("paper").tabIndex = state.region && !state.reference ? 0 : -1;
  $("paper").setAttribute(
    "aria-label",
    state.region
      ? "Region selection area. Use arrow keys to move; Shift and arrows to resize. Escape cancels."
      : "Synthetic report page " + (state.reference ? 2 : state.page),
  );
  $("paper").style.width =
    state.reference || state.zoom === "fit"
      ? ""
      : state.zoom === "100"
        ? "650px"
        : "812px";
  $("paper").style.maxWidth =
    state.reference || state.zoom === "fit" ? "" : "none";
  $("page-status").textContent =
    "Page " +
    (state.reference ? 2 : state.page) +
    " of 4 · " +
    (state.reference && state.changed
      ? "Historical sample"
      : state.scanned && !state.reference
        ? "Region only"
        : "Text available") +
    " · HTML sketch";
  if (unavailable)
    $("page-status").textContent =
      "Page 2 reference · No matching source or capture";
  renderSelection();
  renderDraft();
}
function navigate(page) {
  if (state.reference) return;
  const next = Math.max(1, Math.min(4, Number(page) || 1));
  state.page = next;
  state.selection = null;
  state.region = false;
  state.pages = false;
  render();
  $("page-scroll").scrollTop = 0;
}
function setConcept(concept) {
  const scroll = $("page-scroll").scrollTop;
  state.concept = concept;
  state.pages = false;
  render();
  $("page-scroll").scrollTop = scroll;
}
$("concept-a").onclick = () => setConcept("a");
$("concept-b").onclick = () => setConcept("b");
$("pages-toggle").onclick = () => {
  if (state.concept === "b" && !matchMedia("(max-width:780px)").matches)
    setConcept("a");
  else {
    state.pages = !state.pages;
    render();
  }
};
$("page-rail").onclick = (event) => {
  const button = event.target.closest("[data-page]");
  if (button) navigate(button.dataset.page);
};
$("previous").onclick = () => navigate(state.page - 1);
$("next").onclick = () => navigate(state.page + 1);
$("page-number").onchange = (event) => navigate(event.target.value);
$("zoom").onchange = (event) => {
  state.zoom = event.target.value;
  render();
};
$("sample").onclick = () => {
  state.selection = { kind: "text", page: state.page, quote: sampleQuote };
  state.region = false;
  render();
  notice("Synthetic text selected. Discuss adds it to the draft.");
};
$("scanned").onchange = (event) => {
  state.scanned = event.target.checked;
  if (!state.reference) {
    state.selection = null;
    state.region = false;
  }
  render();
};
$("changed").onchange = (event) => {
  state.changed = event.target.checked;
  render();
};
$("capture").onchange = (event) => {
  state.capture = event.target.checked;
  render();
};
$("region-mode").onclick = () => {
  state.region = !state.region;
  state.selection = state.region
    ? {
        kind: "region",
        page: state.page,
        rect: { x: 0.12, y: 0.24, w: 0.65, h: 0.18 },
      }
    : null;
  window.getSelection()?.removeAllRanges();
  render();
  if (state.region) {
    $("paper").focus({ preventScroll: true });
    notice(
      "Drag on the page, or use arrows to move and Shift + arrows to resize.",
    );
  }
};
$("select-page").onclick = () => {
  state.region = false;
  state.selection = { kind: "page", page: state.page };
  render();
};
$("clear-selection").onclick = () => {
  state.selection = null;
  state.region = false;
  window.getSelection()?.removeAllRanges();
  render();
};
$("discuss").onclick = () => {
  if (!state.selection || state.reference) return;
  state.attachment = structuredClone(state.selection);
  renderDraft();
  if (matchMedia("(max-width:780px)").matches) switchSection("chat");
  $("composer").focus();
  notice("Added to your draft. Nothing was sent.");
};
$("remove-attachment").onclick = () => {
  state.attachment = null;
  renderDraft();
};
$("composer").oninput = renderDraft;
$("send").onclick = () => {
  if ($("send").disabled) return;
  const text = $("composer").value.trim();
  const message = document.createElement("div");
  message.className = "message user";
  message.innerHTML =
    '<span class="byline">You <small>Demo</small></span>' +
    (text ? "<p>" + escapeHtml(text) + "</p>" : "") +
    (state.attachment
      ? '<p class="simulation-note">' +
        escapeHtml(label(state.attachment)) +
        "</p>"
      : "");
  $("messages").append(message);
  $("composer").value = "";
  state.attachment = null;
  renderDraft();
  $("messages").scrollTop = $("messages").scrollHeight;
  notice("Demo message added locally. No Agent was contacted.");
};
$("show-reference").onclick = () => {
  if (!state.reference)
    state.saved = {
      page: state.page,
      zoom: state.zoom,
      selection: structuredClone(state.selection),
      region: state.region,
      scrollTop: $("page-scroll").scrollTop,
      scrollLeft: $("page-scroll").scrollLeft,
    };
  state.reference = true;
  render();
  switchSection("workspace");
  $("page-scroll").scrollTop = 0;
  $("page-scroll").scrollLeft = 0;
  $("return-view").focus({ preventScroll: true });
};
function returnView() {
  const saved = state.saved;
  state.reference = false;
  state.saved = null;
  if (saved) {
    state.page = saved.page;
    state.zoom = saved.zoom;
    state.selection = saved.selection;
    state.region = saved.region;
  }
  render();
  if (saved) {
    $("page-scroll").scrollTop = saved.scrollTop;
    $("page-scroll").scrollLeft = saved.scrollLeft;
  }
  $("region-mode").focus({ preventScroll: true });
}
$("return-view").onclick = returnView;
$("open-current").onclick = () => {
  returnView();
  state.selection = null;
  state.region = false;
  render();
  notice("Current file opened. It does not resolve the historical reference.");
};
$("workspace-tab").onclick = () => switchSection("workspace");
$("chat-tab").onclick = () => switchSection("chat");
function point(event) {
  const box = $("paper").getBoundingClientRect();
  return {
    x: Math.max(0, Math.min(1, (event.clientX - box.left) / box.width)),
    y: Math.max(0, Math.min(1, (event.clientY - box.top) / box.height)),
  };
}
$("paper").addEventListener("pointerdown", (event) => {
  if (!state.region || state.reference || event.button !== 0) return;
  drag = point(event);
  $("paper").setPointerCapture(event.pointerId);
  event.preventDefault();
});
$("paper").addEventListener("pointermove", (event) => {
  if (!drag) return;
  const end = point(event);
  state.selection = {
    kind: "region",
    page: state.page,
    rect: {
      x: Math.min(drag.x, end.x),
      y: Math.min(drag.y, end.y),
      w: Math.abs(drag.x - end.x),
      h: Math.abs(drag.y - end.y),
    },
  };
  renderSelection();
});
$("paper").addEventListener("pointerup", () => {
  if (drag) {
    drag = null;
    if (state.selection.rect.w < 0.01 || state.selection.rect.h < 0.01)
      state.selection = null;
    renderSelection();
    return;
  }
  if (state.reference || state.scanned || state.region) return;
  const selected = window.getSelection();
  if (
    !selected ||
    selected.isCollapsed ||
    !$("paper").contains(selected.anchorNode) ||
    !$("paper").contains(selected.focusNode)
  )
    return;
  const quote = selected.toString().trim();
  if (!quote) return;
  state.selection = { kind: "text", page: state.page, quote };
  renderSelection();
});
$("paper").addEventListener("pointercancel", () => {
  drag = null;
});
$("paper").addEventListener("keydown", (event) => {
  if (!state.region || state.reference) return;
  if (event.key === "Escape") {
    event.preventDefault();
    state.region = false;
    state.selection = null;
    render();
    $("region-mode").focus();
    return;
  }
  if (!["ArrowLeft", "ArrowRight", "ArrowUp", "ArrowDown"].includes(event.key))
    return;
  event.preventDefault();
  const r =
    state.selection?.kind === "region"
      ? { ...state.selection.rect }
      : { x: 0.1, y: 0.2, w: 0.6, h: 0.2 };
  const dx =
    event.key === "ArrowLeft" ? -0.01 : event.key === "ArrowRight" ? 0.01 : 0;
  const dy =
    event.key === "ArrowUp" ? -0.01 : event.key === "ArrowDown" ? 0.01 : 0;
  if (event.shiftKey) {
    r.w = Math.max(0.02, Math.min(1 - r.x, r.w + dx));
    r.h = Math.max(0.02, Math.min(1 - r.y, r.h + dy));
  } else {
    r.x = Math.max(0, Math.min(1 - r.w, r.x + dx));
    r.y = Math.max(0, Math.min(1 - r.h, r.y + dy));
  }
  state.selection = { kind: "region", page: state.page, rect: r };
  renderSelection();
});
document.addEventListener("keydown", (event) => {
  if (event.key === "Escape" && state.pages) {
    state.pages = false;
    render();
    $("pages-toggle").focus();
  }
});
window.addEventListener("resize", () => {
  renderRail();
  drawRegion();
});
render();
